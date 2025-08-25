package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	lambdaclient "github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/google/uuid"

	"seattle-family-activities-scraper/internal/models"
	"seattle-family-activities-scraper/internal/services"
)

// AdminAPIResponse represents the Lambda response
type AdminAPIResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// ResponseBody represents the response body structure
type ResponseBody struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SourceSubmissionRequest represents the request for submitting a new source
type SourceSubmissionRequest struct {
	SourceName      string   `json:"source_name"`
	BaseURL         string   `json:"base_url"`
	SourceType      string   `json:"source_type"`
	Priority        string   `json:"priority"`
	ExpectedContent []string `json:"expected_content"`
	HintURLs        []string `json:"hint_urls"`
	SubmittedBy     string   `json:"submitted_by"`
}

// SourceActivationRequest represents the request for activating a source
type SourceActivationRequest struct {
	AdminNotes     string                 `json:"admin_notes"`
	OverrideConfig map[string]interface{} `json:"override_config,omitempty"`
}

var (
	dynamoService    *services.DynamoDBService
	lambdaClient     *lambdaclient.Client
	sourceAnalyzerFunctionName string
)

func init() {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Initialize DynamoDB client
	dynamoClient := dynamodb.NewFromConfig(cfg)

	// Get table names from environment
	familyActivitiesTable := os.Getenv("FAMILY_ACTIVITIES_TABLE")
	sourceManagementTable := os.Getenv("SOURCE_MANAGEMENT_TABLE")
	scrapingOperationsTable := os.Getenv("SCRAPING_OPERATIONS_TABLE")

	if familyActivitiesTable == "" || sourceManagementTable == "" || scrapingOperationsTable == "" {
		log.Fatal("Required environment variables not set: FAMILY_ACTIVITIES_TABLE, SOURCE_MANAGEMENT_TABLE, SCRAPING_OPERATIONS_TABLE")
	}

	// Initialize DynamoDB service
	dynamoService = services.NewDynamoDBService(
		dynamoClient,
		familyActivitiesTable,
		sourceManagementTable,
		scrapingOperationsTable,
	)

	// Initialize Lambda client for triggering source analyzer
	lambdaClient = lambdaclient.NewFromConfig(cfg)
	sourceAnalyzerFunctionName = os.Getenv("SOURCE_ANALYZER_FUNCTION_NAME")
	if sourceAnalyzerFunctionName == "" {
		log.Fatal("SOURCE_ANALYZER_FUNCTION_NAME environment variable not set")
	}
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (AdminAPIResponse, error) {
	// Set CORS headers
	headers := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
		"Access-Control-Allow-Methods": "GET,POST,PUT,DELETE,OPTIONS",
		"Content-Type":                 "application/json",
	}

	// Handle preflight OPTIONS request
	if request.HTTPMethod == "OPTIONS" {
		return AdminAPIResponse{
			StatusCode: 200,
			Headers:    headers,
			Body:       "",
		}, nil
	}

	// Route requests based on path and method
	path := request.Path
	method := request.HTTPMethod

	log.Printf("Admin API request: %s %s", method, path)

	var responseBody ResponseBody
	var statusCode int

	switch {
	case method == "POST" && path == "/api/sources/submit":
		responseBody, statusCode = handleSourceSubmission(ctx, request.Body)

	case method == "GET" && path == "/api/sources/pending":
		responseBody, statusCode = handleGetPendingSources(ctx, request.QueryStringParameters)

	case method == "GET" && path == "/api/sources/active":
		responseBody, statusCode = handleGetActiveSources(ctx, request.QueryStringParameters)

	case method == "GET" && strings.HasPrefix(path, "/api/sources/") && strings.HasSuffix(path, "/analysis"):
		sourceID := extractSourceIDFromPath(path, "/analysis")
		responseBody, statusCode = handleGetAnalysis(ctx, sourceID)

	case method == "GET" && strings.HasPrefix(path, "/api/sources/") && strings.HasSuffix(path, "/details"):
		sourceID := extractSourceIDFromPath(path, "/details")
		responseBody, statusCode = handleGetSourceDetails(ctx, sourceID, request.QueryStringParameters)

	case method == "POST" && strings.HasPrefix(path, "/api/sources/") && strings.HasSuffix(path, "/trigger"):
		sourceID := extractSourceIDFromPath(path, "/trigger")
		responseBody, statusCode = handleTriggerManualScrape(ctx, sourceID, request.Body)

	case method == "PUT" && strings.HasPrefix(path, "/api/sources/") && strings.HasSuffix(path, "/activate"):
		sourceID := extractSourceIDFromPath(path, "/activate")
		responseBody, statusCode = handleActivateSource(ctx, sourceID, request.Body)

	case method == "PUT" && strings.HasPrefix(path, "/api/sources/") && strings.HasSuffix(path, "/reject"):
		sourceID := extractSourceIDFromPath(path, "/reject")
		responseBody, statusCode = handleRejectSource(ctx, sourceID, request.Body)

	case method == "GET" && path == "/api/analytics":
		responseBody, statusCode = handleGetAnalytics(ctx, request.QueryStringParameters)

	default:
		responseBody = ResponseBody{
			Success: false,
			Error:   "Not found",
		}
		statusCode = 404
	}

	// Marshal response body
	bodyJSON, err := json.Marshal(responseBody)
	if err != nil {
		log.Printf("Error marshaling response body: %v", err)
		return AdminAPIResponse{
			StatusCode: 500,
			Headers:    headers,
			Body:       `{"success":false,"error":"Internal server error"}`,
		}, nil
	}

	return AdminAPIResponse{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       string(bodyJSON),
	}, nil
}

// extractSourceIDFromPath extracts source ID from path like /api/sources/{id}/analysis
func extractSourceIDFromPath(path, suffix string) string {
	// Remove /api/sources/ prefix and suffix
	withoutPrefix := strings.TrimPrefix(path, "/api/sources/")
	sourceID := strings.TrimSuffix(withoutPrefix, suffix)
	return sourceID
}

// handleSourceSubmission handles POST /api/sources/submit
func handleSourceSubmission(ctx context.Context, body string) (ResponseBody, int) {
	var req SourceSubmissionRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		}, 400
	}

	// Generate source ID
	sourceID := generateSourceID(req.SourceName)

	// Create SourceSubmission record
	submission := &models.SourceSubmission{
		PK:              models.CreateSourcePK(sourceID),
		SK:              models.CreateSourceSubmissionSK(),
		SourceID:        sourceID,
		SourceName:      req.SourceName,
		BaseURL:         req.BaseURL,
		SourceType:      req.SourceType,
		Priority:        req.Priority,
		ExpectedContent: req.ExpectedContent,
		HintURLs:        req.HintURLs,
		SubmittedBy:     req.SubmittedBy,
		SubmittedAt:     time.Now(),
		Status:          models.SourceStatusPendingAnalysis,
		StatusKey:       models.GenerateSourceStatusKey(models.SourceStatusPendingAnalysis),
		PriorityKey:     models.GenerateSourcePriorityKey(req.Priority, sourceID),
	}

	// Validate submission
	if err := submission.Validate(); err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Validation error: " + err.Error(),
		}, 400
	}

	// Store submission in DynamoDB
	if err := dynamoService.CreateSourceSubmission(ctx, submission); err != nil {
		log.Printf("Error creating source submission: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to store source submission",
		}, 500
	}

	// Automatically trigger source analyzer Lambda
	if err := triggerSourceAnalyzer(ctx, sourceID); err != nil {
		log.Printf("Error triggering source analyzer: %v", err)
		// Don't fail the request, just log the error
		// The admin can manually trigger analysis later
	}

	return ResponseBody{
		Success: true,
		Message: "Source submitted successfully and analysis started",
		Data: map[string]string{
			"source_id": sourceID,
		},
	}, 201
}

// handleGetPendingSources handles GET /api/sources/pending
func handleGetPendingSources(ctx context.Context, queryParams map[string]string) (ResponseBody, int) {
	limit := int32(50)
	if limitStr, ok := queryParams["limit"]; ok {
		// Parse limit (simplified, should add proper validation)
		if parsedLimit := parseLimit(limitStr); parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get sources with pending_analysis status
	pendingSources, err := dynamoService.QuerySourcesByStatus(ctx, models.SourceStatusPendingAnalysis, limit/2)
	if err != nil {
		log.Printf("Error querying pending sources: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to retrieve pending sources",
		}, 500
	}

	// Get sources with analysis_complete status
	analysisCompleteSources, err := dynamoService.QuerySourcesByStatus(ctx, models.SourceStatusAnalysisComplete, limit/2)
	if err != nil {
		log.Printf("Error querying analysis complete sources: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to retrieve analysis complete sources",
		}, 500
	}

	// Combine results
	allSources := append(pendingSources, analysisCompleteSources...)

	return ResponseBody{
		Success: true,
		Message: "Pending sources retrieved successfully",
		Data:    allSources,
	}, 200
}

// handleGetActiveSources handles GET /api/sources/active
func handleGetActiveSources(ctx context.Context, queryParams map[string]string) (ResponseBody, int) {
	limit := int32(50)
	if limitStr, ok := queryParams["limit"]; ok {
		if parsedLimit := parseLimit(limitStr); parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get active sources
	activeSources, err := dynamoService.QuerySourcesByStatus(ctx, models.SourceStatusActive, limit)
	if err != nil {
		log.Printf("Error querying active sources: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to retrieve active sources",
		}, 500
	}

	// Enhance each source with analytics data
	var enhancedSources []map[string]interface{}
	for _, source := range activeSources {
		enhancedSource, err := enhanceSourceWithAnalytics(ctx, &source)
		if err != nil {
			log.Printf("Error enhancing source %s: %v", source.SourceID, err)
			// Continue with basic data if enhancement fails
			enhancedSource = map[string]interface{}{
				"source_id":         source.SourceID,
				"source_name":       source.SourceName,
				"base_url":          source.BaseURL,
				"source_type":       source.SourceType,
				"status":            source.Status,
				"submitted_at":      source.SubmittedAt,
				"success_rate":      0,
				"activities_found":  0,
				"last_scraped":      nil,
				"scraping_status":   "unknown",
				"scraping_frequency": "daily",
			}
		}
		enhancedSources = append(enhancedSources, enhancedSource)
	}

	return ResponseBody{
		Success: true,
		Message: "Active sources retrieved successfully",
		Data:    enhancedSources,
	}, 200
}

// enhanceSourceWithAnalytics adds performance metrics and status to a source
func enhanceSourceWithAnalytics(ctx context.Context, source *models.SourceSubmission) (map[string]interface{}, error) {

	// Get recent scraping tasks for this source
	recentTasks, err := dynamoService.GetRecentTasksForSource(ctx, source.SourceID, 5)
	var scrapingStatus string
	var lastScraped *time.Time

	if err != nil {
		log.Printf("Could not get recent tasks for %s: %v", source.SourceID, err)
		scrapingStatus = "unknown"
	} else {
		// Determine current scraping status from recent tasks
		scrapingStatus = determineScrapingStatus(recentTasks)
		if len(recentTasks) > 0 {
			lastScraped = &recentTasks[0].UpdatedAt
		}
	}

	// Build enhanced source object
	enhanced := map[string]interface{}{
		"source_id":            source.SourceID,
		"source_name":          source.SourceName,
		"base_url":             source.BaseURL,
		"source_type":          source.SourceType,
		"status":               source.Status,
		"submitted_at":         source.SubmittedAt,
		"activated_at":         source.UpdatedAt, // When status changed to active
		
		// Performance metrics (placeholder values for now)
		"success_rate":         0.0,
		"activities_found":     0,
		"total_scrapes":        len(recentTasks),
		"successful_scrapes":   0,
		"avg_activities":       0.0,
		"last_scraped":         lastScraped,
		
		// Current status and configuration
		"scraping_status":      scrapingStatus,
		"scraping_frequency":   "daily",
		"next_scheduled":       time.Now().Add(24 * time.Hour),
		
		// Task management
		"recent_task_count":    len(recentTasks),
		"has_failed_tasks":     hasFailedTasks(recentTasks),
	}

	return enhanced, nil
}

// determineScrapingStatus analyzes recent tasks to determine current status
func determineScrapingStatus(tasks []models.ScrapingTask) string {
	if len(tasks) == 0 {
		return "ready"
	}
	
	// Check most recent task
	latest := tasks[0]
	switch latest.Status {
	case models.TaskStatusInProgress:
		return "running"
	case models.TaskStatusQueued:
		return "queued"
	case models.TaskStatusCompleted:
		return "completed"
	case models.TaskStatusFailed:
		return "failed"
	default:
		return "ready"
	}
}

// hasFailedTasks checks if any recent tasks failed
func hasFailedTasks(tasks []models.ScrapingTask) bool {
	for _, task := range tasks {
		if task.Status == models.TaskStatusFailed {
			return true
		}
	}
	return false
}

// calculateNextScheduled estimates when the next scrape should occur
func calculateNextScheduled(config *models.DynamoSourceConfig) *time.Time {
	// For now, return a simple 24-hour interval
	next := time.Now().Add(24 * time.Hour)
	return &next
}

// handleGetAnalysis handles GET /api/sources/{id}/analysis
func handleGetAnalysis(ctx context.Context, sourceID string) (ResponseBody, int) {
	analysis, err := dynamoService.GetSourceAnalysis(ctx, sourceID)
	if err != nil {
		log.Printf("Error getting source analysis: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Analysis not found",
		}, 404
	}

	return ResponseBody{
		Success: true,
		Message: "Analysis retrieved successfully",
		Data:    analysis,
	}, 200
}

// handleActivateSource handles PUT /api/sources/{id}/activate
func handleActivateSource(ctx context.Context, sourceID string, body string) (ResponseBody, int) {
	var req SourceActivationRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		}, 400
	}

	// Get source analysis to ensure it's complete
	analysis, err := dynamoService.GetSourceAnalysis(ctx, sourceID)
	if err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Source analysis not found",
		}, 404
	}

	if analysis.Status != "analysis_complete" {
		return ResponseBody{
			Success: false,
			Error:   "Source analysis must be complete before activation",
		}, 400
	}

	// Create DynamoSourceConfig from analysis recommendations
	config, err := createSourceConfigFromAnalysis(ctx, sourceID, analysis, req.AdminNotes)
	if err != nil {
		log.Printf("Error creating source config from analysis: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to create source configuration",
		}, 500
	}

	// Store source configuration
	if err := dynamoService.CreateSourceConfig(ctx, config); err != nil {
		log.Printf("Error creating source config: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to activate source",
		}, 500
	}

	// Create initial scraping task
	if err := createInitialScrapingTask(ctx, sourceID, analysis); err != nil {
		log.Printf("Error creating initial scraping task: %v", err)
		// Don't fail activation, just log the error
	}

	return ResponseBody{
		Success: true,
		Message: "Source activated successfully",
		Data: map[string]string{
			"source_id": sourceID,
			"status":    "active",
		},
	}, 200
}

// handleRejectSource handles PUT /api/sources/{id}/reject
func handleRejectSource(ctx context.Context, sourceID string, body string) (ResponseBody, int) {
	// Update source submission status to rejected
	submission, err := dynamoService.GetSourceSubmission(ctx, sourceID)
	if err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Source submission not found",
		}, 404
	}

	submission.Status = models.SourceStatusRejected
	submission.StatusKey = models.GenerateSourceStatusKey(models.SourceStatusRejected)

	if err := dynamoService.UpdateSourceSubmission(ctx, submission); err != nil {
		log.Printf("Error updating source submission: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to reject source",
		}, 500
	}

	return ResponseBody{
		Success: true,
		Message: "Source rejected successfully",
		Data: map[string]string{
			"source_id": sourceID,
			"status":    "rejected",
		},
	}, 200
}

// handleGetAnalytics handles GET /api/analytics
func handleGetAnalytics(ctx context.Context, queryParams map[string]string) (ResponseBody, int) {
	// For MVP, return mock analytics data
	// In production, this would query source metrics from DynamoDB
	analytics := map[string]interface{}{
		"total_sources_submitted": 12,
		"sources_pending_analysis": 3,
		"sources_active":          6,
		"sources_rejected":        3,
		"avg_analysis_time":       "3.2 minutes",
		"success_rate":            "75%",
	}

	return ResponseBody{
		Success: true,
		Message: "Analytics retrieved successfully",
		Data:    analytics,
	}, 200
}

// Helper functions

func generateSourceID(sourceName string) string {
	// Create a URL-safe ID from source name + UUID
	baseID := strings.ToLower(sourceName)
	baseID = strings.ReplaceAll(baseID, " ", "-")
	baseID = strings.ReplaceAll(baseID, "&", "and")
	
	// Remove special characters
	var cleanID strings.Builder
	for _, r := range baseID {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			cleanID.WriteRune(r)
		}
	}
	
	// Add UUID suffix to ensure uniqueness
	shortUUID := uuid.New().String()[:8]
	return cleanID.String() + "-" + shortUUID
}

func triggerSourceAnalyzer(ctx context.Context, sourceID string) error {
	payload := map[string]interface{}{
		"source_id":    sourceID,
		"trigger_type": "automatic",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = lambdaClient.Invoke(ctx, &lambdaclient.InvokeInput{
		FunctionName:   aws.String(sourceAnalyzerFunctionName),
		InvocationType: "Event", // Async invocation
		Payload:        payloadBytes,
	})

	return err
}

func createSourceConfigFromAnalysis(ctx context.Context, sourceID string, analysis *models.SourceAnalysis, adminNotes string) (*models.DynamoSourceConfig, error) {
	// Get the original source submission to populate fields
	submission, err := dynamoService.GetSourceSubmission(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source submission: %w", err)
	}

	now := time.Now()
	
	return &models.DynamoSourceConfig{
		PK:         models.CreateSourcePK(sourceID),
		SK:         models.CreateSourceConfigSK(),
		SourceID:   sourceID,
		SourceName: submission.SourceName,
		SourceType: submission.SourceType,
		BaseURL:    submission.BaseURL,
		TargetURLs: analysis.RecommendedConfig.TargetURLs,
		ContentSelectors: analysis.RecommendedConfig.BestSelectors,
		ScrapingConfig: models.DynamoScrapingConfig{
			Frequency:         analysis.RecommendedConfig.ScrapingFrequency,
			Priority:          "medium",
			RateLimit:         analysis.RecommendedConfig.RateLimit,
			UserAgent:         "SeattleFamilyActivities/1.0",
			RespectRobotsTxt:  true,
			Timeout:           30,
			MaxRetries:        3,
			BackoffMultiplier: 2.0,
		},
		DataQuality: models.DataQuality{
			ReliabilityScore: analysis.OverallQualityScore,
			ExpectedItemsRange: models.ItemRange{
				Min: 5,
				Max: 50,
			},
		},
		AdaptiveFrequency: models.AdaptiveFrequency{
			BaseFrequency:    analysis.RecommendedConfig.ScrapingFrequency,
			CurrentFrequency: analysis.RecommendedConfig.ScrapingFrequency,
		},
		Status:       models.SourceStatusActive,
		ActivatedBy:  "admin",
		ActivatedAt:  now,
		LastModified: now,
		StatusKey:    models.GenerateSourceStatusKey(models.SourceStatusActive),
	}, nil
}

func createInitialScrapingTask(ctx context.Context, sourceID string, analysis *models.SourceAnalysis) error {
	taskID := uuid.New().String()
	now := time.Now()

	task := &models.ScrapingTask{
		PK:            models.CreateTaskPK(taskID),
		SK:            models.CreateTaskSK("high", sourceID, taskID),
		TaskID:        taskID,
		SourceID:      sourceID,
		TaskType:      models.TaskTypeFullScrape,
		Priority:      models.TaskPriorityHigh,
		ScheduledTime: now.Add(5 * time.Minute), // Schedule 5 minutes from now
		TargetURLs:    analysis.RecommendedConfig.TargetURLs,
		ExtractionRules: analysis.RecommendedConfig.BestSelectors,
		RateLimits:      analysis.RecommendedConfig.RateLimit,
		Timeout:         300, // 5 minutes
		MaxRetries:      3,
		Status:          models.TaskStatusScheduled,
		RetryCount:      0,
		EstimatedDuration: 120, // 2 minutes
		Dependencies:      []string{},
		CreatedAt:         now,
		UpdatedAt:         now,
		TTL:               models.CalculateTaskTTL(now, 90), // 90 days retention
		NextRunKey:        models.GenerateNextRunKey(now.Add(5 * time.Minute)),
		PrioritySourceKey: models.GenerateTaskPrioritySourceKey("high", sourceID),
	}

	return dynamoService.CreateScrapingTask(ctx, task)
}

// handleGetSourceDetails handles GET /api/sources/{id}/details
func handleGetSourceDetails(ctx context.Context, sourceID string, queryParams map[string]string) (ResponseBody, int) {
	// Validate source ID
	if sourceID == "" {
		return ResponseBody{
			Success: false,
			Error:   "Source ID is required",
		}, 400
	}

	log.Printf("Getting details for source: %s", sourceID)

	// Collect all data for this source
	sourceDetails := make(map[string]interface{})

	// 1. Get source submission info
	sourceSubmission, err := dynamoService.GetSourceSubmission(ctx, sourceID)
	if err != nil {
		log.Printf("Error getting source submission: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Source not found",
		}, 404
	}
	
	sourceDetails["source_info"] = map[string]interface{}{
		"source_id":         sourceSubmission.SourceID,
		"source_name":       sourceSubmission.SourceName,
		"base_url":          sourceSubmission.BaseURL,
		"source_type":       sourceSubmission.SourceType,
		"priority":          sourceSubmission.Priority,
		"expected_content":  sourceSubmission.ExpectedContent,
		"hint_urls":         sourceSubmission.HintURLs,
		"submitted_by":      sourceSubmission.SubmittedBy,
		"submitted_at":      sourceSubmission.SubmittedAt,
		"status":            sourceSubmission.Status,
		"updated_at":        sourceSubmission.UpdatedAt,
	}

	// 2. Get source analysis (if available)
	sourceAnalysis, err := dynamoService.GetSourceAnalysis(ctx, sourceID)
	if err != nil {
		log.Printf("No analysis found for source %s: %v", sourceID, err)
		sourceDetails["analysis"] = nil
	} else {
		sourceDetails["analysis"] = map[string]interface{}{
			"quality_score":         sourceAnalysis.OverallQualityScore,
			"content_richness":      0.0, // placeholder
			"extraction_confidence": 0.0, // placeholder
			"recommended_selectors": sourceAnalysis.RecommendedConfig.BestSelectors,
			"target_urls":          sourceAnalysis.RecommendedConfig.TargetURLs,
			"analysis_notes":       "Analysis completed", // placeholder
			"analyzed_at":          sourceAnalysis.AnalysisCompletedAt,
		}
	}

	// 3. Get source configuration (if active)
	sourceConfig, err := dynamoService.GetSourceConfig(ctx, sourceID)
	if err != nil {
		log.Printf("No config found for source %s: %v", sourceID, err)
		sourceDetails["config"] = nil
	} else {
		sourceDetails["config"] = map[string]interface{}{
			"scraping_frequency":       "daily",
			"success_rate":             0.0,
			"total_scrapes":            0,
			"successful_scrapes":       0,
			"total_activities_found":   0,
			"avg_activities_per_scrape": 0.0,
			"last_scraped":             nil,
			"content_selectors":        sourceConfig.ContentSelectors,
			"is_active":                true,
		}
	}

	// 4. Get task history
	taskLimit := 20
	if limitStr, ok := queryParams["task_limit"]; ok {
		if parsed := parseLimit(limitStr); parsed > 0 {
			taskLimit = int(parsed)
		}
	}

	taskHistory, err := dynamoService.GetRecentTasksForSource(ctx, sourceID, taskLimit)
	if err != nil {
		log.Printf("Error getting task history for %s: %v", sourceID, err)
		sourceDetails["task_history"] = []interface{}{}
	} else {
		tasks := make([]map[string]interface{}, len(taskHistory))
		for i, task := range taskHistory {
			tasks[i] = map[string]interface{}{
				"task_id":          task.TaskID,
				"task_type":        task.TaskType,
				"priority":         task.Priority,
				"status":           task.Status,
				"scheduled_time":   task.ScheduledTime,
				"created_at":       task.CreatedAt,
				"updated_at":       task.UpdatedAt,
				"retry_count":      task.RetryCount,
				"error_message":    "", // ErrorMessage field doesn't exist
				"estimated_duration": task.EstimatedDuration,
			}
		}
		sourceDetails["task_history"] = tasks
	}

	// 5. Get performance metrics summary
	if sourceConfig != nil {
		sourceDetails["performance"] = map[string]interface{}{
			"reliability_score":    calculateReliabilityScore(taskHistory),
			"avg_task_duration":    calculateAvgTaskDuration(taskHistory),
			"recent_failure_rate":  calculateRecentFailureRate(taskHistory),
			"last_successful_scrape": getLastSuccessfulScrape(taskHistory),
			"next_estimated_run":   calculateNextEstimatedRun(sourceConfig, taskHistory),
		}
	}

	// 6. Get recent activities extracted (last 50)
	// This would require a new method to get activities by source
	// For now, we'll add placeholder
	sourceDetails["recent_activities"] = map[string]interface{}{
		"count": 0,
		"activities": []interface{}{},
		"note": "Activity extraction details coming soon",
	}

	return ResponseBody{
		Success: true,
		Message: "Source details retrieved successfully",
		Data:    sourceDetails,
	}, 200
}

// Helper functions for source details
func calculateReliabilityScore(tasks []models.ScrapingTask) float64 {
	if len(tasks) == 0 {
		return 0.0
	}
	
	successful := 0
	for _, task := range tasks {
		if task.Status == models.TaskStatusCompleted {
			successful++
		}
	}
	
	return float64(successful) / float64(len(tasks)) * 100
}

func calculateAvgTaskDuration(tasks []models.ScrapingTask) int64 {
	if len(tasks) == 0 {
		return 0
	}
	
	totalDuration := int64(0)
	completedTasks := 0
	
	for _, task := range tasks {
		if task.Status == models.TaskStatusCompleted && task.EstimatedDuration > 0 {
			totalDuration += task.EstimatedDuration
			completedTasks++
		}
	}
	
	if completedTasks == 0 {
		return 0
	}
	
	return totalDuration / int64(completedTasks)
}

func calculateRecentFailureRate(tasks []models.ScrapingTask) float64 {
	// Look at last 10 tasks
	recentTasks := tasks
	if len(tasks) > 10 {
		recentTasks = tasks[:10]
	}
	
	if len(recentTasks) == 0 {
		return 0.0
	}
	
	failed := 0
	for _, task := range recentTasks {
		if task.Status == models.TaskStatusFailed {
			failed++
		}
	}
	
	return float64(failed) / float64(len(recentTasks)) * 100
}

func getLastSuccessfulScrape(tasks []models.ScrapingTask) *time.Time {
	for _, task := range tasks {
		if task.Status == models.TaskStatusCompleted {
			return &task.UpdatedAt
		}
	}
	return nil
}

func calculateNextEstimatedRun(config *models.DynamoSourceConfig, tasks []models.ScrapingTask) *time.Time {
	// Simple calculation: next run in 24 hours
	next := time.Now().Add(24 * time.Hour)
	return &next
}

// handleTriggerManualScrape handles POST /api/sources/{id}/trigger  
func handleTriggerManualScrape(ctx context.Context, sourceID string, body string) (ResponseBody, int) {
	// Validate source ID
	if sourceID == "" {
		return ResponseBody{
			Success: false,
			Error:   "Source ID is required",
		}, 400
	}

	log.Printf("Manual scrape triggered for source: %s", sourceID)

	// Parse optional request body for task configuration
	var req struct {
		TaskType string `json:"task_type,omitempty"` // full_scrape (default), incremental, validation
		Priority string `json:"priority,omitempty"` // high (default), medium, low  
		Notes    string `json:"notes,omitempty"`    // admin notes
	}
	
	if body != "" {
		if err := json.Unmarshal([]byte(body), &req); err != nil {
			log.Printf("Invalid request body for manual trigger: %v", err)
			// Continue with defaults if body is invalid
		}
	}

	// Set defaults
	if req.TaskType == "" {
		req.TaskType = models.TaskTypeFullScrape
	}
	if req.Priority == "" {
		req.Priority = models.TaskPriorityHigh
	}

	// Verify source exists and is active
	sourceSubmission, err := dynamoService.GetSourceSubmission(ctx, sourceID)
	if err != nil {
		log.Printf("Error getting source submission: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Source not found",
		}, 404
	}

	if sourceSubmission.Status != models.SourceStatusActive {
		return ResponseBody{
			Success: false,
			Error:   fmt.Sprintf("Source is not active (status: %s)", sourceSubmission.Status),
		}, 400
	}

	// Get source configuration to build proper task
	sourceConfig, err := dynamoService.GetSourceConfig(ctx, sourceID)
	if err != nil {
		log.Printf("Error getting source config: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Source configuration not found - source may not be properly activated",
		}, 400
	}

	// Create immediate scraping task
	taskID := uuid.New().String()
	now := time.Now()
	
	task := &models.ScrapingTask{
		PK:            models.CreateTaskPK(taskID),
		SK:            models.CreateTaskSK(req.Priority, sourceID, taskID),
		TaskID:        taskID,
		SourceID:      sourceID,
		TaskType:      req.TaskType,
		Priority:      req.Priority,
		ScheduledTime: now.Add(1 * time.Minute), // Run in 1 minute
		TargetURLs:    []string{sourceConfig.BaseURL},
		ExtractionRules: sourceConfig.ContentSelectors,
		RateLimits:      sourceConfig.ScrapingConfig.RateLimit,
		Timeout:         300, // 5 minutes
		MaxRetries:      2,   // Lower retries for manual tasks
		Status:          models.TaskStatusScheduled,
		RetryCount:      0,
		EstimatedDuration: 120, // 2 minutes
		Dependencies:      []string{},
		CreatedAt:         now,
		UpdatedAt:         now,
		TTL:               models.CalculateTaskTTL(now, 30), // 30 days retention for manual tasks
		NextRunKey:        models.GenerateNextRunKey(now.Add(1 * time.Minute)),
		PrioritySourceKey: models.GenerateTaskPrioritySourceKey(req.Priority, sourceID),
		// Note: ErrorMessage field doesn't exist in ScrapingTask
	}

	// Store the task in DynamoDB
	if err := dynamoService.CreateScrapingTask(ctx, task); err != nil {
		log.Printf("Error creating manual scraping task: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to create scraping task",
		}, 500
	}

	// Trigger the orchestrator to process the new task immediately
	// We can invoke the orchestrator Lambda directly for immediate processing
	if err := triggerOrchestratorForSource(ctx, sourceID, req.TaskType); err != nil {
		log.Printf("Error triggering orchestrator: %v", err)
		// Don't fail the request - task is created, orchestrator will pick it up on next run
	}

	return ResponseBody{
		Success: true,
		Message: "Manual scrape triggered successfully",
		Data: map[string]interface{}{
			"task_id":        taskID,
			"source_id":      sourceID,
			"task_type":      req.TaskType,
			"priority":       req.Priority,
			"scheduled_for":  task.ScheduledTime,
			"estimated_completion": now.Add(time.Duration(task.EstimatedDuration) * time.Second),
		},
	}, 201
}

// triggerOrchestratorForSource invokes the orchestrator Lambda for immediate processing
func triggerOrchestratorForSource(ctx context.Context, sourceID, taskType string) error {
	// Get orchestrator function name from environment
	orchestratorFunctionName := os.Getenv("ORCHESTRATOR_FUNCTION_NAME")
	if orchestratorFunctionName == "" {
		return fmt.Errorf("ORCHESTRATOR_FUNCTION_NAME not configured")
	}

	// Create event payload for orchestrator
	event := map[string]interface{}{
		"trigger_type": "manual",
		"source_id":    sourceID,
		"task_type":    taskType,
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal orchestrator event: %w", err)
	}

	// Invoke orchestrator Lambda asynchronously
	_, err = lambdaClient.Invoke(ctx, &lambdaclient.InvokeInput{
		FunctionName:   aws.String(orchestratorFunctionName),
		InvocationType: lambdatypes.InvocationTypeEvent, // Async invocation
		Payload:        eventBytes,
	})

	if err != nil {
		return fmt.Errorf("failed to invoke orchestrator: %w", err)
	}

	log.Printf("Successfully triggered orchestrator for source %s", sourceID)
	return nil
}

func parseLimit(limitStr string) int32 {
	// Simple parsing, should add proper validation
	switch limitStr {
	case "10":
		return 10
	case "25":
		return 25
	case "50":
		return 50
	case "100":
		return 100
	default:
		return 0
	}
}

func main() {
	lambda.Start(handleRequest)
}