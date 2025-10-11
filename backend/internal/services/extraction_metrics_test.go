package services

import (
	"testing"
	"time"

	"seattle-family-activities-scraper/internal/models"
)

func TestExtractionMetrics(t *testing.T) {
	// Get a fresh metrics instance for testing
	metrics := GetExtractionMetrics()
	metrics.ResetMetrics()

	// Test recording extraction attempts
	t.Run("RecordExtractionAttempts", func(t *testing.T) {
		// Record successful extraction
		metrics.RecordExtractionAttempt("https://example.com", true, 5, 2*time.Second, 0.8)
		
		if metrics.TotalExtractions != 1 {
			t.Errorf("Expected 1 total extraction, got %d", metrics.TotalExtractions)
		}
		
		if metrics.SuccessfulExtractions != 1 {
			t.Errorf("Expected 1 successful extraction, got %d", metrics.SuccessfulExtractions)
		}
		
		if metrics.FailedExtractions != 0 {
			t.Errorf("Expected 0 failed extractions, got %d", metrics.FailedExtractions)
		}
		
		// Check source metrics
		sourceMetric := metrics.SourceMetrics["https://example.com"]
		if sourceMetric == nil {
			t.Fatal("Source metric not created")
		}
		
		if sourceMetric.SuccessRate != 1.0 {
			t.Errorf("Expected success rate 1.0, got %f", sourceMetric.SuccessRate)
		}
		
		if sourceMetric.AvgActivitiesPerRun != 5.0 {
			t.Errorf("Expected avg activities 5.0, got %f", sourceMetric.AvgActivitiesPerRun)
		}

		// Record failed extraction
		metrics.RecordExtractionAttempt("https://example.com", false, 0, 1*time.Second, 0.0)
		
		if metrics.TotalExtractions != 2 {
			t.Errorf("Expected 2 total extractions, got %d", metrics.TotalExtractions)
		}
		
		if metrics.FailedExtractions != 1 {
			t.Errorf("Expected 1 failed extraction, got %d", metrics.FailedExtractions)
		}
		
		// Check updated success rate
		if sourceMetric.SuccessRate != 0.5 {
			t.Errorf("Expected success rate 0.5, got %f", sourceMetric.SuccessRate)
		}
	})

	// Test recording conversion attempts
	t.Run("RecordConversionAttempts", func(t *testing.T) {
		qualityMetrics := QualityMetrics{
			ActivitiesWithDates:     1,
			ActivitiesWithLocations: 1,
			ActivitiesWithPricing:   1,
		}
		
		metrics.RecordConversionAttempt(true, qualityMetrics)
		
		if metrics.TotalConversions != 1 {
			t.Errorf("Expected 1 total conversion, got %d", metrics.TotalConversions)
		}
		
		if metrics.SuccessfulConversions != 1 {
			t.Errorf("Expected 1 successful conversion, got %d", metrics.SuccessfulConversions)
		}
		
		if metrics.QualityMetrics.ActivitiesWithDates != 1 {
			t.Errorf("Expected 1 activity with dates, got %d", metrics.QualityMetrics.ActivitiesWithDates)
		}
	})

	// Test alert checking
	t.Run("CheckAlerts", func(t *testing.T) {
		// Reset and create conditions for alerts
		metrics.ResetMetrics()
		
		// Create multiple failed extractions to trigger source-specific success rate alert
		for i := 0; i < 6; i++ {
			metrics.RecordExtractionAttempt("https://failing-source.com", false, 0, 1*time.Second, 0.0)
		}
		
		alerts := metrics.CheckAlerts()
		
		// Should have at least one alert for low success rate (source-specific)
		found := false
		for _, alert := range alerts {
			if alert.Type == "success_rate" && alert.SourceURL == "https://failing-source.com" {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("Expected source-specific success rate alert, but none found")
		}
	})

	// Test dashboard metrics
	t.Run("GetDashboardMetrics", func(t *testing.T) {
		dashboard := metrics.GetDashboardMetrics()
		
		// Check structure
		if dashboard["extraction"] == nil {
			t.Error("Dashboard missing extraction metrics")
		}
		
		if dashboard["conversion"] == nil {
			t.Error("Dashboard missing conversion metrics")
		}
		
		if dashboard["quality"] == nil {
			t.Error("Dashboard missing quality metrics")
		}
		
		if dashboard["sources"] == nil {
			t.Error("Dashboard missing sources metrics")
		}
		
		if dashboard["alerts"] == nil {
			t.Error("Dashboard missing alerts")
		}
	})
}

func TestQualityScoreCalculation(t *testing.T) {
	fc := &FireCrawlClient{}
	
	// Test with complete activity
	completeActivity := models.Activity{
		Title: "Test Event",
		Location: models.Location{
			Name: "Test Venue",
		},
		Schedule: models.Schedule{
			StartDate: "2025-10-10",
			StartTime: "10:00 AM",
		},
		Pricing: models.Pricing{
			Type:        "free",
			Description: "Free event",
		},
		Description: "This is a detailed description of the test event",
	}
	
	activities := []models.Activity{completeActivity}
	diagnostics := &ExtractionDiagnostics{}
	
	score := fc.calculateExtractionQualityScore(activities, diagnostics)
	
	// Should get full score (1.0) for complete activity
	if score != 1.0 {
		t.Errorf("Expected quality score 1.0 for complete activity, got %f", score)
	}
	
	// Test with incomplete activity
	incompleteActivity := models.Activity{
		Title: "Incomplete Event",
		// Missing location, schedule, pricing, description
	}
	
	incompleteActivities := []models.Activity{incompleteActivity}
	incompleteScore := fc.calculateExtractionQualityScore(incompleteActivities, diagnostics)
	
	// Should get only title score (0.3)
	if incompleteScore != 0.3 {
		t.Errorf("Expected quality score 0.3 for incomplete activity, got %f", incompleteScore)
	}
	
	// Test with empty activities
	emptyScore := fc.calculateExtractionQualityScore([]models.Activity{}, diagnostics)
	if emptyScore != 0.0 {
		t.Errorf("Expected quality score 0.0 for empty activities, got %f", emptyScore)
	}
}

func TestConversionQualityMetrics(t *testing.T) {
	scs := &SchemaConversionService{}
	
	// Test with complete activity
	completeActivity := &models.Activity{
		Title: "Test Event",
		Location: models.Location{
			Name: "Test Venue",
		},
		Schedule: models.Schedule{
			StartDate: "2025-10-10",
		},
		Pricing: models.Pricing{
			Type: "free",
		},
	}
	
	metrics := scs.calculateConversionQualityMetrics(completeActivity, []string{})
	
	if metrics.ActivitiesWithDates != 1 {
		t.Errorf("Expected 1 activity with dates, got %d", metrics.ActivitiesWithDates)
	}
	
	if metrics.ActivitiesWithLocations != 1 {
		t.Errorf("Expected 1 activity with locations, got %d", metrics.ActivitiesWithLocations)
	}
	
	if metrics.ActivitiesWithPricing != 1 {
		t.Errorf("Expected 1 activity with pricing, got %d", metrics.ActivitiesWithPricing)
	}
	
	// Test with nil activity
	nilMetrics := scs.calculateConversionQualityMetrics(nil, []string{"error"})
	
	if nilMetrics.ActivitiesWithDates != 0 {
		t.Errorf("Expected 0 activities with dates for nil activity, got %d", nilMetrics.ActivitiesWithDates)
	}
}