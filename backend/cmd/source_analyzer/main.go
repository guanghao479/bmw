package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"seattle-family-activities-scraper/internal/models"
	"seattle-family-activities-scraper/internal/services"
)

// SourceAnalyzerEvent represents the input event for source analysis
type SourceAnalyzerEvent struct {
	SourceID string `json:"source_id"`
	TriggerType string `json:"trigger_type"` // manual, scheduled, webhook
}

// SourceAnalyzerResponse represents the Lambda response
type SourceAnalyzerResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// Response body structure
type ResponseBody struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	SourceID  string `json:"source_id,omitempty"`
	AnalysisID string `json:"analysis_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

var (
	dynamoService    *services.DynamoDBService
	firecrawlClient  *services.FireCrawlClient
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

	// Create FireCrawl client
	var err2 error
	firecrawlClient, err2 = services.NewFireCrawlClient()
	if err2 != nil {
		log.Fatalf("Failed to create FireCrawl client: %v", err2)
	}
}

// handleRequest processes the source analyzer Lambda request
func handleRequest(ctx context.Context, event SourceAnalyzerEvent) (SourceAnalyzerResponse, error) {
	log.Printf("Processing source analysis for source ID: %s (trigger: %s)", event.SourceID, event.TriggerType)

	// Get source submission from DynamoDB
	sourceSubmission, err := dynamoService.GetSourceSubmission(ctx, event.SourceID)
	if err != nil {
		log.Printf("Failed to get source submission: %v", err)
		return createErrorResponse(400, fmt.Sprintf("Source not found: %s", event.SourceID))
	}

	// Check if source is in the correct state for analysis
	if sourceSubmission.Status != models.SourceStatusPendingAnalysis {
		log.Printf("Source %s is not pending analysis (status: %s)", event.SourceID, sourceSubmission.Status)
		return createErrorResponse(400, fmt.Sprintf("Source %s is not ready for analysis", event.SourceID))
	}

	// Perform source analysis
	analysisResults, err := analyzeSource(ctx, sourceSubmission)
	if err != nil {
		log.Printf("Failed to analyze source: %v", err)
		return createErrorResponse(500, fmt.Sprintf("Analysis failed: %v", err))
	}

	// Store analysis results in DynamoDB
	err = dynamoService.CreateSourceAnalysis(ctx, analysisResults)
	if err != nil {
		log.Printf("Failed to store analysis results: %v", err)
		return createErrorResponse(500, "Failed to store analysis results")
	}

	// Update source submission status
	sourceSubmission.Status = models.SourceStatusAnalysisComplete
	err = dynamoService.CreateSourceSubmission(ctx, sourceSubmission) // This will update the existing record
	if err != nil {
		log.Printf("Failed to update source submission status: %v", err)
		// Continue anyway since analysis is done
	}

	log.Printf("Successfully completed analysis for source: %s", event.SourceID)

	responseBody := ResponseBody{
		Success:    true,
		Message:    "Source analysis completed successfully",
		SourceID:   event.SourceID,
		AnalysisID: analysisResults.SourceID,
	}

	return createSuccessResponse(responseBody)
}

// analyzeSource performs comprehensive analysis of a submitted source
func analyzeSource(ctx context.Context, submission *models.SourceSubmission) (*models.SourceAnalysis, error) {
	log.Printf("Starting analysis of source: %s (%s)", submission.SourceName, submission.BaseURL)

	analysis := &models.SourceAnalysis{
		SourceID:        submission.SourceID,
		AnalysisVersion: "1.0",
		Status:          "analysis_complete",
	}

	// Step 1: Website discovery using Jina
	discoveryResults, err := performWebsiteDiscovery(ctx, submission)
	if err != nil {
		log.Printf("Website discovery failed: %v", err)
		analysis.Issues = append(analysis.Issues, fmt.Sprintf("Discovery failed: %v", err))
	} else {
		analysis.DiscoveredPatterns = *discoveryResults
		log.Printf("Discovery found %d content pages", len(discoveryResults.ContentPages))
	}

	// Step 2: Content extraction testing
	extractionResults, err := performExtractionTesting(ctx, submission, discoveryResults)
	if err != nil {
		log.Printf("Extraction testing failed: %v", err)
		analysis.Issues = append(analysis.Issues, fmt.Sprintf("Extraction testing failed: %v", err))
	} else {
		analysis.ExtractionTestResults = *extractionResults
		log.Printf("Extraction testing found %d items with quality score %.2f", 
			extractionResults.ItemsFound, extractionResults.QualityScore)
	}

	// Step 3: Generate configuration recommendations
	recommendations, err := generateConfigurationRecommendations(ctx, submission, discoveryResults, extractionResults)
	if err != nil {
		log.Printf("Failed to generate recommendations: %v", err)
		analysis.Issues = append(analysis.Issues, fmt.Sprintf("Recommendation generation failed: %v", err))
	} else {
		analysis.RecommendedConfig = *recommendations
		log.Printf("Generated recommendations: %s frequency, %s extraction method", 
			recommendations.ScrapingFrequency, recommendations.PreferredExtraction)
	}

	// Step 4: Calculate overall quality score
	analysis.OverallQualityScore = calculateOverallQualityScore(analysis)

	// Step 5: Generate final recommendations
	if analysis.OverallQualityScore >= 0.7 {
		analysis.Recommendations = append(analysis.Recommendations, "High-quality source recommended for immediate activation")
	} else if analysis.OverallQualityScore >= 0.5 {
		analysis.Recommendations = append(analysis.Recommendations, "Medium-quality source requiring manual review")
	} else {
		analysis.Recommendations = append(analysis.Recommendations, "Low-quality source requiring significant configuration")
	}

	log.Printf("Analysis completed with overall quality score: %.2f", analysis.OverallQualityScore)
	return analysis, nil
}

// performWebsiteDiscovery analyzes the website structure and discovers content patterns
func performWebsiteDiscovery(ctx context.Context, submission *models.SourceSubmission) (*models.DiscoveryPatterns, error) {
	discovery := &models.DiscoveryPatterns{
		ContentPages: make([]models.ContentPage, 0),
	}

	// Use FireCrawl to analyze the main website
	mainPageResponse, err := firecrawlClient.ExtractActivities(submission.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract main page content: %w", err)
	}

	// Analyze hint URLs provided by founder
	for _, hintURL := range submission.HintURLs {
		log.Printf("Analyzing hint URL: %s", hintURL)
		
		response, err := firecrawlClient.ExtractActivities(hintURL)
		if err != nil {
			log.Printf("Failed to extract content from %s: %v", hintURL, err)
			continue
		}

		// Determine page type based on extracted activities
		pageType, confidence := analyzePageTypeFromFireCrawl(response, submission.ExpectedContent)

		contentPage := models.ContentPage{
			URL:        hintURL,
			Type:       pageType,
			Confidence: confidence,
			Title:      extractTitleFromFireCrawlResponse(response),
			Language:   "en", // Assume English for Seattle sources
		}

		discovery.ContentPages = append(discovery.ContentPages, contentPage)
	}

	// Generate CSS selectors using FireCrawl analysis
	if len(discovery.ContentPages) > 0 {
		selectors := generateCSSSelectorsFromFireCrawl(mainPageResponse, discovery.ContentPages[0].Type)
		discovery.DataSelectors = *selectors
	}

	// Check for structured data based on FireCrawl extraction success
	if len(mainPageResponse.Data.Activities) > 0 {
		discovery.StructuredDataFound = true
		discovery.SchemaTypes = []string{"Event", "Place"} // Common types for family activities
	}

	return discovery, nil
}

// performExtractionTesting tests data extraction quality
func performExtractionTesting(ctx context.Context, submission *models.SourceSubmission, discovery *models.DiscoveryPatterns) (*models.ExtractionTestResults, error) {
	if len(discovery.ContentPages) == 0 {
		return nil, fmt.Errorf("no content pages found for testing")
	}

	// Test extraction on the first hint URL
	testURL := discovery.ContentPages[0].URL
	log.Printf("Testing extraction on URL: %s", testURL)

	startTime := time.Now()
	response, err := firecrawlClient.ExtractActivities(testURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract test content: %w", err)
	}
	duration := time.Since(startTime).Milliseconds()

	// Convert FireCrawl activities to sample activities format
	sampleActivities := convertFireCrawlToSampleActivities(response.Data.Activities)

	// Calculate quality metrics
	qualityScore := calculateExtractionQualityScore(sampleActivities)
	
	results := &models.ExtractionTestResults{
		TestURL:      testURL,
		ItemsFound:   len(sampleActivities),
		QualityScore: qualityScore,
		SampleData:   sampleActivities,
		TestDuration: duration,
		Metrics: models.ExtractionMetrics{
			TitleCompleteness:       calculateFieldCompleteness(sampleActivities, "title"),
			DateCompleteness:        calculateFieldCompleteness(sampleActivities, "date"),
			DescriptionCompleteness: calculateFieldCompleteness(sampleActivities, "description"),
			LocationCompleteness:    calculateFieldCompleteness(sampleActivities, "location"),
			PriceCompleteness:       calculateFieldCompleteness(sampleActivities, "price"),
		},
	}

	// Calculate overall completeness
	results.Metrics.OverallCompleteness = (results.Metrics.TitleCompleteness + 
		results.Metrics.DateCompleteness + results.Metrics.DescriptionCompleteness + 
		results.Metrics.LocationCompleteness + results.Metrics.PriceCompleteness) / 5.0

	return results, nil
}

// generateConfigurationRecommendations creates scraping configuration recommendations
func generateConfigurationRecommendations(ctx context.Context, submission *models.SourceSubmission, 
	discovery *models.DiscoveryPatterns, extraction *models.ExtractionTestResults) (*models.RecommendedSourceConfig, error) {
	
	config := &models.RecommendedSourceConfig{
		TargetURLs: submission.HintURLs,
		RateLimit: models.RateLimit{
			RequestsPerMinute:    10, // Conservative default
			DelayBetweenRequests: 6000, // 6 seconds
			ConcurrentRequests:   1,
		},
	}

	// Determine scraping frequency based on content type and quality
	if extraction != nil && extraction.QualityScore > 0.8 {
		config.ScrapingFrequency = "daily"
		config.EstimatedContentVolatility = 0.7 // High quality sources tend to update frequently
	} else if extraction != nil && extraction.QualityScore > 0.5 {
		config.ScrapingFrequency = "weekly"
		config.EstimatedContentVolatility = 0.4
	} else {
		config.ScrapingFrequency = "monthly"
		config.EstimatedContentVolatility = 0.2
	}

	// Determine preferred extraction method
	if discovery.StructuredDataFound {
		config.PreferredExtraction = "structured-data"
	} else if len(discovery.ContentPages) > 0 {
		config.PreferredExtraction = "html"
	} else {
		config.PreferredExtraction = "manual"
	}

	// Estimate items per scrape
	if extraction != nil {
		config.EstimatedItemsPerScrape = fmt.Sprintf("%d-%d", extraction.ItemsFound, extraction.ItemsFound*2)
	} else {
		config.EstimatedItemsPerScrape = "5-15"
	}

	// Use discovered selectors
	if discovery != nil {
		config.BestSelectors = discovery.DataSelectors
	}

	return config, nil
}

// Helper functions for FireCrawl integration
func analyzePageTypeFromFireCrawl(response *services.FireCrawlExtractResponse, expectedContent []string) (string, float64) {
	// Analyze the FireCrawl response to determine page type
	if len(response.Data.Activities) > 0 {
		// If we successfully extracted activities, high confidence it's an events page
		return "events", 0.9
	}

	// Fall back to expected content if no activities found
	if len(expectedContent) > 0 {
		return expectedContent[0], 0.5
	}
	return "unknown", 0.3
}

func generateCSSSelectorsFromFireCrawl(response *services.FireCrawlExtractResponse, pageType string) *models.DataSelectors {
	// Generate selectors based on FireCrawl extraction success
	// For now, return default selectors since FireCrawl handles extraction internally
	return &models.DataSelectors{
		Title:       "h1, h2, .title, .event-title",
		Date:        ".date, .event-date, time",
		Description: ".description, .content, p",
		Location:    ".location, .venue, .address",
		Price:       ".price, .cost, .fee",
		AgeRange:    ".age, .age-range, .ages",
	}
}

func convertFireCrawlToSampleActivities(activities []models.Activity) []models.ExtractedActivity {
	sampleActivities := make([]models.ExtractedActivity, len(activities))

	for i, activity := range activities {
		sampleActivities[i] = models.ExtractedActivity{
			Title:       activity.Title,
			Date:        activity.Schedule.StartDate,
			Time:        activity.Schedule.StartTime,
			Description: activity.Description,
			Location:    activity.Location.Name,
			Price:       activity.Pricing.Description,
			AgeRange:    "", // Convert age groups array to string if needed
			Category:    activity.Category,
		}

		// Convert age groups to string
		if len(activity.AgeGroups) > 0 {
			ageGroup := activity.AgeGroups[0]
			if ageGroup.Description != "" {
				sampleActivities[i].AgeRange = ageGroup.Description
			} else {
				sampleActivities[i].AgeRange = ageGroup.Category
			}
		}
	}

	return sampleActivities
}

func extractTitleFromFireCrawlResponse(response *services.FireCrawlExtractResponse) string {
	if len(response.Data.Activities) > 0 {
		return response.Data.Activities[0].Title
	}
	return "Unknown Title"
}

// Helper functions

func calculateExtractionQualityScore(activities []models.ExtractedActivity) float64 {
	if len(activities) == 0 {
		return 0.0
	}
	
	// Simple quality scoring based on field completeness
	totalScore := 0.0
	for _, activity := range activities {
		score := 0.0
		if activity.Title != "" { score += 0.25 }
		if activity.Date != "" { score += 0.25 }
		if activity.Description != "" { score += 0.25 }
		if activity.Location != "" { score += 0.25 }
		totalScore += score
	}
	
	return totalScore / float64(len(activities))
}

func calculateFieldCompleteness(activities []models.ExtractedActivity, field string) float64 {
	if len(activities) == 0 {
		return 0.0
	}
	
	complete := 0
	for _, activity := range activities {
		switch field {
		case "title":
			if activity.Title != "" { complete++ }
		case "date":
			if activity.Date != "" { complete++ }
		case "description":
			if activity.Description != "" { complete++ }
		case "location":
			if activity.Location != "" { complete++ }
		case "price":
			if activity.Price != "" { complete++ }
		}
	}
	
	return float64(complete) / float64(len(activities))
}

func calculateOverallQualityScore(analysis *models.SourceAnalysis) float64 {
	score := 0.0
	factors := 0.0
	
	// Factor in discovery results
	if len(analysis.DiscoveredPatterns.ContentPages) > 0 {
		score += 0.3
		factors += 1.0
	}
	
	// Factor in extraction quality
	if analysis.ExtractionTestResults.QualityScore > 0 {
		score += analysis.ExtractionTestResults.QualityScore * 0.5
		factors += 1.0
	}
	
	// Factor in structured data availability
	if analysis.DiscoveredPatterns.StructuredDataFound {
		score += 0.2
		factors += 1.0
	}
	
	if factors == 0 {
		return 0.0
	}
	
	return score / factors
}

// Response helpers
func createSuccessResponse(body ResponseBody) (SourceAnalyzerResponse, error) {
	bodyBytes, _ := json.Marshal(body)
	return SourceAnalyzerResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(bodyBytes),
	}, nil
}

func createErrorResponse(statusCode int, message string) (SourceAnalyzerResponse, error) {
	body := ResponseBody{
		Success: false,
		Error:   message,
	}
	bodyBytes, _ := json.Marshal(body)
	return SourceAnalyzerResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(bodyBytes),
	}, nil
}

func main() {
	lambda.Start(handleRequest)
}