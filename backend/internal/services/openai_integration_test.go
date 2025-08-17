//go:build integration

package services

import (
	"os"
	"strings"
	"testing"
	"time"
)

// These are integration tests that make real API calls to OpenAI
// Run with: go test -tags=integration ./internal/services -v
// Requires OPENAI_API_KEY environment variable

const testSeattleContent = `
Seattle Music Academy - Fall Classes

Music Classes for Kids
Join us for our fall music program! Classes for all ages in our Capitol Hill location.

Toddler Music & Movement (18 months - 3 years)
Weekly classes on Tuesdays and Thursdays from 10:00-10:45 AM
Parent participation required
16-week session starting September 1st through December 15th
$180 per session, includes all materials
Location: 123 Pine Street, Seattle, WA 98101
Register online at seattlemusicacademy.com

Preschool Piano (ages 3-5)
Individual 30-minute lessons
Flexible scheduling Monday-Friday
$120 per month for weekly lessons
Same location on Capitol Hill
Registration required, some spots still available

Elementary Music Theory (ages 6-10)
Group classes on Saturdays 2:00-3:00 PM
8-week session starting October 1st
$160 for complete session
Prerequisites: basic piano experience

Contact Information:
Phone: (206) 555-1234
Email: info@seattlemusicacademy.com
Address: 123 Pine Street, Seattle, WA 98101
Parking available on street
Wheelchair accessible
`

// Problematic content that might trigger plain text responses
const testProblematicContent = `
<!DOCTYPE html>
<html><head><title>Access Denied</title></head>
<body>
<h1>Access Denied</h1>
<p>Your request has been blocked. Please try again later.</p>
<script>window.location.redirect='/';</script>
</body></html>

Some corrupted characters: ����� ���� ����

Mixed content with JavaScript:
function analytics() { ga('send', 'pageview'); }
var config = {"api": "blocked", "status": 403};

Error: Cannot read property 'content' of undefined
Stack trace: at Object.parse (/app/parser.js:42:13)
`

// Content that looks like an error page
const testErrorPageContent = `
Sorry, we're experiencing technical difficulties.
The page you requested is temporarily unavailable.
Please check back later or contact support.
Error ID: ERR_503_UNAVAILABLE
Server: nginx/1.18.0
Time: 2025-08-17 05:07:27 UTC
`

const testSeattleLibraryContent = `
Seattle Public Library - Children's Programs

Storytime for Toddlers
Ages 2-3 with caregiver
Tuesdays at 10:30 AM
Central Library, Level 1 Children's Center
Free program, no registration required
Songs, stories, and movement activities
1000 4th Ave, Seattle, WA 98104

Baby Lapsit
Ages 0-18 months with caregiver
Wednesdays at 11:00 AM
Free program at Ballard Branch
Simple songs and finger plays
5614 22nd Ave NW, Seattle, WA 98107

Craft Time for Kids
Ages 4-8
Saturdays at 2:00 PM
Capitol Hill Branch Library
Free supplies provided
Drop-in program, no registration needed
712 E Pine St, Seattle, WA 98122

Teen Writing Workshop
Ages 13-18
Fridays at 4:00 PM
University Branch
Free program with published author
Registration required online
5009 Roosevelt Way NE, Seattle, WA 98105
`

func TestOpenAIClient_ExtractActivities_MusicAcademy(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping OpenAI integration test")
	}

	client := NewOpenAIClient()
	
	response, err := client.ExtractActivities(testSeattleContent, "https://seattlemusicacademy.com/classes")
	if err != nil {
		t.Fatalf("Failed to extract activities: %v", err)
	}
	
	// Verify response structure
	if response.TotalFound == 0 {
		t.Error("Should have extracted at least one activity")
	}
	
	if response.TokensUsed <= 0 {
		t.Error("Should have used some tokens")
	}
	
	if response.ProcessingMS <= 0 {
		t.Error("Processing time should be positive")
	}
	
	if response.EstimatedCost <= 0 {
		t.Error("Estimated cost should be positive")
	}
	
	// Verify activities
	if len(response.Activities) == 0 {
		t.Fatal("No activities extracted")
	}
	
	// Check first activity (should be toddler music class)
	activity := response.Activities[0]
	
	if activity.Title == "" {
		t.Error("Activity should have a title")
	}
	
	if activity.Type == "" {
		t.Error("Activity should have a type")
	}
	
	if activity.Category == "" {
		t.Error("Activity should have a category")
	}
	
	// Should be in Seattle
	if !strings.Contains(strings.ToLower(activity.Location.City), "seattle") {
		t.Errorf("Activity should be in Seattle, got: %s", activity.Location.City)
	}
	
	// Should have source information
	if activity.Source.URL == "" {
		t.Error("Activity should have source URL")
	}
	
	if activity.Source.Domain == "" {
		t.Error("Activity should have source domain")
	}
	
	// Should have generated ID
	if activity.ID == "" {
		t.Error("Activity should have generated ID")
	}
	
	t.Logf("Extracted %d activities in %dms using %d tokens (cost: $%.4f)", 
		response.TotalFound, response.ProcessingMS, response.TokensUsed, response.EstimatedCost)
	
	for i, act := range response.Activities {
		t.Logf("Activity %d: %s (%s) at %s", i+1, act.Title, act.Type, act.Location.Name)
	}
}

func TestOpenAIClient_ExtractActivities_LibraryPrograms(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping OpenAI integration test")
	}

	client := NewOpenAIClient()
	
	response, err := client.ExtractActivities(testSeattleLibraryContent, "https://spl.org/programs-and-services/learning/children")
	if err != nil {
		t.Fatalf("Failed to extract library activities: %v", err)
	}
	
	// Should extract multiple library programs
	if response.TotalFound < 2 {
		t.Errorf("Should have extracted at least 2 library programs, got %d", response.TotalFound)
	}
	
	// Verify all activities are free (library programs)
	for i, activity := range response.Activities {
		if activity.Pricing.Type != "free" {
			t.Errorf("Library activity %d should be free, got: %s", i+1, activity.Pricing.Type)
		}
		
		// Should be in Seattle
		if !strings.Contains(strings.ToLower(activity.Location.City), "seattle") {
			t.Errorf("Library activity %d should be in Seattle, got: %s", i+1, activity.Location.City)
		}
		
		// Provider should be library-related
		if !strings.Contains(strings.ToLower(activity.Provider.Name), "library") {
			t.Errorf("Library activity %d should have library provider, got: %s", i+1, activity.Provider.Name)
		}
	}
	
	t.Logf("Extracted %d library programs", response.TotalFound)
}

func TestOpenAIClient_ValidateExtractionResponse(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping OpenAI integration test")
	}

	client := NewOpenAIClient()
	
	response, err := client.ExtractActivities(testSeattleContent, "https://seattlemusicacademy.com/classes")
	if err != nil {
		t.Fatalf("Failed to extract activities: %v", err)
	}
	
	// Validate the response
	issues := client.ValidateExtractionResponse(response)
	
	if len(issues) > 0 {
		t.Errorf("Validation found %d issues:", len(issues))
		for _, issue := range issues {
			t.Errorf("  - %s", issue)
		}
	}
}

func TestOpenAIClient_RealSeattleWebsite_Integration(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping OpenAI integration test")
	}

	// Test full pipeline: Jina + OpenAI
	jinaClient := NewJinaClient()
	openaiClient := NewOpenAIClient()
	
	// Use a simple Seattle events page
	testURL := "https://www.seattleschild.com/things-to-do-with-kids-in-seattle-this-weekend/"
	
	// Step 1: Extract content with Jina
	content, err := jinaClient.ExtractContent(testURL)
	if err != nil {
		t.Fatalf("Failed to extract content with Jina: %v", err)
	}
	
	if len(content) < 500 {
		t.Skipf("Content too short (%d chars), skipping OpenAI test", len(content))
	}
	
	// Step 2: Extract activities with OpenAI
	response, err := openaiClient.ExtractActivities(content, testURL)
	if err != nil {
		t.Fatalf("Failed to extract activities with OpenAI: %v", err)
	}
	
	// Verify end-to-end results
	t.Logf("Full pipeline completed:")
	t.Logf("  - Jina extracted %d characters", len(content))
	t.Logf("  - OpenAI found %d activities", response.TotalFound)
	t.Logf("  - Processing time: %dms", response.ProcessingMS)
	t.Logf("  - Tokens used: %d", response.TokensUsed)
	t.Logf("  - Estimated cost: $%.4f", response.EstimatedCost)
	
	// Validate results
	if response.TotalFound > 0 {
		issues := openaiClient.ValidateExtractionResponse(response)
		if len(issues) > 0 {
			t.Logf("Validation issues (may be expected for real content):")
			for _, issue := range issues {
				t.Logf("  - %s", issue)
			}
		}
		
		// Log sample activities
		for i, activity := range response.Activities {
			if i >= 3 { // Only log first 3
				break
			}
			t.Logf("Activity %d: %s (%s) - %s", i+1, activity.Title, activity.Type, activity.Location.Name)
		}
	}
}

func TestOpenAIClient_ErrorHandling(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping OpenAI integration test")
	}

	client := NewOpenAIClient()
	
	// Test with empty content
	_, err := client.ExtractActivities("", "https://example.com")
	if err == nil {
		t.Error("Should return error for empty content")
	}
	
	// Test with very short content
	_, err = client.ExtractActivities("Not enough content", "https://example.com")
	if err == nil {
		t.Error("Should return error for too short content")
	}
	
	// Test with non-Seattle content
	nonSeattleContent := `
	New York Kids Activities
	Fun activities for children in Manhattan and Brooklyn.
	Great classes at 123 Broadway, New York, NY 10001.
	`
	
	response, err := client.ExtractActivities(nonSeattleContent, "https://nykids.com")
	if err != nil {
		t.Fatalf("Extraction should not fail: %v", err)
	}
	
	// Validation should catch non-Seattle activities
	issues := client.ValidateExtractionResponse(response)
	hasLocationIssue := false
	for _, issue := range issues {
		if strings.Contains(issue, "not in Seattle area") {
			hasLocationIssue = true
			break
		}
	}
	
	if !hasLocationIssue && response.TotalFound > 0 {
		t.Error("Validation should catch non-Seattle activities")
	}
}

func TestOpenAIClient_Configuration(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping OpenAI integration test")
	}

	// Test custom configuration
	client := NewOpenAIClientWithConfig("gpt-4o-mini", 0.0, 2000)
	
	if client.GetModel() != "gpt-4o-mini" {
		t.Errorf("Expected model gpt-4o-mini, got %s", client.GetModel())
	}
	
	if client.GetTemperature() != 0.0 {
		t.Errorf("Expected temperature 0.0, got %f", client.GetTemperature())
	}
	
	if client.GetMaxTokens() != 2000 {
		t.Errorf("Expected max tokens 2000, got %d", client.GetMaxTokens())
	}
	
	// Test changing configuration
	client.SetModel("gpt-4o-mini")
	client.SetTemperature(0.2)
	client.SetMaxTokens(3000)
	
	if client.GetModel() != "gpt-4o-mini" {
		t.Error("Model should be updated")
	}
	
	if client.GetTemperature() != 0.2 {
		t.Error("Temperature should be updated")
	}
	
	if client.GetMaxTokens() != 3000 {
		t.Error("Max tokens should be updated")
	}
}

func TestOpenAIClient_PerformanceTracking(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping OpenAI integration test")
	}

	client := NewOpenAIClient()
	
	start := time.Now()
	response, err := client.ExtractActivities(testSeattleContent, "https://test.com")
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Failed to extract activities: %v", err)
	}
	
	// Performance checks
	if duration > 30*time.Second {
		t.Errorf("OpenAI request took too long: %v", duration)
	}
	
	if response.ProcessingMS <= 0 {
		t.Error("Processing time should be tracked")
	}
	
	if response.TokensUsed <= 0 {
		t.Error("Token usage should be tracked")
	}
	
	if response.EstimatedCost <= 0 {
		t.Error("Cost should be estimated")
	}
	
	// Log performance metrics
	t.Logf("OpenAI Performance:")
	t.Logf("  - Total duration: %v", duration)
	t.Logf("  - Reported processing: %dms", response.ProcessingMS)
	t.Logf("  - Tokens used: %d", response.TokensUsed)
	t.Logf("  - Estimated cost: $%.4f", response.EstimatedCost)
	t.Logf("  - Activities found: %d", response.TotalFound)
}

// Benchmark test for OpenAI performance
func BenchmarkOpenAIClient_ExtractActivities(b *testing.B) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		b.Skip("Skipping integration benchmark")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		b.Skip("OPENAI_API_KEY not set, skipping OpenAI benchmark")
	}

	client := NewOpenAIClient()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := client.ExtractActivities(testSeattleContent, "https://test.com")
		if err != nil {
			b.Fatalf("Failed to extract activities: %v", err)
		}
	}
}

// TestOpenAIClient_ProblematicContent tests the fix for JSON parsing issues
// This reproduces the issue where OpenAI returns plain text instead of JSON
func TestOpenAIClient_ProblematicContent(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}
	
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping OpenAI integration test")
	}

	client := NewOpenAIClient()
	
	testCases := []struct {
		name     string
		content  string
		url      string
	}{
		{
			name:    "Corrupted HTML content",
			content: testProblematicContent,
			url:     "https://example.com/blocked",
		},
		{
			name:    "Error page content", 
			content: testErrorPageContent,
			url:     "https://example.com/error",
		},
		{
			name:    "Very short content",
			content: "Page not found. 404 error.",
			url:     "https://example.com/404",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This should NOT panic or fail JSON parsing
			response, err := client.ExtractActivities(tc.content, tc.url)
			
			if err != nil {
				// Check if it's a JSON parsing error (the bug we're fixing)
				if strings.Contains(err.Error(), "invalid character") {
					t.Errorf("JSON parsing failed (the bug we're fixing): %v", err)
				} else if strings.Contains(err.Error(), "too short") {
					// This is expected for very short content
					t.Logf("Expected error for short content: %v", err)
				} else {
					t.Logf("Other error (may be expected): %v", err)
				}
			} else {
				// Success case - should return empty activities for problematic content
				t.Logf("Successfully processed problematic content, got %d activities", len(response.Activities))
				
				// For problematic content, we expect either 0 activities or valid ones
				if len(response.Activities) > 0 {
					// If activities were extracted, they should be valid
					for i, activity := range response.Activities {
						if activity.Title == "" {
							t.Errorf("Activity %d has empty title", i)
						}
						if activity.Type == "" {
							t.Errorf("Activity %d has empty type", i)
						}
					}
				}
			}
		})
	}
}