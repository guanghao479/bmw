package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// GenerateActivityID creates a unique ID for an activity based on its core attributes
func GenerateActivityID(title, date, location string) string {
	// Normalize inputs
	normalizedTitle := strings.ToLower(strings.TrimSpace(title))
	normalizedDate := strings.ToLower(strings.TrimSpace(date))
	normalizedLocation := strings.ToLower(strings.TrimSpace(location))
	
	// Create hash input
	input := fmt.Sprintf("%s|%s|%s", normalizedTitle, normalizedDate, normalizedLocation)
	
	// Generate SHA256 hash
	hash := sha256.Sum256([]byte(input))
	
	// Return first 8 characters with prefix
	return "act_" + hex.EncodeToString(hash[:])[:8]
}

// GenerateScrapingJobID creates a unique ID for a scraping job
func GenerateScrapingJobID(sourceURL string, timestamp time.Time) string {
	input := fmt.Sprintf("%s|%d", sourceURL, timestamp.Unix())
	hash := sha256.Sum256([]byte(input))
	return "job_" + hex.EncodeToString(hash[:])[:8]
}

// GenerateScrapingRunID creates a unique ID for a scraping run
func GenerateScrapingRunID(timestamp time.Time) string {
	input := fmt.Sprintf("run|%d", timestamp.Unix())
	hash := sha256.Sum256([]byte(input))
	return "run_" + hex.EncodeToString(hash[:])[:8]
}

// ValidateActivityType checks if the activity type is valid
func ValidateActivityType(activityType string) bool {
	validTypes := []string{
		TypeClass,
		TypeCamp,
		TypeEvent,
		TypePerformance,
		TypeFreeActivity,
	}
	
	for _, validType := range validTypes {
		if activityType == validType {
			return true
		}
	}
	return false
}

// ValidateCategory checks if the category is valid
func ValidateCategory(category string) bool {
	validCategories := []string{
		CategoryArtsCreativity,
		CategoryActiveSports,
		CategoryEducationalSTEM,
		CategoryEntertainmentEvents,
		CategoryCampsPrograms,
		CategoryFreeCommunity,
	}
	
	for _, validCategory := range validCategories {
		if category == validCategory {
			return true
		}
	}
	return false
}

// ValidateAgeGroup checks if the age group is valid
func ValidateAgeGroup(ageGroup string) bool {
	validAgeGroups := []string{
		AgeGroupInfant,
		AgeGroupToddler,
		AgeGroupPreschool,
		AgeGroupElementary,
		AgeGroupTween,
		AgeGroupTeen,
		AgeGroupAdult,
		AgeGroupAllAges,
	}
	
	for _, validAgeGroup := range validAgeGroups {
		if ageGroup == validAgeGroup {
			return true
		}
	}
	return false
}

// ValidateScheduleType checks if the schedule type is valid
func ValidateScheduleType(scheduleType string) bool {
	validTypes := []string{
		ScheduleTypeOneTime,
		ScheduleTypeRecurring,
		ScheduleTypeMultiDay,
		ScheduleTypeOngoing,
	}
	
	for _, validType := range validTypes {
		if scheduleType == validType {
			return true
		}
	}
	return false
}

// ValidatePricingType checks if the pricing type is valid
func ValidatePricingType(pricingType string) bool {
	validTypes := []string{
		PricingTypeFree,
		PricingTypePaid,
		PricingTypeDonation,
		PricingTypeVariable,
	}
	
	for _, validType := range validTypes {
		if pricingType == validType {
			return true
		}
	}
	return false
}

// ValidateVenueType checks if the venue type is valid
func ValidateVenueType(venueType string) bool {
	validTypes := []string{
		VenueTypeIndoor,
		VenueTypeOutdoor,
		VenueTypeMixed,
	}
	
	for _, validType := range validTypes {
		if venueType == validType {
			return true
		}
	}
	return false
}

// ValidateImageSourceType checks if the image source type is valid
func ValidateImageSourceType(sourceType string) bool {
	validTypes := []string{
		"event",
		"venue", 
		"activity",
		"gallery",
	}
	
	for _, validType := range validTypes {
		if sourceType == validType {
			return true
		}
	}
	return false
}

// ValidateImageURL performs enhanced URL validation for images
func ValidateImageURL(url string) bool {
	if !IsValidURL(url) {
		return false
	}
	
	// Check for common image extensions
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg"}
	urlLower := strings.ToLower(url)
	
	for _, ext := range imageExtensions {
		if strings.Contains(urlLower, ext) {
			return true
		}
	}
	
	// Allow URLs that might have query parameters or no extension (many CDNs)
	return true
}

// IsValidEmail performs basic email validation
func IsValidEmail(email string) bool {
	if email == "" {
		return false
	}
	
	// Basic email validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return false
	}
	
	if !strings.Contains(parts[1], ".") {
		return false
	}
	
	return true
}

// IsValidURL performs basic URL validation
func IsValidURL(url string) bool {
	if url == "" {
		return false
	}
	
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// IsValidPhoneNumber performs basic phone number validation
func IsValidPhoneNumber(phone string) bool {
	if phone == "" {
		return false
	}
	
	// Remove common formatting characters
	cleaned := strings.ReplaceAll(phone, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "")
	
	// Check if it's all digits and reasonable length
	if len(cleaned) < 10 || len(cleaned) > 15 {
		return false
	}
	
	for _, char := range cleaned {
		if char < '0' || char > '9' {
			return false
		}
	}
	
	return true
}

// FormatTimeSlot formats a time slot for display
func (ts TimeSlot) Format() string {
	if ts.StartTime == "" || ts.EndTime == "" {
		return ""
	}
	
	return fmt.Sprintf("%s - %s", ts.StartTime, ts.EndTime)
}

// GetDisplayName returns a human-readable name for an age group
func GetAgeGroupDisplayName(category string) string {
	displayNames := map[string]string{
		AgeGroupInfant:     "Infant (0-12 months)",
		AgeGroupToddler:    "Toddler (1-3 years)",
		AgeGroupPreschool:  "Preschool (3-5 years)",
		AgeGroupElementary: "Elementary (6-10 years)",
		AgeGroupTween:      "Tween (11-14 years)",
		AgeGroupTeen:       "Teen (15-18 years)",
		AgeGroupAdult:      "Adult (18+ years)",
		AgeGroupAllAges:    "All Ages",
	}
	
	if displayName, exists := displayNames[category]; exists {
		return displayName
	}
	
	return category
}

// GetCategoryDisplayName returns a human-readable name for a category
func GetCategoryDisplayName(category string) string {
	displayNames := map[string]string{
		CategoryArtsCreativity:      "Arts & Creativity",
		CategoryActiveSports:        "Active & Sports",
		CategoryEducationalSTEM:     "Educational & STEM",
		CategoryEntertainmentEvents: "Entertainment & Events",
		CategoryCampsPrograms:       "Camps & Programs",
		CategoryFreeCommunity:       "Free & Community",
	}
	
	if displayName, exists := displayNames[category]; exists {
		return displayName
	}
	
	return category
}

// GetTypeDisplayName returns a human-readable name for an activity type
func GetTypeDisplayName(activityType string) string {
	displayNames := map[string]string{
		TypeClass:        "Class",
		TypeCamp:         "Camp",
		TypeEvent:        "Event",
		TypePerformance:  "Performance",
		TypeFreeActivity: "Free Activity",
	}
	
	if displayName, exists := displayNames[activityType]; exists {
		return displayName
	}
	
	return activityType
}

// CalculateDuplicateSimilarity calculates similarity between two activities (0.0 to 1.0)
func CalculateDuplicateSimilarity(a1, a2 Activity) float64 {
	score := 0.0
	maxScore := 4.0 // 4 comparison criteria
	
	// Title similarity (most important)
	if strings.ToLower(a1.Title) == strings.ToLower(a2.Title) {
		score += 2.0
	} else if strings.Contains(strings.ToLower(a1.Title), strings.ToLower(a2.Title)) ||
		strings.Contains(strings.ToLower(a2.Title), strings.ToLower(a1.Title)) {
		score += 1.0
	}
	
	// Location similarity
	if strings.ToLower(a1.Location.Name) == strings.ToLower(a2.Location.Name) {
		score += 1.0
	} else if strings.Contains(strings.ToLower(a1.Location.Address), strings.ToLower(a2.Location.Address)) {
		score += 0.5
	}
	
	// Date similarity (for recurring activities)
	if a1.Schedule.StartDate == a2.Schedule.StartDate {
		score += 1.0
	}
	
	return score / maxScore
}

// NewActivitiesMetadata creates metadata for activities output
func NewActivitiesMetadata(totalActivities int, sources []string) ActivitiesMetadata {
	return ActivitiesMetadata{
		LastUpdated:     time.Now(),
		TotalActivities: totalActivities,
		Sources:         sources,
		NextUpdate:      time.Now().Add(6 * time.Hour),
		Version:         "1.0.0",
		Region:          "us-west-2",
		Coverage:        "Seattle Metro Area",
	}
}