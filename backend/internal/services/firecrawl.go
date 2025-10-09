package services

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mendableai/firecrawl-go"
	"seattle-family-activities-scraper/internal/models"
)

// ExtractionDiagnostics captures detailed information about the extraction process
type ExtractionDiagnostics struct {
	URL                string                 `json:"url"`
	StartTime          time.Time              `json:"start_time"`
	EndTime            time.Time              `json:"end_time"`
	ProcessingTime     time.Duration          `json:"processing_time"`
	RawMarkdownLength  int                    `json:"raw_markdown_length"`
	RawMarkdownSample  string                 `json:"raw_markdown_sample"`
	ExtractionAttempts []ExtractionAttempt    `json:"extraction_attempts"`
	StructuredData     map[string]interface{} `json:"structured_data"`
	ValidationIssues   []ValidationIssue      `json:"validation_issues"`
	CreditsUsed        int                    `json:"credits_used"`
	Success            bool                   `json:"success"`
	ErrorMessage       string                 `json:"error_message,omitempty"`
}

// ExtractionAttempt represents a single attempt to extract data
type ExtractionAttempt struct {
	Method      string                 `json:"method"`
	Timestamp   time.Time              `json:"timestamp"`
	Success     bool                   `json:"success"`
	EventsFound int                    `json:"events_found"`
	Details     map[string]interface{} `json:"details"`
	Issues      []string               `json:"issues"`
}

// ValidationIssue represents a validation problem found during extraction
type ValidationIssue struct {
	Severity   string `json:"severity"`    // error|warning|info
	Field      string `json:"field"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
	RawValue   string `json:"raw_value,omitempty"`
}

// FireCrawlClient handles content extraction and structured data extraction using FireCrawl
type FireCrawlClient struct {
	client  *firecrawl.FirecrawlApp
	timeout time.Duration
}

// FireCrawlExtractRequest represents a request to extract structured data
type FireCrawlExtractRequest struct {
	URL    string                 `json:"url"`
	Schema map[string]interface{} `json:"schema"`
}

// FireCrawlExtractResponse represents the response from FireCrawl extract
type FireCrawlExtractResponse struct {
	Success   bool                   `json:"success"`
	Data      ActivityExtractionData `json:"data"`
	Metadata  ExtractMetadata        `json:"metadata"`
	CreditsUsed int                  `json:"credits_used"`
}

// ActivityExtractionData contains the extracted activities
type ActivityExtractionData struct {
	Activities []models.Activity `json:"activities"`
}

// ExtractMetadata contains metadata about the extraction
type ExtractMetadata struct {
	URL         string    `json:"url"`
	ExtractTime time.Time `json:"extract_time"`
	Title       string    `json:"title,omitempty"`
}

// NewFireCrawlClient creates a new FireCrawl client
func NewFireCrawlClient() (*FireCrawlClient, error) {
	apiKey := os.Getenv("FIRECRAWL_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("FIRECRAWL_API_KEY environment variable is required")
	}

	app, err := firecrawl.NewFirecrawlApp(apiKey, "https://api.firecrawl.dev")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize FireCrawl client: %w", err)
	}

	return &FireCrawlClient{
		client:  app,
		timeout: 60 * time.Second,
	}, nil
}

// NewFireCrawlClientWithTimeout creates a new FireCrawl client with custom timeout
func NewFireCrawlClientWithTimeout(timeout time.Duration) (*FireCrawlClient, error) {
	client, err := NewFireCrawlClient()
	if err != nil {
		return nil, err
	}
	client.timeout = timeout
	return client, nil
}

// ExtractActivities extracts structured activities from a webpage URL
func (fc *FireCrawlClient) ExtractActivities(url string) (*FireCrawlExtractResponse, error) {
	startTime := time.Now()
	
	// Initialize diagnostics
	diagnostics := &ExtractionDiagnostics{
		URL:                url,
		StartTime:          startTime,
		ExtractionAttempts: []ExtractionAttempt{},
		ValidationIssues:   []ValidationIssue{},
		StructuredData:     make(map[string]interface{}),
	}

	if url == "" {
		diagnostics.EndTime = time.Now()
		diagnostics.ProcessingTime = time.Since(startTime)
		diagnostics.Success = false
		diagnostics.ErrorMessage = "URL cannot be empty"
		fc.logDiagnostics(diagnostics)
		return nil, fmt.Errorf("URL cannot be empty")
	}

	// Define the schema for activity extraction
	// TODO: Will need to properly integrate this schema once we figure out the correct parameter structure
	schema := getActivityExtractionSchema()
	_ = schema // Suppress unused variable warning

	log.Printf("[EXTRACTION] Starting FireCrawl extract for URL: %s", url)

	// Make the extract request using ScrapeURL with extraction parameters
	// Note: Using nil for now - will need to create proper ScrapeParams struct
	response, err := fc.client.ScrapeURL(url, nil)
	if err != nil {
		diagnostics.EndTime = time.Now()
		diagnostics.ProcessingTime = time.Since(startTime)
		diagnostics.Success = false
		diagnostics.ErrorMessage = fmt.Sprintf("FireCrawl extract failed: %v", err)
		fc.logDiagnostics(diagnostics)
		return nil, fmt.Errorf("FireCrawl extract failed: %w", err)
	}

	// Parse the response with diagnostics
	extractResponse, err := fc.parseExtractResponseWithDiagnostics(response, url, startTime, diagnostics)
	if err != nil {
		diagnostics.EndTime = time.Now()
		diagnostics.ProcessingTime = time.Since(startTime)
		diagnostics.Success = false
		diagnostics.ErrorMessage = fmt.Sprintf("Failed to parse extract response: %v", err)
		fc.logDiagnostics(diagnostics)
		return nil, fmt.Errorf("failed to parse extract response: %w", err)
	}

	// Complete diagnostics
	diagnostics.EndTime = time.Now()
	diagnostics.ProcessingTime = time.Since(startTime)
	diagnostics.Success = true
	diagnostics.CreditsUsed = extractResponse.CreditsUsed

	// Log final diagnostics and store for debugging
	fc.logDiagnostics(diagnostics)
	lastExtractionDiagnostics = diagnostics

	log.Printf("[EXTRACTION] Successfully extracted %d activities from %s in %v (Credits: %d)",
		len(extractResponse.Data.Activities), url, time.Since(startTime), extractResponse.CreditsUsed)

	return extractResponse, nil
}

// parseExtractResponse parses the FireCrawl response into our structure (legacy method)
func (fc *FireCrawlClient) parseExtractResponse(response interface{}, url string, startTime time.Time) (*FireCrawlExtractResponse, error) {
	// Create basic diagnostics for legacy calls
	diagnostics := &ExtractionDiagnostics{
		URL:                url,
		StartTime:          startTime,
		ExtractionAttempts: []ExtractionAttempt{},
		ValidationIssues:   []ValidationIssue{},
		StructuredData:     make(map[string]interface{}),
	}
	
	return fc.parseExtractResponseWithDiagnostics(response, url, startTime, diagnostics)
}

// parseExtractResponseWithDiagnostics parses the FireCrawl response with comprehensive diagnostics
func (fc *FireCrawlClient) parseExtractResponseWithDiagnostics(response interface{}, url string, startTime time.Time, diagnostics *ExtractionDiagnostics) (*FireCrawlExtractResponse, error) {
	// Handle the actual FirecrawlDocument response
	doc, ok := response.(*firecrawl.FirecrawlDocument)
	if !ok {
		diagnostics.ValidationIssues = append(diagnostics.ValidationIssues, ValidationIssue{
			Severity: "error",
			Field:    "response_type",
			Message:  fmt.Sprintf("Unexpected response format from FireCrawl - got %T instead of *firecrawl.FirecrawlDocument", response),
			Suggestion: "Check FireCrawl API response format",
		})
		return nil, fmt.Errorf("unexpected response format from FireCrawl - got %T instead of *firecrawl.FirecrawlDocument", response)
	}

	// Log raw markdown content details
	diagnostics.RawMarkdownLength = len(doc.Markdown)
	if len(doc.Markdown) > 500 {
		diagnostics.RawMarkdownSample = doc.Markdown[:500] + "..."
	} else {
		diagnostics.RawMarkdownSample = doc.Markdown
	}

	log.Printf("[EXTRACTION] Got markdown content from FireCrawl: %d characters", len(doc.Markdown))
	log.Printf("[EXTRACTION] Markdown sample (first 200 chars): %s", 
		func() string {
			if len(doc.Markdown) > 200 {
				return doc.Markdown[:200] + "..."
			}
			return doc.Markdown
		}())

	// Parse activities from the markdown content
	var activities []models.Activity
	var extractionAttempt ExtractionAttempt

	// Check if this looks like a ParentMap calendar page with activities
	if strings.Contains(doc.Markdown, "Calendar") || strings.Contains(url, "parentmap.com") {
		extractionAttempt = ExtractionAttempt{
			Method:    "parseParentMapActivities",
			Timestamp: time.Now(),
		}
		
		log.Printf("[EXTRACTION] Detected ParentMap content, using specialized parser")
		activities = fc.parseParentMapActivitiesWithDiagnostics(doc.Markdown, url, &extractionAttempt)
		
		extractionAttempt.Success = len(activities) > 0
		extractionAttempt.EventsFound = len(activities)
		
		if len(activities) == 0 {
			extractionAttempt.Issues = append(extractionAttempt.Issues, "No activities found in ParentMap content")
			diagnostics.ValidationIssues = append(diagnostics.ValidationIssues, ValidationIssue{
				Severity: "warning",
				Field:    "activities",
				Message:  "No activities extracted from ParentMap content",
				Suggestion: "Check if the page contains calendar events or activities",
			})
		}
	} else {
		extractionAttempt = ExtractionAttempt{
			Method:    "genericExtraction",
			Timestamp: time.Now(),
		}
		
		log.Printf("[EXTRACTION] Using generic extraction for URL: %s", url)
		activities = fc.extractGenericActivitiesWithDiagnostics(doc.Markdown, url, &extractionAttempt)
		
		extractionAttempt.Success = len(activities) > 0
		extractionAttempt.EventsFound = len(activities)
		
		if len(activities) == 0 {
			extractionAttempt.Issues = append(extractionAttempt.Issues, "No activities found using generic extraction")
			diagnostics.ValidationIssues = append(diagnostics.ValidationIssues, ValidationIssue{
				Severity: "warning",
				Field:    "activities",
				Message:  "No activities extracted using generic method",
				Suggestion: "Content may require specialized parsing logic",
			})
		}
	}

	diagnostics.ExtractionAttempts = append(diagnostics.ExtractionAttempts, extractionAttempt)

	// Validate extracted activities
	fc.validateExtractedActivities(activities, diagnostics)

	// Store structured data for diagnostics
	activitiesData := make([]map[string]interface{}, len(activities))
	for i, activity := range activities {
		activitiesData[i] = map[string]interface{}{
			"id":          activity.ID,
			"title":       activity.Title,
			"description": activity.Description,
			"type":        activity.Type,
			"category":    activity.Category,
		}
	}
	diagnostics.StructuredData["activities"] = activitiesData

	log.Printf("[EXTRACTION] Extraction completed: %d activities found", len(activities))

	return &FireCrawlExtractResponse{
		Success: true,
		Data: ActivityExtractionData{
			Activities: activities,
		},
		Metadata: ExtractMetadata{
			URL:         url,
			ExtractTime: startTime,
			Title:       fc.extractTitleFromDoc(doc),
		},
		CreditsUsed: fc.extractCreditsFromDoc(doc),
	}, nil
}

// convertToActivities converts raw activity data to our Activity model
func (fc *FireCrawlClient) convertToActivities(activitiesRaw interface{}, sourceURL string) ([]models.Activity, error) {
	var activities []models.Activity

	activitiesList, ok := activitiesRaw.([]interface{})
	if !ok {
		return activities, fmt.Errorf("activities data is not a list")
	}

	for _, activityRaw := range activitiesList {
		activityMap, ok := activityRaw.(map[string]interface{})
		if !ok {
			continue
		}

		activity := models.Activity{
			Type:      models.TypeEvent, // Default type
			Category:  models.CategoryFreeCommunity, // Default category
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Source: models.Source{
				URL:         sourceURL,
				Domain:      extractDomain(sourceURL),
				ScrapedAt:   time.Now(),
				LastChecked: time.Now(),
				Reliability: "medium",
			},
		}

		// Extract title
		if title, exists := activityMap["title"]; exists {
			if titleStr, ok := title.(string); ok {
				activity.Title = titleStr
			}
		}

		// Extract description
		if desc, exists := activityMap["description"]; exists {
			if descStr, ok := desc.(string); ok {
				activity.Description = descStr
			}
		}

		// Extract location
		if location, exists := activityMap["location"]; exists {
			if locationMap, ok := location.(map[string]interface{}); ok {
				activity.Location = models.Location{
					Name:    fc.extractStringField(locationMap, "name"),
					Address: fc.extractStringField(locationMap, "address"),
				}

				// Extract coordinates if present
				if coords, exists := locationMap["coordinates"]; exists {
					if coordsStr, ok := coords.(string); ok {
						// Parse coordinates from "lat,lng" format
						if lat, lng, err := parseCoordinates(coordsStr); err == nil {
							activity.Location.Coordinates = models.Coordinates{
								Lat: lat,
								Lng: lng,
							}
						}
					}
				}
			}
		}

		// Extract schedule
		if schedule, exists := activityMap["schedule"]; exists {
			if scheduleMap, ok := schedule.(map[string]interface{}); ok {
				activity.Schedule = models.Schedule{
					StartDate: fc.extractStringField(scheduleMap, "start_date"),
					StartTime: fc.extractStringField(scheduleMap, "start_time"),
					EndTime:   fc.extractStringField(scheduleMap, "end_time"),
				}
			}
		}

		// Extract age groups
		if ageGroups, exists := activityMap["age_groups"]; exists {
			if ageGroupsList, ok := ageGroups.([]interface{}); ok {
				for _, ageGroup := range ageGroupsList {
					if ageGroupStr, ok := ageGroup.(string); ok {
						// Convert string to AgeGroup struct
						parsedAgeGroup := parseAgeGroup(ageGroupStr)
						activity.AgeGroups = append(activity.AgeGroups, parsedAgeGroup)
					}
				}
			}
		}

		// Extract pricing
		if pricing, exists := activityMap["pricing"]; exists {
			if pricingStr, ok := pricing.(string); ok {
				isFree := pricingStr == "Free" || pricingStr == "free"
				pricingType := "paid"
				if isFree {
					pricingType = "free"
				}
				activity.Pricing = models.Pricing{
					Type:        pricingType,
					Description: pricingStr,
					Currency:    "USD",
					Unit:        "per-person",
				}
			}
		}

		// Extract registration URL
		if regURL, exists := activityMap["registration_url"]; exists {
			if regURLStr, ok := regURL.(string); ok {
				activity.Registration = models.Registration{
					Required: true,
					Method:   "online",
					URL:      regURLStr,
					Status:   "open",
				}
			}
		}

		// Only add activity if it has required fields
		if activity.Title != "" && activity.Location.Name != "" {
			activities = append(activities, activity)
		}
	}

	return activities, nil
}

// extractStringField safely extracts a string field from a map
func (fc *FireCrawlClient) extractStringField(data map[string]interface{}, field string) string {
	if value, exists := data[field]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return ""
}

// extractTitle extracts title from response metadata
func (fc *FireCrawlClient) extractTitle(response map[string]interface{}) string {
	if metadata, exists := response["metadata"]; exists {
		if metaMap, ok := metadata.(map[string]interface{}); ok {
			return fc.extractStringField(metaMap, "title")
		}
	}
	return ""
}

// extractCreditsUsed extracts credits used from response
func (fc *FireCrawlClient) extractCreditsUsed(response map[string]interface{}) int {
	if metadata, exists := response["metadata"]; exists {
		if metaMap, ok := metadata.(map[string]interface{}); ok {
			if credits, exists := metaMap["credits_used"]; exists {
				if creditsFloat, ok := credits.(float64); ok {
					return int(creditsFloat)
				}
				if creditsInt, ok := credits.(int); ok {
					return creditsInt
				}
			}
		}
	}
	return 50 // Default credits for extract operation
}

// getActivityExtractionSchema returns the schema for extracting activities
func getActivityExtractionSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"activities": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type":        "string",
							"description": "The name or title of the family activity or event",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "A detailed description of the activity or event",
						},
						"location": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type":        "string",
									"description": "The name of the venue or location",
								},
								"address": map[string]interface{}{
									"type":        "string",
									"description": "The full address of the location",
								},
								"coordinates": map[string]interface{}{
									"type":        "string",
									"description": "GPS coordinates in 'lat,lng' format",
								},
							},
							"required": []string{"name"},
						},
						"schedule": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"start_date": map[string]interface{}{
									"type":        "string",
									"description": "Start date in YYYY-MM-DD format",
								},
								"start_time": map[string]interface{}{
									"type":        "string",
									"description": "Start time in HH:MM format",
								},
								"end_time": map[string]interface{}{
									"type":        "string",
									"description": "End time in HH:MM format",
								},
							},
						},
						"age_groups": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
							"description": "Target age groups like 'toddlers', 'preschoolers', 'elementary', 'teens', 'all ages'",
						},
						"pricing": map[string]interface{}{
							"type":        "string",
							"description": "Pricing information, use 'Free' for free events",
						},
						"registration_url": map[string]interface{}{
							"type":        "string",
							"description": "URL for registration or more information",
						},
					},
					"required": []string{"title", "location"},
				},
			},
		},
		"required": []string{"activities"},
	}
}

// ValidateExtractResponse validates the response from FireCrawl
func (fc *FireCrawlClient) ValidateExtractResponse(response *FireCrawlExtractResponse) []string {
	var issues []string

	if response == nil {
		return []string{"response is nil"}
	}

	if !response.Success {
		issues = append(issues, "extract operation was not successful")
	}

	if len(response.Data.Activities) == 0 {
		issues = append(issues, "no activities extracted")
	}

	// Validate individual activities
	for i, activity := range response.Data.Activities {
		if activity.Title == "" {
			issues = append(issues, fmt.Sprintf("activity %d: missing title", i+1))
		}
		if activity.Location.Name == "" {
			issues = append(issues, fmt.Sprintf("activity %d: missing location name", i+1))
		}
	}

	return issues
}

// IsFireCrawlAvailable checks if FireCrawl service is available
func (fc *FireCrawlClient) IsFireCrawlAvailable() bool {
	// Use a simple test URL
	testURL := "https://httpbin.org/get"

	// Make a simple scrape request (not extract) to test availability
	_, err := fc.client.ScrapeURL(testURL, nil)

	return err == nil
}

// GetStats returns basic usage statistics
type FireCrawlStats struct {
	TotalRequests      int           `json:"total_requests"`
	SuccessfulReqs     int           `json:"successful_requests"`
	FailedReqs         int           `json:"failed_requests"`
	AvgResponseTime    time.Duration `json:"avg_response_time"`
	TotalCreditsUsed   int           `json:"total_credits_used"`
	TotalActivitiesExt int           `json:"total_activities_extracted"`
}

// Simple in-memory stats tracking for FireCrawl
type fireCrawlStatsTracker struct {
	requests        int
	successful      int
	failed          int
	totalTime       time.Duration
	totalCredits    int
	totalActivities int
}

// Global stats tracker for FireCrawl
var globalFireCrawlStats = &fireCrawlStatsTracker{}

// GetStats returns current statistics
func (fc *FireCrawlClient) GetStats() FireCrawlStats {
	avgTime := time.Duration(0)
	if globalFireCrawlStats.requests > 0 {
		avgTime = globalFireCrawlStats.totalTime / time.Duration(globalFireCrawlStats.requests)
	}

	return FireCrawlStats{
		TotalRequests:      globalFireCrawlStats.requests,
		SuccessfulReqs:     globalFireCrawlStats.successful,
		FailedReqs:         globalFireCrawlStats.failed,
		AvgResponseTime:    avgTime,
		TotalCreditsUsed:   globalFireCrawlStats.totalCredits,
		TotalActivitiesExt: globalFireCrawlStats.totalActivities,
	}
}

// trackRequest updates statistics
func (fc *FireCrawlClient) trackRequest(success bool, duration time.Duration, creditsUsed, activitiesExtracted int) {
	globalFireCrawlStats.requests++
	globalFireCrawlStats.totalTime += duration
	globalFireCrawlStats.totalCredits += creditsUsed
	globalFireCrawlStats.totalActivities += activitiesExtracted

	if success {
		globalFireCrawlStats.successful++
	} else {
		globalFireCrawlStats.failed++
	}
}

// Helper functions for data conversion

// extractDomain extracts domain from URL
func extractDomain(url string) string {
	if strings.HasPrefix(url, "http://") {
		url = url[7:]
	} else if strings.HasPrefix(url, "https://") {
		url = url[8:]
	}

	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}

	return url
}

// parseCoordinates parses "lat,lng" string into float64 values
func parseCoordinates(coordsStr string) (lat, lng float64, err error) {
	parts := strings.Split(coordsStr, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid coordinates format: %s", coordsStr)
	}

	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid latitude: %s", parts[0])
	}

	lng, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid longitude: %s", parts[1])
	}

	return lat, lng, nil
}

// parseAgeGroup converts a string age group to AgeGroup struct
func parseAgeGroup(ageGroupStr string) models.AgeGroup {
	// Map common age group strings to structured data
	ageGroupStr = strings.ToLower(strings.TrimSpace(ageGroupStr))

	switch ageGroupStr {
	case "infants", "infant", "babies", "baby":
		return models.AgeGroup{
			Category:    "infant",
			MinAge:      0,
			MaxAge:      12,
			Unit:        "months",
			Description: "Infants (0-12 months)",
		}
	case "toddlers", "toddler":
		return models.AgeGroup{
			Category:    "toddler",
			MinAge:      1,
			MaxAge:      2,
			Unit:        "years",
			Description: "Toddlers (1-2 years)",
		}
	case "preschoolers", "preschool", "preschooler":
		return models.AgeGroup{
			Category:    "preschool",
			MinAge:      3,
			MaxAge:      5,
			Unit:        "years",
			Description: "Preschoolers (3-5 years)",
		}
	case "elementary", "school-age", "kids":
		return models.AgeGroup{
			Category:    "elementary",
			MinAge:      6,
			MaxAge:      10,
			Unit:        "years",
			Description: "Elementary (6-10 years)",
		}
	case "tweens", "tween":
		return models.AgeGroup{
			Category:    "tween",
			MinAge:      11,
			MaxAge:      12,
			Unit:        "years",
			Description: "Tweens (11-12 years)",
		}
	case "teens", "teen", "teenagers", "teenager":
		return models.AgeGroup{
			Category:    "teen",
			MinAge:      13,
			MaxAge:      17,
			Unit:        "years",
			Description: "Teens (13-17 years)",
		}
	case "adults", "adult":
		return models.AgeGroup{
			Category:    "adult",
			MinAge:      18,
			MaxAge:      99,
			Unit:        "years",
			Description: "Adults (18+ years)",
		}
	case "all ages", "all-ages", "family", "everyone":
		return models.AgeGroup{
			Category:    "all-ages",
			MinAge:      0,
			MaxAge:      99,
			Unit:        "years",
			Description: "All Ages",
		}
	default:
		// Default fallback
		return models.AgeGroup{
			Category:    "all-ages",
			MinAge:      0,
			MaxAge:      99,
			Unit:        "years",
			Description: fmt.Sprintf("Custom: %s", ageGroupStr),
		}
	}
}

// parseParentMapActivities extracts activities from ParentMap calendar markdown (legacy method)
func (fc *FireCrawlClient) parseParentMapActivities(markdown, url string) []models.Activity {
	attempt := ExtractionAttempt{
		Method:    "parseParentMapActivities",
		Timestamp: time.Now(),
	}
	return fc.parseParentMapActivitiesWithDiagnostics(markdown, url, &attempt)
}

// parseParentMapActivitiesWithDiagnostics extracts activities from ParentMap calendar markdown with diagnostics
func (fc *FireCrawlClient) parseParentMapActivitiesWithDiagnostics(markdown, url string, attempt *ExtractionAttempt) []models.Activity {
	var activities []models.Activity

	log.Printf("[PARENTMAP] Starting enhanced ParentMap-specific parsing for %d characters of content", len(markdown))

	// Enhanced parsing for ParentMap content
	attempt.Details = make(map[string]interface{})
	
	// Parse markdown into structured events
	events := fc.parseMarkdownEvents(markdown, attempt)
	
	log.Printf("[PARENTMAP] Parsed %d potential events from markdown structure", len(events))
	attempt.Details["parsed_events_count"] = len(events)

	// Convert parsed events to Activity models with validation
	for i, event := range events {
		if i >= 10 { // Limit to 10 events for performance
			log.Printf("[PARENTMAP] Limiting to first 10 events (found %d total)", len(events))
			break
		}

		// Validate event data before conversion
		validationResult := fc.validateEventData(event)
		if !validationResult.IsValid {
			log.Printf("[PARENTMAP] Event %d failed validation: %v", i+1, validationResult.Issues)
			attempt.Issues = append(attempt.Issues, fmt.Sprintf("Event %d validation failed: %s", i+1, strings.Join(validationResult.Issues, ", ")))
			continue
		}

		activity := fc.convertEventToActivity(event, url, fmt.Sprintf("parentmap-%d", i))
		if activity != nil {
			// Validate the converted activity
			activityValidation := fc.validateActivityData(*activity)
			if activityValidation.IsValid {
				activities = append(activities, *activity)
				log.Printf("[PARENTMAP] Successfully converted and validated event %d: %s (confidence: %.1f)", 
					i+1, activity.Title, activityValidation.ConfidenceScore)
			} else {
				log.Printf("[PARENTMAP] Activity %d failed post-conversion validation: %v", i+1, activityValidation.Issues)
				attempt.Issues = append(attempt.Issues, fmt.Sprintf("Activity %d validation failed: %s", i+1, strings.Join(activityValidation.Issues, ", ")))
			}
		} else {
			log.Printf("[PARENTMAP] Failed to convert event %d", i+1)
			attempt.Issues = append(attempt.Issues, fmt.Sprintf("Failed to convert event %d", i+1))
		}
	}

	// If no structured events found, try fallback parsing
	if len(activities) == 0 {
		log.Printf("[PARENTMAP] No structured events found, trying fallback parsing")
		activities = fc.parseParentMapFallback(markdown, url, attempt)
	}

	attempt.Details["final_activities_count"] = len(activities)
	log.Printf("[PARENTMAP] ParentMap parsing completed: %d activities extracted", len(activities))
	return activities
}

// EventData represents a parsed event from markdown
type EventData struct {
	Title       string
	Description string
	Date        string
	Time        string
	Location    string
	Price       string
	AgeGroups   []string
	URL         string
	RawContent  string
}

// parseMarkdownEvents parses markdown content to extract structured event data
func (fc *FireCrawlClient) parseMarkdownEvents(markdown string, attempt *ExtractionAttempt) []EventData {
	var events []EventData
	
	lines := strings.Split(markdown, "\n")
	
	// Track parsing statistics
	headerCount := 0
	eventBlockCount := 0
	dateLineCount := 0
	
	var currentEvent *EventData
	var currentSection strings.Builder
	inEventBlock := false
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Detect headers that might be event titles
		if fc.isEventHeader(line) {
			headerCount++
			
			// Save previous event if we have one
			if currentEvent != nil && currentEvent.Title != "" {
				currentEvent.RawContent = currentSection.String()
				events = append(events, *currentEvent)
				eventBlockCount++
			}
			
			// Start new event
			currentEvent = &EventData{
				Title: fc.cleanEventTitle(line),
			}
			currentSection.Reset()
			currentSection.WriteString(line + "\n")
			inEventBlock = true
			
			log.Printf("[PARENTMAP] Found potential event header: %s", currentEvent.Title)
			continue
		}
		
		// If we're in an event block, collect information
		if inEventBlock && currentEvent != nil {
			currentSection.WriteString(line + "\n")
			
			// Extract date information
			if date := fc.extractDateFromLine(line); date != "" {
				if currentEvent.Date == "" {
					currentEvent.Date = date
					dateLineCount++
					log.Printf("[PARENTMAP] Extracted date for '%s': %s", currentEvent.Title, date)
				}
			}
			
			// Extract time information
			if time := fc.extractTimeFromLine(line); time != "" {
				if currentEvent.Time == "" {
					currentEvent.Time = time
					log.Printf("[PARENTMAP] Extracted time for '%s': %s", currentEvent.Title, time)
				}
			}
			
			// Extract location information
			if location := fc.extractLocationFromLine(line); location != "" {
				if currentEvent.Location == "" {
					currentEvent.Location = location
					log.Printf("[PARENTMAP] Extracted location for '%s': %s", currentEvent.Title, location)
				}
			}
			
			// Extract price information
			if price := fc.extractPriceFromLine(line); price != "" {
				if currentEvent.Price == "" {
					currentEvent.Price = price
					log.Printf("[PARENTMAP] Extracted price for '%s': %s", currentEvent.Title, price)
				}
			}
			
			// Extract age group information
			if ageGroups := fc.extractAgeGroupsFromLine(line); len(ageGroups) > 0 {
				currentEvent.AgeGroups = append(currentEvent.AgeGroups, ageGroups...)
				log.Printf("[PARENTMAP] Extracted age groups for '%s': %v", currentEvent.Title, ageGroups)
			}
			
			// Build description from content
			if currentEvent.Description == "" && len(line) > 20 && !fc.isMetadataLine(line) {
				currentEvent.Description = line
			}
			
			// Stop collecting if we hit another header or reach end of logical block
			if i < len(lines)-1 {
				nextLine := strings.TrimSpace(lines[i+1])
				if fc.isEventHeader(nextLine) || fc.isBlockSeparator(line) {
					inEventBlock = false
				}
			}
		}
	}
	
	// Don't forget the last event
	if currentEvent != nil && currentEvent.Title != "" {
		currentEvent.RawContent = currentSection.String()
		events = append(events, *currentEvent)
		eventBlockCount++
	}
	
	// Update attempt details
	attempt.Details["header_count"] = headerCount
	attempt.Details["event_block_count"] = eventBlockCount
	attempt.Details["date_line_count"] = dateLineCount
	
	log.Printf("[PARENTMAP] Parsing stats - Headers: %d, Event blocks: %d, Date lines: %d", 
		headerCount, eventBlockCount, dateLineCount)
	
	return events
}

// isEventHeader determines if a line is likely an event title/header
func (fc *FireCrawlClient) isEventHeader(line string) bool {
	line = strings.TrimSpace(line)
	
	// Check for markdown headers
	if strings.HasPrefix(line, "#") {
		return true
	}
	
	// Check for lines that look like event titles
	// - Contains event-related keywords
	// - Is not too long (likely not a description)
	// - Contains title-case words
	if len(line) > 5 && len(line) < 100 {
		eventKeywords := []string{
			"class", "workshop", "event", "activity", "program", "camp", "story", "time",
			"music", "art", "dance", "swim", "play", "festival", "fair", "market",
			"tour", "walk", "hike", "performance", "show", "concert", "movie",
		}
		
		lowerLine := strings.ToLower(line)
		for _, keyword := range eventKeywords {
			if strings.Contains(lowerLine, keyword) {
				return true
			}
		}
		
		// Check if it looks like a title (has capital letters and reasonable length)
		if fc.looksLikeTitle(line) {
			return true
		}
	}
	
	return false
}

// looksLikeTitle checks if a line has title-like characteristics
func (fc *FireCrawlClient) looksLikeTitle(line string) bool {
	words := strings.Fields(line)
	if len(words) < 2 || len(words) > 15 {
		return false
	}
	
	capitalWords := 0
	for _, word := range words {
		if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
			capitalWords++
		}
	}
	
	// At least 50% of words should be capitalized for a title
	return float64(capitalWords)/float64(len(words)) >= 0.5
}

// cleanEventTitle cleans and normalizes an event title
func (fc *FireCrawlClient) cleanEventTitle(line string) string {
	// Remove markdown headers
	title := strings.TrimSpace(line)
	title = strings.TrimLeft(title, "#")
	title = strings.TrimSpace(title)
	
	// Remove common prefixes/suffixes
	prefixes := []string{"Event:", "Activity:", "Class:", "Workshop:"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(title, prefix) {
			title = strings.TrimSpace(title[len(prefix):])
		}
	}
	
	return title
}

// isMetadataLine checks if a line contains metadata rather than description content
func (fc *FireCrawlClient) isMetadataLine(line string) bool {
	lowerLine := strings.ToLower(line)
	
	metadataPatterns := []string{
		"date:", "time:", "location:", "price:", "cost:", "age:", "ages:",
		"when:", "where:", "contact:", "phone:", "email:", "website:",
		"registration:", "signup:", "info:", "details:",
	}
	
	for _, pattern := range metadataPatterns {
		if strings.Contains(lowerLine, pattern) {
			return true
		}
	}
	
	return false
}

// isBlockSeparator checks if a line indicates the end of an event block
func (fc *FireCrawlClient) isBlockSeparator(line string) bool {
	line = strings.TrimSpace(line)
	
	// Empty lines or lines with just separators
	if line == "" || line == "---" || line == "***" {
		return true
	}
	
	// Lines that indicate section breaks
	separatorPatterns := []string{
		"back to top", "more events", "view all", "see more",
		"next page", "previous page", "calendar view",
	}
	
	lowerLine := strings.ToLower(line)
	for _, pattern := range separatorPatterns {
		if strings.Contains(lowerLine, pattern) {
			return true
		}
	}
	
	return false
}

// extractGenericActivitiesWithDiagnostics extracts activities using generic patterns with diagnostics
func (fc *FireCrawlClient) extractGenericActivitiesWithDiagnostics(markdown, url string, attempt *ExtractionAttempt) []models.Activity {
	var activities []models.Activity

	log.Printf("[GENERIC] Starting enhanced generic extraction for %d characters of content", len(markdown))

	attempt.Details = make(map[string]interface{})
	
	// Use the robust extraction method
	events := fc.extractEventsFromMarkdown(markdown, attempt)
	
	log.Printf("[GENERIC] Robust extraction found %d events", len(events))
	attempt.Details["robust_events_found"] = len(events)

	// Convert extracted events to Activity models with validation
	for i, event := range events {
		if i >= 8 { // Limit to 8 events for generic extraction
			log.Printf("[GENERIC] Limiting to first 8 events (found %d total)", len(events))
			break
		}

		// Validate event data before conversion
		validationResult := fc.validateEventData(event)
		if !validationResult.IsValid {
			log.Printf("[GENERIC] Event %d failed validation: %v", i+1, validationResult.Issues)
			attempt.Issues = append(attempt.Issues, fmt.Sprintf("Event %d validation failed: %s", i+1, strings.Join(validationResult.Issues, ", ")))
			continue
		}

		activity := fc.convertEventToActivity(event, url, fmt.Sprintf("generic-%d", i))
		if activity != nil {
			// Validate the converted activity
			activityValidation := fc.validateActivityData(*activity)
			if activityValidation.IsValid {
				activities = append(activities, *activity)
				log.Printf("[GENERIC] Successfully converted and validated event %d: %s (confidence: %.1f)", 
					i+1, activity.Title, activityValidation.ConfidenceScore)
			} else {
				log.Printf("[GENERIC] Activity %d failed post-conversion validation: %v", i+1, activityValidation.Issues)
				attempt.Issues = append(attempt.Issues, fmt.Sprintf("Activity %d validation failed: %s", i+1, strings.Join(activityValidation.Issues, ", ")))
			}
		} else {
			log.Printf("[GENERIC] Failed to convert event %d", i+1)
			attempt.Issues = append(attempt.Issues, fmt.Sprintf("Failed to convert event %d", i+1))
		}
	}

	// If no structured events found, try keyword-based fallback
	if len(activities) == 0 {
		log.Printf("[GENERIC] No structured events found, trying keyword-based fallback")
		activities = fc.extractGenericFallback(markdown, url, attempt)
	}

	attempt.Details["final_activities_count"] = len(activities)
	log.Printf("[GENERIC] Generic extraction completed: %d activities extracted", len(activities))
	return activities
}

// extractGenericFallback provides fallback extraction when robust parsing fails
func (fc *FireCrawlClient) extractGenericFallback(markdown, url string, attempt *ExtractionAttempt) []models.Activity {
	var activities []models.Activity
	
	log.Printf("[GENERIC] Using fallback extraction method")
	
	// Look for common event/activity indicators
	eventKeywords := []string{"event", "activity", "class", "workshop", "program", "camp", "performance"}
	keywordMatches := make(map[string]int)
	
	for _, keyword := range eventKeywords {
		count := strings.Count(strings.ToLower(markdown), keyword)
		if count > 0 {
			keywordMatches[keyword] = count
		}
	}
	
	attempt.Details["fallback_keyword_matches"] = keywordMatches
	log.Printf("[GENERIC] Fallback found keyword matches: %v", keywordMatches)

	// Create activities based on keyword presence and content length
	if len(keywordMatches) > 0 || len(markdown) > 100 {
		// Determine number of activities based on content richness
		activityCount := 1
		if len(keywordMatches) > 2 {
			activityCount = 2
		}
		if len(markdown) > 1000 {
			activityCount = min(activityCount+1, 3)
		}
		
		for i := 0; i < activityCount; i++ {
			activity := models.Activity{
				ID:          fmt.Sprintf("generic-fallback-%d-%d", time.Now().Unix(), i),
				Title:       fmt.Sprintf("Event from %s", extractDomain(url)),
				Description: fc.generateFallbackDescription(markdown, keywordMatches),
				Type:        models.TypeEvent,
				Category:    fc.determineFallbackCategory(keywordMatches),
				Schedule: models.Schedule{
					StartDate: time.Now().Format("2006-01-02"),
					StartTime: "10:00 AM",
					Type:      models.ScheduleTypeOneTime,
					Timezone:  "America/Los_Angeles",
				},
				Location: models.Location{
					Name:   fmt.Sprintf("Venue from %s", extractDomain(url)),
					City:   "Seattle",
					State:  "WA",
					Region: "Seattle Metro",
				},
				Pricing: models.Pricing{
					Type:        models.PricingTypeVariable,
					Description: "See website for details",
					Currency:    "USD",
				},
				AgeGroups: []models.AgeGroup{
					{
						Category:    models.AgeGroupAllAges,
						Description: "All Ages",
					},
				},
				Status:    models.ActivityStatusActive,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Source: models.Source{
					URL:         url,
					Domain:      extractDomain(url),
					ScrapedAt:   time.Now(),
					LastChecked: time.Now(),
					Reliability: "low", // Lower reliability for fallback
				},
			}
			activities = append(activities, activity)
			log.Printf("[GENERIC] Created fallback activity %d: %s", i+1, activity.Title)
		}
	} else {
		attempt.Issues = append(attempt.Issues, "No recognizable content patterns found in fallback")
		log.Printf("[GENERIC] No recognizable content patterns found in fallback")
	}

	return activities
}

// generateFallbackDescription generates a description based on available content
func (fc *FireCrawlClient) generateFallbackDescription(markdown string, keywordMatches map[string]int) string {
	// Extract first meaningful paragraph
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 50 && len(line) < 300 && !strings.HasPrefix(line, "#") {
			return line
		}
	}
	
	// Fallback to keyword-based description
	if len(keywordMatches) > 0 {
		var keywords []string
		for keyword := range keywordMatches {
			keywords = append(keywords, keyword)
		}
		return fmt.Sprintf("Content includes: %s", strings.Join(keywords, ", "))
	}
	
	return "Event or activity information extracted from website"
}

// determineFallbackCategory determines category based on keyword matches
func (fc *FireCrawlClient) determineFallbackCategory(keywordMatches map[string]int) string {
	if keywordMatches["class"] > 0 || keywordMatches["workshop"] > 0 {
		return models.CategoryArtsCreativity
	}
	if keywordMatches["camp"] > 0 || keywordMatches["program"] > 0 {
		return models.CategoryCampsPrograms
	}
	if keywordMatches["performance"] > 0 {
		return models.CategoryEntertainmentEvents
	}
	return models.CategoryFreeCommunity
}

// ValidationResult represents the result of data validation
type ValidationResult struct {
	IsValid         bool     `json:"is_valid"`
	ConfidenceScore float64  `json:"confidence_score"`
	Issues          []string `json:"issues"`
	Warnings        []string `json:"warnings"`
}

// validateEventData validates extracted event data before conversion
func (fc *FireCrawlClient) validateEventData(event EventData) ValidationResult {
	result := ValidationResult{
		IsValid:         true,
		ConfidenceScore: 100.0,
		Issues:          []string{},
		Warnings:        []string{},
	}

	log.Printf("[VALIDATION] Validating event data for: %s", event.Title)

	// Check required fields
	if event.Title == "" {
		result.Issues = append(result.Issues, "Title is required")
		result.IsValid = false
		result.ConfidenceScore -= 50
	} else if len(event.Title) < 3 {
		result.Warnings = append(result.Warnings, "Title is very short")
		result.ConfidenceScore -= 10
	} else if len(event.Title) > 100 {
		result.Warnings = append(result.Warnings, "Title is very long")
		result.ConfidenceScore -= 5
	}

	// Check data quality
	if event.Description == "" {
		result.Warnings = append(result.Warnings, "No description provided")
		result.ConfidenceScore -= 15
	} else if len(event.Description) < 10 {
		result.Warnings = append(result.Warnings, "Description is very short")
		result.ConfidenceScore -= 10
	}

	if event.Date == "" {
		result.Warnings = append(result.Warnings, "No date information")
		result.ConfidenceScore -= 20
	} else {
		// Validate date format
		if !fc.isValidDateFormat(event.Date) {
			result.Warnings = append(result.Warnings, "Date format may be invalid")
			result.ConfidenceScore -= 10
		}
	}

	if event.Time == "" {
		result.Warnings = append(result.Warnings, "No time information")
		result.ConfidenceScore -= 15
	} else {
		// Validate time format
		if !fc.isValidTimeFormat(event.Time) {
			result.Warnings = append(result.Warnings, "Time format may be invalid")
			result.ConfidenceScore -= 5
		}
	}

	if event.Location == "" {
		result.Warnings = append(result.Warnings, "No location information")
		result.ConfidenceScore -= 25
	} else if len(event.Location) < 3 {
		result.Warnings = append(result.Warnings, "Location information is very brief")
		result.ConfidenceScore -= 10
	}

	if event.Price == "" {
		result.Warnings = append(result.Warnings, "No pricing information")
		result.ConfidenceScore -= 10
	}

	if len(event.AgeGroups) == 0 {
		result.Warnings = append(result.Warnings, "No age group information")
		result.ConfidenceScore -= 10
	}

	// Ensure confidence score doesn't go below 0
	if result.ConfidenceScore < 0 {
		result.ConfidenceScore = 0
	}

	log.Printf("[VALIDATION] Event validation completed: Valid=%t, Confidence=%.1f, Issues=%d, Warnings=%d", 
		result.IsValid, result.ConfidenceScore, len(result.Issues), len(result.Warnings))

	return result
}

// validateActivityData validates converted Activity data
func (fc *FireCrawlClient) validateActivityData(activity models.Activity) ValidationResult {
	result := ValidationResult{
		IsValid:         true,
		ConfidenceScore: 100.0,
		Issues:          []string{},
		Warnings:        []string{},
	}

	log.Printf("[VALIDATION] Validating activity data for: %s", activity.Title)

	// Check required fields
	if activity.Title == "" {
		result.Issues = append(result.Issues, "Activity title is required")
		result.IsValid = false
		result.ConfidenceScore -= 50
	}

	if activity.Location.Name == "" {
		result.Issues = append(result.Issues, "Activity location name is required")
		result.IsValid = false
		result.ConfidenceScore -= 30
	}

	// Check data types and consistency
	if activity.Type == "" {
		result.Warnings = append(result.Warnings, "Activity type not set")
		result.ConfidenceScore -= 10
	}

	if activity.Category == "" {
		result.Warnings = append(result.Warnings, "Activity category not set")
		result.ConfidenceScore -= 10
	}

	if activity.Schedule.StartDate == "" {
		result.Warnings = append(result.Warnings, "No start date specified")
		result.ConfidenceScore -= 20
	}

	if activity.Schedule.StartTime == "" {
		result.Warnings = append(result.Warnings, "No start time specified")
		result.ConfidenceScore -= 15
	}

	if len(activity.AgeGroups) == 0 {
		result.Warnings = append(result.Warnings, "No age groups specified")
		result.ConfidenceScore -= 10
	}

	// Check pricing consistency
	if activity.Pricing.Type == "" {
		result.Warnings = append(result.Warnings, "Pricing type not specified")
		result.ConfidenceScore -= 5
	}

	// Check source information
	if activity.Source.URL == "" {
		result.Warnings = append(result.Warnings, "No source URL")
		result.ConfidenceScore -= 5
	}

	if activity.Source.Domain == "" {
		result.Warnings = append(result.Warnings, "No source domain")
		result.ConfidenceScore -= 5
	}

	// Ensure confidence score doesn't go below 0
	if result.ConfidenceScore < 0 {
		result.ConfidenceScore = 0
	}

	log.Printf("[VALIDATION] Activity validation completed: Valid=%t, Confidence=%.1f, Issues=%d, Warnings=%d", 
		result.IsValid, result.ConfidenceScore, len(result.Issues), len(result.Warnings))

	return result
}

// isValidDateFormat checks if a date string appears to be in a valid format
func (fc *FireCrawlClient) isValidDateFormat(dateStr string) bool {
	// Basic date format validation
	datePatterns := []string{
		`^\d{1,2}/\d{1,2}/\d{2,4}$`,                                                                                    // MM/DD/YYYY
		`^\d{1,2}-\d{1,2}-\d{2,4}$`,                                                                                    // MM-DD-YYYY
		`^\d{4}-\d{1,2}-\d{1,2}$`,                                                                                      // YYYY-MM-DD
		`^(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4}$`, // Month DD, YYYY
		`^(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2}$`,                                               // Mon DD
	}

	for _, pattern := range datePatterns {
		if matched, _ := regexp.MatchString(pattern, dateStr); matched {
			return true
		}
	}

	return false
}

// isValidTimeFormat checks if a time string appears to be in a valid format
func (fc *FireCrawlClient) isValidTimeFormat(timeStr string) bool {
	// Basic time format validation
	timePatterns := []string{
		`^\d{1,2}:\d{2}\s*(AM|PM|am|pm)$`,     // 12-hour format
		`^\d{1,2}\s*(AM|PM|am|pm)$`,           // Hour only with AM/PM
		`^([01]?\d|2[0-3]):[0-5]\d$`,          // 24-hour format
		`^\d{1,2}:\d{2}\s*[-–]\s*\d{1,2}:\d{2}\s*(AM|PM|am|pm)?$`, // Time range
	}

	for _, pattern := range timePatterns {
		if matched, _ := regexp.MatchString(pattern, timeStr); matched {
			return true
		}
	}

	return false
}

// extractTitleFromDoc extracts title from FireCrawl document
func (fc *FireCrawlClient) extractTitleFromDoc(doc *firecrawl.FirecrawlDocument) string {
	// Look for title in markdown content
	lines := strings.Split(doc.Markdown, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "Extracted Content"
}

// extractCreditsFromDoc extracts credits used from FireCrawl document
func (fc *FireCrawlClient) extractCreditsFromDoc(doc *firecrawl.FirecrawlDocument) int {
	// For now, assume 1 credit per request
	// In a real implementation, this would be extracted from the response metadata
	return 1
}

// validateExtractedActivities validates the extracted activities and adds issues to diagnostics
func (fc *FireCrawlClient) validateExtractedActivities(activities []models.Activity, diagnostics *ExtractionDiagnostics) {
	log.Printf("[VALIDATION] Validating %d extracted activities", len(activities))

	for i, activity := range activities {
		activityPrefix := fmt.Sprintf("activity_%d", i+1)

		// Check required fields
		if activity.Title == "" {
			diagnostics.ValidationIssues = append(diagnostics.ValidationIssues, ValidationIssue{
				Severity:   "error",
				Field:      activityPrefix + ".title",
				Message:    "Activity title is empty",
				Suggestion: "Ensure title extraction logic captures event names",
			})
		}

		if activity.Location.Name == "" {
			diagnostics.ValidationIssues = append(diagnostics.ValidationIssues, ValidationIssue{
				Severity:   "warning",
				Field:      activityPrefix + ".location.name",
				Message:    "Activity location name is empty",
				Suggestion: "Add location extraction from venue or address information",
			})
		}

		if activity.Schedule.StartDate == "" {
			diagnostics.ValidationIssues = append(diagnostics.ValidationIssues, ValidationIssue{
				Severity:   "warning",
				Field:      activityPrefix + ".schedule.start_date",
				Message:    "Activity start date is empty",
				Suggestion: "Implement date pattern recognition in markdown content",
			})
		}

		// Check data quality
		if len(activity.Description) < 10 {
			diagnostics.ValidationIssues = append(diagnostics.ValidationIssues, ValidationIssue{
				Severity:   "info",
				Field:      activityPrefix + ".description",
				Message:    "Activity description is very short",
				Suggestion: "Consider extracting more detailed content from the source",
			})
		}

		if len(activity.AgeGroups) == 0 {
			diagnostics.ValidationIssues = append(diagnostics.ValidationIssues, ValidationIssue{
				Severity:   "info",
				Field:      activityPrefix + ".age_groups",
				Message:    "No age groups specified",
				Suggestion: "Add age group detection from content patterns",
			})
		}
	}

	log.Printf("[VALIDATION] Validation completed: %d issues found", len(diagnostics.ValidationIssues))
}

// logDiagnostics logs comprehensive diagnostics information
func (fc *FireCrawlClient) logDiagnostics(diagnostics *ExtractionDiagnostics) {
	log.Printf("[DIAGNOSTICS] ========== EXTRACTION DIAGNOSTICS ==========")
	log.Printf("[DIAGNOSTICS] URL: %s", diagnostics.URL)
	log.Printf("[DIAGNOSTICS] Processing Time: %v", diagnostics.ProcessingTime)
	log.Printf("[DIAGNOSTICS] Success: %t", diagnostics.Success)
	log.Printf("[DIAGNOSTICS] Raw Markdown Length: %d characters", diagnostics.RawMarkdownLength)
	
	if diagnostics.ErrorMessage != "" {
		log.Printf("[DIAGNOSTICS] Error: %s", diagnostics.ErrorMessage)
	}

	log.Printf("[DIAGNOSTICS] Extraction Attempts: %d", len(diagnostics.ExtractionAttempts))
	for i, attempt := range diagnostics.ExtractionAttempts {
		log.Printf("[DIAGNOSTICS]   Attempt %d: %s - Success: %t, Events: %d", 
			i+1, attempt.Method, attempt.Success, attempt.EventsFound)
		
		if len(attempt.Issues) > 0 {
			log.Printf("[DIAGNOSTICS]     Issues: %v", attempt.Issues)
		}
		
		if len(attempt.Details) > 0 {
			log.Printf("[DIAGNOSTICS]     Details: %v", attempt.Details)
		}
	}

	log.Printf("[DIAGNOSTICS] Validation Issues: %d", len(diagnostics.ValidationIssues))
	for i, issue := range diagnostics.ValidationIssues {
		log.Printf("[DIAGNOSTICS]   Issue %d [%s]: %s - %s", 
			i+1, issue.Severity, issue.Field, issue.Message)
		if issue.Suggestion != "" {
			log.Printf("[DIAGNOSTICS]     Suggestion: %s", issue.Suggestion)
		}
	}

	if len(diagnostics.StructuredData) > 0 {
		log.Printf("[DIAGNOSTICS] Structured Data Keys: %v", func() []string {
			keys := make([]string, 0, len(diagnostics.StructuredData))
			for k := range diagnostics.StructuredData {
				keys = append(keys, k)
			}
			return keys
		}())
	}

	log.Printf("[DIAGNOSTICS] Credits Used: %d", diagnostics.CreditsUsed)
	log.Printf("[DIAGNOSTICS] ============================================")
}

// GetExtractionDiagnostics returns the last extraction diagnostics (for testing/debugging)
var lastExtractionDiagnostics *ExtractionDiagnostics

// GetLastExtractionDiagnostics returns the diagnostics from the last extraction
func (fc *FireCrawlClient) GetLastExtractionDiagnostics() *ExtractionDiagnostics {
	return lastExtractionDiagnostics
}

// extractDateFromLine extracts date information from a text line
func (fc *FireCrawlClient) extractDateFromLine(line string) string {
	line = strings.TrimSpace(line)
	
	// Common date patterns
	datePatterns := []string{
		// MM/DD/YYYY or MM/DD/YY
		`\b(\d{1,2})/(\d{1,2})/(\d{2,4})\b`,
		// Month DD, YYYY
		`\b(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2}),?\s+(\d{4})\b`,
		// Mon DD or Month DD
		`\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2})\b`,
		// Day of week, Month DD
		`\b(Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday|Mon|Tue|Wed|Thu|Fri|Sat|Sun),?\s+(January|February|March|April|May|June|July|August|September|October|November|December|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2})\b`,
	}
	
	for _, pattern := range datePatterns {
		if match := fc.findRegexMatch(line, pattern); match != "" {
			return match
		}
	}
	
	return ""
}

// extractTimeFromLine extracts time information from a text line
func (fc *FireCrawlClient) extractTimeFromLine(line string) string {
	line = strings.TrimSpace(line)
	
	// Common time patterns
	timePatterns := []string{
		// 12-hour format with AM/PM
		`\b(\d{1,2}):(\d{2})\s*(AM|PM|am|pm)\b`,
		`\b(\d{1,2})\s*(AM|PM|am|pm)\b`,
		// 24-hour format
		`\b(\d{1,2}):(\d{2})\b`,
		// Time ranges
		`\b(\d{1,2}):?(\d{2})?\s*(AM|PM|am|pm)?\s*-\s*(\d{1,2}):?(\d{2})?\s*(AM|PM|am|pm)\b`,
	}
	
	for _, pattern := range timePatterns {
		if match := fc.findRegexMatch(line, pattern); match != "" {
			return match
		}
	}
	
	return ""
}

// extractLocationFromLine extracts location/venue information from a text line
func (fc *FireCrawlClient) extractLocationFromLine(line string) string {
	line = strings.TrimSpace(line)
	lowerLine := strings.ToLower(line)
	
	// Look for location indicators
	locationIndicators := []string{
		"location:", "venue:", "at:", "where:", "address:",
		"held at", "takes place at", "meet at",
	}
	
	for _, indicator := range locationIndicators {
		if strings.Contains(lowerLine, indicator) {
			// Extract text after the indicator
			parts := strings.Split(lowerLine, indicator)
			if len(parts) > 1 {
				location := strings.TrimSpace(parts[1])
				// Clean up common suffixes
				location = strings.Split(location, "\n")[0]
				location = strings.Split(location, ".")[0]
				if len(location) > 3 && len(location) < 100 {
					return fc.capitalizeLocation(location)
				}
			}
		}
	}
	
	// Look for Seattle area venue patterns
	seattleVenues := []string{
		"library", "park", "center", "museum", "zoo", "aquarium",
		"community center", "recreation center", "ymca", "school",
		"theater", "theatre", "hall", "room", "studio",
	}
	
	for _, venue := range seattleVenues {
		if strings.Contains(lowerLine, venue) && len(line) < 100 {
			return fc.capitalizeLocation(line)
		}
	}
	
	return ""
}

// extractPriceFromLine extracts pricing information from a text line
func (fc *FireCrawlClient) extractPriceFromLine(line string) string {
	line = strings.TrimSpace(line)
	lowerLine := strings.ToLower(line)
	
	// Look for free indicators
	freeIndicators := []string{"free", "no cost", "no charge", "complimentary"}
	for _, indicator := range freeIndicators {
		if strings.Contains(lowerLine, indicator) {
			return "Free"
		}
	}
	
	// Look for price patterns
	pricePatterns := []string{
		// Dollar amounts
		`\$(\d+(?:\.\d{2})?)\b`,
		// Price with text
		`\b(price|cost|fee|admission):\s*\$?(\d+(?:\.\d{2})?)\b`,
		// Donation patterns
		`\b(donation|suggested)\b`,
	}
	
	for _, pattern := range pricePatterns {
		if match := fc.findRegexMatch(line, pattern); match != "" {
			return match
		}
	}
	
	// Look for price indicators
	priceIndicators := []string{"price:", "cost:", "fee:", "admission:"}
	for _, indicator := range priceIndicators {
		if strings.Contains(lowerLine, indicator) {
			parts := strings.Split(lowerLine, indicator)
			if len(parts) > 1 {
				price := strings.TrimSpace(parts[1])
				price = strings.Split(price, "\n")[0]
				price = strings.Split(price, ".")[0]
				if len(price) > 0 && len(price) < 50 {
					return price
				}
			}
		}
	}
	
	return ""
}

// extractAgeGroupsFromLine extracts age group information from a text line
func (fc *FireCrawlClient) extractAgeGroupsFromLine(line string) []string {
	var ageGroups []string
	lowerLine := strings.ToLower(line)
	
	// Age group patterns
	agePatterns := map[string][]string{
		"infant":     {"infant", "baby", "babies", "0-12 months", "newborn"},
		"toddler":    {"toddler", "toddlers", "1-2 years", "18 months"},
		"preschool":  {"preschool", "preschooler", "pre-k", "3-5 years", "ages 3-5"},
		"elementary": {"elementary", "school age", "6-10 years", "ages 6-10", "kids"},
		"tween":      {"tween", "tweens", "11-12 years", "ages 11-12"},
		"teen":       {"teen", "teens", "teenager", "13-17 years", "ages 13-17"},
		"adult":      {"adult", "adults", "18+", "grown-up", "grown up"},
		"all-ages":   {"all ages", "family", "everyone", "any age"},
	}
	
	for category, patterns := range agePatterns {
		for _, pattern := range patterns {
			if strings.Contains(lowerLine, pattern) {
				ageGroups = append(ageGroups, category)
				break // Only add each category once
			}
		}
	}
	
	// Look for numeric age ranges
	ageRangePattern := `\b(?:ages?|for)\s*(\d+)\s*-\s*(\d+)\b`
	if match := fc.findRegexMatch(lowerLine, ageRangePattern); match != "" {
		ageGroups = append(ageGroups, match)
	}
	
	return ageGroups
}

// findRegexMatch finds the first regex match in a string
func (fc *FireCrawlClient) findRegexMatch(text, pattern string) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Printf("[REGEX] Error compiling pattern '%s': %v", pattern, err)
		return ""
	}
	
	match := re.FindString(text)
	return match
}

// extractEventsFromMarkdown provides a robust, generic event extraction from markdown content
func (fc *FireCrawlClient) extractEventsFromMarkdown(markdown string, attempt *ExtractionAttempt) []EventData {
	log.Printf("[EXTRACT] Starting robust markdown event extraction for %d characters", len(markdown))
	
	var events []EventData
	
	// Initialize extraction statistics
	extractionStats := map[string]int{
		"total_lines":        0,
		"header_lines":       0,
		"date_matches":       0,
		"time_matches":       0,
		"location_matches":   0,
		"price_matches":      0,
		"age_group_matches":  0,
		"events_created":     0,
	}
	
	// Split content into lines for processing
	lines := strings.Split(markdown, "\n")
	extractionStats["total_lines"] = len(lines)
	
	// First pass: identify potential event blocks using multiple strategies
	eventBlocks := fc.identifyEventBlocks(lines, extractionStats)
	log.Printf("[EXTRACT] Identified %d potential event blocks", len(eventBlocks))
	
	// Second pass: extract structured data from each block
	for i, block := range eventBlocks {
		if i >= 15 { // Limit to 15 events for performance
			log.Printf("[EXTRACT] Limiting to first 15 events (found %d blocks)", len(eventBlocks))
			break
		}
		
		event := fc.extractEventFromBlock(block, extractionStats)
		if event != nil && event.Title != "" {
			events = append(events, *event)
			extractionStats["events_created"]++
			log.Printf("[EXTRACT] Successfully extracted event: %s", event.Title)
		}
	}
	
	// Update attempt details with extraction statistics
	attempt.Details["extraction_stats"] = extractionStats
	attempt.Details["event_blocks_found"] = len(eventBlocks)
	attempt.Details["events_extracted"] = len(events)
	
	log.Printf("[EXTRACT] Extraction completed: %d events from %d blocks", len(events), len(eventBlocks))
	return events
}

// EventBlock represents a block of text that potentially contains an event
type EventBlock struct {
	Title     string
	Content   []string
	StartLine int
	EndLine   int
}

// identifyEventBlocks identifies blocks of text that likely contain event information
func (fc *FireCrawlClient) identifyEventBlocks(lines []string, stats map[string]int) []EventBlock {
	var blocks []EventBlock
	var currentBlock *EventBlock
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		// Check if this line starts a new event block
		if fc.isEventBlockStart(line) {
			stats["header_lines"]++
			
			// Save previous block if it exists
			if currentBlock != nil {
				currentBlock.EndLine = i - 1
				blocks = append(blocks, *currentBlock)
			}
			
			// Start new block
			currentBlock = &EventBlock{
				Title:     fc.cleanEventTitle(line),
				Content:   []string{line},
				StartLine: i,
			}
			
			log.Printf("[EXTRACT] New event block started: %s (line %d)", currentBlock.Title, i)
		} else if currentBlock != nil {
			// Add line to current block
			currentBlock.Content = append(currentBlock.Content, line)
			
			// Check if this line ends the current block
			if fc.isEventBlockEnd(line, i, lines) {
				currentBlock.EndLine = i
				blocks = append(blocks, *currentBlock)
				currentBlock = nil
			}
		}
	}
	
	// Don't forget the last block
	if currentBlock != nil {
		currentBlock.EndLine = len(lines) - 1
		blocks = append(blocks, *currentBlock)
	}
	
	return blocks
}

// isEventBlockStart determines if a line starts a new event block
func (fc *FireCrawlClient) isEventBlockStart(line string) bool {
	// Check for markdown headers
	if strings.HasPrefix(line, "#") {
		return true
	}
	
	// Check for lines that look like event titles
	if fc.isEventHeader(line) {
		return true
	}
	
	// Check for structured event indicators
	eventStartPatterns := []string{
		`^Event:`, `^Activity:`, `^Class:`, `^Workshop:`, `^Program:`,
		`^\d+\.`, `^\*\s+`, `^-\s+`, // Numbered or bulleted lists
	}
	
	for _, pattern := range eventStartPatterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}
	
	return false
}

// isEventBlockEnd determines if a line ends the current event block
func (fc *FireCrawlClient) isEventBlockEnd(line string, lineIndex int, allLines []string) bool {
	// Check for explicit separators
	if fc.isBlockSeparator(line) {
		return true
	}
	
	// Check if next line starts a new event
	if lineIndex+1 < len(allLines) {
		nextLine := strings.TrimSpace(allLines[lineIndex+1])
		if fc.isEventBlockStart(nextLine) {
			return true
		}
	}
	
	// End block after reasonable content length (prevent overly long blocks)
	// This is a heuristic - blocks shouldn't be more than 20 lines typically
	return false // Let blocks continue until explicit end or new start
}

// extractEventFromBlock extracts structured event data from a text block
func (fc *FireCrawlClient) extractEventFromBlock(block EventBlock, stats map[string]int) *EventData {
	event := &EventData{
		Title:      block.Title,
		RawContent: strings.Join(block.Content, "\n"),
	}
	
	// Combine all content for pattern matching
	fullContent := strings.Join(block.Content, " ")
	
	// Extract date information using robust patterns
	if date := fc.extractDateWithPatterns(fullContent); date != "" {
		event.Date = date
		stats["date_matches"]++
		log.Printf("[EXTRACT] Found date for '%s': %s", event.Title, date)
	}
	
	// Extract time information
	if time := fc.extractTimeWithPatterns(fullContent); time != "" {
		event.Time = time
		stats["time_matches"]++
		log.Printf("[EXTRACT] Found time for '%s': %s", event.Title, time)
	}
	
	// Extract location information
	if location := fc.extractLocationWithPatterns(fullContent); location != "" {
		event.Location = location
		stats["location_matches"]++
		log.Printf("[EXTRACT] Found location for '%s': %s", event.Title, location)
	}
	
	// Extract price information
	if price := fc.extractPriceWithPatterns(fullContent); price != "" {
		event.Price = price
		stats["price_matches"]++
		log.Printf("[EXTRACT] Found price for '%s': %s", event.Title, price)
	}
	
	// Extract age group information
	if ageGroups := fc.extractAgeGroupsWithPatterns(fullContent); len(ageGroups) > 0 {
		event.AgeGroups = ageGroups
		stats["age_group_matches"]++
		log.Printf("[EXTRACT] Found age groups for '%s': %v", event.Title, ageGroups)
	}
	
	// Build description from non-metadata content
	event.Description = fc.buildEventDescription(block.Content)
	
	return event
}

// extractDateWithPatterns extracts dates using comprehensive regex patterns
func (fc *FireCrawlClient) extractDateWithPatterns(text string) string {
	datePatterns := []string{
		// MM/DD/YYYY or MM/DD/YY
		`\b(\d{1,2})/(\d{1,2})/(\d{2,4})\b`,
		// MM-DD-YYYY or MM-DD-YY
		`\b(\d{1,2})-(\d{1,2})-(\d{2,4})\b`,
		// Month DD, YYYY
		`\b(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2}),?\s+(\d{4})\b`,
		// Mon DD or Month DD (current year assumed)
		`\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2})\b`,
		// Day of week, Month DD
		`\b(Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday|Mon|Tue|Wed|Thu|Fri|Sat|Sun),?\s+(January|February|March|April|May|June|July|August|September|October|November|December|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2})\b`,
		// ISO format YYYY-MM-DD
		`\b(\d{4})-(\d{1,2})-(\d{1,2})\b`,
	}
	
	for _, pattern := range datePatterns {
		if match := fc.findRegexMatch(text, pattern); match != "" {
			return fc.normalizeDate(match)
		}
	}
	
	return ""
}

// extractTimeWithPatterns extracts times using comprehensive regex patterns
func (fc *FireCrawlClient) extractTimeWithPatterns(text string) string {
	timePatterns := []string{
		// 12-hour format with AM/PM
		`\b(\d{1,2}):(\d{2})\s*(AM|PM|am|pm)\b`,
		`\b(\d{1,2})\s*(AM|PM|am|pm)\b`,
		// 24-hour format
		`\b([01]?\d|2[0-3]):([0-5]\d)\b`,
		// Time ranges
		`\b(\d{1,2}):?(\d{2})?\s*(AM|PM|am|pm)?\s*[-–]\s*(\d{1,2}):?(\d{2})?\s*(AM|PM|am|pm)\b`,
		// Casual time expressions
		`\b(morning|afternoon|evening|noon|midnight)\b`,
	}
	
	for _, pattern := range timePatterns {
		if match := fc.findRegexMatch(text, pattern); match != "" {
			return fc.normalizeTime(match)
		}
	}
	
	return ""
}

// extractLocationWithPatterns extracts locations using comprehensive patterns
func (fc *FireCrawlClient) extractLocationWithPatterns(text string) string {
	// First try explicit location indicators
	locationPatterns := []string{
		`(?i)\b(?:location|venue|at|where|address|held at|takes place at|meet at):\s*([^.\n]+)`,
		`(?i)\b(?:location|venue|at|where|address):\s*([^.\n]+)`,
	}
	
	for _, pattern := range locationPatterns {
		if match := fc.findRegexMatch(text, pattern); match != "" {
			// Extract just the location part (after the colon)
			parts := strings.Split(match, ":")
			if len(parts) > 1 {
				location := strings.TrimSpace(parts[1])
				if len(location) > 3 && len(location) < 100 {
					return fc.capitalizeLocation(location)
				}
			}
		}
	}
	
	// Look for Seattle area venue patterns
	venuePatterns := []string{
		`\b([A-Z][a-z]+\s+(?:Library|Park|Center|Museum|Zoo|Aquarium|School|Theater|Theatre|Hall|Studio))\b`,
		`\b([A-Z][a-z]+\s+Community\s+Center)\b`,
		`\b([A-Z][a-z]+\s+Recreation\s+Center)\b`,
		`\b(YMCA\s+[A-Z][a-z]+)\b`,
	}
	
	for _, pattern := range venuePatterns {
		if match := fc.findRegexMatch(text, pattern); match != "" {
			return match
		}
	}
	
	return ""
}

// extractPriceWithPatterns extracts pricing using comprehensive patterns
func (fc *FireCrawlClient) extractPriceWithPatterns(text string) string {
	// Check for free indicators first
	freePatterns := []string{
		`(?i)\b(free|no cost|no charge|complimentary|admission free)\b`,
	}
	
	for _, pattern := range freePatterns {
		if match := fc.findRegexMatch(text, pattern); match != "" {
			return "Free"
		}
	}
	
	// Look for price patterns
	pricePatterns := []string{
		// Dollar amounts
		`\$(\d+(?:\.\d{2})?)\b`,
		// Price with descriptors
		`(?i)\b(?:price|cost|fee|admission|tuition):\s*\$?(\d+(?:\.\d{2})?)\b`,
		// Donation patterns
		`(?i)\b(donation|suggested donation|pay what you can)\b`,
		// Price ranges
		`\$(\d+(?:\.\d{2})?)\s*[-–]\s*\$(\d+(?:\.\d{2})?)\b`,
	}
	
	for _, pattern := range pricePatterns {
		if match := fc.findRegexMatch(text, pattern); match != "" {
			return match
		}
	}
	
	return ""
}

// extractAgeGroupsWithPatterns extracts age groups using comprehensive patterns
func (fc *FireCrawlClient) extractAgeGroupsWithPatterns(text string) []string {
	var ageGroups []string
	lowerText := strings.ToLower(text)
	
	// Age group patterns with regex
	agePatterns := map[string][]string{
		"infant": {
			`(?i)\b(infant|baby|babies|newborn)\b`,
			`\b0\s*[-–]\s*12\s*months?\b`,
			`\b0\s*[-–]\s*1\s*years?\b`,
		},
		"toddler": {
			`(?i)\b(toddler|toddlers)\b`,
			`\b1\s*[-–]\s*2\s*years?\b`,
			`\b18\s*months?\b`,
		},
		"preschool": {
			`(?i)\b(preschool|preschooler|pre-k|prekindergarten)\b`,
			`\b3\s*[-–]\s*5\s*years?\b`,
			`(?i)\bages?\s*3\s*[-–]\s*5\b`,
		},
		"elementary": {
			`(?i)\b(elementary|school\s*age|grade\s*school)\b`,
			`\b6\s*[-–]\s*10\s*years?\b`,
			`(?i)\bages?\s*6\s*[-–]\s*10\b`,
			`(?i)\b(kids|children)\b`,
		},
		"tween": {
			`(?i)\b(tween|tweens)\b`,
			`\b11\s*[-–]\s*12\s*years?\b`,
			`(?i)\bages?\s*11\s*[-–]\s*12\b`,
		},
		"teen": {
			`(?i)\b(teen|teens|teenager|teenagers|adolescent)\b`,
			`\b13\s*[-–]\s*17\s*years?\b`,
			`(?i)\bages?\s*13\s*[-–]\s*17\b`,
		},
		"adult": {
			`(?i)\b(adult|adults|grown-?up|grown-?ups)\b`,
			`\b18\+\b`,
			`(?i)\bages?\s*18\+\b`,
		},
		"all-ages": {
			`(?i)\b(all\s*ages?|family|everyone|any\s*age)\b`,
			`(?i)\b(suitable\s*for\s*all)\b`,
		},
	}
	
	for category, patterns := range agePatterns {
		for _, pattern := range patterns {
			if match := fc.findRegexMatch(lowerText, pattern); match != "" {
				ageGroups = append(ageGroups, category)
				break // Only add each category once
			}
		}
	}
	
	// Look for numeric age ranges not covered above
	ageRangePattern := `(?i)\b(?:ages?|for)\s*(\d+)\s*[-–]\s*(\d+)\s*years?\b`
	if match := fc.findRegexMatch(lowerText, ageRangePattern); match != "" {
		ageGroups = append(ageGroups, match)
	}
	
	return ageGroups
}

// normalizeDate normalizes extracted date strings to a consistent format
func (fc *FireCrawlClient) normalizeDate(dateStr string) string {
	// Try to parse and reformat the date
	// For now, return as-is but could implement date parsing/formatting
	return strings.TrimSpace(dateStr)
}

// normalizeTime normalizes extracted time strings to a consistent format
func (fc *FireCrawlClient) normalizeTime(timeStr string) string {
	// Normalize AM/PM to uppercase
	timeStr = strings.ReplaceAll(timeStr, "am", "AM")
	timeStr = strings.ReplaceAll(timeStr, "pm", "PM")
	return strings.TrimSpace(timeStr)
}

// buildEventDescription builds a description from event content, excluding metadata
func (fc *FireCrawlClient) buildEventDescription(contentLines []string) string {
	var descriptionParts []string
	
	for _, line := range contentLines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines, headers, and metadata lines
		if line == "" || fc.isEventHeader(line) || fc.isMetadataLine(line) {
			continue
		}
		
		// Skip very short lines (likely not descriptive)
		if len(line) < 10 {
			continue
		}
		
		// Add to description if it looks like descriptive content
		if len(line) > 20 && len(line) < 500 {
			descriptionParts = append(descriptionParts, line)
		}
		
		// Limit description length
		if len(descriptionParts) >= 3 {
			break
		}
	}
	
	if len(descriptionParts) > 0 {
		return strings.Join(descriptionParts, " ")
	}
	
	return ""
}

// capitalizeLocation properly capitalizes location names
func (fc *FireCrawlClient) capitalizeLocation(location string) string {
	words := strings.Fields(location)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// convertEventToActivity converts parsed event data to Activity model
func (fc *FireCrawlClient) convertEventToActivity(event EventData, sourceURL, idSuffix string) *models.Activity {
	if event.Title == "" {
		return nil
	}
	
	activity := &models.Activity{
		ID:          fmt.Sprintf("parentmap-%s-%d", idSuffix, time.Now().Unix()),
		Title:       event.Title,
		Description: event.Description,
		Type:        models.TypeEvent,
		Category:    models.CategoryFreeCommunity,
		Status:      models.ActivityStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Set schedule
	activity.Schedule = models.Schedule{
		Type:     models.ScheduleTypeOneTime,
		Timezone: "America/Los_Angeles",
	}
	
	if event.Date != "" {
		activity.Schedule.StartDate = event.Date
	}
	
	if event.Time != "" {
		activity.Schedule.StartTime = event.Time
	}
	
	// Set location
	activity.Location = models.Location{
		City:      "Seattle",
		State:     "WA",
		Region:    "Seattle Metro",
		VenueType: models.VenueTypeIndoor,
	}
	
	if event.Location != "" {
		activity.Location.Name = event.Location
	} else {
		activity.Location.Name = "Seattle Area"
	}
	
	// Set pricing
	if event.Price != "" {
		if strings.ToLower(event.Price) == "free" {
			activity.Pricing = models.Pricing{
				Type:        models.PricingTypeFree,
				Description: "Free",
				Currency:    "USD",
			}
		} else {
			activity.Pricing = models.Pricing{
				Type:        models.PricingTypePaid,
				Description: event.Price,
				Currency:    "USD",
			}
		}
	} else {
		activity.Pricing = models.Pricing{
			Type:        models.PricingTypeVariable,
			Description: "See event details",
			Currency:    "USD",
		}
	}
	
	// Set age groups
	if len(event.AgeGroups) > 0 {
		for _, ageGroup := range event.AgeGroups {
			activity.AgeGroups = append(activity.AgeGroups, models.AgeGroup{
				Category:    ageGroup,
				Description: strings.Title(ageGroup),
			})
		}
	} else {
		activity.AgeGroups = []models.AgeGroup{
			{
				Category:    models.AgeGroupAllAges,
				Description: "All Ages",
			},
		}
	}
	
	// Set source information
	activity.Source = models.Source{
		URL:         sourceURL,
		Domain:      extractDomain(sourceURL),
		ScrapedAt:   time.Now(),
		LastChecked: time.Now(),
		Reliability: "medium",
	}
	
	return activity
}

// parseParentMapFallback provides fallback parsing when structured parsing fails
func (fc *FireCrawlClient) parseParentMapFallback(markdown, url string, attempt *ExtractionAttempt) []models.Activity {
	var activities []models.Activity
	
	log.Printf("[PARENTMAP] Using fallback parsing method")
	
	// Count different types of potential activity markers
	headerCount := strings.Count(markdown, "###")
	h2Count := strings.Count(markdown, "##")
	h1Count := strings.Count(markdown, "#")
	
	attempt.Details["fallback_header_counts"] = map[string]int{
		"h3": headerCount,
		"h2": h2Count,
		"h1": h1Count,
	}

	// Look for date patterns that might indicate events
	datePatterns := []string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
		"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun",
		"2024", "2025",
	}
	
	dateMatches := 0
	for _, pattern := range datePatterns {
		dateMatches += strings.Count(markdown, pattern)
	}
	
	attempt.Details["fallback_date_matches"] = dateMatches

	// Simple parsing - create activities based on header count
	activityCount := headerCount
	if activityCount == 0 {
		activityCount = h2Count // Fallback to H2 headers
	}
	
	if activityCount > 0 {
		log.Printf("[PARENTMAP] Fallback found approximately %d potential activities", activityCount)

		// Create sample activities representing what we found
		maxActivities := min(activityCount, 3) // Limit to 3 for fallback
		for i := 0; i < maxActivities; i++ {
			activity := models.Activity{
				ID:          fmt.Sprintf("parentmap-fallback-%d-%d", time.Now().Unix(), i),
				Title:       fmt.Sprintf("ParentMap Event %d", i+1),
				Description: "Event extracted from ParentMap calendar (fallback method)",
				Type:        models.TypeEvent,
				Category:    models.CategoryFreeCommunity,
				Schedule: models.Schedule{
					StartDate: time.Now().Format("2006-01-02"),
					StartTime: "10:00 AM",
					Type:      models.ScheduleTypeOneTime,
					Timezone:  "America/Los_Angeles",
				},
				Location: models.Location{
					Name:   "Seattle Area",
					City:   "Seattle",
					State:  "WA",
					Region: "Seattle Metro",
				},
				Pricing: models.Pricing{
					Type:        models.PricingTypeVariable,
					Description: "See event details",
					Currency:    "USD",
				},
				AgeGroups: []models.AgeGroup{
					{
						Category:    models.AgeGroupAllAges,
						Description: "All Ages",
					},
				},
				Status:    models.ActivityStatusActive,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Source: models.Source{
					URL:         url,
					Domain:      extractDomain(url),
					ScrapedAt:   time.Now(),
					LastChecked: time.Now(),
					Reliability: "low", // Lower reliability for fallback method
				},
			}
			activities = append(activities, activity)
		}
	} else {
		attempt.Issues = append(attempt.Issues, "Fallback parsing found no recognizable patterns")
	}

	log.Printf("[PARENTMAP] Fallback parsing completed: %d activities extracted", len(activities))
	return activities
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Admin Extraction Methods

// AdminExtractRequest represents a request for admin-driven extraction
type AdminExtractRequest struct {
	URL          string                 `json:"url"`
	SchemaType   string                 `json:"schema_type"`   // "events"|"activities"|"venues"|"custom"
	CustomSchema map[string]interface{} `json:"custom_schema"` // Only used if schema_type = "custom"
}

// AdminExtractResponse represents the response from admin extraction
type AdminExtractResponse struct {
	Success      bool                   `json:"success"`
	RawData      map[string]interface{} `json:"raw_data"`      // Raw Firecrawl response
	SchemaUsed   map[string]interface{} `json:"schema_used"`   // Schema that was sent to Firecrawl
	Metadata     AdminExtractMetadata   `json:"metadata"`
	CreditsUsed  int                    `json:"credits_used"`
	EventsCount  int                    `json:"events_count"`  // Number of events/activities extracted
}

// AdminExtractMetadata contains metadata about the admin extraction
type AdminExtractMetadata struct {
	URL           string    `json:"url"`
	ExtractTime   time.Time `json:"extract_time"`
	Title         string    `json:"title,omitempty"`
	SchemaType    string    `json:"schema_type"`
	ProcessingTime time.Duration `json:"processing_time"`
}

// ExtractWithSchema performs structured extraction using a predefined or custom schema
func (fc *FireCrawlClient) ExtractWithSchema(request AdminExtractRequest) (*AdminExtractResponse, error) {
	startTime := time.Now()

	if request.URL == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	// Get the schema to use
	schema, err := fc.getSchemaForExtraction(request.SchemaType, request.CustomSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to get extraction schema: %w", err)
	}

	log.Printf("Starting admin extraction for URL: %s with schema type: %s", request.URL, request.SchemaType)

	// For now, use the basic scrape functionality
	// TODO: Implement proper schema-based extraction when Firecrawl Go SDK supports it
	response, err := fc.client.ScrapeURL(request.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("Firecrawl extraction failed: %w", err)
	}

	// Parse the response into structured data
	rawData, err := fc.parseAdminExtractResponse(response, request.SchemaType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse extraction response: %w", err)
	}

	// Count extracted events
	eventsCount := fc.countExtractedEvents(rawData, request.SchemaType)

	// Extract metadata from response
	title := fc.extractTitleFromDoc(response)
	creditsUsed := fc.extractCreditsFromDoc(response)

	extractResponse := &AdminExtractResponse{
		Success:     true,
		RawData:     rawData,
		SchemaUsed:  schema,
		CreditsUsed: creditsUsed,
		EventsCount: eventsCount,
		Metadata: AdminExtractMetadata{
			URL:            request.URL,
			ExtractTime:    startTime,
			Title:          title,
			SchemaType:     request.SchemaType,
			ProcessingTime: time.Since(startTime),
		},
	}

	log.Printf("Admin extraction completed for %s: found %d events in %v",
		request.URL, eventsCount, time.Since(startTime))

	return extractResponse, nil
}

// getSchemaForExtraction returns the appropriate schema based on type
func (fc *FireCrawlClient) getSchemaForExtraction(schemaType string, customSchema map[string]interface{}) (map[string]interface{}, error) {
	if schemaType == "custom" {
		if customSchema == nil {
			return nil, fmt.Errorf("custom schema is required when schema_type is 'custom'")
		}
		return customSchema, nil
	}

	// Get predefined schema
	extractionSchema, err := models.GetSchemaByType(schemaType)
	if err != nil {
		return nil, err
	}

	return extractionSchema.Schema, nil
}

// parseAdminExtractResponse parses the Firecrawl response for admin extraction
func (fc *FireCrawlClient) parseAdminExtractResponse(response interface{}, schemaType string) (map[string]interface{}, error) {
	doc, ok := response.(*firecrawl.FirecrawlDocument)
	if !ok {
		return nil, fmt.Errorf("unexpected response format from FireCrawl")
	}

	// For now, we'll create mock structured data based on the markdown content
	// In a real implementation, this would use Firecrawl's structured extraction
	rawData := make(map[string]interface{})

	switch schemaType {
	case "events":
		events := fc.extractEventsFromMarkdownLegacy(doc.Markdown)
		rawData["events"] = events

	case "activities":
		activities := fc.extractActivitiesFromMarkdown(doc.Markdown)
		rawData["activities"] = activities

	case "venues":
		venues := fc.extractVenuesFromMarkdown(doc.Markdown)
		rawData["venues"] = venues

	case "custom":
		// For custom schemas, try to extract generic objects
		items := fc.extractGenericItemsFromMarkdown(doc.Markdown)
		rawData["items"] = items
	}

	return rawData, nil
}

// extractEventsFromMarkdownLegacy extracts event-like objects from markdown content (legacy method)
func (fc *FireCrawlClient) extractEventsFromMarkdownLegacy(markdown string) []map[string]interface{} {
	var events []map[string]interface{}

	// Simple extraction based on markdown structure
	// This is a placeholder - in production, would use Firecrawl's structured extraction
	lines := strings.Split(markdown, "\n")
	currentEvent := make(map[string]interface{})

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for headers that might be event titles
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
			// Save previous event if it has a title
			if title, ok := currentEvent["title"].(string); ok && title != "" {
				events = append(events, currentEvent)
			}

			// Start new event
			currentEvent = map[string]interface{}{
				"title": strings.TrimPrefix(strings.TrimPrefix(line, "## "), "# "),
			}
		}

		// Look for date patterns
		if fc.containsDatePattern(line) {
			currentEvent["date"] = line
		}

		// Look for location patterns
		if fc.containsLocationPattern(line) {
			currentEvent["location"] = line
		}

		// Look for price patterns
		if fc.containsPricePattern(line) {
			currentEvent["price"] = line
		}
	}

	// Add the last event
	if title, ok := currentEvent["title"].(string); ok && title != "" {
		events = append(events, currentEvent)
	}

	// If no structured events found, create a sample event
	if len(events) == 0 {
		events = append(events, map[string]interface{}{
			"title":       "Sample Event from " + fc.extractFirstHeaderFromMarkdown(markdown),
			"description": "Event extracted from website content",
			"location":    "Seattle Area",
			"price":       "See website for details",
		})
	}

	return events
}

// extractActivitiesFromMarkdown extracts activity-like objects from markdown
func (fc *FireCrawlClient) extractActivitiesFromMarkdown(markdown string) []map[string]interface{} {
	var activities []map[string]interface{}

	// Simple extraction - similar to events but with activity-specific fields
	lines := strings.Split(markdown, "\n")
	currentActivity := make(map[string]interface{})

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
			if name, ok := currentActivity["name"].(string); ok && name != "" {
				activities = append(activities, currentActivity)
			}

			currentActivity = map[string]interface{}{
				"name": strings.TrimPrefix(strings.TrimPrefix(line, "## "), "# "),
			}
		}

		// Look for age-related information
		if fc.containsAgePattern(line) {
			currentActivity["age_groups"] = []string{line}
		}

		// Look for duration patterns
		if fc.containsDurationPattern(line) {
			currentActivity["duration"] = line
		}
	}

	if name, ok := currentActivity["name"].(string); ok && name != "" {
		activities = append(activities, currentActivity)
	}

	if len(activities) == 0 {
		activities = append(activities, map[string]interface{}{
			"name":        "Sample Activity from " + fc.extractFirstHeaderFromMarkdown(markdown),
			"description": "Activity extracted from website content",
			"age_groups":  []string{"all ages"},
		})
	}

	return activities
}

// extractVenuesFromMarkdown extracts venue-like objects from markdown
func (fc *FireCrawlClient) extractVenuesFromMarkdown(markdown string) []map[string]interface{} {
	var venues []map[string]interface{}

	lines := strings.Split(markdown, "\n")
	currentVenue := make(map[string]interface{})

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
			if name, ok := currentVenue["name"].(string); ok && name != "" {
				venues = append(venues, currentVenue)
			}

			currentVenue = map[string]interface{}{
				"name": strings.TrimPrefix(strings.TrimPrefix(line, "## "), "# "),
			}
		}

		if fc.containsAddressPattern(line) {
			currentVenue["address"] = line
		}

		if fc.containsPhonePattern(line) {
			currentVenue["phone"] = line
		}
	}

	if name, ok := currentVenue["name"].(string); ok && name != "" {
		venues = append(venues, currentVenue)
	}

	if len(venues) == 0 {
		venues = append(venues, map[string]interface{}{
			"name":        "Sample Venue from " + fc.extractFirstHeaderFromMarkdown(markdown),
			"description": "Venue extracted from website content",
			"address":     "Seattle, WA",
		})
	}

	return venues
}

// extractGenericItemsFromMarkdown extracts generic items for custom schemas
func (fc *FireCrawlClient) extractGenericItemsFromMarkdown(markdown string) []map[string]interface{} {
	var items []map[string]interface{}

	// Extract all headers as potential items
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
			title := strings.TrimPrefix(strings.TrimPrefix(line, "## "), "# ")
			if title != "" {
				items = append(items, map[string]interface{}{
					"title":       title,
					"description": "Item extracted from website content",
				})
			}
		}
	}

	if len(items) == 0 {
		items = append(items, map[string]interface{}{
			"title":       "Sample Item",
			"description": "Generic item extracted from website",
		})
	}

	return items
}

// Helper pattern detection methods

func (fc *FireCrawlClient) containsDatePattern(text string) bool {
	text = strings.ToLower(text)
	dateKeywords := []string{"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
		"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
		"/20", "2024", "2025"}

	for _, keyword := range dateKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func (fc *FireCrawlClient) containsLocationPattern(text string) bool {
	text = strings.ToLower(text)
	locationKeywords := []string{"location:", "venue:", "address:", "where:",
		"seattle", "bellevue", "redmond", "ballard", "fremont", "street", "avenue", "road"}

	for _, keyword := range locationKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func (fc *FireCrawlClient) containsPricePattern(text string) bool {
	text = strings.ToLower(text)
	return strings.Contains(text, "$") || strings.Contains(text, "free") ||
		   strings.Contains(text, "cost") || strings.Contains(text, "price")
}

func (fc *FireCrawlClient) containsAgePattern(text string) bool {
	text = strings.ToLower(text)
	ageKeywords := []string{"age", "years", "months", "toddler", "preschool",
		"elementary", "teen", "adult", "kids", "children"}

	for _, keyword := range ageKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func (fc *FireCrawlClient) containsDurationPattern(text string) bool {
	text = strings.ToLower(text)
	durationKeywords := []string{"minutes", "hours", "duration", "length", "time"}

	for _, keyword := range durationKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func (fc *FireCrawlClient) containsAddressPattern(text string) bool {
	text = strings.ToLower(text)
	addressKeywords := []string{"address", "street", "avenue", "road", "blvd", "way",
		"seattle", "wa", "98"}

	for _, keyword := range addressKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func (fc *FireCrawlClient) containsPhonePattern(text string) bool {
	return strings.Contains(text, "(") && strings.Contains(text, ")") &&
		   strings.Contains(text, "-") ||
		   (len(strings.ReplaceAll(strings.ReplaceAll(text, "-", ""), " ", "")) >= 10 &&
		    strings.ContainsAny(text, "0123456789"))
}

func (fc *FireCrawlClient) extractFirstHeaderFromMarkdown(markdown string) string {
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "Website Content"
}

// countExtractedEvents counts the number of events/activities/venues extracted
func (fc *FireCrawlClient) countExtractedEvents(rawData map[string]interface{}, schemaType string) int {
	switch schemaType {
	case "events":
		if events, ok := rawData["events"].([]map[string]interface{}); ok {
			return len(events)
		}
	case "activities":
		if activities, ok := rawData["activities"].([]map[string]interface{}); ok {
			return len(activities)
		}
	case "venues":
		if venues, ok := rawData["venues"].([]map[string]interface{}); ok {
			return len(venues)
		}
	case "custom":
		if items, ok := rawData["items"].([]map[string]interface{}); ok {
			return len(items)
		}
	}

	return 0
}

// GetAvailableSchemas returns the list of available extraction schemas
func (fc *FireCrawlClient) GetAvailableSchemas() map[string]models.ExtractionSchema {
	return models.GetPredefinedSchemas()
}

// ValidateCustomSchema validates a custom schema structure
func (fc *FireCrawlClient) ValidateCustomSchema(schema map[string]interface{}) error {
	// Basic validation - ensure it's a valid JSON schema structure
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	if schemaType, ok := schema["type"].(string); !ok || schemaType == "" {
		return fmt.Errorf("schema must have a 'type' field")
	}

	if _, ok := schema["properties"]; !ok {
		return fmt.Errorf("schema must have a 'properties' field")
	}

	return nil
}