package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/url"
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
	dynamoService         *services.DynamoDBService
	firecrawlService      *services.FireCrawlClient
	conversionService     *services.SchemaConversionService
	lambdaClient          *lambdaclient.Client
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
	adminEventsTable := os.Getenv("ADMIN_EVENTS_TABLE")

	if familyActivitiesTable == "" || sourceManagementTable == "" || scrapingOperationsTable == "" || adminEventsTable == "" {
		log.Fatal("Required environment variables not set: FAMILY_ACTIVITIES_TABLE, SOURCE_MANAGEMENT_TABLE, SCRAPING_OPERATIONS_TABLE, ADMIN_EVENTS_TABLE")
	}

	// Initialize DynamoDB service
	dynamoService = services.NewDynamoDBService(
		dynamoClient,
		familyActivitiesTable,
		sourceManagementTable,
		scrapingOperationsTable,
		adminEventsTable,
	)

	// Initialize Firecrawl service
	firecrawlService, err = services.NewFireCrawlClient()
	if err != nil {
		log.Printf("Warning: Failed to initialize Firecrawl service: %v", err)
		// Don't fail startup, just log the warning
	}

	// Initialize schema conversion service
	conversionService = services.NewSchemaConversionService()

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

	// Admin Crawling Endpoints
	case method == "POST" && path == "/api/crawl/submit":
		responseBody, statusCode = handleCrawlSubmission(ctx, request.Body)

	// Debug Endpoints
	case method == "POST" && path == "/api/debug/extract":
		responseBody, statusCode = handleDebugExtraction(ctx, request.Body)

	case method == "GET" && path == "/api/events/pending":
		responseBody, statusCode = handleGetPendingEvents(ctx, request.QueryStringParameters)

	case method == "GET" && strings.HasPrefix(path, "/api/events/") && !strings.Contains(path[12:], "/"):
		eventID := strings.TrimPrefix(path, "/api/events/")
		responseBody, statusCode = handleGetEvent(ctx, eventID)

	case method == "PUT" && strings.HasPrefix(path, "/api/events/") && strings.HasSuffix(path, "/approve"):
		eventID := extractEventIDFromPath(path, "/approve")
		responseBody, statusCode = handleApproveEvent(ctx, eventID, request.Body)

	case method == "PUT" && strings.HasPrefix(path, "/api/events/") && strings.HasSuffix(path, "/reject"):
		eventID := extractEventIDFromPath(path, "/reject")
		responseBody, statusCode = handleRejectEvent(ctx, eventID, request.Body)

	case method == "PUT" && strings.HasPrefix(path, "/api/events/") && strings.HasSuffix(path, "/edit"):
		eventID := extractEventIDFromPath(path, "/edit")
		responseBody, statusCode = handleEditEvent(ctx, eventID, request.Body)

	case method == "GET" && path == "/api/schemas":
		responseBody, statusCode = handleGetSchemas(ctx)

	// Public Events API for main frontend
	case method == "GET" && path == "/api/events/approved":
		responseBody, statusCode = handleGetApprovedEvents(ctx, request.QueryStringParameters)

	// Source Management API for admin interface
	case method == "GET" && path == "/api/sources/active":
		responseBody, statusCode = handleGetActiveSources(ctx, request.QueryStringParameters)

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

// extractEventIDFromPath extracts event ID from path like /api/events/{id}/approve
func extractEventIDFromPath(path, suffix string) string {
	// Remove /api/events/ prefix and suffix
	withoutPrefix := strings.TrimPrefix(path, "/api/events/")
	eventID := strings.TrimSuffix(withoutPrefix, suffix)
	return eventID
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

// Admin Crawling Handler Functions

// handleCrawlSubmission handles POST /api/crawl/submit
func handleCrawlSubmission(ctx context.Context, body string) (ResponseBody, int) {
	if firecrawlService == nil {
		return ResponseBody{
			Success: false,
			Error:   "Firecrawl service not available",
		}, 500
	}

	var req models.CrawlSubmissionRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		}, 400
	}

	// Validate the request
	if err := req.Validate(); err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Validation error: " + err.Error(),
		}, 400
	}

	// Check for duplicate URLs in pending/approved admin events
	existingEvent, err := dynamoService.GetAdminEventByURL(ctx, req.URL)
	if err == nil && existingEvent != nil {
		return ResponseBody{
			Success: false,
			Error:   fmt.Sprintf("URL already exists with status: %s. Event ID: %s", existingEvent.Status, existingEvent.EventID),
		}, 409 // Conflict
	}

	// Check if URL is already configured as a source
	existingSource, err := dynamoService.GetSourceByURL(ctx, req.URL)
	if err == nil && existingSource != nil {
		return ResponseBody{
			Success: false,
			Error:   fmt.Sprintf("URL already exists as source: %s (ID: %s)", existingSource.SourceName, existingSource.SourceID),
		}, 409 // Conflict
	}

	// Create firecrawl extract request
	extractRequest := services.AdminExtractRequest{
		URL:          req.URL,
		SchemaType:   req.SchemaType,
		CustomSchema: req.CustomSchema,
	}

	// Perform extraction
	extractResponse, err := firecrawlService.ExtractWithSchema(extractRequest)
	if err != nil {
		log.Printf("Error extracting with Firecrawl: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to extract data from URL: " + err.Error(),
		}, 500
	}

	if !extractResponse.Success {
		return ResponseBody{
			Success: false,
			Error:   "Extraction was not successful",
		}, 500
	}

	// Generate unique event ID for this extraction
	eventID := uuid.New().String()

	// Create admin event record
	adminEvent := &models.AdminEvent{
		EventID:            eventID,
		SourceURL:          req.URL,
		SchemaType:         req.SchemaType,
		SchemaUsed:         extractResponse.SchemaUsed,
		RawExtractedData:   extractResponse.RawData,
		Status:             models.AdminEventStatusPending,
		ExtractedByUser:    req.ExtractedByUser,
		SubmissionID:       uuid.New().String(),
		AdminNotes:         req.AdminNotes,
	}

	// Generate conversion preview
	conversionResult, err := conversionService.ConvertToActivity(adminEvent)
	if err != nil {
		log.Printf("Error generating conversion preview: %v", err)
		// Continue without preview - admin can still review raw data
	} else {
		// Store conversion preview and issues
		if conversionResult.Activity != nil {
			activityJSON, _ := json.Marshal(conversionResult.Activity)
			var activityMap map[string]interface{}
			json.Unmarshal(activityJSON, &activityMap)
			adminEvent.ConvertedData = activityMap
		}
		adminEvent.ConversionIssues = conversionResult.Issues
	}

	// Store in DynamoDB
	if err := dynamoService.CreateAdminEvent(ctx, adminEvent); err != nil {
		log.Printf("Error storing admin event: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to store extracted events",
		}, 500
	}

	// Create or update source record if extraction was successful
	err = createOrUpdateSourceRecord(ctx, req, extractResponse.EventsCount)
	if err != nil {
		log.Printf("Warning: Failed to create/update source record: %v", err)
		// Don't fail the entire request for source management issues
	}

	return ResponseBody{
		Success: true,
		Message: fmt.Sprintf("Successfully extracted %d events from URL", extractResponse.EventsCount),
		Data: map[string]interface{}{
			"event_id":      eventID,
			"events_count":  extractResponse.EventsCount,
			"credits_used":  extractResponse.CreditsUsed,
			"processing_time": extractResponse.Metadata.ProcessingTime.String(),
		},
	}, 201
}

// handleDebugExtraction handles POST /api/debug/extract
func handleDebugExtraction(ctx context.Context, body string) (ResponseBody, int) {
	if firecrawlService == nil {
		return ResponseBody{
			Success: false,
			Error:   "Firecrawl service not available",
		}, 500
	}

	var req models.DebugExtractionRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		}, 400
	}

	// Validate the request
	if req.URL == "" {
		return ResponseBody{
			Success: false,
			Error:   "URL is required",
		}, 400
	}

	if req.SchemaType == "" {
		req.SchemaType = "events" // Default schema type
	}

	// Create firecrawl extract request
	extractRequest := services.AdminExtractRequest{
		URL:          req.URL,
		SchemaType:   req.SchemaType,
		CustomSchema: req.CustomSchema,
	}

	// Perform extraction with detailed diagnostics
	extractResponse, err := firecrawlService.ExtractWithSchema(extractRequest)
	if err != nil {
		log.Printf("Error extracting with Firecrawl: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to extract data from URL: " + err.Error(),
		}, 500
	}

	// Create a temporary admin event for conversion testing
	tempEventID := "debug-" + uuid.New().String()
	tempAdminEvent := &models.AdminEvent{
		EventID:          tempEventID,
		SourceURL:        req.URL,
		SchemaType:       req.SchemaType,
		SchemaUsed:       extractResponse.SchemaUsed,
		RawExtractedData: extractResponse.RawData,
		Status:           models.AdminEventStatusPending,
		ExtractedByUser:  "debug-user",
		SubmissionID:     tempEventID,
		AdminNotes:       "Debug extraction - not stored",
		ExtractedAt:      time.Now(),
	}

	// Perform conversion with detailed diagnostics
	conversionResult, conversionErr := conversionService.ConvertToActivity(tempAdminEvent)

	// Get detailed diagnostics from the services
	extractionDiagnostics := firecrawlService.GetLastExtractionDiagnostics()
	conversionDiagnostics := conversionService.GetLastConversionDiagnostics()

	// Build comprehensive debug response
	debugResponse := map[string]interface{}{
		"extraction": map[string]interface{}{
			"url":             req.URL,
			"schema_type":     req.SchemaType,
			"success":         extractResponse.Success,
			"events_count":    extractResponse.EventsCount,
			"credits_used":    extractResponse.CreditsUsed,
			"processing_time": extractResponse.Metadata.ProcessingTime.String(),
			"schema_used":     extractResponse.SchemaUsed,
		},
		"raw_data": map[string]interface{}{
			"structured_data": extractResponse.RawData,
		},
		"conversion": map[string]interface{}{
			"success": conversionErr == nil,
		},
	}

	// Add extraction diagnostics if available
	if extractionDiagnostics != nil {
		debugResponse["raw_data"].(map[string]interface{})["markdown_length"] = extractionDiagnostics.RawMarkdownLength
		debugResponse["raw_data"].(map[string]interface{})["markdown_sample"] = extractionDiagnostics.RawMarkdownSample
		debugResponse["extraction_diagnostics"] = extractionDiagnostics
		
		// Add validation issues from extraction
		if len(extractionDiagnostics.ValidationIssues) > 0 {
			debugResponse["extraction_validation"] = map[string]interface{}{
				"issues": extractionDiagnostics.ValidationIssues,
			}
		}
	}

	// Add conversion details if successful
	if conversionErr == nil && conversionResult != nil {
		debugResponse["conversion"].(map[string]interface{})["activity"] = conversionResult.Activity
		debugResponse["conversion"].(map[string]interface{})["issues"] = conversionResult.Issues
		debugResponse["conversion"].(map[string]interface{})["field_mappings"] = conversionResult.FieldMappings
		debugResponse["conversion"].(map[string]interface{})["confidence_score"] = conversionResult.ConfidenceScore
		
		// Add detailed mappings and validation results if available
		if conversionResult.DetailedMappings != nil {
			debugResponse["conversion"].(map[string]interface{})["detailed_mappings"] = conversionResult.DetailedMappings
		}
		if conversionResult.ValidationResults != nil {
			debugResponse["conversion"].(map[string]interface{})["validation_results"] = conversionResult.ValidationResults
		}
	} else if conversionErr != nil {
		debugResponse["conversion"].(map[string]interface{})["error"] = conversionErr.Error()
	}

	// Add conversion diagnostics if available
	if conversionDiagnostics != nil {
		debugResponse["conversion_diagnostics"] = conversionDiagnostics
	}

	// Add suggestions for improvement
	suggestions := generateExtractionSuggestions(extractResponse, conversionResult, conversionErr)
	if len(suggestions) > 0 {
		debugResponse["suggestions"] = suggestions
	}

	return ResponseBody{
		Success: true,
		Message: "Debug extraction completed",
		Data:    debugResponse,
	}, 200
}

// generateExtractionSuggestions provides actionable suggestions based on extraction and conversion results
func generateExtractionSuggestions(extractResponse *services.AdminExtractResponse, conversionResult *models.ConversionResult, conversionErr error) []string {
	var suggestions []string

	// Extraction-level suggestions
	if !extractResponse.Success {
		suggestions = append(suggestions, "Extraction failed - check if the URL is accessible and contains structured content")
	}

	if extractResponse.EventsCount == 0 {
		suggestions = append(suggestions, "No events found - try a different schema type or check if the page contains event information")
	}

	// Conversion-level suggestions
	if conversionErr != nil {
		suggestions = append(suggestions, fmt.Sprintf("Conversion failed: %s", conversionErr.Error()))
	}

	if conversionResult != nil {
		if conversionResult.ConfidenceScore < 50 {
			suggestions = append(suggestions, "Low confidence score - extracted data may be incomplete or malformed")
		}

		if len(conversionResult.Issues) > 3 {
			suggestions = append(suggestions, "Multiple conversion issues detected - consider using a different schema type or improving source data quality")
		}

		// Check for specific field mapping issues
		if conversionResult.FieldMappings != nil {
			missingFields := []string{}
			for field, source := range conversionResult.FieldMappings {
				if source == "not_found" || source == "generated" {
					missingFields = append(missingFields, field)
				}
			}
			if len(missingFields) > 0 {
				suggestions = append(suggestions, fmt.Sprintf("Missing or generated fields: %s - check if source contains this information", strings.Join(missingFields, ", ")))
			}
		}
	}

	// Schema-specific suggestions
	if extractResponse.SchemaUsed != nil && extractResponse.EventsCount > 0 {
		suggestions = append(suggestions, "Successfully extracted events - consider this schema configuration for production")
	}

	return suggestions
}

// truncateString truncates a string to maxLength characters
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}

// generateConversionDetails creates detailed conversion information for an admin event
func generateConversionDetails(ctx context.Context, event *models.AdminEvent) map[string]interface{} {
	details := map[string]interface{}{
		"has_conversion_preview": event.ConvertedData != nil,
		"conversion_issues_count": len(event.ConversionIssues),
		"conversion_status": "unknown",
	}

	// Attempt to regenerate conversion to get latest diagnostics
	if conversionService != nil {
		conversionResult, err := conversionService.ConvertToActivity(event)
		if err != nil {
			details["conversion_status"] = "failed"
			details["conversion_error"] = err.Error()
		} else if conversionResult != nil {
			details["conversion_status"] = "success"
			details["confidence_score"] = conversionResult.ConfidenceScore
			details["field_mappings"] = conversionResult.FieldMappings
			details["issues_count"] = len(conversionResult.Issues)
			
			// Add detailed mappings if available
			if conversionResult.DetailedMappings != nil {
				details["detailed_mappings"] = conversionResult.DetailedMappings
			}
			
			// Add validation results if available
			if conversionResult.ValidationResults != nil {
				details["validation_results"] = conversionResult.ValidationResults
			}

			// Categorize issues by severity
			issuesByType := make(map[string][]string)
			for _, issue := range conversionResult.Issues {
				// Simple categorization based on keywords
				if strings.Contains(strings.ToLower(issue), "missing") {
					issuesByType["missing_data"] = append(issuesByType["missing_data"], issue)
				} else if strings.Contains(strings.ToLower(issue), "invalid") || strings.Contains(strings.ToLower(issue), "format") {
					issuesByType["format_issues"] = append(issuesByType["format_issues"], issue)
				} else {
					issuesByType["other"] = append(issuesByType["other"], issue)
				}
			}
			details["issues_by_type"] = issuesByType
		}
	}

	return details
}

// generateRawDataSample creates a sample of the raw extracted data for debugging
func generateRawDataSample(rawData map[string]interface{}) map[string]interface{} {
	sample := map[string]interface{}{
		"structure": analyzeDataStructure(rawData),
		"sample_fields": make(map[string]interface{}),
		"total_fields": len(rawData),
	}

	// Add samples of each top-level field
	fieldCount := 0
	for key, value := range rawData {
		if fieldCount >= 5 { // Limit to first 5 fields
			break
		}

		switch v := value.(type) {
		case string:
			sample["sample_fields"].(map[string]interface{})[key] = map[string]interface{}{
				"type": "string",
				"length": len(v),
				"sample": truncateString(v, 100),
			}
		case []interface{}:
			sample["sample_fields"].(map[string]interface{})[key] = map[string]interface{}{
				"type": "array",
				"length": len(v),
				"sample": truncateArray(v, 2),
			}
		case map[string]interface{}:
			sample["sample_fields"].(map[string]interface{})[key] = map[string]interface{}{
				"type": "object",
				"fields": len(v),
				"sample": truncateObject(v, 3),
			}
		default:
			sample["sample_fields"].(map[string]interface{})[key] = map[string]interface{}{
				"type": fmt.Sprintf("%T", v),
				"value": v,
			}
		}
		fieldCount++
	}

	return sample
}

// assessDataQuality provides a quality assessment of the extracted data
func assessDataQuality(event *models.AdminEvent) map[string]interface{} {
	assessment := map[string]interface{}{
		"overall_score": 0.0,
		"factors": make(map[string]interface{}),
		"recommendations": []string{},
	}

	score := 100.0
	factors := make(map[string]interface{})

	// Check if we have extracted data
	if event.RawExtractedData == nil || len(event.RawExtractedData) == 0 {
		score -= 50
		factors["data_availability"] = map[string]interface{}{
			"score": 0,
			"message": "No extracted data available",
		}
		assessment["recommendations"] = append(assessment["recommendations"].([]string), "Re-run extraction with different schema or check source URL")
	} else {
		factors["data_availability"] = map[string]interface{}{
			"score": 100,
			"message": "Data successfully extracted",
		}
	}

	// Check conversion success
	if event.ConvertedData != nil {
		factors["conversion_success"] = map[string]interface{}{
			"score": 100,
			"message": "Successfully converted to Activity model",
		}
	} else {
		score -= 30
		factors["conversion_success"] = map[string]interface{}{
			"score": 0,
			"message": "Failed to convert to Activity model",
		}
		assessment["recommendations"] = append(assessment["recommendations"].([]string), "Check conversion issues and consider different schema type")
	}

	// Check conversion issues
	issueCount := len(event.ConversionIssues)
	if issueCount == 0 {
		factors["conversion_issues"] = map[string]interface{}{
			"score": 100,
			"message": "No conversion issues",
		}
	} else if issueCount <= 2 {
		score -= 10
		factors["conversion_issues"] = map[string]interface{}{
			"score": 80,
			"message": fmt.Sprintf("%d minor conversion issues", issueCount),
		}
	} else {
		score -= 20
		factors["conversion_issues"] = map[string]interface{}{
			"score": 60,
			"message": fmt.Sprintf("%d conversion issues detected", issueCount),
		}
		assessment["recommendations"] = append(assessment["recommendations"].([]string), "Review conversion issues and improve source data quality")
	}

	// Check events count
	eventsCount := event.GetExtractedEventsCount()
	if eventsCount == 0 {
		score -= 40
		factors["events_count"] = map[string]interface{}{
			"score": 0,
			"message": "No events found in extracted data",
		}
		assessment["recommendations"] = append(assessment["recommendations"].([]string), "Try different schema type or check if URL contains event information")
	} else if eventsCount >= 1 && eventsCount <= 50 {
		factors["events_count"] = map[string]interface{}{
			"score": 100,
			"message": fmt.Sprintf("%d events found", eventsCount),
		}
	} else {
		score -= 10
		factors["events_count"] = map[string]interface{}{
			"score": 90,
			"message": fmt.Sprintf("%d events found (unusually high)", eventsCount),
		}
		assessment["recommendations"] = append(assessment["recommendations"].([]string), "Verify extraction accuracy - high event count may indicate over-extraction")
	}

	assessment["overall_score"] = math.Max(0, score)
	assessment["factors"] = factors

	return assessment
}

// Helper functions for data sampling
func truncateArray(arr []interface{}, maxItems int) []interface{} {
	if len(arr) <= maxItems {
		return arr
	}
	return arr[:maxItems]
}

func truncateObject(obj map[string]interface{}, maxFields int) map[string]interface{} {
	if len(obj) <= maxFields {
		return obj
	}
	
	result := make(map[string]interface{})
	count := 0
	for k, v := range obj {
		if count >= maxFields {
			break
		}
		result[k] = v
		count++
	}
	return result
}

func analyzeDataStructure(data map[string]interface{}) map[string]interface{} {
	structure := map[string]interface{}{
		"total_fields": len(data),
		"field_types": make(map[string]int),
		"array_fields": []string{},
		"object_fields": []string{},
		"string_fields": []string{},
	}

	fieldTypes := make(map[string]int)
	var arrayFields, objectFields, stringFields []string

	for key, value := range data {
		switch value.(type) {
		case string:
			fieldTypes["string"]++
			stringFields = append(stringFields, key)
		case []interface{}:
			fieldTypes["array"]++
			arrayFields = append(arrayFields, key)
		case map[string]interface{}:
			fieldTypes["object"]++
			objectFields = append(objectFields, key)
		case int, int64, float64:
			fieldTypes["number"]++
		case bool:
			fieldTypes["boolean"]++
		default:
			fieldTypes["other"]++
		}
	}

	structure["field_types"] = fieldTypes
	structure["array_fields"] = arrayFields
	structure["object_fields"] = objectFields
	structure["string_fields"] = stringFields

	return structure
}

// handleGetPendingEvents handles GET /api/events/pending
func handleGetPendingEvents(ctx context.Context, queryParams map[string]string) (ResponseBody, int) {
	limit := int32(50)
	if limitStr, ok := queryParams["limit"]; ok {
		if parsedLimit := parseLimit(limitStr); parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get all pending events (pending + edited)
	pendingEvents, err := dynamoService.GetAllPendingAdminEvents(ctx, limit)
	if err != nil {
		log.Printf("Error getting pending events: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to retrieve pending events",
		}, 500
	}

	// Enhance each event with detailed conversion and diagnostic information
	var enhancedEvents []map[string]interface{}
	for _, event := range pendingEvents {
		enhanced := map[string]interface{}{
			"event_id":             event.EventID,
			"source_url":           event.SourceURL,
			"schema_type":          event.SchemaType,
			"status":               event.Status,
			"extracted_at":         event.ExtractedAt,
			"extracted_by_user":    event.ExtractedByUser,
			"events_count":         event.GetExtractedEventsCount(),
			"conversion_issues":    event.ConversionIssues,
			"can_approve":          event.CanBeApproved(),
			"admin_notes":          event.AdminNotes,
		}

		// Add conversion preview if available
		if event.ConvertedData != nil {
			enhanced["conversion_preview"] = event.ConvertedData
		}

		// Generate detailed conversion information
		conversionDetails := generateConversionDetails(ctx, &event)
		enhanced["conversion_details"] = conversionDetails

		// Add raw data sample for debugging
		rawDataSample := generateRawDataSample(event.RawExtractedData)
		enhanced["raw_data_sample"] = rawDataSample

		// Add data quality assessment
		qualityAssessment := assessDataQuality(&event)
		enhanced["quality_assessment"] = qualityAssessment

		enhancedEvents = append(enhancedEvents, enhanced)
	}

	return ResponseBody{
		Success: true,
		Message: "Pending events retrieved successfully",
		Data:    enhancedEvents,
	}, 200
}

// handleGetEvent handles GET /api/events/{id}
func handleGetEvent(ctx context.Context, eventID string) (ResponseBody, int) {
	if eventID == "" {
		return ResponseBody{
			Success: false,
			Error:   "Event ID is required",
		}, 400
	}

	// Get the admin event by ID
	adminEvent, err := dynamoService.GetAdminEventByID(ctx, eventID)
	if err != nil {
		log.Printf("Error getting admin event: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Event not found",
		}, 404
	}

	// Generate fresh conversion preview
	conversionPreview, err := conversionService.PreviewConversion(adminEvent)
	if err != nil {
		log.Printf("Error generating conversion preview: %v", err)
		conversionPreview = map[string]interface{}{
			"error": "Could not generate conversion preview",
		}
	}

	eventDetails := map[string]interface{}{
		"event_id":             adminEvent.EventID,
		"source_url":           adminEvent.SourceURL,
		"schema_type":          adminEvent.SchemaType,
		"schema_used":          adminEvent.SchemaUsed,
		"raw_extracted_data":   adminEvent.RawExtractedData,
		"conversion_preview":   conversionPreview,
		"status":               adminEvent.Status,
		"extracted_at":         adminEvent.ExtractedAt,
		"extracted_by_user":    adminEvent.ExtractedByUser,
		"admin_notes":          adminEvent.AdminNotes,
		"conversion_issues":    adminEvent.ConversionIssues,
		"can_approve":          adminEvent.CanBeApproved(),
		"events_count":         adminEvent.GetExtractedEventsCount(),
	}

	return ResponseBody{
		Success: true,
		Message: "Event details retrieved successfully",
		Data:    eventDetails,
	}, 200
}

// handleApproveEvent handles PUT /api/events/{id}/approve
func handleApproveEvent(ctx context.Context, eventID string, body string) (ResponseBody, int) {
	if eventID == "" {
		return ResponseBody{
			Success: false,
			Error:   "Event ID is required",
		}, 400
	}

	var req models.AdminEventReview
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		}, 400
	}

	// Get the admin event
	adminEvent, err := dynamoService.GetAdminEventByID(ctx, eventID)
	if err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Event not found",
		}, 404
	}

	// Check if event can be approved
	if !adminEvent.IsPending() {
		return ResponseBody{
			Success: false,
			Error:   fmt.Sprintf("Event cannot be approved - current status: %s", adminEvent.Status),
		}, 400
	}

	// Convert to Activity model with detailed diagnostics
	conversionResult, err := conversionService.ConvertToActivity(adminEvent)
	if err != nil {
		// Get detailed conversion diagnostics for better error reporting
		conversionDiagnostics := conversionService.GetLastConversionDiagnostics()
		
		errorDetails := map[string]interface{}{
			"conversion_error": err.Error(),
			"event_id": eventID,
			"source_url": adminEvent.SourceURL,
			"schema_type": adminEvent.SchemaType,
		}
		
		if conversionDiagnostics != nil {
			errorDetails["diagnostics"] = map[string]interface{}{
				"processing_time": conversionDiagnostics.ProcessingTime.String(),
				"conversion_issues": conversionDiagnostics.ConversionIssues,
				"field_mappings": conversionDiagnostics.FieldMappings,
				"confidence_score": conversionDiagnostics.ConfidenceScore,
			}
		}
		
		return ResponseBody{
			Success: false,
			Error:   "Failed to convert event to activity - see details for more information",
			Data:    errorDetails,
		}, 500
	}

	if conversionResult.Activity == nil {
		errorDetails := map[string]interface{}{
			"conversion_issues": conversionResult.Issues,
			"field_mappings": conversionResult.FieldMappings,
			"confidence_score": conversionResult.ConfidenceScore,
			"event_id": eventID,
			"source_url": adminEvent.SourceURL,
			"suggestions": []string{
				"Check if the extracted data contains valid event information",
				"Try using a different schema type for extraction",
				"Review the conversion issues for specific problems",
			},
		}
		
		if conversionResult.DetailedMappings != nil {
			errorDetails["detailed_mappings"] = conversionResult.DetailedMappings
		}
		
		if conversionResult.ValidationResults != nil {
			errorDetails["validation_results"] = conversionResult.ValidationResults
		}
		
		return ResponseBody{
			Success: false,
			Error:   "Could not generate valid activity from event data - see details for diagnostic information",
			Data:    errorDetails,
		}, 400
	}

	// Store the converted activity in the main activities table
	activities := []*models.Activity{conversionResult.Activity}
	if err := dynamoService.BatchPutActivities(ctx, activities); err != nil {
		log.Printf("Error storing approved activity: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to publish approved event",
		}, 500
	}

	// Update admin event status
	now := time.Now()
	adminEvent.Status = models.AdminEventStatusApproved
	adminEvent.ReviewedAt = &now
	adminEvent.ReviewedBy = req.ReviewedBy
	adminEvent.AdminNotes = req.AdminNotes

	if err := dynamoService.UpdateAdminEvent(ctx, adminEvent); err != nil {
		log.Printf("Error updating admin event status: %v", err)
		// Event was published but status update failed - log but don't fail
	}

	// Get final conversion diagnostics for success response
	conversionDiagnostics := conversionService.GetLastConversionDiagnostics()
	
	successData := map[string]interface{}{
		"event_id":    eventID,
		"activity_id": conversionResult.Activity.ID,
		"status":      "approved",
		"conversion_summary": map[string]interface{}{
			"confidence_score": conversionResult.ConfidenceScore,
			"issues_count": len(conversionResult.Issues),
			"field_mappings_count": len(conversionResult.FieldMappings),
		},
	}
	
	// Add detailed conversion information if available
	if conversionDiagnostics != nil {
		successData["conversion_details"] = map[string]interface{}{
			"processing_time": conversionDiagnostics.ProcessingTime.String(),
			"success": conversionDiagnostics.Success,
			"field_mappings": conversionDiagnostics.FieldMappings,
		}
	}
	
	// Include any conversion issues as warnings
	if len(conversionResult.Issues) > 0 {
		successData["warnings"] = conversionResult.Issues
	}

	return ResponseBody{
		Success: true,
		Message: "Event approved and published successfully",
		Data:    successData,
	}, 200
}

// handleRejectEvent handles PUT /api/events/{id}/reject
func handleRejectEvent(ctx context.Context, eventID string, body string) (ResponseBody, int) {
	if eventID == "" {
		return ResponseBody{
			Success: false,
			Error:   "Event ID is required",
		}, 400
	}

	var req models.AdminEventReview
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		}, 400
	}

	// Get the admin event
	adminEvent, err := dynamoService.GetAdminEventByID(ctx, eventID)
	if err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Event not found",
		}, 404
	}

	// Update admin event status
	now := time.Now()
	adminEvent.Status = models.AdminEventStatusRejected
	adminEvent.ReviewedAt = &now
	adminEvent.ReviewedBy = req.ReviewedBy
	adminEvent.AdminNotes = req.AdminNotes

	if err := dynamoService.UpdateAdminEvent(ctx, adminEvent); err != nil {
		log.Printf("Error updating admin event status: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to reject event",
		}, 500
	}

	// Generate diagnostic information for the rejection
	rejectionData := map[string]interface{}{
		"event_id": eventID,
		"status":   "rejected",
		"rejection_details": map[string]interface{}{
			"rejected_by": req.ReviewedBy,
			"rejected_at": now,
			"admin_notes": req.AdminNotes,
			"source_url": adminEvent.SourceURL,
			"schema_type": adminEvent.SchemaType,
		},
	}
	
	// Add conversion analysis to help understand why it was rejected
	if conversionService != nil {
		conversionResult, err := conversionService.ConvertToActivity(adminEvent)
		if err != nil {
			rejectionData["conversion_analysis"] = map[string]interface{}{
				"conversion_failed": true,
				"error": err.Error(),
			}
		} else if conversionResult != nil {
			rejectionData["conversion_analysis"] = map[string]interface{}{
				"conversion_succeeded": true,
				"confidence_score": conversionResult.ConfidenceScore,
				"issues_count": len(conversionResult.Issues),
				"issues": conversionResult.Issues,
				"field_mappings": conversionResult.FieldMappings,
			}
		}
	}
	
	// Add data quality assessment
	qualityAssessment := assessDataQuality(adminEvent)
	rejectionData["quality_assessment"] = qualityAssessment

	return ResponseBody{
		Success: true,
		Message: "Event rejected successfully",
		Data:    rejectionData,
	}, 200
}

// handleEditEvent handles PUT /api/events/{id}/edit
func handleEditEvent(ctx context.Context, eventID string, body string) (ResponseBody, int) {
	if eventID == "" {
		return ResponseBody{
			Success: false,
			Error:   "Event ID is required",
		}, 400
	}

	var req models.AdminEventReview
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		}, 400
	}

	// Get the admin event
	adminEvent, err := dynamoService.GetAdminEventByID(ctx, eventID)
	if err != nil {
		return ResponseBody{
			Success: false,
			Error:   "Event not found",
		}, 404
	}

	// Update raw extracted data with edited data
	if req.EditedData != nil {
		adminEvent.RawExtractedData = req.EditedData
	}

	// Update status to edited
	now := time.Now()
	adminEvent.Status = models.AdminEventStatusEdited
	adminEvent.ReviewedAt = &now
	adminEvent.ReviewedBy = req.ReviewedBy
	adminEvent.AdminNotes = req.AdminNotes

	// Regenerate conversion preview with edited data
	conversionResult, err := conversionService.ConvertToActivity(adminEvent)
	if err != nil {
		log.Printf("Error regenerating conversion preview: %v", err)
	} else {
		if conversionResult.Activity != nil {
			activityJSON, _ := json.Marshal(conversionResult.Activity)
			var activityMap map[string]interface{}
			json.Unmarshal(activityJSON, &activityMap)
			adminEvent.ConvertedData = activityMap
		}
		adminEvent.ConversionIssues = conversionResult.Issues
	}

	if err := dynamoService.UpdateAdminEvent(ctx, adminEvent); err != nil {
		log.Printf("Error updating admin event: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to save edited event",
		}, 500
	}

	return ResponseBody{
		Success: true,
		Message: "Event edited successfully",
		Data: map[string]interface{}{
			"event_id": eventID,
			"status":   "edited",
		},
	}, 200
}

// handleGetSchemas handles GET /api/schemas
func handleGetSchemas(ctx context.Context) (ResponseBody, int) {
	schemas := models.GetPredefinedSchemas()

	// Format schemas for frontend consumption
	formattedSchemas := make(map[string]interface{})
	for key, schema := range schemas {
		formattedSchemas[key] = map[string]interface{}{
			"name":        schema.Name,
			"description": schema.Description,
			"examples":    schema.Examples,
			"schema":      schema.Schema,
		}
	}

	return ResponseBody{
		Success: true,
		Message: "Available extraction schemas",
		Data:    formattedSchemas,
	}, 200
}

// handleGetApprovedEvents handles GET /api/events/approved - Public endpoint for main frontend
func handleGetApprovedEvents(ctx context.Context, queryParams map[string]string) (ResponseBody, int) {
	// Parse query parameters
	limit := int32(100) // Default limit
	if limitStr, ok := queryParams["limit"]; ok {
		if parsedLimit := parseLimit(limitStr); parsedLimit > 0 && parsedLimit <= 500 {
			limit = parsedLimit
		}
	}

	offset := int32(0)
	if offsetStr, ok := queryParams["offset"]; ok {
		if parsedOffset := parseLimit(offsetStr); parsedOffset > 0 {
			offset = parsedOffset
		}
	}

	// Get all approved admin events
	approvedEvents, err := dynamoService.GetApprovedAdminEvents(ctx, limit+offset) // Get extra for offset
	if err != nil {
		log.Printf("Error getting approved events: %v", err)
		return ResponseBody{
			Success: false,
			Error:   "Failed to retrieve approved events",
		}, 500
	}

	// Apply offset if specified
	if offset > 0 && int(offset) < len(approvedEvents) {
		approvedEvents = approvedEvents[offset:]
	}

	// Apply limit
	if int(limit) < len(approvedEvents) {
		approvedEvents = approvedEvents[:limit]
	}

	// Convert AdminEvents to Activity format for frontend compatibility
	var activities []map[string]interface{}
	for _, event := range approvedEvents {
		activity, err := convertAdminEventToActivity(&event)
		if err != nil {
			log.Printf("Error converting admin event to activity: %v", err)
			continue // Skip this event rather than fail entire request
		}
		activities = append(activities, activity)
	}

	// Create response metadata
	meta := map[string]interface{}{
		"total":         len(activities),
		"limit":         limit,
		"offset":        offset,
		"last_updated":  time.Now().Format(time.RFC3339),
		"cache_duration": 300, // 5 minutes cache suggestion
	}

	// Apply additional filters if provided
	if category, ok := queryParams["category"]; ok && category != "" {
		activities = filterActivitiesByCategory(activities, category)
		meta["filtered_by_category"] = category
	}

	if dateFrom, ok := queryParams["date_from"]; ok && dateFrom != "" {
		activities = filterActivitiesByDate(activities, dateFrom)
		meta["filtered_from_date"] = dateFrom
	}

	if updatedSince, ok := queryParams["updated_since"]; ok && updatedSince != "" {
		activities = filterActivitiesByUpdatedSince(activities, updatedSince)
		meta["filtered_updated_since"] = updatedSince
	}

	// Update final count after filtering
	meta["total"] = len(activities)

	return ResponseBody{
		Success: true,
		Message: fmt.Sprintf("Retrieved %d approved events", len(activities)),
		Data: map[string]interface{}{
			"activities": activities,
			"meta":       meta,
		},
	}, 200
}

// Helper functions for approved events endpoint

// convertAdminEventToActivity converts an AdminEvent to Activity format for frontend
func convertAdminEventToActivity(event *models.AdminEvent) (map[string]interface{}, error) {
	// Use the conversion service if available, otherwise create basic mapping
	if conversionService != nil {
		conversionResult, err := conversionService.ConvertToActivity(event)
		if err != nil {
			return nil, fmt.Errorf("conversion service failed: %w", err)
		}
		if conversionResult.Activity != nil {
			// Convert Activity struct to map for JSON response
			activityJSON, _ := json.Marshal(conversionResult.Activity)
			var activityMap map[string]interface{}
			json.Unmarshal(activityJSON, &activityMap)

			// Add admin metadata
			activityMap["admin_metadata"] = map[string]interface{}{
				"extracted_at":     event.ExtractedAt,
				"extracted_by":     event.ExtractedByUser,
				"event_id":         event.EventID,
				"source_url":       event.SourceURL,
				"schema_type":      event.SchemaType,
			}

			return activityMap, nil
		}
	}

	// Fallback: create basic activity from raw data
	activity := map[string]interface{}{
		"id":          event.EventID,
		"source":      map[string]interface{}{
			"url":       event.SourceURL,
			"scraped_at": event.ExtractedAt,
		},
		"updated_at":  event.UpdatedAt,
		"created_at":  event.ExtractedAt,
	}

	// Try to extract basic fields from raw data
	if rawData := event.RawExtractedData; rawData != nil {
		if events, ok := rawData["events"].([]interface{}); ok && len(events) > 0 {
			if firstEvent, ok := events[0].(map[string]interface{}); ok {
				activity["title"] = firstEvent["title"]
				activity["description"] = firstEvent["description"]
				activity["location"] = firstEvent["location"]
				activity["schedule"] = firstEvent["date"]
				activity["pricing"] = firstEvent["price"]
			}
		}
	}

	return activity, nil
}

// filterActivitiesByCategory filters activities by category type
func filterActivitiesByCategory(activities []map[string]interface{}, category string) []map[string]interface{} {
	var filtered []map[string]interface{}
	for _, activity := range activities {
		// Check if activity matches category
		if activityCategory, ok := activity["category"].(string); ok && activityCategory == category {
			filtered = append(filtered, activity)
		}
	}
	return filtered
}

// filterActivitiesByDate filters activities from a specific date
func filterActivitiesByDate(activities []map[string]interface{}, dateFrom string) []map[string]interface{} {
	var filtered []map[string]interface{}
	fromDate, err := time.Parse("2006-01-02", dateFrom)
	if err != nil {
		return activities // Return unfiltered if date parsing fails
	}

	for _, activity := range activities {
		// Check activity date
		if schedule, ok := activity["schedule"].(map[string]interface{}); ok {
			if startDate, ok := schedule["start_date"].(string); ok {
				if activityDate, err := time.Parse("2006-01-02", startDate); err == nil {
					if activityDate.After(fromDate) || activityDate.Equal(fromDate) {
						filtered = append(filtered, activity)
					}
				}
			}
		}
	}
	return filtered
}

// filterActivitiesByUpdatedSince filters activities updated since a timestamp
func filterActivitiesByUpdatedSince(activities []map[string]interface{}, updatedSince string) []map[string]interface{} {
	var filtered []map[string]interface{}
	sinceTime, err := time.Parse(time.RFC3339, updatedSince)
	if err != nil {
		return activities // Return unfiltered if timestamp parsing fails
	}

	for _, activity := range activities {
		if updatedAt, ok := activity["updated_at"].(time.Time); ok {
			if updatedAt.After(sinceTime) {
				filtered = append(filtered, activity)
			}
		}
	}
	return filtered
}

// createOrUpdateSourceRecord creates or updates a source record when a URL is successfully crawled
func createOrUpdateSourceRecord(ctx context.Context, req models.CrawlSubmissionRequest, eventsCount int) error {
	// Check if source already exists
	existingSource, err := dynamoService.GetSourceByURL(ctx, req.URL)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("failed to check existing source: %w", err)
	}

	if existingSource != nil {
		// Update existing source with latest extraction stats
		existingSource.UpdatedAt = time.Now()

		// If source was inactive, activate it since extraction was successful
		if existingSource.Status != "active" {
			existingSource.Status = "active"
			existingSource.StatusKey = "STATUS#active"
			log.Printf("Activated source %s due to successful extraction", existingSource.SourceID)
		}

		log.Printf("Updated existing source %s - extracted %d events", existingSource.SourceID, eventsCount)
		return dynamoService.UpdateSourceSubmission(ctx, existingSource)
	}

	// Create new source record
	sourceID := generateSourceIDFromURL(req.URL)

	sourceRecord := &models.SourceSubmission{
		PK:           fmt.Sprintf("SOURCE#%s", sourceID),
		SK:           "SUBMISSION",
		SourceID:     sourceID,
		SourceName:   extractSourceNameFromURL(req.URL),
		BaseURL:      req.URL,
		SourceType:   "auto-discovered", // Mark as auto-discovered from crawl
		Priority:     "medium",
		ExpectedContent: []string{req.SchemaType}, // Use the schema type that was used
		HintURLs:     []string{req.URL},
		SubmittedBy:  fmt.Sprintf("auto-discovery-by-%s", req.ExtractedByUser),
		SubmittedAt:  time.Now(),
		UpdatedAt:    time.Now(),
		Status:       "active", // Auto-approve since extraction was successful
		StatusKey:    "STATUS#active",
		PriorityKey:  fmt.Sprintf("PRIORITY#medium#%s", sourceID),
	}

	log.Printf("Creating new auto-discovered source: %s (%s)", sourceRecord.SourceName, sourceID)
	return dynamoService.CreateSourceSubmission(ctx, sourceRecord)
}

// generateSourceIDFromURL creates a source ID from a URL
func generateSourceIDFromURL(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		// Fallback to simple slug generation
		return strings.ReplaceAll(strings.ToLower(urlStr), "/", "-")
	}

	// Use domain name as base for ID
	domain := parsedURL.Host
	if strings.HasPrefix(domain, "www.") {
		domain = domain[4:]
	}

	// Remove common TLD for cleaner ID
	if strings.HasSuffix(domain, ".com") {
		domain = domain[:len(domain)-4]
	} else if strings.HasSuffix(domain, ".org") {
		domain = domain[:len(domain)-4]
	}

	// Replace dots with dashes for valid ID
	sourceID := strings.ReplaceAll(domain, ".", "-")

	// Add random suffix to prevent collisions
	return fmt.Sprintf("%s-%s", sourceID, uuid.New().String()[:8])
}

// extractSourceNameFromURL creates a human-readable source name from URL
func extractSourceNameFromURL(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	domain := parsedURL.Host
	if strings.HasPrefix(domain, "www.") {
		domain = domain[4:]
	}

	// Convert domain to title case
	parts := strings.Split(domain, ".")
	if len(parts) > 0 {
		baseName := parts[0]
		// Convert kebab-case or underscore to title case
		baseName = strings.ReplaceAll(baseName, "-", " ")
		baseName = strings.ReplaceAll(baseName, "_", " ")

		// Title case each word
		words := strings.Fields(baseName)
		for i, word := range words {
			if len(word) > 0 {
				words[i] = strings.ToUpper(word[:1]) + word[1:]
			}
		}
		return strings.Join(words, " ")
	}

	return domain
}


func main() {
	lambda.Start(handleRequest)
}