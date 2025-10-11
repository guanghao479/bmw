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

	// Test Remlinger Farms URL
	testURL := "https://www.remlingerfarms.com/events"
	
	log.Printf("Testing Remlinger Farms parsing with URL: %s", testURL)
	
	// Extract activities using the new parsing logic
	response, err := client.ExtractActivities(testURL)
	if err != nil {
		log.Fatalf("Failed to extract activities: %v", err)
	}

	fmt.Printf("=== REMLINGER FARMS EXTRACTION RESULTS ===\n")
	fmt.Printf("Success: %v\n", response.Success)
	fmt.Printf("Activities found: %d\n", len(response.Data.Activities))
	fmt.Printf("Credits used: %d\n", response.CreditsUsed)
	
	// Display each activity
	for i, activity := range response.Data.Activities {
		fmt.Printf("\n--- ACTIVITY %d ---\n", i+1)
		fmt.Printf("ID: %s\n", activity.ID)
		fmt.Printf("Title: %s\n", activity.Title)
		fmt.Printf("Description: %s\n", activity.Description)
		fmt.Printf("Type: %s\n", activity.Type)
		fmt.Printf("Category: %s\n", activity.Category)
		
		// Schedule
		fmt.Printf("Schedule:\n")
		fmt.Printf("  Start Date: %s\n", activity.Schedule.StartDate)
		fmt.Printf("  Start Time: %s\n", activity.Schedule.StartTime)
		fmt.Printf("  End Time: %s\n", activity.Schedule.EndTime)
		
		// Location
		fmt.Printf("Location:\n")
		fmt.Printf("  Name: %s\n", activity.Location.Name)
		fmt.Printf("  Address: %s\n", activity.Location.Address)
		fmt.Printf("  City: %s\n", activity.Location.City)
		fmt.Printf("  State: %s\n", activity.Location.State)
		
		// Pricing
		fmt.Printf("Pricing:\n")
		fmt.Printf("  Type: %s\n", activity.Pricing.Type)
		fmt.Printf("  Description: %s\n", activity.Pricing.Description)
		
		// Age Groups
		if len(activity.AgeGroups) > 0 {
			fmt.Printf("Age Groups:\n")
			for _, ag := range activity.AgeGroups {
				fmt.Printf("  %s: %s\n", ag.Category, ag.Description)
			}
		}
		
		// Source
		fmt.Printf("Source:\n")
		fmt.Printf("  URL: %s\n", activity.Source.URL)
		fmt.Printf("  Domain: %s\n", activity.Source.Domain)
	}
	
	// Validate the extraction
	issues := client.ValidateExtractResponse(response)
	if len(issues) > 0 {
		fmt.Printf("\n=== VALIDATION ISSUES ===\n")
		for _, issue := range issues {
			fmt.Printf("- %s\n", issue)
		}
	} else {
		fmt.Printf("\n=== VALIDATION PASSED ===\n")
	}
}