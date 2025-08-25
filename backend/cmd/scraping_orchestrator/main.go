package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"seattle-family-activities-scraper/internal/models"
	"seattle-family-activities-scraper/internal/services"
)

// ScrapingOrchestratorEvent represents the input event for orchestrator
type ScrapingOrchestratorEvent struct {
	TriggerType string `json:"trigger_type"` // scheduled, manual, on-demand
	SourceID    string `json:"source_id,omitempty"` // optional: scrape specific source
	TaskType    string `json:"task_type,omitempty"` // full_scrape, incremental, validation
}

// ScrapingOrchestratorResponse represents the Lambda response
type ScrapingOrchestratorResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// Response body structure
type ResponseBody struct {
	Success        bool                    `json:"success"`
	Message        string                  `json:"message"`
	TasksScheduled int                     `json:"tasks_scheduled"`
	TaskSummary    []TaskSummary           `json:"task_summary,omitempty"`
	Error          string                  `json:"error,omitempty"`
}

type TaskSummary struct {
	SourceID     string    `json:"source_id"`
	SourceName   string    `json:"source_name"`
	TaskType     string    `json:"task_type"`
	Priority     string    `json:"priority"`
	ScheduledFor time.Time `json:"scheduled_for"`
	TaskID       string    `json:"task_id"`
}

var (
	dynamoService *services.DynamoDBService
	jinaClient    *services.JinaClient
	openAIClient  *services.OpenAIClient
	s3Client      *services.S3Client
	sqsClient     *sqs.Client
	taskQueueURL  string
)

func init() {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create DynamoDB client and service
	dynamoClient := dynamodb.NewFromConfig(cfg)
	dynamoService = services.NewDynamoDBService(
		dynamoClient,
		os.Getenv("FAMILY_ACTIVITIES_TABLE"),
		os.Getenv("SOURCE_MANAGEMENT_TABLE"),
		os.Getenv("SCRAPING_OPERATIONS_TABLE"),
	)

	// Create external API clients
	jinaClient = services.NewJinaClient()
	openAIClient = services.NewOpenAIClient()
	
	// Create S3 client for data storage
	s3Client, err = services.NewS3Client()
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}

	// Create SQS client for task queue
	sqsClient = sqs.NewFromConfig(cfg)
	taskQueueURL = os.Getenv("SCRAPING_TASK_QUEUE_URL")
	if taskQueueURL == "" {
		log.Fatalf("SCRAPING_TASK_QUEUE_URL environment variable is required")
	}
	
	log.Printf("Orchestrator initialized with task queue: %s", taskQueueURL)
}

// handleRequest processes the scraping orchestrator Lambda request
func handleRequest(ctx context.Context, event ScrapingOrchestratorEvent) (ScrapingOrchestratorResponse, error) {
	log.Printf("Processing scraping orchestration request: trigger=%s, source=%s, task_type=%s", 
		event.TriggerType, event.SourceID, event.TaskType)

	var tasksScheduled []TaskSummary
	var totalTasks int

	if event.SourceID != "" {
		// Process specific source
		tasks, err := processSpecificSource(ctx, event.SourceID, event.TaskType, event.TriggerType)
		if err != nil {
			log.Printf("Failed to process specific source %s: %v", event.SourceID, err)
			return createErrorResponse(500, fmt.Sprintf("Failed to process source %s: %v", event.SourceID, err))
		}
		tasksScheduled = tasks
		totalTasks = len(tasks)
	} else {
		// Process all active sources
		tasks, err := processAllActiveSources(ctx, event.TaskType, event.TriggerType)
		if err != nil {
			log.Printf("Failed to process active sources: %v", err)
			return createErrorResponse(500, fmt.Sprintf("Failed to process active sources: %v", err))
		}
		tasksScheduled = tasks
		totalTasks = len(tasks)
	}

	log.Printf("Successfully scheduled %d scraping tasks", totalTasks)

	responseBody := ResponseBody{
		Success:        true,
		Message:        fmt.Sprintf("Successfully scheduled %d scraping tasks", totalTasks),
		TasksScheduled: totalTasks,
		TaskSummary:    tasksScheduled,
	}

	return createSuccessResponse(responseBody)
}

// processSpecificSource creates scraping tasks for a specific source
func processSpecificSource(ctx context.Context, sourceID, taskType, triggerType string) ([]TaskSummary, error) {
	log.Printf("Processing specific source: %s", sourceID)

	// Get source configuration
	sourceConfig, err := dynamoService.GetSourceConfig(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source config: %w", err)
	}

	if sourceConfig.Status != models.SourceStatusActive {
		return nil, fmt.Errorf("source %s is not active (status: %s)", sourceID, sourceConfig.Status)
	}

	// Create scraping task for this source
	task, err := createScrapingTask(ctx, sourceConfig, taskType, triggerType)
	if err != nil {
		return nil, fmt.Errorf("failed to create scraping task: %w", err)
	}

	// Store the task in DynamoDB
	err = dynamoService.CreateScrapingTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to store scraping task: %w", err)
	}

	// Send task to SQS queue for execution
	err = sendTaskToQueue(ctx, task, sourceConfig)
	if err != nil {
		log.Printf("Failed to send scraping task for %s to queue: %v", sourceID, err)
		// Don't return error - log and continue
	}

	summary := TaskSummary{
		SourceID:     sourceConfig.SourceID,
		SourceName:   sourceConfig.SourceName,
		TaskType:     task.TaskType,
		Priority:     task.Priority,
		ScheduledFor: task.ScheduledTime,
		TaskID:       task.TaskID,
	}

	return []TaskSummary{summary}, nil
}

// processAllActiveSources creates scraping tasks for all active sources
func processAllActiveSources(ctx context.Context, taskType, triggerType string) ([]TaskSummary, error) {
	log.Printf("Processing all active sources")

	// Query all active sources
	activeSources, err := dynamoService.QuerySourcesByStatus(ctx, models.SourceStatusActive, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to query active sources: %w", err)
	}

	log.Printf("Found %d active sources", len(activeSources))

	var taskSummaries []TaskSummary

	for _, submission := range activeSources {
		// Get source configuration for each active source
		sourceConfig, err := dynamoService.GetSourceConfig(ctx, submission.SourceID)
		if err != nil {
			log.Printf("Failed to get config for source %s: %v", submission.SourceID, err)
			continue
		}

		// Check if source should be scraped based on frequency and last scrape time
		if !shouldScrapeSource(sourceConfig, triggerType) {
			log.Printf("Skipping source %s - not due for scraping", sourceConfig.SourceID)
			continue
		}

		// Determine task type if not specified
		actualTaskType := taskType
		if actualTaskType == "" {
			actualTaskType = determineTaskType(sourceConfig, triggerType)
		}

		// Create scraping task
		task, err := createScrapingTask(ctx, sourceConfig, actualTaskType, triggerType)
		if err != nil {
			log.Printf("Failed to create task for source %s: %v", sourceConfig.SourceID, err)
			continue
		}

		// Store the task in DynamoDB
		err = dynamoService.CreateScrapingTask(ctx, task)
		if err != nil {
			log.Printf("Failed to store task for source %s: %v", sourceConfig.SourceID, err)
			continue
		}

		// Send task to SQS queue for execution
		err = sendTaskToQueue(ctx, task, sourceConfig)
		if err != nil {
			log.Printf("Failed to send scraping task for %s to queue: %v", sourceConfig.SourceID, err)
			// Continue with other sources
		}

		summary := TaskSummary{
			SourceID:     sourceConfig.SourceID,
			SourceName:   sourceConfig.SourceName,
			TaskType:     task.TaskType,
			Priority:     task.Priority,
			ScheduledFor: task.ScheduledTime,
			TaskID:       task.TaskID,
		}

		taskSummaries = append(taskSummaries, summary)
	}

	return taskSummaries, nil
}

// shouldScrapeSource determines if a source should be scraped based on frequency and last scrape
func shouldScrapeSource(sourceConfig *models.DynamoSourceConfig, triggerType string) bool {
	// Always scrape for manual triggers
	if triggerType == "manual" || triggerType == "on-demand" {
		return true
	}

	// Check last successful scrape time
	lastScrape := sourceConfig.DataQuality.LastSuccessfulScrape
	if lastScrape.IsZero() {
		// Never scraped before, should scrape
		return true
	}

	// Determine scraping interval based on frequency
	var interval time.Duration
	switch sourceConfig.AdaptiveFrequency.CurrentFrequency {
	case "daily":
		interval = 24 * time.Hour
	case "weekly":
		interval = 7 * 24 * time.Hour
	case "monthly":
		interval = 30 * 24 * time.Hour
	default:
		interval = 24 * time.Hour // Default to daily
	}

	// Check if enough time has passed
	nextScrapeTime := lastScrape.Add(interval)
	return time.Now().After(nextScrapeTime)
}

// determineTaskType determines the appropriate task type based on source state
func determineTaskType(sourceConfig *models.DynamoSourceConfig, triggerType string) string {
	// If this is the first scrape, do a full scrape
	if sourceConfig.DataQuality.LastSuccessfulScrape.IsZero() {
		return models.TaskTypeFullScrape
	}

	// If there have been recent failures, do validation
	if sourceConfig.DataQuality.ConsecutiveFailures > 3 {
		return models.TaskTypeValidation
	}

	// For scheduled triggers, typically do incremental
	if triggerType == "scheduled" {
		return models.TaskTypeIncremental
	}

	// Default to full scrape
	return models.TaskTypeFullScrape
}

// createScrapingTask creates a new scraping task for a source
func createScrapingTask(ctx context.Context, sourceConfig *models.DynamoSourceConfig, taskType, triggerType string) (*models.ScrapingTask, error) {
	taskID := generateTaskID(sourceConfig.SourceID, taskType)
	
	task := &models.ScrapingTask{
		TaskID:       taskID,
		SourceID:     sourceConfig.SourceID,
		TaskType:     taskType,
		Priority:     sourceConfig.ScrapingConfig.Priority,
		ScheduledTime: time.Now(),
		TargetURLs:   sourceConfig.TargetURLs,
		ExtractionRules: sourceConfig.ContentSelectors,
		Status:       models.TaskStatusScheduled,
		MaxRetries:   sourceConfig.ScrapingConfig.MaxRetries,
		Timeout:      sourceConfig.ScrapingConfig.Timeout,
	}

	// Set primary keys
	task.PK = models.CreateTaskPK(taskID)
	task.SK = models.CreateTaskSK(task.Priority, task.SourceID, taskID)

	return task, nil
}

// executeScrapingTask performs the actual scraping work
func executeScrapingTask(ctx context.Context, task *models.ScrapingTask, sourceConfig *models.DynamoSourceConfig) error {
	log.Printf("Executing scraping task %s for source %s", task.TaskID, task.SourceID)

	// Update task status to in_progress
	task.Status = models.TaskStatusInProgress
	task.UpdatedAt = time.Now()
	
	err := dynamoService.CreateScrapingTask(ctx, task)
	if err != nil {
		log.Printf("Failed to update task status: %v", err)
	}

	// Create scraping execution record
	execution := &models.ScrapingExecution{
		ExecutionID: generateExecutionID(task.TaskID),
		TaskID:      task.TaskID,
		SourceID:    task.SourceID,
		StartedAt:   time.Now(),
		Status:      "in_progress",
	}
	execution.PK = "EXECUTION#" + execution.ExecutionID
	execution.SK = "TASK#" + task.TaskID

	var allActivities []models.Activity
	var totalErrors []string

	// Scrape each target URL
	for _, url := range task.TargetURLs {
		log.Printf("Scraping URL: %s", url)
		
		// Extract content using Jina
		content, err := jinaClient.ExtractContent(url)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to extract content from %s: %v", url, err)
			log.Printf(errorMsg)
			totalErrors = append(totalErrors, errorMsg)
			continue
		}

		// Extract activities using OpenAI
		activities, err := extractActivitiesFromContent(ctx, content, url, sourceConfig)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to extract activities from %s: %v", url, err)
			log.Printf(errorMsg)
			totalErrors = append(totalErrors, errorMsg)
			continue
		}

		allActivities = append(allActivities, activities...)
		log.Printf("Extracted %d activities from %s", len(activities), url)
	}

	// Update execution record
	execution.CompletedAt = time.Now()
	execution.Duration = execution.CompletedAt.Sub(execution.StartedAt).Milliseconds()
	execution.ItemsExtracted = len(allActivities)
	execution.ItemsProcessed = len(allActivities)
	execution.ItemsStored = len(allActivities)
	execution.ErrorCount = len(totalErrors)

	// Convert error strings to ExecutionError structs
	var executionErrors []models.ExecutionError
	for _, errMsg := range totalErrors {
		executionErrors = append(executionErrors, models.ExecutionError{
			Type:    "error",
			Code:    "SCRAPING_ERROR",
			Message: errMsg,
		})
	}
	execution.Errors = executionErrors

	if len(totalErrors) == 0 {
		execution.Status = "completed"
		task.Status = models.TaskStatusCompleted
	} else if len(allActivities) > 0 {
		execution.Status = "completed_with_errors"
		task.Status = models.TaskStatusCompleted
	} else {
		execution.Status = "failed"
		task.Status = models.TaskStatusFailed
	}

	// Store activities in DynamoDB family-activities table
	err = storeActivitiesInDynamoDB(ctx, allActivities, task.SourceID)
	if err != nil {
		log.Printf("Failed to store activities in DynamoDB: %v", err)
		totalErrors = append(totalErrors, fmt.Sprintf("Storage failed: %v", err))
	}

	// Also store in S3 for compatibility with existing frontend
	err = storeActivitiesInS3(ctx, allActivities)
	if err != nil {
		log.Printf("Failed to store activities in S3: %v", err)
		// Don't fail the task for S3 storage issues
	}

	// Update task completion  
	task.UpdatedAt = time.Now()

	// Store final task state
	err = dynamoService.CreateScrapingTask(ctx, task)
	if err != nil {
		log.Printf("Failed to update final task status: %v", err)
	}

	// Update source data quality metrics
	updateSourceQualityMetrics(ctx, sourceConfig, execution)

	log.Printf("Completed scraping task %s: found %d activities, %d errors", 
		task.TaskID, len(allActivities), len(totalErrors))

	return nil
}

// extractActivitiesFromContent uses OpenAI to extract activities from content
func extractActivitiesFromContent(ctx context.Context, content, url string, sourceConfig *models.DynamoSourceConfig) ([]models.Activity, error) {
	// Use the existing OpenAI client to extract activities
	response, err := openAIClient.ExtractActivities(content, url)
	if err != nil {
		return nil, fmt.Errorf("OpenAI extraction failed: %w", err)
	}

	// The activities are already enriched by the OpenAI client with source information
	return response.Activities, nil
}

// storeActivitiesInDynamoDB stores activities in the family-activities table
func storeActivitiesInDynamoDB(ctx context.Context, activities []models.Activity, sourceID string) error {
	for _, activity := range activities {
		// Convert Activity to FamilyActivity
		familyActivity := convertActivityToFamilyActivity(activity, sourceID)
		
		err := dynamoService.CreateFamilyActivity(ctx, familyActivity)
		if err != nil {
			log.Printf("Failed to store activity %s: %v", activity.Title, err)
			// Continue with other activities
		}
	}
	return nil
}

// storeActivitiesInS3 stores activities in S3 for frontend compatibility
func storeActivitiesInS3(ctx context.Context, activities []models.Activity) error {
	// Store in S3 using existing S3 client
	_, err := s3Client.UploadLatestActivities(activities)
	return err
}

// convertActivityToFamilyActivity converts the legacy Activity model to FamilyActivity
func convertActivityToFamilyActivity(activity models.Activity, sourceID string) *models.FamilyActivity {
	// Generate appropriate entity type and ID
	entityType := determineEntityType(activity)
	entityID := generateEntityID(activity, entityType)

	familyActivity := &models.FamilyActivity{
		PK:         entityType + "#" + entityID,
		SK:         models.SortKeyMetadata,
		EntityType: entityType,
		EntityID:   entityID,
		Name:       activity.Title,
		Description: activity.Description,
		Category:   activity.Category,
		Subcategory: activity.Subcategory,
		Location: models.ActivityLocation{
			Location: activity.Location,
			VenueType: "unknown", // Would need to be determined from content
		},
		AgeGroups: activity.AgeGroups,
		Pricing: models.ActivityPricing{
			Pricing: activity.Pricing,
		},
		ProviderID:   "PROVIDER#" + sourceID,
		ProviderName: "Source Provider", // Activity model doesn't have SourceName field
		Status:       "active",
		Featured:     false,
		SourceID:     sourceID,
	}

	return familyActivity
}

// Helper functions
func determineEntityType(activity models.Activity) string {
	// Simple heuristics to determine entity type
	title := strings.ToLower(activity.Title)
	description := strings.ToLower(activity.Description)
	
	if strings.Contains(title, "camp") || strings.Contains(description, "camp") {
		return models.EntityTypeProgram
	}
	if strings.Contains(title, "class") || strings.Contains(description, "class") {
		return models.EntityTypeProgram
	}
	// Check if it has recurring schedule patterns
	if activity.Schedule.Type == "recurring" || activity.Schedule.Frequency != "" {
		return models.EntityTypeProgram
	}
	
	return models.EntityTypeEvent // Default
}

func generateEntityID(activity models.Activity, entityType string) string {
	// Generate stable ID based on title and date
	base := strings.ToLower(activity.Title)
	base = regexp.MustCompile(`[^a-z0-9\s]`).ReplaceAllString(base, "")
	base = strings.ReplaceAll(base, " ", "-")
	
	// Use the schedule start date if available
	if activity.Schedule.StartDate != "" {
		base += "-" + activity.Schedule.StartDate
	}
	
	return base[:min(50, len(base))]
}

func generateTaskID(sourceID, taskType string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s-%s", sourceID, taskType, timestamp)
}

func generateExecutionID(taskID string) string {
	timestamp := time.Now().Format("20060102-150405-000")
	return fmt.Sprintf("exec-%s-%s", taskID, timestamp)
}

func updateSourceQualityMetrics(ctx context.Context, sourceConfig *models.DynamoSourceConfig, execution *models.ScrapingExecution) {
	// Update data quality metrics based on execution results
	if execution.Status == "completed" {
		sourceConfig.DataQuality.LastSuccessfulScrape = execution.CompletedAt
		sourceConfig.DataQuality.ConsecutiveFailures = 0
		sourceConfig.DataQuality.TotalSuccessfulScrapes++
	} else {
		sourceConfig.DataQuality.ConsecutiveFailures++
		sourceConfig.DataQuality.TotalFailedScrapes++
	}

	sourceConfig.DataQuality.LastAttemptedScrape = execution.CompletedAt
	sourceConfig.LastModified = time.Now()

	// Update average items per scrape
	total := sourceConfig.DataQuality.TotalSuccessfulScrapes + sourceConfig.DataQuality.TotalFailedScrapes
	if total > 0 {
		currentAvg := sourceConfig.DataQuality.AverageItemsPerScrape
		newAvg := (currentAvg*float64(total-1) + float64(execution.ItemsExtracted)) / float64(total)
		sourceConfig.DataQuality.AverageItemsPerScrape = newAvg
	}

	// Calculate reliability score
	if total > 0 {
		successRate := float64(sourceConfig.DataQuality.TotalSuccessfulScrapes) / float64(total)
		sourceConfig.DataQuality.ReliabilityScore = successRate
	}

	// Store updated source config
	err := dynamoService.CreateSourceConfig(ctx, sourceConfig)
	if err != nil {
		log.Printf("Failed to update source quality metrics: %v", err)
	}
}

// sendTaskToQueue sends a scraping task to the SQS queue for execution
func sendTaskToQueue(ctx context.Context, task *models.ScrapingTask, sourceConfig *models.DynamoSourceConfig) error {
	// Create task message for SQS
	taskMessage := map[string]interface{}{
		"task_id":       task.TaskID,
		"source_id":     task.SourceID,
		"source_name":   sourceConfig.SourceName,
		"base_url":      sourceConfig.BaseURL,
		"task_type":     task.TaskType,
		"priority":      task.Priority,
		"config":        sourceConfig,
		"scheduled_for": task.ScheduledTime,
	}

	// Marshal to JSON
	messageBody, err := json.Marshal(taskMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal task message: %w", err)
	}

	// Send message to SQS
	_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &taskQueueURL,
		MessageBody: aws.String(string(messageBody)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"TaskType": {
				DataType:    aws.String("String"),
				StringValue: aws.String(task.TaskType),
			},
			"Priority": {
				DataType:    aws.String("String"),
				StringValue: aws.String(task.Priority),
			},
			"SourceID": {
				DataType:    aws.String("String"),
				StringValue: aws.String(task.SourceID),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to send message to SQS: %w", err)
	}

	// Update task status to queued
	task.Status = models.TaskStatusQueued
	task.UpdatedAt = time.Now()
	if updateErr := dynamoService.UpdateScrapingTask(ctx, task); updateErr != nil {
		log.Printf("Failed to update task status to queued: %v", updateErr)
		// Don't return error - task was queued successfully
	}

	log.Printf("Successfully queued task %s for source %s", task.TaskID, task.SourceID)
	return nil
}

// Helper functions for converting to legacy format (simplified)
func convertToLegacyFormat(activities []models.Activity) interface{} {
	// This would implement the conversion logic from the existing system
	// For now, return the activities as-is
	return map[string]interface{}{
		"events":     activities,
		"activities": []interface{}{},
		"venues":     []interface{}{},
		"timestamp":  time.Now().Format(time.RFC3339),
	}
}

// Response helpers
func createSuccessResponse(body ResponseBody) (ScrapingOrchestratorResponse, error) {
	bodyBytes, _ := json.Marshal(body)
	return ScrapingOrchestratorResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(bodyBytes),
	}, nil
}

func createErrorResponse(statusCode int, message string) (ScrapingOrchestratorResponse, error) {
	body := ResponseBody{
		Success: false,
		Error:   message,
	}
	bodyBytes, _ := json.Marshal(body)
	return ScrapingOrchestratorResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(bodyBytes),
	}, nil
}

// Helper function for min since it's not available in older Go versions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	lambda.Start(handleRequest)
}