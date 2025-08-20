package models

import "time"

// Entity type constants for the family-activities table
const (
	EntityTypeVenue      = "VENUE"
	EntityTypeEvent      = "EVENT" 
	EntityTypeProgram    = "PROGRAM"
	EntityTypeAttraction = "ATTRACTION"
)

// Sort key constants
const (
	SortKeyMetadata = "METADATA"
	SortKeyInstance = "INSTANCE"
)

// FamilyActivity represents the base structure for all family activity entities
type FamilyActivity struct {
	// Primary Keys
	PK string `json:"PK" dynamodbav:"PK"` // Partition Key: ENTITY_TYPE#id
	SK string `json:"SK" dynamodbav:"SK"` // Sort Key: METADATA or INSTANCE#timestamp

	// Core Entity Information
	EntityType string `json:"entity_type" dynamodbav:"entity_type"` // venue, event, program, attraction
	EntityID   string `json:"entity_id" dynamodbav:"entity_id"`     // Unique identifier for the entity

	// Common Fields
	Name        string `json:"name" dynamodbav:"name"`
	Description string `json:"description" dynamodbav:"description"`
	Category    string `json:"category" dynamodbav:"category"`       // arts-creativity, active-sports, etc.
	Subcategory string `json:"subcategory" dynamodbav:"subcategory"` // music, soccer, science, etc.

	// Location Information
	Location ActivityLocation `json:"location" dynamodbav:"location"`

	// Age Groups
	AgeGroups []AgeGroup `json:"age_groups" dynamodbav:"age_groups"`

	// Pricing
	Pricing ActivityPricing `json:"pricing" dynamodbav:"pricing"`

	// Provider Information
	ProviderID   string `json:"provider_id" dynamodbav:"provider_id"`
	ProviderName string `json:"provider_name" dynamodbav:"provider_name"`

	// Status and Metadata
	Status    string    `json:"status" dynamodbav:"status"`       // active, inactive, cancelled
	Featured  bool      `json:"featured" dynamodbav:"featured"`
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt time.Time `json:"updated_at" dynamodbav:"updated_at"`

	// Source Tracking
	SourceID string `json:"source_id" dynamodbav:"source_id"`

	// GSI Keys (computed fields for efficient querying)
	LocationKey      string `json:"LocationKey,omitempty" dynamodbav:"LocationKey,omitempty"`           // GEO#{region}#{city}
	DateTypeKey      string `json:"DateTypeKey,omitempty" dynamodbav:"DateTypeKey,omitempty"`           // DATE#{date}#TYPE#{entity_type}#{entity_id}
	CategoryAgeKey   string `json:"CategoryAgeKey,omitempty" dynamodbav:"CategoryAgeKey,omitempty"`     // CAT#{category}#{age_group}
	DateFeaturedKey  string `json:"DateFeaturedKey,omitempty" dynamodbav:"DateFeaturedKey,omitempty"`   // DATE#{date}#FEATURED#{featured}#{entity_id}
	VenueKey         string `json:"VenueKey,omitempty" dynamodbav:"VenueKey,omitempty"`                 // VENUE#{venue_id}
	TypeDateKey      string `json:"TypeDateKey,omitempty" dynamodbav:"TypeDateKey,omitempty"`           // TYPE#{entity_type}#{start_date}#{entity_id}
	ProviderKey      string `json:"ProviderKey,omitempty" dynamodbav:"ProviderKey,omitempty"`           // PROVIDER#{provider_id}
	TypeStatusKey    string `json:"TypeStatusKey,omitempty" dynamodbav:"TypeStatusKey,omitempty"`       // TYPE#{entity_type}#STATUS#{status}#{entity_id}
}

// Venue represents a physical location where activities take place
type Venue struct {
	FamilyActivity

	// Venue-specific fields
	VenueName       string            `json:"venue_name" dynamodbav:"venue_name"`
	VenueType       string            `json:"venue_type" dynamodbav:"venue_type"`           // indoor, outdoor, mixed
	Address         string            `json:"address" dynamodbav:"address"`
	Coordinates     Coordinates       `json:"coordinates" dynamodbav:"coordinates"`
	Region          string            `json:"region" dynamodbav:"region"`                   // seattle-downtown, eastside, etc.
	Amenities       []string          `json:"amenities" dynamodbav:"amenities"`             // parking, restrooms, food, accessibility
	OperatingHours  map[string]string `json:"operating_hours" dynamodbav:"operating_hours"` // monday: "10:00-22:00"
	ContactInfo     ContactInfo       `json:"contact_info" dynamodbav:"contact_info"`
	Website         string            `json:"website" dynamodbav:"website"`
}

// Event represents a time-bound happening
type Event struct {
	FamilyActivity

	// Event-specific fields
	EventName string    `json:"event_name" dynamodbav:"event_name"`
	EventType string    `json:"event_type" dynamodbav:"event_type"` // festival, workshop, performance, etc.
	VenueID   string    `json:"venue_id" dynamodbav:"venue_id"`     // Reference to venue
	Schedule  Schedule  `json:"schedule" dynamodbav:"schedule"`
	Registration Registration `json:"registration" dynamodbav:"registration"`
	Images    []Image   `json:"images" dynamodbav:"images"`
	DetailURL string    `json:"detail_url" dynamodbav:"detail_url"`
	Tags      []string  `json:"tags" dynamodbav:"tags"`
}

// Program represents recurring structured activities
type Program struct {
	FamilyActivity

	// Program-specific fields
	ProgramName  string       `json:"program_name" dynamodbav:"program_name"`
	ProgramType  string       `json:"program_type" dynamodbav:"program_type"` // class, camp, league, etc.
	VenueID      string       `json:"venue_id" dynamodbav:"venue_id"`         // Reference to venue
	Schedule     Schedule     `json:"schedule" dynamodbav:"schedule"`
	Registration Registration `json:"registration" dynamodbav:"registration"`
	SessionCount int          `json:"session_count" dynamodbav:"session_count"`
	Duration     string       `json:"duration" dynamodbav:"duration"` // "45 minutes", "2 hours"
}

// ProgramInstance represents individual sessions of a recurring program
type ProgramInstance struct {
	// Primary Keys
	PK string `json:"PK" dynamodbav:"PK"` // PROGRAM#{program_id}
	SK string `json:"SK" dynamodbav:"SK"` // INSTANCE#{date}T{time}

	// Instance Details
	ProgramID         string    `json:"program_id" dynamodbav:"program_id"`
	InstanceDate      string    `json:"instance_date" dynamodbav:"instance_date"`         // YYYY-MM-DD
	InstanceTime      string    `json:"instance_time" dynamodbav:"instance_time"`         // HH:MM-HH:MM
	Status            string    `json:"status" dynamodbav:"status"`                       // scheduled, cancelled, full, waitlist
	RegistrationStatus string   `json:"registration_status" dynamodbav:"registration_status"` // open, closed, waitlist
	CurrentEnrollment int       `json:"current_enrollment" dynamodbav:"current_enrollment"`
	MaxEnrollment     int       `json:"max_enrollment" dynamodbav:"max_enrollment"`
	SpecialNotes      string    `json:"special_notes" dynamodbav:"special_notes"`
	CreatedAt         time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" dynamodbav:"updated_at"`
}

// Attraction represents ongoing venue features
type Attraction struct {
	FamilyActivity

	// Attraction-specific fields
	AttractionName string `json:"attraction_name" dynamodbav:"attraction_name"`
	AttractionType string `json:"attraction_type" dynamodbav:"attraction_type"` // exhibit, playground, ride, etc.
	VenueID        string `json:"venue_id" dynamodbav:"venue_id"`               // Reference to venue
	Availability   string `json:"availability" dynamodbav:"availability"`       // ongoing, seasonal, temporary
	Images         []Image `json:"images" dynamodbav:"images"`
}

// ActivityLocation provides detailed location information (extends existing Location)
type ActivityLocation struct {
	Location                                                                       // embed existing Location type
	VenueType     string `json:"venue_type" dynamodbav:"venue_type"`             // indoor, outdoor, mixed
	Accessibility string `json:"accessibility" dynamodbav:"accessibility"`       // ADA accessible details
	Parking       string `json:"parking" dynamodbav:"parking"`                   // parking availability info
	PublicTransit string `json:"public_transit" dynamodbav:"public_transit"`     // public transit information
}

// ActivityPricing contains cost and payment information (extends existing Pricing)
type ActivityPricing struct {
	Pricing                                                                        // embed existing Pricing type
	IncludesSupplies bool `json:"includes_supplies" dynamodbav:"includes_supplies"` // whether supplies are included
}

// ContactInfo represents contact information
type ContactInfo struct {
	Phone   string `json:"phone" dynamodbav:"phone"`     // contact phone number
	Email   string `json:"email" dynamodbav:"email"`     // contact email
	Website string `json:"website" dynamodbav:"website"` // website URL
}

// Additional entity status constants (extends existing constants from activity.go)
const (
	StatusExpired = "expired" // added to existing status constants
)

// Helper functions to create primary keys
func CreateVenuePK(venueID string) string {
	return EntityTypeVenue + "#" + venueID
}

func CreateEventPK(eventID string) string {
	return EntityTypeEvent + "#" + eventID
}

func CreateProgramPK(programID string) string {
	return EntityTypeProgram + "#" + programID
}

func CreateAttractionPK(attractionID string) string {
	return EntityTypeAttraction + "#" + attractionID
}

func CreateInstanceSK(date, time string) string {
	return SortKeyInstance + "#" + date + "T" + time
}

// Helper functions to generate GSI keys
func GenerateLocationKey(region, city string) string {
	return "GEO#" + region + "#" + city
}

func GenerateDateTypeKey(date, entityType, entityID string) string {
	return "DATE#" + date + "#TYPE#" + entityType + "#" + entityID
}

func GenerateCategoryAgeKey(category, ageGroup string) string {
	return "CAT#" + category + "#" + ageGroup
}

func GenerateDateFeaturedKey(date, featured, entityID string) string {
	return "DATE#" + date + "#FEATURED#" + featured + "#" + entityID
}

func GenerateVenueKey(venueID string) string {
	return "VENUE#" + venueID
}

func GenerateTypeStatusKey(entityType, status, entityID string) string {
	return "TYPE#" + entityType + "#STATUS#" + status + "#" + entityID
}