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

	return ResponseBody{
		Success: true,
		Message: "Active sources retrieved successfully",
		Data:    activeSources,
	}, 200
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