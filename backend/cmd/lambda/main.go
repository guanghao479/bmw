package main

// Force redeploy: Fixed ParentMap extraction to get all 20 activities

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"seattle-family-activities-scraper/internal/models"
	"seattle-family-activities-scraper/internal/services"
)

// LambdaEvent represents the EventBridge trigger event
type LambdaEvent struct {
	Source       string                 `json:"source"`
	DetailType   string                 `json:"detail-type"`
	Detail       map[string]interface{} `json:"detail"`
	TriggerType  string                 `json:"trigger-type,omitempty"`  // manual, scheduled, webhook
	SourceFilter []string               `json:"source-filter,omitempty"` // optional filter for specific sources
}

// LambdaResponse represents the function response
type LambdaResponse struct {
	Success         bool                    `json:"success"`
	Message         string                  `json:"message"`
	ScrapingRunID   string                  `json:"scraping_run_id"`
	TotalActivities int                     `json:"total_activities"`
	ProcessingTime  int64                   `json:"processing_time_ms"`
	Cost            float64                 `json:"estimated_cost"`
	Summary         *ScrapingSummary        `json:"summary"`
	Errors          []string                `json:"errors,omitempty"`
}

// ScrapingSummary provides detailed results
type ScrapingSummary struct {
	TotalSources         int                   `json:"total_sources"`
	SuccessfulSources    int                   `json:"successful_sources"`
	FailedSources        int                   `json:"failed_sources"`
	TotalActivities      int                   `json:"total_activities"`
	NewActivities        int                   `json:"new_activities"`
	DuplicatesRemoved    int                   `json:"duplicates_removed"`
	SourceResults        []SourceResult        `json:"source_results"`
	TotalTokensUsed      int                   `json:"total_tokens_used"`
	TotalCost            float64               `json:"total_cost"`
	UploadedFiles        []string              `json:"uploaded_files"`
	AverageQualityScore  float64               `json:"average_quality_score"`  // Average quality across all sources
	QualityBreakdown     map[string]interface{} `json:"quality_breakdown"`      // Aggregated quality metrics
}

// SourceResult represents the result from scraping a single source
type SourceResult struct {
	Name            string        `json:"name"`
	URL             string        `json:"url"`
	Success         bool          `json:"success"`
	ActivitiesFound int           `json:"activities_found"`
	ProcessingTime  time.Duration `json:"processing_time"`
	TokensUsed      int           `json:"tokens_used,omitempty"`
	Cost            float64       `json:"cost,omitempty"`
	Error           string        `json:"error,omitempty"`
	QualityScore    float64       `json:"quality_score,omitempty"`    // 0-100 quality score
	QualityReport   map[string]interface{} `json:"quality_report,omitempty"` // Detailed quality breakdown
}

// SeattleSource represents a Seattle family activity source
type SeattleSource struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Domain      string `json:"domain"`
	Priority    int    `json:"priority"`    // 1-10, higher = more important
	Enabled     bool   `json:"enabled"`
	Timeout     int    `json:"timeout"`     // timeout in seconds
	RetryCount  int    `json:"retry_count"`
}

// GetSeattleSources returns the list of Seattle family activity sources to scrape
func GetSeattleSources() []SeattleSource {
	return []SeattleSource{
		{
			Name:       "Seattle's Child",
			URL:        "https://www.seattleschild.com/events-calendar/",
			Domain:     "seattleschild.com",
			Priority:   10,
			Enabled:    false, // Temporarily disabled due to 403 protection
			Timeout:    90, // Increased timeout for anti-scraping protection
			RetryCount: 3,  // More retries for 403 errors
		},
		{
			Name:       "ParentMap Calendar",
			URL:        "https://www.parentmap.com/calendar",
			Domain:     "parentmap.com",
			Priority:   9,
			Enabled:    true,
			Timeout:    60,
			RetryCount: 2,
		},
		{
			Name:       "Tinybeans Seattle",
			URL:        "https://tinybeans.com/seattle",
			Domain:     "tinybeans.com",
			Priority:   8,
			Enabled:    true,
			Timeout:    60,
			RetryCount: 2,
		},
		{
			Name:       "West Seattle Macaroni KID",
			URL:        "https://westseattle.macaronikid.com/",
			Domain:     "westseattle.macaronikid.com",
			Priority:   7,
			Enabled:    true,
			Timeout:    60,
			RetryCount: 2,
		},
		{
			Name:       "Seattle Fun for Kids",
			URL:        "https://www.seattlefunforkids.com/",
			Domain:     "seattlefunforkids.com",
			Priority:   6,
			Enabled:    false, // Disabled due to DNS issues (ENOTFOUND)
			Timeout:    60,
			RetryCount: 1,
		},
		{
			Name:       "PEPS Events",
			URL:        "https://www.peps.org/",
			Domain:     "peps.org",
			Priority:   5,
			Enabled:    true,
			Timeout:    90, // Increased timeout for SSL certificate issues
			RetryCount: 3,  // More retries for SSL handshake problems
		},
	}
}

// ScrapingOrchestrator handles the complete scraping workflow
type ScrapingOrchestrator struct {
	jinaClient   *services.JinaClient
	openaiClient *services.OpenAIClient
	s3Client     *services.S3Client
	runID        string
	startTime    time.Time
}

// NewScrapingOrchestrator creates a new orchestrator with all required services
func NewScrapingOrchestrator() (*ScrapingOrchestrator, error) {
	// Initialize Jina client
	jinaClient := services.NewJinaClient()

	// Initialize OpenAI client
	openaiClient := services.NewOpenAIClient()

	// Initialize S3 client
	s3Client, err := services.NewS3Client()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
	}

	// Generate run ID
	runID := models.GenerateScrapingRunID(time.Now())

	return &ScrapingOrchestrator{
		jinaClient:   jinaClient,
		openaiClient: openaiClient,
		s3Client:     s3Client,
		runID:        runID,
		startTime:    time.Now(),
	}, nil
}

// ScrapeSource scrapes a single source and returns the result
func (so *ScrapingOrchestrator) ScrapeSource(source SeattleSource) SourceResult {
	start := time.Now()
	
	result := SourceResult{
		Name: source.Name,
		URL:  source.URL,
	}

	log.Printf("Starting to scrape source: %s (%s)", source.Name, source.URL)

	// Pre-process source-specific configurations
	if err := so.prepareSourceSpecificConfig(source); err != nil {
		result.Error = fmt.Sprintf("Source configuration failed: %v", err)
		result.ProcessingTime = time.Since(start)
		log.Printf("Failed to configure source %s: %v", source.Name, err)
		return result
	}

	// Step 1: Extract content with Jina (with source-specific retry logic)
	content, err := so.extractContentWithRetries(source)
	if err != nil {
		result.Error = fmt.Sprintf("Content extraction failed: %v", err)
		result.ProcessingTime = time.Since(start)
		log.Printf("Failed to extract content from %s: %v", source.Name, err)
		return result
	}

	log.Printf("Extracted %d characters from %s", len(content), source.Name)

	// Skip if content is too short
	if len(content) < 500 {
		result.Error = fmt.Sprintf("Content too short (%d chars)", len(content))
		result.ProcessingTime = time.Since(start)
		log.Printf("Content too short for %s: %d characters", source.Name, len(content))
		return result
	}

	// Step 2: Extract activities with OpenAI
	openaiResponse, err := so.openaiClient.ExtractActivities(content, source.URL)
	if err != nil {
		result.Error = fmt.Sprintf("OpenAI extraction failed: %v", err)
		result.ProcessingTime = time.Since(start)
		log.Printf("Failed to extract activities from %s: %v", source.Name, err)
		return result
	}

	// Step 3: Validate extracted activities
	issues := so.openaiClient.ValidateExtractionResponse(openaiResponse)
	if len(issues) > 0 {
		log.Printf("Validation issues for %s: %v", source.Name, issues)
		// Log issues but don't fail - some issues may be acceptable
	}

	// Step 4: Calculate quality score and generate report
	qualityScore := so.openaiClient.CalculateQualityScore(openaiResponse)
	qualityReport := so.openaiClient.GenerateQualityReport(openaiResponse)

	// Success
	result.Success = true
	result.ActivitiesFound = openaiResponse.TotalFound
	result.TokensUsed = openaiResponse.TokensUsed
	result.Cost = openaiResponse.EstimatedCost
	result.QualityScore = qualityScore
	result.QualityReport = qualityReport
	result.ProcessingTime = time.Since(start)

	log.Printf("Successfully scraped %s: %d activities, %d tokens, $%.4f, quality: %.1f%%", 
		source.Name, result.ActivitiesFound, result.TokensUsed, result.Cost, qualityScore)

	return result
}

// prepareSourceSpecificConfig configures source-specific settings before scraping
func (so *ScrapingOrchestrator) prepareSourceSpecificConfig(source SeattleSource) error {
	switch source.Domain {
	case "seattleschild.com":
		// Add more realistic user agents for anti-scraping protection
		additionalUserAgents := []string{
			"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			"Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		}
		for _, ua := range additionalUserAgents {
			so.jinaClient.AddUserAgent(ua)
		}
		log.Printf("Enhanced user agent rotation for %s", source.Domain)
		
	case "peps.org":
		// No specific preparation needed - SSL issues are handled in transport config
		log.Printf("Using enhanced SSL configuration for %s", source.Domain)
		
	case "seattlefunforkids.com":
		// This source is disabled due to DNS issues, should not reach here
		return fmt.Errorf("source %s is disabled due to DNS resolution issues", source.Domain)
	}
	
	return nil
}

// extractContentWithRetries handles source-specific retry logic
func (so *ScrapingOrchestrator) extractContentWithRetries(source SeattleSource) (string, error) {
	var lastErr error
	
	for attempt := 0; attempt < source.RetryCount; attempt++ {
		content, err := so.jinaClient.ExtractContent(source.URL)
		if err == nil {
			return content, nil
		}
		
		lastErr = err
		
		// Source-specific error handling
		if err := so.handleSourceSpecificError(source, err, attempt); err != nil {
			return "", err
		}
		
		// Wait before retry
		if attempt < source.RetryCount-1 {
			waitTime := time.Duration(attempt+1) * 2 * time.Second
			log.Printf("Retrying %s in %v (attempt %d/%d): %v", 
				source.Name, waitTime, attempt+1, source.RetryCount, err)
			time.Sleep(waitTime)
		}
	}
	
	return "", fmt.Errorf("failed after %d attempts: %w", source.RetryCount, lastErr)
}

// handleSourceSpecificError provides source-specific error handling
func (so *ScrapingOrchestrator) handleSourceSpecificError(source SeattleSource, err error, attempt int) error {
	errStr := err.Error()
	
	switch source.Domain {
	case "seattleschild.com":
		if strings.Contains(errStr, "403") || strings.Contains(errStr, "forbidden") {
			log.Printf("Anti-scraping protection detected for %s, attempt %d", source.Name, attempt+1)
			// Continue retrying - the enhanced headers might work
			return nil
		}
		
	case "peps.org":
		if strings.Contains(errStr, "tls") || strings.Contains(errStr, "certificate") || strings.Contains(errStr, "ssl") {
			log.Printf("SSL/TLS issue detected for %s, attempt %d", source.Name, attempt+1)
			// Continue retrying - the enhanced TLS config might work
			return nil
		}
		
	case "seattlefunforkids.com":
		if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "ENOTFOUND") {
			// Don't retry DNS failures
			return fmt.Errorf("DNS resolution failed for %s: %w", source.Domain, err)
		}
	}
	
	// Default: continue retrying for other errors
	return nil
}

// ScrapeAllSources orchestrates scraping from all enabled Seattle sources
func (so *ScrapingOrchestrator) ScrapeAllSources(sources []SeattleSource, sourceFilter []string) (*ScrapingSummary, []models.Activity, error) {
	log.Printf("Starting scraping run %s with %d sources", so.runID, len(sources))

	// Filter sources if filter is provided
	if len(sourceFilter) > 0 {
		filteredSources := []SeattleSource{}
		for _, source := range sources {
			for _, filter := range sourceFilter {
				if source.Domain == filter || source.Name == filter {
					filteredSources = append(filteredSources, source)
					break
				}
			}
		}
		sources = filteredSources
		log.Printf("Filtered to %d sources based on filter", len(sources))
	}

	// Filter to enabled sources only
	enabledSources := []SeattleSource{}
	for _, source := range sources {
		if source.Enabled {
			enabledSources = append(enabledSources, source)
		}
	}
	sources = enabledSources

	if len(sources) == 0 {
		return nil, nil, fmt.Errorf("no enabled sources to scrape")
	}

	log.Printf("Scraping %d enabled sources", len(sources))

	// Prepare for concurrent scraping
	var wg sync.WaitGroup
	results := make([]SourceResult, len(sources))
	allActivities := make([][]models.Activity, len(sources))

	// Scrape sources concurrently (limited concurrency)
	maxConcurrency := 3 // Limit to avoid overwhelming APIs
	semaphore := make(chan struct{}, maxConcurrency)

	for i, source := range sources {
		wg.Add(1)
		go func(index int, src SeattleSource) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Scrape the source
			result := so.ScrapeSource(src)
			results[index] = result

			// If successful, extract activities
			if result.Success {
				// Re-extract activities for final collection
				content, err := so.jinaClient.ExtractContent(src.URL)
				if err == nil && len(content) >= 500 {
					openaiResponse, err := so.openaiClient.ExtractActivities(content, src.URL)
					if err == nil {
						allActivities[index] = openaiResponse.Activities
					}
				}
			}
		}(i, source)
	}

	wg.Wait()
	log.Printf("Completed scraping all sources")

	// Aggregate results
	summary := &ScrapingSummary{
		TotalSources:   len(sources),
		SourceResults:  results,
		UploadedFiles:  []string{},
	}

	// Collect all activities and calculate quality metrics
	var finalActivities []models.Activity
	var qualityScores []float64
	var totalQualityBreakdown = map[string]interface{}{
		"with_images": 0,
		"with_coordinates": 0,
		"with_specific_times": 0,
		"with_registration_url": 0,
		"with_detail_url": 0,
		"with_contact_info": 0,
	}
	
	for i, activities := range allActivities {
		if activities != nil {
			finalActivities = append(finalActivities, activities...)
			summary.SuccessfulSources++
		} else {
			summary.FailedSources++
		}
		
		// Aggregate metrics
		result := results[i]
		summary.TotalTokensUsed += result.TokensUsed
		summary.TotalCost += result.Cost
		
		// Aggregate quality metrics
		if result.QualityScore > 0 {
			qualityScores = append(qualityScores, result.QualityScore)
		}
		
		if result.QualityReport != nil {
			if breakdown, ok := result.QualityReport["quality_breakdown"].(map[string]interface{}); ok {
				for key, value := range breakdown {
					if currentVal, exists := totalQualityBreakdown[key]; exists {
						if intVal, ok := value.(int); ok {
							totalQualityBreakdown[key] = currentVal.(int) + intVal
						}
					}
				}
			}
		}
	}
	
	// Calculate average quality score
	if len(qualityScores) > 0 {
		var totalScore float64
		for _, score := range qualityScores {
			totalScore += score
		}
		summary.AverageQualityScore = totalScore / float64(len(qualityScores))
	}
	
	summary.QualityBreakdown = totalQualityBreakdown

	// Remove duplicates
	uniqueActivities := so.removeDuplicates(finalActivities)
	summary.TotalActivities = len(uniqueActivities)
	summary.NewActivities = len(uniqueActivities) // For simplicity, treat all as new in MVP
	summary.DuplicatesRemoved = len(finalActivities) - len(uniqueActivities)

	log.Printf("Aggregated %d unique activities from %d total (%d duplicates removed) with %.1f%% average quality score", 
		len(uniqueActivities), len(finalActivities), summary.DuplicatesRemoved, summary.AverageQualityScore)

	return summary, uniqueActivities, nil
}

// removeDuplicates removes duplicate activities based on title and location similarity
func (so *ScrapingOrchestrator) removeDuplicates(activities []models.Activity) []models.Activity {
	if len(activities) <= 1 {
		return activities
	}

	unique := make([]models.Activity, 0, len(activities))
	seen := make(map[string]bool)

	for _, activity := range activities {
		// Create a simple key for duplicate detection
		key := fmt.Sprintf("%s|%s|%s", 
			strings.ToLower(activity.Title),
			strings.ToLower(activity.Location.Name),
			activity.Schedule.StartDate)

		if !seen[key] {
			seen[key] = true
			unique = append(unique, activity)
		}
	}

	return unique
}

// UploadResults uploads the scraping results to S3
func (so *ScrapingOrchestrator) UploadResults(activities []models.Activity, summary *ScrapingSummary) error {
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")

	// Upload latest activities for frontend consumption
	if len(activities) > 0 {
		latestResult, err := so.s3Client.UploadLatestActivities(activities)
		if err != nil {
			return fmt.Errorf("failed to upload latest activities: %w", err)
		}
		summary.UploadedFiles = append(summary.UploadedFiles, latestResult.Key)
		log.Printf("Uploaded latest activities: %s", latestResult.PublicURL)

		// Create backup
		backupResult, err := so.s3Client.BackupActivities(activities)
		if err != nil {
			log.Printf("Warning: Failed to create backup: %v", err)
		} else {
			summary.UploadedFiles = append(summary.UploadedFiles, backupResult.Key)
			log.Printf("Created backup: %s", backupResult.Key)
		}
	}

	// Upload scraping run data
	scrapingRun := &models.ScrapingRun{
		ID:                so.runID,
		StartedAt:         so.startTime,
		CompletedAt:       time.Now(),
		Duration:          time.Since(so.startTime).Milliseconds(),
		Status:            models.ScrapingStatusCompleted,
		TotalSources:      summary.TotalSources,
		SuccessfulSources: summary.SuccessfulSources,
		FailedSources:     summary.FailedSources,
		TotalActivities:   summary.TotalActivities,
		NewActivities:     summary.NewActivities,
		DuplicatesRemoved: summary.DuplicatesRemoved,
		TotalTokensUsed:   summary.TotalTokensUsed,
		EstimatedCost:     summary.TotalCost,
		TriggerType:       models.TriggerTypeScheduled,
		ScrapingVersion:   "1.0.0",
		Jobs:              []models.ScrapingJob{}, // Could be enhanced to include individual jobs
	}

	runKey := fmt.Sprintf("scraping-runs/%s.json", timestamp)
	runResult, err := so.s3Client.UploadScrapingRun(scrapingRun, runKey)
	if err != nil {
		log.Printf("Warning: Failed to upload scraping run: %v", err)
	} else {
		summary.UploadedFiles = append(summary.UploadedFiles, runResult.Key)
		log.Printf("Uploaded scraping run: %s", runResult.Key)
	}

	return nil
}

// HandleLambdaEvent is the main Lambda handler function
func HandleLambdaEvent(ctx context.Context, event LambdaEvent) (LambdaResponse, error) {
	start := time.Now()

	log.Printf("Lambda function started with event: %+v", event)

	// Initialize orchestrator
	orchestrator, err := NewScrapingOrchestrator()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to initialize orchestrator: %v", err)
		log.Printf("ERROR: %s", errorMsg)
		return LambdaResponse{
			Success:        false,
			Message:        errorMsg,
			ProcessingTime: time.Since(start).Milliseconds(),
		}, err
	}

	// Get Seattle sources
	sources := GetSeattleSources()

	// Determine trigger type
	triggerType := event.TriggerType
	if triggerType == "" {
		if event.Source == "aws.events" {
			triggerType = "scheduled"
		} else {
			triggerType = "manual"
		}
	}

	log.Printf("Starting scraping with trigger type: %s", triggerType)

	// Scrape all sources
	summary, activities, err := orchestrator.ScrapeAllSources(sources, event.SourceFilter)
	if err != nil {
		errorMsg := fmt.Sprintf("Scraping failed: %v", err)
		log.Printf("ERROR: %s", errorMsg)
		return LambdaResponse{
			Success:        false,
			Message:        errorMsg,
			ScrapingRunID:  orchestrator.runID,
			ProcessingTime: time.Since(start).Milliseconds(),
		}, err
	}

	// Upload results to S3
	err = orchestrator.UploadResults(activities, summary)
	if err != nil {
		log.Printf("WARNING: Failed to upload some results: %v", err)
		// Don't fail the whole function for upload issues
	}

	// Prepare response
	response := LambdaResponse{
		Success:         summary.SuccessfulSources > 0,
		Message:         fmt.Sprintf("Scraped %d activities from %d/%d sources", summary.TotalActivities, summary.SuccessfulSources, summary.TotalSources),
		ScrapingRunID:   orchestrator.runID,
		TotalActivities: summary.TotalActivities,
		ProcessingTime:  time.Since(start).Milliseconds(),
		Cost:            summary.TotalCost,
		Summary:         summary,
	}

	// Collect errors from failed sources
	var errors []string
	for _, result := range summary.SourceResults {
		if !result.Success && result.Error != "" {
			errors = append(errors, fmt.Sprintf("%s: %s", result.Name, result.Error))
		}
	}
	response.Errors = errors

	log.Printf("Lambda function completed successfully: %s", response.Message)
	log.Printf("Total processing time: %dms, Cost: $%.4f", response.ProcessingTime, response.Cost)

	return response, nil
}

// main is the entry point for the Lambda function
func main() {
	lambda.Start(HandleLambdaEvent)
}