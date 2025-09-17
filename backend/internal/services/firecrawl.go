package services

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mendableai/firecrawl-go"
	"seattle-family-activities-scraper/internal/models"
)

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

	if url == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	// Define the schema for activity extraction
	// TODO: Will need to properly integrate this schema once we figure out the correct parameter structure
	schema := getActivityExtractionSchema()
	_ = schema // Suppress unused variable warning

	log.Printf("Starting FireCrawl extract for URL: %s", url)

	// Make the extract request using ScrapeURL with extraction parameters
	// Note: Using nil for now - will need to create proper ScrapeParams struct
	response, err := fc.client.ScrapeURL(url, nil)
	if err != nil {
		return nil, fmt.Errorf("FireCrawl extract failed: %w", err)
	}

	// Parse the response
	extractResponse, err := fc.parseExtractResponse(response, url, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse extract response: %w", err)
	}

	log.Printf("Successfully extracted %d activities from %s in %v",
		len(extractResponse.Data.Activities), url, time.Since(startTime))

	return extractResponse, nil
}

// parseExtractResponse parses the FireCrawl response into our structure
func (fc *FireCrawlClient) parseExtractResponse(response interface{}, url string, startTime time.Time) (*FireCrawlExtractResponse, error) {
	// Convert response to our expected format
	// Note: The actual implementation will depend on the exact structure returned by FireCrawl
	// This is a placeholder that we'll need to adjust based on the actual API response

	responseMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format from FireCrawl")
	}

	// Extract activities from the response
	var activities []models.Activity

	// Check if data field exists and contains activities
	if data, exists := responseMap["data"]; exists {
		if dataMap, ok := data.(map[string]interface{}); ok {
			if activitiesRaw, exists := dataMap["activities"]; exists {
				// Convert raw activities to our Activity model
				activities, _ = fc.convertToActivities(activitiesRaw, url)
			}
		}
	}

	return &FireCrawlExtractResponse{
		Success: true,
		Data: ActivityExtractionData{
			Activities: activities,
		},
		Metadata: ExtractMetadata{
			URL:         url,
			ExtractTime: startTime,
			Title:       fc.extractTitle(responseMap),
		},
		CreditsUsed: fc.extractCreditsUsed(responseMap),
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