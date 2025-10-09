package services

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"seattle-family-activities-scraper/internal/models"
)

// ConversionDiagnostics captures detailed information about the schema conversion process
type ConversionDiagnostics struct {
	AdminEventID       string                 `json:"admin_event_id"`
	SourceURL          string                 `json:"source_url"`
	SchemaType         string                 `json:"schema_type"`
	StartTime          time.Time              `json:"start_time"`
	EndTime            time.Time              `json:"end_time"`
	ProcessingTime     time.Duration          `json:"processing_time"`
	RawDataStructure   map[string]interface{} `json:"raw_data_structure"`
	RawDataSample      map[string]interface{} `json:"raw_data_sample"`
	ExtractionAttempts []ConversionAttempt    `json:"extraction_attempts"`
	FieldMappings      map[string]string      `json:"field_mappings"`
	ConversionIssues   []ConversionIssue      `json:"conversion_issues"`
	ConfidenceScore    float64                `json:"confidence_score"`
	Success            bool                   `json:"success"`
	ErrorMessage       string                 `json:"error_message,omitempty"`
}

// ConversionAttempt represents a single attempt to convert data
type ConversionAttempt struct {
	Step        string                 `json:"step"`
	Timestamp   time.Time              `json:"timestamp"`
	Success     bool                   `json:"success"`
	EventsFound int                    `json:"events_found"`
	Details     map[string]interface{} `json:"details"`
	Issues      []string               `json:"issues"`
}

// ConversionIssue represents a conversion problem
type ConversionIssue struct {
	Type       string `json:"type"`        // missing_field|invalid_format|low_confidence|data_quality
	Field      string `json:"field"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
	RawValue   string `json:"raw_value,omitempty"`
	Severity   string `json:"severity"`    // error|warning|info
}

// SchemaConversionService handles conversion from raw extracted data to Activity model
type SchemaConversionService struct{}

// NewSchemaConversionService creates a new schema conversion service
func NewSchemaConversionService() *SchemaConversionService {
	return &SchemaConversionService{}
}

// ConvertToActivity converts raw extracted data to Activity model
func (scs *SchemaConversionService) ConvertToActivity(adminEvent *models.AdminEvent) (*models.ConversionResult, error) {
	startTime := time.Now()
	
	// Initialize conversion diagnostics
	diagnostics := &ConversionDiagnostics{
		AdminEventID:       adminEvent.EventID,
		SourceURL:          adminEvent.SourceURL,
		SchemaType:         adminEvent.SchemaType,
		StartTime:          startTime,
		ExtractionAttempts: []ConversionAttempt{},
		ConversionIssues:   []ConversionIssue{},
		FieldMappings:      make(map[string]string),
		RawDataStructure:   make(map[string]interface{}),
		RawDataSample:      make(map[string]interface{}),
	}

	rawData := adminEvent.RawExtractedData
	var issues []string
	fieldMappings := make(map[string]string)

	log.Printf("[CONVERSION] Starting conversion for AdminEvent %s (Schema: %s, URL: %s)", 
		adminEvent.EventID, adminEvent.SchemaType, adminEvent.SourceURL)

	// Analyze raw data structure
	scs.analyzeRawDataStructure(rawData, diagnostics)

	// Extract events array from raw data
	extractionAttempt := ConversionAttempt{
		Step:      "extractEventsFromRawData",
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	events, err := scs.extractEventsFromRawDataWithDiagnostics(rawData, adminEvent.SchemaType, &extractionAttempt, diagnostics)
	if err != nil {
		diagnostics.EndTime = time.Now()
		diagnostics.ProcessingTime = time.Since(startTime)
		diagnostics.Success = false
		diagnostics.ErrorMessage = fmt.Sprintf("Failed to extract events from raw data: %v", err)
		scs.logConversionDiagnostics(diagnostics)
		return nil, fmt.Errorf("failed to extract events from raw data: %w", err)
	}

	extractionAttempt.Success = len(events) > 0
	extractionAttempt.EventsFound = len(events)
	diagnostics.ExtractionAttempts = append(diagnostics.ExtractionAttempts, extractionAttempt)

	if len(events) == 0 {
		issues = append(issues, "No events found in extracted data")
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "missing_field",
			Field:      "events",
			Message:    "No events found in extracted data",
			Suggestion: "Check if the raw data contains an events array or similar structure",
			Severity:   "error",
		})

		diagnostics.EndTime = time.Now()
		diagnostics.ProcessingTime = time.Since(startTime)
		diagnostics.Success = false
		diagnostics.ConfidenceScore = 0.0
		scs.logConversionDiagnostics(diagnostics)

		return &models.ConversionResult{
			Activity:        nil,
			Issues:          issues,
			FieldMappings:   fieldMappings,
			ConfidenceScore: 0.0,
		}, nil
	}

	log.Printf("[CONVERSION] Found %d events in raw data, converting first event", len(events))

	// For now, convert the first event (later we can handle multiple events)
	firstEvent := events[0]
	
	conversionAttempt := ConversionAttempt{
		Step:      "convertSingleEvent",
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	activity, mappings, conversionIssues := scs.convertSingleEventWithDiagnostics(firstEvent, adminEvent, &conversionAttempt, diagnostics)

	conversionAttempt.Success = activity != nil
	if activity != nil {
		conversionAttempt.EventsFound = 1
	}
	diagnostics.ExtractionAttempts = append(diagnostics.ExtractionAttempts, conversionAttempt)

	// Merge field mappings
	for k, v := range mappings {
		fieldMappings[k] = v
		diagnostics.FieldMappings[k] = v
	}

	issues = append(issues, conversionIssues...)

	// Calculate confidence score
	confidence := scs.calculateConfidenceScore(activity, issues)
	diagnostics.ConfidenceScore = confidence

	// Complete diagnostics
	diagnostics.EndTime = time.Now()
	diagnostics.ProcessingTime = time.Since(startTime)
	diagnostics.Success = activity != nil

	// Log final diagnostics and store for debugging
	scs.logConversionDiagnostics(diagnostics)
	lastConversionDiagnostics = diagnostics

	log.Printf("[CONVERSION] Conversion completed: Success=%t, Confidence=%.1f, Issues=%d", 
		activity != nil, confidence, len(issues))

	return &models.ConversionResult{
		Activity:        activity,
		Issues:          issues,
		FieldMappings:   fieldMappings,
		ConfidenceScore: confidence,
	}, nil
}

// extractEventsFromRawData extracts events array from different schema types (legacy method)
func (scs *SchemaConversionService) extractEventsFromRawData(rawData map[string]interface{}, schemaType string) ([]map[string]interface{}, error) {
	attempt := ConversionAttempt{
		Step:      "extractEventsFromRawData",
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}
	diagnostics := &ConversionDiagnostics{}
	return scs.extractEventsFromRawDataWithDiagnostics(rawData, schemaType, &attempt, diagnostics)
}

// extractEventsFromRawDataWithDiagnostics extracts events array from different schema types with diagnostics
func (scs *SchemaConversionService) extractEventsFromRawDataWithDiagnostics(rawData map[string]interface{}, schemaType string, attempt *ConversionAttempt, diagnostics *ConversionDiagnostics) ([]map[string]interface{}, error) {
	var events []map[string]interface{}

	log.Printf("[CONVERSION] Extracting events from raw data (Schema: %s)", schemaType)

	// Log available keys in raw data
	availableKeys := make([]string, 0, len(rawData))
	for k := range rawData {
		availableKeys = append(availableKeys, k)
	}
	attempt.Details["available_keys"] = availableKeys
	log.Printf("[CONVERSION] Available keys in raw data: %v", availableKeys)

	switch schemaType {
	case "events":
		log.Printf("[CONVERSION] Looking for 'events' array in raw data")
		if eventsArray, ok := rawData["events"].([]interface{}); ok {
			log.Printf("[CONVERSION] Found 'events' array with %d items", len(eventsArray))
			attempt.Details["events_array_length"] = len(eventsArray)
			
			for i, event := range eventsArray {
				if eventMap, ok := event.(map[string]interface{}); ok {
					events = append(events, eventMap)
					log.Printf("[CONVERSION] Successfully parsed event %d", i+1)
				} else {
					attempt.Issues = append(attempt.Issues, fmt.Sprintf("Event %d is not a valid object", i+1))
					log.Printf("[CONVERSION] Event %d is not a valid object: %T", i+1, event)
				}
			}
		} else {
			attempt.Issues = append(attempt.Issues, "No 'events' array found in raw data")
			log.Printf("[CONVERSION] No 'events' array found in raw data")
			
			// Check if events data might be under a different key
			for key, value := range rawData {
				if array, ok := value.([]interface{}); ok && len(array) > 0 {
					log.Printf("[CONVERSION] Found potential events array under key '%s' with %d items", key, len(array))
					attempt.Details["alternative_array_key"] = key
					attempt.Details["alternative_array_length"] = len(array)
				}
			}
		}

	case "activities":
		log.Printf("[CONVERSION] Looking for 'activities' array in raw data")
		if activitiesArray, ok := rawData["activities"].([]interface{}); ok {
			log.Printf("[CONVERSION] Found 'activities' array with %d items", len(activitiesArray))
			attempt.Details["activities_array_length"] = len(activitiesArray)
			
			for i, activity := range activitiesArray {
				if activityMap, ok := activity.(map[string]interface{}); ok {
					events = append(events, activityMap)
					log.Printf("[CONVERSION] Successfully parsed activity %d", i+1)
				} else {
					attempt.Issues = append(attempt.Issues, fmt.Sprintf("Activity %d is not a valid object", i+1))
					log.Printf("[CONVERSION] Activity %d is not a valid object: %T", i+1, activity)
				}
			}
		} else {
			attempt.Issues = append(attempt.Issues, "No 'activities' array found in raw data")
			log.Printf("[CONVERSION] No 'activities' array found in raw data")
		}

	case "venues":
		log.Printf("[CONVERSION] Looking for 'venues' array in raw data")
		if venuesArray, ok := rawData["venues"].([]interface{}); ok {
			log.Printf("[CONVERSION] Found 'venues' array with %d items", len(venuesArray))
			attempt.Details["venues_array_length"] = len(venuesArray)
			
			for i, venue := range venuesArray {
				if venueMap, ok := venue.(map[string]interface{}); ok {
					events = append(events, venueMap)
					log.Printf("[CONVERSION] Successfully parsed venue %d", i+1)
				} else {
					attempt.Issues = append(attempt.Issues, fmt.Sprintf("Venue %d is not a valid object", i+1))
					log.Printf("[CONVERSION] Venue %d is not a valid object: %T", i+1, venue)
				}
			}
		} else {
			attempt.Issues = append(attempt.Issues, "No 'venues' array found in raw data")
			log.Printf("[CONVERSION] No 'venues' array found in raw data")
		}

	case "custom":
		log.Printf("[CONVERSION] Looking for any array in raw data (custom schema)")
		// For custom schemas, try to find any array of objects
		foundArray := false
		for key, value := range rawData {
			if array, ok := value.([]interface{}); ok {
				log.Printf("[CONVERSION] Found array under key '%s' with %d items", key, len(array))
				attempt.Details["custom_array_key"] = key
				attempt.Details["custom_array_length"] = len(array)
				
				for i, item := range array {
					if itemMap, ok := item.(map[string]interface{}); ok {
						events = append(events, itemMap)
						log.Printf("[CONVERSION] Successfully parsed custom item %d", i+1)
					} else {
						attempt.Issues = append(attempt.Issues, fmt.Sprintf("Custom item %d is not a valid object", i+1))
						log.Printf("[CONVERSION] Custom item %d is not a valid object: %T", i+1, item)
					}
				}
				foundArray = true
				break // Use first array found
			}
		}
		
		if !foundArray {
			attempt.Issues = append(attempt.Issues, "No arrays found in raw data for custom schema")
			log.Printf("[CONVERSION] No arrays found in raw data for custom schema")
		}

	default:
		attempt.Issues = append(attempt.Issues, fmt.Sprintf("Unknown schema type: %s", schemaType))
		log.Printf("[CONVERSION] Unknown schema type: %s", schemaType)
	}

	log.Printf("[CONVERSION] Event extraction completed: %d events found", len(events))
	return events, nil
}

// convertSingleEvent converts a single event to Activity model (legacy method)
func (scs *SchemaConversionService) convertSingleEvent(eventData map[string]interface{}, adminEvent *models.AdminEvent) (*models.Activity, map[string]string, []string) {
	attempt := ConversionAttempt{
		Step:      "convertSingleEvent",
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}
	diagnostics := &ConversionDiagnostics{}
	return scs.convertSingleEventWithDiagnostics(eventData, adminEvent, &attempt, diagnostics)
}

// convertSingleEventWithDiagnostics converts a single event to Activity model with diagnostics
func (scs *SchemaConversionService) convertSingleEventWithDiagnostics(eventData map[string]interface{}, adminEvent *models.AdminEvent, attempt *ConversionAttempt, diagnostics *ConversionDiagnostics) (*models.Activity, map[string]string, []string) {
	var issues []string
	fieldMappings := make(map[string]string)

	log.Printf("[CONVERSION] Converting single event to Activity model")

	// Log available fields in event data
	availableFields := make([]string, 0, len(eventData))
	for k := range eventData {
		availableFields = append(availableFields, k)
	}
	attempt.Details["available_fields"] = availableFields
	log.Printf("[CONVERSION] Available fields in event data: %v", availableFields)

	activity := &models.Activity{
		ID:        uuid.New().String(),
		Status:    models.ActivityStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Extract title (required field)
	title := scs.extractStringWithFallbacks(eventData, []string{"title", "name", "event_name", "activity_name"})
	if title == "" {
		issues = append(issues, "Missing title/name - this is required")
		title = "Untitled Event"
	}
	activity.Title = title
	fieldMappings["title"] = scs.findSourceField(eventData, []string{"title", "name", "event_name", "activity_name"})

	// Extract description
	description := scs.extractStringWithFallbacks(eventData, []string{"description", "summary", "details"})
	if description == "" {
		issues = append(issues, "Missing description")
	}
	activity.Description = description
	fieldMappings["description"] = scs.findSourceField(eventData, []string{"description", "summary", "details"})

	// Determine type based on content and schema
	activity.Type = scs.determineActivityType(eventData, adminEvent.SchemaType, title, description)
	fieldMappings["type"] = fmt.Sprintf("auto-detected from schema_type '%s'", adminEvent.SchemaType)

	// Determine category
	activity.Category = scs.determineCategory(title, description)
	fieldMappings["category"] = "auto-classified"

	// Extract and convert schedule
	schedule, scheduleIssues := scs.extractSchedule(eventData)
	activity.Schedule = schedule
	issues = append(issues, scheduleIssues...)
	fieldMappings["schedule"] = scs.findSourceField(eventData, []string{"date", "time", "schedule"})

	// Extract and convert location
	location, locationIssues := scs.extractLocation(eventData, adminEvent.SourceURL)
	activity.Location = location
	issues = append(issues, locationIssues...)
	fieldMappings["location"] = scs.findSourceField(eventData, []string{"location", "venue", "address"})

	// Extract and convert pricing
	pricing, pricingIssues := scs.extractPricing(eventData)
	activity.Pricing = pricing
	issues = append(issues, pricingIssues...)
	fieldMappings["pricing"] = scs.findSourceField(eventData, []string{"price", "cost", "fee", "admission_fee"})

	// Extract age groups
	ageGroups, ageIssues := scs.extractAgeGroups(eventData)
	activity.AgeGroups = ageGroups
	issues = append(issues, ageIssues...)
	fieldMappings["age_groups"] = scs.findSourceField(eventData, []string{"age_groups", "age_suitability", "ages"})

	// Extract registration info
	registration, regIssues := scs.extractRegistration(eventData)
	activity.Registration = registration
	issues = append(issues, regIssues...)
	fieldMappings["registration"] = scs.findSourceField(eventData, []string{"registration_url", "website", "url"})

	// Set provider info
	activity.Provider = models.Provider{
		Name:     scs.extractDomainFromURL(adminEvent.SourceURL),
		Type:     "external",
		Website:  adminEvent.SourceURL,
		Verified: false,
	}

	// Set source info
	activity.Source = models.Source{
		URL:         adminEvent.SourceURL,
		Domain:      scs.extractDomainFromURL(adminEvent.SourceURL),
		ScrapedAt:   adminEvent.ExtractedAt,
		LastChecked: adminEvent.ExtractedAt,
		Reliability: "medium",
	}

	return activity, fieldMappings, issues
}

// extractStringWithFallbacks tries multiple field names to extract a string value
func (scs *SchemaConversionService) extractStringWithFallbacks(data map[string]interface{}, fieldNames []string) string {
	for _, fieldName := range fieldNames {
		if value, ok := data[fieldName]; ok {
			if strValue, ok := value.(string); ok && strValue != "" {
				return strings.TrimSpace(strValue)
			}
		}
	}
	return ""
}

// findSourceField returns the field name that was actually used
func (scs *SchemaConversionService) findSourceField(data map[string]interface{}, fieldNames []string) string {
	for _, fieldName := range fieldNames {
		if value, ok := data[fieldName]; ok {
			if strValue, ok := value.(string); ok && strValue != "" {
				return fieldName
			}
		}
	}
	return "not_found"
}

// determineActivityType determines the activity type based on schema and content
func (scs *SchemaConversionService) determineActivityType(eventData map[string]interface{}, schemaType, title, description string) string {
	// First, use schema type as a hint
	switch schemaType {
	case "events":
		return models.TypeEvent
	case "activities":
		// Check if it's a recurring class or one-time activity
		if scs.containsKeywords(title+description, []string{"class", "classes", "lesson", "course", "weekly", "monthly"}) {
			return models.TypeClass
		}
		if scs.containsKeywords(title+description, []string{"camp", "summer", "day camp", "week"}) {
			return models.TypeCamp
		}
		return models.TypeFreeActivity
	case "venues":
		return models.TypeFreeActivity
	}

	// Content-based classification
	content := strings.ToLower(title + " " + description)

	if scs.containsKeywords(content, []string{"performance", "show", "concert", "play", "theater"}) {
		return models.TypePerformance
	}

	if scs.containsKeywords(content, []string{"class", "lesson", "course", "workshop"}) {
		return models.TypeClass
	}

	if scs.containsKeywords(content, []string{"camp", "summer camp", "day camp"}) {
		return models.TypeCamp
	}

	// Default to event
	return models.TypeEvent
}

// determineCategory determines the category based on content analysis
func (scs *SchemaConversionService) determineCategory(title, description string) string {
	content := strings.ToLower(title + " " + description)

	if scs.containsKeywords(content, []string{"art", "paint", "craft", "music", "dance", "theater", "creative", "drawing"}) {
		return models.CategoryArtsCreativity
	}

	if scs.containsKeywords(content, []string{"sport", "soccer", "basketball", "swim", "run", "bike", "active", "fitness", "martial arts"}) {
		return models.CategoryActiveSports
	}

	if scs.containsKeywords(content, []string{"science", "stem", "math", "engineering", "coding", "robot", "experiment", "tech"}) {
		return models.CategoryEducationalSTEM
	}

	if scs.containsKeywords(content, []string{"performance", "show", "concert", "festival", "movie", "entertainment"}) {
		return models.CategoryEntertainmentEvents
	}

	if scs.containsKeywords(content, []string{"camp", "program", "course", "academy", "school"}) {
		return models.CategoryCampsPrograms
	}

	// Default to free community
	return models.CategoryFreeCommunity
}

// containsKeywords checks if content contains any of the keywords
func (scs *SchemaConversionService) containsKeywords(content string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// extractSchedule extracts and converts schedule information
func (scs *SchemaConversionService) extractSchedule(data map[string]interface{}) (models.Schedule, []string) {
	var issues []string
	schedule := models.Schedule{
		Type:     models.ScheduleTypeOneTime,
		Timezone: "America/Los_Angeles", // Seattle timezone
	}

	// Extract date
	date := scs.extractStringWithFallbacks(data, []string{"date", "start_date", "event_date"})
	if date != "" {
		// Try to parse and format the date
		if formattedDate, err := scs.parseAndFormatDate(date); err == nil {
			schedule.StartDate = formattedDate
		} else {
			schedule.StartDate = date // Keep original if parsing fails
			issues = append(issues, fmt.Sprintf("Could not parse date '%s'", date))
		}
	} else {
		issues = append(issues, "Missing date information")
	}

	// Extract time
	time := scs.extractStringWithFallbacks(data, []string{"time", "start_time", "event_time"})
	if time != "" {
		schedule.StartTime = time
	}

	// Extract duration
	duration := scs.extractStringWithFallbacks(data, []string{"duration", "length"})
	if duration != "" {
		schedule.Duration = duration
	}

	// Check for recurring patterns
	scheduleText := scs.extractStringWithFallbacks(data, []string{"schedule", "frequency", "recurring"})
	if scheduleText != "" {
		if scs.containsKeywords(strings.ToLower(scheduleText), []string{"weekly", "every week", "monday", "tuesday", "wednesday", "thursday", "friday"}) {
			schedule.Type = models.ScheduleTypeRecurring
			schedule.Frequency = "weekly"
		}
	}

	return schedule, issues
}

// parseAndFormatDate attempts to parse various date formats and return YYYY-MM-DD
func (scs *SchemaConversionService) parseAndFormatDate(dateStr string) (string, error) {
	// Common date formats to try
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"1/2/2006",
		"January 2, 2006",
		"Jan 2, 2006",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("could not parse date: %s", dateStr)
}

// extractLocation extracts and converts location information
func (scs *SchemaConversionService) extractLocation(data map[string]interface{}, sourceURL string) (models.Location, []string) {
	var issues []string
	location := models.Location{
		City:         "Seattle", // Default for this system
		State:        "WA",
		Region:       "Seattle Metro",
		VenueType:    models.VenueTypeIndoor, // Default
	}

	// Extract location name
	name := scs.extractStringWithFallbacks(data, []string{"location", "venue", "venue_name", "place"})
	if name == "" {
		issues = append(issues, "Missing location/venue name")
		name = scs.extractDomainFromURL(sourceURL) // Use source domain as fallback
	}
	location.Name = name

	// Extract address
	address := scs.extractStringWithFallbacks(data, []string{"address", "location_address", "venue_address"})
	if address != "" {
		location.Address = address

		// Try to extract city and neighborhood from address
		if city, neighborhood := scs.parseLocationFromAddress(address); city != "" {
			location.City = city
			if neighborhood != "" {
				location.Neighborhood = neighborhood
			}
		}
	} else {
		issues = append(issues, "Missing address information")
	}

	return location, issues
}

// parseLocationFromAddress attempts to extract city and neighborhood from address
func (scs *SchemaConversionService) parseLocationFromAddress(address string) (city, neighborhood string) {
	// Simple parsing for Seattle area locations
	lower := strings.ToLower(address)

	seattleAreas := map[string]string{
		"ballard":        "Ballard",
		"capitol hill":   "Capitol Hill",
		"fremont":        "Fremont",
		"wallingford":    "Wallingford",
		"green lake":     "Green Lake",
		"queen anne":     "Queen Anne",
		"belltown":       "Belltown",
		"university":     "University District",
		"georgetown":     "Georgetown",
		"beacon hill":    "Beacon Hill",
	}

	for area, formal := range seattleAreas {
		if strings.Contains(lower, area) {
			return "Seattle", formal
		}
	}

	// Check for other cities
	if strings.Contains(lower, "bellevue") {
		return "Bellevue", ""
	}
	if strings.Contains(lower, "redmond") {
		return "Redmond", ""
	}
	if strings.Contains(lower, "kirkland") {
		return "Kirkland", ""
	}

	return "Seattle", "" // Default
}

// extractPricing extracts and converts pricing information
func (scs *SchemaConversionService) extractPricing(data map[string]interface{}) (models.Pricing, []string) {
	var issues []string
	pricing := models.Pricing{
		Currency: "USD",
		Unit:     "per-person",
	}

	// Extract price/cost
	priceStr := scs.extractStringWithFallbacks(data, []string{"price", "cost", "fee", "admission_fee"})
	if priceStr == "" {
		issues = append(issues, "Missing pricing information")
		pricing.Type = models.PricingTypeVariable
		pricing.Description = "Contact for pricing"
		return pricing, issues
	}

	// Clean and analyze price string
	priceStr = strings.TrimSpace(strings.ToLower(priceStr))
	pricing.Description = priceStr

	// Check for free
	if scs.containsKeywords(priceStr, []string{"free", "$0", "no cost", "complimentary"}) {
		pricing.Type = models.PricingTypeFree
		pricing.Cost = 0
		pricing.Description = "Free"
		return pricing, issues
	}

	// Check for donation
	if scs.containsKeywords(priceStr, []string{"donation", "suggested", "pay what you can"}) {
		pricing.Type = models.PricingTypeDonation
		pricing.Description = priceStr
		return pricing, issues
	}

	// Try to extract numeric cost
	if cost, err := scs.extractCostFromString(priceStr); err == nil {
		pricing.Type = models.PricingTypePaid
		pricing.Cost = cost
	} else {
		pricing.Type = models.PricingTypeVariable
		issues = append(issues, fmt.Sprintf("Could not parse cost from '%s'", priceStr))
	}

	return pricing, issues
}

// extractCostFromString attempts to extract a numeric cost from a price string
func (scs *SchemaConversionService) extractCostFromString(priceStr string) (float64, error) {
	// Remove common non-numeric characters
	cleaned := strings.ReplaceAll(priceStr, "$", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.TrimSpace(cleaned)

	// Try to parse as float
	if cost, err := strconv.ParseFloat(cleaned, 64); err == nil {
		return cost, nil
	}

	// Try to find first number in string
	parts := strings.Fields(cleaned)
	for _, part := range parts {
		if cost, err := strconv.ParseFloat(part, 64); err == nil {
			return cost, nil
		}
	}

	return 0, fmt.Errorf("no numeric cost found in: %s", priceStr)
}

// extractAgeGroups extracts and converts age group information
func (scs *SchemaConversionService) extractAgeGroups(data map[string]interface{}) ([]models.AgeGroup, []string) {
	var issues []string
	var ageGroups []models.AgeGroup

	// Try to extract age groups array
	if ageGroupsArray, ok := data["age_groups"].([]interface{}); ok {
		for _, ageGroup := range ageGroupsArray {
			if ageGroupStr, ok := ageGroup.(string); ok {
				parsed := scs.parseAgeGroup(ageGroupStr)
				ageGroups = append(ageGroups, parsed)
			}
		}
	} else {
		// Try single age suitability field
		ageSuitability := scs.extractStringWithFallbacks(data, []string{"age_suitability", "ages", "age_range"})
		if ageSuitability != "" {
			parsed := scs.parseAgeGroup(ageSuitability)
			ageGroups = append(ageGroups, parsed)
		}
	}

	// Default if no age groups found
	if len(ageGroups) == 0 {
		issues = append(issues, "No age group information found, defaulting to 'all ages'")
		ageGroups = append(ageGroups, models.AgeGroup{
			Category:    models.AgeGroupAllAges,
			MinAge:      0,
			MaxAge:      99,
			Unit:        "years",
			Description: "All Ages",
		})
	}

	return ageGroups, issues
}

// parseAgeGroup converts a string age group to AgeGroup struct
func (scs *SchemaConversionService) parseAgeGroup(ageGroupStr string) models.AgeGroup {
	ageGroupStr = strings.ToLower(strings.TrimSpace(ageGroupStr))

	// Map common age group strings
	switch {
	case scs.containsKeywords(ageGroupStr, []string{"infant", "baby", "babies"}):
		return models.AgeGroup{
			Category:    models.AgeGroupInfant,
			MinAge:      0,
			MaxAge:      12,
			Unit:        "months",
			Description: "Infants (0-12 months)",
		}
	case scs.containsKeywords(ageGroupStr, []string{"toddler"}):
		return models.AgeGroup{
			Category:    models.AgeGroupToddler,
			MinAge:      1,
			MaxAge:      2,
			Unit:        "years",
			Description: "Toddlers (1-2 years)",
		}
	case scs.containsKeywords(ageGroupStr, []string{"preschool", "pre-k"}):
		return models.AgeGroup{
			Category:    models.AgeGroupPreschool,
			MinAge:      3,
			MaxAge:      5,
			Unit:        "years",
			Description: "Preschoolers (3-5 years)",
		}
	case scs.containsKeywords(ageGroupStr, []string{"elementary", "school-age", "kids"}):
		return models.AgeGroup{
			Category:    models.AgeGroupElementary,
			MinAge:      6,
			MaxAge:      10,
			Unit:        "years",
			Description: "Elementary (6-10 years)",
		}
	case scs.containsKeywords(ageGroupStr, []string{"tween"}):
		return models.AgeGroup{
			Category:    models.AgeGroupTween,
			MinAge:      11,
			MaxAge:      12,
			Unit:        "years",
			Description: "Tweens (11-12 years)",
		}
	case scs.containsKeywords(ageGroupStr, []string{"teen", "teenager"}):
		return models.AgeGroup{
			Category:    models.AgeGroupTeen,
			MinAge:      13,
			MaxAge:      17,
			Unit:        "years",
			Description: "Teens (13-17 years)",
		}
	case scs.containsKeywords(ageGroupStr, []string{"adult"}):
		return models.AgeGroup{
			Category:    models.AgeGroupAdult,
			MinAge:      18,
			MaxAge:      99,
			Unit:        "years",
			Description: "Adults (18+ years)",
		}
	default:
		return models.AgeGroup{
			Category:    models.AgeGroupAllAges,
			MinAge:      0,
			MaxAge:      99,
			Unit:        "years",
			Description: "All Ages",
		}
	}
}

// extractRegistration extracts registration information
func (scs *SchemaConversionService) extractRegistration(data map[string]interface{}) (models.Registration, []string) {
	var issues []string
	registration := models.Registration{
		Required: false,
		Method:   "walk-in",
		Status:   models.RegistrationStatusOpen,
	}

	// Extract registration URL
	regURL := scs.extractStringWithFallbacks(data, []string{"registration_url", "website", "url", "link"})
	if regURL != "" {
		registration.URL = regURL
		registration.Required = true
		registration.Method = "online"
	}

	// Check if registration is required
	if regRequired, ok := data["registration_required"].(bool); ok {
		registration.Required = regRequired
	}

	return registration, issues
}

// calculateConfidenceScore calculates a confidence score for the conversion
func (scs *SchemaConversionService) calculateConfidenceScore(activity *models.Activity, issues []string) float64 {
	score := 100.0

	// Deduct points for missing required fields
	if activity.Title == "" || activity.Title == "Untitled Event" {
		score -= 30
	}
	if activity.Description == "" {
		score -= 10
	}
	if activity.Location.Name == "" {
		score -= 20
	}
	if activity.Schedule.StartDate == "" {
		score -= 15
	}

	// Deduct points for each issue
	score -= float64(len(issues)) * 5

	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}

	return score
}

// extractDomainFromURL extracts domain name from URL
func (scs *SchemaConversionService) extractDomainFromURL(url string) string {
	// Remove protocol
	domain := url
	if strings.HasPrefix(domain, "http://") {
		domain = domain[7:]
	} else if strings.HasPrefix(domain, "https://") {
		domain = domain[8:]
	}

	// Remove path
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}

	// Remove www
	if strings.HasPrefix(domain, "www.") {
		domain = domain[4:]
	}

	return domain
}

// analyzeRawDataStructure analyzes the structure of raw data for diagnostics
func (scs *SchemaConversionService) analyzeRawDataStructure(rawData map[string]interface{}, diagnostics *ConversionDiagnostics) {
	log.Printf("[CONVERSION] Analyzing raw data structure")

	// Capture the structure of the raw data
	for key, value := range rawData {
		switch v := value.(type) {
		case []interface{}:
			diagnostics.RawDataStructure[key] = fmt.Sprintf("array[%d]", len(v))
			log.Printf("[CONVERSION] Field '%s': array with %d items", key, len(v))
			
			// Sample first item if it's an object
			if len(v) > 0 {
				if firstItem, ok := v[0].(map[string]interface{}); ok {
					sampleKey := key + "_sample"
					diagnostics.RawDataSample[sampleKey] = firstItem
					
					// Log fields in first item
					itemFields := make([]string, 0, len(firstItem))
					for k := range firstItem {
						itemFields = append(itemFields, k)
					}
					log.Printf("[CONVERSION] First item in '%s' has fields: %v", key, itemFields)
				}
			}
		case map[string]interface{}:
			diagnostics.RawDataStructure[key] = "object"
			diagnostics.RawDataSample[key] = v
			log.Printf("[CONVERSION] Field '%s': object with %d fields", key, len(v))
		case string:
			diagnostics.RawDataStructure[key] = "string"
			if len(v) > 100 {
				diagnostics.RawDataSample[key] = v[:100] + "..."
			} else {
				diagnostics.RawDataSample[key] = v
			}
			log.Printf("[CONVERSION] Field '%s': string (%d chars)", key, len(v))
		default:
			diagnostics.RawDataStructure[key] = fmt.Sprintf("%T", v)
			diagnostics.RawDataSample[key] = v
			log.Printf("[CONVERSION] Field '%s': %T", key, v)
		}
	}
}

// logConversionDiagnostics logs comprehensive conversion diagnostics
func (scs *SchemaConversionService) logConversionDiagnostics(diagnostics *ConversionDiagnostics) {
	log.Printf("[CONVERSION-DIAGNOSTICS] ========== CONVERSION DIAGNOSTICS ==========")
	log.Printf("[CONVERSION-DIAGNOSTICS] Admin Event ID: %s", diagnostics.AdminEventID)
	log.Printf("[CONVERSION-DIAGNOSTICS] Source URL: %s", diagnostics.SourceURL)
	log.Printf("[CONVERSION-DIAGNOSTICS] Schema Type: %s", diagnostics.SchemaType)
	log.Printf("[CONVERSION-DIAGNOSTICS] Processing Time: %v", diagnostics.ProcessingTime)
	log.Printf("[CONVERSION-DIAGNOSTICS] Success: %t", diagnostics.Success)
	log.Printf("[CONVERSION-DIAGNOSTICS] Confidence Score: %.1f", diagnostics.ConfidenceScore)

	if diagnostics.ErrorMessage != "" {
		log.Printf("[CONVERSION-DIAGNOSTICS] Error: %s", diagnostics.ErrorMessage)
	}

	log.Printf("[CONVERSION-DIAGNOSTICS] Raw Data Structure:")
	for key, structure := range diagnostics.RawDataStructure {
		log.Printf("[CONVERSION-DIAGNOSTICS]   %s: %s", key, structure)
	}

	log.Printf("[CONVERSION-DIAGNOSTICS] Conversion Attempts: %d", len(diagnostics.ExtractionAttempts))
	for i, attempt := range diagnostics.ExtractionAttempts {
		log.Printf("[CONVERSION-DIAGNOSTICS]   Attempt %d: %s - Success: %t, Events: %d", 
			i+1, attempt.Step, attempt.Success, attempt.EventsFound)
		
		if len(attempt.Issues) > 0 {
			log.Printf("[CONVERSION-DIAGNOSTICS]     Issues: %v", attempt.Issues)
		}
		
		if len(attempt.Details) > 0 {
			log.Printf("[CONVERSION-DIAGNOSTICS]     Details: %v", attempt.Details)
		}
	}

	log.Printf("[CONVERSION-DIAGNOSTICS] Field Mappings: %d", len(diagnostics.FieldMappings))
	for field, source := range diagnostics.FieldMappings {
		log.Printf("[CONVERSION-DIAGNOSTICS]   %s -> %s", field, source)
	}

	log.Printf("[CONVERSION-DIAGNOSTICS] Conversion Issues: %d", len(diagnostics.ConversionIssues))
	for i, issue := range diagnostics.ConversionIssues {
		log.Printf("[CONVERSION-DIAGNOSTICS]   Issue %d [%s/%s]: %s - %s", 
			i+1, issue.Severity, issue.Type, issue.Field, issue.Message)
		if issue.Suggestion != "" {
			log.Printf("[CONVERSION-DIAGNOSTICS]     Suggestion: %s", issue.Suggestion)
		}
		if issue.RawValue != "" {
			log.Printf("[CONVERSION-DIAGNOSTICS]     Raw Value: %s", issue.RawValue)
		}
	}

	log.Printf("[CONVERSION-DIAGNOSTICS] ===============================================")
}

// GetConversionDiagnostics returns the last conversion diagnostics (for testing/debugging)
var lastConversionDiagnostics *ConversionDiagnostics

// GetLastConversionDiagnostics returns the diagnostics from the last conversion
func (scs *SchemaConversionService) GetLastConversionDiagnostics() *ConversionDiagnostics {
	return lastConversionDiagnostics
}

// PreviewConversion generates a preview of what the conversion would look like
func (scs *SchemaConversionService) PreviewConversion(adminEvent *models.AdminEvent) (map[string]interface{}, error) {
	result, err := scs.ConvertToActivity(adminEvent)
	if err != nil {
		return nil, err
	}

	preview := map[string]interface{}{
		"activity":         result.Activity,
		"issues":           result.Issues,
		"field_mappings":   result.FieldMappings,
		"confidence_score": result.ConfidenceScore,
		"can_approve":      len(result.Issues) == 0 && result.ConfidenceScore > 50,
	}

	return preview, nil
}