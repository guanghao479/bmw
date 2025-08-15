package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// JinaClient handles content extraction from web pages using Jina AI Reader
type JinaClient struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
}

// JinaResponse represents the response from Jina AI Reader
type JinaResponse struct {
	Content      string `json:"content"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	StatusCode   int    `json:"status_code"`
	Length       int    `json:"length"`
	ProcessingMS int64  `json:"processing_ms"`
}

// NewJinaClient creates a new Jina AI client
func NewJinaClient() *JinaClient {
	return &JinaClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:   "https://r.jina.ai",
		userAgent: "SeattleFamilyActivities/1.0 (+https://github.com/family-activities)",
	}
}

// NewJinaClientWithTimeout creates a new Jina client with custom timeout
func NewJinaClientWithTimeout(timeout time.Duration) *JinaClient {
	return &JinaClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL:   "https://r.jina.ai",
		userAgent: "SeattleFamilyActivities/1.0 (+https://github.com/family-activities)",
	}
}

// ExtractContent extracts clean content from a webpage URL using Jina AI Reader
func (j *JinaClient) ExtractContent(url string) (string, error) {
	startTime := time.Now()
	
	// Validate URL
	if url == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}
	
	// Construct Jina Reader URL
	jinaURL := fmt.Sprintf("%s/%s", j.baseURL, url)
	
	// Create request
	req, err := http.NewRequest("GET", jinaURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("User-Agent", j.userAgent)
	req.Header.Set("Accept", "text/plain")
	
	// Make request
	resp, err := j.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("jina request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("jina returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Read response body
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read jina response: %w", err)
	}
	
	contentStr := string(content)
	
	// Basic validation
	if len(contentStr) < 100 {
		return "", fmt.Errorf("content too short (%d chars), might be an error page", len(contentStr))
	}
	
	// Log processing time (for debugging)
	processingTime := time.Since(startTime)
	if processingTime > 10*time.Second {
		fmt.Printf("Warning: Jina processing took %v for URL: %s\n", processingTime, url)
	}
	
	return contentStr, nil
}

// ExtractContentWithMetadata extracts content and returns detailed metadata
func (j *JinaClient) ExtractContentWithMetadata(url string) (*JinaResponse, error) {
	startTime := time.Now()
	
	content, err := j.ExtractContent(url)
	if err != nil {
		return nil, err
	}
	
	return &JinaResponse{
		Content:      content,
		URL:          url,
		StatusCode:   200,
		Length:       len(content),
		ProcessingMS: time.Since(startTime).Milliseconds(),
	}, nil
}

// ValidateURL performs basic URL validation before sending to Jina
func (j *JinaClient) ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	
	if len(url) > 2048 {
		return fmt.Errorf("URL too long: %d characters", len(url))
	}
	
	// Basic URL format check
	if !(len(url) > 7 && (url[:7] == "http://" || url[:8] == "https://")) {
		return fmt.Errorf("URL must start with http:// or https://")
	}
	
	return nil
}

// IsJinaAvailable checks if Jina AI Reader service is available
func (j *JinaClient) IsJinaAvailable() bool {
	// Test with a simple, fast URL
	testURL := "https://httpbin.org/get"
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/%s", j.baseURL, testURL), nil)
	if err != nil {
		return false
	}
	
	resp, err := j.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == 200
}

// SetUserAgent allows customizing the user agent string
func (j *JinaClient) SetUserAgent(userAgent string) {
	j.userAgent = userAgent
}

// GetStats returns basic usage statistics (for monitoring)
type JinaStats struct {
	TotalRequests    int           `json:"total_requests"`
	SuccessfulReqs   int           `json:"successful_requests"`
	FailedReqs       int           `json:"failed_requests"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	TotalContentSize int64         `json:"total_content_size"`
}

// Simple in-memory stats tracking (for basic monitoring)
type statsTracker struct {
	requests     int
	successful   int
	failed       int
	totalTime    time.Duration
	totalContent int64
}

// Global stats tracker (simple implementation for MVP)
var globalStats = &statsTracker{}

// GetStats returns current statistics
func (j *JinaClient) GetStats() JinaStats {
	avgTime := time.Duration(0)
	if globalStats.requests > 0 {
		avgTime = globalStats.totalTime / time.Duration(globalStats.requests)
	}
	
	return JinaStats{
		TotalRequests:    globalStats.requests,
		SuccessfulReqs:   globalStats.successful,
		FailedReqs:       globalStats.failed,
		AvgResponseTime:  avgTime,
		TotalContentSize: globalStats.totalContent,
	}
}

// trackRequest updates statistics (called internally)
func (j *JinaClient) trackRequest(success bool, duration time.Duration, contentSize int) {
	globalStats.requests++
	globalStats.totalTime += duration
	globalStats.totalContent += int64(contentSize)
	
	if success {
		globalStats.successful++
	} else {
		globalStats.failed++
	}
}