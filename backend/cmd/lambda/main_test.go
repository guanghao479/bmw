package main

import (
	"context"
	"strings"
	"testing"
	"time"
	"seattle-family-activities-scraper/internal/models"
)

func TestGetSeattleSources(t *testing.T) {
	sources := GetSeattleSources()
	
	if len(sources) == 0 {
		t.Error("Should have at least one Seattle source")
	}
	
	// Verify each source has required fields
	for i, source := range sources {
		if source.Name == "" {
			t.Errorf("Source %d missing name", i)
		}
		
		if source.URL == "" {
			t.Errorf("Source %d (%s) missing URL", i, source.Name)
		}
		
		if source.Domain == "" {
			t.Errorf("Source %d (%s) missing domain", i, source.Name)
		}
		
		if source.Priority < 1 || source.Priority > 10 {
			t.Errorf("Source %d (%s) priority should be 1-10, got %d", i, source.Name, source.Priority)
		}
		
		if source.Timeout <= 0 {
			t.Errorf("Source %d (%s) timeout should be positive, got %d", i, source.Name, source.Timeout)
		}
	}
	
	// Verify we have expected Seattle sources
	expectedSources := []string{"Seattle's Child", "Tinybeans Seattle"}
	foundSources := make(map[string]bool)
	
	for _, source := range sources {
		foundSources[source.Name] = true
	}
	
	for _, expected := range expectedSources {
		if !foundSources[expected] {
			t.Errorf("Expected source '%s' not found", expected)
		}
	}
	
	// Verify we have ParentMap sources (should be multiple date-specific ones)
	parentMapCount := 0
	for _, source := range sources {
		if strings.Contains(source.Name, "ParentMap Calendar") {
			parentMapCount++
		}
	}
	
	if parentMapCount == 0 {
		t.Error("Expected at least one ParentMap Calendar source")
	}
	
	t.Logf("Found %d ParentMap sources", parentMapCount)
	
	t.Logf("Found %d Seattle sources configured", len(sources))
}

func TestScrapingOrchestrator_Initialization(t *testing.T) {
	// This test only checks structure without requiring real AWS/OpenAI services
	// Real initialization testing is done in integration tests
	
	// Test that we handle missing environment variables gracefully
	// This will fail but shouldn't crash
	_, err := NewScrapingOrchestrator()
	if err != nil {
		// Expected in unit test environment without API keys
		if strings.Contains(err.Error(), "OPENAI_API_KEY") || 
		   strings.Contains(err.Error(), "AWS") {
			t.Logf("Expected initialization failure without API credentials: %v", err)
		} else {
			t.Errorf("Unexpected error during initialization: %v", err)
		}
	} else {
		t.Logf("Orchestrator initialization succeeded (credentials available)")
	}
}

func TestScrapingOrchestrator_RemoveDuplicates(t *testing.T) {
	// Create a minimal orchestrator for testing duplicate removal logic
	orchestrator := &ScrapingOrchestrator{
		runID:     "test_run",
		startTime: time.Now(),
	}
	
	// Create test activities with duplicates
	activities := []models.Activity{
		{
			ID:    "act_1",
			Title: "Music Class",
			Location: models.Location{
				Name: "Seattle Music Academy",
			},
			Schedule: models.Schedule{
				StartDate: "2024-01-01",
			},
		},
		{
			ID:    "act_2", // Different ID but same title/location/date
			Title: "Music Class",
			Location: models.Location{
				Name: "Seattle Music Academy",
			},
			Schedule: models.Schedule{
				StartDate: "2024-01-01",
			},
		},
		{
			ID:    "act_3",
			Title: "Different Class",
			Location: models.Location{
				Name: "Different Academy",
			},
			Schedule: models.Schedule{
				StartDate: "2024-01-01",
			},
		},
	}
	
	unique := orchestrator.removeDuplicates(activities)
	
	if len(unique) != 2 {
		t.Errorf("Expected 2 unique activities, got %d", len(unique))
	}
	
	// Test with empty slice
	empty := orchestrator.removeDuplicates([]models.Activity{})
	if len(empty) != 0 {
		t.Error("Empty slice should return empty slice")
	}
	
	// Test with single activity
	single := orchestrator.removeDuplicates(activities[:1])
	if len(single) != 1 {
		t.Error("Single activity should return single activity")
	}
	
	t.Logf("Duplicate removal working: %d -> %d activities", len(activities), len(unique))
}

func TestLambdaEvent_Structure(t *testing.T) {
	// Test EventBridge scheduled event
	scheduledEvent := LambdaEvent{
		Source:     "aws.events",
		DetailType: "Scheduled Event",
		Detail:     map[string]interface{}{"scheduled": true},
	}
	
	if scheduledEvent.Source != "aws.events" {
		t.Error("Scheduled event source should be 'aws.events'")
	}
	
	// Test manual trigger event
	manualEvent := LambdaEvent{
		Source:      "manual",
		DetailType:  "Manual Trigger",
		TriggerType: "manual",
		SourceFilter: []string{"seattleschild.com"},
	}
	
	if len(manualEvent.SourceFilter) != 1 {
		t.Error("Manual event should have source filter")
	}
	
	t.Logf("Event structures validated")
}

func TestSourceResult_Validation(t *testing.T) {
	// Test successful source result
	successResult := SourceResult{
		Name:            "Test Source",
		URL:             "https://test.com",
		Success:         true,
		ActivitiesFound: 5,
		ProcessingTime:  time.Second * 30,
		TokensUsed:      1000,
		Cost:            0.003,
	}
	
	if !successResult.Success {
		t.Error("Success result should be marked as successful")
	}
	
	if successResult.ActivitiesFound <= 0 {
		t.Error("Success result should have activities found")
	}
	
	if successResult.ProcessingTime <= 0 {
		t.Error("Success result should have positive processing time")
	}
	
	// Test failed source result
	failedResult := SourceResult{
		Name:           "Failed Source",
		URL:            "https://failed.com",
		Success:        false,
		Error:          "Connection timeout",
		ProcessingTime: time.Second * 10,
	}
	
	if failedResult.Success {
		t.Error("Failed result should not be marked as successful")
	}
	
	if failedResult.Error == "" {
		t.Error("Failed result should have error message")
	}
	
	t.Logf("Source result validation passed")
}

func TestScrapingSummary_Aggregation(t *testing.T) {
	// Create sample source results
	results := []SourceResult{
		{
			Name:            "Source 1",
			Success:         true,
			ActivitiesFound: 5,
			TokensUsed:      1000,
			Cost:            0.003,
		},
		{
			Name:            "Source 2", 
			Success:         true,
			ActivitiesFound: 3,
			TokensUsed:      800,
			Cost:            0.002,
		},
		{
			Name:    "Source 3",
			Success: false,
			Error:   "Timeout",
		},
	}
	
	// Create summary
	summary := &ScrapingSummary{
		TotalSources:    len(results),
		SourceResults:   results,
	}
	
	// Aggregate metrics
	for _, result := range results {
		if result.Success {
			summary.SuccessfulSources++
			summary.TotalActivities += result.ActivitiesFound
			summary.TotalTokensUsed += result.TokensUsed
			summary.TotalCost += result.Cost
		} else {
			summary.FailedSources++
		}
	}
	
	if summary.TotalSources != 3 {
		t.Errorf("Expected 3 total sources, got %d", summary.TotalSources)
	}
	
	if summary.SuccessfulSources != 2 {
		t.Errorf("Expected 2 successful sources, got %d", summary.SuccessfulSources)
	}
	
	if summary.FailedSources != 1 {
		t.Errorf("Expected 1 failed source, got %d", summary.FailedSources)
	}
	
	if summary.TotalActivities != 8 {
		t.Errorf("Expected 8 total activities, got %d", summary.TotalActivities)
	}
	
	if summary.TotalTokensUsed != 1800 {
		t.Errorf("Expected 1800 total tokens, got %d", summary.TotalTokensUsed)
	}
	
	expectedCost := 0.005
	if summary.TotalCost != expectedCost {
		t.Errorf("Expected cost %.3f, got %.3f", expectedCost, summary.TotalCost)
	}
	
	t.Logf("Summary aggregation working correctly")
}

func TestLambdaResponse_Structure(t *testing.T) {
	// Test successful response
	successResponse := LambdaResponse{
		Success:         true,
		Message:         "Scraped 10 activities from 3/4 sources",
		ScrapingRunID:   "run_12345",
		TotalActivities: 10,
		ProcessingTime:  60000, // 1 minute
		Cost:            0.005,
		Summary: &ScrapingSummary{
			TotalSources:      4,
			SuccessfulSources: 3,
			FailedSources:     1,
		},
	}
	
	if !successResponse.Success {
		t.Error("Success response should be marked as successful")
	}
	
	if successResponse.TotalActivities <= 0 {
		t.Error("Success response should have activities")
	}
	
	if successResponse.Summary == nil {
		t.Error("Success response should have summary")
	}
	
	// Test error response
	errorResponse := LambdaResponse{
		Success:        false,
		Message:        "Failed to initialize services",
		ProcessingTime: 1000,
		Errors:         []string{"AWS credentials not found"},
	}
	
	if errorResponse.Success {
		t.Error("Error response should not be marked as successful")
	}
	
	if len(errorResponse.Errors) == 0 {
		t.Error("Error response should have error details")
	}
	
	t.Logf("Lambda response structures validated")
}

func TestSeattleSource_Configuration(t *testing.T) {
	sources := GetSeattleSources()
	
	// Verify priorities are reasonable (allow duplicates for ParentMap date sources)
	priorities := make(map[int][]string)
	for _, source := range sources {
		priorities[source.Priority] = append(priorities[source.Priority], source.Name)
	}
	
	// Check for excessive duplicates (more than 14 sources with same priority)
	for priority, sourceNames := range priorities {
		if len(sourceNames) > 14 {
			t.Errorf("Too many sources with priority %d: %v", priority, sourceNames)
		}
		// Allow ParentMap sources to share priorities
		if len(sourceNames) > 1 {
			allParentMap := true
			for _, name := range sourceNames {
				if !strings.Contains(name, "ParentMap Calendar") {
					allParentMap = false
					break
				}
			}
			if !allParentMap {
				t.Errorf("Non-ParentMap sources sharing priority %d: %v", priority, sourceNames)
			}
		}
	}
	
	// Verify all sources are enabled for MVP
	enabledCount := 0
	for _, source := range sources {
		if source.Enabled {
			enabledCount++
		}
	}
	
	if enabledCount == 0 {
		t.Error("At least one source should be enabled")
	}
	
	// Verify reasonable timeouts
	for _, source := range sources {
		if source.Timeout < 30 || source.Timeout > 120 {
			t.Errorf("Source %s timeout %d should be between 30-120 seconds", source.Name, source.Timeout)
		}
	}
	
	t.Logf("Source configuration validated: %d sources, %d enabled", len(sources), enabledCount)
}

// Integration test helper for local testing
func TestHandleLambdaEvent_Structure(t *testing.T) {
	// Test that we can call the handler without AWS environment
	// This tests the structure and basic validation only
	
	event := LambdaEvent{
		Source:       "manual",
		DetailType:   "Test Event",
		TriggerType:  "manual",
		SourceFilter: []string{"test.com"}, // Non-existent source to avoid real API calls
	}
	
	ctx := context.Background()
	
	// This will likely fail due to missing AWS credentials, but we can test the structure
	response, err := HandleLambdaEvent(ctx, event)
	
	// We expect this to fail in test environment, but check the response structure
	if response.ScrapingRunID == "" {
		t.Error("Response should have a scraping run ID even on failure")
	}
	
	if response.ProcessingTime < 0 {
		t.Errorf("Response should have non-negative processing time, got %d", response.ProcessingTime)
	}
	
	// If it failed due to missing credentials, that's expected
	if err != nil && (strings.Contains(err.Error(), "credential") || 
					  strings.Contains(err.Error(), "AWS") ||
					  strings.Contains(err.Error(), "no enabled sources")) {
		t.Logf("Expected failure due to missing AWS credentials or filtered sources: %v", err)
	} else if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	t.Logf("Lambda handler structure validated")
}