package services

import (
	"testing"
	"time"

	"seattle-family-activities-scraper/internal/models"
)

// TestEventDataValidation tests the validation of extracted event data
func TestEventDataValidation(t *testing.T) {
	fc := &FireCrawlClient{}

	// Test valid event data
	validEvent := EventData{
		Title:       "Test Event",
		Description: "This is a test event with sufficient description",
		Date:        "12/25/2024",
		Time:        "10:00 AM",
		Location:    "Test Location",
		Price:       "Free",
		AgeGroups:   []string{"all-ages"},
	}

	result := fc.validateEventData(validEvent)
	if !result.IsValid {
		t.Errorf("Expected valid event to pass validation, got issues: %v", result.Issues)
	}
	if result.ConfidenceScore < 70 {
		t.Errorf("Expected high confidence score for valid event, got: %.1f", result.ConfidenceScore)
	}
	t.Logf("Valid event validation: Confidence=%.1f, Warnings=%d", result.ConfidenceScore, len(result.Warnings))

	// Test invalid event data (missing title)
	invalidEvent := EventData{
		Title:       "", // Missing required field
		Description: "Event without title",
		Date:        "invalid-date",
		Time:        "25:99", // Invalid time
		Location:    "",      // Missing location
	}

	result = fc.validateEventData(invalidEvent)
	if result.IsValid {
		t.Error("Expected invalid event to fail validation")
	}
	if len(result.Issues) == 0 {
		t.Error("Expected validation issues for invalid event")
	}
	t.Logf("Invalid event validation: Valid=%t, Issues=%d, Confidence=%.1f", 
		result.IsValid, len(result.Issues), result.ConfidenceScore)

	// Test event with warnings but still valid
	warningEvent := EventData{
		Title:       "Short", // Very short title
		Description: "Short",  // Very short description
		Date:        "",       // Missing date
		Time:        "",       // Missing time
		Location:    "Loc",    // Very short location
	}

	result = fc.validateEventData(warningEvent)
	if !result.IsValid {
		t.Error("Expected event with warnings to still be valid")
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for low-quality event data")
	}
	if result.ConfidenceScore > 50 {
		t.Errorf("Expected low confidence score for warning event, got: %.1f", result.ConfidenceScore)
	}
	t.Logf("Warning event validation: Confidence=%.1f, Warnings=%d", result.ConfidenceScore, len(result.Warnings))
}

// TestActivityDataValidation tests the validation of converted Activity data
func TestActivityDataValidation(t *testing.T) {
	fc := &FireCrawlClient{}

	// Test valid activity
	validActivity := models.Activity{
		Title:       "Test Activity",
		Description: "Test activity description",
		Type:        models.TypeEvent,
		Category:    models.CategoryFreeCommunity,
		Schedule: models.Schedule{
			StartDate: "2024-12-25",
			StartTime: "10:00 AM",
			Type:      models.ScheduleTypeOneTime,
		},
		Location: models.Location{
			Name: "Test Location",
			City: "Seattle",
		},
		Pricing: models.Pricing{
			Type: models.PricingTypeFree,
		},
		AgeGroups: []models.AgeGroup{
			{Category: models.AgeGroupAllAges},
		},
		Source: models.Source{
			URL:    "https://test.com",
			Domain: "test.com",
		},
	}

	result := fc.validateActivityData(validActivity)
	if !result.IsValid {
		t.Errorf("Expected valid activity to pass validation, got issues: %v", result.Issues)
	}
	if result.ConfidenceScore < 80 {
		t.Errorf("Expected high confidence score for valid activity, got: %.1f", result.ConfidenceScore)
	}
	t.Logf("Valid activity validation: Confidence=%.1f, Warnings=%d", result.ConfidenceScore, len(result.Warnings))

	// Test invalid activity (missing required fields)
	invalidActivity := models.Activity{
		Title: "", // Missing title
		Location: models.Location{
			Name: "", // Missing location name
		},
	}

	result = fc.validateActivityData(invalidActivity)
	if result.IsValid {
		t.Error("Expected invalid activity to fail validation")
	}
	if len(result.Issues) == 0 {
		t.Error("Expected validation issues for invalid activity")
	}
	t.Logf("Invalid activity validation: Valid=%t, Issues=%d, Confidence=%.1f", 
		result.IsValid, len(result.Issues), result.ConfidenceScore)
}

// TestDateTimeValidation tests date and time format validation
func TestDateTimeValidation(t *testing.T) {
	fc := &FireCrawlClient{}

	// Test valid date formats
	validDates := []string{
		"12/25/2024",
		"12-25-2024",
		"2024-12-25",
		"December 25, 2024",
		"Dec 25",
	}

	for _, date := range validDates {
		if !fc.isValidDateFormat(date) {
			t.Errorf("Expected '%s' to be a valid date format", date)
		}
	}

	// Test invalid date formats
	invalidDates := []string{
		"invalid-date",
		"25/12/2024", // Ambiguous format
		"2024/25/12", // Invalid day
		"",
	}

	for _, date := range invalidDates {
		if fc.isValidDateFormat(date) {
			t.Errorf("Expected '%s' to be an invalid date format", date)
		}
	}

	// Test valid time formats
	validTimes := []string{
		"10:00 AM",
		"2:30 PM",
		"14:30",
		"9 AM",
		"10:00 - 11:00 AM",
	}

	for _, time := range validTimes {
		if !fc.isValidTimeFormat(time) {
			t.Errorf("Expected '%s' to be a valid time format", time)
		}
	}

	// Test invalid time formats
	invalidTimes := []string{
		"25:99",
		"invalid-time",
		"",
		"10:99 AM",
	}

	for _, time := range invalidTimes {
		if fc.isValidTimeFormat(time) {
			t.Errorf("Expected '%s' to be an invalid time format", time)
		}
	}
}

// TestValidationIntegration tests the integration of validation with extraction
func TestValidationIntegration(t *testing.T) {
	fc := &FireCrawlClient{}

	// Create a test extraction attempt
	attempt := &ExtractionAttempt{
		Method:    "test",
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
		Issues:    []string{},
	}

	// Test markdown with valid event structure
	validMarkdown := `
# Family Fun Day
Join us for a fun family event!

Date: December 25, 2024
Time: 10:00 AM - 2:00 PM
Location: Seattle Community Center
Price: Free
Ages: All ages welcome

Come enjoy games, activities, and refreshments for the whole family.
`

	events := fc.extractEventsFromMarkdown(validMarkdown, attempt)
	if len(events) == 0 {
		t.Error("Expected to extract at least one event from valid markdown")
	}

	// Validate the first extracted event
	if len(events) > 0 {
		result := fc.validateEventData(events[0])
		if !result.IsValid {
			t.Errorf("Expected extracted event to be valid, got issues: %v", result.Issues)
		}
		t.Logf("Extracted event validation: Title='%s', Confidence=%.1f", 
			events[0].Title, result.ConfidenceScore)
	}

	// Test markdown with poor structure
	poorMarkdown := `
Some random text without clear event structure.
Maybe there's an event somewhere but it's not clear.
No dates, times, or locations mentioned.
`

	events = fc.extractEventsFromMarkdown(poorMarkdown, attempt)
	t.Logf("Poor markdown extraction found %d events", len(events))

	// Even if events are extracted, they should have low confidence
	for i, event := range events {
		result := fc.validateEventData(event)
		t.Logf("Poor event %d validation: Valid=%t, Confidence=%.1f, Warnings=%d", 
			i+1, result.IsValid, result.ConfidenceScore, len(result.Warnings))
	}
}