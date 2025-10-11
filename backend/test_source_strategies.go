package main

import (
	"fmt"
	"log"
	"os"

	"seattle-family-activities-scraper/internal/services"
)

func main() {
	// Check for API key
	apiKey := os.Getenv("FIRECRAWL_API_KEY")
	if apiKey == "" {
		log.Fatal("FIRECRAWL_API_KEY environment variable is required")
	}

	// Initialize FireCrawl client
	client, err := services.NewFireCrawlClient()
	if err != nil {
		log.Fatalf("Failed to initialize FireCrawl client: %v", err)
	}

	// Test URLs for different sources
	testURLs := []string{
		"https://www.parentmap.com/calendar?date=2025-01-15",
		"https://www.remlingerfarms.com/events",
		"https://www.zoo.org/events", // Generic test
	}
	
	for i, testURL := range testURLs {
		fmt.Printf("\n=== TEST %d: %s ===\n", i+1, testURL)
		
		// Extract activities using the new strategy system
		response, err := client.ExtractActivities(testURL)
		if err != nil {
			log.Printf("Failed to extract activities from %s: %v", testURL, err)
			continue
		}

		fmt.Printf("Success: %v\n", response.Success)
		fmt.Printf("Activities found: %d\n", len(response.Data.Activities))
		fmt.Printf("Credits used: %d\n", response.CreditsUsed)
		
		// Show first activity as example
		if len(response.Data.Activities) > 0 {
			activity := response.Data.Activities[0]
			fmt.Printf("First Activity:\n")
			fmt.Printf("  Title: %s\n", activity.Title)
			fmt.Printf("  Location: %s\n", activity.Location.Name)
			fmt.Printf("  Date: %s\n", activity.Schedule.StartDate)
			fmt.Printf("  Time: %s\n", activity.Schedule.StartTime)
		}
		
		// Validate the extraction
		issues := client.ValidateExtractResponse(response)
		if len(issues) > 0 {
			fmt.Printf("Validation Issues: %d\n", len(issues))
		} else {
			fmt.Printf("Validation: PASSED\n")
		}
	}
}