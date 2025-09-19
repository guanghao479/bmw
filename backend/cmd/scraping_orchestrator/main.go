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
	TriggerType string `json:"trigger_type"` // scheduled, manual, on-demand
	SourceID    string `json:"source_id,omitempty"` // optional: scrape specific source
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
	s3Client        *services.S3Client
)

// Hardcoded sources for now (replace with DynamoDB later if needed)
var seattleSources = []Source{
	{
		ID:          "parentmap",
		Name:        "ParentMap",
		BaseURL:     "https://www.parentmap.com",
		TargetURLs:  []string{"https://www.parentmap.com/calendar"},
		Enabled:     true,
		Priority:    "high",
		Category:    "events",
	},
	{
		ID:          "seattleschild",
		Name:        "Seattle's Child",
		BaseURL:     "https://www.seattleschild.com",
		TargetURLs:  []string{"https://www.seattleschild.com/calendar/", "https://www.seattleschild.com/things-to-do/"},
		Enabled:     true,
		Priority:    "high",
		Category:    "events",
	},
	{
		ID:          "peps",
		Name:        "PEPS",
		BaseURL:     "https://www.peps.org",
		TargetURLs:  []string{"https://www.peps.org/events/", "https://www.peps.org/classes/"},
		Enabled:     true,
		Priority:    "medium",
		Category:    "classes",
	},
	{
		ID:          "tinybeans",
		Name:        "Tinybeans Seattle",
		BaseURL:     "https://tinybeans.com",
		TargetURLs:  []string{"https://tinybeans.com/seattle/"},
		Enabled:     true,
		Priority:    "medium",
		Category:    "activities",
	},
	{
		ID:          "seattlefunforkids",
		Name:        "Seattle Fun for Kids",
		BaseURL:     "https://www.seattlefunforkids.com",
		TargetURLs:  []string{"https://www.seattlefunforkids.com/events/"},
		Enabled:     true,
		Priority:    "low",
		Category:    "activities",
	},
	{
		ID:          "macaronikid",
		Name:        "Macaroni Kid West Seattle",
		BaseURL:     "https://westseattle.macaronikid.com",
		TargetURLs:  []string{"https://westseattle.macaronikid.com/events"},
		Enabled:     true,
		Priority:    "low",
		Category:    "local-events",
	},
}

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

	// Create S3 client for data storage
	s3Client, err = services.NewS3Client()
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}
}

func handleRequest(ctx context.Context, event ScrapingOrchestratorEvent) (ScrapingOrchestratorResponse, error) {
	start := time.Now()

	log.Printf("Starting scraping orchestrator with trigger type: %s", event.TriggerType)

	var allActivities []models.Activity
	var errors []string
	processedSources := 0

	// Filter sources if specific source requested
	sources := seattleSources
	if event.SourceID != "" {
		sources = filterSourceByID(sources, event.SourceID)
	}

	log.Printf("Processing %d sources", len(sources))

	// Process each source directly with FireCrawl
	for _, source := range sources {
		if !source.Enabled {
			log.Printf("Skipping disabled source: %s", source.Name)
			continue
		}

		log.Printf("Processing source: %s", source.Name)

		// Process each target URL for the source
		for _, url := range source.TargetURLs {
			log.Printf("Extracting activities from: %s", url)

			activities, err := extractActivitiesFromURL(url, source.Name)
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

	// Store activities in S3
	if len(allActivities) > 0 {
		err := storeActivitiesInS3(allActivities)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to store activities in S3: %v", err)
			log.Printf("ERROR: %s", errorMsg)
			errors = append(errors, errorMsg)
		} else {
			log.Printf("Successfully stored %d activities in S3", len(allActivities))
		}
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

func filterSourceByID(sources []Source, targetID string) []Source {
	for _, source := range sources {
		if source.ID == targetID {
			return []Source{source}
		}
	}
	return []Source{}
}

func extractActivitiesFromURL(url, sourceName string) ([]models.Activity, error) {
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

func storeActivitiesInS3(activities []models.Activity) error {
	// Upload to S3 as latest.json
	result, err := s3Client.UploadLatestActivities(activities)
	if err != nil {
		return fmt.Errorf("failed to upload latest activities: %w", err)
	}
	log.Printf("Uploaded latest activities to S3: %s", result.Location)

	// Also create a timestamped backup
	backupResult, err := s3Client.UploadActivitiesWithTimestamp(activities)
	if err != nil {
		log.Printf("Warning: Failed to create timestamped backup: %v", err)
		// Don't fail the entire operation for backup failure
	} else {
		log.Printf("Created timestamped backup: %s", backupResult.Location)
	}

	return nil
}

func extractDomain(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		log.Printf("Warning: Failed to parse URL %s: %v", urlStr, err)
		return urlStr
	}
	return parsedURL.Host
}

func main() {
	lambda.Start(handleRequest)
}