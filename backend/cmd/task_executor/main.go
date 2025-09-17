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
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"seattle-family-activities-scraper/internal/models"
	"seattle-family-activities-scraper/internal/services"
)

// TaskExecutorEvent represents the SQS message payload for task execution
type TaskExecutorEvent struct {
	TaskID         string `json:"task_id"`
	SourceID       string `json:"source_id"`
	SourceName     string `json:"source_name"`
	BaseURL        string `json:"base_url"`
	TaskType       string `json:"task_type"`
	Priority       string `json:"priority"`
	ScheduledTime  string `json:"scheduled_time"`
	TargetURLs     []string `json:"target_urls"`
}

var (
	// AWS services
	dynamoService *services.DynamoDBService

	// Scraping services
	firecrawlClient *services.FireCrawlClient

	// Environment variables
	familyActivitiesTableName  = os.Getenv("FAMILY_ACTIVITIES_TABLE")
	sourceManagementTableName  = os.Getenv("SOURCE_MANAGEMENT_TABLE")
	scrapingOperationsTableName = os.Getenv("SCRAPING_OPERATIONS_TABLE")
	adminEventsTableName       = os.Getenv("ADMIN_EVENTS_TABLE")
	s3BucketName               = os.Getenv("S3_BUCKET_NAME")
)

func init() {
	// Initialize AWS config
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Initialize DynamoDB service
	dynamoClient := dynamodb.NewFromConfig(cfg)
	dynamoService = services.NewDynamoDBService(
		dynamoClient,
		familyActivitiesTableName,
		sourceManagementTableName,
		scrapingOperationsTableName,
		adminEventsTableName,
	)
	
	// Initialize FireCrawl client
	firecrawlClient, err = services.NewFireCrawlClient()
	if err != nil {
		log.Fatalf("Failed to create FireCrawl client: %v", err)
	}
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	log.Printf("Processing %d SQS messages", len(sqsEvent.Records))

	for _, record := range sqsEvent.Records {
		if err := processMessage(ctx, record); err != nil {
			log.Printf("Failed to process message %s: %v", record.MessageId, err)
			return err // This will cause the message to be retried
		}
	}

	return nil
}

func processMessage(ctx context.Context, record events.SQSMessage) error {
	log.Printf("Processing SQS message: %s", record.Body)

	var taskEvent TaskExecutorEvent
	if err := json.Unmarshal([]byte(record.Body), &taskEvent); err != nil {
		return fmt.Errorf("failed to unmarshal SQS message: %w", err)
	}

	// Execute the scraping task
	if err := executeScrapingTask(ctx, taskEvent); err != nil {
		log.Printf("Scraping task failed: %v", err)
		return err
	}

	log.Printf("Successfully completed task %s for source %s", taskEvent.TaskID, taskEvent.SourceName)
	return nil
}

func executeScrapingTask(ctx context.Context, taskEvent TaskExecutorEvent) error {
	log.Printf("Starting scraping task %s for source %s (%s)", 
		taskEvent.TaskID, taskEvent.SourceName, taskEvent.BaseURL)

	// Find the task in DynamoDB to update status
	task, err := getScrapingTask(ctx, taskEvent.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get scraping task: %w", err)
	}

	// Update status to in_progress
	if err := updateTaskStatus(ctx, task, models.TaskStatusInProgress, ""); err != nil {
		return fmt.Errorf("failed to update task status to in_progress: %w", err)
	}

	// Perform actual scraping
	activities, err := performScraping(ctx, taskEvent)
	if err != nil {
		log.Printf("Scraping failed: %v", err)
		if statusErr := updateTaskStatus(ctx, task, models.TaskStatusFailed, err.Error()); statusErr != nil {
			log.Printf("Failed to update task status to failed: %v", statusErr)
		}
		return fmt.Errorf("scraping failed: %w", err)
	}

	log.Printf("Successfully extracted %d activities from %s", len(activities), taskEvent.BaseURL)

	// Update status to completed
	if err := updateTaskStatus(ctx, task, models.TaskStatusCompleted, ""); err != nil {
		return fmt.Errorf("failed to update task status to completed: %w", err)
	}

	log.Printf("Task %s completed successfully", taskEvent.TaskID)
	return nil
}

func getScrapingTask(ctx context.Context, taskID string) (*models.ScrapingTask, error) {
	// Use a DynamoDB scan to find the task by TaskID
	// This is less efficient but more reliable than using GSI queries
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	dynamoClient := dynamodb.NewFromConfig(cfg)
	
	result, err := dynamoClient.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(scrapingOperationsTableName),
		FilterExpression: aws.String("task_id = :task_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":task_id": &types.AttributeValueMemberS{Value: taskID},
		},
		Limit: aws.Int32(10), // Should only be 1 task
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan scraping tasks: %w", err)
	}
	
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	
	var task models.ScrapingTask
	err = attributevalue.UnmarshalMap(result.Items[0], &task)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal scraping task: %w", err)
	}
	
	return &task, nil
}

func performScraping(ctx context.Context, taskEvent TaskExecutorEvent) ([]models.Activity, error) {
	var allActivities []models.Activity
	
	// Process each target URL
	for _, url := range taskEvent.TargetURLs {
		log.Printf("Extracting content from URL: %s", url)
		
		// Extract structured activities using FireCrawl
		response, err := firecrawlClient.ExtractActivities(url)
		if err != nil {
			log.Printf("Failed to extract activities from %s: %v", url, err)
			continue // Continue with other URLs
		}

		log.Printf("FireCrawl extracted %d activities from %s", len(response.Data.Activities), url)
		
		// Step 3: Convert to DynamoDB format and store
		for _, activity := range response.Data.Activities {
			// Convert Activity to FamilyActivity for DynamoDB storage
			// Use source name as fallback for sourceID if not provided
			sourceID := taskEvent.SourceID
			if sourceID == "" {
				sourceID = strings.ToLower(strings.ReplaceAll(taskEvent.SourceName, " ", "-"))
			}
			familyActivity := convertActivityToFamilyActivity(activity, sourceID, url)
			
			// Store in DynamoDB
			if err := dynamoService.CreateFamilyActivity(ctx, familyActivity); err != nil {
				log.Printf("Failed to store activity %s: %v", activity.Title, err)
				continue
			}
			
			log.Printf("Stored activity: %s", activity.Title)
			allActivities = append(allActivities, activity)
		}
	}
	
	return allActivities, nil
}

func convertActivityToFamilyActivity(activity models.Activity, sourceID, sourceURL string) *models.FamilyActivity {
	// Generate a unique ID for the family activity
	entityID := fmt.Sprintf("%s-%d", sourceID, time.Now().UnixNano())
	
	// Determine entity type from activity type
	entityType := models.EntityTypeEvent // Default to event
	switch activity.Type {
	case "venue":
		entityType = models.EntityTypeVenue
	case "program", "class", "camp":
		entityType = models.EntityTypeProgram
	case "attraction":
		entityType = models.EntityTypeAttraction
	default:
		entityType = models.EntityTypeEvent
	}
	
	// Create primary keys
	pk := entityType + "#" + entityID
	sk := models.SortKeyMetadata
	
	// Convert Activity to FamilyActivity with correct field mapping
	familyActivity := &models.FamilyActivity{
		PK:           pk,
		SK:           sk,
		EntityType:   entityType,
		EntityID:     entityID,
		Name:         activity.Title,
		Description:  activity.Description,
		Category:     activity.Category,
		Subcategory:  activity.Subcategory,
		AgeGroups:    activity.AgeGroups,
		ProviderID:   sourceID,
		ProviderName: strings.ReplaceAll(sourceID, "-", " "), // Convert ID back to readable name
		Status:       "active",
		Featured:     false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		SourceID:     sourceID,
	}
	
	// Convert location if available
	if activity.Location.Address != "" || activity.Location.City != "" {
		familyActivity.Location = models.ActivityLocation{
			Location: activity.Location, // Embed the full Location struct
		}
	}
	
	// Convert pricing if available
	if activity.Pricing.Type != "" {
		familyActivity.Pricing = models.ActivityPricing{
			Pricing: activity.Pricing, // Embed the full Pricing struct
		}
	}
	
	return familyActivity
}

func updateTaskStatus(ctx context.Context, task *models.ScrapingTask, status models.ScrapingTaskStatus, errorMessage string) error {
	task.Status = status
	task.UpdatedAt = time.Now()
	
	if err := dynamoService.UpdateScrapingTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	
	log.Printf("Updated task %s status to %s", task.TaskID, status)
	return nil
}