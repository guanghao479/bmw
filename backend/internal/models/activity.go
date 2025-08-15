package models

import "time"

// ActivitiesOutput represents the complete JSON structure for activities data
type ActivitiesOutput struct {
	Metadata   ActivitiesMetadata `json:"metadata"`
	Activities []Activity         `json:"activities"`
}

// ActivitiesMetadata contains metadata about the activities dataset
type ActivitiesMetadata struct {
	LastUpdated     time.Time `json:"lastUpdated"`
	TotalActivities int       `json:"totalActivities"`
	Sources         []string  `json:"sources"`
	NextUpdate      time.Time `json:"nextUpdate"`
	Version         string    `json:"version"`
	Region          string    `json:"region"`
	Coverage        string    `json:"coverage"`
}

// Activity represents a single family activity/event/venue
type Activity struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`

	// Core Classification
	Type        string `json:"type"`        // class|camp|event|performance|free-activity
	Category    string `json:"category"`    // arts-creativity|active-sports|educational-stem|entertainment-events|camps-programs|free-community
	Subcategory string `json:"subcategory"` // music|soccer|science|etc

	// Scheduling
	Schedule Schedule `json:"schedule"`

	// Age & Audience
	AgeGroups  []AgeGroup `json:"ageGroups"`
	FamilyType string     `json:"familyType"` // drop-off|parent-child|family-friendly|adult-only

	// Location
	Location Location `json:"location"`

	// Pricing
	Pricing Pricing `json:"pricing"`

	// Registration
	Registration Registration `json:"registration"`

	// Content & Links
	Images    []Image  `json:"images,omitempty"`
	DetailURL string   `json:"detailUrl,omitempty"` // direct link to event/activity details
	Tags      []string `json:"tags"`

	// Provider
	Provider Provider `json:"provider"`

	// Source Tracking
	Source Source `json:"source"`

	// System Fields
	Featured  bool      `json:"featured"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Status    string    `json:"status"` // active|inactive|expired|cancelled
}

// Schedule defines when an activity occurs
type Schedule struct {
	Type       string     `json:"type"`                 // one-time|recurring|multi-day|ongoing
	StartDate  string     `json:"startDate"`            // ISO date (YYYY-MM-DD)
	EndDate    string     `json:"endDate,omitempty"`    // ISO date, optional
	StartTime  string     `json:"startTime,omitempty"`  // HH:MM format (24-hour) - primary start time
	EndTime    string     `json:"endTime,omitempty"`    // HH:MM format (24-hour) - primary end time
	Timezone   string     `json:"timezone,omitempty"`   // "America/Los_Angeles" for Seattle
	IsAllDay   bool       `json:"isAllDay"`             // true for all-day events
	Frequency  string     `json:"frequency,omitempty"`  // daily|weekly|monthly|seasonal
	DaysOfWeek []string   `json:"daysOfWeek,omitempty"` // monday, tuesday, etc.
	Times      []TimeSlot `json:"times"`                // additional time slots for age-specific times
	Duration   string     `json:"duration,omitempty"`   // "45 minutes", "2 hours"
	Sessions   int        `json:"sessions,omitempty"`   // total number of sessions
}

// TimeSlot represents a specific time period for an activity
type TimeSlot struct {
	StartTime string `json:"startTime"`          // HH:MM format (24-hour)
	EndTime   string `json:"endTime"`            // HH:MM format (24-hour)
	AgeGroup  string `json:"ageGroup,omitempty"` // specific age group for this time slot
}

// AgeGroup defines the target age range for an activity
type AgeGroup struct {
	Category    string `json:"category"`    // infant|toddler|preschool|elementary|tween|teen|adult|all-ages
	MinAge      int    `json:"minAge"`      // minimum age
	MaxAge      int    `json:"maxAge"`      // maximum age
	Unit        string `json:"unit"`        // months|years
	Description string `json:"description"` // human-readable description
}

// Location provides detailed venue information
type Location struct {
	Name          string      `json:"name"`                    // venue name
	Address       string      `json:"address"`                 // full street address
	City          string      `json:"city"`                    // Seattle, Bellevue, etc.
	State         string      `json:"state,omitempty"`         // WA
	ZipCode       string      `json:"zipCode,omitempty"`       // postal code
	Neighborhood  string      `json:"neighborhood,omitempty"`  // Capitol Hill, Ballard, etc.
	Region        string      `json:"region"`                  // Seattle Metro, Eastside, etc.
	Coordinates   Coordinates `json:"coordinates,omitempty"`   // lat/lng
	VenueType     string      `json:"venueType"`               // indoor|outdoor|mixed
	Accessibility string      `json:"accessibility,omitempty"` // ADA accessible details
	Parking       string      `json:"parking,omitempty"`       // parking availability info
	PublicTransit string      `json:"publicTransit,omitempty"` // public transit information
}

// Coordinates represents geographical coordinates
type Coordinates struct {
	Lat float64 `json:"lat"` // latitude
	Lng float64 `json:"lng"` // longitude
}

// Pricing contains cost and payment information
type Pricing struct {
	Type             string     `json:"type"`                      // free|paid|donation|variable
	Cost             float64    `json:"cost,omitempty"`            // numeric cost
	Currency         string     `json:"currency"`                  // USD, CAD, etc.
	Unit             string     `json:"unit"`                      // per-person|per-family|per-session|per-class|per-week
	Description      string     `json:"description"`               // human-readable pricing info
	Discounts        []Discount `json:"discounts,omitempty"`       // available discounts
	IncludesSupplies bool       `json:"includesSupplies"`          // whether supplies are included
}

// Discount represents a pricing discount
type Discount struct {
	Type        string `json:"type"`        // sibling|senior|member|student
	Description string `json:"description"` // description of the discount
}

// Registration contains signup and contact information
type Registration struct {
	Required     bool   `json:"required"`             // whether registration is required
	Method       string `json:"method"`               // online|phone|in-person|walk-in
	URL          string `json:"url,omitempty"`        // registration URL
	Phone        string `json:"phone,omitempty"`      // contact phone
	Email        string `json:"email,omitempty"`      // contact email
	Deadline     string `json:"deadline,omitempty"`   // registration deadline (ISO date)
	OpenDate     string `json:"openDate,omitempty"`   // when registration opens (ISO date)
	Status       string `json:"status"`               // open|waitlist|closed|sold-out
	ContactPhone string `json:"contactPhone,omitempty"` // formatted contact phone
	ContactEmail string `json:"contactEmail,omitempty"` // formatted contact email
}

// Image represents an activity image
type Image struct {
	URL        string `json:"url"`                  // image URL
	AltText    string `json:"altText,omitempty"`    // alt text for accessibility
	Caption    string `json:"caption,omitempty"`    // optional caption
	SourceType string `json:"sourceType,omitempty"` // event|venue|activity|gallery
	Width      int    `json:"width,omitempty"`      // image width in pixels
	Height     int    `json:"height,omitempty"`     // image height in pixels
}

// Provider represents the organization offering the activity
type Provider struct {
	Name        string `json:"name"`                    // provider name
	Type        string `json:"type"`                    // business|non-profit|government|community|individual
	Website     string `json:"website,omitempty"`       // provider website
	Description string `json:"description,omitempty"`   // brief description
	Verified    bool   `json:"verified"`                // whether provider is verified
}

// Source tracks where the activity data came from
type Source struct {
	URL         string    `json:"url"`         // source URL
	Domain      string    `json:"domain"`      // source domain
	ScrapedAt   time.Time `json:"scrapedAt"`   // when it was scraped
	LastChecked time.Time `json:"lastChecked"` // last verification time
	Reliability string    `json:"reliability"` // high|medium|low
}

// Activity type constants
const (
	TypeClass        = "class"
	TypeCamp         = "camp"
	TypeEvent        = "event"
	TypePerformance  = "performance"
	TypeFreeActivity = "free-activity"
)

// Category constants
const (
	CategoryArtsCreativity      = "arts-creativity"
	CategoryActiveSports        = "active-sports"
	CategoryEducationalSTEM     = "educational-stem"
	CategoryEntertainmentEvents = "entertainment-events"
	CategoryCampsPrograms       = "camps-programs"
	CategoryFreeCommunity       = "free-community"
)

// Age group constants
const (
	AgeGroupInfant     = "infant"
	AgeGroupToddler    = "toddler"
	AgeGroupPreschool  = "preschool"
	AgeGroupElementary = "elementary"
	AgeGroupTween      = "tween"
	AgeGroupTeen       = "teen"
	AgeGroupAdult      = "adult"
	AgeGroupAllAges    = "all-ages"
)

// Schedule type constants
const (
	ScheduleTypeOneTime   = "one-time"
	ScheduleTypeRecurring = "recurring"
	ScheduleTypeMultiDay  = "multi-day"
	ScheduleTypeOngoing   = "ongoing"
)

// Pricing type constants
const (
	PricingTypeFree     = "free"
	PricingTypePaid     = "paid"
	PricingTypeDonation = "donation"
	PricingTypeVariable = "variable"
)

// Venue type constants
const (
	VenueTypeIndoor  = "indoor"
	VenueTypeOutdoor = "outdoor"
	VenueTypeMixed   = "mixed"
)

// Family type constants
const (
	FamilyTypeDropOff        = "drop-off"
	FamilyTypeParentChild    = "parent-child"
	FamilyTypeFamilyFriendly = "family-friendly"
	FamilyTypeAdultOnly      = "adult-only"
)

// Registration status constants
const (
	RegistrationStatusOpen     = "open"
	RegistrationStatusWaitlist = "waitlist"
	RegistrationStatusClosed   = "closed"
	RegistrationStatusSoldOut  = "sold-out"
)

// Activity status constants
const (
	ActivityStatusActive    = "active"
	ActivityStatusInactive  = "inactive"
	ActivityStatusExpired   = "expired"
	ActivityStatusCancelled = "cancelled"
)