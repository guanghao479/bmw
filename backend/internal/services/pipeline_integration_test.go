//go:build integration

package services

import (
	"os"
	"strings"
	"testing"
	"time"
)

// These tests verify the complete Jina + OpenAI pipeline with real Seattle websites
// Run with: go test -tags=integration ./internal/services -run TestPipeline -v

// Seattle family activity websites for testing
var seattleTestSources = []struct {
	name string
	url  string
}{
	{"Seattle's Child", "https://www.seattleschild.com/things-to-do-with-kids-in-seattle-this-weekend/"},
	{"ParentMap Calendar", "https://www.parentmap.com/calendar"},
	{"Tinybeans Seattle", "https://tinybeans.com/seattle"},
	{"West Seattle Macaroni KID", "https://westseattle.macaronikid.com/"},
}

func TestPipeline_EndToEnd_SeattleSources(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping pipeline integration test")
	}

	jinaClient := NewJinaClient()
	openaiClient := NewOpenAIClient()
	
	// Track overall results
	var totalActivities int
	var totalCost float64
	var totalTokens int
	var successfulSources int
	var totalProcessingTime int64
	
	t.Logf("Testing complete pipeline with %d Seattle sources", len(seattleTestSources))
	
	for i, source := range seattleTestSources {
		t.Run(source.name, func(t *testing.T) {
			pipelineStart := time.Now()
			
			t.Logf("Processing source %d/%d: %s", i+1, len(seattleTestSources), source.name)
			t.Logf("URL: %s", source.url)
			
			// Step 1: Extract content with Jina
			jinaStart := time.Now()
			content, err := jinaClient.ExtractContent(source.url)
			jinaTime := time.Since(jinaStart)
			
			if err != nil {
				t.Errorf("Jina extraction failed for %s: %v", source.name, err)
				return
			}
			
			t.Logf("  ✓ Jina extracted %d characters in %v", len(content), jinaTime)
			
			if len(content) < 500 {
				t.Logf("  ⚠ Content too short (%d chars), skipping OpenAI processing", len(content))
				return
			}
			
			// Step 2: Extract activities with OpenAI
			openaiStart := time.Now()
			response, err := openaiClient.ExtractActivities(content, source.url)
			openaiTime := time.Since(openaiStart)
			
			if err != nil {
				t.Errorf("OpenAI extraction failed for %s: %v", source.name, err)
				return
			}
			
			t.Logf("  ✓ OpenAI found %d activities in %v", response.TotalFound, openaiTime)
			t.Logf("    - Tokens used: %d", response.TokensUsed)
			t.Logf("    - Estimated cost: $%.4f", response.EstimatedCost)
			
			// Step 3: Validate results
			issues := openaiClient.ValidateExtractionResponse(response)
			if len(issues) > 0 {
				t.Logf("  ⚠ Validation issues found:")
				for _, issue := range issues {
					t.Logf("    - %s", issue)
				}
			} else {
				t.Logf("  ✓ All activities passed validation")
			}
			
			// Step 4: Log sample activities
			if response.TotalFound > 0 {
				t.Logf("  📋 Sample activities:")
				for j, activity := range response.Activities {
					if j >= 3 { // Limit to first 3
						break
					}
					t.Logf("    %d. %s (%s) at %s", j+1, activity.Title, activity.Type, activity.Location.Name)
				}
			}
			
			// Update totals
			totalActivities += response.TotalFound
			totalCost += response.EstimatedCost
			totalTokens += response.TokensUsed
			totalProcessingTime += time.Since(pipelineStart).Milliseconds()
			successfulSources++
			
			t.Logf("  ✓ Pipeline completed in %v", time.Since(pipelineStart))
		})
	}
	
	// Summary report
	t.Logf("\n" + strings.Repeat("=", 60))
	t.Logf("PIPELINE INTEGRATION TEST SUMMARY")
	t.Logf(strings.Repeat("=", 60))
	t.Logf("Sources processed: %d/%d", successfulSources, len(seattleTestSources))
	t.Logf("Total activities found: %d", totalActivities)
	t.Logf("Total tokens used: %d", totalTokens)
	t.Logf("Total estimated cost: $%.4f", totalCost)
	t.Logf("Average processing time: %dms per source", totalProcessingTime/int64(len(seattleTestSources)))
	t.Logf("Average activities per source: %.1f", float64(totalActivities)/float64(successfulSources))
	
	if successfulSources == 0 {
		t.Error("No sources were successfully processed")
	}
}

func TestPipeline_ErrorRecovery(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping pipeline test")
	}

	jinaClient := NewJinaClient()
	openaiClient := NewOpenAIClient()
	
	// Test with problematic URLs
	problemSources := []struct {
		name string
		url  string
		expectJinaFail bool
		expectOpenAIFail bool
	}{
		{"Invalid Domain", "https://this-domain-definitely-does-not-exist-12345.com", true, false},
		{"Empty URL", "", true, false},
		{"Bad Protocol", "ftp://invalid-protocol.com", true, false},
	}
	
	for _, source := range problemSources {
		t.Run(source.name, func(t *testing.T) {
			// Test Jina step
			content, jinaErr := jinaClient.ExtractContent(source.url)
			
			if source.expectJinaFail && jinaErr == nil {
				t.Errorf("Expected Jina to fail for %s but it succeeded", source.name)
			}
			
			if !source.expectJinaFail && jinaErr != nil {
				t.Errorf("Jina unexpectedly failed for %s: %v", source.name, jinaErr)
			}
			
			// If Jina succeeded, test OpenAI step
			if jinaErr == nil && len(content) > 200 {
				_, openaiErr := openaiClient.ExtractActivities(content, source.url)
				
				if source.expectOpenAIFail && openaiErr == nil {
					t.Errorf("Expected OpenAI to fail for %s but it succeeded", source.name)
				}
				
				if !source.expectOpenAIFail && openaiErr != nil {
					t.Errorf("OpenAI unexpectedly failed for %s: %v", source.name, openaiErr)
				}
			}
		})
	}
}

func TestPipeline_Performance_Concurrent(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping pipeline test")
	}

	// Test concurrent processing of multiple sources
	urls := []string{
		"https://httpbin.org/get",
		"https://httpbin.org/user-agent",
		"https://httpbin.org/headers",
	}
	
	results := make(chan PipelineResult, len(urls))
	
	for _, url := range urls {
		go func(u string) {
			result := processPipelineConcurrent(u)
			results <- result
		}(url)
	}
	
	// Collect results
	var successful, failed int
	var totalDuration time.Duration
	
	for i := 0; i < len(urls); i++ {
		result := <-results
		if result.Error != nil {
			failed++
			t.Logf("Failed processing %s: %v", result.URL, result.Error)
		} else {
			successful++
			totalDuration += result.Duration
			t.Logf("Successfully processed %s in %v (found %d activities)", 
				result.URL, result.Duration, result.ActivitiesFound)
		}
	}
	
	if successful == 0 {
		t.Error("No concurrent requests succeeded")
	}
	
	avgDuration := totalDuration / time.Duration(successful)
	t.Logf("Concurrent performance: %d successful, %d failed, avg duration: %v", 
		successful, failed, avgDuration)
}

type PipelineResult struct {
	URL             string
	ActivitiesFound int
	Duration        time.Duration
	Error           error
}

func processPipelineConcurrent(url string) PipelineResult {
	start := time.Now()
	
	openaiClient := NewOpenAIClient()
	
	// Simple test content for concurrent testing
	testContent := `
	Seattle Children's Museum
	Interactive exhibits for kids ages 0-10
	Located at Seattle Center, 305 Harrison St, Seattle, WA 98109
	Admission: $12 for children, $10 for adults
	Open Tuesday-Sunday 10 AM - 5 PM
	Hands-on learning experiences in art, science, and culture
	`
	
	response, err := openaiClient.ExtractActivities(testContent, url)
	if err != nil {
		return PipelineResult{
			URL:      url,
			Duration: time.Since(start),
			Error:    err,
		}
	}
	
	return PipelineResult{
		URL:             url,
		ActivitiesFound: response.TotalFound,
		Duration:        time.Since(start),
		Error:           nil,
	}
}

func TestPipeline_CostTracking(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping pipeline test")
	}

	client := NewOpenAIClient()
	
	// Test with different content sizes to track cost variations
	testCases := []struct {
		name    string
		content string
	}{
		{"Small", "Small event in Seattle for kids"},
		{"Medium", strings.Repeat("Seattle kids activity with details. ", 20)},
		{"Large", strings.Repeat("Detailed Seattle family event with comprehensive information. ", 50)},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := client.ExtractActivities(tc.content, "https://test.com")
			if err != nil {
				t.Fatalf("Failed to extract activities: %v", err)
			}
			
			costPerToken := response.EstimatedCost / float64(response.TokensUsed)
			
			t.Logf("%s content:", tc.name)
			t.Logf("  - Content length: %d chars", len(tc.content))
			t.Logf("  - Tokens used: %d", response.TokensUsed)
			t.Logf("  - Estimated cost: $%.4f", response.EstimatedCost)
			t.Logf("  - Cost per token: $%.6f", costPerToken)
			t.Logf("  - Activities found: %d", response.TotalFound)
		})
	}
}

func TestPipeline_QualityAssurance(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping pipeline test")
	}

	client := NewOpenAIClient()
	
	// Test quality with realistic Seattle content
	qualityTestContent := `
	Pacific Science Center
	200 2nd Ave N, Seattle, WA 98109
	
	Toddler Time
	Ages 6 months to 3 years with caregivers
	Tuesdays and Thursdays, 10:00-11:30 AM
	$15 per child, caregivers free
	Hands-on science activities designed for little ones
	Registration required online
	
	Science on a Sphere Planetarium Shows
	All ages welcome
	Shows every hour from 11 AM to 4 PM
	Included with general admission ($24.95 adults, $19.95 youth)
	Learn about earth, weather, and space
	
	Summer Science Camp
	Ages 6-12 years
	Week-long camps June through August
	Monday-Friday 9 AM - 3 PM
	$275 per week, includes lunch and materials
	Topics include robotics, chemistry, and marine biology
	Extended care available 8 AM - 5 PM for additional fee
	`
	
	response, err := client.ExtractActivities(qualityTestContent, "https://pacsci.org")
	if err != nil {
		t.Fatalf("Failed to extract activities: %v", err)
	}
	
	// Quality checks
	if response.TotalFound < 2 {
		t.Errorf("Should extract multiple activities from detailed content, got %d", response.TotalFound)
	}
	
	// Check for data quality
	for i, activity := range response.Activities {
		t.Logf("Quality check for activity %d: %s", i+1, activity.Title)
		
		// Required fields
		if activity.Title == "" {
			t.Errorf("Activity %d missing title", i+1)
		}
		
		if activity.Location.Name == "" {
			t.Errorf("Activity %d missing location name", i+1)
		}
		
		if activity.Location.Address == "" {
			t.Errorf("Activity %d missing address", i+1)
		}
		
		// Seattle validation
		if !strings.Contains(strings.ToLower(activity.Location.City), "seattle") {
			t.Errorf("Activity %d not in Seattle: %s", i+1, activity.Location.City)
		}
		
		// Pricing validation
		if activity.Pricing.Type == "" {
			t.Errorf("Activity %d missing pricing type", i+1)
		}
		
		// Age group validation
		if len(activity.AgeGroups) == 0 {
			t.Errorf("Activity %d missing age groups", i+1)
		}
		
		t.Logf("  ✓ Passed basic quality checks")
	}
	
	// Validation check
	issues := client.ValidateExtractionResponse(response)
	if len(issues) > 0 {
		t.Logf("Validation issues found (may indicate areas for improvement):")
		for _, issue := range issues {
			t.Logf("  - %s", issue)
		}
	}
	
	t.Logf("Quality assurance completed: %d activities extracted and validated", response.TotalFound)
}