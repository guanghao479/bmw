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
	FieldMappings      map[string]FieldMapping `json:"field_mappings"`
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
	Type       string `json:"type"`        // missing_field|invalid_format|low_confidence|data_quality|validation_error
	Field      string `json:"field"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
	RawValue   string `json:"raw_value,omitempty"`
	Severity   string `json:"severity"`    // error|warning|info
}

// FieldMapping tracks which source field was used for each Activity field
type FieldMapping struct {
	ActivityField string   `json:"activity_field"`    // The field in the Activity model
	SourceField   string   `json:"source_field"`      // The field from raw data that was used
	SourceFields  []string `json:"source_fields"`     // All fields that were attempted
	MappingType   string   `json:"mapping_type"`      // direct|fallback|derived|default
	Confidence    float64  `json:"confidence"`        // 0.0-1.0 confidence in the mapping
	ValidationStatus string `json:"validation_status"` // valid|invalid|warning|not_validated
}

// FieldValidationResult represents the result of validating a field
type FieldValidationResult struct {
	IsValid     bool     `json:"is_valid"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
	Confidence  float64  `json:"confidence"`
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
		FieldMappings:      make(map[string]FieldMapping),
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
		fieldMappings[k] = v.SourceField
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

	// Prepare detailed mappings for the result
	detailedMappings := make(map[string]interface{})
	validationResults := make(map[string]interface{})
	simpleMappings := make(map[string]string)
	
	for field, mapping := range diagnostics.FieldMappings {
		detailedMappings[field] = mapping
		simpleMappings[field] = mapping.SourceField
		validationResults[field] = map[string]interface{}{
			"validation_status": mapping.ValidationStatus,
			"confidence":        mapping.Confidence,
			"mapping_type":      mapping.MappingType,
		}
	}

	return &models.ConversionResult{
		Activity:          activity,
		Issues:            issues,
		FieldMappings:     simpleMappings,
		ConfidenceScore:   confidence,
		DetailedMappings:  detailedMappings,
		ValidationResults: validationResults,
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

// extractEventsFromRawDataWithDiagnostics extracts events array from different schema types with enhanced error reporting
func (scs *SchemaConversionService) extractEventsFromRawDataWithDiagnostics(rawData map[string]interface{}, schemaType string, attempt *ConversionAttempt, diagnostics *ConversionDiagnostics) ([]map[string]interface{}, error) {
	log.Printf("[CONVERSION] Starting enhanced event extraction from raw data (Schema: %s)", schemaType)

	// Validate input parameters
	if rawData == nil {
		err := fmt.Errorf("raw data is nil")
		attempt.Issues = append(attempt.Issues, err.Error())
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "missing_field",
			Field:      "raw_data",
			Message:    "Raw data is nil",
			Suggestion: "Ensure FireCrawl extraction returned valid data",
			Severity:   "error",
		})
		return nil, err
	}

	if len(rawData) == 0 {
		err := fmt.Errorf("raw data is empty")
		attempt.Issues = append(attempt.Issues, err.Error())
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "missing_field",
			Field:      "raw_data",
			Message:    "Raw data contains no fields",
			Suggestion: "Check if FireCrawl extraction was successful",
			Severity:   "error",
		})
		return nil, err
	}

	// Analyze raw data structure in detail
	dataStructure := scs.analyzeDataStructure(rawData)
	attempt.Details["data_structure_analysis"] = dataStructure
	
	// Log available keys in raw data with type information
	availableKeys := make([]string, 0, len(rawData))
	keyTypes := make(map[string]string)
	for k, v := range rawData {
		availableKeys = append(availableKeys, k)
		keyTypes[k] = fmt.Sprintf("%T", v)
	}
	attempt.Details["available_keys"] = availableKeys
	attempt.Details["key_types"] = keyTypes
	
	log.Printf("[CONVERSION] Available keys in raw data: %v", availableKeys)
	log.Printf("[CONVERSION] Key types: %v", keyTypes)

	// Extract events based on schema type with comprehensive error reporting
	switch schemaType {
	case "events":
		events, err := scs.extractEventsArrayWithValidation(rawData, "events", attempt, diagnostics)
		if err != nil {
			return nil, fmt.Errorf("failed to extract events array: %w", err)
		}
		return events, nil

	case "activities":
		events, err := scs.extractEventsArrayWithValidation(rawData, "activities", attempt, diagnostics)
		if err != nil {
			return nil, fmt.Errorf("failed to extract activities array: %w", err)
		}
		return events, nil

	case "venues":
		events, err := scs.extractEventsArrayWithValidation(rawData, "venues", attempt, diagnostics)
		if err != nil {
			return nil, fmt.Errorf("failed to extract venues array: %w", err)
		}
		return events, nil

	case "custom":
		events, err := scs.extractCustomArrayWithValidation(rawData, attempt, diagnostics)
		if err != nil {
			return nil, fmt.Errorf("failed to extract custom array: %w", err)
		}
		return events, nil

	default:
		err := fmt.Errorf("unknown schema type: %s", schemaType)
		attempt.Issues = append(attempt.Issues, err.Error())
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "invalid_format",
			Field:      "schema_type",
			Message:    fmt.Sprintf("Unknown schema type: %s", schemaType),
			Suggestion: "Use one of: events, activities, venues, custom",
			Severity:   "error",
		})
		log.Printf("[CONVERSION] Unknown schema type: %s", schemaType)
		return nil, err
	}
}

// convertSingleEvent converts a single event to Activity model (legacy method)
func (scs *SchemaConversionService) convertSingleEvent(eventData map[string]interface{}, adminEvent *models.AdminEvent) (*models.Activity, map[string]string, []string) {
	attempt := ConversionAttempt{
		Step:      "convertSingleEvent",
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}
	diagnostics := &ConversionDiagnostics{}
	activity, mappings, issues := scs.convertSingleEventWithDiagnostics(eventData, adminEvent, &attempt, diagnostics)
	
	// Convert FieldMapping to simple string mapping for legacy compatibility
	simpleMappings := make(map[string]string)
	for k, v := range mappings {
		simpleMappings[k] = v.SourceField
	}
	
	return activity, simpleMappings, issues
}

// convertSingleEventWithDiagnostics converts a single event to Activity model with diagnostics
func (scs *SchemaConversionService) convertSingleEventWithDiagnostics(eventData map[string]interface{}, adminEvent *models.AdminEvent, attempt *ConversionAttempt, diagnostics *ConversionDiagnostics) (*models.Activity, map[string]FieldMapping, []string) {
	var issues []string
	fieldMappings := make(map[string]FieldMapping)

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

	// Extract title with comprehensive field mapping and validation
	title, titleMapping, titleIssues := scs.extractTitleWithValidation(eventData, attempt, diagnostics)
	activity.Title = title
	fieldMappings["title"] = titleMapping
	diagnostics.FieldMappings["title"] = titleMapping
	issues = append(issues, titleIssues...)

	// Extract description with comprehensive field mapping and validation
	description, descMapping, descIssues := scs.extractDescriptionWithValidation(eventData, attempt, diagnostics)
	activity.Description = description
	fieldMappings["description"] = descMapping
	diagnostics.FieldMappings["description"] = descMapping
	issues = append(issues, descIssues...)

	// Determine type based on content and schema
	activity.Type = scs.determineActivityType(eventData, adminEvent.SchemaType, title, description)
	typeMapping := scs.createFieldMapping("type", "schema_type", []string{"schema_type"}, "derived", adminEvent.SchemaType, FieldValidationResult{IsValid: true, Confidence: 1.0})
	fieldMappings["type"] = typeMapping
	diagnostics.FieldMappings["type"] = typeMapping

	// Determine category
	activity.Category = scs.determineCategory(title, description)
	categoryMapping := scs.createFieldMapping("category", "auto_classified", []string{"title", "description"}, "derived", activity.Category, FieldValidationResult{IsValid: true, Confidence: 0.8})
	fieldMappings["category"] = categoryMapping
	diagnostics.FieldMappings["category"] = categoryMapping

	// Extract and convert schedule with comprehensive validation
	schedule, scheduleMapping, scheduleIssues := scs.extractScheduleWithValidation(eventData, attempt, diagnostics)
	activity.Schedule = schedule
	fieldMappings["schedule"] = scheduleMapping
	diagnostics.FieldMappings["schedule"] = scheduleMapping
	issues = append(issues, scheduleIssues...)

	// Extract and convert location with comprehensive validation
	location, locationMapping, locationIssues := scs.extractLocationWithValidation(eventData, adminEvent.SourceURL, attempt, diagnostics)
	activity.Location = location
	fieldMappings["location"] = locationMapping
	diagnostics.FieldMappings["location"] = locationMapping
	issues = append(issues, locationIssues...)

	// Extract and convert pricing with comprehensive validation
	pricing, pricingMapping, pricingIssues := scs.extractPricingWithValidation(eventData, attempt, diagnostics)
	activity.Pricing = pricing
	fieldMappings["pricing"] = pricingMapping
	diagnostics.FieldMappings["pricing"] = pricingMapping
	issues = append(issues, pricingIssues...)

	// Extract age groups
	ageGroups, ageIssues := scs.extractAgeGroups(eventData)
	activity.AgeGroups = ageGroups
	issues = append(issues, ageIssues...)
	ageGroupsSourceField := scs.findSourceField(eventData, []string{"age_groups", "age_suitability", "ages"})
	ageGroupsMapping := scs.createFieldMapping("age_groups", ageGroupsSourceField, []string{"age_groups", "age_suitability", "ages"}, "direct", ageGroups, FieldValidationResult{IsValid: true, Confidence: 0.7})
	fieldMappings["age_groups"] = ageGroupsMapping
	diagnostics.FieldMappings["age_groups"] = ageGroupsMapping

	// Extract registration info
	registration, regIssues := scs.extractRegistration(eventData)
	activity.Registration = registration
	issues = append(issues, regIssues...)
	registrationSourceField := scs.findSourceField(eventData, []string{"registration_url", "website", "url"})
	registrationMapping := scs.createFieldMapping("registration", registrationSourceField, []string{"registration_url", "website", "url"}, "direct", registration, FieldValidationResult{IsValid: true, Confidence: 0.8})
	fieldMappings["registration"] = registrationMapping
	diagnostics.FieldMappings["registration"] = registrationMapping

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
	for field, mapping := range diagnostics.FieldMappings {
		log.Printf("[CONVERSION-DIAGNOSTICS]   %s -> %s (%s, confidence: %.2f)", field, mapping.SourceField, mapping.MappingType, mapping.Confidence)
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

// analyzeDataStructure provides detailed analysis of raw data structure
func (scs *SchemaConversionService) analyzeDataStructure(rawData map[string]interface{}) map[string]interface{} {
	analysis := make(map[string]interface{})
	
	analysis["total_keys"] = len(rawData)
	
	// Analyze each key
	keyAnalysis := make(map[string]interface{})
	arrayKeys := []string{}
	objectKeys := []string{}
	primitiveKeys := []string{}
	
	for key, value := range rawData {
		keyInfo := make(map[string]interface{})
		keyInfo["type"] = fmt.Sprintf("%T", value)
		
		switch v := value.(type) {
		case []interface{}:
			arrayKeys = append(arrayKeys, key)
			keyInfo["length"] = len(v)
			keyInfo["category"] = "array"
			
			// Analyze array contents
			if len(v) > 0 {
				firstItem := v[0]
				keyInfo["item_type"] = fmt.Sprintf("%T", firstItem)
				
				if itemMap, ok := firstItem.(map[string]interface{}); ok {
					itemKeys := make([]string, 0, len(itemMap))
					for k := range itemMap {
						itemKeys = append(itemKeys, k)
					}
					keyInfo["sample_item_keys"] = itemKeys
				}
			}
			
		case map[string]interface{}:
			objectKeys = append(objectKeys, key)
			keyInfo["keys_count"] = len(v)
			keyInfo["category"] = "object"
			
			// List object keys
			objKeys := make([]string, 0, len(v))
			for k := range v {
				objKeys = append(objKeys, k)
			}
			keyInfo["object_keys"] = objKeys
			
		default:
			primitiveKeys = append(primitiveKeys, key)
			keyInfo["category"] = "primitive"
			
			if str, ok := value.(string); ok {
				keyInfo["length"] = len(str)
			}
		}
		
		keyAnalysis[key] = keyInfo
	}
	
	analysis["key_analysis"] = keyAnalysis
	analysis["array_keys"] = arrayKeys
	analysis["object_keys"] = objectKeys
	analysis["primitive_keys"] = primitiveKeys
	
	return analysis
}

// extractEventsArrayWithValidation extracts and validates an array from raw data
func (scs *SchemaConversionService) extractEventsArrayWithValidation(rawData map[string]interface{}, arrayKey string, attempt *ConversionAttempt, diagnostics *ConversionDiagnostics) ([]map[string]interface{}, error) {
	var events []map[string]interface{}
	
	log.Printf("[CONVERSION] Looking for '%s' array in raw data", arrayKey)
	
	// Check if the expected key exists
	if _, exists := rawData[arrayKey]; !exists {
		// Log what keys are actually available
		availableKeys := make([]string, 0, len(rawData))
		for k := range rawData {
			availableKeys = append(availableKeys, k)
		}
		
		err := fmt.Sprintf("Key '%s' not found in raw data. Available keys: %v", arrayKey, availableKeys)
		attempt.Issues = append(attempt.Issues, err)
		
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "missing_field",
			Field:      arrayKey,
			Message:    fmt.Sprintf("Expected key '%s' not found in raw data", arrayKey),
			Suggestion: fmt.Sprintf("Check if FireCrawl extraction uses different key names. Available: %v", availableKeys),
			Severity:   "error",
		})
		
		// Try to find alternative arrays
		alternatives := scs.findAlternativeArrays(rawData, arrayKey)
		if len(alternatives) > 0 {
			log.Printf("[CONVERSION] Found potential alternative arrays: %v", alternatives)
			attempt.Details["alternative_arrays"] = alternatives
			
			// Use the first alternative
			firstAlt := alternatives[0]
			log.Printf("[CONVERSION] Attempting to use alternative array: %s", firstAlt)
			return scs.extractEventsArrayWithValidation(rawData, firstAlt, attempt, diagnostics)
		}
		
		return nil, fmt.Errorf("no '%s' array found in raw data", arrayKey)
	}
	
	// Validate that the value is actually an array
	arrayValue, ok := rawData[arrayKey].([]interface{})
	if !ok {
		actualType := fmt.Sprintf("%T", rawData[arrayKey])
		err := fmt.Sprintf("Key '%s' is not an array (actual type: %s)", arrayKey, actualType)
		attempt.Issues = append(attempt.Issues, err)
		
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "invalid_format",
			Field:      arrayKey,
			Message:    fmt.Sprintf("Expected array but got %s", actualType),
			Suggestion: "Check FireCrawl extraction schema configuration",
			RawValue:   fmt.Sprintf("%v", rawData[arrayKey]),
			Severity:   "error",
		})
		
		return nil, fmt.Errorf("key '%s' is not an array (type: %s)", arrayKey, actualType)
	}
	
	log.Printf("[CONVERSION] Found '%s' array with %d items", arrayKey, len(arrayValue))
	attempt.Details[arrayKey+"_array_length"] = len(arrayValue)
	
	// Validate array is not empty
	if len(arrayValue) == 0 {
		err := fmt.Sprintf("Array '%s' is empty", arrayKey)
		attempt.Issues = append(attempt.Issues, err)
		
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "missing_field",
			Field:      arrayKey,
			Message:    fmt.Sprintf("Array '%s' contains no items", arrayKey),
			Suggestion: "Check if the source website contains the expected data",
			Severity:   "warning",
		})
		
		return events, nil // Return empty array, not error
	}
	
	// Process each item in the array
	validItems := 0
	invalidItems := 0
	
	for i, item := range arrayValue {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			invalidItems++
			itemType := fmt.Sprintf("%T", item)
			issue := fmt.Sprintf("Item %d in '%s' array is not an object (type: %s)", i+1, arrayKey, itemType)
			attempt.Issues = append(attempt.Issues, issue)
			
			diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
				Type:       "invalid_format",
				Field:      fmt.Sprintf("%s[%d]", arrayKey, i),
				Message:    fmt.Sprintf("Array item is not an object (type: %s)", itemType),
				Suggestion: "Check FireCrawl extraction schema - items should be objects",
				RawValue:   fmt.Sprintf("%v", item),
				Severity:   "warning",
			})
			
			log.Printf("[CONVERSION] Item %d in '%s' array is not a valid object: %T", i+1, arrayKey, item)
			continue
		}
		
		// Validate the object has some content
		if len(itemMap) == 0 {
			invalidItems++
			issue := fmt.Sprintf("Item %d in '%s' array is empty", i+1, arrayKey)
			attempt.Issues = append(attempt.Issues, issue)
			
			diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
				Type:       "missing_field",
				Field:      fmt.Sprintf("%s[%d]", arrayKey, i),
				Message:    "Array item is empty",
				Suggestion: "Check if extraction captured the expected fields",
				Severity:   "warning",
			})
			
			continue
		}
		
		validItems++
		events = append(events, itemMap)
		log.Printf("[CONVERSION] Successfully parsed item %d in '%s' array (%d fields)", i+1, arrayKey, len(itemMap))
		
		// Log sample fields for first item
		if i == 0 {
			itemKeys := make([]string, 0, len(itemMap))
			for k := range itemMap {
				itemKeys = append(itemKeys, k)
			}
			attempt.Details[arrayKey+"_sample_fields"] = itemKeys
			log.Printf("[CONVERSION] Sample fields in '%s' items: %v", arrayKey, itemKeys)
		}
	}
	
	// Log processing summary
	attempt.Details[arrayKey+"_valid_items"] = validItems
	attempt.Details[arrayKey+"_invalid_items"] = invalidItems
	
	log.Printf("[CONVERSION] Array '%s' processing complete: %d valid, %d invalid items", arrayKey, validItems, invalidItems)
	
	if validItems == 0 {
		err := fmt.Sprintf("No valid items found in '%s' array", arrayKey)
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "data_quality",
			Field:      arrayKey,
			Message:    "No valid items in array",
			Suggestion: "Check source data quality and extraction schema",
			Severity:   "error",
		})
		return nil, fmt.Errorf(err)
	}
	
	return events, nil
}

// extractCustomArrayWithValidation extracts arrays from custom schema data
func (scs *SchemaConversionService) extractCustomArrayWithValidation(rawData map[string]interface{}, attempt *ConversionAttempt, diagnostics *ConversionDiagnostics) ([]map[string]interface{}, error) {
	log.Printf("[CONVERSION] Looking for any array in raw data (custom schema)")
	
	// Find all arrays in the data
	foundArrays := make(map[string]int)
	for key, value := range rawData {
		if array, ok := value.([]interface{}); ok {
			foundArrays[key] = len(array)
		}
	}
	
	attempt.Details["found_arrays"] = foundArrays
	
	if len(foundArrays) == 0 {
		err := "No arrays found in raw data for custom schema"
		attempt.Issues = append(attempt.Issues, err)
		
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "missing_field",
			Field:      "arrays",
			Message:    "No arrays found in custom schema data",
			Suggestion: "Check if FireCrawl extraction returned the expected structure",
			Severity:   "error",
		})
		
		return nil, fmt.Errorf(err)
	}
	
	// Use the largest array (most likely to contain the events)
	var bestKey string
	var bestLength int
	for key, length := range foundArrays {
		if length > bestLength {
			bestKey = key
			bestLength = length
		}
	}
	
	log.Printf("[CONVERSION] Using array '%s' with %d items for custom schema", bestKey, bestLength)
	attempt.Details["selected_array"] = bestKey
	attempt.Details["selected_array_length"] = bestLength
	
	// Extract using the selected array
	return scs.extractEventsArrayWithValidation(rawData, bestKey, attempt, diagnostics)
}

// findAlternativeArrays finds potential alternative array keys
func (scs *SchemaConversionService) findAlternativeArrays(rawData map[string]interface{}, expectedKey string) []string {
	var alternatives []string
	
	// Look for arrays that might be alternatives
	for key, value := range rawData {
		if array, ok := value.([]interface{}); ok && len(array) > 0 {
			// Skip the expected key if it exists but is wrong type
			if key == expectedKey {
				continue
			}
			
			// Check if it looks like it could contain events/activities
			if scs.looksLikeEventArray(key, array) {
				alternatives = append(alternatives, key)
			}
		}
	}
	
	return alternatives
}

// looksLikeEventArray determines if an array might contain event data
func (scs *SchemaConversionService) looksLikeEventArray(key string, array []interface{}) bool {
	// Check key name for event-related terms
	lowerKey := strings.ToLower(key)
	eventTerms := []string{"event", "activity", "item", "result", "data", "content"}
	
	for _, term := range eventTerms {
		if strings.Contains(lowerKey, term) {
			return true
		}
	}
	
	// Check if array contains objects (likely event data)
	if len(array) > 0 {
		if _, ok := array[0].(map[string]interface{}); ok {
			return true
		}
	}
	
	return false
}

// Enhanced extraction methods with comprehensive validation

// Helper methods for fallback strategies

// constructTitleFromFields attempts to construct a title from other available fields
func (scs *SchemaConversionService) constructTitleFromFields(eventData map[string]interface{}) string {
	// Try to combine type + location or similar
	eventType := scs.extractStringWithFallbacks(eventData, []string{"type", "category", "kind"})
	location := scs.extractStringWithFallbacks(eventData, []string{"location", "venue"})
	
	if eventType != "" && location != "" {
		return fmt.Sprintf("%s at %s", eventType, location)
	}
	
	if eventType != "" {
		return eventType
	}
	
	if location != "" {
		return fmt.Sprintf("Event at %s", location)
	}
	
	return ""
}

// generateFallbackTitle generates a title when none can be found
func (scs *SchemaConversionService) generateFallbackTitle(eventData map[string]interface{}) string {
	// Use first available string field as basis
	for _, value := range eventData {
		if strValue, ok := value.(string); ok && len(strValue) > 5 && len(strValue) < 100 {
			return fmt.Sprintf("Event: %s", strings.TrimSpace(strValue))
		}
	}
	
	return "Untitled Event"
}

// constructDescriptionFromFields constructs description from available fields
func (scs *SchemaConversionService) constructDescriptionFromFields(eventData map[string]interface{}) string {
	var parts []string
	
	// Collect descriptive fields
	descriptiveFields := []string{"info", "text", "content", "notes", "comments"}
	for _, field := range descriptiveFields {
		if value := scs.extractStringWithFallbacks(eventData, []string{field}); value != "" && len(value) > 10 {
			parts = append(parts, value)
		}
	}
	
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	
	return ""
}

// parseDateTimeString attempts to parse a combined date/time string
func (scs *SchemaConversionService) parseDateTimeString(dateTimeStr string) (date, time string) {
	// Simple parsing - could be enhanced with more sophisticated date parsing
	dateTimeStr = strings.TrimSpace(dateTimeStr)
	
	// Look for time patterns in the string
	timePatterns := []string{"AM", "PM", "am", "pm", ":"}
	hasTime := false
	for _, pattern := range timePatterns {
		if strings.Contains(dateTimeStr, pattern) {
			hasTime = true
			break
		}
	}
	
	if hasTime {
		// Try to split date and time
		parts := strings.Fields(dateTimeStr)
		if len(parts) >= 2 {
			// Assume first part is date, rest is time
			date = parts[0]
			time = strings.Join(parts[1:], " ")
		} else {
			// Whole string might be time
			time = dateTimeStr
		}
	} else {
		// Assume it's just a date
		date = dateTimeStr
	}
	
	return date, time
}

// parseTimeString parses a time string
func (scs *SchemaConversionService) parseTimeString(timeStr string) string {
	// Basic time string cleanup
	timeStr = strings.TrimSpace(timeStr)
	
	// Normalize AM/PM
	timeStr = strings.ReplaceAll(timeStr, "am", "AM")
	timeStr = strings.ReplaceAll(timeStr, "pm", "PM")
	
	return timeStr
}

// parsePricingString parses a pricing string into structured pricing
func (scs *SchemaConversionService) parsePricingString(priceStr string) models.Pricing {
	pricing := models.Pricing{
		Currency: "USD",
		Unit:     "per-person",
	}
	
	lowerPrice := strings.ToLower(strings.TrimSpace(priceStr))
	
	// Check for free
	if strings.Contains(lowerPrice, "free") || strings.Contains(lowerPrice, "no cost") {
		pricing.Type = models.PricingTypeFree
		pricing.Description = "Free"
		pricing.Cost = 0
		return pricing
	}
	
	// Check for donation
	if strings.Contains(lowerPrice, "donation") || strings.Contains(lowerPrice, "suggested") {
		pricing.Type = models.PricingTypeDonation
		pricing.Description = priceStr
		return pricing
	}
	
	// Try to extract numeric cost
	if cost, err := scs.extractCostFromString(priceStr); err == nil {
		pricing.Type = models.PricingTypePaid
		pricing.Cost = cost
		pricing.Description = priceStr
		return pricing
	}
	
	// Default to variable pricing
	pricing.Type = models.PricingTypeVariable
	pricing.Description = priceStr
	return pricing
}

// generateLocationFromURL generates a location name from the source URL
func (scs *SchemaConversionService) generateLocationFromURL(url string) string {
	domain := scs.extractDomainFromURL(url)
	
	// Clean up domain name
	domain = strings.ReplaceAll(domain, "www.", "")
	domain = strings.ReplaceAll(domain, ".com", "")
	domain = strings.ReplaceAll(domain, ".org", "")
	domain = strings.ReplaceAll(domain, "-", " ")
	domain = strings.Title(domain)
	
	return fmt.Sprintf("Venue from %s", domain)
}

// validateDateField validates a date string and provides suggestions
func (scs *SchemaConversionService) validateDateField(dateStr string, fieldName string) FieldValidationResult {
	result := FieldValidationResult{
		IsValid:     false,
		Issues:      []string{},
		Suggestions: []string{},
		Confidence:  0.0,
	}

	if dateStr == "" {
		result.Issues = append(result.Issues, fmt.Sprintf("%s is empty", fieldName))
		result.Suggestions = append(result.Suggestions, "Provide a date in YYYY-MM-DD format")
		return result
	}

	// Try to parse the date
	if _, err := scs.parseAndFormatDate(dateStr); err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Invalid date format: %s", dateStr))
		result.Suggestions = append(result.Suggestions, "Use formats like: YYYY-MM-DD, MM/DD/YYYY, or 'January 1, 2024'")
		result.Confidence = 0.2
		return result
	}

	// Check if date is in the past (for events)
	if parsedDate, err := time.Parse("2006-01-02", dateStr); err == nil {
		if parsedDate.Before(time.Now().AddDate(0, 0, -1)) {
			result.Issues = append(result.Issues, "Date appears to be in the past")
			result.Suggestions = append(result.Suggestions, "Verify this is not an expired event")
			result.Confidence = 0.7
		} else {
			result.Confidence = 1.0
		}
	}

	result.IsValid = len(result.Issues) == 0 || result.Confidence > 0.5
	return result
}

// validateTimeField validates a time string
func (scs *SchemaConversionService) validateTimeField(timeStr string, fieldName string) FieldValidationResult {
	result := FieldValidationResult{
		IsValid:     false,
		Issues:      []string{},
		Suggestions: []string{},
		Confidence:  0.0,
	}

	if timeStr == "" {
		result.Issues = append(result.Issues, fmt.Sprintf("%s is empty", fieldName))
		result.Suggestions = append(result.Suggestions, "Provide time in HH:MM format or '2:00 PM' format")
		return result
	}

	// Basic time format validation
	timeFormats := []string{
		"15:04",      // 24-hour format
		"3:04 PM",    // 12-hour with AM/PM
		"3:04PM",     // 12-hour without space
		"15:04:05",   // with seconds
	}

	validFormat := false
	for _, format := range timeFormats {
		if _, err := time.Parse(format, timeStr); err == nil {
			validFormat = true
			break
		}
	}

	if !validFormat {
		result.Issues = append(result.Issues, fmt.Sprintf("Invalid time format: %s", timeStr))
		result.Suggestions = append(result.Suggestions, "Use formats like: 14:30, 2:30 PM, or 2:30PM")
		result.Confidence = 0.2
		return result
	}

	result.IsValid = true
	result.Confidence = 1.0
	return result
}

// validateLocationField validates location information
func (scs *SchemaConversionService) validateLocationField(location models.Location, fieldName string) FieldValidationResult {
	result := FieldValidationResult{
		IsValid:     false,
		Issues:      []string{},
		Suggestions: []string{},
		Confidence:  0.0,
	}

	score := 0.0
	maxScore := 4.0

	// Check required fields
	if location.Name == "" {
		result.Issues = append(result.Issues, "Location name is missing")
		result.Suggestions = append(result.Suggestions, "Provide a venue name or location description")
	} else {
		score += 1.0
	}

	if location.Address == "" {
		result.Issues = append(result.Issues, "Address is missing")
		result.Suggestions = append(result.Suggestions, "Provide a street address for better discoverability")
	} else {
		score += 1.0
	}

	if location.City == "" {
		result.Issues = append(result.Issues, "City is missing")
		result.Suggestions = append(result.Suggestions, "Specify the city (e.g., Seattle, Bellevue)")
	} else {
		score += 1.0
	}

	if location.Region == "" {
		result.Issues = append(result.Issues, "Region is missing")
		result.Suggestions = append(result.Suggestions, "Specify the region (e.g., Seattle Metro, Eastside)")
	} else {
		score += 1.0
	}

	result.Confidence = score / maxScore
	result.IsValid = result.Confidence > 0.5
	return result
}

// validatePricingField validates pricing information
func (scs *SchemaConversionService) validatePricingField(pricing models.Pricing, fieldName string) FieldValidationResult {
	result := FieldValidationResult{
		IsValid:     false,
		Issues:      []string{},
		Suggestions: []string{},
		Confidence:  0.0,
	}

	// Check pricing type consistency
	switch pricing.Type {
	case models.PricingTypeFree:
		if pricing.Cost != 0 {
			result.Issues = append(result.Issues, "Free pricing type but cost is not zero")
			result.Suggestions = append(result.Suggestions, "Set cost to 0 for free activities")
		}
		result.Confidence = 1.0
	case models.PricingTypePaid:
		if pricing.Cost <= 0 {
			result.Issues = append(result.Issues, "Paid pricing type but cost is zero or negative")
			result.Suggestions = append(result.Suggestions, "Provide a positive cost amount")
		}
		result.Confidence = 0.8
	case models.PricingTypeDonation:
		result.Confidence = 0.9
	case models.PricingTypeVariable:
		if pricing.Description == "" {
			result.Issues = append(result.Issues, "Variable pricing needs description")
			result.Suggestions = append(result.Suggestions, "Provide pricing details in description")
		}
		result.Confidence = 0.7
	default:
		result.Issues = append(result.Issues, fmt.Sprintf("Unknown pricing type: %s", pricing.Type))
		result.Suggestions = append(result.Suggestions, "Use: free, paid, donation, or variable")
		result.Confidence = 0.2
	}

	// Validate currency
	if pricing.Currency == "" {
		result.Issues = append(result.Issues, "Currency is missing")
		result.Suggestions = append(result.Suggestions, "Specify currency (e.g., USD)")
	}

	result.IsValid = len(result.Issues) == 0 && result.Confidence > 0.5
	return result
}

// extractTitleWithValidation extracts title with comprehensive field mapping and validation
func (scs *SchemaConversionService) extractTitleWithValidation(eventData map[string]interface{}, attempt *ConversionAttempt, diagnostics *ConversionDiagnostics) (string, FieldMapping, []string) {
	var issues []string
	attemptedFields := []string{"title", "name", "event_name", "activity_name", "subject", "heading"}
	
	// Try to extract title using fallback strategy
	title := ""
	sourceField := "not_found"
	mappingType := "default"
	
	for _, field := range attemptedFields {
		if value, ok := eventData[field]; ok {
			if strValue, ok := value.(string); ok && strings.TrimSpace(strValue) != "" {
				title = strings.TrimSpace(strValue)
				sourceField = field
				mappingType = "direct"
				break
			}
		}
	}
	
	// Use default if no title found
	if title == "" {
		title = "Untitled Event"
		issues = append(issues, "No title found in source data, using default")
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "missing_field",
			Field:      "title",
			Message:    "No title found in source data",
			Suggestion: "Ensure source data includes a title, name, or heading field",
			Severity:   "warning",
		})
	}
	
	// Validate title
	validation := scs.validateTitleField(title)
	if !validation.IsValid {
		issues = append(issues, validation.Issues...)
		for _, issue := range validation.Issues {
			diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
				Type:       "validation_error",
				Field:      "title",
				Message:    issue,
				Suggestion: strings.Join(validation.Suggestions, "; "),
				RawValue:   title,
				Severity:   "warning",
			})
		}
	}
	
	mapping := scs.createFieldMapping("title", sourceField, attemptedFields, mappingType, title, validation)
	return title, mapping, issues
}

// extractDescriptionWithValidation extracts description with comprehensive field mapping and validation
func (scs *SchemaConversionService) extractDescriptionWithValidation(eventData map[string]interface{}, attempt *ConversionAttempt, diagnostics *ConversionDiagnostics) (string, FieldMapping, []string) {
	var issues []string
	attemptedFields := []string{"description", "details", "summary", "content", "about", "info"}
	
	description := ""
	sourceField := "not_found"
	mappingType := "default"
	
	for _, field := range attemptedFields {
		if value, ok := eventData[field]; ok {
			if strValue, ok := value.(string); ok && strings.TrimSpace(strValue) != "" {
				description = strings.TrimSpace(strValue)
				sourceField = field
				mappingType = "direct"
				break
			}
		}
	}
	
	if description == "" {
		issues = append(issues, "No description found in source data")
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "missing_field",
			Field:      "description",
			Message:    "No description found in source data",
			Suggestion: "Include description, details, or summary field in source data",
			Severity:   "info",
		})
	}
	
	// Validate description
	validation := scs.validateDescriptionField(description)
	if !validation.IsValid {
		issues = append(issues, validation.Issues...)
		for _, issue := range validation.Issues {
			diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
				Type:       "validation_error",
				Field:      "description",
				Message:    issue,
				Suggestion: strings.Join(validation.Suggestions, "; "),
				RawValue:   description,
				Severity:   "info",
			})
		}
	}
	
	mapping := scs.createFieldMapping("description", sourceField, attemptedFields, mappingType, description, validation)
	return description, mapping, issues
}

// extractScheduleWithValidation extracts schedule with comprehensive validation
func (scs *SchemaConversionService) extractScheduleWithValidation(eventData map[string]interface{}, attempt *ConversionAttempt, diagnostics *ConversionDiagnostics) (models.Schedule, FieldMapping, []string) {
	var issues []string
	schedule := models.Schedule{
		Type:     models.ScheduleTypeOneTime,
		Timezone: "America/Los_Angeles",
	}
	
	attemptedFields := []string{"date", "start_date", "event_date", "schedule_date"}
	sourceField := "not_found"
	mappingType := "default"
	
	// Extract date
	dateStr := ""
	for _, field := range attemptedFields {
		if value, ok := eventData[field]; ok {
			if strValue, ok := value.(string); ok && strings.TrimSpace(strValue) != "" {
				dateStr = strings.TrimSpace(strValue)
				sourceField = field
				mappingType = "direct"
				break
			}
		}
	}
	
	if dateStr != "" {
		// Validate and format date
		dateValidation := scs.validateDateField(dateStr, "start_date")
		if dateValidation.IsValid {
			if formattedDate, err := scs.parseAndFormatDate(dateStr); err == nil {
				schedule.StartDate = formattedDate
			} else {
				schedule.StartDate = dateStr
				issues = append(issues, fmt.Sprintf("Could not parse date '%s'", dateStr))
			}
		} else {
			schedule.StartDate = dateStr
			issues = append(issues, dateValidation.Issues...)
			for _, issue := range dateValidation.Issues {
				diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
					Type:       "validation_error",
					Field:      "schedule.start_date",
					Message:    issue,
					Suggestion: strings.Join(dateValidation.Suggestions, "; "),
					RawValue:   dateStr,
					Severity:   "warning",
				})
			}
		}
	} else {
		issues = append(issues, "Missing date information")
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "missing_field",
			Field:      "schedule.start_date",
			Message:    "No date information found",
			Suggestion: "Include date, start_date, or event_date field",
			Severity:   "error",
		})
	}
	
	// Extract time
	timeFields := []string{"time", "start_time", "event_time"}
	timeStr := ""
	for _, field := range timeFields {
		if value, ok := eventData[field]; ok {
			if strValue, ok := value.(string); ok && strings.TrimSpace(strValue) != "" {
				timeStr = strings.TrimSpace(strValue)
				break
			}
		}
	}
	
	if timeStr != "" {
		timeValidation := scs.validateTimeField(timeStr, "start_time")
		if timeValidation.IsValid {
			schedule.StartTime = scs.parseTimeString(timeStr)
		} else {
			schedule.StartTime = timeStr
			issues = append(issues, timeValidation.Issues...)
			for _, issue := range timeValidation.Issues {
				diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
					Type:       "validation_error",
					Field:      "schedule.start_time",
					Message:    issue,
					Suggestion: strings.Join(timeValidation.Suggestions, "; "),
					RawValue:   timeStr,
					Severity:   "warning",
				})
			}
		}
	}
	
	// Extract duration
	duration := scs.extractStringWithFallbacks(eventData, []string{"duration", "length"})
	if duration != "" {
		schedule.Duration = duration
	}
	
	// Validate overall schedule
	scheduleValidation := scs.validateScheduleField(schedule)
	if !scheduleValidation.IsValid {
		issues = append(issues, scheduleValidation.Issues...)
	}
	
	mapping := scs.createFieldMapping("schedule", sourceField, attemptedFields, mappingType, schedule, scheduleValidation)
	return schedule, mapping, issues
}

// extractLocationWithValidation extracts location with comprehensive validation
func (scs *SchemaConversionService) extractLocationWithValidation(eventData map[string]interface{}, sourceURL string, attempt *ConversionAttempt, diagnostics *ConversionDiagnostics) (models.Location, FieldMapping, []string) {
	var issues []string
	location := models.Location{
		City:      "Seattle",
		State:     "WA",
		Region:    "Seattle Metro",
		VenueType: models.VenueTypeIndoor,
	}
	
	attemptedFields := []string{"location", "venue", "venue_name", "place"}
	sourceField := "not_found"
	mappingType := "default"
	
	// Extract location name
	name := ""
	for _, field := range attemptedFields {
		if value, ok := eventData[field]; ok {
			if strValue, ok := value.(string); ok && strings.TrimSpace(strValue) != "" {
				name = strings.TrimSpace(strValue)
				sourceField = field
				mappingType = "direct"
				break
			}
		}
	}
	
	if name == "" {
		name = scs.generateLocationFromURL(sourceURL)
		mappingType = "derived"
		issues = append(issues, "No location name found, generated from source URL")
	}
	location.Name = name
	
	// Extract address
	addressFields := []string{"address", "location_address", "venue_address"}
	address := ""
	for _, field := range addressFields {
		if value, ok := eventData[field]; ok {
			if strValue, ok := value.(string); ok && strings.TrimSpace(strValue) != "" {
				address = strings.TrimSpace(strValue)
				break
			}
		}
	}
	
	if address != "" {
		location.Address = address
		if city, neighborhood := scs.parseLocationFromAddress(address); city != "" {
			location.City = city
			if neighborhood != "" {
				location.Neighborhood = neighborhood
			}
		}
	} else {
		issues = append(issues, "Missing address information")
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "missing_field",
			Field:      "location.address",
			Message:    "No address information found",
			Suggestion: "Include address, location_address, or venue_address field",
			Severity:   "warning",
		})
	}
	
	// Validate location
	locationValidation := scs.validateLocationField(location, "location")
	if !locationValidation.IsValid {
		issues = append(issues, locationValidation.Issues...)
		for _, issue := range locationValidation.Issues {
			diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
				Type:       "validation_error",
				Field:      "location",
				Message:    issue,
				Suggestion: strings.Join(locationValidation.Suggestions, "; "),
				Severity:   "warning",
			})
		}
	}
	
	mapping := scs.createFieldMapping("location", sourceField, attemptedFields, mappingType, location, locationValidation)
	return location, mapping, issues
}

// extractPricingWithValidation extracts pricing with comprehensive validation
func (scs *SchemaConversionService) extractPricingWithValidation(eventData map[string]interface{}, attempt *ConversionAttempt, diagnostics *ConversionDiagnostics) (models.Pricing, FieldMapping, []string) {
	var issues []string
	pricing := models.Pricing{
		Currency: "USD",
		Unit:     "per-person",
	}
	
	attemptedFields := []string{"price", "cost", "fee", "admission_fee", "pricing"}
	sourceField := "not_found"
	mappingType := "default"
	
	// Extract price/cost
	priceStr := ""
	for _, field := range attemptedFields {
		if value, ok := eventData[field]; ok {
			if strValue, ok := value.(string); ok && strings.TrimSpace(strValue) != "" {
				priceStr = strings.TrimSpace(strValue)
				sourceField = field
				mappingType = "direct"
				break
			}
		}
	}
	
	if priceStr == "" {
		issues = append(issues, "Missing pricing information")
		pricing.Type = models.PricingTypeVariable
		pricing.Description = "Contact for pricing"
		diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
			Type:       "missing_field",
			Field:      "pricing",
			Message:    "No pricing information found",
			Suggestion: "Include price, cost, or fee field in source data",
			Severity:   "info",
		})
	} else {
		// Parse pricing string
		pricing = scs.parsePricingString(priceStr)
	}
	
	// Validate pricing
	pricingValidation := scs.validatePricingField(pricing, "pricing")
	if !pricingValidation.IsValid {
		issues = append(issues, pricingValidation.Issues...)
		for _, issue := range pricingValidation.Issues {
			diagnostics.ConversionIssues = append(diagnostics.ConversionIssues, ConversionIssue{
				Type:       "validation_error",
				Field:      "pricing",
				Message:    issue,
				Suggestion: strings.Join(pricingValidation.Suggestions, "; "),
				RawValue:   priceStr,
				Severity:   "warning",
			})
		}
	}
	
	mapping := scs.createFieldMapping("pricing", sourceField, attemptedFields, mappingType, pricing, pricingValidation)
	return pricing, mapping, issues
}

// Additional validation functions
func (scs *SchemaConversionService) validateTitleField(title string) FieldValidationResult {
	result := FieldValidationResult{
		IsValid:     false,
		Issues:      []string{},
		Suggestions: []string{},
		Confidence:  0.0,
	}
	
	if title == "" || title == "Untitled Event" {
		result.Issues = append(result.Issues, "Title is missing or using default value")
		result.Suggestions = append(result.Suggestions, "Provide a descriptive title for the activity")
		result.Confidence = 0.1
		return result
	}
	
	if len(title) < 5 {
		result.Issues = append(result.Issues, "Title is very short")
		result.Suggestions = append(result.Suggestions, "Consider a more descriptive title")
		result.Confidence = 0.6
	} else if len(title) > 100 {
		result.Issues = append(result.Issues, "Title is very long")
		result.Suggestions = append(result.Suggestions, "Consider shortening the title")
		result.Confidence = 0.8
	} else {
		result.Confidence = 1.0
	}
	
	result.IsValid = result.Confidence > 0.5
	return result
}

func (scs *SchemaConversionService) validateDescriptionField(description string) FieldValidationResult {
	result := FieldValidationResult{
		IsValid:     true,
		Issues:      []string{},
		Suggestions: []string{},
		Confidence:  0.7, // Default confidence for optional field
	}
	
	if description == "" {
		result.Issues = append(result.Issues, "Description is empty")
		result.Suggestions = append(result.Suggestions, "Add a description to help families understand the activity")
		result.Confidence = 0.5
		return result
	}
	
	if len(description) < 20 {
		result.Issues = append(result.Issues, "Description is very short")
		result.Suggestions = append(result.Suggestions, "Consider adding more details about the activity")
		result.Confidence = 0.7
	} else {
		result.Confidence = 1.0
	}
	
	return result
}

func (scs *SchemaConversionService) validateScheduleField(schedule models.Schedule) FieldValidationResult {
	result := FieldValidationResult{
		IsValid:     false,
		Issues:      []string{},
		Suggestions: []string{},
		Confidence:  0.0,
	}
	
	score := 0.0
	maxScore := 3.0
	
	// Check start date
	if schedule.StartDate != "" {
		score += 1.0
	} else {
		result.Issues = append(result.Issues, "Start date is missing")
		result.Suggestions = append(result.Suggestions, "Provide a start date for the activity")
	}
	
	// Check time information
	if schedule.StartTime != "" {
		score += 1.0
	} else {
		result.Issues = append(result.Issues, "Start time is missing")
		result.Suggestions = append(result.Suggestions, "Provide a start time if applicable")
	}
	
	// Check schedule type
	if schedule.Type != "" {
		score += 1.0
	}
	
	result.Confidence = score / maxScore
	result.IsValid = result.Confidence > 0.3 // Lower threshold since time might be optional
	return result
}

// createFieldMapping creates a comprehensive field mapping record
func (scs *SchemaConversionService) createFieldMapping(activityField string, sourceField string, attemptedFields []string, mappingType string, rawValue interface{}, validationResult FieldValidationResult) FieldMapping {
	confidence := 0.5 // default confidence

	// Adjust confidence based on mapping type
	switch mappingType {
	case "direct":
		confidence = 0.9
	case "fallback":
		confidence = 0.7
	case "derived":
		confidence = 0.6
	case "default":
		confidence = 0.3
	}

	// Adjust confidence based on validation
	if validationResult.IsValid {
		confidence = confidence * validationResult.Confidence
	} else {
		confidence = confidence * 0.5
	}

	validationStatus := "not_validated"
	if validationResult.Confidence > 0 {
		if validationResult.IsValid {
			validationStatus = "valid"
		} else if validationResult.Confidence > 0.5 {
			validationStatus = "warning"
		} else {
			validationStatus = "invalid"
		}
	}

	return FieldMapping{
		ActivityField:    activityField,
		SourceField:      sourceField,
		SourceFields:     attemptedFields,
		MappingType:      mappingType,
		Confidence:       confidence,
		ValidationStatus: validationStatus,
	}
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