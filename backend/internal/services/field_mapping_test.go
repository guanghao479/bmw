package services

import (
	"testing"
	"time"

	"seattle-family-activities-scraper/internal/models"
)

func TestComprehensiveFieldMappingAndValidation(t *testing.T) {
	scs := NewSchemaConversionService()

	// Create test admin event with various data quality issues
	adminEvent := &models.AdminEvent{
		EventID:    "test-field-mapping-123",
		SourceURL:  "https://parentmap.com/test-event",
		SchemaType: "events",
		RawExtractedData: map[string]interface{}{
			"events": []interface{}{
				map[string]interface{}{
					"title":       "Kids Art Workshop",
					"description": "Fun art activities for children",
					"location":    "Community Center",
					"date":        "2024-12-15", // Valid future date
					"time":        "2:00 PM",    // Valid time format
					"price":       "$25",        // Valid price
					"ages":        "5-10 years", // Age information
				},
			},
		},
		ExtractedAt: time.Now(),
	}

	// Test conversion with comprehensive validation
	result, err := scs.ConvertToActivity(adminEvent)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	if result.Activity == nil {
		t.Fatal("Expected activity to be created")
	}

	// Verify detailed mappings are present
	if result.DetailedMappings == nil {
		t.Fatal("Expected detailed mappings to be present")
	}

	// Verify validation results are present
	if result.ValidationResults == nil {
		t.Fatal("Expected validation results to be present")
	}

	// Check specific field mappings
	expectedMappings := map[string]string{
		"title":       "title",
		"description": "description",
		"location":    "location",
		"schedule":    "date",
		"pricing":     "price",
	}

	for field, expectedSource := range expectedMappings {
		if actualSource, exists := result.FieldMappings[field]; !exists {
			t.Errorf("Expected field mapping for %s, but not found", field)
		} else if actualSource != expectedSource {
			t.Errorf("Expected field mapping for %s to be %s, got %s", field, expectedSource, actualSource)
		}
	}

	// Verify detailed mapping information
	titleMapping, exists := result.DetailedMappings["title"]
	if !exists {
		t.Fatal("Expected detailed mapping for title")
	}

	titleMappingMap, ok := titleMapping.(FieldMapping)
	if !ok {
		t.Fatal("Expected title mapping to be FieldMapping type")
	}

	if titleMappingMap.ActivityField != "title" {
		t.Errorf("Expected activity field to be 'title', got %s", titleMappingMap.ActivityField)
	}

	if titleMappingMap.SourceField != "title" {
		t.Errorf("Expected source field to be 'title', got %s", titleMappingMap.SourceField)
	}

	if titleMappingMap.MappingType != "direct" {
		t.Errorf("Expected mapping type to be 'direct', got %s", titleMappingMap.MappingType)
	}

	if titleMappingMap.Confidence <= 0 || titleMappingMap.Confidence > 1 {
		t.Errorf("Expected confidence to be between 0 and 1, got %f", titleMappingMap.Confidence)
	}

	// Verify validation results
	titleValidation, exists := result.ValidationResults["title"]
	if !exists {
		t.Fatal("Expected validation result for title")
	}

	titleValidationMap, ok := titleValidation.(map[string]interface{})
	if !ok {
		t.Fatal("Expected title validation to be map")
	}

	if titleValidationMap["validation_status"] != "valid" {
		t.Errorf("Expected title validation status to be 'valid', got %s", titleValidationMap["validation_status"])
	}

	// Test with problematic data
	t.Run("ProblematicData", func(t *testing.T) {
		problematicEvent := &models.AdminEvent{
			EventID:    "test-problematic-123",
			SourceURL:  "https://example.com/bad-data",
			SchemaType: "events",
			RawExtractedData: map[string]interface{}{
				"events": []interface{}{
					map[string]interface{}{
						"name":        "Event", // Short title in non-standard field
						"info":        "Brief", // Short description in non-standard field
						"venue":       "Place", // Location in non-standard field
						"when":        "invalid-date", // Invalid date format
						"cost":        "varies", // Variable pricing
					},
				},
			},
			ExtractedAt: time.Now(),
		}

		result, err := scs.ConvertToActivity(problematicEvent)
		if err != nil {
			t.Fatalf("Conversion failed: %v", err)
		}

		// Should still create activity but with lower confidence
		if result.Activity == nil {
			t.Fatal("Expected activity to be created even with problematic data")
		}

		if result.ConfidenceScore >= 80 {
			t.Errorf("Expected lower confidence score for problematic data, got %f", result.ConfidenceScore)
		}

		// Should have validation issues
		if len(result.Issues) == 0 {
			t.Error("Expected validation issues for problematic data")
		}

		// Check that field mappings show fallback strategies
		titleMapping := result.FieldMappings["title"]
		if titleMapping != "name" {
			t.Errorf("Expected title to be mapped from 'name' field, got %s", titleMapping)
		}

		descMapping := result.FieldMappings["description"]
		if descMapping != "info" {
			t.Errorf("Expected description to be mapped from 'info' field, got %s", descMapping)
		}
	})
}

func TestFieldValidationFunctions(t *testing.T) {
	scs := NewSchemaConversionService()

	t.Run("DateValidation", func(t *testing.T) {
		// Test valid date
		result := scs.validateDateField("2024-12-15", "start_date")
		if !result.IsValid {
			t.Error("Expected valid date to pass validation")
		}

		// Test invalid date
		result = scs.validateDateField("invalid-date", "start_date")
		if result.IsValid {
			t.Error("Expected invalid date to fail validation")
		}

		if len(result.Issues) == 0 {
			t.Error("Expected validation issues for invalid date")
		}

		if len(result.Suggestions) == 0 {
			t.Error("Expected suggestions for invalid date")
		}
	})

	t.Run("TimeValidation", func(t *testing.T) {
		// Test valid times
		validTimes := []string{"14:30", "2:30 PM", "2:30PM"}
		for _, timeStr := range validTimes {
			result := scs.validateTimeField(timeStr, "start_time")
			if !result.IsValid {
				t.Errorf("Expected valid time %s to pass validation", timeStr)
			}
		}

		// Test invalid time
		result := scs.validateTimeField("25:70", "start_time")
		if result.IsValid {
			t.Error("Expected invalid time to fail validation")
		}
	})

	t.Run("TitleValidation", func(t *testing.T) {
		// Test good title
		result := scs.validateTitleField("Kids Art Workshop")
		if !result.IsValid {
			t.Error("Expected good title to pass validation")
		}

		// Test short title
		result = scs.validateTitleField("Art")
		if result.Confidence >= 0.8 {
			t.Error("Expected lower confidence for short title")
		}

		// Test empty title
		result = scs.validateTitleField("")
		if result.IsValid {
			t.Error("Expected empty title to fail validation")
		}
	})
}