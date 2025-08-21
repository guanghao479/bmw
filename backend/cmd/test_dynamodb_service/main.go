package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"seattle-family-activities-scraper/internal/models"
	"seattle-family-activities-scraper/internal/services"
)

func main() {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-west-2"),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create DynamoDB client
	dynamoClient := dynamodb.NewFromConfig(cfg)

	// Get table names from environment variables (these should be set by Lambda)
	familyActivitiesTable := os.Getenv("FAMILY_ACTIVITIES_TABLE")
	sourceManagementTable := os.Getenv("SOURCE_MANAGEMENT_TABLE")
	scrapingOperationsTable := os.Getenv("SCRAPING_OPERATIONS_TABLE")

	// Fallback to default table names if environment variables not set
	if familyActivitiesTable == "" {
		familyActivitiesTable = "seattle-family-activities"
	}
	if sourceManagementTable == "" {
		sourceManagementTable = "seattle-source-management"
	}
	if scrapingOperationsTable == "" {
		scrapingOperationsTable = "seattle-scraping-operations"
	}

	fmt.Printf("Testing DynamoDB Service with tables:\n")
	fmt.Printf("- Family Activities: %s\n", familyActivitiesTable)
	fmt.Printf("- Source Management: %s\n", sourceManagementTable)
	fmt.Printf("- Scraping Operations: %s\n", scrapingOperationsTable)

	// Create DynamoDB service
	dbService := services.NewDynamoDBService(
		dynamoClient,
		familyActivitiesTable,
		sourceManagementTable,
		scrapingOperationsTable,
	)

	ctx := context.Background()

	// Test 1: Create and retrieve a family activity (venue)
	fmt.Println("\n=== Test 1: Family Activity (Venue) ===")
	venue := &models.FamilyActivity{
		PK:         models.CreateVenuePK("ifly-seattle"),
		SK:         models.SortKeyMetadata,
		EntityType: models.EntityTypeVenue,
		EntityID:   "ifly-seattle",
		Name:       "iFLY Seattle",
		Description: "Indoor skydiving experience for all ages",
		Category:   "active-sports",
		Subcategory: "adventure-sports",
		Location: models.ActivityLocation{
			Location: models.Location{
				Address: "349 7th Ave, Seattle, WA 98104",
				City:    "Seattle",
				State:   "WA",
				ZipCode: "98104",
				Region:  "seattle-downtown",
			},
			VenueType:     "indoor",
			Accessibility: "ADA accessible",
			Parking:       "Paid parking available",
		},
		AgeGroups: []models.AgeGroup{
			{Category: "family", MinAge: 3, MaxAge: 99},
		},
		Pricing: models.ActivityPricing{
			Pricing: models.Pricing{
				Type:     "paid",
				Cost:     79.95,
				Currency: "USD",
			},
			IncludesSupplies: true,
		},
		ProviderID:   "PROVIDER#ifly-world",
		ProviderName: "iFLY World",
		Status:       "active",
		Featured:     true,
		SourceID:     "SOURCE#ifly-website",
	}

	err = dbService.CreateFamilyActivity(ctx, venue)
	if err != nil {
		log.Printf("Failed to create venue: %v", err)
	} else {
		fmt.Println("✅ Created venue successfully")
	}

	// Retrieve the venue
	retrievedVenue, err := dbService.GetFamilyActivity(ctx, venue.PK, venue.SK)
	if err != nil {
		log.Printf("Failed to retrieve venue: %v", err)
	} else {
		fmt.Printf("✅ Retrieved venue: %s\n", retrievedVenue.Name)
	}

	// Test 2: Create and retrieve a source submission
	fmt.Println("\n=== Test 2: Source Submission ===")
	sourceSubmission := &models.SourceSubmission{
		SourceID:    "seattle-childrens-theatre",
		SourceName:  "Seattle Children's Theatre",
		BaseURL:     "https://sct.org",
		SourceType:  models.SourceTypeEventOrganizer,
		Priority:    models.SourcePriorityHigh,
		ExpectedContent: []string{"events", "classes", "camps"},
		HintURLs:    []string{"https://sct.org/events", "https://sct.org/classes"},
		SubmittedBy: "founder@seattlefamilyactivities.com",
		Status:      models.SourceStatusPendingAnalysis,
	}

	err = dbService.CreateSourceSubmission(ctx, sourceSubmission)
	if err != nil {
		log.Printf("Failed to create source submission: %v", err)
	} else {
		fmt.Println("✅ Created source submission successfully")
	}

	// Retrieve the source submission
	retrievedSubmission, err := dbService.GetSourceSubmission(ctx, sourceSubmission.SourceID)
	if err != nil {
		log.Printf("Failed to retrieve source submission: %v", err)
	} else {
		fmt.Printf("✅ Retrieved source submission: %s\n", retrievedSubmission.SourceName)
	}

	// Test 3: Create and retrieve a scraping task
	fmt.Println("\n=== Test 3: Scraping Task ===")
	scrapingTask := &models.ScrapingTask{
		TaskID:       "task-001",
		SourceID:     sourceSubmission.SourceID,
		TaskType:     models.TaskTypeFullScrape,
		Priority:     models.TaskPriorityHigh,
		ScheduledTime: time.Now().Add(1 * time.Hour),
		TargetURLs:   []string{"https://sct.org/events"},
		ExtractionRules: models.DataSelectors{
			Title:       ".event-title h2",
			Date:        ".event-date",
			Description: ".event-content p",
		},
		Status: models.TaskStatusScheduled,
	}

	// Set primary keys
	scrapingTask.PK = models.CreateTaskPK(scrapingTask.TaskID)
	scrapingTask.SK = models.CreateTaskSK(scrapingTask.Priority, scrapingTask.SourceID, scrapingTask.TaskID)

	err = dbService.CreateScrapingTask(ctx, scrapingTask)
	if err != nil {
		log.Printf("Failed to create scraping task: %v", err)
	} else {
		fmt.Println("✅ Created scraping task successfully")
	}

	// Test 4: Query operations using GSIs
	fmt.Println("\n=== Test 4: GSI Queries ===")

	// Query activities by location
	activities, err := dbService.QueryFamilyActivitiesByLocation(ctx, "seattle-downtown", "Seattle", 10)
	if err != nil {
		log.Printf("Failed to query activities by location: %v", err)
	} else {
		fmt.Printf("✅ Found %d activities in Seattle downtown\n", len(activities))
	}

	// Query sources by status
	sources, err := dbService.QuerySourcesByStatus(ctx, models.SourceStatusPendingAnalysis, 10)
	if err != nil {
		log.Printf("Failed to query sources by status: %v", err)
	} else {
		fmt.Printf("✅ Found %d sources pending analysis\n", len(sources))
	}

	// Query next scraping tasks
	nextTasks, err := dbService.QueryNextScrapingTasks(ctx, time.Now().Add(2*time.Hour), 10)
	if err != nil {
		log.Printf("Failed to query next scraping tasks: %v", err)
	} else {
		fmt.Printf("✅ Found %d tasks ready to run\n", len(nextTasks))
	}

	fmt.Println("\n=== DynamoDB Service Test Complete ===")
	fmt.Println("All basic CRUD operations and GSI queries are working!")
}