package services

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"seattle-family-activities-scraper/internal/models"
)

// TestExtractionPipeline tests the complete end-to-end extraction pipeline
func TestExtractionPipeline(t *testing.T) {
	// Skip if no API key
	if os.Getenv("FIRECRAWL_API_KEY") == "" {
		t.Skip("FIRECRAWL_API_KEY not set, skipping integration test")
	}

	tests := []struct {
		name           string
		url            string
		expectedEvents int
		minEvents      int
		maxEvents      int
		validateFunc   func(*testing.T, []models.Activity)
	}{
		{
			name:           "ParentMap Calendar Extraction",
			url:            "https://www.parentmap.com/calendar",
			expectedEvents: 10,
			minEvents:      5,
			maxEvents:      50,
			validateFunc:   validateParentMapActivities,
		},
		{
			name:           "Remlinger Farms Events",
			url:            "https://www.remlingerfarms.com/events",
			expectedEvents: 15,
			minEvents:      5,
			maxEvents:      50,
			validateFunc:   validateRemlingerActivities,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Printf("=== Testing %s ===", tt.name)
			
			// Test the complete pipeline
			activities, diagnostics, err := testCompleteExtractionPipeline(t, tt.url)
			
			// Validate no errors
			if err != nil {
				t.Fatalf("Pipeline failed with error: %v", err)
			}
			
			// Validate we got activities
			if len(activities) == 0 {
				t.Errorf("No activities extracted from %s", tt.url)
				logDiagnostics(t, diagnostics)
				return
			}
			
			// Validate activity count is reasonable
			if len(activities) < tt.minEvents {
				t.Errorf("Too few activities extracted: got %d, expected at least %d", len(activities), tt.minEvents)
			}
			
			if len(activities) > tt.maxEvents {
				t.Errorf("Too many activities extracted: got %d, expected at most %d", len(activities), tt.maxEvents)
			}
			
			log.Printf("Successfully extracted %d activities from %s", len(activities), tt.url)
			
			// Run source-specific validation
			if tt.validateFunc != nil {
				tt.validateFunc(t, activities)
			}
			
			// Validate all activities have required fields
			validateRequiredFields(t, activities)
			
			// Log sample activities for manual verification
			logSampleActivities(t, activities, 3)
		})
	}
}

// testCompleteExtractionPipeline tests the complete pipeline from URL to Activity models
func testCompleteExtractionPipeline(t *testing.T, url string) ([]models.Activity, *ExtractionDiagnostics, error) {
	log.Printf("Starting complete extraction pipeline test for: %s", url)
	
	// Step 1: Initialize FireCrawl client
	firecrawlClient, err := NewFireCrawlClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create FireCrawl client: %w", err)
	}
	
	// Step 2: Extract activities using FireCrawl
	log.Printf("Step 1: Extracting activities from %s", url)
	extractResponse, err := firecrawlClient.ExtractActivities(url)
	if err != nil {
		return nil, nil, fmt.Errorf("FireCrawl extraction failed: %w", err)
	}
	
	log.Printf("FireCrawl extraction completed: Success=%t, Activities=%d", 
		extractResponse.Success, len(extractResponse.Data.Activities))
	
	if !extractResponse.Success {
		return nil, nil, fmt.Errorf("FireCrawl extraction was not successful")
	}
	
	// Get extraction diagnostics
	var diagnostics *ExtractionDiagnostics
	if lastExtractionDiagnostics != nil {
		diagnostics = lastExtractionDiagnostics
	}
	
	// Step 3: Validate extracted activities
	activities := extractResponse.Data.Activities
	if len(activities) == 0 {
		return nil, diagnostics, fmt.Errorf("no activities extracted from FireCrawl")
	}
	
	// Step 4: Test schema conversion pipeline
	log.Printf("Step 2: Testing schema conversion pipeline")
	conversionService := NewSchemaConversionService()
	
	// Create a mock AdminEvent for conversion testing
	adminEvent := &models.AdminEvent{
		EventID:           "test-event-" + fmt.Sprintf("%d", time.Now().Unix()),
		SourceURL:         url,
		SchemaType:        "activities",
		ExtractedAt:       time.Now(),
		RawExtractedData:  convertActivitiesToRawData(activities),
	}
	
	// Test conversion
	conversionResult, err := conversionService.ConvertToActivity(adminEvent)
	if err != nil {
		return nil, diagnostics, fmt.Errorf("schema conversion failed: %w", err)
	}
	
	if conversionResult.Activity == nil {
		return nil, diagnostics, fmt.Errorf("conversion result contains no activity")
	}
	
	log.Printf("Schema conversion completed: Success=%t, Issues=%d, Confidence=%.1f", 
		conversionResult.Activity != nil, len(conversionResult.Issues), conversionResult.ConfidenceScore)
	
	// Step 5: Validate the converted activity
	convertedActivity := *conversionResult.Activity
	if err := validateActivityModel(convertedActivity); err != nil {
		return nil, diagnostics, fmt.Errorf("converted activity validation failed: %w", err)
	}
	
	log.Printf("Complete pipeline test successful: %d activities extracted and validated", len(activities))
	
	return activities, diagnostics, nil
}

// convertActivitiesToRawData converts Activity models back to raw data for conversion testing
func convertActivitiesToRawData(activities []models.Activity) map[string]interface{} {
	rawActivities := make([]map[string]interface{}, len(activities))
	
	for i, activity := range activities {
		rawActivities[i] = map[string]interface{}{
			"title":       activity.Title,
			"description": activity.Description,
			"location": map[string]interface{}{
				"name":    activity.Location.Name,
				"address": activity.Location.Address,
			},
			"schedule": map[string]interface{}{
				"start_date": activity.Schedule.StartDate,
				"start_time": activity.Schedule.StartTime,
				"end_time":   activity.Schedule.EndTime,
			},
			"pricing": activity.Pricing.Description,
		}
		
		// Add age groups if present
		if len(activity.AgeGroups) > 0 {
			ageGroups := make([]string, len(activity.AgeGroups))
			for j, ag := range activity.AgeGroups {
				ageGroups[j] = ag.Description
			}
			rawActivities[i]["age_groups"] = ageGroups
		}
	}
	
	return map[string]interface{}{
		"activities": rawActivities,
	}
}

// validateActivityModel validates that an Activity model has all required fields
func validateActivityModel(activity models.Activity) error {
	if activity.ID == "" {
		return fmt.Errorf("activity missing ID")
	}
	
	if activity.Title == "" {
		return fmt.Errorf("activity missing title")
	}
	
	if activity.Location.Name == "" {
		return fmt.Errorf("activity missing location name")
	}
	
	if activity.Type == "" {
		return fmt.Errorf("activity missing type")
	}
	
	if activity.Category == "" {
		return fmt.Errorf("activity missing category")
	}
	
	if activity.Status == "" {
		return fmt.Errorf("activity missing status")
	}
	
	return nil
}

// validateParentMapActivities validates ParentMap-specific activity data
func validateParentMapActivities(t *testing.T, activities []models.Activity) {
	log.Printf("Validating %d ParentMap activities", len(activities))
	
	for i, activity := range activities {
		// ParentMap activities should have titles
		if activity.Title == "" {
			t.Errorf("ParentMap activity %d missing title", i+1)
		}
		
		// Should have location information
		if activity.Location.Name == "" {
			t.Errorf("ParentMap activity %d ('%s') missing location", i+1, activity.Title)
		}
		
		// Should have some form of schedule information
		if activity.Schedule.StartDate == "" && activity.Schedule.StartTime == "" {
			t.Errorf("ParentMap activity %d ('%s') missing schedule information", i+1, activity.Title)
		}
		
		// Should be categorized appropriately
		if activity.Category == "" {
			t.Errorf("ParentMap activity %d ('%s') missing category", i+1, activity.Title)
		}
		
		// Log first few for manual inspection
		if i < 3 {
			log.Printf("ParentMap Activity %d: %s at %s (%s)", 
				i+1, activity.Title, activity.Location.Name, activity.Schedule.StartDate)
		}
	}
}

// validateRemlingerActivities validates Remlinger Farms-specific activity data
func validateRemlingerActivities(t *testing.T, activities []models.Activity) {
	log.Printf("Validating %d Remlinger Farms activities", len(activities))
	
	for i, activity := range activities {
		// Remlinger activities should have titles
		if activity.Title == "" {
			t.Errorf("Remlinger activity %d missing title", i+1)
		}
		
		// Should have location (likely "Remlinger Farms" or similar)
		if activity.Location.Name == "" {
			t.Errorf("Remlinger activity %d ('%s') missing location", i+1, activity.Title)
		}
		
		// Should have pricing information (many Remlinger events have admission)
		if activity.Pricing.Type == "" {
			t.Errorf("Remlinger activity %d ('%s') missing pricing type", i+1, activity.Title)
		}
		
		// Should be categorized appropriately
		if activity.Category == "" {
			t.Errorf("Remlinger activity %d ('%s') missing category", i+1, activity.Title)
		}
		
		// Log first few for manual inspection
		if i < 3 {
			log.Printf("Remlinger Activity %d: %s at %s (Price: %s)", 
				i+1, activity.Title, activity.Location.Name, activity.Pricing.Description)
		}
	}
}

// validateRequiredFields validates that all activities have the minimum required fields
func validateRequiredFields(t *testing.T, activities []models.Activity) {
	for i, activity := range activities {
		if activity.Title == "" {
			t.Errorf("Activity %d missing required field: title", i+1)
		}
		
		if activity.Location.Name == "" {
			t.Errorf("Activity %d ('%s') missing required field: location.name", i+1, activity.Title)
		}
		
		if activity.Type == "" {
			t.Errorf("Activity %d ('%s') missing required field: type", i+1, activity.Title)
		}
		
		if activity.Category == "" {
			t.Errorf("Activity %d ('%s') missing required field: category", i+1, activity.Title)
		}
		
		if activity.Status == "" {
			t.Errorf("Activity %d ('%s') missing required field: status", i+1, activity.Title)
		}
		
		// Validate that IDs are generated
		if activity.ID == "" {
			t.Errorf("Activity %d ('%s') missing generated ID", i+1, activity.Title)
		}
		
		// Validate timestamps
		if activity.CreatedAt.IsZero() {
			t.Errorf("Activity %d ('%s') missing CreatedAt timestamp", i+1, activity.Title)
		}
		
		if activity.UpdatedAt.IsZero() {
			t.Errorf("Activity %d ('%s') missing UpdatedAt timestamp", i+1, activity.Title)
		}
	}
}

// logSampleActivities logs a sample of activities for manual verification
func logSampleActivities(t *testing.T, activities []models.Activity, count int) {
	if count > len(activities) {
		count = len(activities)
	}
	
	log.Printf("=== Sample Activities (showing %d of %d) ===", count, len(activities))
	
	for i := 0; i < count; i++ {
		activity := activities[i]
		log.Printf("Activity %d:", i+1)
		log.Printf("  Title: %s", activity.Title)
		log.Printf("  Location: %s", activity.Location.Name)
		log.Printf("  Address: %s", activity.Location.Address)
		log.Printf("  Type: %s", activity.Type)
		log.Printf("  Category: %s", activity.Category)
		log.Printf("  Schedule: %s at %s", activity.Schedule.StartDate, activity.Schedule.StartTime)
		log.Printf("  Pricing: %s (%s)", activity.Pricing.Description, activity.Pricing.Type)
		
		if len(activity.AgeGroups) > 0 {
			log.Printf("  Age Groups: %s", activity.AgeGroups[0].Description)
		}
		
		if activity.Description != "" {
			desc := activity.Description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			log.Printf("  Description: %s", desc)
		}
		log.Printf("")
	}
}

// logDiagnostics logs extraction diagnostics for debugging
func logDiagnostics(t *testing.T, diagnostics *ExtractionDiagnostics) {
	if diagnostics == nil {
		log.Printf("No diagnostics available")
		return
	}
	
	log.Printf("=== Extraction Diagnostics ===")
	log.Printf("URL: %s", diagnostics.URL)
	log.Printf("Success: %t", diagnostics.Success)
	log.Printf("Processing Time: %v", diagnostics.ProcessingTime)
	log.Printf("Raw Markdown Length: %d", diagnostics.RawMarkdownLength)
	log.Printf("Extraction Attempts: %d", len(diagnostics.ExtractionAttempts))
	log.Printf("Validation Issues: %d", len(diagnostics.ValidationIssues))
	
	if diagnostics.ErrorMessage != "" {
		log.Printf("Error: %s", diagnostics.ErrorMessage)
	}
	
	// Log extraction attempts
	for i, attempt := range diagnostics.ExtractionAttempts {
		log.Printf("Attempt %d: %s - Success: %t, Events: %d", 
			i+1, attempt.Method, attempt.Success, attempt.EventsFound)
		
		if len(attempt.Issues) > 0 {
			log.Printf("  Issues: %v", attempt.Issues)
		}
	}
	
	// Log validation issues
	for i, issue := range diagnostics.ValidationIssues {
		log.Printf("Issue %d: [%s] %s - %s", 
			i+1, issue.Severity, issue.Field, issue.Message)
	}
	
	// Log sample of raw markdown
	if len(diagnostics.RawMarkdownSample) > 0 {
		log.Printf("Raw Markdown Sample (first 300 chars):")
		sample := diagnostics.RawMarkdownSample
		if len(sample) > 300 {
			sample = sample[:300] + "..."
		}
		log.Printf("%s", sample)
	}
}

// TestAdminInterfaceIntegration tests that extracted activities appear correctly in admin interface
func TestAdminInterfaceIntegration(t *testing.T) {
	// Skip if no API key
	if os.Getenv("FIRECRAWL_API_KEY") == "" {
		t.Skip("FIRECRAWL_API_KEY not set, skipping integration test")
	}
	
	// This test simulates the complete flow from extraction to admin interface
	testURL := "https://www.parentmap.com/calendar"
	
	log.Printf("Testing admin interface integration with %s", testURL)
	
	// Step 1: Extract activities
	firecrawlClient, err := NewFireCrawlClient()
	if err != nil {
		t.Fatalf("Failed to create FireCrawl client: %v", err)
	}
	
	extractResponse, err := firecrawlClient.ExtractActivities(testURL)
	if err != nil {
		t.Fatalf("FireCrawl extraction failed: %v", err)
	}
	
	if len(extractResponse.Data.Activities) == 0 {
		t.Fatalf("No activities extracted")
	}
	
	// Step 2: Convert to AdminEvent format (simulating what the admin API would do)
	adminEvent := &models.AdminEvent{
		EventID:           "test-admin-event-" + fmt.Sprintf("%d", time.Now().Unix()),
		SourceURL:         testURL,
		SchemaType:        "activities",
		ExtractedAt:       time.Now(),
		RawExtractedData:  convertActivitiesToRawData(extractResponse.Data.Activities),
		Status:            "pending_review",
	}
	
	// Step 3: Test schema conversion
	conversionService := NewSchemaConversionService()
	conversionResult, err := conversionService.ConvertToActivity(adminEvent)
	if err != nil {
		t.Fatalf("Schema conversion failed: %v", err)
	}
	
	if conversionResult.Activity == nil {
		t.Fatalf("Conversion produced no activity")
	}
	
	// Step 4: Validate the activity would be suitable for admin review
	activity := *conversionResult.Activity
	
	// Check that activity has all fields needed for admin interface
	if activity.Title == "" {
		t.Errorf("Activity missing title for admin interface")
	}
	
	if activity.Location.Name == "" {
		t.Errorf("Activity missing location for admin interface")
	}
	
	if activity.Description == "" {
		t.Logf("Warning: Activity missing description (may impact admin review)")
	}
	
	// Check that conversion issues are reasonable
	if len(conversionResult.Issues) > 5 {
		t.Errorf("Too many conversion issues (%d) - may indicate extraction problems", len(conversionResult.Issues))
	}
	
	// Check confidence score
	if conversionResult.ConfidenceScore < 0.5 {
		t.Errorf("Low confidence score (%.1f) - may indicate extraction problems", conversionResult.ConfidenceScore)
	}
	
	log.Printf("Admin interface integration test successful:")
	log.Printf("  - Extracted %d activities", len(extractResponse.Data.Activities))
	log.Printf("  - Converted activity: %s", activity.Title)
	log.Printf("  - Confidence score: %.1f", conversionResult.ConfidenceScore)
	log.Printf("  - Conversion issues: %d", len(conversionResult.Issues))
	
	// Log conversion issues for review
	if len(conversionResult.Issues) > 0 {
		log.Printf("Conversion issues:")
		for i, issue := range conversionResult.Issues {
			log.Printf("  %d. %s", i+1, issue)
		}
	}
}

// TestExtractionQualityMetrics tests the quality of extracted data
func TestExtractionQualityMetrics(t *testing.T) {
	// Skip if no API key
	if os.Getenv("FIRECRAWL_API_KEY") == "" {
		t.Skip("FIRECRAWL_API_KEY not set, skipping integration test")
	}
	
	testCases := []struct {
		name string
		url  string
	}{
		{"ParentMap", "https://www.parentmap.com/calendar"},
		{"Remlinger Farms", "https://www.remlingerfarms.com/events"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log.Printf("Testing extraction quality for %s", tc.name)
			
			// Extract activities
			activities, diagnostics, err := testCompleteExtractionPipeline(t, tc.url)
			if err != nil {
				t.Fatalf("Pipeline failed: %v", err)
			}
			
			// Calculate quality metrics
			metrics := calculateQualityMetrics(activities, diagnostics)
			
			// Validate quality thresholds
			if metrics.CompletionRate < 0.8 {
				t.Errorf("Low completion rate: %.1f%% (expected >= 80%%)", metrics.CompletionRate*100)
			}
			
			if metrics.FieldCoverage < 0.6 {
				t.Errorf("Low field coverage: %.1f%% (expected >= 60%%)", metrics.FieldCoverage*100)
			}
			
			if metrics.DataQualityScore < 0.7 {
				t.Errorf("Low data quality score: %.1f (expected >= 0.7)", metrics.DataQualityScore)
			}
			
			log.Printf("Quality metrics for %s:", tc.name)
			log.Printf("  - Completion Rate: %.1f%%", metrics.CompletionRate*100)
			log.Printf("  - Field Coverage: %.1f%%", metrics.FieldCoverage*100)
			log.Printf("  - Data Quality Score: %.1f", metrics.DataQualityScore)
			log.Printf("  - Activities with Dates: %d/%d", metrics.ActivitiesWithDates, len(activities))
			log.Printf("  - Activities with Locations: %d/%d", metrics.ActivitiesWithLocations, len(activities))
			log.Printf("  - Activities with Pricing: %d/%d", metrics.ActivitiesWithPricing, len(activities))
		})
	}
}

// PipelineQualityMetrics represents extraction quality metrics for pipeline testing
type PipelineQualityMetrics struct {
	CompletionRate          float64
	FieldCoverage           float64
	DataQualityScore        float64
	ActivitiesWithDates     int
	ActivitiesWithLocations int
	ActivitiesWithPricing   int
	ActivitiesWithAgeGroups int
}

// calculateQualityMetrics calculates quality metrics for extracted activities
func calculateQualityMetrics(activities []models.Activity, diagnostics *ExtractionDiagnostics) PipelineQualityMetrics {
	if len(activities) == 0 {
		return PipelineQualityMetrics{}
	}
	
	metrics := PipelineQualityMetrics{}
	
	// Count activities with various fields
	for _, activity := range activities {
		if activity.Schedule.StartDate != "" || activity.Schedule.StartTime != "" {
			metrics.ActivitiesWithDates++
		}
		
		if activity.Location.Name != "" {
			metrics.ActivitiesWithLocations++
		}
		
		if activity.Pricing.Type != "" || activity.Pricing.Description != "" {
			metrics.ActivitiesWithPricing++
		}
		
		if len(activity.AgeGroups) > 0 {
			metrics.ActivitiesWithAgeGroups++
		}
	}
	
	totalActivities := float64(len(activities))
	
	// Calculate rates
	dateRate := float64(metrics.ActivitiesWithDates) / totalActivities
	locationRate := float64(metrics.ActivitiesWithLocations) / totalActivities
	pricingRate := float64(metrics.ActivitiesWithPricing) / totalActivities
	ageGroupRate := float64(metrics.ActivitiesWithAgeGroups) / totalActivities
	
	// Completion rate (all activities have title and location)
	completeActivities := 0
	for _, activity := range activities {
		if activity.Title != "" && activity.Location.Name != "" {
			completeActivities++
		}
	}
	metrics.CompletionRate = float64(completeActivities) / totalActivities
	
	// Field coverage (average of all field rates)
	metrics.FieldCoverage = (dateRate + locationRate + pricingRate + ageGroupRate) / 4.0
	
	// Data quality score (weighted average)
	metrics.DataQualityScore = (metrics.CompletionRate*0.4 + 
		locationRate*0.3 + 
		dateRate*0.2 + 
		pricingRate*0.1)
	
	return metrics
}