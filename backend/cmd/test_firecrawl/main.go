package main

import (
	"log"
	"os"

	"seattle-family-activities-scraper/internal/services"
)

func main() {
	log.Println("Testing FireCrawl Integration...")

	// Check if FIRECRAWL_API_KEY is set
	apiKey := os.Getenv("FIRECRAWL_API_KEY")
	if apiKey == "" {
		log.Println("⚠️  FIRECRAWL_API_KEY not set - this test will fail")
		log.Println("   Set FIRECRAWL_API_KEY environment variable to test FireCrawl integration")
		log.Println("   You can get a free API key from https://firecrawl.dev")
		return
	}

	// Initialize FireCrawl client
	client, err := services.NewFireCrawlClient()
	if err != nil {
		log.Fatalf("❌ Failed to create FireCrawl client: %v", err)
	}

	log.Println("✅ FireCrawl client initialized successfully")

	// Test 1: Check service availability
	log.Println("\n--- Test 1: Service Availability ---")
	if client.IsFireCrawlAvailable() {
		log.Println("✅ FireCrawl service is available")
	} else {
		log.Println("❌ FireCrawl service is not available")
		return
	}

	// Test 2: Extract activities from a sample ParentMap URL
	log.Println("\n--- Test 2: Activity Extraction ---")
	testURL := "https://www.parentmap.com/calendar?date=2025-01-15"
	log.Printf("Testing extraction from: %s", testURL)

	response, err := client.ExtractActivities(testURL)
	if err != nil {
		log.Printf("❌ Failed to extract activities: %v", err)

		// Try a simpler test URL
		log.Println("Trying with a simpler test URL...")
		testURL = "https://httpbin.org/get"
		response, err = client.ExtractActivities(testURL)
		if err != nil {
			log.Printf("❌ Failed with simple URL too: %v", err)
			return
		}
	}

	log.Printf("✅ Extraction successful!")
	log.Printf("   - Activities found: %d", len(response.Data.Activities))
	log.Printf("   - Credits used: %d", response.CreditsUsed)
	log.Printf("   - Extract time: %s", response.Metadata.ExtractTime.Format("15:04:05"))
	log.Printf("   - URL processed: %s", response.Metadata.URL)

	// Display first few activities
	if len(response.Data.Activities) > 0 {
		log.Println("\n--- Sample Activities ---")
		maxDisplay := 3
		if len(response.Data.Activities) < maxDisplay {
			maxDisplay = len(response.Data.Activities)
		}

		for i := 0; i < maxDisplay; i++ {
			activity := response.Data.Activities[i]
			log.Printf("Activity %d:", i+1)
			log.Printf("  Title: %s", activity.Title)
			log.Printf("  Location: %s", activity.Location.Name)
			if activity.Location.Address != "" {
				log.Printf("  Address: %s", activity.Location.Address)
			}
			if activity.Schedule.StartDate != "" {
				log.Printf("  Schedule: %s %s", activity.Schedule.StartDate, activity.Schedule.StartTime)
			}
			if len(activity.AgeGroups) > 0 {
				log.Printf("  Age Groups: %v", activity.AgeGroups)
			}
			if activity.Pricing.Description != "" {
				log.Printf("  Pricing: %s", activity.Pricing.Description)
			}
			if activity.Registration.URL != "" {
				log.Printf("  Registration: %s", activity.Registration.URL)
			}
			log.Println()
		}
	} else {
		log.Println("ℹ️  No activities extracted (this might be expected for test URLs)")
	}

	// Test 3: Validate response
	log.Println("\n--- Test 3: Response Validation ---")
	issues := client.ValidateExtractResponse(response)
	if len(issues) == 0 {
		log.Println("✅ Response validation passed")
	} else {
		log.Printf("⚠️  Response validation found %d issues:", len(issues))
		for _, issue := range issues {
			log.Printf("   - %s", issue)
		}
	}

	// Test 4: Statistics
	log.Println("\n--- Test 4: Client Statistics ---")
	stats := client.GetStats()
	log.Printf("Total requests: %d", stats.TotalRequests)
	log.Printf("Successful requests: %d", stats.SuccessfulReqs)
	log.Printf("Failed requests: %d", stats.FailedReqs)
	log.Printf("Average response time: %v", stats.AvgResponseTime)
	log.Printf("Total credits used: %d", stats.TotalCreditsUsed)
	log.Printf("Total activities extracted: %d", stats.TotalActivitiesExt)

	log.Println("\n🎉 FireCrawl integration test completed!")

	if len(response.Data.Activities) == 0 {
		log.Println("\nℹ️  Note: No activities were extracted in this test.")
		log.Println("   This is normal for test URLs that don't contain activity data.")
		log.Println("   To test with real data, try setting a ParentMap URL with recent events.")
	}
}