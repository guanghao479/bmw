package models

import (
	"fmt"
	"strings"
	"time"
)

// AdminEvent represents an event extracted via admin crawling that's pending approval
type AdminEvent struct {
	// DynamoDB Keys
	PK string `json:"pk" dynamodb:"PK"` // EVENT#{event_id}
	SK string `json:"sk" dynamodb:"SK"` // SUBMISSION#{timestamp}

	// Core Fields
	EventID            string                 `json:"event_id"`
	SourceURL          string                 `json:"source_url"`
	SchemaType         string                 `json:"schema_type"`         // "events"|"activities"|"venues"|"custom"
	SchemaUsed         map[string]interface{} `json:"schema_used"`         // Actual schema sent to Firecrawl
	RawExtractedData   map[string]interface{} `json:"raw_extracted_data"`  // Original Firecrawl response
	ConvertedData      map[string]interface{} `json:"converted_data"`      // Preview of Activity conversion
	ConversionIssues   []string               `json:"conversion_issues"`   // Validation warnings

	// Status and Review
	Status     AdminEventStatus `json:"status"`      // pending, approved, rejected, edited
	StatusKey  string           `json:"status_key"`  // GSI key for status queries
	AdminNotes string           `json:"admin_notes"` // Admin comments/notes

	// Timestamps
	ExtractedAt time.Time  `json:"extracted_at"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy  string     `json:"reviewed_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Metadata
	ExtractedByUser string `json:"extracted_by_user"` // Who submitted the crawl request
	SubmissionID    string `json:"submission_id"`     // Unique submission identifier
}

// AdminEventStatus represents the status of an admin event
type AdminEventStatus string

const (
	AdminEventStatusPending  AdminEventStatus = "pending"
	AdminEventStatusApproved AdminEventStatus = "approved"
	AdminEventStatusRejected AdminEventStatus = "rejected"
	AdminEventStatusEdited   AdminEventStatus = "edited"
)

// ExtractionSchema represents a predefined schema for Firecrawl extraction
type ExtractionSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema"`
	Examples    []string               `json:"examples,omitempty"`
}

// CrawlSubmissionRequest represents a request to crawl a website
type CrawlSubmissionRequest struct {
	URL              string                 `json:"url"`
	SchemaType       string                 `json:"schema_type"`         // "events"|"activities"|"venues"|"custom"
	CustomSchema     map[string]interface{} `json:"custom_schema,omitempty"` // Only used if schema_type = "custom"
	ExtractedByUser  string                 `json:"extracted_by_user"`
	AdminNotes       string                 `json:"admin_notes,omitempty"`
}

// DebugExtractionRequest represents a request for debug extraction
type DebugExtractionRequest struct {
	URL          string                 `json:"url"`
	SchemaType   string                 `json:"schema_type"`         // "events"|"activities"|"venues"|"custom"
	CustomSchema map[string]interface{} `json:"custom_schema,omitempty"` // Only used if schema_type = "custom"
}

// AdminEventReview represents a review action on an admin event
type AdminEventReview struct {
	Action     string                 `json:"action"`      // "approve"|"reject"|"edit"
	AdminNotes string                 `json:"admin_notes"` // Review comments
	EditedData map[string]interface{} `json:"edited_data,omitempty"` // Modified data if editing
	ReviewedBy string                 `json:"reviewed_by"`
}

// ConversionResult represents the result of converting raw data to Activity model
type ConversionResult struct {
	Activity         *Activity `json:"activity"`
	Issues           []string  `json:"issues"`
	FieldMappings    map[string]string `json:"field_mappings"`
	ConfidenceScore  float64   `json:"confidence_score"`
	DetailedMappings map[string]interface{} `json:"detailed_mappings,omitempty"` // Enhanced field mapping details
	ValidationResults map[string]interface{} `json:"validation_results,omitempty"` // Field validation results
}

// DynamoDB Key Generation Functions

// CreateAdminEventPK creates the primary key for an admin event
func CreateAdminEventPK(eventID string) string {
	return fmt.Sprintf("EVENT#%s", eventID)
}

// CreateAdminEventSK creates the sort key for an admin event
func CreateAdminEventSK(timestamp time.Time) string {
	return fmt.Sprintf("SUBMISSION#%s", timestamp.Format("2006-01-02T15:04:05Z"))
}

// GenerateAdminEventStatusKey creates a GSI key for querying by status
func GenerateAdminEventStatusKey(status AdminEventStatus) string {
	return fmt.Sprintf("STATUS#%s", string(status))
}

// Validation Functions

// Validate validates an admin event
func (ae *AdminEvent) Validate() error {
	if ae.EventID == "" {
		return fmt.Errorf("event_id is required")
	}
	if ae.SourceURL == "" {
		return fmt.Errorf("source_url is required")
	}
	if ae.SchemaType == "" {
		return fmt.Errorf("schema_type is required")
	}
	if ae.SchemaUsed == nil {
		return fmt.Errorf("schema_used is required")
	}
	if ae.RawExtractedData == nil {
		return fmt.Errorf("raw_extracted_data is required")
	}
	if ae.Status == "" {
		return fmt.Errorf("status is required")
	}
	if ae.ExtractedByUser == "" {
		return fmt.Errorf("extracted_by_user is required")
	}

	// Validate status
	switch ae.Status {
	case AdminEventStatusPending, AdminEventStatusApproved, AdminEventStatusRejected, AdminEventStatusEdited:
		// Valid statuses
	default:
		return fmt.Errorf("invalid status: %s", ae.Status)
	}

	// Validate schema type
	switch ae.SchemaType {
	case "events", "activities", "venues", "custom":
		// Valid schema types
	default:
		return fmt.Errorf("invalid schema_type: %s", ae.SchemaType)
	}

	return nil
}

// Validate validates a crawl submission request
func (csr *CrawlSubmissionRequest) Validate() error {
	if csr.URL == "" {
		return fmt.Errorf("url is required")
	}
	if csr.SchemaType == "" {
		return fmt.Errorf("schema_type is required")
	}
	if csr.ExtractedByUser == "" {
		return fmt.Errorf("extracted_by_user is required")
	}

	// Validate URL format
	if !strings.HasPrefix(csr.URL, "http://") && !strings.HasPrefix(csr.URL, "https://") {
		return fmt.Errorf("url must start with http:// or https://")
	}

	// Validate schema type
	switch csr.SchemaType {
	case "events", "activities", "venues", "custom":
		// Valid schema types
	default:
		return fmt.Errorf("invalid schema_type: %s", csr.SchemaType)
	}

	// If custom schema, validate it's provided
	if csr.SchemaType == "custom" && csr.CustomSchema == nil {
		return fmt.Errorf("custom_schema is required when schema_type is 'custom'")
	}

	return nil
}

// Helper Functions

// IsApproved returns true if the event has been approved
func (ae *AdminEvent) IsApproved() bool {
	return ae.Status == AdminEventStatusApproved
}

// IsRejected returns true if the event has been rejected
func (ae *AdminEvent) IsRejected() bool {
	return ae.Status == AdminEventStatusRejected
}

// IsPending returns true if the event is pending review
func (ae *AdminEvent) IsPending() bool {
	return ae.Status == AdminEventStatusPending || ae.Status == AdminEventStatusEdited
}

// CanBeApproved returns true if the event can be approved
func (ae *AdminEvent) CanBeApproved() bool {
	return ae.IsPending() && len(ae.ConversionIssues) == 0
}

// GetExtractedEventsCount returns the number of events extracted
func (ae *AdminEvent) GetExtractedEventsCount() int {
	// Try to count events in various possible structures
	if events, ok := ae.RawExtractedData["events"].([]interface{}); ok {
		return len(events)
	}
	if activities, ok := ae.RawExtractedData["activities"].([]interface{}); ok {
		return len(activities)
	}
	if venues, ok := ae.RawExtractedData["venues"].([]interface{}); ok {
		return len(venues)
	}

	// If it's a single item, return 1
	if ae.RawExtractedData != nil && len(ae.RawExtractedData) > 0 {
		return 1
	}

	return 0
}

// Predefined Extraction Schemas

// GetPredefinedSchemas returns the available predefined extraction schemas
func GetPredefinedSchemas() map[string]ExtractionSchema {
	return map[string]ExtractionSchema{
		"events": {
			Name:        "Events",
			Description: "Extract events with title, date, location, and pricing",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"events": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"title": map[string]interface{}{
									"type":        "string",
									"description": "The name or title of the event",
								},
								"description": map[string]interface{}{
									"type":        "string",
									"description": "A detailed description of the event",
								},
								"date": map[string]interface{}{
									"type":        "string",
									"description": "Event date in YYYY-MM-DD format",
								},
								"time": map[string]interface{}{
									"type":        "string",
									"description": "Event time in HH:MM format",
								},
								"location": map[string]interface{}{
									"type":        "string",
									"description": "Event location or venue name",
								},
								"address": map[string]interface{}{
									"type":        "string",
									"description": "Full address of the event location",
								},
								"price": map[string]interface{}{
									"type":        "string",
									"description": "Event price or 'Free' for free events",
								},
								"registration_url": map[string]interface{}{
									"type":        "string",
									"description": "URL for registration or more information",
								},
								"age_groups": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "string",
									},
									"description": "Target age groups like 'toddlers', 'elementary', 'teens', 'all ages'",
								},
							},
							"required": []string{"title", "location"},
						},
					},
				},
				"required": []string{"events"},
			},
			Examples: []string{
				"Event listing pages",
				"Community calendars",
				"Performance schedules",
				"Festival programs",
			},
		},
		"activities": {
			Name:        "Activities",
			Description: "Extract activities with name, age groups, duration, and details",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"activities": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type":        "string",
									"description": "The name of the activity or program",
								},
								"description": map[string]interface{}{
									"type":        "string",
									"description": "Detailed description of the activity",
								},
								"age_groups": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "string",
									},
									"description": "Target age groups for the activity",
								},
								"duration": map[string]interface{}{
									"type":        "string",
									"description": "Duration of the activity (e.g., '45 minutes', '2 hours')",
								},
								"schedule": map[string]interface{}{
									"type":        "string",
									"description": "When the activity runs (e.g., 'Mondays 10 AM')",
								},
								"location": map[string]interface{}{
									"type":        "string",
									"description": "Where the activity takes place",
								},
								"cost": map[string]interface{}{
									"type":        "string",
									"description": "Cost of the activity or 'Free'",
								},
								"instructor": map[string]interface{}{
									"type":        "string",
									"description": "Name of the instructor or facilitator",
								},
								"registration_required": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether registration is required",
								},
							},
							"required": []string{"name", "age_groups"},
						},
					},
				},
				"required": []string{"activities"},
			},
			Examples: []string{
				"Class listings",
				"Program catalogs",
				"Activity schedules",
				"Camp programs",
			},
		},
		"venues": {
			Name:        "Venues",
			Description: "Extract venue information with name, address, and facilities",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"venues": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type":        "string",
									"description": "The name of the venue",
								},
								"address": map[string]interface{}{
									"type":        "string",
									"description": "Full address of the venue",
								},
								"phone": map[string]interface{}{
									"type":        "string",
									"description": "Contact phone number",
								},
								"website": map[string]interface{}{
									"type":        "string",
									"description": "Venue website URL",
								},
								"description": map[string]interface{}{
									"type":        "string",
									"description": "Description of the venue and its offerings",
								},
								"facilities": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "string",
									},
									"description": "Available facilities (e.g., 'playground', 'parking', 'restrooms')",
								},
								"age_suitability": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "string",
									},
									"description": "Age groups the venue is suitable for",
								},
								"admission_fee": map[string]interface{}{
									"type":        "string",
									"description": "Admission fee or 'Free'",
								},
							},
							"required": []string{"name", "address"},
						},
					},
				},
				"required": []string{"venues"},
			},
			Examples: []string{
				"Museum listings",
				"Park directories",
				"Recreation center pages",
				"Venue directories",
			},
		},
	}
}

// GetSchemaByType returns a predefined schema by type
func GetSchemaByType(schemaType string) (ExtractionSchema, error) {
	schemas := GetPredefinedSchemas()
	schema, exists := schemas[schemaType]
	if !exists {
		return ExtractionSchema{}, fmt.Errorf("unknown schema type: %s", schemaType)
	}
	return schema, nil
}