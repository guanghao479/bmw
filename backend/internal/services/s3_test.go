package services

import (
	"strings"
	"testing"
	"time"
	"seattle-family-activities-scraper/internal/models"
)

func TestS3Client_Configuration(t *testing.T) {
	// Test default configuration
	client, err := NewS3Client()
	if err != nil {
		t.Skipf("Skipping S3 test - no AWS credentials available: %v", err)
	}
	
	if client.GetBucketName() == "" {
		t.Error("Bucket name should not be empty")
	}
	
	if client.GetRegion() == "" {
		t.Error("Region should not be empty")
	}
	
	// Test custom configuration
	config := S3Config{
		BucketName: "test-bucket",
		Region:     "us-west-2",
	}
	
	customClient, err := NewS3ClientWithConfig(config)
	if err != nil {
		t.Skipf("Skipping S3 custom config test - no AWS credentials: %v", err)
	}
	
	if customClient.GetBucketName() != "test-bucket" {
		t.Errorf("Expected bucket name 'test-bucket', got %s", customClient.GetBucketName())
	}
}

func TestS3Client_PublicURL(t *testing.T) {
	config := S3Config{
		BucketName: "test-bucket",
		Region:     "us-west-2",
	}
	
	client, err := NewS3ClientWithConfig(config)
	if err != nil {
		t.Skipf("Skipping S3 test - no AWS credentials: %v", err)
	}
	
	tests := []struct {
		key      string
		expected string
	}{
		{
			key:      "activities/latest.json",
			expected: "https://test-bucket.s3.us-west-2.amazonaws.com/activities/latest.json",
		},
		{
			key:      "/activities/latest.json", // Leading slash should be handled
			expected: "https://test-bucket.s3.us-west-2.amazonaws.com/activities/latest.json",
		},
		{
			key:      "scraping-runs/2024-01-15T10-30-00Z.json",
			expected: "https://test-bucket.s3.us-west-2.amazonaws.com/scraping-runs/2024-01-15T10-30-00Z.json",
		},
	}
	
	for _, test := range tests {
		url := client.GetPublicURL(test.key)
		if url != test.expected {
			t.Errorf("For key %s, expected URL %s, got %s", test.key, test.expected, url)
		}
	}
}

func TestS3Client_ActivitiesStructure(t *testing.T) {
	// Create sample activities for testing
	activities := []models.Activity{
		{
			ID:          "act_test1234",
			Title:       "Test Music Class",
			Description: "A test music class",
			Type:        models.TypeClass,
			Category:    models.CategoryArtsCreativity,
			AgeGroups: []models.AgeGroup{
				{
					Category:    models.AgeGroupToddler,
					MinAge:      18,
					MaxAge:      36,
					Unit:        "months",
					Description: "18 months - 3 years",
				},
			},
			Location: models.Location{
				Name:    "Test Music Academy",
				Address: "123 Test Street, Seattle, WA 98101",
				City:    "Seattle",
				Region:  "Seattle Metro",
			},
			Pricing: models.Pricing{
				Type:     models.PricingTypePaid,
				Cost:     100.0,
				Currency: "USD",
			},
			Provider: models.Provider{
				Name: "Test Academy",
				Type: "business",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Status:    models.ActivityStatusActive,
		},
	}
	
	// Test that we can create the JSON structure without errors
	config := S3Config{
		BucketName: "test-bucket", 
		Region:     "us-west-2",
	}
	
	client, err := NewS3ClientWithConfig(config)
	if err != nil {
		t.Skipf("Skipping S3 test - no AWS credentials: %v", err)
	}
	
	// Test different upload methods (without actually uploading)
	testCases := []struct {
		name string
		test func() error
	}{
		{
			name: "UploadActivities",
			test: func() error {
				_, err := client.UploadActivities(activities, "test/activities.json")
				return err
			},
		},
		{
			name: "UploadActivitiesWithTimestamp", 
			test: func() error {
				_, err := client.UploadActivitiesWithTimestamp(activities)
				return err
			},
		},
		{
			name: "UploadLatestActivities",
			test: func() error {
				_, err := client.UploadLatestActivities(activities)
				return err
			},
		},
		{
			name: "BackupActivities",
			test: func() error {
				_, err := client.BackupActivities(activities)
				return err
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.test()
			// We expect AWS errors since we're not actually connected
			// but we shouldn't get JSON marshaling or validation errors
			if err != nil && !isAWSConnectionError(err) {
				t.Errorf("%s failed with non-AWS error: %v", tc.name, err)
			}
		})
	}
}

func TestS3Client_ScrapingDataStructure(t *testing.T) {
	config := S3Config{
		BucketName: "test-bucket",
		Region:     "us-west-2", 
	}
	
	client, err := NewS3ClientWithConfig(config)
	if err != nil {
		t.Skipf("Skipping S3 test - no AWS credentials: %v", err)
	}
	
	// Test scraping status structure
	status := &models.ScrapingStatus{
		LastRun: &models.ScrapingRun{
			ID:        "run_test1234",
			StartedAt: time.Now(),
			Status:    models.ScrapingStatusCompleted,
		},
		NextScheduledRun:    time.Now().Add(6 * time.Hour),
		TotalRuns:           10,
		SuccessfulRuns:      8,
		FailedRuns:          2,
		SystemHealth:        models.HealthStatusHealthy,
		LastHealthCheck:     time.Now(),
		UpdatedAt:           time.Now(),
	}
	
	_, err = client.UploadScrapingStatus(status, "test/status.json")
	if err != nil && !isAWSConnectionError(err) {
		t.Errorf("UploadScrapingStatus failed with non-AWS error: %v", err)
	}
	
	// Test scraping run structure
	run := &models.ScrapingRun{
		ID:                "run_test5678",
		StartedAt:         time.Now().Add(-1 * time.Hour),
		CompletedAt:       time.Now(),
		Status:            models.ScrapingStatusCompleted,
		TotalSources:      4,
		SuccessfulSources: 3,
		FailedSources:     1,
		TotalActivities:   15,
		NewActivities:     12,
		TriggerType:       models.TriggerTypeScheduled,
		Jobs: []models.ScrapingJob{
			{
				ID:              "job_test1111",
				SourceURL:       "https://test.com",
				Domain:          "test.com",
				Status:          models.ScrapingStatusCompleted,
				ActivitiesFound: 5,
				ActivitiesNew:   4,
			},
		},
	}
	
	_, err = client.UploadScrapingRun(run, "test/run.json")
	if err != nil && !isAWSConnectionError(err) {
		t.Errorf("UploadScrapingRun failed with non-AWS error: %v", err)
	}
}

func TestS3Client_KeyHandling(t *testing.T) {
	config := S3Config{
		BucketName: "test-bucket",
		Region:     "us-west-2",
	}
	
	client, err := NewS3ClientWithConfig(config)
	if err != nil {
		t.Skipf("Skipping S3 test - no AWS credentials: %v", err)
	}
	
	// Test key normalization (leading slash removal)
	testKeys := []string{
		"activities/latest.json",
		"/activities/latest.json",
		"scraping-runs/test.json",
		"/scraping-runs/test.json",
	}
	
	for _, key := range testKeys {
		url := client.GetPublicURL(key)
		// URL should not contain double slashes after domain
		if strings.Contains(url, "amazonaws.com//") {
			t.Errorf("Public URL contains double slash for key %s: %s", key, url)
		}
	}
}

func TestS3Client_TimestampKeys(t *testing.T) {
	activities := []models.Activity{
		{
			ID:       "test_activity",
			Title:    "Test Activity",
			Type:     models.TypeClass,
			Category: models.CategoryArtsCreativity,
			Status:   models.ActivityStatusActive,
		},
	}
	
	config := S3Config{
		BucketName: "test-bucket",
		Region:     "us-west-2",
	}
	
	client, err := NewS3ClientWithConfig(config)
	if err != nil {
		t.Skipf("Skipping S3 test - no AWS credentials: %v", err)
	}
	
	// Test timestamp-based uploads
	_, err = client.UploadActivitiesWithTimestamp(activities)
	if err != nil && !isAWSConnectionError(err) {
		t.Errorf("UploadActivitiesWithTimestamp failed: %v", err)
	}
	
	_, err = client.BackupActivities(activities)
	if err != nil && !isAWSConnectionError(err) {
		t.Errorf("BackupActivities failed: %v", err)
	}
}

// Helper function to check if error is AWS connection related
func isAWSConnectionError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	awsErrors := []string{
		"no such host",
		"connection refused", 
		"timeout",
		"credential",
		"no AWS profile",
		"unable to load SDK config",
		"NoCredentialProviders",
		"ExpiredToken",
		"InvalidAccessKeyId",
		"SignatureDoesNotMatch",
		"AccessDenied",
		"NoSuchBucket",
	}
	
	for _, awsErr := range awsErrors {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(awsErr)) {
			return true
		}
	}
	
	return false
}