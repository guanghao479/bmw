package services

import (
	"strings"
	"testing"
	"time"
)

// TestMarkdownParsingRealWorldSamples tests markdown parsing with realistic content from target sources
func TestMarkdownParsingRealWorldSamples(t *testing.T) {
	fc := &FireCrawlClient{}

	t.Run("ParentMapCalendarFormat", func(t *testing.T) {
		// Based on actual ParentMap calendar structure
		parentMapContent := `
# Seattle Family Events - December 2024

## Weekend Activities

### Saturday, December 14

#### Kids Art Workshop
**When:** Saturday, December 14, 2024, 10:00 AM - 12:00 PM  
**Where:** Seattle Community Center, 123 Main Street, Seattle, WA 98101  
**Ages:** 5-10 years  
**Cost:** $25 per child  
**Registration:** Required - call (206) 555-0123  

Join us for a creative art workshop where children will explore painting, drawing, and crafts. All materials provided. Drop-off event for parents.

#### Family Movie in the Park
**When:** Saturday, December 14, 2024, 7:00 PM  
**Where:** Volunteer Park, 1247 15th Ave E, Seattle, WA  
**Ages:** All ages welcome  
**Cost:** Free (donations appreciated)  
**Bring:** Blankets and snacks  

Outdoor screening of a family-friendly holiday movie. Event may be cancelled due to weather.

### Sunday, December 15

#### Story Time at Central Library
**When:** Sunday, December 15, 2024, 2:00 PM - 3:00 PM  
**Where:** Seattle Central Library, 1000 4th Ave, Seattle, WA  
**Ages:** 0-5 years with parent/caregiver  
**Cost:** Free  
**Registration:** Not required  

Weekly story time featuring holiday-themed books and songs. Perfect for toddlers and preschoolers.

## Ongoing Programs

#### Swimming Lessons
**When:** Mondays and Wednesdays, 4:00 PM - 5:00 PM  
**Where:** Seattle Aquatic Center, 1920 4th Ave W, Seattle, WA  
**Ages:** 6-12 years  
**Cost:** $120 for 8-week session  
**Registration:** Online at seattleaquatics.com  

Learn to swim in a fun, supportive environment. All skill levels welcome.
`

		attempt := &ExtractionAttempt{
			Method:    "test_parentmap_real",
			Timestamp: time.Now(),
		}
		events := fc.extractEventsFromMarkdown(parentMapContent, attempt)

		if len(events) < 3 {
			t.Errorf("Expected at least 3 events from ParentMap content, got %d", len(events))
		}

		// Validate specific events
		foundArtWorkshop := false
		foundMovieNight := false
		foundStoryTime := false
		foundSwimming := false

		for _, event := range events {
			title := strings.ToLower(event.Title)
			
			if strings.Contains(title, "art") && strings.Contains(title, "workshop") {
				foundArtWorkshop = true
				
				// Validate art workshop details
				if event.Date == "" {
					t.Error("Art workshop should have a date")
				}
				if event.Time == "" {
					t.Error("Art workshop should have a time")
				}
				if event.Location == "" {
					t.Error("Art workshop should have a location")
				}
				if len(event.AgeGroups) == 0 {
					t.Error("Art workshop should have age group information")
				}
				if event.Price == "" {
					t.Error("Art workshop should have price information")
				}
				
				t.Logf("Art Workshop: Date=%s, Time=%s, Location=%s, Price=%s", 
					event.Date, event.Time, event.Location, event.Price)
			}
			
			if strings.Contains(title, "movie") {
				foundMovieNight = true
				
				// Should be free
				if !strings.Contains(strings.ToLower(event.Price), "free") {
					t.Error("Movie night should be marked as free")
				}
			}
			
			if strings.Contains(title, "story") {
				foundStoryTime = true
				
				// Should have age restrictions
				if len(event.AgeGroups) == 0 {
					t.Error("Story time should have age group information")
				}
			}
			
			if strings.Contains(title, "swim") {
				foundSwimming = true
				
				// Should be recurring
				if !strings.Contains(strings.ToLower(event.Date), "monday") && 
				   !strings.Contains(strings.ToLower(event.Date), "wednesday") {
					t.Error("Swimming lessons should indicate recurring schedule")
				}
			}
		}

		if !foundArtWorkshop {
			t.Error("Should find Art Workshop event")
		}
		if !foundMovieNight {
			t.Error("Should find Movie Night event")
		}
		if !foundStoryTime {
			t.Error("Should find Story Time event")
		}
		if !foundSwimming {
			t.Error("Should find Swimming Lessons event")
		}
	})

	t.Run("RemlingerFarmsEventFormat", func(t *testing.T) {
		// Based on actual Remlinger Farms event structure
		remlingerContent := `
# Remlinger Farms Events & Activities

## Seasonal Events

### Pumpkin Patch & Fall Festival
**Dates:** October 1-31, 2024  
**Hours:** Daily 10:00 AM - 6:00 PM  
**Admission:** $15 adults, $12 children (2-12), Free under 2  
**Activities Included:** Pumpkin picking, hayrides, corn maze, farm animals  

Come experience the magic of fall at our pumpkin patch! Pick your perfect pumpkin from our 10-acre patch, enjoy a scenic hayride through the farm, and get lost in our challenging corn maze. Don't forget to visit our friendly farm animals!

**Special Weekend Activities:**
- Pumpkin carving demonstrations (Saturdays 2-4 PM)
- Live music (Sundays 1-3 PM)
- Face painting (Weekends 11 AM - 5 PM)

### Holiday Light Spectacular
**Dates:** November 25, 2024 - January 1, 2025  
**Hours:** 5:00 PM - 9:00 PM daily  
**Admission:** $20 per vehicle (up to 8 people)  
**Duration:** Approximately 30 minutes drive-through  

Drive through our winter wonderland featuring over 1 million twinkling lights! This magical display includes animated light shows, holiday music, and special themed areas. Hot cocoa and treats available for purchase.

## Year-Round Activities

### Farm Animal Experience
**Available:** Daily year-round  
**Hours:** 10:00 AM - 4:00 PM (weather permitting)  
**Cost:** Included with farm admission  
**Perfect for:** All ages, especially toddlers and young children  

Feed and interact with our friendly farm animals including goats, sheep, chickens, and miniature horses. Animal feed available for purchase ($2 per cup).

### Train Rides
**Available:** Weekends and holidays  
**Hours:** 11:00 AM - 5:00 PM  
**Cost:** $5 per person (free under 2)  
**Duration:** 15-minute scenic ride  

Take a relaxing ride on our vintage train through the beautiful countryside surrounding the farm.

### Pick-Your-Own Berries
**Season:** June - September  
**Hours:** 8:00 AM - 6:00 PM daily during season  
**Cost:** $8 per pound  
**Varieties:** Strawberries (June), Blueberries (July-August), Blackberries (August-September)  

Experience the joy of picking your own fresh berries! Containers provided. Call ahead to check availability and ripeness.

## Special Programs

### Birthday Parties
**Available:** Year-round by reservation  
**Duration:** 2 hours  
**Cost:** Starting at $200 for up to 10 children  
**Includes:** Private party area, farm activities, birthday child rides free  

Make your child's birthday unforgettable with a farm-themed party! Includes access to all farm activities, decorated party area, and special birthday surprises.

### School Field Trips
**Available:** September - June  
**Duration:** 2-3 hours  
**Cost:** $12 per student, teachers free  
**Educational Focus:** Farm life, animal care, agriculture  

Educational and fun field trips designed for elementary school groups. Includes guided tour, animal interactions, and hands-on learning activities.
`

		attempt := &ExtractionAttempt{
			Method:    "test_remlinger_real",
			Timestamp: time.Now(),
		}
		events := fc.extractEventsFromMarkdown(remlingerContent, attempt)

		if len(events) < 5 {
			t.Errorf("Expected at least 5 events from Remlinger content, got %d", len(events))
		}

		// Validate specific events
		foundPumpkinPatch := false
		foundHolidayLights := false
		foundAnimalExperience := false
		foundTrainRides := false
		foundBerryPicking := false

		for _, event := range events {
			title := strings.ToLower(event.Title)
			
			if strings.Contains(title, "pumpkin") {
				foundPumpkinPatch = true
				
				// Should have date range
				if !strings.Contains(strings.ToLower(event.Date), "october") {
					t.Error("Pumpkin patch should have October dates")
				}
				
				// Should have pricing information
				if event.Price == "" {
					t.Error("Pumpkin patch should have pricing information")
				}
				
				t.Logf("Pumpkin Patch: Date=%s, Price=%s", event.Date, event.Price)
			}
			
			if strings.Contains(title, "holiday") || strings.Contains(title, "light") {
				foundHolidayLights = true
				
				// Should be vehicle-based pricing
				if !strings.Contains(strings.ToLower(event.Price), "vehicle") {
					t.Error("Holiday lights should mention vehicle pricing")
				}
			}
			
			if strings.Contains(title, "animal") {
				foundAnimalExperience = true
				
				// Should be year-round
				if !strings.Contains(strings.ToLower(event.Date), "daily") && 
				   !strings.Contains(strings.ToLower(event.Date), "year") {
					t.Error("Animal experience should indicate year-round availability")
				}
			}
			
			if strings.Contains(title, "train") {
				foundTrainRides = true
				
				// Should have duration information in description
				if !strings.Contains(event.Description, "15") {
					t.Error("Train rides should have duration information in description")
				}
			}
			
			if strings.Contains(title, "berr") || strings.Contains(title, "pick") {
				foundBerryPicking = true
				
				// Should be seasonal
				if !strings.Contains(strings.ToLower(event.Date), "june") && 
				   !strings.Contains(strings.ToLower(event.Date), "september") {
					t.Error("Berry picking should indicate seasonal availability")
				}
			}
		}

		if !foundPumpkinPatch {
			t.Error("Should find Pumpkin Patch event")
		}
		if !foundHolidayLights {
			t.Error("Should find Holiday Lights event")
		}
		if !foundAnimalExperience {
			t.Error("Should find Animal Experience event")
		}
		if !foundTrainRides {
			t.Error("Should find Train Rides event")
		}
		if !foundBerryPicking {
			t.Error("Should find Berry Picking event")
		}
	})

	t.Run("PoorlyStructuredContent", func(t *testing.T) {
		// Test with content that has poor structure but still contains event information
		poorContent := `
Welcome to our website! We have lots of great activities for families.

There's an art class on December 15th at 2pm at the community center. It costs $20 and is good for kids ages 5-8.

Also, don't forget about movie night! It's every Friday at 7pm in the park. Totally free and fun for the whole family.

Swimming lessons are available Mondays and Wednesdays from 4-5pm. Call us to register. $15 per session.

We also have a holiday party coming up. December 22nd from 6-8pm. $5 per person. All ages welcome. Food and drinks provided.

For more information about any of these activities, please contact us at info@example.com or call 555-1234.
`

		attempt := &ExtractionAttempt{
			Method:    "test_poor_structure",
			Timestamp: time.Now(),
		}
		events := fc.extractEventsFromMarkdown(poorContent, attempt)

		// Even with poor structure, should extract some events
		if len(events) < 2 {
			t.Logf("Only extracted %d events from poorly structured content (may be expected)", len(events))
		}

		// Log what was extracted for analysis
		for i, event := range events {
			t.Logf("Extracted event %d: Title='%s', Date='%s', Time='%s', Location='%s', Price='%s'", 
				i+1, event.Title, event.Date, event.Time, event.Location, event.Price)
		}
	})

	t.Run("MixedContentTypes", func(t *testing.T) {
		// Test with content that mixes events with other information
		mixedContent := `
# Community Center Newsletter

## Upcoming Events

### Art Workshop for Kids
Date: December 15, 2024
Time: 10 AM - 12 PM
Location: Main Activity Room
Cost: $25 per child
Ages: 5-10 years

This hands-on workshop will teach children basic painting and drawing techniques.

## Facility Information

Our community center is open Monday through Friday from 8 AM to 8 PM. We offer various amenities including:
- Gymnasium
- Meeting rooms
- Kitchen facilities
- Playground

## Staff Directory

- Director: Jane Smith (jsmith@community.org)
- Program Coordinator: Bob Johnson (bjohnson@community.org)
- Maintenance: Mike Wilson (mwilson@community.org)

## More Events

### Holiday Party
When: December 22, 2024 at 6 PM
Where: Main Hall
Cost: $10 per family
All ages welcome! Food, games, and entertainment provided.

## Membership Information

Annual membership is $50 per family and includes access to all facilities and discounted event pricing.

### New Year's Eve Family Fun
Date: December 31, 2024
Time: 8 PM - 11 PM (kid-friendly early celebration!)
Location: Gymnasium
Cost: $15 per person, $40 per family
Ages: All ages

Ring in the new year with games, dancing, and a balloon drop at 10 PM!
`

		attempt := &ExtractionAttempt{
			Method:    "test_mixed_content",
			Timestamp: time.Now(),
		}
		events := fc.extractEventsFromMarkdown(mixedContent, attempt)

		if len(events) < 3 {
			t.Errorf("Expected at least 3 events from mixed content, got %d", len(events))
		}

		// Should extract actual events and ignore non-event content
		foundArtWorkshop := false
		foundHolidayParty := false
		foundNewYears := false

		for _, event := range events {
			title := strings.ToLower(event.Title)
			
			if strings.Contains(title, "art") {
				foundArtWorkshop = true
			}
			if strings.Contains(title, "holiday") || strings.Contains(title, "party") {
				foundHolidayParty = true
			}
			if strings.Contains(title, "new year") {
				foundNewYears = true
			}
			
			// Should not extract non-event content
			if strings.Contains(title, "staff") || strings.Contains(title, "membership") || strings.Contains(title, "facility") {
				t.Errorf("Should not extract non-event content as event: %s", event.Title)
			}
		}

		if !foundArtWorkshop {
			t.Error("Should find Art Workshop event")
		}
		if !foundHolidayParty {
			t.Error("Should find Holiday Party event")
		}
		if !foundNewYears {
			t.Error("Should find New Year's Eve event")
		}
	})
}

// TestMarkdownParsingEdgeCases tests various edge cases in markdown parsing
func TestMarkdownParsingEdgeCases(t *testing.T) {
	fc := &FireCrawlClient{}

	t.Run("EmptyContent", func(t *testing.T) {
		attempt := &ExtractionAttempt{
			Method:    "test_empty",
			Timestamp: time.Now(),
		}
		events := fc.extractEventsFromMarkdown("", attempt)
		
		if len(events) > 0 {
			t.Error("Should not extract events from empty content")
		}
	})

	t.Run("OnlyHeaders", func(t *testing.T) {
		headerOnlyContent := `
# Main Header
## Sub Header
### Another Header
#### Yet Another Header
`

		attempt := &ExtractionAttempt{
			Method:    "test_headers_only",
			Timestamp: time.Now(),
		}
		events := fc.extractEventsFromMarkdown(headerOnlyContent, attempt)
		
		if len(events) > 0 {
			t.Error("Should not extract events from header-only content")
		}
	})

	t.Run("VeryLongContent", func(t *testing.T) {
		// Create very long content with events scattered throughout
		var longContent strings.Builder
		longContent.WriteString("# Very Long Document\n\n")
		
		// Add lots of filler content
		for i := 0; i < 100; i++ {
			longContent.WriteString("This is paragraph ")
			longContent.WriteString(string(rune('0' + (i % 10))))
			longContent.WriteString(" with lots of text that doesn't contain event information. ")
			longContent.WriteString("It's just here to make the document very long and test performance. ")
		}
		
		// Add a real event in the middle
		longContent.WriteString("\n\n## Real Event\n")
		longContent.WriteString("Date: December 15, 2024\n")
		longContent.WriteString("Time: 2 PM\n")
		longContent.WriteString("Location: Test Location\n")
		longContent.WriteString("This is a real event buried in lots of content.\n\n")
		
		// Add more filler
		for i := 0; i < 100; i++ {
			longContent.WriteString("More filler content here. ")
		}

		attempt := &ExtractionAttempt{
			Method:    "test_long_content",
			Timestamp: time.Now(),
		}
		events := fc.extractEventsFromMarkdown(longContent.String(), attempt)

		// Should find the real event
		if len(events) == 0 {
			t.Error("Should find at least one event in very long content")
		}

		foundRealEvent := false
		for _, event := range events {
			if strings.Contains(strings.ToLower(event.Title), "real event") {
				foundRealEvent = true
				break
			}
		}

		if !foundRealEvent {
			t.Error("Should find the real event in long content")
		}
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		specialCharContent := `
# Events with Special Characters

## Art & Craft Workshop
Date: December 15, 2024 @ 2:00 PM
Location: Community Center (Building #3)
Cost: $25/child - includes all materials!
Ages: 5-10 years old

Join us for "Creative Fun" - an exciting workshop featuring:
• Painting & drawing
• Clay modeling
• Jewelry making

## Movie Night: "The Lion King"
Date: Friday, Dec 20th
Time: 7:00-9:30 PM
Location: Outdoor Theater (weather permitting*)
Cost: FREE! (donations welcome)
*Rain date: December 21st
`

		attempt := &ExtractionAttempt{
			Method:    "test_special_chars",
			Timestamp: time.Now(),
		}
		events := fc.extractEventsFromMarkdown(specialCharContent, attempt)

		if len(events) < 2 {
			t.Errorf("Expected at least 2 events with special characters, got %d", len(events))
		}

		// Check that special characters are handled properly
		for _, event := range events {
			if event.Title == "" {
				t.Error("Event title should not be empty despite special characters")
			}
			
			// Log for manual inspection
			t.Logf("Event with special chars: Title='%s', Date='%s', Price='%s'", 
				event.Title, event.Date, event.Price)
		}
	})
}