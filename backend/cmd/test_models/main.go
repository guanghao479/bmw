package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"seattle-family-activities-scraper/internal/models"
)

func main() {
	fmt.Println("Testing DynamoDB Go Models...")
	
	// Test 1: Family Activities Models
	testFamilyActivitiesModels()
	
	// Test 2: Source Management Models  
	testSourceManagementModels()
	
	// Test 3: Scraping Operations Models
	testScrapingOperationsModels()
	
	fmt.Println("All model tests completed successfully!")
}

func testFamilyActivitiesModels() {
	fmt.Println("\n=== Testing Family Activities Models ===")
	
	// Test Venue creation
	venue := &models.Venue{
		FamilyActivity: models.FamilyActivity{
			PK:         models.CreateVenuePK("ifly-seattle"),
			SK:         models.SortKeyMetadata,
			EntityType: models.EntityTypeVenue,
			EntityID:   "ifly-seattle",
			Name:       "iFLY Seattle",
			Category:   models.CategoryActiveSports,
			Status:     models.ActivityStatusActive,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		VenueName: "iFLY Seattle",
		VenueType: models.VenueTypeIndoor,
		Address:   "349 7th Ave, Seattle, WA 98104",
		Region:    "seattle-downtown",
	}
	
	// Test JSON marshaling
	venueJSON, err := json.MarshalIndent(venue, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal venue: %v", err)
	}
	fmt.Printf("Venue JSON: %s\n", string(venueJSON)[:200] + "...")
	
	// Test Event creation
	event := &models.Event{
		FamilyActivity: models.FamilyActivity{
			PK:         models.CreateEventPK("winter-festival-2025"),
			SK:         models.SortKeyMetadata,
			EntityType: models.EntityTypeEvent,
			EntityID:   "winter-festival-2025",
			Name:       "Seattle Winter Festival",
			Category:   models.CategoryEntertainmentEvents,
			Status:     models.ActivityStatusActive,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		EventName: "Seattle Winter Festival",
		EventType: "community-festival",
		VenueID:   "VENUE#seattle-center",
		Schedule: models.Schedule{
			Type:      models.ScheduleTypeMultiDay,
			StartDate: "2025-02-01",
			EndDate:   "2025-02-03",
			StartTime: "10:00",
			EndTime:   "18:00",
		},
	}
	
	// Test GSI key generation
	event.LocationKey = models.GenerateLocationKey("seattle-metro", "seattle")
	event.DateTypeKey = models.GenerateDateTypeKey("2025-02-01", models.EntityTypeEvent, "winter-festival-2025")
	event.CategoryAgeKey = models.GenerateCategoryAgeKey(models.CategoryEntertainmentEvents, models.AgeGroupAllAges)
	
	fmt.Printf("Event PK: %s, SK: %s\n", event.PK, event.SK)
	fmt.Printf("LocationKey: %s\n", event.LocationKey)
	fmt.Printf("DateTypeKey: %s\n", event.DateTypeKey)
	
	// Test Program with instances
	program := &models.Program{
		FamilyActivity: models.FamilyActivity{
			PK:         models.CreateProgramPK("soccer-tots-spring-2025"),
			SK:         models.SortKeyMetadata,
			EntityType: models.EntityTypeProgram,
			EntityID:   "soccer-tots-spring-2025",
			Name:       "Soccer Tots Spring Session",
			Category:   models.CategoryActiveSports,
			Status:     models.ActivityStatusActive,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		ProgramName:  "Soccer Tots Spring Session",
		ProgramType:  "sports-class",
		VenueID:      "VENUE#magnuson-park",
		SessionCount: 8,
		Duration:     "1 hour",
	}
	
	// Test Program Instance
	instance := &models.ProgramInstance{
		PK:                models.CreateProgramPK("soccer-tots-spring-2025"),
		SK:                models.CreateInstanceSK("2025-03-01", "10:00:00"),
		ProgramID:         "soccer-tots-spring-2025",
		InstanceDate:      "2025-03-01",
		InstanceTime:      "10:00-11:00",
		Status:            "scheduled",
		RegistrationStatus: "open",
		CurrentEnrollment: 12,
		MaxEnrollment:     15,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	
	fmt.Printf("Program PK: %s, SK: %s\n", program.PK, program.SK)
	fmt.Printf("Instance PK: %s, SK: %s\n", instance.PK, instance.SK)
}

func testSourceManagementModels() {
	fmt.Println("\n=== Testing Source Management Models ===")
	
	// Test Source Submission
	submission := &models.SourceSubmission{
		PK:              models.CreateSourcePK("seattle-childrens-theatre"),
		SK:              models.CreateSourceSubmissionSK(),
		SourceID:        "seattle-childrens-theatre",
		SourceName:      "Seattle Children's Theatre",
		BaseURL:         "https://sct.org",
		SourceType:      models.SourceTypeVenue,
		Priority:        models.SourcePriorityHigh,
		ExpectedContent: []string{"events", "classes"},
		HintURLs:        []string{"https://sct.org/events", "https://sct.org/classes"},
		SubmittedBy:     "founder1",
		SubmittedAt:     time.Now(),
		Status:          models.SourceStatusPendingAnalysis,
	}
	
	// Generate GSI keys
	submission.StatusKey = models.GenerateSourceStatusKey(models.SourceStatusPendingAnalysis)
	submission.PriorityKey = models.GenerateSourcePriorityKey(models.SourcePriorityHigh, "seattle-childrens-theatre")
	
	// Test validation
	if err := submission.Validate(); err != nil {
		log.Fatalf("Source submission validation failed: %v", err)
	}
	
	fmt.Printf("Submission PK: %s, SK: %s\n", submission.PK, submission.SK)
	fmt.Printf("StatusKey: %s, PriorityKey: %s\n", submission.StatusKey, submission.PriorityKey)
	
	// Test Source Analysis
	analysis := &models.SourceAnalysis{
		PK:                  models.CreateSourcePK("seattle-childrens-theatre"),
		SK:                  models.CreateSourceAnalysisSK(),
		SourceID:            "seattle-childrens-theatre",
		AnalysisCompletedAt: time.Now(),
		AnalysisVersion:     "1.0",
		DiscoveredPatterns: models.DiscoveryPatterns{
			SitemapFound: true,
			SitemapURL:   "https://sct.org/sitemap.xml",
			ContentPages: []models.ContentPage{
				{
					URL:        "https://sct.org/events",
					Type:       "events",
					Confidence: 0.95,
					Title:      "Events & Shows",
				},
			},
			DataSelectors: models.DataSelectors{
				Title:       ".event-title h2",
				Date:        ".event-date",
				Description: ".event-content p",
				Price:       ".price-info",
			},
		},
		ExtractionTestResults: models.ExtractionTestResults{
			TestURL:      "https://sct.org/events",
			ItemsFound:   12,
			QualityScore: 0.92,
			TestDuration: 1500,
		},
		OverallQualityScore: 0.92,
		Status:              models.SourceStatusAnalysisComplete,
	}
	
	fmt.Printf("Analysis PK: %s, SK: %s\n", analysis.PK, analysis.SK)
	fmt.Printf("Quality Score: %.2f\n", analysis.OverallQualityScore)
	
	// Test Source Config
	config := &models.DynamoSourceConfig{
		PK:         models.CreateSourcePK("seattle-childrens-theatre"),
		SK:         models.CreateSourceConfigSK(),
		SourceID:   "seattle-childrens-theatre",
		SourceName: "Seattle Children's Theatre",
		SourceType: models.SourceTypeVenue,
		BaseURL:    "https://sct.org",
		TargetURLs: []string{"https://sct.org/events", "https://sct.org/classes"},
		ContentSelectors: models.DataSelectors{
			Title:       ".event-title h2",
			Date:        ".event-date",
			Description: ".event-content p",
		},
		ScrapingConfig: models.DynamoScrapingConfig{
			Frequency: "weekly",
			Priority:  models.SourcePriorityHigh,
			RateLimit: models.RateLimit{
				RequestsPerMinute:    5,
				DelayBetweenRequests: 2000,
				ConcurrentRequests:   2,
			},
		},
		Status:      models.SourceStatusActive,
		ActivatedBy: "founder1",
		ActivatedAt: time.Now(),
	}
	
	// Test validation
	if err := config.Validate(); err != nil {
		log.Fatalf("Source config validation failed: %v", err)
	}
	
	fmt.Printf("Config PK: %s, SK: %s\n", config.PK, config.SK)
}

func testScrapingOperationsModels() {
	fmt.Println("\n=== Testing Scraping Operations Models ===")
	
	// Test Scraping Task
	task := &models.ScrapingTask{
		PK:            models.CreateTaskPK("task-001"),
		SK:            models.CreateTaskSK(models.TaskPriorityHigh, "seattle-childrens-theatre", "task-001"),
		TaskID:        "task-001",
		SourceID:      "seattle-childrens-theatre",
		TaskType:      models.TaskTypeFullScrape,
		Priority:      models.TaskPriorityHigh,
		ScheduledTime: time.Now().Add(time.Hour),
		TargetURLs:    []string{"https://sct.org/events"},
		Status:        models.TaskStatusScheduled,
		MaxRetries:    3,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		TTL:           models.CalculateTaskTTL(time.Now(), 30), // 30 days retention
	}
	
	// Generate GSI keys
	task.NextRunKey = models.GenerateNextRunKey(task.ScheduledTime)
	task.PrioritySourceKey = models.GenerateTaskPrioritySourceKey(models.TaskPriorityHigh, "seattle-childrens-theatre")
	
	// Test validation
	if err := task.Validate(); err != nil {
		log.Fatalf("Scraping task validation failed: %v", err)
	}
	
	// Test status transitions
	if !task.CanTransitionTo(models.TaskStatusInProgress) {
		log.Fatalf("Task should be able to transition from scheduled to in_progress")
	}
	
	fmt.Printf("Task PK: %s, SK: %s\n", task.PK, task.SK)
	fmt.Printf("NextRunKey: %s\n", task.NextRunKey)
	fmt.Printf("TTL: %d (expires: %s)\n", task.TTL, time.Unix(task.TTL, 0).Format(time.RFC3339))
	
	// Test Scraping Execution
	execution := &models.ScrapingExecution{
		PK:             models.CreateExecutionPK("exec-001"),
		SK:             "STATUS",
		ExecutionID:    "exec-001",
		TaskID:         "task-001",
		SourceID:       "seattle-childrens-theatre",
		StartedAt:      time.Now(),
		Status:         "running",
		ItemsExtracted: 0,
		ItemsProcessed: 0,
		ItemsStored:    0,
		TTL:            models.CalculateExecutionTTL(time.Now(), 90), // 90 days retention
	}
	
	// Test validation
	if err := execution.Validate(); err != nil {
		log.Fatalf("Scraping execution validation failed: %v", err)
	}
	
	fmt.Printf("Execution PK: %s, SK: %s\n", execution.PK, execution.SK)
	
	// Test Source Metrics
	metrics := &models.SourceMetrics{
		PK:                    models.CreateSourcePK("seattle-childrens-theatre"),
		SK:                    models.CreateMetricsSK("2025-01-16"),
		SourceID:              "seattle-childrens-theatre",
		MetricsDate:           "2025-01-16",
		TotalRuns:             5,
		SuccessfulRuns:        4,
		FailedRuns:            1,
		AverageDuration:       45000, // 45 seconds
		TotalItemsFound:       60,
		AverageItemsFound:     12.0,
		SuccessRate:           0.8,
		DataQualityScore:      0.92,
		ContentStabilityScore: 0.88,
		UpdatedAt:             time.Now(),
		TTL:                   models.CalculateMetricsTTL(time.Now(), 365), // 1 year retention
	}
	
	fmt.Printf("Metrics PK: %s, SK: %s\n", metrics.PK, metrics.SK)
	fmt.Printf("Success Rate: %.1f%%, Quality Score: %.2f\n", metrics.SuccessRate*100, metrics.DataQualityScore)
}