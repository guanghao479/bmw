package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"seattle-family-activities-scraper/internal/models"
)

func main() {
	fmt.Println("Testing DynamoDB SDK Integration...")
	
	// Test 1: DynamoDB AttributeValue marshaling/unmarshaling
	testAttributeValueConversion()
	
	// Test 2: GSI key validation
	testGSIKeys()
	
	// Test 3: TTL calculations
	testTTLCalculations()
	
	fmt.Println("All DynamoDB SDK tests completed successfully!")
}

func testAttributeValueConversion() {
	fmt.Println("\n=== Testing DynamoDB AttributeValue Conversion ===")
	
	// Create a venue with all fields populated
	venue := &models.Venue{
		FamilyActivity: models.FamilyActivity{
			PK:         models.CreateVenuePK("ifly-seattle"),
			SK:         models.SortKeyMetadata,
			EntityType: models.EntityTypeVenue,
			EntityID:   "ifly-seattle",
			Name:       "iFLY Seattle",
			Description: "Indoor skydiving experience for all ages",
			Category:   models.CategoryActiveSports,
			Subcategory: "adventure-sports",
			AgeGroups: []models.AgeGroup{
				{
					Category:    models.AgeGroupAllAges,
					MinAge:      3,
					MaxAge:      99,
					Unit:        "years",
					Description: "All ages welcome",
				},
			},
			Pricing: models.ActivityPricing{
				Pricing: models.Pricing{
					Type:        models.PricingTypePaid,
					Cost:        79.95,
					Currency:    "USD",
					Unit:        "per-person",
					Description: "First-time flyer package",
				},
				IncludesSupplies: true,
			},
			ProviderID:   "PROVIDER#ifly-world",
			ProviderName: "iFLY World",
			Status:       models.ActivityStatusActive,
			Featured:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			SourceID:     "SOURCE#ifly-website",
		},
		VenueName: "iFLY Seattle",
		VenueType: models.VenueTypeIndoor,
		Address:   "349 7th Ave, Seattle, WA 98104",
		Coordinates: models.Coordinates{
			Lat: 47.6062,
			Lng: -122.3321,
		},
		Region: "seattle-downtown",
		Amenities: []string{"parking", "restrooms", "accessibility", "birthday-parties"},
		OperatingHours: map[string]string{
			"monday":    "10:00-22:00",
			"tuesday":   "10:00-22:00",
			"wednesday": "10:00-22:00",
			"thursday":  "10:00-22:00",
			"friday":    "10:00-22:00",
			"saturday":  "09:00-23:00",
			"sunday":    "09:00-21:00",
		},
		ContactInfo: models.ContactInfo{
			Phone:   "(206) 555-0123",
			Email:   "seattle@iflyworld.com",
			Website: "https://iflyworld.com/seattle",
		},
	}
	
	// Generate GSI keys
	venue.LocationKey = models.GenerateLocationKey("seattle-metro", "seattle")
	venue.CategoryAgeKey = models.GenerateCategoryAgeKey(models.CategoryActiveSports, models.AgeGroupAllAges)
	venue.ProviderKey = "PROVIDER#ifly-world"
	venue.TypeStatusKey = models.GenerateTypeStatusKey(models.EntityTypeVenue, models.ActivityStatusActive, "ifly-seattle")
	
	// Test marshaling to DynamoDB AttributeValue
	avMap, err := attributevalue.MarshalMap(venue)
	if err != nil {
		log.Fatalf("Failed to marshal venue to AttributeValue: %v", err)
	}
	
	fmt.Printf("Successfully marshaled venue to %d DynamoDB attributes\n", len(avMap))
	
	// Print some key attributes
	if pk, ok := avMap["PK"]; ok {
		fmt.Printf("PK: %v\n", pk)
	}
	if sk, ok := avMap["SK"]; ok {
		fmt.Printf("SK: %v\n", sk)
	}
	if locationKey, ok := avMap["LocationKey"]; ok {
		fmt.Printf("LocationKey: %v\n", locationKey)
	}
	
	// Test unmarshaling back from AttributeValue
	var unmarshaledVenue models.Venue
	err = attributevalue.UnmarshalMap(avMap, &unmarshaledVenue)
	if err != nil {
		log.Fatalf("Failed to unmarshal venue from AttributeValue: %v", err)
	}
	
	// Verify critical fields
	if unmarshaledVenue.PK != venue.PK {
		log.Fatalf("PK mismatch: expected %s, got %s", venue.PK, unmarshaledVenue.PK)
	}
	if unmarshaledVenue.VenueName != venue.VenueName {
		log.Fatalf("VenueName mismatch: expected %s, got %s", venue.VenueName, unmarshaledVenue.VenueName)
	}
	if len(unmarshaledVenue.AgeGroups) != len(venue.AgeGroups) {
		log.Fatalf("AgeGroups count mismatch: expected %d, got %d", len(venue.AgeGroups), len(unmarshaledVenue.AgeGroups))
	}
	
	fmt.Println("✅ AttributeValue marshaling/unmarshaling works correctly")
	
	// Test source management models
	submission := &models.SourceSubmission{
		PK:              models.CreateSourcePK("test-source"),
		SK:              models.CreateSourceSubmissionSK(),
		SourceID:        "test-source",
		SourceName:      "Test Source",
		BaseURL:         "https://test.com",
		SourceType:      models.SourceTypeVenue,
		Priority:        models.SourcePriorityHigh,
		ExpectedContent: []string{"events", "classes"},
		HintURLs:        []string{"https://test.com/events"},
		SubmittedBy:     "founder1",
		SubmittedAt:     time.Now(),
		Status:          models.SourceStatusPendingAnalysis,
	}
	
	submissionAV, err := attributevalue.MarshalMap(submission)
	if err != nil {
		log.Fatalf("Failed to marshal source submission: %v", err)
	}
	
	var unmarshaledSubmission models.SourceSubmission
	err = attributevalue.UnmarshalMap(submissionAV, &unmarshaledSubmission)
	if err != nil {
		log.Fatalf("Failed to unmarshal source submission: %v", err)
	}
	
	fmt.Println("✅ Source Management models work correctly")
	
	// Test scraping operations models
	task := &models.ScrapingTask{
		PK:            models.CreateTaskPK("test-task"),
		SK:            models.CreateTaskSK("high", "test-source", "test-task"),
		TaskID:        "test-task",
		SourceID:      "test-source",
		TaskType:      models.TaskTypeFullScrape,
		Priority:      models.TaskPriorityHigh,
		ScheduledTime: time.Now().Add(time.Hour),
		TargetURLs:    []string{"https://test.com/events"},
		Status:        models.TaskStatusScheduled,
		MaxRetries:    3,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		TTL:           models.CalculateTaskTTL(time.Now(), 30),
	}
	
	taskAV, err := attributevalue.MarshalMap(task)
	if err != nil {
		log.Fatalf("Failed to marshal scraping task: %v", err)
	}
	
	var unmarshaledTask models.ScrapingTask
	err = attributevalue.UnmarshalMap(taskAV, &unmarshaledTask)
	if err != nil {
		log.Fatalf("Failed to unmarshal scraping task: %v", err)
	}
	
	fmt.Println("✅ Scraping Operations models work correctly")
}

func testGSIKeys() {
	fmt.Println("\n=== Testing GSI Key Generation ===")
	
	// Test location-based keys
	locationKey := models.GenerateLocationKey("seattle-metro", "seattle")
	expectedLocationKey := "GEO#seattle-metro#seattle"
	if locationKey != expectedLocationKey {
		log.Fatalf("LocationKey mismatch: expected %s, got %s", expectedLocationKey, locationKey)
	}
	fmt.Printf("✅ LocationKey: %s\n", locationKey)
	
	// Test date-type keys
	dateTypeKey := models.GenerateDateTypeKey("2025-03-01", models.EntityTypeEvent, "test-event")
	expectedDateTypeKey := "DATE#2025-03-01#TYPE#EVENT#test-event"
	if dateTypeKey != expectedDateTypeKey {
		log.Fatalf("DateTypeKey mismatch: expected %s, got %s", expectedDateTypeKey, dateTypeKey)
	}
	fmt.Printf("✅ DateTypeKey: %s\n", dateTypeKey)
	
	// Test category-age keys
	categoryAgeKey := models.GenerateCategoryAgeKey(models.CategoryActiveSports, models.AgeGroupToddler)
	expectedCategoryAgeKey := "CAT#active-sports#toddler"
	if categoryAgeKey != expectedCategoryAgeKey {
		log.Fatalf("CategoryAgeKey mismatch: expected %s, got %s", expectedCategoryAgeKey, categoryAgeKey)
	}
	fmt.Printf("✅ CategoryAgeKey: %s\n", categoryAgeKey)
	
	// Test scraping operation keys
	nextRunKey := models.GenerateNextRunKey(time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC))
	expectedNextRunKey := "NEXT_RUN#2025-03-01T12:00:00Z"
	if nextRunKey != expectedNextRunKey {
		log.Fatalf("NextRunKey mismatch: expected %s, got %s", expectedNextRunKey, nextRunKey)
	}
	fmt.Printf("✅ NextRunKey: %s\n", nextRunKey)
	
	fmt.Println("All GSI keys generated correctly!")
}

func testTTLCalculations() {
	fmt.Println("\n=== Testing TTL Calculations ===")
	
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	
	// Test task TTL (30 days retention)
	taskTTL := models.CalculateTaskTTL(now, 30)
	expectedTaskExpiry := now.AddDate(0, 0, 30)
	if taskTTL != expectedTaskExpiry.Unix() {
		log.Fatalf("TaskTTL mismatch: expected %d, got %d", expectedTaskExpiry.Unix(), taskTTL)
	}
	fmt.Printf("✅ Task TTL: %d (expires: %s)\n", taskTTL, time.Unix(taskTTL, 0).Format(time.RFC3339))
	
	// Test execution TTL (90 days retention)
	executionTTL := models.CalculateExecutionTTL(now, 90)
	expectedExecutionExpiry := now.AddDate(0, 0, 90)
	if executionTTL != expectedExecutionExpiry.Unix() {
		log.Fatalf("ExecutionTTL mismatch: expected %d, got %d", expectedExecutionExpiry.Unix(), executionTTL)
	}
	fmt.Printf("✅ Execution TTL: %d (expires: %s)\n", executionTTL, time.Unix(executionTTL, 0).Format(time.RFC3339))
	
	// Test metrics TTL (365 days retention)
	metricsTTL := models.CalculateMetricsTTL(now, 365)
	expectedMetricsExpiry := now.AddDate(0, 0, 365)
	if metricsTTL != expectedMetricsExpiry.Unix() {
		log.Fatalf("MetricsTTL mismatch: expected %d, got %d", expectedMetricsExpiry.Unix(), metricsTTL)
	}
	fmt.Printf("✅ Metrics TTL: %d (expires: %s)\n", metricsTTL, time.Unix(metricsTTL, 0).Format(time.RFC3339))
	
	fmt.Println("All TTL calculations are correct!")
}

func printAttributeMap(name string, avMap map[string]types.AttributeValue) {
	fmt.Printf("\n%s AttributeValue Map:\n", name)
	for key, value := range avMap {
		valueJSON, _ := json.MarshalIndent(value, "", "  ")
		fmt.Printf("  %s: %s\n", key, string(valueJSON))
		if len(avMap) > 5 { // Limit output for large maps
			fmt.Printf("  ... (%d more attributes)\n", len(avMap)-5)
			break
		}
	}
}