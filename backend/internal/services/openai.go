package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"seattle-family-activities-scraper/internal/models"
)

// OpenAIClient handles activity extraction using OpenAI
type OpenAIClient struct {
	client      *openai.Client
	model       string
	temperature float32
	maxTokens   int
}

// OpenAIExtractionResponse represents the response from OpenAI
type OpenAIExtractionResponse struct {
	Activities    []models.Activity `json:"activities"`
	TotalFound    int              `json:"total_found"`
	ProcessingMS  int64            `json:"processing_ms"`
	TokensUsed    int              `json:"tokens_used"`
	EstimatedCost float64          `json:"estimated_cost"`
	SourceURL     string           `json:"source_url"`
	ExtractionID  string           `json:"extraction_id"`
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient() *OpenAIClient {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	return &OpenAIClient{
		client:      openai.NewClient(apiKey),
		model:       "gpt-4o-mini",
		temperature: 0.1,
		maxTokens:   4000,
	}
}

// NewOpenAIClientWithConfig creates a new OpenAI client with custom configuration
func NewOpenAIClientWithConfig(model string, temperature float32, maxTokens int) *OpenAIClient {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	return &OpenAIClient{
		client:      openai.NewClient(apiKey),
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}
}

// ExtractActivities extracts structured activities from webpage content
func (o *OpenAIClient) ExtractActivities(content, sourceURL string) (*OpenAIExtractionResponse, error) {
	startTime := time.Now()

	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	if len(content) < 200 {
		return nil, fmt.Errorf("content too short (%d chars) to extract meaningful activities", len(content))
	}

	// Generate extraction ID for tracking
	extractionID := fmt.Sprintf("ext_%d", time.Now().Unix())

	// Create the system prompt with Seattle-specific guidelines
	systemPrompt := o.buildSeattleSystemPrompt()

	// Create the user prompt with the content
	userPrompt := o.buildUserPrompt(content, sourceURL)

	// Make the OpenAI request
	resp, err := o.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       o.model,
			Temperature: o.temperature,
			MaxTokens:   o.maxTokens,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices from OpenAI")
	}

	// Parse the JSON response
	responseContent := resp.Choices[0].Message.Content
	
	// Clean the response - remove markdown code blocks if present
	cleanedContent := o.cleanJSONResponse(responseContent)
	
	var activitiesData struct {
		Activities []models.Activity `json:"activities"`
	}

	err = json.Unmarshal([]byte(cleanedContent), &activitiesData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response JSON: %w\nResponse: %s", err, cleanedContent)
	}

	// Calculate processing metrics
	processingTime := time.Since(startTime).Milliseconds()
	tokensUsed := resp.Usage.TotalTokens
	estimatedCost := o.calculateCost(tokensUsed)

	// Set source information on all activities
	for i := range activitiesData.Activities {
		activitiesData.Activities[i].Source.URL = sourceURL
		activitiesData.Activities[i].Source.Domain = o.extractDomain(sourceURL)
		activitiesData.Activities[i].Source.ScrapedAt = time.Now()
		activitiesData.Activities[i].Source.LastChecked = time.Now()
		activitiesData.Activities[i].Source.Reliability = "high"
		
		// Set timestamps
		activitiesData.Activities[i].CreatedAt = time.Now()
		activitiesData.Activities[i].UpdatedAt = time.Now()
		activitiesData.Activities[i].Status = models.ActivityStatusActive
		
		// Generate ID if not set
		if activitiesData.Activities[i].ID == "" {
			activitiesData.Activities[i].ID = models.GenerateActivityID(
				activitiesData.Activities[i].Title,
				activitiesData.Activities[i].Schedule.StartDate,
				activitiesData.Activities[i].Location.Name,
			)
		}
	}

	return &OpenAIExtractionResponse{
		Activities:    activitiesData.Activities,
		TotalFound:    len(activitiesData.Activities),
		ProcessingMS:  processingTime,
		TokensUsed:    tokensUsed,
		EstimatedCost: estimatedCost,
		SourceURL:     sourceURL,
		ExtractionID:  extractionID,
	}, nil
}

// buildSeattleSystemPrompt creates the system prompt for Seattle family activities
func (o *OpenAIClient) buildSeattleSystemPrompt() string {
	return `You are an expert at extracting structured data about Seattle-area family activities, events, and venues from web content.

Your task is to analyze the provided content and extract family-friendly activities, classes, camps, events, and venues in the Seattle metropolitan area.

CRITICAL DATA EXTRACTION REQUIREMENTS:

🖼️ IMAGE EXTRACTION (HIGH PRIORITY):
- Extract ALL image URLs from the content, prioritizing:
  1. Event-specific photos (highest priority)
  2. Venue/location photos (medium priority)
  3. Activity/program photos (low priority)
- For each image, provide:
  - Full URL (convert relative paths to absolute using source domain)
  - Alt text for accessibility
  - Source type: "event", "venue", "activity", or "gallery"
- Source-specific patterns:
  - ParentMap: Look for .event_square class images
  - Tinybeans: Handle lazy-loaded images with srcset attributes
  - Macaroni KID: Extract event listing thumbnails
- Validation requirements:
  - Ensure URLs are accessible and properly formed
  - Prefer images >300px width when size info available
  - Avoid social media icons, logos, or navigation images
  - Include multiple images per activity when available

📅 DATE/TIME EXTRACTION (HIGH PRIORITY):
- ALWAYS normalize to Pacific Time Zone (Seattle timezone)
- Handle partial dates (e.g., "Aug 15") by inferring current or next year
- Extract specific start and end times when available
- For multi-day events, extract both start and end dates
- Identify recurring patterns (daily, weekly, monthly)
- Parse duration when explicitly mentioned (e.g., "2 hours", "90 minutes")
- Mark all-day events appropriately with is_all_day flag
- Convert time formats to 24-hour HH:MM format
- Examples:
  - "Aug 15, 2-4 PM" → start_date: "2025-08-15", start_time: "14:00", end_time: "16:00"
  - "Every Saturday 10 AM" → recurring: weekly, days_of_week: ["saturday"], start_time: "10:00"

🔗 LINK EXTRACTION (HIGH PRIORITY):
- Extract primary event detail page URLs
- Find registration/ticket purchase links
- Identify contact information (website, email, phone)
- Source-specific patterns:
  - ParentMap: Calendar event detail pages
  - Tinybeans: City-specific event URLs (/seattle/)
  - Macaroni KID: Event submission and detail links
- Ensure all URLs are properly formed and accessible
- Extract deep-link URLs to specific event pages, not just home pages

📍 LOCATION ENHANCEMENT (HIGH PRIORITY):
- Extract complete addresses including street, city, state, zip
- Identify specific venue names and types (indoor/outdoor/mixed)
- Extract neighborhood information when available
- Look for accessibility information (wheelchair accessible, parking details)
- Parse venue amenities (parking: free/paid/street, public transit info)
- Convert addresses to specific Seattle-area neighborhoods when possible

IMPORTANT GUIDELINES:
1. Only extract activities that are explicitly located in the Seattle metro area (Seattle, Bellevue, Redmond, Bothell, Kirkland, Renton, Federal Way, Tacoma, Everett, etc.)
2. Focus on activities suitable for families with children (ages 0-18)
3. Include both free and paid activities
4. Extract activities from all categories: arts, sports, education, entertainment, camps, community events

ACTIVITY TYPES:
- "class": Ongoing classes or lessons (music, art, sports, etc.)
- "camp": Day camps, summer camps, holiday camps
- "event": One-time events, performances, festivals
- "performance": Shows, concerts, theater for families
- "free-activity": Free community activities, library events, park programs

CATEGORIES:
- "arts-creativity": Art classes, music, theater, crafts
- "active-sports": Sports, swimming, gymnastics, martial arts
- "educational-stem": Science classes, coding, robotics, museums
- "entertainment-events": Shows, festivals, family entertainment
- "camps-programs": All types of camps and extended programs
- "free-community": Free library events, park activities, community programs

AGE GROUPS (use these exact values):
- "infant": 0-12 months
- "toddler": 1-3 years
- "preschool": 3-5 years
- "elementary": 6-10 years
- "tween": 11-14 years
- "teen": 15-18 years
- "adult": 18+ years
- "all-ages": All ages welcome

FAMILY TYPES:
- "parent-child": Requires parent participation
- "drop-off": Child attends independently
- "family": Designed for whole family participation

OUTPUT FORMAT:
Return a JSON object with this exact structure:
{
  "activities": [
    {
      "id": "",
      "title": "Activity Name",
      "description": "Detailed description",
      "type": "class|camp|event|performance|free-activity",
      "category": "category from list above",
      "subcategory": "specific subcategory if applicable",
      "images": [
        {
          "url": "https://example.com/image.jpg",
          "alt_text": "Description of image for accessibility",
          "source_type": "event|venue|activity|gallery",
          "width": 400,
          "height": 300
        }
      ],
      "schedule": {
        "type": "one-time|recurring|multi-day|ongoing",
        "start_date": "YYYY-MM-DD",
        "end_date": "YYYY-MM-DD",
        "start_time": "HH:MM",
        "end_time": "HH:MM",
        "timezone": "America/Los_Angeles",
        "is_all_day": false,
        "frequency": "weekly|daily|monthly",
        "days_of_week": ["monday", "tuesday"],
        "duration": "45 minutes",
        "sessions": 8,
        "times": [
          {
            "start_time": "HH:MM",
            "end_time": "HH:MM",
            "age_group": "age group from list"
          }
        ]
      },
      "age_groups": [
        {
          "category": "age group from list",
          "min_age": 24,
          "max_age": 48,
          "unit": "months|years",
          "description": "2-4 years"
        }
      ],
      "family_type": "parent-child|drop-off|family",
      "location": {
        "name": "Venue Name",
        "address": "Full street address",
        "city": "Seattle",
        "state": "WA",
        "zip_code": "98101",
        "neighborhood": "Capitol Hill",
        "region": "Seattle Metro",
        "latitude": 47.6062,
        "longitude": -122.3321,
        "venue_type": "indoor|outdoor|mixed",
        "accessibility": "wheelchair accessible, elevator available",
        "parking": "free|paid|street|none",
        "public_transit": "Metro bus lines 10, 11, 49"
      },
      "pricing": {
        "type": "free|paid|donation|variable",
        "cost": 0.0,
        "currency": "USD",
        "unit": "session|class|month|drop-in",
        "description": "Free with registration",
        "includes_supplies": true
      },
      "registration": {
        "required": true,
        "method": "online|phone|in-person|walk-in",
        "url": "https://example.com/register",
        "deadline": "YYYY-MM-DD",
        "status": "open|closed|waitlist|full",
        "contact_phone": "(206) 555-0123",
        "contact_email": "info@example.com"
      },
      "detail_url": "https://example.com/event-details",
      "tags": ["music", "toddler", "parent-child"],
      "provider": {
        "name": "Organization Name",
        "type": "business|nonprofit|government|library",
        "website": "https://example.com",
        "verified": true
      }
    }
  ]
}

EXTRACTION RULES:

🎯 PRIORITIZED EXTRACTION (CRITICAL):
1. **Images**: Extract ALL available images - this is the #1 data quality priority
2. **Event Detail URLs**: Extract specific event page links, not just home pages
3. **Precise Dates/Times**: Convert all times to Pacific timezone with specific start/end times
4. **Complete Addresses**: Include street address, city, state, zip when available
5. **Registration Information**: Extract actual registration URLs and contact details

📋 STANDARD RULES:
- Extract complete, accurate information from the actual content
- Don't make up details not present in the content
- Use "TBD" or leave empty if information is truly not available
- Ensure all locations are in Seattle metro area
- Include registration URLs when provided - validate they are real URLs
- Extract pricing information accurately including any discounts or special offers
- Pay attention to age requirements and restrictions
- Note if activities require advance registration or have deadlines
- Include any special requirements or supplies needed
- Convert relative image URLs to absolute URLs using the source domain

🔍 DATA QUALITY VALIDATION:
- All image URLs must be complete and properly formatted
- All dates must be in YYYY-MM-DD format
- All times must be in 24-hour HH:MM format
- Location coordinates should be estimated based on Seattle area knowledge when addresses are provided
- Registration URLs must be complete and accessible links
- Phone numbers should be formatted consistently: (206) 555-0123

⚡ EXTRACTION PRIORITY:
Focus on accuracy and completeness over quantity. It's better to extract fewer activities with complete, high-quality data (especially images, precise times, and working links) than many incomplete ones. Each activity should have at least one image if any are available in the content.`
}

// buildUserPrompt creates the user prompt with content and source URL
func (o *OpenAIClient) buildUserPrompt(content, sourceURL string) string {
	return fmt.Sprintf(`Please analyze the following web content from %s and extract all Seattle-area family activities, events, classes, camps, and venues.

Source URL: %s

Content to analyze:
%s

Extract the activities as structured JSON following the schema provided in the system prompt. Focus on accuracy and completeness.`, sourceURL, sourceURL, content)
}

// extractDomain extracts domain from URL
func (o *OpenAIClient) extractDomain(url string) string {
	if url == "" {
		return ""
	}
	
	// Remove protocol
	domain := strings.TrimPrefix(url, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	
	// Remove path
	if idx := strings.Index(domain, "/"); idx >= 0 {
		domain = domain[:idx]
	}
	
	// Remove www prefix
	domain = strings.TrimPrefix(domain, "www.")
	
	return domain
}

// calculateCost estimates the cost based on tokens used
func (o *OpenAIClient) calculateCost(tokensUsed int) float64 {
	// GPT-4o-mini pricing (as of 2024)
	// Input: $0.00015 per 1K tokens
	// Output: $0.0006 per 1K tokens
	// We'll use a blended rate of ~$0.0003 per 1K tokens for estimation
	return float64(tokensUsed) * 0.0003 / 1000.0
}

// ValidateExtractionResponse validates the extracted activities
func (o *OpenAIClient) ValidateExtractionResponse(response *OpenAIExtractionResponse) []string {
	var issues []string
	
	if response == nil {
		return []string{"response is nil"}
	}
	
	if len(response.Activities) == 0 {
		issues = append(issues, "no activities extracted")
	}
	
	for i, activity := range response.Activities {
		prefix := fmt.Sprintf("Activity %d (%s):", i+1, activity.Title)
		
		// Validate required fields
		if activity.Title == "" {
			issues = append(issues, prefix+" missing title")
		}
		
		if activity.Type == "" {
			issues = append(issues, prefix+" missing type")
		} else if !models.ValidateActivityType(activity.Type) {
			issues = append(issues, prefix+" invalid activity type: "+activity.Type)
		}
		
		if activity.Category == "" {
			issues = append(issues, prefix+" missing category")
		} else if !models.ValidateCategory(activity.Category) {
			issues = append(issues, prefix+" invalid category: "+activity.Category)
		}
		
		// Validate location (must be Seattle area)
		if activity.Location.City == "" {
			issues = append(issues, prefix+" missing city")
		} else if !o.isSeattleArea(activity.Location.City) {
			issues = append(issues, prefix+" not in Seattle area: "+activity.Location.City)
		}
		
		// Validate age groups
		for j, ageGroup := range activity.AgeGroups {
			if !models.ValidateAgeGroup(ageGroup.Category) {
				issues = append(issues, fmt.Sprintf("%s age group %d invalid: %s", prefix, j+1, ageGroup.Category))
			}
		}
		
		// Validate pricing
		if activity.Pricing.Type == "" {
			issues = append(issues, prefix+" missing pricing type")
		} else if !models.ValidatePricingType(activity.Pricing.Type) {
			issues = append(issues, prefix+" invalid pricing type: "+activity.Pricing.Type)
		}
		
		// Validate images
		for j, image := range activity.Images {
			imagePrefix := fmt.Sprintf("%s image %d:", prefix, j+1)
			if image.URL == "" {
				issues = append(issues, imagePrefix+" missing image URL")
			} else if !models.ValidateImageURL(image.URL) {
				issues = append(issues, imagePrefix+" invalid image URL: "+image.URL)
			}
			
			if image.SourceType != "" && !models.ValidateImageSourceType(image.SourceType) {
				issues = append(issues, imagePrefix+" invalid source type: "+image.SourceType)
			}
		}
		
		// Validate venue type
		if activity.Location.VenueType != "" && !models.ValidateVenueType(activity.Location.VenueType) {
			issues = append(issues, prefix+" invalid venue type: "+activity.Location.VenueType)
		}
		
		// Validate detail URL
		if activity.DetailURL != "" && !models.IsValidURL(activity.DetailURL) {
			issues = append(issues, prefix+" invalid detail URL: "+activity.DetailURL)
		}
		
		// Validate registration URL
		if activity.Registration.URL != "" && !models.IsValidURL(activity.Registration.URL) {
			issues = append(issues, prefix+" invalid registration URL: "+activity.Registration.URL)
		}
		
		// Validate contact information
		if activity.Registration.ContactEmail != "" && !models.IsValidEmail(activity.Registration.ContactEmail) {
			issues = append(issues, prefix+" invalid contact email: "+activity.Registration.ContactEmail)
		}
		
		if activity.Registration.ContactPhone != "" && !models.IsValidPhoneNumber(activity.Registration.ContactPhone) {
			issues = append(issues, prefix+" invalid contact phone: "+activity.Registration.ContactPhone)
		}
	}
	
	return issues
}

// CalculateQualityScore calculates a data quality score for extracted activities
func (o *OpenAIClient) CalculateQualityScore(response *OpenAIExtractionResponse) float64 {
	if response == nil || len(response.Activities) == 0 {
		return 0.0
	}
	
	totalScore := 0.0
	maxPossibleScore := 0.0
	
	for _, activity := range response.Activities {
		activityScore, maxScore := o.calculateActivityQualityScore(activity)
		totalScore += activityScore
		maxPossibleScore += maxScore
	}
	
	if maxPossibleScore == 0 {
		return 0.0
	}
	
	return (totalScore / maxPossibleScore) * 100.0 // Return as percentage
}

// calculateActivityQualityScore calculates quality score for a single activity
func (o *OpenAIClient) calculateActivityQualityScore(activity models.Activity) (float64, float64) {
	score := 0.0
	maxScore := 0.0
	
	// Core required fields (20 points each)
	coreFields := []struct {
		name   string
		points float64
		hasValue bool
	}{
		{"title", 20.0, activity.Title != ""},
		{"description", 15.0, activity.Description != ""},
		{"type", 20.0, activity.Type != ""},
		{"category", 15.0, activity.Category != ""},
		{"city", 20.0, activity.Location.City != ""},
	}
	
	for _, field := range coreFields {
		maxScore += field.points
		if field.hasValue {
			score += field.points
		}
	}
	
	// Image quality (25 points total)
	maxScore += 25.0
	if len(activity.Images) > 0 {
		imageScore := 0.0
		for _, img := range activity.Images {
			if img.URL != "" && models.ValidateImageURL(img.URL) {
				imageScore += 8.0 // 8 points per valid image
				if img.AltText != "" {
					imageScore += 2.0 // Bonus for alt text
				}
			}
		}
		// Cap image score at 25 points
		if imageScore > 25.0 {
			imageScore = 25.0
		}
		score += imageScore
	}
	
	// Date/Time quality (20 points total)
	maxScore += 20.0
	if activity.Schedule.StartDate != "" {
		score += 10.0
		if activity.Schedule.StartTime != "" {
			score += 5.0
		}
		if activity.Schedule.Timezone != "" {
			score += 3.0
		}
		if !activity.Schedule.IsAllDay && activity.Schedule.EndTime != "" {
			score += 2.0
		}
	}
	
	// Location quality (15 points total)
	maxScore += 15.0
	if activity.Location.Address != "" {
		score += 5.0
	}
	if activity.Location.Coordinates.Lat != 0 && activity.Location.Coordinates.Lng != 0 {
		score += 5.0
	}
	if activity.Location.VenueType != "" {
		score += 3.0
	}
	if activity.Location.Parking != "" || activity.Location.PublicTransit != "" {
		score += 2.0
	}
	
	// Registration/Contact quality (10 points total)
	maxScore += 10.0
	if activity.Registration.URL != "" && models.IsValidURL(activity.Registration.URL) {
		score += 5.0
	}
	if activity.DetailURL != "" && models.IsValidURL(activity.DetailURL) {
		score += 3.0
	}
	if activity.Registration.ContactPhone != "" || activity.Registration.ContactEmail != "" {
		score += 2.0
	}
	
	// Age group specificity (5 points total)
	maxScore += 5.0
	if len(activity.AgeGroups) > 0 {
		score += 3.0
		// Bonus for specific age ranges
		for _, ageGroup := range activity.AgeGroups {
			if ageGroup.MinAge > 0 || ageGroup.MaxAge > 0 {
				score += 2.0
				break
			}
		}
	}
	
	return score, maxScore
}

// GenerateQualityReport generates a detailed quality report for extracted data
func (o *OpenAIClient) GenerateQualityReport(response *OpenAIExtractionResponse) map[string]interface{} {
	report := map[string]interface{}{
		"overall_score": o.CalculateQualityScore(response),
		"total_activities": len(response.Activities),
		"quality_breakdown": map[string]interface{}{
			"with_images": 0,
			"with_coordinates": 0,
			"with_specific_times": 0,
			"with_registration_url": 0,
			"with_detail_url": 0,
			"with_contact_info": 0,
		},
		"common_issues": []string{},
	}
	
	breakdown := report["quality_breakdown"].(map[string]interface{})
	var issues []string
	
	for _, activity := range response.Activities {
		// Count quality indicators
		if len(activity.Images) > 0 {
			breakdown["with_images"] = breakdown["with_images"].(int) + 1
		}
		
		if activity.Location.Coordinates.Lat != 0 && activity.Location.Coordinates.Lng != 0 {
			breakdown["with_coordinates"] = breakdown["with_coordinates"].(int) + 1
		}
		
		if activity.Schedule.StartTime != "" {
			breakdown["with_specific_times"] = breakdown["with_specific_times"].(int) + 1
		}
		
		if activity.Registration.URL != "" && models.IsValidURL(activity.Registration.URL) {
			breakdown["with_registration_url"] = breakdown["with_registration_url"].(int) + 1
		}
		
		if activity.DetailURL != "" && models.IsValidURL(activity.DetailURL) {
			breakdown["with_detail_url"] = breakdown["with_detail_url"].(int) + 1
		}
		
		if activity.Registration.ContactPhone != "" || activity.Registration.ContactEmail != "" {
			breakdown["with_contact_info"] = breakdown["with_contact_info"].(int) + 1
		}
		
		// Track common issues
		if len(activity.Images) == 0 {
			issues = append(issues, "Missing images")
		}
		if activity.Location.Coordinates.Lat == 0 && activity.Location.Coordinates.Lng == 0 {
			issues = append(issues, "Missing coordinates")
		}
		if activity.Schedule.StartTime == "" {
			issues = append(issues, "Missing specific start time")
		}
	}
	
	// Remove duplicates from issues
	uniqueIssues := make(map[string]bool)
	for _, issue := range issues {
		uniqueIssues[issue] = true
	}
	
	var finalIssues []string
	for issue := range uniqueIssues {
		finalIssues = append(finalIssues, issue)
	}
	
	report["common_issues"] = finalIssues
	return report
}

// isSeattleArea checks if a city is in the Seattle metropolitan area
func (o *OpenAIClient) isSeattleArea(city string) bool {
	seattleAreaCities := []string{
		"seattle", "bellevue", "redmond", "kirkland", "bothell", "woodinville",
		"renton", "tukwila", "federal way", "kent", "auburn", "burien",
		"des moines", "seatac", "normandy park", "mercer island",
		"issaquah", "sammamish", "snoqualmie", "north bend",
		"lynnwood", "edmonds", "mukilteo", "mill creek", "everett",
		"shoreline", "lake forest park", "mountlake terrace", "brier",
		"tacoma", "lakewood", "university place", "fife", "puyallup",
		"sumner", "bonney lake", "enumclaw", "black diamond",
		"vashon", "bainbridge island", "poulsbo", "bremerton",
	}
	
	cityLower := strings.ToLower(strings.TrimSpace(city))
	
	for _, validCity := range seattleAreaCities {
		if cityLower == validCity {
			return true
		}
	}
	
	return false
}

// GetModel returns the current OpenAI model being used
func (o *OpenAIClient) GetModel() string {
	return o.model
}

// SetModel sets the OpenAI model to use
func (o *OpenAIClient) SetModel(model string) {
	o.model = model
}

// GetTemperature returns the current temperature setting
func (o *OpenAIClient) GetTemperature() float32 {
	return o.temperature
}

// SetTemperature sets the temperature for OpenAI requests
func (o *OpenAIClient) SetTemperature(temperature float32) {
	o.temperature = temperature
}

// GetMaxTokens returns the current max tokens setting
func (o *OpenAIClient) GetMaxTokens() int {
	return o.maxTokens
}

// SetMaxTokens sets the max tokens for OpenAI requests
func (o *OpenAIClient) SetMaxTokens(maxTokens int) {
	o.maxTokens = maxTokens
}

// cleanJSONResponse removes markdown code blocks and other formatting from OpenAI response
func (o *OpenAIClient) cleanJSONResponse(response string) string {
	// Remove markdown code blocks
	cleaned := strings.TrimSpace(response)
	
	// Remove ```json prefix
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
	}
	
	// Remove ``` prefix (in case it's just ```)
	if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
	}
	
	// Remove ``` suffix
	if strings.HasSuffix(cleaned, "```") {
		cleaned = strings.TrimSuffix(cleaned, "```")
	}
	
	// Trim whitespace again
	cleaned = strings.TrimSpace(cleaned)
	
	return cleaned
}