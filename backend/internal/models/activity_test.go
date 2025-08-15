package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestActivityModel(t *testing.T) {
	// Create a sample activity
	activity := Activity{
		ID:          GenerateActivityID("Test Music Class", "2024-09-01", "Seattle Music Academy"),
		Title:       "Test Music Class",
		Description: "A test music class for toddlers",
		Type:        TypeClass,
		Category:    CategoryArtsCreativity,
		Subcategory: "music",
		Schedule: Schedule{
			Type:      ScheduleTypeRecurring,
			StartDate: "2024-09-01",
			EndDate:   "2024-12-15",
			Frequency: "weekly",
			DaysOfWeek: []string{"tuesday", "thursday"},
			Times: []TimeSlot{
				{
					StartTime: "10:00",
					EndTime:   "10:45",
					AgeGroup:  AgeGroupToddler,
				},
			},
			Duration: "45 minutes",
			Sessions: 16,
		},
		AgeGroups: []AgeGroup{
			{
				Category:    AgeGroupToddler,
				MinAge:      18,
				MaxAge:      36,
				Unit:        "months",
				Description: "18 months - 3 years",
			},
		},
		FamilyType: FamilyTypeParentChild,
		Location: Location{
			Name:         "Seattle Music Academy",
			Address:      "123 Pine Street, Seattle, WA 98101",
			Neighborhood: "Capitol Hill",
			City:         "Seattle",
			Region:       "Seattle Metro",
			ZipCode:      "98101",
			VenueType:     VenueTypeIndoor,
			Accessibility: "wheelchair accessible, elevator available",
			Parking:       "street",
		},
		Pricing: Pricing{
			Type:             PricingTypePaid,
			Cost:             180.0,
			Currency:         "USD",
			Unit:             "session",
			Description:      "$180 for 16-week session",
			IncludesSupplies: true,
		},
		Registration: Registration{
			Required: true,
			Method:   "online",
			URL:      "https://seattlemusicacademy.com/register",
			Status:   RegistrationStatusOpen,
		},
		Tags: []string{"music", "movement", "toddler", "parent-child"},
		Provider: Provider{
			Name:     "Seattle Music Academy",
			Type:     "business",
			Website:  "https://seattlemusicacademy.com",
			Verified: true,
		},
		Source: Source{
			URL:         "https://www.seattleschild.com/music-classes",
			Domain:      "seattleschild.com",
			ScrapedAt:   time.Now(),
			LastChecked: time.Now(),
			Reliability: "high",
		},
		Featured:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    ActivityStatusActive,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(activity)
	if err != nil {
		t.Fatalf("Failed to marshal activity to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled Activity
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal activity from JSON: %v", err)
	}

	// Verify key fields
	if unmarshaled.Title != activity.Title {
		t.Errorf("Expected title %s, got %s", activity.Title, unmarshaled.Title)
	}

	if unmarshaled.Type != TypeClass {
		t.Errorf("Expected type %s, got %s", TypeClass, unmarshaled.Type)
	}

	if unmarshaled.Category != CategoryArtsCreativity {
		t.Errorf("Expected category %s, got %s", CategoryArtsCreativity, unmarshaled.Category)
	}

	if len(unmarshaled.AgeGroups) != 1 {
		t.Errorf("Expected 1 age group, got %d", len(unmarshaled.AgeGroups))
	}

	if unmarshaled.Pricing.Cost != 180.0 {
		t.Errorf("Expected cost 180.0, got %f", unmarshaled.Pricing.Cost)
	}
}

func TestValidationFunctions(t *testing.T) {
	// Test activity type validation
	if !ValidateActivityType(TypeClass) {
		t.Error("ValidateActivityType should return true for valid type")
	}

	if ValidateActivityType("invalid-type") {
		t.Error("ValidateActivityType should return false for invalid type")
	}

	// Test category validation
	if !ValidateCategory(CategoryArtsCreativity) {
		t.Error("ValidateCategory should return true for valid category")
	}

	if ValidateCategory("invalid-category") {
		t.Error("ValidateCategory should return false for invalid category")
	}

	// Test age group validation
	if !ValidateAgeGroup(AgeGroupToddler) {
		t.Error("ValidateAgeGroup should return true for valid age group")
	}

	if ValidateAgeGroup("invalid-age-group") {
		t.Error("ValidateAgeGroup should return false for invalid age group")
	}
}

func TestIDGeneration(t *testing.T) {
	// Test activity ID generation
	id1 := GenerateActivityID("Music Class", "2024-09-01", "Seattle Music Academy")
	id2 := GenerateActivityID("Music Class", "2024-09-01", "Seattle Music Academy")
	id3 := GenerateActivityID("Different Class", "2024-09-01", "Seattle Music Academy")

	// Same inputs should generate same ID
	if id1 != id2 {
		t.Errorf("Same inputs should generate same ID: %s != %s", id1, id2)
	}

	// Different inputs should generate different IDs
	if id1 == id3 {
		t.Errorf("Different inputs should generate different IDs: %s == %s", id1, id3)
	}

	// IDs should have correct prefix
	if len(id1) != 12 || id1[:4] != "act_" {
		t.Errorf("Activity ID should be 12 characters starting with 'act_', got: %s", id1)
	}
}

func TestActivitiesOutput(t *testing.T) {
	// Create sample activities
	activities := []Activity{
		{
			ID:       "act_12345678",
			Title:    "Test Activity 1",
			Type:     TypeClass,
			Category: CategoryArtsCreativity,
			Status:   ActivityStatusActive,
		},
		{
			ID:       "act_87654321",
			Title:    "Test Activity 2", 
			Type:     TypeEvent,
			Category: CategoryEntertainmentEvents,
			Status:   ActivityStatusActive,
		},
	}

	// Create metadata
	metadata := NewActivitiesMetadata(len(activities), []string{"test-source.com"})

	// Create output structure
	output := ActivitiesOutput{
		Metadata:   metadata,
		Activities: activities,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal ActivitiesOutput to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled ActivitiesOutput
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ActivitiesOutput from JSON: %v", err)
	}

	// Verify structure
	if unmarshaled.Metadata.TotalActivities != 2 {
		t.Errorf("Expected 2 total activities, got %d", unmarshaled.Metadata.TotalActivities)
	}

	if len(unmarshaled.Activities) != 2 {
		t.Errorf("Expected 2 activities, got %d", len(unmarshaled.Activities))
	}

	if unmarshaled.Metadata.Coverage != "Seattle Metro Area" {
		t.Errorf("Expected 'Seattle Metro Area' coverage, got %s", unmarshaled.Metadata.Coverage)
	}
}

func TestUtilityFunctions(t *testing.T) {
	// Test email validation
	if !IsValidEmail("test@example.com") {
		t.Error("Should validate correct email")
	}

	if IsValidEmail("invalid-email") {
		t.Error("Should reject invalid email")
	}

	// Test URL validation
	if !IsValidURL("https://example.com") {
		t.Error("Should validate correct HTTPS URL")
	}

	if !IsValidURL("http://example.com") {
		t.Error("Should validate correct HTTP URL")
	}

	if IsValidURL("invalid-url") {
		t.Error("Should reject invalid URL")
	}

	// Test phone validation
	if !IsValidPhoneNumber("(206) 555-1234") {
		t.Error("Should validate formatted phone number")
	}

	if !IsValidPhoneNumber("2065551234") {
		t.Error("Should validate unformatted phone number")
	}

	if IsValidPhoneNumber("123") {
		t.Error("Should reject too short phone number")
	}
}

func TestDisplayNames(t *testing.T) {
	// Test age group display names
	display := GetAgeGroupDisplayName(AgeGroupToddler)
	if display != "Toddler (1-3 years)" {
		t.Errorf("Expected 'Toddler (1-3 years)', got %s", display)
	}

	// Test category display names
	display = GetCategoryDisplayName(CategoryArtsCreativity)
	if display != "Arts & Creativity" {
		t.Errorf("Expected 'Arts & Creativity', got %s", display)
	}

	// Test type display names
	display = GetTypeDisplayName(TypeClass)
	if display != "Class" {
		t.Errorf("Expected 'Class', got %s", display)
	}
}