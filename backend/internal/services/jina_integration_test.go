//go:build integration

package services

import (
	"os"
	"strings"
	"testing"
	"time"
)

// These are integration tests that make real API calls to Jina AI
// Run with: go test -tags=integration ./internal/services -v

func TestJinaClient_RealAPICall(t *testing.T) {
	// Skip if running in CI without proper setup
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client := NewJinaClient()

	// Test with a simple, reliable URL
	testURL := "https://httpbin.org/get"
	content, err := client.ExtractContent(testURL)
	
	if err != nil {
		t.Fatalf("Failed to extract content: %v", err)
	}
	
	if len(content) < 50 {
		t.Errorf("Content too short: %d characters", len(content))
	}
	
	// Content should contain expected elements from httpbin
	if !strings.Contains(content, "httpbin.org") {
		t.Error("Content should contain httpbin.org")
	}
}

func TestJinaClient_SeattleChildWebsite(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client := NewJinaClient()
	
	// Test with Seattle's Child events page
	seattleChildURL := "https://www.seattleschild.com/things-to-do-with-kids-in-seattle-this-weekend/"
	
	content, err := client.ExtractContent(seattleChildURL)
	if err != nil {
		t.Fatalf("Failed to extract Seattle's Child content: %v", err)
	}
	
	// Verify we got substantial content
	if len(content) < 1000 {
		t.Errorf("Content too short for Seattle's Child page: %d characters", len(content))
	}
	
	// Check for expected Seattle-related content
	lowerContent := strings.ToLower(content)
	expectedTerms := []string{
		"seattle",
		"kids",
		"children",
		"activities",
		"events",
	}
	
	for _, term := range expectedTerms {
		if !strings.Contains(lowerContent, term) {
			t.Errorf("Content should contain '%s'", term)
		}
	}
	
	t.Logf("Successfully extracted %d characters from Seattle's Child", len(content))
}

func TestJinaClient_TinybeansWebsite(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client := NewJinaClient()
	
	// Test with Tinybeans Seattle page
	tinybeansURL := "https://tinybeans.com/seattle"
	
	content, err := client.ExtractContent(tinybeansURL)
	if err != nil {
		t.Fatalf("Failed to extract Tinybeans content: %v", err)
	}
	
	// Verify we got substantial content
	if len(content) < 500 {
		t.Errorf("Content too short for Tinybeans page: %d characters", len(content))
	}
	
	// Check for expected content
	lowerContent := strings.ToLower(content)
	if !strings.Contains(lowerContent, "seattle") {
		t.Error("Content should contain 'seattle'")
	}
	
	t.Logf("Successfully extracted %d characters from Tinybeans", len(content))
}

func TestJinaClient_ParentMapWebsite(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client := NewJinaClient()
	
	// Test with ParentMap events page
	parentMapURL := "https://www.parentmap.com/calendar"
	
	content, err := client.ExtractContent(parentMapURL)
	if err != nil {
		t.Fatalf("Failed to extract ParentMap content: %v", err)
	}
	
	// Verify we got substantial content
	if len(content) < 800 {
		t.Errorf("Content too short for ParentMap page: %d characters", len(content))
	}
	
	// Check for expected content
	lowerContent := strings.ToLower(content)
	expectedTerms := []string{
		"events",
		"calendar",
		"family",
	}
	
	for _, term := range expectedTerms {
		if !strings.Contains(lowerContent, term) {
			t.Errorf("Content should contain '%s'", term)
		}
	}
	
	t.Logf("Successfully extracted %d characters from ParentMap", len(content))
}

func TestJinaClient_ExtractContentWithMetadata(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client := NewJinaClient()
	
	testURL := "https://httpbin.org/get"
	response, err := client.ExtractContentWithMetadata(testURL)
	
	if err != nil {
		t.Fatalf("Failed to extract content with metadata: %v", err)
	}
	
	// Verify response structure
	if response.URL != testURL {
		t.Errorf("Expected URL %s, got %s", testURL, response.URL)
	}
	
	if response.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", response.StatusCode)
	}
	
	if response.Length <= 0 {
		t.Error("Content length should be positive")
	}
	
	if response.ProcessingMS <= 0 {
		t.Error("Processing time should be positive")
	}
	
	if len(response.Content) != response.Length {
		t.Errorf("Content length mismatch: expected %d, got %d", response.Length, len(response.Content))
	}
	
	t.Logf("Metadata: Status=%d, Length=%d, ProcessingMS=%d", 
		response.StatusCode, response.Length, response.ProcessingMS)
}

func TestJinaClient_ErrorHandling(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client := NewJinaClient()
	
	// Test with invalid URL
	_, err := client.ExtractContent("https://this-domain-definitely-does-not-exist-12345.com")
	if err == nil {
		t.Error("Expected error for invalid domain")
	}
	
	// Test with empty URL
	_, err = client.ExtractContent("")
	if err == nil {
		t.Error("Expected error for empty URL")
	}
	
	// Test URL validation
	err = client.ValidateURL("not-a-url")
	if err == nil {
		t.Error("Expected error for invalid URL format")
	}
	
	err = client.ValidateURL("https://valid-url.com")
	if err != nil {
		t.Errorf("Valid URL should not return error: %v", err)
	}
}

func TestJinaClient_Performance(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client := NewJinaClient()
	
	testURL := "https://httpbin.org/get"
	
	start := time.Now()
	content, err := client.ExtractContent(testURL)
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Failed to extract content: %v", err)
	}
	
	// Should complete within reasonable time (30 seconds is our timeout)
	if duration > 25*time.Second {
		t.Errorf("Request took too long: %v", duration)
	}
	
	if len(content) < 50 {
		t.Error("Content too short")
	}
	
	t.Logf("Extraction completed in %v, content length: %d", duration, len(content))
}

func TestJinaClient_ConcurrentRequests(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client := NewJinaClient()
	
	urls := []string{
		"https://httpbin.org/get",
		"https://httpbin.org/user-agent",
		"https://httpbin.org/headers",
	}
	
	results := make(chan error, len(urls))
	
	// Launch concurrent requests
	for _, url := range urls {
		go func(u string) {
			_, err := client.ExtractContent(u)
			results <- err
		}(url)
	}
	
	// Collect results
	errors := 0
	for i := 0; i < len(urls); i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent request failed: %v", err)
			errors++
		}
	}
	
	if errors > 0 {
		t.Errorf("Failed %d out of %d concurrent requests", errors, len(urls))
	}
}

func TestJinaClient_IsJinaAvailable(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client := NewJinaClient()
	
	available := client.IsJinaAvailable()
	if !available {
		t.Error("Jina AI should be available for integration tests")
	}
	
	t.Log("Jina AI service is available")
}

// Benchmark test for Jina performance
func BenchmarkJinaClient_ExtractContent(b *testing.B) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		b.Skip("Skipping integration benchmark")
	}

	client := NewJinaClient()
	testURL := "https://httpbin.org/get"
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := client.ExtractContent(testURL)
		if err != nil {
			b.Fatalf("Failed to extract content: %v", err)
		}
	}
}