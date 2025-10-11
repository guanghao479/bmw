package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/mendableai/firecrawl-go"
)

func main() {
	// Check for API key
	apiKey := os.Getenv("FIRECRAWL_API_KEY")
	if apiKey == "" {
		log.Fatal("FIRECRAWL_API_KEY environment variable is required")
	}

	// Initialize FireCrawl client
	app, err := firecrawl.NewFirecrawlApp(apiKey, "https://api.firecrawl.dev")
	if err != nil {
		log.Fatalf("Failed to initialize FireCrawl client: %v", err)
	}

	// Test Remlinger Farms URL
	testURL := "https://www.remlingerfarms.com/events"
	
	log.Printf("Analyzing Remlinger Farms structure from: %s", testURL)
	
	// Scrape the URL to get markdown content
	doc, err := app.ScrapeURL(testURL, nil)
	if err != nil {
		log.Fatalf("Failed to scrape URL: %v", err)
	}

	fmt.Printf("=== REMLINGER FARMS ANALYSIS ===\n")
	fmt.Printf("URL: %s\n", testURL)
	fmt.Printf("Markdown Length: %d characters\n", len(doc.Markdown))
	
	lines := strings.Split(doc.Markdown, "\n")
	
	// Find event patterns
	fmt.Printf("\n=== EVENT PATTERN ANALYSIS ===\n")
	
	eventCount := 0
	inEventBlock := false
	currentEvent := []string{}
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Look for potential event title patterns
		if isRemlingerEventTitle(line) {
			// Save previous event if we have one
			if len(currentEvent) > 0 {
				analyzeRemlingerEvent(currentEvent, eventCount)
				eventCount++
			}
			
			// Start new event
			currentEvent = []string{line}
			inEventBlock = true
			continue
		}
		
		// Collect content in event blocks
		if inEventBlock && line != "" {
			currentEvent = append(currentEvent, line)
			
			// Stop collecting if we hit another event or reach a logical break
			if len(currentEvent) > 15 || (i < len(lines)-1 && isRemlingerEventBreak(lines[i+1])) {
				inEventBlock = false
			}
		}
	}
	
	// Don't forget the last event
	if len(currentEvent) > 0 {
		analyzeRemlingerEvent(currentEvent, eventCount)
		eventCount++
	}
	
	fmt.Printf("\nTotal potential events found: %d\n", eventCount)
	
	// Show general structure analysis
	fmt.Printf("\n=== GENERAL STRUCTURE ANALYSIS ===\n")
	
	headerCount := 0
	listItemCount := 0
	datePatternCount := 0
	timePatternCount := 0
	
	fmt.Printf("\n=== SAMPLE LINES BY TYPE ===\n")
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Check for headers
		if strings.HasPrefix(line, "#") {
			headerCount++
			if headerCount <= 5 {
				fmt.Printf("HEADER %d: %s\n", headerCount, line)
			}
		}
		
		// Check for list items
		if strings.HasPrefix(line, "*") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "+") {
			listItemCount++
			if listItemCount <= 5 {
				fmt.Printf("LIST %d: %s\n", listItemCount, line)
			}
		}
		
		// Check for date patterns
		if containsDatePattern(line) {
			datePatternCount++
			if datePatternCount <= 5 {
				fmt.Printf("DATE %d: %s\n", datePatternCount, line)
			}
		}
		
		// Check for time patterns
		if containsTimePattern(line) {
			timePatternCount++
			if timePatternCount <= 5 {
				fmt.Printf("TIME %d: %s\n", timePatternCount, line)
			}
		}
		
		// Show first 20 non-empty lines for general structure
		if i <= 20 && line != "" {
			fmt.Printf("LINE %d: %s\n", i+1, line)
		}
	}
	
	fmt.Printf("\n=== PATTERN COUNTS ===\n")
	fmt.Printf("Headers: %d\n", headerCount)
	fmt.Printf("List items: %d\n", listItemCount)
	fmt.Printf("Date patterns: %d\n", datePatternCount)
	fmt.Printf("Time patterns: %d\n", timePatternCount)
}

func isRemlingerEventTitle(line string) bool {
	line = strings.TrimSpace(line)
	
	// Check for markdown headers
	if strings.HasPrefix(line, "#") && len(line) > 5 {
		return true
	}
	
	// Check for event-like titles
	if len(line) > 10 && len(line) < 150 {
		eventKeywords := []string{
			"event", "activity", "class", "workshop", "program", "camp", "story", "time",
			"music", "art", "dance", "play", "festival", "fair", "market", "farm",
			"tour", "walk", "hike", "performance", "show", "concert", "movie", "pumpkin",
			"harvest", "christmas", "halloween", "easter", "spring", "summer", "fall", "winter",
		}
		
		lowerLine := strings.ToLower(line)
		for _, keyword := range eventKeywords {
			if strings.Contains(lowerLine, keyword) {
				return true
			}
		}
		
		// Check if it looks like a title (has capital letters and reasonable length)
		if looksLikeTitle(line) {
			return true
		}
	}
	
	return false
}

func isRemlingerEventBreak(line string) bool {
	line = strings.TrimSpace(line)
	
	// Check if this line indicates a new event or section
	if strings.HasPrefix(line, "#") {
		return true
	}
	
	return false
}

func analyzeRemlingerEvent(eventLines []string, eventNum int) {
	fmt.Printf("\n--- POTENTIAL REMLINGER EVENT %d ---\n", eventNum+1)
	
	var title, dateTime, location, description string
	var ageGroups, pricing []string
	
	for _, line := range eventLines {
		line = strings.TrimSpace(line)
		
		// Extract title from first line or header
		if title == "" && (strings.HasPrefix(line, "#") || looksLikeTitle(line)) {
			title = strings.TrimLeft(line, "#")
			title = strings.TrimSpace(title)
		}
		
		// Look for date/time patterns
		if containsDatePattern(line) || containsTimePattern(line) {
			if dateTime == "" {
				dateTime = line
			}
		}
		
		// Look for location indicators
		if containsLocationPattern(line) {
			if location == "" {
				location = line
			}
		}
		
		// Look for age group indicators
		if containsAgePattern(line) {
			ageGroups = append(ageGroups, extractAgeGroups(line)...)
		}
		
		// Look for pricing information
		if containsPricePattern(line) {
			pricing = append(pricing, extractPricing(line)...)
		}
		
		// Collect description text (non-header, substantial lines)
		if !strings.HasPrefix(line, "#") && len(line) > 20 && description == "" {
			description = line
		}
	}
	
	// Display extracted information
	fmt.Printf("TITLE: %s\n", title)
	if dateTime != "" {
		fmt.Printf("DATE/TIME: %s\n", dateTime)
	}
	if location != "" {
		fmt.Printf("LOCATION: %s\n", location)
	}
	if description != "" {
		fmt.Printf("DESCRIPTION: %s\n", description)
	}
	if len(ageGroups) > 0 {
		fmt.Printf("AGE GROUPS: %v\n", ageGroups)
	}
	if len(pricing) > 0 {
		fmt.Printf("PRICING: %v\n", pricing)
	}
	
	// Show raw lines for debugging
	fmt.Printf("RAW LINES:\n")
	for i, line := range eventLines {
		if i < 8 { // Show first 8 lines
			fmt.Printf("  %s\n", line)
		}
	}
}

func containsDatePattern(line string) bool {
	line = strings.ToLower(line)
	dateKeywords := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
		"jan", "feb", "mar", "apr", "may", "jun",
		"jul", "aug", "sep", "oct", "nov", "dec",
		"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
		"mon", "tue", "wed", "thu", "fri", "sat", "sun",
	}
	
	for _, keyword := range dateKeywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}
	
	// Check for numeric date patterns
	return strings.Contains(line, "/") && (strings.Contains(line, "2024") || strings.Contains(line, "2025"))
}

func containsTimePattern(line string) bool {
	line = strings.ToLower(line)
	timeKeywords := []string{"am", "pm", ":", "noon", "midnight", "morning", "afternoon", "evening"}
	
	for _, keyword := range timeKeywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}
	return false
}

func containsLocationPattern(line string) bool {
	line = strings.ToLower(line)
	locationKeywords := []string{
		"farm", "barn", "field", "pumpkin patch", "corn maze", "hayride",
		"carnation", "washington", "wa", "address", "location", "venue",
		"street", "avenue", "road", "way", "drive", "boulevard",
	}
	
	for _, keyword := range locationKeywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}
	return false
}

func containsAgePattern(line string) bool {
	line = strings.ToLower(line)
	ageKeywords := []string{
		"ages", "age", "toddler", "preschool", "elementary", "teen", "adult",
		"infant", "baby", "child", "kid", "family", "all ages",
	}
	
	for _, keyword := range ageKeywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}
	return false
}

func extractAgeGroups(line string) []string {
	var groups []string
	line = strings.ToLower(line)
	
	if strings.Contains(line, "toddler") {
		groups = append(groups, "toddlers")
	}
	if strings.Contains(line, "preschool") {
		groups = append(groups, "preschoolers")
	}
	if strings.Contains(line, "elementary") {
		groups = append(groups, "elementary")
	}
	if strings.Contains(line, "teen") {
		groups = append(groups, "teens")
	}
	if strings.Contains(line, "all ages") || strings.Contains(line, "family") {
		groups = append(groups, "all ages")
	}
	
	// Look for age ranges like "ages 3-5"
	ageRangePattern := regexp.MustCompile(`ages?\s+(\d+)[-–](\d+)`)
	if matches := ageRangePattern.FindStringSubmatch(line); len(matches) > 2 {
		groups = append(groups, fmt.Sprintf("ages %s-%s", matches[1], matches[2]))
	}
	
	return groups
}

func containsPricePattern(line string) bool {
	line = strings.ToLower(line)
	priceKeywords := []string{
		"$", "free", "cost", "price", "fee", "admission", "ticket", "donation",
	}
	
	for _, keyword := range priceKeywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}
	return false
}

func extractPricing(line string) []string {
	var prices []string
	
	if strings.Contains(strings.ToLower(line), "free") {
		prices = append(prices, "Free")
	}
	
	// Look for dollar amounts
	pricePattern := regexp.MustCompile(`\$\d+(?:\.\d{2})?`)
	if matches := pricePattern.FindAllString(line, -1); len(matches) > 0 {
		prices = append(prices, matches...)
	}
	
	return prices
}

func looksLikeTitle(line string) bool {
	words := strings.Fields(line)
	if len(words) < 2 || len(words) > 15 {
		return false
	}
	
	// Check if most words are capitalized
	capitalizedWords := 0
	for _, word := range words {
		if len(word) > 0 && strings.ToUpper(word[:1]) == word[:1] {
			capitalizedWords++
		}
	}
	
	return float64(capitalizedWords)/float64(len(words)) > 0.5
}