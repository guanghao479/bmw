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
      "schedule": {
        "type": "one-time|recurring|multi-day|ongoing",
        "start_date": "YYYY-MM-DD",
        "end_date": "YYYY-MM-DD",
        "frequency": "weekly|daily|monthly",
        "days_of_week": ["monday", "tuesday"],
        "times": [
          {
            "start_time": "HH:MM",
            "end_time": "HH:MM",
            "age_group": "age group from list"
          }
        ],
        "duration": "45 minutes",
        "sessions": 8
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
        "address": "Full address including Seattle, WA",
        "neighborhood": "Capitol Hill",
        "city": "Seattle",
        "region": "Seattle Metro",
        "zip_code": "98101",
        "venue_type": "indoor|outdoor|mixed",
        "accessibility": true,
        "parking": "free|paid|street|none"
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
        "url": "registration URL",
        "status": "open|closed|waitlist|full"
      },
      "tags": ["music", "toddler", "parent-child"],
      "provider": {
        "name": "Organization Name",
        "type": "business|nonprofit|government|library",
        "website": "website URL",
        "verified": true
      }
    }
  ]
}

EXTRACTION RULES:
- Extract complete, accurate information
- Don't make up details not present in the content
- Use "TBD" or leave empty if information is not available
- Ensure all locations are in Seattle metro area
- Include registration URLs when provided
- Extract pricing information accurately
- Pay attention to age requirements and restrictions
- Note if activities require advance registration
- Include any special requirements or supplies needed

Focus on accuracy over quantity. It's better to extract fewer, complete activities than many incomplete ones.`
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
	}
	
	return issues
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