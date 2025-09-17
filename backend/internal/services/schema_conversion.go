package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"seattle-family-activities-scraper/internal/models"
)

// SchemaConversionService handles conversion from raw extracted data to Activity model
type SchemaConversionService struct{}

// NewSchemaConversionService creates a new schema conversion service
func NewSchemaConversionService() *SchemaConversionService {
	return &SchemaConversionService{}
}

// ConvertToActivity converts raw extracted data to Activity model
func (scs *SchemaConversionService) ConvertToActivity(adminEvent *models.AdminEvent) (*models.ConversionResult, error) {
	rawData := adminEvent.RawExtractedData
	var issues []string
	fieldMappings := make(map[string]string)

	// Extract events array from raw data
	events, err := scs.extractEventsFromRawData(rawData, adminEvent.SchemaType)
	if err != nil {
		return nil, fmt.Errorf("failed to extract events from raw data: %w", err)
	}

	if len(events) == 0 {
		issues = append(issues, "No events found in extracted data")
		return &models.ConversionResult{
			Activity:        nil,
			Issues:          issues,
			FieldMappings:   fieldMappings,
			ConfidenceScore: 0.0,
		}, nil
	}

	// For now, convert the first event (later we can handle multiple events)
	firstEvent := events[0]
	activity, mappings, conversionIssues := scs.convertSingleEvent(firstEvent, adminEvent)

	// Merge field mappings
	for k, v := range mappings {
		fieldMappings[k] = v
	}

	issues = append(issues, conversionIssues...)

	// Calculate confidence score
	confidence := scs.calculateConfidenceScore(activity, issues)

	return &models.ConversionResult{
		Activity:        activity,
		Issues:          issues,
		FieldMappings:   fieldMappings,
		ConfidenceScore: confidence,
	}, nil
}

// extractEventsFromRawData extracts events array from different schema types
func (scs *SchemaConversionService) extractEventsFromRawData(rawData map[string]interface{}, schemaType string) ([]map[string]interface{}, error) {
	var events []map[string]interface{}

	switch schemaType {
	case "events":
		if eventsArray, ok := rawData["events"].([]interface{}); ok {
			for _, event := range eventsArray {
				if eventMap, ok := event.(map[string]interface{}); ok {
					events = append(events, eventMap)
				}
			}
		}
	case "activities":
		if activitiesArray, ok := rawData["activities"].([]interface{}); ok {
			for _, activity := range activitiesArray {
				if activityMap, ok := activity.(map[string]interface{}); ok {
					events = append(events, activityMap)
				}
			}
		}
	case "venues":
		if venuesArray, ok := rawData["venues"].([]interface{}); ok {
			for _, venue := range venuesArray {
				if venueMap, ok := venue.(map[string]interface{}); ok {
					events = append(events, venueMap)
				}
			}
		}
	case "custom":
		// For custom schemas, try to find any array of objects
		for _, value := range rawData {
			if array, ok := value.([]interface{}); ok {
				for _, item := range array {
					if itemMap, ok := item.(map[string]interface{}); ok {
						events = append(events, itemMap)
					}
				}
				break // Use first array found
			}
		}
	}

	return events, nil
}

// convertSingleEvent converts a single event to Activity model
func (scs *SchemaConversionService) convertSingleEvent(eventData map[string]interface{}, adminEvent *models.AdminEvent) (*models.Activity, map[string]string, []string) {
	var issues []string
	fieldMappings := make(map[string]string)

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