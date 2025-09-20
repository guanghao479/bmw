package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"seattle-family-activities-scraper/internal/models"
	"seattle-family-activities-scraper/internal/services"
)

// Simple Source struct for hardcoded sources
type Source struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	BaseURL    string   `json:"base_url"`
	TargetURLs []string `json:"target_urls"`
	Enabled    bool     `json:"enabled"`
	Priority   string   `json:"priority"`
	Category   string   `json:"category"`
}

// ScrapingOrchestratorEvent represents the input event for orchestrator
type ScrapingOrchestratorEvent struct {
	SourceID string `json:"source_id,omitempty"` // optional: scrape specific source
}

// ScrapingOrchestratorResponse represents the Lambda response
type ScrapingOrchestratorResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// ResponseBody structure
type ResponseBody struct {
	Success         bool     `json:"success"`
	Message         string   `json:"message"`
	TotalSources    int      `json:"total_sources"`
	ProcessedSources int     `json:"processed_sources"`
	TotalActivities int      `json:"total_activities"`
	ProcessingTime  int64    `json:"processing_time_ms"`
	Errors          []string `json:"errors,omitempty"`
}

var (
	dynamoService   *services.DynamoDBService
	firecrawlClient *services.FireCrawlClient
)

// Note: All sources are now managed dynamically through the admin interface
// No hardcoded sources - all sources come from DynamoDB

func init() {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create DynamoDB client and service (for storing results)
	dynamoClient := dynamodb.NewFromConfig(cfg)
	dynamoService = services.NewDynamoDBService(
		dynamoClient,
		os.Getenv("FAMILY_ACTIVITIES_TABLE"),
		os.Getenv("SOURCE_MANAGEMENT_TABLE"),
		os.Getenv("SCRAPING_OPERATIONS_TABLE"),
		os.Getenv("ADMIN_EVENTS_TABLE"),
	)

	// Create FireCrawl client
	firecrawlClient, err = services.NewFireCrawlClient()
	if err != nil {
		log.Fatalf("Failed to create FireCrawl client: %v", err)
	}
}

func handleRequest(ctx context.Context, event ScrapingOrchestratorEvent) (ScrapingOrchestratorResponse, error) {
	start := time.Now()

	log.Printf("Starting scraping orchestrator")

	var allActivities []models.Activity
	var errors []string
	processedSources := 0

	// Get active sources from DynamoDB
	sources, err := getActiveSources(ctx, event.SourceID)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get active sources: %v", err)
		log.Printf("ERROR: %s", errorMsg)
		return ScrapingOrchestratorResponse{
			StatusCode: 500,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: fmt.Sprintf(`{"success": false, "message": "%s"}`, errorMsg),
		}, nil
	}

	log.Printf("Processing %d sources", len(sources))

	// Process each source directly with FireCrawl
	for _, source := range sources {
		if !source.Enabled {
			log.Printf("Skipping disabled source: %s", source.Name)
			continue
		}

		log.Printf("Processing source: %s", source.Name)

		// Save source to DynamoDB if not already exists
		err := ensureSourceInDatabase(source)
		if err != nil {
			log.Printf("Warning: Failed to save source %s to database: %v", source.Name, err)
			// Continue processing even if database save fails
		}

		// Process each target URL for the source
		for _, url := range source.TargetURLs {
			log.Printf("Extracting activities from: %s", url)

			activities, err := extractActivitiesFromURL(url, source)
			if err != nil {
				errorMsg := fmt.Sprintf("Failed to extract from %s (%s): %v", source.Name, url, err)
				log.Printf("ERROR: %s", errorMsg)
				errors = append(errors, errorMsg)
				continue
			}

			log.Printf("Extracted %d activities from %s", len(activities), url)
			allActivities = append(allActivities, activities...)
		}

		processedSources++
	}

	log.Printf("Total activities extracted: %d", len(allActivities))

	// Note: Activities are now stored directly via admin API flow
	// The orchestrator extracts activities and they go through the admin approval process
	// No direct storage needed here - activities will be approved and served via database API
	if len(allActivities) > 0 {
		log.Printf("Extracted %d activities - these will be available via admin interface for review", len(allActivities))
	}

	processingTime := time.Since(start).Milliseconds()

	// Create response
	success := len(errors) == 0
	message := "Scraping completed successfully"
	if !success {
		message = fmt.Sprintf("Scraping completed with %d errors", len(errors))
	}

	responseBody := ResponseBody{
		Success:         success,
		Message:         message,
		TotalSources:    len(sources),
		ProcessedSources: processedSources,
		TotalActivities: len(allActivities),
		ProcessingTime:  processingTime,
		Errors:          errors,
	}

	bodyJSON, err := json.Marshal(responseBody)
	if err != nil {
		return ScrapingOrchestratorResponse{
			StatusCode: 500,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"success": false, "message": "Failed to marshal response"}`,
		}, err
	}

	statusCode := 200
	if !success {
		statusCode = 207 // Multi-status (partial success)
	}

	return ScrapingOrchestratorResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(bodyJSON),
	}, nil
}

// getActiveSources retrieves active sources from DynamoDB, optionally filtered by source ID
func getActiveSources(ctx context.Context, sourceID string) ([]Source, error) {
	if sourceID != "" {
		// Get specific source
		sourceSubmission, err := dynamoService.GetSourceSubmission(ctx, sourceID)
		if err != nil {
			return nil, fmt.Errorf("source %s not found: %w", sourceID, err)
		}
		if sourceSubmission.Status != "active" {
			return nil, fmt.Errorf("source %s is not active (status: %s)", sourceID, sourceSubmission.Status)
		}
		source := convertSourceSubmissionToSource(sourceSubmission)
		return []Source{source}, nil
	}

	// Get all active sources
	sourceSubmissions, err := dynamoService.QuerySourcesByStatus(ctx, "active", 50) // Limit to 50 sources
	if err != nil {
		return nil, fmt.Errorf("failed to query active sources: %w", err)
	}

	// Convert to Source format and filter enabled sources
	var sources []Source
	for _, submission := range sourceSubmissions {
		source := convertSourceSubmissionToSource(&submission)
		if source.Enabled {
			sources = append(sources, source)
		}
	}

	// If no sources found in DynamoDB, return empty slice
	if len(sources) == 0 {
		log.Printf("No active sources found in DynamoDB - sources must be added via admin interface")
		return []Source{}, nil
	}

	return sources, nil
}

// convertSourceSubmissionToSource converts a DynamoDB SourceSubmission to the Source format used by orchestrator
func convertSourceSubmissionToSource(submission *models.SourceSubmission) Source {
	return Source{
		ID:         submission.SourceID,
		Name:       submission.SourceName,
		BaseURL:    submission.BaseURL,
		TargetURLs: submission.HintURLs,
		Enabled:    submission.Status == "active", // Convert status to enabled flag
		Priority:   submission.Priority,
		Category:   determineCategory(submission.ExpectedContent),
	}
}

// determineCategory maps expected content to category
func determineCategory(expectedContent []string) string {
	if len(expectedContent) == 0 {
		return "events"
	}
	// Use first expected content type as category
	switch expectedContent[0] {
	case "classes":
		return "classes"
	case "activities":
		return "activities"
	case "local-events":
		return "local-events"
	default:
		return "events"
	}
}

func extractActivitiesFromURL(url string, source Source) ([]models.Activity, error) {
	// Use FireCrawl Extract API to get structured data
	response, err := firecrawlClient.ExtractActivities(url)
	if err != nil {
		return nil, fmt.Errorf("FireCrawl extraction failed: %w", err)
	}

	if response == nil || len(response.Data.Activities) == 0 {
		log.Printf("No activities extracted from %s", url)
		return []models.Activity{}, nil
	}

	// Add source metadata to each activity
	now := time.Now()
	for i := range response.Data.Activities {
		response.Data.Activities[i].Source = models.Source{
			URL:         url,
			Domain:      extractDomain(url),
			ScrapedAt:   now,
			LastChecked: now,
			Reliability: "medium",
		}
		response.Data.Activities[i].UpdatedAt = now
		if response.Data.Activities[i].CreatedAt.IsZero() {
			response.Data.Activities[i].CreatedAt = now
		}

		// Associate with source via Provider field
		response.Data.Activities[i].Provider = models.Provider{
			Name:    source.Name,
			Type:    "community-calendar",
			Website: source.BaseURL,
		}

		// Generate ID if not provided
		if response.Data.Activities[i].ID == "" {
			response.Data.Activities[i].ID = models.GenerateActivityID(
				response.Data.Activities[i].Title,
				response.Data.Activities[i].Schedule.StartDate,
				response.Data.Activities[i].Location.Name,
			)
		}
	}

	return response.Data.Activities, nil
}

// Note: S3 storage function removed - activities now flow through admin API for approval

func extractDomain(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		log.Printf("Warning: Failed to parse URL %s: %v", urlStr, err)
		return urlStr
	}
	return parsedURL.Host
}

// ensureSourceInDatabase saves the source to DynamoDB if it doesn't already exist
func ensureSourceInDatabase(source Source) error {
	ctx := context.Background()

	// Check if source already exists
	existing, err := dynamoService.GetSourceSubmission(ctx, source.ID)
	if err == nil && existing != nil {
		// Source exists, optionally update LastChecked
		log.Printf("Source %s already exists in database", source.ID)
		return nil
	}

	// Source doesn't exist, create it
	sourceRecord := models.SourceSubmission{
		PK:           fmt.Sprintf("SOURCE#%s", source.ID),
		SK:           "SUBMISSION",
		SourceID:     source.ID,
		SourceName:   source.Name,
		BaseURL:      source.BaseURL,
		SourceType:   "community-calendar", // Default type for auto-registered sources
		Priority:     source.Priority,
		ExpectedContent: []string{"events", "activities"},
		HintURLs:     source.TargetURLs,
		SubmittedBy:  "system-auto-registration",
		SubmittedAt:  time.Now(),
		UpdatedAt:    time.Now(),
		Status:       "active", // Auto-approve system sources
		StatusKey:    "STATUS#active",
		PriorityKey:  fmt.Sprintf("PRIORITY#%s#%s", source.Priority, source.ID),
	}

	log.Printf("Creating new source record for %s", source.ID)
	return dynamoService.CreateSourceSubmission(ctx, &sourceRecord)
}

func main() {
	lambda.Start(handleRequest)
}