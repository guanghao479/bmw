//go:build integration

package services

import (
	"os"
	"strings"
	"testing"
	"time"
	"seattle-family-activities-scraper/internal/models"
)

// These are integration tests that make real S3 API calls
// Run with: go test -tags=integration ./internal/services -run TestS3 -v

func TestS3Client_RealAWSConnection(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client, err := NewS3Client()
	if err != nil {
		t.Skipf("Skipping S3 integration test - no AWS credentials: %v", err)
	}

	// Verify bucket configuration
	bucketName := client.GetBucketName()
	if bucketName == "" {
		t.Error("Bucket name should not be empty")
	}

	region := client.GetRegion()
	if region == "" {
		t.Error("Region should not be empty")
	}

	t.Logf("S3 Client configured: Bucket=%s, Region=%s", bucketName, region)
}

func TestS3Client_UploadDownloadActivities(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client, err := NewS3Client()
	if err != nil {
		t.Skipf("Skipping S3 integration test - no AWS credentials: %v", err)
	}

	// Create test activities
	activities := []models.Activity{
		{
			ID:          models.GenerateActivityID("Integration Test Activity", "2024-01-01", "Test Location"),
			Title:       "Integration Test Music Class",
			Description: "A test music class for S3 integration testing",
			Type:        models.TypeClass,
			Category:    models.CategoryArtsCreativity,
			Subcategory: "music",
			Schedule: models.Schedule{
				Type:      models.ScheduleTypeRecurring,
				StartDate: "2024-01-01",
				EndDate:   "2024-03-31",
				Frequency: "weekly",
				DaysOfWeek: []string{"tuesday"},
				Times: []models.TimeSlot{
					{
						StartTime: "10:00",
						EndTime:   "11:00",
						AgeGroup:  models.AgeGroupToddler,
					},
				},
				Duration: "60 minutes",
				Sessions: 12,
			},
			AgeGroups: []models.AgeGroup{
				{
					Category:    models.AgeGroupToddler,
					MinAge:      18,
					MaxAge:      36,
					Unit:        "months",
					Description: "18 months - 3 years",
				},
			},
			FamilyType: models.FamilyTypeParentChild,
			Location: models.Location{
				Name:         "Test Music Academy",
				Address:      "123 Test Street, Seattle, WA 98101",
				Neighborhood: "Test Hill",
				City:         "Seattle",
				Region:       "Seattle Metro",
				ZipCode:      "98101",
				VenueType:    models.VenueTypeIndoor,
				Accessibility: true,
				Parking:      "street",
			},
			Pricing: models.Pricing{
				Type:        models.PricingTypePaid,
				Cost:        150.0,
				Currency:    "USD",
				Unit:        "session",
				Description: "12-week session",
			},
			Registration: models.Registration{
				Required: true,
				Method:   "online",
				URL:      "https://test-academy.com/register",
				Status:   models.RegistrationStatusOpen,
			},
			Tags: []string{"music", "toddler", "test", "integration"},
			Provider: models.Provider{
				Name:     "Test Music Academy",
				Type:     "business",
				Website:  "https://test-academy.com",
				Verified: false,
			},
			Source: models.Source{
				URL:         "https://test-source.com",
				Domain:      "test-source.com",
				ScrapedAt:   time.Now(),
				LastChecked: time.Now(),
				Reliability: "test",
			},
			Featured:  false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Status:    models.ActivityStatusActive,
		},
	}

	// Test upload with timestamp key
	testKey := "integration-test/activities-" + time.Now().Format("2006-01-02T15-04-05Z") + ".json"
	
	uploadResult, err := client.UploadActivities(activities, testKey)
	if err != nil {
		t.Fatalf("Failed to upload activities: %v", err)
	}

	// Verify upload result
	if uploadResult.Key != testKey {
		t.Errorf("Expected key %s, got %s", testKey, uploadResult.Key)
	}

	if uploadResult.Size <= 0 {
		t.Error("Upload size should be positive")
	}

	if uploadResult.ContentType != "application/json" {
		t.Errorf("Expected content type 'application/json', got %s", uploadResult.ContentType)
	}

	if !strings.Contains(uploadResult.PublicURL, client.GetBucketName()) {
		t.Error("Public URL should contain bucket name")
	}

	t.Logf("Successfully uploaded: %s (%d bytes)", uploadResult.Key, uploadResult.Size)
	t.Logf("Public URL: %s", uploadResult.PublicURL)

	// Test download
	downloadedData, err := client.DownloadActivities(testKey)
	if err != nil {
		t.Fatalf("Failed to download activities: %v", err)
	}

	// Verify downloaded data
	if len(downloadedData.Activities) != 1 {
		t.Errorf("Expected 1 activity, got %d", len(downloadedData.Activities))
	}

	if downloadedData.Activities[0].Title != activities[0].Title {
		t.Errorf("Expected title %s, got %s", activities[0].Title, downloadedData.Activities[0].Title)
	}

	if downloadedData.Metadata.TotalActivities != 1 {
		t.Errorf("Expected metadata total 1, got %d", downloadedData.Metadata.TotalActivities)
	}

	t.Logf("Successfully downloaded and verified activities data")

	// Clean up - delete test file
	err = client.DeleteFile(testKey)
	if err != nil {
		t.Logf("Warning: Failed to clean up test file %s: %v", testKey, err)
	} else {
		t.Logf("Successfully cleaned up test file")
	}
}

func TestS3Client_FileOperations(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client, err := NewS3Client()
	if err != nil {
		t.Skipf("Skipping S3 integration test - no AWS credentials: %v", err)
	}

	testKey := "integration-test/file-ops-" + time.Now().Format("2006-01-02T15-04-05Z") + ".json"
	
	// Create minimal test data
	testActivities := []models.Activity{
		{
			ID:       "test_file_ops",
			Title:    "File Operations Test",
			Type:     models.TypeEvent,
			Category: models.CategoryFreeCommunity,
			Status:   models.ActivityStatusActive,
		},
	}

	// Test upload
	_, err = client.UploadActivities(testActivities, testKey)
	if err != nil {
		t.Fatalf("Failed to upload test data: %v", err)
	}

	// Test file exists
	exists, err := client.FileExists(testKey)
	if err != nil {
		t.Fatalf("Failed to check if file exists: %v", err)
	}

	if !exists {
		t.Error("File should exist after upload")
	}

	// Test get file info
	fileInfo, err := client.GetFileInfo(testKey)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	if fileInfo.Key != testKey {
		t.Errorf("Expected key %s, got %s", testKey, fileInfo.Key)
	}

	if fileInfo.Size <= 0 {
		t.Error("File size should be positive")
	}

	if fileInfo.ContentType != "application/json" {
		t.Errorf("Expected content type 'application/json', got %s", fileInfo.ContentType)
	}

	t.Logf("File info: Key=%s, Size=%d, ContentType=%s, LastModified=%s", 
		fileInfo.Key, fileInfo.Size, fileInfo.ContentType, fileInfo.LastModified)

	// Test public URL generation
	publicURL := client.GetPublicURL(testKey)
	expectedURL := "https://" + client.GetBucketName() + ".s3." + client.GetRegion() + ".amazonaws.com/" + testKey
	
	if publicURL != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, publicURL)
	}

	// Test delete
	err = client.DeleteFile(testKey)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Verify file no longer exists
	exists, err = client.FileExists(testKey)
	if err != nil {
		t.Fatalf("Failed to check if file exists after deletion: %v", err)
	}

	if exists {
		t.Error("File should not exist after deletion")
	}

	t.Logf("Successfully tested all file operations")
}

func TestS3Client_ListFiles(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client, err := NewS3Client()
	if err != nil {
		t.Skipf("Skipping S3 integration test - no AWS credentials: %v", err)
	}

	// List files with integration-test prefix
	files, err := client.ListFiles("integration-test/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}

	t.Logf("Found %d files with 'integration-test/' prefix", len(files))

	for i, file := range files {
		if i >= 5 { // Limit output
			t.Logf("... and %d more files", len(files)-5)
			break
		}
		t.Logf("File %d: %s (%d bytes, modified: %s)", i+1, file.Key, file.Size, file.LastModified)
	}

	// List all files (without prefix)
	allFiles, err := client.ListFiles("")
	if err != nil {
		t.Fatalf("Failed to list all files: %v", err)
	}

	t.Logf("Total files in bucket: %d", len(allFiles))
}

func TestS3Client_ScrapingDataWorkflow(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client, err := NewS3Client()
	if err != nil {
		t.Skipf("Skipping S3 integration test - no AWS credentials: %v", err)
	}

	timestamp := time.Now().Format("2006-01-02T15-04-05Z")

	// Test scraping run upload
	scrapingRun := &models.ScrapingRun{
		ID:                models.GenerateScrapingRunID(time.Now()),
		StartedAt:         time.Now().Add(-2 * time.Hour),
		CompletedAt:       time.Now(),
		Status:            models.ScrapingStatusCompleted,
		TotalSources:      3,
		SuccessfulSources: 2,
		FailedSources:     1,
		TotalActivities:   15,
		NewActivities:     12,
		TriggerType:       models.TriggerTypeScheduled,
		ScrapingVersion:   "1.0.0",
	}

	runKey := "integration-test/scraping-runs/run-" + timestamp + ".json"
	
	uploadResult, err := client.UploadScrapingRun(scrapingRun, runKey)
	if err != nil {
		t.Fatalf("Failed to upload scraping run: %v", err)
	}

	t.Logf("Uploaded scraping run: %s (%d bytes)", uploadResult.Key, uploadResult.Size)

	// Test scraping status upload
	scrapingStatus := &models.ScrapingStatus{
		LastRun:             scrapingRun,
		NextScheduledRun:    time.Now().Add(6 * time.Hour),
		TotalRuns:           25,
		SuccessfulRuns:      22,
		FailedRuns:          3,
		SystemHealth:        models.HealthStatusHealthy,
		LastHealthCheck:     time.Now(),
		TotalActivitiesEver: 450,
		UpdatedAt:           time.Now(),
	}

	statusKey := "integration-test/scraping-status/status-" + timestamp + ".json"
	
	statusResult, err := client.UploadScrapingStatus(scrapingStatus, statusKey)
	if err != nil {
		t.Fatalf("Failed to upload scraping status: %v", err)
	}

	t.Logf("Uploaded scraping status: %s (%d bytes)", statusResult.Key, statusResult.Size)

	// Test download and verify
	downloadedStatus, err := client.DownloadScrapingStatus(statusKey)
	if err != nil {
		t.Fatalf("Failed to download scraping status: %v", err)
	}

	if downloadedStatus.TotalRuns != 25 {
		t.Errorf("Expected 25 total runs, got %d", downloadedStatus.TotalRuns)
	}

	if downloadedStatus.SystemHealth != models.HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %s", downloadedStatus.SystemHealth)
	}

	// Clean up
	err = client.DeleteFile(runKey)
	if err != nil {
		t.Logf("Warning: Failed to clean up run file: %v", err)
	}

	err = client.DeleteFile(statusKey)
	if err != nil {
		t.Logf("Warning: Failed to clean up status file: %v", err)
	} else {
		t.Logf("Successfully cleaned up test files")
	}
}

func TestS3Client_LatestActivitiesWorkflow(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	client, err := NewS3Client()
	if err != nil {
		t.Skipf("Skipping S3 integration test - no AWS credentials: %v", err)
	}

	// Create test activities for "latest" workflow
	activities := []models.Activity{
		{
			ID:       "latest_test_1",
			Title:    "Latest Activities Test 1",
			Type:     models.TypeClass,
			Category: models.CategoryArtsCreativity,
			Status:   models.ActivityStatusActive,
		},
		{
			ID:       "latest_test_2",
			Title:    "Latest Activities Test 2",
			Type:     models.TypeEvent,
			Category: models.CategoryFreeCommunity,
			Status:   models.ActivityStatusActive,
		},
	}

	// Test uploading as latest
	result, err := client.UploadLatestActivities(activities)
	if err != nil {
		t.Fatalf("Failed to upload latest activities: %v", err)
	}

	if result.Key != "activities/latest.json" {
		t.Errorf("Expected key 'activities/latest.json', got %s", result.Key)
	}

	t.Logf("Uploaded latest activities: %s", result.PublicURL)

	// Test backup functionality
	backupResult, err := client.BackupActivities(activities)
	if err != nil {
		t.Fatalf("Failed to backup activities: %v", err)
	}

	if !strings.HasPrefix(backupResult.Key, "activities/backups/") {
		t.Errorf("Backup key should start with 'activities/backups/', got %s", backupResult.Key)
	}

	t.Logf("Created backup: %s", backupResult.Key)

	// Clean up backup (leave latest.json for frontend testing)
	err = client.DeleteFile(backupResult.Key)
	if err != nil {
		t.Logf("Warning: Failed to clean up backup file: %v", err)
	} else {
		t.Logf("Successfully cleaned up backup file")
	}
}