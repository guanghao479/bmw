package services

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// JinaClient handles content extraction from web pages using Jina AI Reader
type JinaClient struct {
	httpClient  *http.Client
	baseURL     string
	userAgents  []string
	retryConfig RetryConfig
}

// RetryConfig defines retry behavior for failed requests
type RetryConfig struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
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

// NewJinaClient creates a new Jina AI client with enhanced anti-scraping support
func NewJinaClient() *JinaClient {
	// Create HTTP client with enhanced TLS configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS13,
		},
		DisableKeepAlives: false,
		IdleConnTimeout:   90 * time.Second,
	}

	return &JinaClient{
		httpClient: &http.Client{
			Timeout:   45 * time.Second, // Increased timeout for problematic sites
			Transport: transport,
		},
		baseURL: "https://r.jina.ai",
		userAgents: []string{
			// Realistic browser user agents for better site compatibility
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edge/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
		retryConfig: RetryConfig{
			MaxRetries:    3,
			InitialDelay:  1 * time.Second,
			MaxDelay:      10 * time.Second,
			BackoffFactor: 2.0,
		},
	}
}

// NewJinaClientWithTimeout creates a new Jina client with custom timeout
func NewJinaClientWithTimeout(timeout time.Duration) *JinaClient {
	client := NewJinaClient()
	client.httpClient.Timeout = timeout
	return client
}

// ExtractContent extracts clean content from a webpage URL using Jina AI Reader
func (j *JinaClient) ExtractContent(url string) (string, error) {
	startTime := time.Now()
	
	// Validate URL
	if url == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}
	
	var lastErr error
	
	// Retry loop with exponential backoff
	for attempt := 0; attempt <= j.retryConfig.MaxRetries; attempt++ {
		content, err := j.attemptExtraction(url, attempt)
		if err == nil {
			// Log processing time (for debugging)
			processingTime := time.Since(startTime)
			if processingTime > 10*time.Second {
				fmt.Printf("Warning: Jina processing took %v for URL: %s (attempt %d)\n", processingTime, url, attempt+1)
			}
			return content, nil
		}
		
		lastErr = err
		
		// Don't retry if it's a client error (4xx) or URL validation error
		if strings.Contains(err.Error(), "status 4") || strings.Contains(err.Error(), "cannot be empty") {
			break
		}
		
		// Calculate delay for next attempt
		if attempt < j.retryConfig.MaxRetries {
			delay := j.calculateDelay(attempt)
			fmt.Printf("Attempt %d failed for %s, retrying in %v: %v\n", attempt+1, url, delay, err)
			time.Sleep(delay)
		}
	}
	
	return "", fmt.Errorf("failed after %d attempts: %w", j.retryConfig.MaxRetries+1, lastErr)
}

// attemptExtraction performs a single extraction attempt
func (j *JinaClient) attemptExtraction(url string, attempt int) (string, error) {
	// Construct Jina Reader URL
	jinaURL := fmt.Sprintf("%s/%s", j.baseURL, url)
	
	// Create request
	req, err := http.NewRequest("GET", jinaURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set enhanced headers to mimic real browser behavior
	j.setEnhancedHeaders(req, url, attempt)
	
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
	
	// Handle gzip encoding if present
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}
	
	// Read response body
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read jina response: %w", err)
	}
	
	contentStr := string(content)
	
	// Basic validation
	if len(contentStr) < 100 {
		return "", fmt.Errorf("content too short (%d chars), might be an error page", len(contentStr))
	}
	
	return contentStr, nil
}

// setEnhancedHeaders sets realistic browser headers to avoid anti-scraping measures
func (j *JinaClient) setEnhancedHeaders(req *http.Request, url string, attempt int) {
	// Rotate user agent on retries
	userAgent := j.userAgents[attempt%len(j.userAgents)]
	req.Header.Set("User-Agent", userAgent)
	
	// Set realistic browser headers - disable compression for debugging
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "identity")  // Disable compression to debug encoding issues
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	
	// Set referer header for some sources that require it
	if strings.Contains(url, "seattleschild.com") || strings.Contains(url, "parentmap.com") {
		req.Header.Set("Referer", "https://www.google.com/")
	}
	
	// Cache control for retry attempts
	if attempt > 0 {
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Pragma", "no-cache")
	}
}

// calculateDelay calculates exponential backoff delay
func (j *JinaClient) calculateDelay(attempt int) time.Duration {
	delay := float64(j.retryConfig.InitialDelay) * 
		(j.retryConfig.BackoffFactor * float64(attempt)) + 
		(rand.Float64() * 0.1 * float64(j.retryConfig.InitialDelay)) // Add jitter
	
	if delay > float64(j.retryConfig.MaxDelay) {
		delay = float64(j.retryConfig.MaxDelay)
	}
	
	return time.Duration(delay)
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

// SetUserAgents allows customizing the user agent strings for rotation
func (j *JinaClient) SetUserAgents(userAgents []string) {
	if len(userAgents) > 0 {
		j.userAgents = userAgents
	}
}

// AddUserAgent adds a new user agent to the rotation list
func (j *JinaClient) AddUserAgent(userAgent string) {
	j.userAgents = append(j.userAgents, userAgent)
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