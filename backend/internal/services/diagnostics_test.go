package services

import (
	"testing"
	"time"

	"seattle-family-activities-scraper/internal/models"
)

// TestFireCrawlDiagnostics tests the diagnostic logging in FireCrawl service
func TestFireCrawlDiagnostics(t *testing.T) {
	// This test verifies that diagnostic structures are properly initialized
	// and logging functions work without errors

	diagnostics := &ExtractionDiagnostics{
		URL:                "https://test.example.com",
		StartTime:          time.Now(),
		ExtractionAttempts: []ExtractionAttempt{},
		ValidationIssues:   []ValidationIssue{},
		StructuredData:     make(map[string]interface{}),
	}

	// Test validation issue creation
	diagnostics.ValidationIssues = append(diagnostics.ValidationIssues, ValidationIssue{
		Severity:   "warning",
		Field:      "test_field",
		Message:    "Test validation message",
		Suggestion: "Test suggestion",
	})

	// Test extraction attempt creation
	attempt := ExtractionAttempt{
		Method:      "test_method",
		Timestamp:   time.Now(),
		Success:     true,
		EventsFound: 5,
		Details:     make(map[string]interface{}),
		Issues:      []string{"test issue"},
	}
	diagnostics.ExtractionAttempts = append(diagnostics.ExtractionAttempts, attempt)

	// Complete diagnostics
	diagnostics.EndTime = time.Now()
	diagnostics.ProcessingTime = time.Since(diagnostics.StartTime)
	diagnostics.Success = true

	// Test that we can create a FireCrawl client and call logging
	// Note: We can't test actual FireCrawl calls without API key
	if client, err := NewFireCrawlClient(); err == nil {
		client.logDiagnostics(diagnostics)
		t.Log("FireCrawl diagnostics logging completed successfully")
	} else {
		t.Logf("Skipping FireCrawl client test (no API key): %v", err)
		// Still test the logging function directly
		fc := &FireCrawlClient{}
		fc.logDiagnostics(diagnostics)
		t.Log("FireCrawl diagnostics logging (without client) completed successfully")
	}
}

// TestSchemaConversionDiagnostics tests the diagnostic logging in schema conversion service
func TestSchemaConversionDiagnostics(t *testing.T) {
	scs := NewSchemaConversionService()

	// Create a test admin event
	adminEvent := &models.AdminEvent{
		EventID:    "test-event-123",
		SourceURL:  "https://test.example.com",
		SchemaType: "events",
		RawExtractedData: map[string]interface{}{
			"events": []interface{}{
				map[string]interface{}{
					"title":       "Test Event",
					"description": "Test event description",
					"location":    "Test Location",
					"date":        "2024-12-01",
					"price":       "Free",
				},
			},
		},
		ExtractedAt: time.Now(),
	}

	// Test conversion with diagnostics
	result, err := scs.ConvertToActivity(adminEvent)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected conversion result, got nil")
	}

	if result.Activity == nil {
		t.Error("Expected activity to be created, got nil")
	}

	t.Logf("Conversion completed with confidence score: %.1f", result.ConfidenceScore)
	t.Logf("Issues found: %d", len(result.Issues))
	t.Logf("Field mappings: %d", len(result.FieldMappings))

	// Test that diagnostics were stored
	lastDiagnostics := scs.GetLastConversionDiagnostics()
	if lastDiagnostics == nil {
		t.Error("Expected diagnostics to be stored, got nil")
	} else {
		t.Logf("Diagnostics captured: Success=%t, ProcessingTime=%v", 
			lastDiagnostics.Success, lastDiagnostics.ProcessingTime)
	}
}

// TestDiagnosticsStructures tests that diagnostic structures can be created and populated
func TestDiagnosticsStructures(t *testing.T) {
	// Test ExtractionDiagnostics
	extractionDiag := &ExtractionDiagnostics{
		URL:               "https://test.com",
		StartTime:         time.Now(),
		RawMarkdownLength: 1000,
		RawMarkdownSample: "Sample markdown content...",
		Success:           true,
	}

	if extractionDiag.URL != "https://test.com" {
		t.Error("ExtractionDiagnostics URL not set correctly")
	}

	// Test ConversionDiagnostics
	conversionDiag := &ConversionDiagnostics{
		AdminEventID:     "test-123",
		SourceURL:        "https://test.com",
		SchemaType:       "events",
		StartTime:        time.Now(),
		ConfidenceScore:  85.5,
		Success:          true,
	}

	if conversionDiag.AdminEventID != "test-123" {
		t.Error("ConversionDiagnostics AdminEventID not set correctly")
	}

	// Test ValidationIssue
	validationIssue := ValidationIssue{
		Severity:   "error",
		Field:      "title",
		Message:    "Title is required",
		Suggestion: "Ensure title field is present",
		RawValue:   "",
	}

	if validationIssue.Severity != "error" {
		t.Error("ValidationIssue severity not set correctly")
	}

	// Test ConversionIssue
	conversionIssue := ConversionIssue{
		Type:       "missing_field",
		Field:      "location",
		Message:    "Location field is missing",
		Suggestion: "Add location extraction logic",
		Severity:   "warning",
	}

	if conversionIssue.Type != "missing_field" {
		t.Error("ConversionIssue type not set correctly")
	}

	t.Log("All diagnostic structures created and populated successfully")
}