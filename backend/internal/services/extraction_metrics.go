package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	"seattle-family-activities-scraper/internal/models"
)

// ExtractionMetrics tracks success rates and quality metrics for the extraction pipeline
type ExtractionMetrics struct {
	mu                    sync.RWMutex
	TotalExtractions      int64                    `json:"total_extractions"`
	SuccessfulExtractions int64                    `json:"successful_extractions"`
	FailedExtractions     int64                    `json:"failed_extractions"`
	TotalConversions      int64                    `json:"total_conversions"`
	SuccessfulConversions int64                    `json:"successful_conversions"`
	FailedConversions     int64                    `json:"failed_conversions"`
	SourceMetrics         map[string]*SourceMetric `json:"source_metrics"`
	QualityMetrics        *QualityMetrics          `json:"quality_metrics"`
	AlertThresholds       *AlertThresholds         `json:"alert_thresholds"`
	LastUpdated           time.Time                `json:"last_updated"`
}

// SourceMetric tracks metrics for a specific source
type SourceMetric struct {
	SourceURL             string    `json:"source_url"`
	TotalAttempts         int64     `json:"total_attempts"`
	SuccessfulExtractions int64     `json:"successful_extractions"`
	FailedExtractions     int64     `json:"failed_extractions"`
	TotalActivitiesFound  int64     `json:"total_activities_found"`
	AvgActivitiesPerRun   float64   `json:"avg_activities_per_run"`
	AvgProcessingTime     float64   `json:"avg_processing_time_ms"`
	LastSuccessfulRun     time.Time `json:"last_successful_run"`
	LastFailedRun         time.Time `json:"last_failed_run"`
	SuccessRate           float64   `json:"success_rate"`
	QualityScore          float64   `json:"quality_score"`
}

// QualityMetrics tracks overall data quality
type QualityMetrics struct {
	OverallQualityScore   float64 `json:"overall_quality_score"`
	AvgCompletionRate     float64 `json:"avg_completion_rate"`
	AvgFieldCoverage      float64 `json:"avg_field_coverage"`
	ActivitiesWithDates   int64   `json:"activities_with_dates"`
	ActivitiesWithLocations int64 `json:"activities_with_locations"`
	ActivitiesWithPricing int64   `json:"activities_with_pricing"`
	TotalActivitiesProcessed int64 `json:"total_activities_processed"`
}

// AlertThresholds defines when to trigger alerts
type AlertThresholds struct {
	MinSuccessRate      float64 `json:"min_success_rate"`       // Alert if success rate drops below this
	MinQualityScore     float64 `json:"min_quality_score"`      // Alert if quality score drops below this
	MaxFailureStreak    int     `json:"max_failure_streak"`     // Alert after this many consecutive failures
	MaxProcessingTimeMs int64   `json:"max_processing_time_ms"` // Alert if processing takes longer than this
}

// ExtractionAlert represents an alert condition
type ExtractionAlert struct {
	Type        string    `json:"type"`        // success_rate|quality_score|failure_streak|processing_time
	Severity    string    `json:"severity"`    // warning|error|critical
	Message     string    `json:"message"`
	SourceURL   string    `json:"source_url,omitempty"`
	Metric      string    `json:"metric"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Timestamp   time.Time `json:"timestamp"`
	Acknowledged bool     `json:"acknowledged"`
}

// Global metrics instance
var globalExtractionMetrics *ExtractionMetrics
var metricsOnce sync.Once

// GetExtractionMetrics returns the global metrics instance
func GetExtractionMetrics() *ExtractionMetrics {
	metricsOnce.Do(func() {
		globalExtractionMetrics = &ExtractionMetrics{
			SourceMetrics: make(map[string]*SourceMetric),
			QualityMetrics: &QualityMetrics{},
			AlertThresholds: &AlertThresholds{
				MinSuccessRate:      0.8,  // 80%
				MinQualityScore:     0.7,  // 70%
				MaxFailureStreak:    3,    // 3 consecutive failures
				MaxProcessingTimeMs: 30000, // 30 seconds
			},
			LastUpdated: time.Now(),
		}
	})
	return globalExtractionMetrics
}

// RecordExtractionAttempt records an extraction attempt
func (em *ExtractionMetrics) RecordExtractionAttempt(sourceURL string, success bool, activitiesFound int, processingTime time.Duration, qualityScore float64) {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Update global metrics
	em.TotalExtractions++
	if success {
		em.SuccessfulExtractions++
	} else {
		em.FailedExtractions++
	}

	// Update source-specific metrics
	if em.SourceMetrics[sourceURL] == nil {
		em.SourceMetrics[sourceURL] = &SourceMetric{
			SourceURL: sourceURL,
		}
	}

	sourceMetric := em.SourceMetrics[sourceURL]
	sourceMetric.TotalAttempts++
	
	if success {
		sourceMetric.SuccessfulExtractions++
		sourceMetric.TotalActivitiesFound += int64(activitiesFound)
		sourceMetric.LastSuccessfulRun = time.Now()
		
		// Update average activities per run
		if sourceMetric.SuccessfulExtractions > 0 {
			sourceMetric.AvgActivitiesPerRun = float64(sourceMetric.TotalActivitiesFound) / float64(sourceMetric.SuccessfulExtractions)
		}
	} else {
		sourceMetric.FailedExtractions++
		sourceMetric.LastFailedRun = time.Now()
	}

	// Update success rate
	if sourceMetric.TotalAttempts > 0 {
		sourceMetric.SuccessRate = float64(sourceMetric.SuccessfulExtractions) / float64(sourceMetric.TotalAttempts)
	}

	// Update processing time
	processingTimeMs := float64(processingTime.Nanoseconds()) / 1e6
	if sourceMetric.AvgProcessingTime == 0 {
		sourceMetric.AvgProcessingTime = processingTimeMs
	} else {
		// Exponential moving average
		sourceMetric.AvgProcessingTime = 0.8*sourceMetric.AvgProcessingTime + 0.2*processingTimeMs
	}

	// Update quality score
	if qualityScore > 0 {
		sourceMetric.QualityScore = qualityScore
	}

	em.LastUpdated = time.Now()

	log.Printf("[METRICS] Recorded extraction: URL=%s, Success=%t, Activities=%d, Time=%.1fms, Quality=%.1f", 
		sourceURL, success, activitiesFound, processingTimeMs, qualityScore)
}

// RecordConversionAttempt records a schema conversion attempt
func (em *ExtractionMetrics) RecordConversionAttempt(success bool, qualityMetrics QualityMetrics) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.TotalConversions++
	if success {
		em.SuccessfulConversions++
	} else {
		em.FailedConversions++
	}

	// Update quality metrics
	em.QualityMetrics.TotalActivitiesProcessed++
	if qualityMetrics.ActivitiesWithDates > 0 {
		em.QualityMetrics.ActivitiesWithDates += qualityMetrics.ActivitiesWithDates
	}
	if qualityMetrics.ActivitiesWithLocations > 0 {
		em.QualityMetrics.ActivitiesWithLocations += qualityMetrics.ActivitiesWithLocations
	}
	if qualityMetrics.ActivitiesWithPricing > 0 {
		em.QualityMetrics.ActivitiesWithPricing += qualityMetrics.ActivitiesWithPricing
	}

	// Update averages
	if em.QualityMetrics.TotalActivitiesProcessed > 0 {
		em.QualityMetrics.AvgCompletionRate = float64(em.SuccessfulConversions) / float64(em.TotalConversions)
		
		dateRate := float64(em.QualityMetrics.ActivitiesWithDates) / float64(em.QualityMetrics.TotalActivitiesProcessed)
		locationRate := float64(em.QualityMetrics.ActivitiesWithLocations) / float64(em.QualityMetrics.TotalActivitiesProcessed)
		pricingRate := float64(em.QualityMetrics.ActivitiesWithPricing) / float64(em.QualityMetrics.TotalActivitiesProcessed)
		
		em.QualityMetrics.AvgFieldCoverage = (dateRate + locationRate + pricingRate) / 3.0
		em.QualityMetrics.OverallQualityScore = (em.QualityMetrics.AvgCompletionRate*0.4 + 
			locationRate*0.3 + 
			dateRate*0.2 + 
			pricingRate*0.1)
	}

	em.LastUpdated = time.Now()

	log.Printf("[METRICS] Recorded conversion: Success=%t, Overall Quality=%.1f", success, em.QualityMetrics.OverallQualityScore)
}

// CheckAlerts checks for alert conditions and returns any active alerts
func (em *ExtractionMetrics) CheckAlerts() []ExtractionAlert {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var alerts []ExtractionAlert
	now := time.Now()

	// Check global success rate
	if em.TotalExtractions > 10 { // Only check after we have some data
		globalSuccessRate := float64(em.SuccessfulExtractions) / float64(em.TotalExtractions)
		if globalSuccessRate < em.AlertThresholds.MinSuccessRate {
			alerts = append(alerts, ExtractionAlert{
				Type:      "success_rate",
				Severity:  "warning",
				Message:   fmt.Sprintf("Global extraction success rate (%.1f%%) is below threshold (%.1f%%)", globalSuccessRate*100, em.AlertThresholds.MinSuccessRate*100),
				Metric:    "global_success_rate",
				Value:     globalSuccessRate,
				Threshold: em.AlertThresholds.MinSuccessRate,
				Timestamp: now,
			})
		}
	}

	// Check global quality score
	if em.QualityMetrics.OverallQualityScore > 0 && em.QualityMetrics.OverallQualityScore < em.AlertThresholds.MinQualityScore {
		alerts = append(alerts, ExtractionAlert{
			Type:      "quality_score",
			Severity:  "warning",
			Message:   fmt.Sprintf("Overall quality score (%.1f) is below threshold (%.1f)", em.QualityMetrics.OverallQualityScore, em.AlertThresholds.MinQualityScore),
			Metric:    "overall_quality_score",
			Value:     em.QualityMetrics.OverallQualityScore,
			Threshold: em.AlertThresholds.MinQualityScore,
			Timestamp: now,
		})
	}

	// Check source-specific metrics
	for sourceURL, sourceMetric := range em.SourceMetrics {
		// Check source success rate
		if sourceMetric.TotalAttempts > 5 && sourceMetric.SuccessRate < em.AlertThresholds.MinSuccessRate {
			alerts = append(alerts, ExtractionAlert{
				Type:      "success_rate",
				Severity:  "error",
				Message:   fmt.Sprintf("Source %s success rate (%.1f%%) is below threshold (%.1f%%)", sourceURL, sourceMetric.SuccessRate*100, em.AlertThresholds.MinSuccessRate*100),
				SourceURL: sourceURL,
				Metric:    "source_success_rate",
				Value:     sourceMetric.SuccessRate,
				Threshold: em.AlertThresholds.MinSuccessRate,
				Timestamp: now,
			})
		}

		// Check processing time
		if sourceMetric.AvgProcessingTime > float64(em.AlertThresholds.MaxProcessingTimeMs) {
			alerts = append(alerts, ExtractionAlert{
				Type:      "processing_time",
				Severity:  "warning",
				Message:   fmt.Sprintf("Source %s average processing time (%.1fms) exceeds threshold (%dms)", sourceURL, sourceMetric.AvgProcessingTime, em.AlertThresholds.MaxProcessingTimeMs),
				SourceURL: sourceURL,
				Metric:    "avg_processing_time",
				Value:     sourceMetric.AvgProcessingTime,
				Threshold: float64(em.AlertThresholds.MaxProcessingTimeMs),
				Timestamp: now,
			})
		}

		// Check for recent failures
		if !sourceMetric.LastFailedRun.IsZero() && sourceMetric.LastFailedRun.After(sourceMetric.LastSuccessfulRun) {
			timeSinceFailure := now.Sub(sourceMetric.LastFailedRun)
			if timeSinceFailure < 24*time.Hour { // Recent failure within 24 hours
				alerts = append(alerts, ExtractionAlert{
					Type:      "recent_failure",
					Severity:  "error",
					Message:   fmt.Sprintf("Source %s had a recent failure %v ago", sourceURL, timeSinceFailure.Round(time.Minute)),
					SourceURL: sourceURL,
					Metric:    "recent_failure",
					Value:     float64(timeSinceFailure.Minutes()),
					Threshold: 0,
					Timestamp: now,
				})
			}
		}
	}

	return alerts
}

// GetDashboardMetrics returns metrics formatted for dashboard display
func (em *ExtractionMetrics) GetDashboardMetrics() map[string]interface{} {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// Calculate global success rate
	var globalSuccessRate float64
	if em.TotalExtractions > 0 {
		globalSuccessRate = float64(em.SuccessfulExtractions) / float64(em.TotalExtractions)
	}

	// Calculate conversion success rate
	var conversionSuccessRate float64
	if em.TotalConversions > 0 {
		conversionSuccessRate = float64(em.SuccessfulConversions) / float64(em.TotalConversions)
	}

	// Get top performing sources
	topSources := make([]map[string]interface{}, 0)
	for _, sourceMetric := range em.SourceMetrics {
		if sourceMetric.TotalAttempts > 0 {
			topSources = append(topSources, map[string]interface{}{
				"url":                sourceMetric.SourceURL,
				"success_rate":       sourceMetric.SuccessRate,
				"avg_activities":     sourceMetric.AvgActivitiesPerRun,
				"total_attempts":     sourceMetric.TotalAttempts,
				"last_successful":    sourceMetric.LastSuccessfulRun,
				"avg_processing_time": sourceMetric.AvgProcessingTime,
			})
		}
	}

	return map[string]interface{}{
		"extraction": map[string]interface{}{
			"total_attempts":    em.TotalExtractions,
			"successful":        em.SuccessfulExtractions,
			"failed":           em.FailedExtractions,
			"success_rate":     globalSuccessRate,
		},
		"conversion": map[string]interface{}{
			"total_attempts":    em.TotalConversions,
			"successful":        em.SuccessfulConversions,
			"failed":           em.FailedConversions,
			"success_rate":     conversionSuccessRate,
		},
		"quality": map[string]interface{}{
			"overall_score":      em.QualityMetrics.OverallQualityScore,
			"completion_rate":    em.QualityMetrics.AvgCompletionRate,
			"field_coverage":     em.QualityMetrics.AvgFieldCoverage,
			"activities_with_dates":    em.QualityMetrics.ActivitiesWithDates,
			"activities_with_locations": em.QualityMetrics.ActivitiesWithLocations,
			"activities_with_pricing":   em.QualityMetrics.ActivitiesWithPricing,
			"total_processed":    em.QualityMetrics.TotalActivitiesProcessed,
		},
		"sources": topSources,
		"alerts":  em.CheckAlerts(),
		"last_updated": em.LastUpdated,
	}
}

// ResetMetrics resets all metrics (useful for testing)
func (em *ExtractionMetrics) ResetMetrics() {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.TotalExtractions = 0
	em.SuccessfulExtractions = 0
	em.FailedExtractions = 0
	em.TotalConversions = 0
	em.SuccessfulConversions = 0
	em.FailedConversions = 0
	em.SourceMetrics = make(map[string]*SourceMetric)
	em.QualityMetrics = &QualityMetrics{}
	em.LastUpdated = time.Now()

	log.Printf("[METRICS] Metrics reset")
}

// calculateExtractionQualityScore calculates a quality score for extracted activities
func (fc *FireCrawlClient) calculateExtractionQualityScore(activities []models.Activity, diagnostics *ExtractionDiagnostics) float64 {
	if len(activities) == 0 {
		return 0.0
	}

	var totalScore float64
	for _, activity := range activities {
		score := 0.0
		
		// Title (required) - 30%
		if activity.Title != "" {
			score += 0.3
		}
		
		// Location (required) - 25%
		if activity.Location.Name != "" {
			score += 0.25
		}
		
		// Schedule/Date - 20%
		if activity.Schedule.StartDate != "" || activity.Schedule.StartTime != "" {
			score += 0.2
		}
		
		// Pricing - 15%
		if activity.Pricing.Type != "" || activity.Pricing.Description != "" {
			score += 0.15
		}
		
		// Description - 10%
		if activity.Description != "" && len(activity.Description) > 10 {
			score += 0.1
		}
		
		totalScore += score
	}
	
	return totalScore / float64(len(activities))
}

// calculateConversionQualityMetrics calculates quality metrics for a converted activity
func (scs *SchemaConversionService) calculateConversionQualityMetrics(activity *models.Activity, issues []string) QualityMetrics {
	metrics := QualityMetrics{}
	
	if activity == nil {
		return metrics
	}
	
	// Count activities with various fields
	if activity.Schedule.StartDate != "" || activity.Schedule.StartTime != "" {
		metrics.ActivitiesWithDates = 1
	}
	
	if activity.Location.Name != "" {
		metrics.ActivitiesWithLocations = 1
	}
	
	if activity.Pricing.Type != "" || activity.Pricing.Description != "" {
		metrics.ActivitiesWithPricing = 1
	}
	
	return metrics
}

// LogMetricsSummary logs a summary of current metrics
func (em *ExtractionMetrics) LogMetricsSummary() {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var globalSuccessRate float64
	if em.TotalExtractions > 0 {
		globalSuccessRate = float64(em.SuccessfulExtractions) / float64(em.TotalExtractions)
	}

	log.Printf("[METRICS] === EXTRACTION METRICS SUMMARY ===")
	log.Printf("[METRICS] Total Extractions: %d (Success: %d, Failed: %d, Rate: %.1f%%)", 
		em.TotalExtractions, em.SuccessfulExtractions, em.FailedExtractions, globalSuccessRate*100)
	log.Printf("[METRICS] Total Conversions: %d (Success: %d, Failed: %d)", 
		em.TotalConversions, em.SuccessfulConversions, em.FailedConversions)
	log.Printf("[METRICS] Quality Score: %.1f, Completion Rate: %.1f%%, Field Coverage: %.1f%%", 
		em.QualityMetrics.OverallQualityScore, em.QualityMetrics.AvgCompletionRate*100, em.QualityMetrics.AvgFieldCoverage*100)
	log.Printf("[METRICS] Active Sources: %d", len(em.SourceMetrics))
	
	alerts := em.CheckAlerts()
	if len(alerts) > 0 {
		log.Printf("[METRICS] Active Alerts: %d", len(alerts))
		for _, alert := range alerts {
			log.Printf("[METRICS] ALERT [%s]: %s", alert.Severity, alert.Message)
		}
	}
	log.Printf("[METRICS] =====================================")
}