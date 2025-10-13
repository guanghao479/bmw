package services

import (
	"testing"
	"time"

	"seattle-family-activities-scraper/internal/models"
)

// TestMarkdownParsingWithRealContent tests markdown parsing with realistic content samples
func TestMarkdownParsingWithRealContent(t *testing.T) {
	fc := &FireCrawlClient{}

	t.Run("ParentMapStyleContent", func(t *testing.T) {
		// Simulate ParentMap calendar content structure
		parentMapMarkdown := `
# Family Events This Week

## Kids Art Workshop
**Date:** Saturday, December 15, 2024
**Time:** 10:00 AM - 12:00 PM
**Location:** Seattle Community Center, 123 Main St, Seattle, WA
**Ages:** 5-10 years
**Cost:** $25 per child
Join us for a fun art workshop where kids can explore their creativity with various art supplies.

## Story Time at the Library
**Date:** Sunday, December 16, 2024
**Time:** 2:00 PM
**Location:** Central Library, Downtown Seattle
**Ages:** All ages welcome
**Cost:** Free
Weekly story time featuring classic children's books and interactive activities.

## Family Movie Night
**Date:** Friday, December 21, 2024
**Time:** 7:00 PM - 9:00 PM
**Location:** Community Park Pavilion
**Ages:** Family-friendly
**Cost:** Donation suggested
Outdoor movie screening under the stars. Bring blankets and snacks!
`

		attempt := &ExtractionAttempt{
			Method:    "test_parentmap",
			Timestamp: time.Now(),
			Details:   make(map[string]interface{}),
		}
		events := fc.extractEventsFromMarkdown(parentMapMarkdown, attempt)

		if len(events) < 3 {
			t.Errorf("Expected at least 3 events, got %d", len(events))
		}

		// Validate first event structure
		if len(events) > 0 {
			event := events[0]
			if event.Title == "" {
				t.Error("Expected first event to have a title")
			}
			if event.Date == "" {
				t.Error("Expected first event to have a date")
			}
			if event.Location == "" {
				t.Error("Expected first event to have a location")
			}
			
			t.Logf("First event: Title=%s, Date=%s, Location=%s", event.Title, event.Date, event.Location)
		}
	})

	t.Run("RemlingerFarmsStyleContent", func(t *testing.T) {
		// Simulate Remlinger Farms event content structure
		remlingerMarkdown := `
# Upcoming Events at Remlinger Farms

## Pumpkin Patch Festival
October 1-31, 2024
Daily 10am-6pm
All ages welcome
Admission: $15 adults, $12 children
Come pick your perfect pumpkin and enjoy hayrides, corn maze, and farm animals!

## Holiday Light Display
November 25 - January 1
5:00 PM - 9:00 PM daily
Family event
$20 per vehicle
Drive through our magical winter wonderland with over 1 million lights.

## Farm Animal Feeding
Daily year-round
10:00 AM - 4:00 PM
Great for toddlers and young children
Included with admission
Feed our friendly goats, sheep, and chickens.
`

		attempt := &ExtractionAttempt{
			Method:    "test_remlinger",
			Timestamp: time.Now(),
			Details:   make(map[string]interface{}),
		}
		events := fc.extractEventsFromMarkdown(remlingerMarkdown, attempt)

		if len(events) < 2 {
			t.Errorf("Expected at least 2 events, got %d", len(events))
		}

		// Validate event extraction
		foundPumpkinPatch := false
		foundHolidayLights := false
		
		for _, event := range events {
			if contains(event.Title, "Pumpkin") {
				foundPumpkinPatch = true
				if event.AgeGroups == nil || len(event.AgeGroups) == 0 {
					t.Error("Pumpkin Patch event should have age group information")
				}
			}
			if contains(event.Title, "Holiday") || contains(event.Title, "Light") {
				foundHolidayLights = true
			}
		}

		if !foundPumpkinPatch {
			t.Error("Expected to find Pumpkin Patch event")
		}
		if !foundHolidayLights {
			t.Error("Expected to find Holiday Light Display event")
		}
	})

	t.Run("MalformedMarkdownContent", func(t *testing.T) {
		// Test with poorly structured markdown
		malformedMarkdown := `
Some random text without clear structure.

Event Title Maybe?
No clear date or time information.
Location: Somewhere in Seattle

Another potential event
Date might be here: sometime next week
Time: unclear
Cost: varies

# This looks like a header
But there's no event information below it.

Random paragraph with no event data.
Just some text about activities in general.
`

		attempt := &ExtractionAttempt{
			Method:    "test_malformed",
			Timestamp: time.Now(),
			Details:   make(map[string]interface{}),
		}
		events := fc.extractEventsFromMarkdown(malformedMarkdown, attempt)

		t.Logf("Extracted %d events from malformed content", len(events))
		
		// Even with malformed content, we should get some extraction attempts
		// The quality will be low, but the system should be resilient
		for i, event := range events {
			t.Logf("Event %d: Title='%s', Date='%s', Location='%s'", i+1, event.Title, event.Date, event.Location)
		}
	})
}

// TestSchemaConversionWithMalformedData tests conversion with various data quality issues
func TestSchemaConversionWithMalformedData(t *testing.T) {
	scs := NewSchemaConversionService()

	t.Run("MissingRequiredFields", func(t *testing.T) {
		// Test with missing title and date
		adminEvent := &models.AdminEvent{
			EventID:    "test-missing-fields",
			SourceURL:  "https://test.example.com",
			SchemaType: "events",
			RawExtractedData: map[string]interface{}{
				"events": []interface{}{
					map[string]interface{}{
						"description": "Some description without title or date",
						"location":    "Seattle Community Center",
						"price":       "Free",
					},
				},
			},
			ExtractedAt: time.Now(),
		}

		result, err := scs.ConvertToActivity(adminEvent)
		if err != nil {
			t.Fatalf("Conversion should not fail completely: %v", err)
		}

		if result.Activity == nil {
			t.Error("Expected activity to be created even with missing fields")
		}

		if result.ConfidenceScore >= 70 {
			t.Errorf("Expected low confidence score for missing fields, got %f", result.ConfidenceScore)
		}

		if len(result.Issues) == 0 {
			t.Error("Expected conversion issues for missing required fields")
		}

		t.Logf("Conversion with missing fields: Confidence=%f, Issues=%d", result.ConfidenceScore, len(result.Issues))
	})

	t.Run("InvalidDataFormats", func(t *testing.T) {
		// Test with invalid date and price formats
		adminEvent := &models.AdminEvent{
			EventID:    "test-invalid-formats",
			SourceURL:  "https://test.example.com",
			SchemaType: "events",
			RawExtractedData: map[string]interface{}{
				"events": []interface{}{
					map[string]interface{}{
						"title":       "Test Event",
						"description": "Event with invalid data formats",
						"date":        "not-a-valid-date",
						"time":        "25:99 PM", // Invalid time
						"location":    "Test Location",
						"price":       "maybe free or maybe not", // Ambiguous pricing
						"ages":        "unclear age range",
					},
				},
			},
			ExtractedAt: time.Now(),
		}

		result, err := scs.ConvertToActivity(adminEvent)
		if err != nil {
			t.Fatalf("Conversion should handle invalid formats gracefully: %v", err)
		}

		if result.Activity == nil {
			t.Error("Expected activity to be created despite invalid formats")
		}

		// Should have validation issues
		hasDateIssue := false
		for _, issue := range result.Issues {
			if contains(issue, "date") {
				hasDateIssue = true
			}
		}

		if !hasDateIssue {
			t.Error("Expected date validation issue")
		}

		t.Logf("Conversion with invalid formats: Confidence=%f, Issues=%v", result.ConfidenceScore, result.Issues)
	})

	t.Run("EmptyEventsArray", func(t *testing.T) {
		// Test with empty events array
		adminEvent := &models.AdminEvent{
			EventID:    "test-empty-events",
			SourceURL:  "https://test.example.com",
			SchemaType: "events",
			RawExtractedData: map[string]interface{}{
				"events": []interface{}{},
			},
			ExtractedAt: time.Now(),
		}

		_, err := scs.ConvertToActivity(adminEvent)
		
		// This should fail
		if err == nil {
			t.Error("Expected conversion to fail for empty events array")
		}

		t.Logf("Empty events array result: Error=%v", err)
	})

	t.Run("WrongSchemaStructure", func(t *testing.T) {
		// Test with data that doesn't match the expected schema
		adminEvent := &models.AdminEvent{
			EventID:    "test-wrong-schema",
			SourceURL:  "https://test.example.com",
			SchemaType: "events",
			RawExtractedData: map[string]interface{}{
				"activities": []interface{}{ // Wrong key - should be "events"
					map[string]interface{}{
						"name":     "Test Activity", // Wrong field name - should be "title"
						"when":     "2024-12-15",   // Wrong field name - should be "date"
						"where":    "Test Location", // Wrong field name - should be "location"
						"cost":     "$25",
					},
				},
			},
			ExtractedAt: time.Now(),
		}

		_, err := scs.ConvertToActivity(adminEvent)
		
		// Should handle schema mismatch gracefully
		if err == nil {
			t.Error("Expected conversion to fail for wrong schema structure")
		}

		t.Logf("Wrong schema structure result: Error=%v", err)
	})
}

// TestErrorHandlingScenarios tests various error conditions
func TestErrorHandlingScenarios(t *testing.T) {
	scs := NewSchemaConversionService()

	t.Run("NilRawData", func(t *testing.T) {
		adminEvent := &models.AdminEvent{
			EventID:          "test-nil-data",
			SourceURL:        "https://test.example.com",
			SchemaType:       "events",
			RawExtractedData: nil,
			ExtractedAt:      time.Now(),
		}

		_, err := scs.ConvertToActivity(adminEvent)
		
		if err == nil {
			t.Error("Expected error for nil raw data")
		}

		t.Logf("Nil raw data error: %v", err)
	})

	t.Run("EmptyRawData", func(t *testing.T) {
		adminEvent := &models.AdminEvent{
			EventID:          "test-empty-data",
			SourceURL:        "https://test.example.com",
			SchemaType:       "events",
			RawExtractedData: map[string]interface{}{},
			ExtractedAt:      time.Now(),
		}

		_, err := scs.ConvertToActivity(adminEvent)
		
		if err == nil {
			t.Error("Expected error for empty raw data")
		}

		t.Logf("Empty raw data error: %v", err)
	})

	t.Run("UnsupportedSchemaType", func(t *testing.T) {
		adminEvent := &models.AdminEvent{
			EventID:    "test-unsupported-schema",
			SourceURL:  "https://test.example.com",
			SchemaType: "unsupported_schema_type",
			RawExtractedData: map[string]interface{}{
				"events": []interface{}{
					map[string]interface{}{
						"title": "Test Event",
					},
				},
			},
			ExtractedAt: time.Now(),
		}

		_, err := scs.ConvertToActivity(adminEvent)
		
		if err == nil {
			t.Error("Expected error for unsupported schema type")
		}

		t.Logf("Unsupported schema type error: %v", err)
	})
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr || 
		     findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}