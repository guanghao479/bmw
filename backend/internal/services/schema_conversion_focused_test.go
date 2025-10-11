package services

import (
	"testing"
	"time"

	"seattle-family-activities-scraper/internal/models"
)

// TestSchemaConversionFieldMapping tests the comprehensive field mapping functionality
func TestSchemaConversionFieldMapping(t *testing.T) {
	scs := NewSchemaConversionService()

	t.Run("DirectFieldMapping", func(t *testing.T) {
		// Test with perfect field mapping
		adminEvent := &models.AdminEvent{
			EventID:    "test-direct-mapping",
			SourceURL:  "https://test.example.com",
			SchemaType: "events",
			RawExtractedData: map[string]interface{}{
				"events": []interface{}{
					map[string]interface{}{
						"title":       "Perfect Event",
						"description": "This event has all the right fields",
						"date":        "2024-12-15",
						"time":        "2:00 PM",
						"location":    "Seattle Community Center",
						"address":     "123 Main St, Seattle, WA",
						"price":       "$25",
						"ages":        "5-10 years",
					},
				},
			},
			ExtractedAt: time.Now(),
		}

		result, err := scs.ConvertToActivity(adminEvent)
		if err != nil {
			t.Fatalf("Conversion failed: %v", err)
		}

		if result.Activity == nil {
			t.Fatal("Expected activity to be created")
		}

		// Check field mappings
		expectedMappings := map[string]string{
			"title":       "title",
			"description": "description",
			"schedule":    "date",
			"location":    "location",
			"pricing":     "price",
		}

		for field, expectedSource := range expectedMappings {
			if actualSource, exists := result.FieldMappings[field]; !exists {
				t.Errorf("Expected field mapping for %s", field)
			} else if actualSource != expectedSource {
				t.Errorf("Expected %s to map to %s, got %s", field, expectedSource, actualSource)
			}
		}

		// Check confidence score
		if result.ConfidenceScore < 80 {
			t.Errorf("Expected high confidence score for perfect mapping, got %f", result.ConfidenceScore)
		}

		// Check detailed mappings if available
		if result.DetailedMappings != nil {
			titleMapping, exists := result.DetailedMappings["title"]
			if exists {
				if mapping, ok := titleMapping.(FieldMapping); ok {
					if mapping.MappingType != "direct" {
						t.Errorf("Expected direct mapping type for title, got %s", mapping.MappingType)
					}
					if mapping.Confidence < 0.8 {
						t.Errorf("Expected high confidence for direct mapping, got %f", mapping.Confidence)
					}
				}
			}
		}
	})

	t.Run("FallbackFieldMapping", func(t *testing.T) {
		// Test with non-standard field names that require fallback
		adminEvent := &models.AdminEvent{
			EventID:    "test-fallback-mapping",
			SourceURL:  "https://test.example.com",
			SchemaType: "events",
			RawExtractedData: map[string]interface{}{
				"events": []interface{}{
					map[string]interface{}{
						"name":        "Event with Non-Standard Fields", // Should map to title
						"info":        "Event details here",             // Should map to description
						"when":        "2024-12-15",                     // Should map to date
						"where":       "Community Center",               // Should map to location
						"cost":        "Free",                           // Should map to price
						"age_range":   "All ages",                       // Should map to ages
					},
				},
			},
			ExtractedAt: time.Now(),
		}

		result, err := scs.ConvertToActivity(adminEvent)
		if err != nil {
			t.Fatalf("Conversion failed: %v", err)
		}

		if result.Activity == nil {
			t.Fatal("Expected activity to be created")
		}

		// Check that fallback mappings worked
		expectedMappings := map[string]string{
			"title":       "name",
			"description": "info",
			"pricing":     "cost",
		}

		for field, expectedSource := range expectedMappings {
			if actualSource, exists := result.FieldMappings[field]; !exists {
				t.Errorf("Expected field mapping for %s", field)
			} else if actualSource != expectedSource {
				t.Errorf("Expected %s to map to %s, got %s", field, expectedSource, actualSource)
			}
		}

		// Should have lower confidence than direct mapping
		if result.ConfidenceScore >= 90 {
			t.Errorf("Expected lower confidence score for fallback mapping, got %f", result.ConfidenceScore)
		}
	})

	t.Run("MixedMappingQuality", func(t *testing.T) {
		// Test with mix of good and poor field mappings
		adminEvent := &models.AdminEvent{
			EventID:    "test-mixed-mapping",
			SourceURL:  "https://test.example.com",
			SchemaType: "events",
			RawExtractedData: map[string]interface{}{
				"events": []interface{}{
					map[string]interface{}{
						"title":       "Mixed Quality Event",
						"description": "Good description field",
						"when":        "sometime next week", // Poor date format
						"venue":       "Some Location",      // Non-standard location field
						// Missing price information
						"random_field": "irrelevant data",
					},
				},
			},
			ExtractedAt: time.Now(),
		}

		result, err := scs.ConvertToActivity(adminEvent)
		if err != nil {
			t.Fatalf("Conversion failed: %v", err)
		}

		if result.Activity == nil {
			t.Fatal("Expected activity to be created")
		}

		// Should have moderate confidence
		if result.ConfidenceScore < 30 || result.ConfidenceScore > 80 {
			t.Errorf("Expected moderate confidence score for mixed quality, got %f", result.ConfidenceScore)
		}

		// Should have some issues
		if len(result.Issues) == 0 {
			t.Error("Expected some conversion issues for mixed quality data")
		}

		// Check that some fields have "not_found" mappings
		notFoundCount := 0
		for _, source := range result.FieldMappings {
			if source == "not_found" {
				notFoundCount++
			}
		}

		if notFoundCount == 0 {
			t.Error("Expected some fields to have 'not_found' mappings")
		}
	})
}

// TestValidationFunctionality tests the validation system
func TestValidationFunctionality(t *testing.T) {
	scs := NewSchemaConversionService()

	t.Run("DateValidation", func(t *testing.T) {
		testCases := []struct {
			date     string
			expected bool
			name     string
		}{
			{"2024-12-15", true, "ISO format"},
			{"12/15/2024", true, "US format"},
			{"December 15, 2024", true, "Long format"},
			{"invalid-date", false, "Invalid format"},
			{"", false, "Empty date"},
			{"2020-01-01", false, "Past date"}, // Should be flagged as potentially problematic
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := scs.validateDateField(tc.date, "test_date")
				
				if tc.expected && !result.IsValid && result.Confidence < 0.5 {
					t.Errorf("Expected %s to be valid or have reasonable confidence, got IsValid=%t, Confidence=%f", 
						tc.date, result.IsValid, result.Confidence)
				}
				
				if !tc.expected && result.IsValid && result.Confidence > 0.8 {
					t.Errorf("Expected %s to be invalid or have low confidence, got IsValid=%t, Confidence=%f", 
						tc.date, result.IsValid, result.Confidence)
				}

				t.Logf("Date '%s': IsValid=%t, Confidence=%f, Issues=%v", 
					tc.date, result.IsValid, result.Confidence, result.Issues)
			})
		}
	})

	t.Run("TimeValidation", func(t *testing.T) {
		testCases := []struct {
			time     string
			expected bool
			name     string
		}{
			{"2:00 PM", true, "12-hour format"},
			{"14:30", true, "24-hour format"},
			{"9 AM", true, "Simple format"},
			{"25:99", false, "Invalid time"},
			{"", false, "Empty time"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := scs.validateTimeField(tc.time, "test_time")
				
				if tc.expected && !result.IsValid {
					t.Errorf("Expected %s to be valid, got IsValid=%t", tc.time, result.IsValid)
				}
				
				if !tc.expected && result.IsValid {
					t.Errorf("Expected %s to be invalid, got IsValid=%t", tc.time, result.IsValid)
				}

				t.Logf("Time '%s': IsValid=%t, Confidence=%f", tc.time, result.IsValid, result.Confidence)
			})
		}
	})

	t.Run("TitleValidation", func(t *testing.T) {
		testCases := []struct {
			title    string
			minConf  float64
			name     string
		}{
			{"Great Kids Art Workshop", 0.9, "Good title"},
			{"Art", 0.3, "Too short"},
			{"", 0.1, "Empty title"},
			{"Untitled Event", 0.1, "Default title"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := scs.validateTitleField(tc.title)
				
				if result.Confidence < tc.minConf {
					t.Errorf("Expected confidence >= %f for '%s', got %f", tc.minConf, tc.title, result.Confidence)
				}

				t.Logf("Title '%s': IsValid=%t, Confidence=%f", tc.title, result.IsValid, result.Confidence)
			})
		}
	})
}

// TestConversionDiagnostics tests the diagnostic information generation
func TestConversionDiagnostics(t *testing.T) {
	scs := NewSchemaConversionService()

	adminEvent := &models.AdminEvent{
		EventID:    "test-diagnostics",
		SourceURL:  "https://test.example.com",
		SchemaType: "events",
		RawExtractedData: map[string]interface{}{
			"events": []interface{}{
				map[string]interface{}{
					"title":       "Diagnostic Test Event",
					"description": "Testing diagnostic information",
					"date":        "2024-12-15",
					"location":    "Test Location",
					"price":       "$25",
				},
			},
		},
		ExtractedAt: time.Now(),
	}

	_, err := scs.ConvertToActivity(adminEvent)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	// Get diagnostics
	diagnostics := scs.GetLastConversionDiagnostics()
	if diagnostics == nil {
		t.Fatal("Expected diagnostics to be available")
	}

	// Check diagnostic fields
	if diagnostics.AdminEventID != adminEvent.EventID {
		t.Errorf("Expected AdminEventID %s, got %s", adminEvent.EventID, diagnostics.AdminEventID)
	}

	if diagnostics.SourceURL != adminEvent.SourceURL {
		t.Errorf("Expected SourceURL %s, got %s", adminEvent.SourceURL, diagnostics.SourceURL)
	}

	if diagnostics.SchemaType != adminEvent.SchemaType {
		t.Errorf("Expected SchemaType %s, got %s", adminEvent.SchemaType, diagnostics.SchemaType)
	}

	if diagnostics.ProcessingTime <= 0 {
		t.Error("Expected positive processing time")
	}

	if len(diagnostics.ExtractionAttempts) == 0 {
		t.Error("Expected extraction attempts to be recorded")
	}

	if len(diagnostics.FieldMappings) == 0 {
		t.Error("Expected field mappings to be recorded")
	}

	// Check that field mappings have proper structure
	for field, mapping := range diagnostics.FieldMappings {
		if mapping.ActivityField != field {
			t.Errorf("Field mapping inconsistency: key=%s, ActivityField=%s", field, mapping.ActivityField)
		}
		
		if mapping.Confidence < 0 || mapping.Confidence > 1 {
			t.Errorf("Invalid confidence score for field %s: %f", field, mapping.Confidence)
		}
		
		if mapping.MappingType == "" {
			t.Errorf("Missing mapping type for field %s", field)
		}
	}

	t.Logf("Diagnostics: ProcessingTime=%v, ExtractionAttempts=%d, FieldMappings=%d, Issues=%d", 
		diagnostics.ProcessingTime, len(diagnostics.ExtractionAttempts), len(diagnostics.FieldMappings), len(diagnostics.ConversionIssues))
}

// TestErrorRecovery tests how well the system recovers from various error conditions
func TestErrorRecovery(t *testing.T) {
	scs := NewSchemaConversionService()

	t.Run("PartialDataRecovery", func(t *testing.T) {
		// Test with some valid and some invalid events in the same array
		adminEvent := &models.AdminEvent{
			EventID:    "test-partial-recovery",
			SourceURL:  "https://test.example.com",
			SchemaType: "events",
			RawExtractedData: map[string]interface{}{
				"events": []interface{}{
					// Valid event
					map[string]interface{}{
						"title":    "Good Event",
						"date":     "2024-12-15",
						"location": "Seattle",
					},
					// Invalid event (missing critical fields)
					map[string]interface{}{
						"random_field": "no useful data",
					},
					// Another valid event
					map[string]interface{}{
						"title":    "Another Good Event",
						"date":     "2024-12-16",
						"location": "Bellevue",
					},
				},
			},
			ExtractedAt: time.Now(),
		}

		result, err := scs.ConvertToActivity(adminEvent)
		if err != nil {
			t.Fatalf("Conversion should handle partial data gracefully: %v", err)
		}

		// Should successfully convert the first valid event
		if result.Activity == nil {
			t.Error("Expected to recover and convert at least one valid event")
		}

		// Should have reasonable confidence despite some invalid data
		if result.ConfidenceScore < 50 {
			t.Errorf("Expected reasonable confidence for partial recovery, got %f", result.ConfidenceScore)
		}

		t.Logf("Partial recovery: Confidence=%f, Issues=%d", result.ConfidenceScore, len(result.Issues))
	})

	t.Run("GracefulDegradation", func(t *testing.T) {
		// Test with progressively worse data quality
		testCases := []struct {
			name string
			data map[string]interface{}
		}{
			{
				"MissingOptionalFields",
				map[string]interface{}{
					"title":    "Basic Event",
					"date":     "2024-12-15",
					"location": "Seattle",
					// Missing description, price, etc.
				},
			},
			{
				"MissingLocation",
				map[string]interface{}{
					"title": "Event Without Location",
					"date":  "2024-12-15",
					// Missing location
				},
			},
			{
				"OnlyTitle",
				map[string]interface{}{
					"title": "Minimal Event",
					// Missing everything else
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				adminEvent := &models.AdminEvent{
					EventID:    "test-degradation-" + tc.name,
					SourceURL:  "https://test.example.com",
					SchemaType: "events",
					RawExtractedData: map[string]interface{}{
						"events": []interface{}{tc.data},
					},
					ExtractedAt: time.Now(),
				}

				result, err := scs.ConvertToActivity(adminEvent)
				if err != nil {
					t.Fatalf("Conversion should degrade gracefully: %v", err)
				}

				if result.Activity == nil {
					t.Error("Expected activity to be created even with minimal data")
				}

				// Confidence should decrease with data quality
				t.Logf("%s: Confidence=%f, Issues=%d", tc.name, result.ConfidenceScore, len(result.Issues))
			})
		}
	})
}