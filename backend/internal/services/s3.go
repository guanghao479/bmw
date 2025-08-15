package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"seattle-family-activities-scraper/internal/models"
)

// S3Client handles all S3 operations for the Seattle Family Activities data
type S3Client struct {
	client     *s3.Client
	bucketName string
	region     string
}

// S3Config holds configuration for S3 client
type S3Config struct {
	BucketName string
	Region     string
	Profile    string // AWS profile to use
}

// S3FileInfo represents metadata about files in S3
type S3FileInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	ETag         string    `json:"etag"`
	ContentType  string    `json:"content_type"`
}

// S3UploadResult represents the result of an S3 upload operation
type S3UploadResult struct {
	Key          string    `json:"key"`
	Location     string    `json:"location"`
	ETag         string    `json:"etag"`
	Size         int64     `json:"size"`
	UploadedAt   time.Time `json:"uploaded_at"`
	ContentType  string    `json:"content_type"`
	PublicURL    string    `json:"public_url"`
}

// NewS3Client creates a new S3 client with AWS SDK v2
func NewS3Client() (*S3Client, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Get bucket name from environment or use default
	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		bucketName = "seattle-family-activities-mvp-data-usw2"
	}

	return &S3Client{
		client:     s3.NewFromConfig(cfg),
		bucketName: bucketName,
		region:     cfg.Region,
	}, nil
}

// NewS3ClientWithConfig creates an S3 client with custom configuration
func NewS3ClientWithConfig(s3Config S3Config) (*S3Client, error) {
	var cfg aws.Config
	var err error

	if s3Config.Profile != "" {
		// Load config with specific profile
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithSharedConfigProfile(s3Config.Profile),
		)
	} else {
		// Load default config
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Override region if specified
	if s3Config.Region != "" {
		cfg.Region = s3Config.Region
	}

	return &S3Client{
		client:     s3.NewFromConfig(cfg),
		bucketName: s3Config.BucketName,
		region:     cfg.Region,
	}, nil
}

// UploadActivities uploads activities data to S3 as JSON
func (s *S3Client) UploadActivities(activities []models.Activity, key string) (*S3UploadResult, error) {
	// Create metadata
	metadata := models.NewActivitiesMetadata(len(activities), []string{"scraped"})
	
	// Create output structure
	output := models.ActivitiesOutput{
		Metadata:   metadata,
		Activities: activities,
	}

	// Marshal to JSON with proper formatting
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal activities to JSON: %w", err)
	}

	return s.uploadJSON(jsonData, key, "application/json")
}

// UploadScrapingStatus uploads scraping status/monitoring data to S3
func (s *S3Client) UploadScrapingStatus(status *models.ScrapingStatus, key string) (*S3UploadResult, error) {
	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scraping status to JSON: %w", err)
	}

	return s.uploadJSON(jsonData, key, "application/json")
}

// UploadScrapingRun uploads scraping run results to S3
func (s *S3Client) UploadScrapingRun(run *models.ScrapingRun, key string) (*S3UploadResult, error) {
	jsonData, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scraping run to JSON: %w", err)
	}

	return s.uploadJSON(jsonData, key, "application/json")
}

// uploadJSON is a helper method to upload JSON data to S3
func (s *S3Client) uploadJSON(data []byte, key, contentType string) (*S3UploadResult, error) {
	// Ensure key doesn't start with /
	key = strings.TrimPrefix(key, "/")

	uploadInput := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		// Set cache control for frontend consumption
		CacheControl: aws.String("public, max-age=300"), // 5 minutes
		// Add metadata
		Metadata: map[string]string{
			"uploaded-by":   "seattle-family-activities-scraper",
			"content-type":  contentType,
			"upload-time":   time.Now().UTC().Format(time.RFC3339),
		},
	}

	result, err := s.client.PutObject(context.TODO(), uploadInput)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Construct public URL
	publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, key)

	return &S3UploadResult{
		Key:         key,
		Location:    publicURL,
		ETag:        strings.Trim(*result.ETag, `"`),
		Size:        int64(len(data)),
		UploadedAt:  time.Now(),
		ContentType: contentType,
		PublicURL:   publicURL,
	}, nil
}

// DownloadActivities downloads and parses activities JSON from S3
func (s *S3Client) DownloadActivities(key string) (*models.ActivitiesOutput, error) {
	data, err := s.downloadJSON(key)
	if err != nil {
		return nil, err
	}

	var output models.ActivitiesOutput
	err = json.Unmarshal(data, &output)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal activities JSON: %w", err)
	}

	return &output, nil
}

// DownloadScrapingStatus downloads scraping status from S3
func (s *S3Client) DownloadScrapingStatus(key string) (*models.ScrapingStatus, error) {
	data, err := s.downloadJSON(key)
	if err != nil {
		return nil, err
	}

	var status models.ScrapingStatus
	err = json.Unmarshal(data, &status)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal scraping status JSON: %w", err)
	}

	return &status, nil
}

// downloadJSON is a helper method to download JSON data from S3
func (s *S3Client) downloadJSON(key string) ([]byte, error) {
	// Ensure key doesn't start with /
	key = strings.TrimPrefix(key, "/")

	getInput := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(context.TODO(), getInput)
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object body: %w", err)
	}

	return data, nil
}

// ListFiles lists all files in the S3 bucket with optional prefix filter
func (s *S3Client) ListFiles(prefix string) ([]S3FileInfo, error) {
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
	}

	if prefix != "" {
		listInput.Prefix = aws.String(prefix)
	}

	result, err := s.client.ListObjectsV2(context.TODO(), listInput)
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	files := make([]S3FileInfo, 0, len(result.Contents))
	for _, obj := range result.Contents {
		files = append(files, S3FileInfo{
			Key:          *obj.Key,
			Size:         obj.Size,
			LastModified: *obj.LastModified,
			ETag:         strings.Trim(*obj.ETag, `"`),
		})
	}

	return files, nil
}

// DeleteFile deletes a file from S3
func (s *S3Client) DeleteFile(key string) error {
	key = strings.TrimPrefix(key, "/")

	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(context.TODO(), deleteInput)
	if err != nil {
		return fmt.Errorf("failed to delete S3 object: %w", err)
	}

	return nil
}

// FileExists checks if a file exists in S3
func (s *S3Client) FileExists(key string) (bool, error) {
	key = strings.TrimPrefix(key, "/")

	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	_, err := s.client.HeadObject(context.TODO(), headInput)
	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if S3 object exists: %w", err)
	}

	return true, nil
}

// GetFileInfo gets metadata about a file in S3
func (s *S3Client) GetFileInfo(key string) (*S3FileInfo, error) {
	key = strings.TrimPrefix(key, "/")

	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	result, err := s.client.HeadObject(context.TODO(), headInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object metadata: %w", err)
	}

	contentType := ""
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	return &S3FileInfo{
		Key:          key,
		Size:         result.ContentLength,
		LastModified: *result.LastModified,
		ETag:         strings.Trim(*result.ETag, `"`),
		ContentType:  contentType,
	}, nil
}

// GetBucketName returns the configured bucket name
func (s *S3Client) GetBucketName() string {
	return s.bucketName
}

// GetRegion returns the configured AWS region
func (s *S3Client) GetRegion() string {
	return s.region
}

// GetPublicURL generates the public URL for an S3 object
func (s *S3Client) GetPublicURL(key string) string {
	key = strings.TrimPrefix(key, "/")
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, key)
}

// UploadActivitiesWithTimestamp uploads activities with a timestamp-based key
func (s *S3Client) UploadActivitiesWithTimestamp(activities []models.Activity) (*S3UploadResult, error) {
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	key := fmt.Sprintf("activities/%s.json", timestamp)
	return s.UploadActivities(activities, key)
}

// UploadLatestActivities uploads activities as the "latest" version for frontend consumption
func (s *S3Client) UploadLatestActivities(activities []models.Activity) (*S3UploadResult, error) {
	return s.UploadActivities(activities, "activities/latest.json")
}

// CreateDataStructure creates the standard directory structure in S3
func (s *S3Client) CreateDataStructure() error {
	// Create placeholder files to establish directory structure
	placeholders := map[string]string{
		"activities/.keep":       "Directory for activity data files",
		"scraping-runs/.keep":    "Directory for scraping run logs", 
		"scraping-status/.keep":  "Directory for scraping status files",
		"monitoring/.keep":       "Directory for monitoring and metrics",
	}

	for key, content := range placeholders {
		_, err := s.uploadJSON([]byte(content), key, "text/plain")
		if err != nil {
			return fmt.Errorf("failed to create directory structure for %s: %w", key, err)
		}
	}

	return nil
}

// BackupActivities creates a backup of current activities with timestamp
func (s *S3Client) BackupActivities(activities []models.Activity) (*S3UploadResult, error) {
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	key := fmt.Sprintf("activities/backups/%s.json", timestamp)
	return s.UploadActivities(activities, key)
}