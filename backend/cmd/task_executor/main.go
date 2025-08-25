package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

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
	
	// Environment variables
	familyActivitiesTableName  = os.Getenv("FAMILY_ACTIVITIES_TABLE")
	sourceManagementTableName  = os.Getenv("SOURCE_MANAGEMENT_TABLE")
	scrapingOperationsTableName = os.Getenv("SCRAPING_OPERATIONS_TABLE")
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
	)
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

	// TODO: Implement actual scraping logic here
	// For now, simulate processing by waiting and then marking as completed
	log.Printf("Simulating scraping process for %s...", taskEvent.BaseURL)
	time.Sleep(2 * time.Second)

	// Update status to completed
	if err := updateTaskStatus(ctx, task, models.TaskStatusCompleted, ""); err != nil {
		return fmt.Errorf("failed to update task status to completed: %w", err)
	}

	log.Printf("Task %s completed successfully", taskEvent.TaskID)
	return nil
}

func getScrapingTask(ctx context.Context, taskID string) (*models.ScrapingTask, error) {
	// For now, create a minimal task object
	// TODO: Implement proper task retrieval from DynamoDB
	task := &models.ScrapingTask{
		TaskID: taskID,
		Status: models.TaskStatusQueued,
	}
	return task, nil
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