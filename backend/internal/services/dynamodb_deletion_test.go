package services

import (
	"testing"

	"seattle-family-activities-scraper/internal/models"
)

// Test helper functions for source record keys
func TestGetSourceRecordKeys(t *testing.T) {
	sourceID := "test-source-123"
	keys := models.GetSourceRecordKeys(sourceID)
	
	if len(keys) != 3 {
		t.Errorf("Expected 3 source record keys, got %d", len(keys))
	}
	
	expectedPK := models.CreateSourcePK(sourceID)
	for _, key := range keys {
		if key.PK != expectedPK {
			t.Errorf("Expected PK %s, got %s", expectedPK, key.PK)
		}
	}
	
	// Check that we have all expected SK values
	expectedSKs := map[string]bool{
		models.CreateSourceSubmissionSK(): false,
		models.CreateSourceAnalysisSK():   false,
		models.CreateSourceConfigSK():     false,
	}
	
	for _, key := range keys {
		if _, exists := expectedSKs[key.SK]; exists {
			expectedSKs[key.SK] = true
		}
	}
	
	for sk, found := range expectedSKs {
		if !found {
			t.Errorf("Expected SK %s not found in keys", sk)
		}
	}
}

func TestGetActivityRecordPrefix(t *testing.T) {
	sourceID := "test-source-456"
	prefix := models.GetActivityRecordPrefix(sourceID)
	
	expected := "ACTIVITY#" + sourceID
	if prefix != expected {
		t.Errorf("Expected prefix %s, got %s", expected, prefix)
	}
}

func TestDeletionResult_Validate(t *testing.T) {
	tests := []struct {
		name        string
		result      models.DeletionResult
		expectError bool
	}{
		{
			name: "Valid deletion result",
			result: models.DeletionResult{
				SourceID:          "test-source",
				SubmissionDeleted: true,
				ActivitiesDeleted: 5,
				TotalRecords:      6,
			},
			expectError: false,
		},
		{
			name: "Missing source ID",
			result: models.DeletionResult{
				SubmissionDeleted: true,
				TotalRecords:      1,
			},
			expectError: true,
		},
		{
			name: "Negative activities deleted",
			result: models.DeletionResult{
				SourceID:          "test-source",
				ActivitiesDeleted: -1,
				TotalRecords:      0,
			},
			expectError: true,
		},
		{
			name: "Negative total records",
			result: models.DeletionResult{
				SourceID:     "test-source",
				TotalRecords: -1,
			},
			expectError: true,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.result.Validate()
			if test.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !test.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestSourceDeletionEvent_Validate(t *testing.T) {
	validDeletionData := models.DeletionResult{
		SourceID:     "test-source",
		TotalRecords: 1,
	}
	
	tests := []struct {
		name        string
		event       models.SourceDeletionEvent
		expectError bool
	}{
		{
			name: "Valid deletion event",
			event: models.SourceDeletionEvent{
				EventID:      "event-123",
				EventType:    models.AdminEventTypeDeletion,
				AdminUser:    "admin@test.com",
				SourceID:     "test-source",
				SourceName:   "Test Source",
				DeletionData: validDeletionData,
			},
			expectError: false,
		},
		{
			name: "Missing event ID",
			event: models.SourceDeletionEvent{
				EventType:    models.AdminEventTypeDeletion,
				AdminUser:    "admin@test.com",
				SourceID:     "test-source",
				SourceName:   "Test Source",
				DeletionData: validDeletionData,
			},
			expectError: true,
		},
		{
			name: "Wrong event type",
			event: models.SourceDeletionEvent{
				EventID:      "event-123",
				EventType:    models.AdminEventTypeExtraction,
				AdminUser:    "admin@test.com",
				SourceID:     "test-source",
				SourceName:   "Test Source",
				DeletionData: validDeletionData,
			},
			expectError: true,
		},
		{
			name: "Missing admin user",
			event: models.SourceDeletionEvent{
				EventID:      "event-123",
				EventType:    models.AdminEventTypeDeletion,
				SourceID:     "test-source",
				SourceName:   "Test Source",
				DeletionData: validDeletionData,
			},
			expectError: true,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.event.Validate()
			if test.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !test.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

