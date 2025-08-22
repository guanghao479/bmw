package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"seattle-family-activities-scraper/internal/models"
)

// DynamoDBService provides CRUD operations for all DynamoDB tables
type DynamoDBService struct {
	client             *dynamodb.Client
	familyActivitiesTable string
	sourceManagementTable string
	scrapingOperationsTable string
}

// NewDynamoDBService creates a new DynamoDB service instance
func NewDynamoDBService(client *dynamodb.Client, familyActivitiesTable, sourceManagementTable, scrapingOperationsTable string) *DynamoDBService {
	return &DynamoDBService{
		client:                  client,
		familyActivitiesTable:   familyActivitiesTable,
		sourceManagementTable:   sourceManagementTable,
		scrapingOperationsTable: scrapingOperationsTable,
	}
}

// Family Activities Table Operations

// CreateFamilyActivity stores a family activity in DynamoDB
func (s *DynamoDBService) CreateFamilyActivity(ctx context.Context, activity *models.FamilyActivity) error {
	// Set timestamps
	now := time.Now()
	activity.CreatedAt = now
	activity.UpdatedAt = now

	// Generate GSI keys
	s.populateFamilyActivityGSIKeys(activity)

	// Marshal to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(activity)
	if err != nil {
		return fmt.Errorf("failed to marshal family activity: %w", err)
	}

	// Put item
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.familyActivitiesTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create family activity: %w", err)
	}

	return nil
}

// GetFamilyActivity retrieves a family activity by primary key
func (s *DynamoDBService) GetFamilyActivity(ctx context.Context, pk, sk string) (*models.FamilyActivity, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.familyActivitiesTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get family activity: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("family activity not found")
	}

	var activity models.FamilyActivity
	err = attributevalue.UnmarshalMap(result.Item, &activity)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal family activity: %w", err)
	}

	return &activity, nil
}

// UpdateFamilyActivity updates an existing family activity
func (s *DynamoDBService) UpdateFamilyActivity(ctx context.Context, activity *models.FamilyActivity) error {
	// Update timestamp
	activity.UpdatedAt = time.Now()

	// Generate GSI keys
	s.populateFamilyActivityGSIKeys(activity)

	// Marshal to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(activity)
	if err != nil {
		return fmt.Errorf("failed to marshal family activity: %w", err)
	}

	// Put item (upsert)
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.familyActivitiesTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to update family activity: %w", err)
	}

	return nil
}

// DeleteFamilyActivity removes a family activity
func (s *DynamoDBService) DeleteFamilyActivity(ctx context.Context, pk, sk string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.familyActivitiesTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete family activity: %w", err)
	}

	return nil
}

// QueryFamilyActivitiesByLocation queries activities by location using GSI
func (s *DynamoDBService) QueryFamilyActivitiesByLocation(ctx context.Context, region, city string, limit int32) ([]models.FamilyActivity, error) {
	locationKey := models.GenerateLocationKey(region, city)

	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.familyActivitiesTable),
		IndexName:              aws.String("location-date-index"),
		KeyConditionExpression: aws.String("LocationKey = :locationKey"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":locationKey": &types.AttributeValueMemberS{Value: locationKey},
		},
		Limit: aws.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query activities by location: %w", err)
	}

	var activities []models.FamilyActivity
	err = attributevalue.UnmarshalListOfMaps(result.Items, &activities)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal activities: %w", err)
	}

	return activities, nil
}

// QueryFamilyActivitiesByCategory queries activities by category and age group using GSI
func (s *DynamoDBService) QueryFamilyActivitiesByCategory(ctx context.Context, category, ageGroup string, limit int32) ([]models.FamilyActivity, error) {
	categoryAgeKey := models.GenerateCategoryAgeKey(category, ageGroup)

	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.familyActivitiesTable),
		IndexName:              aws.String("category-age-index"),
		KeyConditionExpression: aws.String("CategoryAgeKey = :categoryAgeKey"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":categoryAgeKey": &types.AttributeValueMemberS{Value: categoryAgeKey},
		},
		Limit: aws.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query activities by category: %w", err)
	}

	var activities []models.FamilyActivity
	err = attributevalue.UnmarshalListOfMaps(result.Items, &activities)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal activities: %w", err)
	}

	return activities, nil
}

// QueryFamilyActivitiesByVenue queries activities at a specific venue using GSI
func (s *DynamoDBService) QueryFamilyActivitiesByVenue(ctx context.Context, venueID string, limit int32) ([]models.FamilyActivity, error) {
	venueKey := models.GenerateVenueKey(venueID)

	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.familyActivitiesTable),
		IndexName:              aws.String("venue-activity-index"),
		KeyConditionExpression: aws.String("VenueKey = :venueKey"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":venueKey": &types.AttributeValueMemberS{Value: venueKey},
		},
		Limit: aws.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query activities by venue: %w", err)
	}

	var activities []models.FamilyActivity
	err = attributevalue.UnmarshalListOfMaps(result.Items, &activities)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal activities: %w", err)
	}

	return activities, nil
}

// Source Management Table Operations

// CreateSourceSubmission creates a new source submission
func (s *DynamoDBService) CreateSourceSubmission(ctx context.Context, submission *models.SourceSubmission) error {
	// Set timestamps and keys
	now := time.Now()
	submission.SubmittedAt = now
	submission.PK = models.CreateSourcePK(submission.SourceID)
	submission.SK = models.CreateSourceSubmissionSK()
	submission.StatusKey = models.GenerateSourceStatusKey(submission.Status)
	submission.PriorityKey = models.GenerateSourcePriorityKey(submission.Priority, submission.SourceID)

	// Marshal to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(submission)
	if err != nil {
		return fmt.Errorf("failed to marshal source submission: %w", err)
	}

	// Put item
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.sourceManagementTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create source submission: %w", err)
	}

	return nil
}

// GetSourceSubmission retrieves a source submission
func (s *DynamoDBService) GetSourceSubmission(ctx context.Context, sourceID string) (*models.SourceSubmission, error) {
	pk := models.CreateSourcePK(sourceID)
	sk := models.CreateSourceSubmissionSK()

	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.sourceManagementTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get source submission: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("source submission not found")
	}

	var submission models.SourceSubmission
	err = attributevalue.UnmarshalMap(result.Item, &submission)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal source submission: %w", err)
	}

	return &submission, nil
}

// UpdateSourceSubmission updates an existing source submission
func (s *DynamoDBService) UpdateSourceSubmission(ctx context.Context, submission *models.SourceSubmission) error {
	// Set updated timestamp
	now := time.Now()
	submission.UpdatedAt = now

	// Marshal to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(submission)
	if err != nil {
		return fmt.Errorf("failed to marshal source submission: %w", err)
	}

	// Put item (overwrites existing)
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.sourceManagementTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to update source submission: %w", err)
	}

	return nil
}

// CreateSourceAnalysis stores analysis results
func (s *DynamoDBService) CreateSourceAnalysis(ctx context.Context, analysis *models.SourceAnalysis) error {
	// Set timestamps and keys
	now := time.Now()
	analysis.AnalysisCompletedAt = now
	analysis.PK = models.CreateSourcePK(analysis.SourceID)
	analysis.SK = models.CreateSourceAnalysisSK()

	// Marshal to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(analysis)
	if err != nil {
		return fmt.Errorf("failed to marshal source analysis: %w", err)
	}

	// Put item
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.sourceManagementTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create source analysis: %w", err)
	}

	return nil
}

// GetSourceAnalysis retrieves analysis results
func (s *DynamoDBService) GetSourceAnalysis(ctx context.Context, sourceID string) (*models.SourceAnalysis, error) {
	pk := models.CreateSourcePK(sourceID)
	sk := models.CreateSourceAnalysisSK()

	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.sourceManagementTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get source analysis: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("source analysis not found")
	}

	var analysis models.SourceAnalysis
	err = attributevalue.UnmarshalMap(result.Item, &analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal source analysis: %w", err)
	}

	return &analysis, nil
}

// CreateSourceConfig creates production configuration for an active source
func (s *DynamoDBService) CreateSourceConfig(ctx context.Context, config *models.DynamoSourceConfig) error {
	// Set timestamps and keys
	now := time.Now()
	config.ActivatedAt = now
	config.LastModified = now
	config.PK = models.CreateSourcePK(config.SourceID)
	config.SK = models.CreateSourceConfigSK()
	config.StatusKey = models.GenerateSourceStatusKey(config.Status)
	config.PriorityKey = models.GenerateSourcePriorityKey(config.ScrapingConfig.Priority, config.SourceID)

	// Marshal to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(config)
	if err != nil {
		return fmt.Errorf("failed to marshal source config: %w", err)
	}

	// Put item
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.sourceManagementTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create source config: %w", err)
	}

	return nil
}

// GetSourceConfig retrieves production configuration
func (s *DynamoDBService) GetSourceConfig(ctx context.Context, sourceID string) (*models.DynamoSourceConfig, error) {
	pk := models.CreateSourcePK(sourceID)
	sk := models.CreateSourceConfigSK()

	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.sourceManagementTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get source config: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("source config not found")
	}

	var config models.DynamoSourceConfig
	err = attributevalue.UnmarshalMap(result.Item, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal source config: %w", err)
	}

	return &config, nil
}

// QuerySourcesByStatus queries sources by status using GSI
func (s *DynamoDBService) QuerySourcesByStatus(ctx context.Context, status string, limit int32) ([]models.SourceSubmission, error) {
	statusKey := models.GenerateSourceStatusKey(status)

	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.sourceManagementTable),
		IndexName:              aws.String("status-priority-index"),
		KeyConditionExpression: aws.String("StatusKey = :statusKey"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":statusKey": &types.AttributeValueMemberS{Value: statusKey},
		},
		Limit: aws.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query sources by status: %w", err)
	}

	var sources []models.SourceSubmission
	err = attributevalue.UnmarshalListOfMaps(result.Items, &sources)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal sources: %w", err)
	}

	return sources, nil
}

// Scraping Operations Table Operations

// CreateScrapingTask creates a new scraping task
func (s *DynamoDBService) CreateScrapingTask(ctx context.Context, task *models.ScrapingTask) error {
	// Set timestamps and TTL
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now
	
	// Set TTL (90 days from now)
	task.TTL = models.CalculateTTL(90 * 24 * time.Hour)

	// Generate GSI keys
	task.NextRunKey = models.GenerateNextRunKey(task.ScheduledTime)
	task.PrioritySourceKey = models.GeneratePrioritySourceKey(task.Priority, task.SourceID, task.TaskID)

	// Marshal to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(task)
	if err != nil {
		return fmt.Errorf("failed to marshal scraping task: %w", err)
	}

	// Put item
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.scrapingOperationsTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create scraping task: %w", err)
	}

	return nil
}

// GetScrapingTask retrieves a scraping task
func (s *DynamoDBService) GetScrapingTask(ctx context.Context, taskID string) (*models.ScrapingTask, error) {
	pk := models.CreateTaskPK(taskID)
	sk := models.CreateTaskSK("medium", "default-source", taskID) // This would need proper implementation

	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.scrapingOperationsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get scraping task: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("scraping task not found")
	}

	var task models.ScrapingTask
	err = attributevalue.UnmarshalMap(result.Item, &task)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal scraping task: %w", err)
	}

	return &task, nil
}

// QueryNextScrapingTasks queries tasks ready to run using GSI
func (s *DynamoDBService) QueryNextScrapingTasks(ctx context.Context, maxTime time.Time, limit int32) ([]models.ScrapingTask, error) {
	nextRunKey := models.GenerateNextRunKey(maxTime)

	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.scrapingOperationsTable),
		IndexName:              aws.String("next-run-index"),
		KeyConditionExpression: aws.String("NextRunKey <= :nextRunKey"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":nextRunKey": &types.AttributeValueMemberS{Value: nextRunKey},
		},
		Limit: aws.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query next scraping tasks: %w", err)
	}

	var tasks []models.ScrapingTask
	err = attributevalue.UnmarshalListOfMaps(result.Items, &tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal scraping tasks: %w", err)
	}

	return tasks, nil
}

// Helper function to populate GSI keys for family activities
func (s *DynamoDBService) populateFamilyActivityGSIKeys(activity *models.FamilyActivity) {
	// Generate location key
	if activity.Location.Region != "" && activity.Location.City != "" {
		activity.LocationKey = models.GenerateLocationKey(activity.Location.Region, activity.Location.City)
	}

	// Generate category-age key (use first age group if available)
	if activity.Category != "" && len(activity.AgeGroups) > 0 {
		activity.CategoryAgeKey = models.GenerateCategoryAgeKey(activity.Category, activity.AgeGroups[0].Category)
	}

	// Generate venue key for events and programs
	switch activity.EntityType {
	case models.EntityTypeEvent, models.EntityTypeProgram:
		// For events and programs, we'd need to extract venue_id from the specific type
		// This would be populated when creating typed entities (Event, Program)
	}

	// Generate provider key
	if activity.ProviderID != "" {
		activity.ProviderKey = "PROVIDER#" + activity.ProviderID
	}

	// Generate type-status key
	if activity.EntityType != "" && activity.Status != "" {
		activity.TypeStatusKey = models.GenerateTypeStatusKey(activity.EntityType, activity.Status, activity.EntityID)
	}
}