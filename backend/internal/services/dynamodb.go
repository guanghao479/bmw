package services

import (
	"context"
	"fmt"
	"log"
	"sort"
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
	adminEventsTable string
}

// NewDynamoDBService creates a new DynamoDB service instance
func NewDynamoDBService(client *dynamodb.Client, familyActivitiesTable, sourceManagementTable, scrapingOperationsTable, adminEventsTable string) *DynamoDBService {
	return &DynamoDBService{
		client:                  client,
		familyActivitiesTable:   familyActivitiesTable,
		sourceManagementTable:   sourceManagementTable,
		scrapingOperationsTable: scrapingOperationsTable,
		adminEventsTable:        adminEventsTable,
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
	submission.UpdatedAt = now
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

// BatchPutActivities stores multiple activities in batches (for task executor)
func (s *DynamoDBService) BatchPutActivities(ctx context.Context, activities []*models.Activity) error {
	if len(activities) == 0 {
		return nil
	}

	// Convert Activity to FamilyActivity format for storage
	var familyActivities []*models.FamilyActivity
	for _, activity := range activities {
		familyActivity := s.convertActivityToFamilyActivity(activity)
		familyActivities = append(familyActivities, familyActivity)
	}

	// Process in batches of 25 (DynamoDB limit)
	batchSize := 25
	for i := 0; i < len(familyActivities); i += batchSize {
		end := i + batchSize
		if end > len(familyActivities) {
			end = len(familyActivities)
		}

		batch := familyActivities[i:end]
		if err := s.batchWriteFamilyActivities(ctx, batch); err != nil {
			return fmt.Errorf("failed to write batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// batchWriteFamilyActivities writes a batch of family activities
func (s *DynamoDBService) batchWriteFamilyActivities(ctx context.Context, activities []*models.FamilyActivity) error {
	writeRequests := make([]types.WriteRequest, 0, len(activities))

	for _, activity := range activities {
		// Set timestamps
		now := time.Now()
		activity.CreatedAt = now
		activity.UpdatedAt = now

		// Populate GSI keys
		s.populateFamilyActivityGSIKeys(activity)

		// Marshal activity
		item, err := attributevalue.MarshalMap(activity)
		if err != nil {
			return fmt.Errorf("failed to marshal activity %s: %w", activity.EntityID, err)
		}

		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: item},
		})
	}

	// Execute batch write
	_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			s.familyActivitiesTable: writeRequests,
		},
	})

	return err
}

// GetAllActivities retrieves all activities from the family activities table (for S3 export)
func (s *DynamoDBService) GetAllActivities(ctx context.Context) ([]*models.Activity, error) {
	result, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(s.familyActivitiesTable),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan activities: %w", err)
	}

	var familyActivities []models.FamilyActivity
	err = attributevalue.UnmarshalListOfMaps(result.Items, &familyActivities)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal activities: %w", err)
	}

	// Convert to Activity format
	var activities []*models.Activity
	for _, fa := range familyActivities {
		activity := s.convertFamilyActivityToActivity(&fa)
		activities = append(activities, activity)
	}

	return activities, nil
}

// convertActivityToFamilyActivity converts a simple Activity to the complex FamilyActivity format
func (s *DynamoDBService) convertActivityToFamilyActivity(activity *models.Activity) *models.FamilyActivity {
	// TODO: Implement proper conversion when needed
	// For now, return a minimal FamilyActivity to satisfy the interface
	return &models.FamilyActivity{
		EntityID:    activity.ID,
		EntityType:  models.EntityTypeEvent,
		Name:        activity.Title,
		Description: activity.Description,
		Status:      models.ActivityStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// convertFamilyActivityToActivity converts a complex FamilyActivity to simple Activity format
func (s *DynamoDBService) convertFamilyActivityToActivity(fa *models.FamilyActivity) *models.Activity {
	// TODO: Implement proper conversion when needed
	// For now, return a minimal Activity to satisfy the interface
	return &models.Activity{
		ID:          fa.EntityID,
		Title:       fa.Name,
		Description: fa.Description,
		Type:        string(fa.EntityType),
	}
}

// GetRecentTasksForSource retrieves recent scraping tasks for a specific source
func (s *DynamoDBService) GetRecentTasksForSource(ctx context.Context, sourceID string, limit int) ([]models.ScrapingTask, error) {
	// Query scraping operations table for tasks from this source
	// We'll use a scan with filter since we need to search by source_id
	result, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(s.scrapingOperationsTable),
		FilterExpression: aws.String("source_id = :source_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":source_id": &types.AttributeValueMemberS{Value: sourceID},
		},
		Limit: aws.Int32(int32(limit)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan tasks for source %s: %w", sourceID, err)
	}

	var tasks []models.ScrapingTask
	err = attributevalue.UnmarshalListOfMaps(result.Items, &tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal scraping tasks: %w", err)
	}

	// Sort by UpdatedAt descending (most recent first)
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})

	// Return up to the requested limit
	if len(tasks) > limit {
		tasks = tasks[:limit]
	}

	return tasks, nil
}

// UpdateScrapingTask updates an existing scraping task
func (s *DynamoDBService) UpdateScrapingTask(ctx context.Context, task *models.ScrapingTask) error {
	// Update timestamp
	task.UpdatedAt = time.Now()

	// Create update expression for the fields that might change
	updateExpr := "SET #status = :status, #updated_at = :updated_at"
	exprAttrNames := map[string]string{
		"#status":     "status",
		"#updated_at": "updated_at",
	}
	exprAttrValues := map[string]types.AttributeValue{
		":status":     &types.AttributeValueMemberS{Value: string(task.Status)},
		":updated_at": &types.AttributeValueMemberS{Value: task.UpdatedAt.Format(time.RFC3339)},
	}

	// Add retry count if it has changed
	if task.RetryCount > 0 {
		updateExpr += ", #retry_count = :retry_count"
		exprAttrNames["#retry_count"] = "retry_count"
		exprAttrValues[":retry_count"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", task.RetryCount)}
	}

	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.scrapingOperationsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: task.PK},
			"SK": &types.AttributeValueMemberS{Value: task.SK},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
	})

	if err != nil {
		return fmt.Errorf("failed to update scraping task %s: %w", task.TaskID, err)
	}

	return nil
}

// Admin Events Table Operations

// CreateAdminEvent stores an admin event in DynamoDB
func (s *DynamoDBService) CreateAdminEvent(ctx context.Context, event *models.AdminEvent) error {
	// Set timestamps
	now := time.Now()
	event.CreatedAt = now
	event.UpdatedAt = now
	event.ExtractedAt = now

	// Generate keys
	event.PK = models.CreateAdminEventPK(event.EventID)
	event.SK = models.CreateAdminEventSK(event.ExtractedAt)
	event.StatusKey = models.GenerateAdminEventStatusKey(event.Status)

	// Marshal to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("failed to marshal admin event: %w", err)
	}

	// Put item
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.adminEventsTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create admin event: %w", err)
	}

	return nil
}

// GetAdminEvent retrieves an admin event by primary key
func (s *DynamoDBService) GetAdminEvent(ctx context.Context, eventID string, extractedAt time.Time) (*models.AdminEvent, error) {
	pk := models.CreateAdminEventPK(eventID)
	sk := models.CreateAdminEventSK(extractedAt)

	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.adminEventsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get admin event: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("admin event not found")
	}

	var event models.AdminEvent
	err = attributevalue.UnmarshalMap(result.Item, &event)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal admin event: %w", err)
	}

	return &event, nil
}

// GetAdminEventByID retrieves an admin event by event ID (scans for latest submission)
func (s *DynamoDBService) GetAdminEventByID(ctx context.Context, eventID string) (*models.AdminEvent, error) {
	pk := models.CreateAdminEventPK(eventID)

	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.adminEventsTable),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: pk},
		},
		ScanIndexForward: aws.Bool(false), // Get latest first
		Limit:           aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query admin event: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("admin event not found")
	}

	var event models.AdminEvent
	err = attributevalue.UnmarshalMap(result.Items[0], &event)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal admin event: %w", err)
	}

	return &event, nil
}

// GetAdminEventByURL finds an admin event by source URL
func (s *DynamoDBService) GetAdminEventByURL(ctx context.Context, sourceURL string) (*models.AdminEvent, error) {
	// We need to scan the table to find events by URL since URL is not a key
	// For better performance, we could add a GSI on source_url in the future
	result, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(s.adminEventsTable),
		FilterExpression: aws.String("source_url = :url"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":url": &types.AttributeValueMemberS{Value: sourceURL},
		},
		Limit: aws.Int32(1), // We only need to find one match
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan for admin event by URL: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("admin event not found for URL: %s", sourceURL)
	}

	var event models.AdminEvent
	err = attributevalue.UnmarshalMap(result.Items[0], &event)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal admin event: %w", err)
	}

	return &event, nil
}

// GetSourceByURL finds a source submission by base URL
func (s *DynamoDBService) GetSourceByURL(ctx context.Context, baseURL string) (*models.SourceSubmission, error) {
	// Scan the source management table to find sources by URL
	// For better performance, we could add a GSI on base_url in the future
	result, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(s.sourceManagementTable),
		FilterExpression: aws.String("base_url = :url"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":url": &types.AttributeValueMemberS{Value: baseURL},
		},
		Limit: aws.Int32(1), // We only need to find one match
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan for source by URL: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("source not found for URL: %s", baseURL)
	}

	var source models.SourceSubmission
	err = attributevalue.UnmarshalMap(result.Items[0], &source)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal source submission: %w", err)
	}

	return &source, nil
}

// GetApprovedAdminEvents retrieves all approved admin events
func (s *DynamoDBService) GetApprovedAdminEvents(ctx context.Context, limit int32) ([]models.AdminEvent, error) {
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.adminEventsTable),
		IndexName:              aws.String("StatusIndex"), // Assumes GSI on status exists
		KeyConditionExpression: aws.String("status_key = :status"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: models.GenerateAdminEventStatusKey(models.AdminEventStatusApproved)},
		},
		ScanIndexForward: aws.Bool(false), // Get newest first
		Limit:           aws.Int32(limit),
	})
	if err != nil {
		// If GSI doesn't exist, fall back to scan
		log.Printf("Query failed, falling back to scan: %v", err)
		return s.scanForApprovedEvents(ctx, limit)
	}

	var events []models.AdminEvent
	for _, item := range result.Items {
		var event models.AdminEvent
		err = attributevalue.UnmarshalMap(item, &event)
		if err != nil {
			log.Printf("Failed to unmarshal admin event: %v", err)
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// scanForApprovedEvents is a fallback method when GSI is not available
func (s *DynamoDBService) scanForApprovedEvents(ctx context.Context, limit int32) ([]models.AdminEvent, error) {
	result, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(s.adminEventsTable),
		FilterExpression: aws.String("#status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(models.AdminEventStatusApproved)},
		},
		Limit: aws.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan for approved events: %w", err)
	}

	var events []models.AdminEvent
	for _, item := range result.Items {
		var event models.AdminEvent
		err = attributevalue.UnmarshalMap(item, &event)
		if err != nil {
			log.Printf("Failed to unmarshal admin event: %v", err)
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// UpdateAdminEvent updates an existing admin event
func (s *DynamoDBService) UpdateAdminEvent(ctx context.Context, event *models.AdminEvent) error {
	// Update timestamp
	event.UpdatedAt = time.Now()
	event.StatusKey = models.GenerateAdminEventStatusKey(event.Status)

	// Marshal to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("failed to marshal admin event: %w", err)
	}

	// Put item (upsert)
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.adminEventsTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to update admin event: %w", err)
	}

	return nil
}

// QueryAdminEventsByStatus queries admin events by status using GSI
func (s *DynamoDBService) QueryAdminEventsByStatus(ctx context.Context, status models.AdminEventStatus, limit int32) ([]models.AdminEvent, error) {
	statusKey := models.GenerateAdminEventStatusKey(status)

	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.adminEventsTable),
		IndexName:              aws.String("status-date-index"),
		KeyConditionExpression: aws.String("StatusKey = :statusKey"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":statusKey": &types.AttributeValueMemberS{Value: statusKey},
		},
		ScanIndexForward: aws.Bool(false), // Get newest first
		Limit:           aws.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query admin events by status: %w", err)
	}

	var events []models.AdminEvent
	err = attributevalue.UnmarshalListOfMaps(result.Items, &events)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal admin events: %w", err)
	}

	return events, nil
}

// GetAllPendingAdminEvents retrieves all admin events that need review
func (s *DynamoDBService) GetAllPendingAdminEvents(ctx context.Context, limit int32) ([]models.AdminEvent, error) {
	// Get both pending and edited events
	pendingEvents, err := s.QueryAdminEventsByStatus(ctx, models.AdminEventStatusPending, limit/2)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending events: %w", err)
	}

	editedEvents, err := s.QueryAdminEventsByStatus(ctx, models.AdminEventStatusEdited, limit/2)
	if err != nil {
		return nil, fmt.Errorf("failed to get edited events: %w", err)
	}

	// Combine and sort by extracted_at descending
	allEvents := append(pendingEvents, editedEvents...)
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].ExtractedAt.After(allEvents[j].ExtractedAt)
	})

	// Limit results
	if len(allEvents) > int(limit) {
		allEvents = allEvents[:limit]
	}

	return allEvents, nil
}

// DeleteAdminEvent removes an admin event
func (s *DynamoDBService) DeleteAdminEvent(ctx context.Context, eventID string, extractedAt time.Time) error {
	pk := models.CreateAdminEventPK(eventID)
	sk := models.CreateAdminEventSK(extractedAt)

	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.adminEventsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete admin event: %w", err)
	}

	return nil
}

// BatchCreateAdminEvents creates multiple admin events from a crawl submission
func (s *DynamoDBService) BatchCreateAdminEvents(ctx context.Context, events []*models.AdminEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Process in batches of 25 (DynamoDB limit)
	batchSize := 25
	for i := 0; i < len(events); i += batchSize {
		end := i + batchSize
		if end > len(events) {
			end = len(events)
		}

		batch := events[i:end]
		if err := s.batchWriteAdminEvents(ctx, batch); err != nil {
			return fmt.Errorf("failed to write batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// batchWriteAdminEvents writes a batch of admin events
func (s *DynamoDBService) batchWriteAdminEvents(ctx context.Context, events []*models.AdminEvent) error {
	writeRequests := make([]types.WriteRequest, 0, len(events))

	for _, event := range events {
		// Set timestamps and keys
		now := time.Now()
		event.CreatedAt = now
		event.UpdatedAt = now
		event.ExtractedAt = now
		event.PK = models.CreateAdminEventPK(event.EventID)
		event.SK = models.CreateAdminEventSK(event.ExtractedAt)
		event.StatusKey = models.GenerateAdminEventStatusKey(event.Status)

		// Marshal event
		item, err := attributevalue.MarshalMap(event)
		if err != nil {
			return fmt.Errorf("failed to marshal admin event %s: %w", event.EventID, err)
		}

		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: item},
		})
	}

	// Execute batch write
	_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			s.adminEventsTable: writeRequests,
		},
	})

	return err
}

// Source Deletion Operations

// DeleteSourceCompletely removes a source and all associated data using transactions
func (s *DynamoDBService) DeleteSourceCompletely(ctx context.Context, sourceID string) (*models.DeletionResult, error) {
	result := &models.DeletionResult{
		SourceID: sourceID,
	}

	// First, query all associated activity records to count them
	activities, err := s.queryActivitiesBySource(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query activities for source %s: %w", sourceID, err)
	}
	result.ActivitiesDeleted = len(activities)

	// Get source record keys to check what exists
	sourceKeys := models.GetSourceRecordKeys(sourceID)
	var transactItems []types.TransactWriteItem

	// Check and prepare deletion for each source record type
	for _, key := range sourceKeys {
		exists, err := s.checkRecordExists(ctx, s.sourceManagementTable, key.PK, key.SK)
		if err != nil {
			return nil, fmt.Errorf("failed to check record existence %s/%s: %w", key.PK, key.SK, err)
		}

		if exists {
			// Add to transaction
			transactItems = append(transactItems, types.TransactWriteItem{
				Delete: &types.Delete{
					TableName: aws.String(s.sourceManagementTable),
					Key: map[string]types.AttributeValue{
						"PK": &types.AttributeValueMemberS{Value: key.PK},
						"SK": &types.AttributeValueMemberS{Value: key.SK},
					},
				},
			})

			// Track what we're deleting
			switch key.SK {
			case models.CreateSourceSubmissionSK():
				result.SubmissionDeleted = true
			case models.CreateSourceAnalysisSK():
				result.AnalysisDeleted = true
			case models.CreateSourceConfigSK():
				result.ConfigDeleted = true
			}
		}
	}

	// Add activity deletions to transaction (in batches if needed)
	for _, activity := range activities {
		transactItems = append(transactItems, types.TransactWriteItem{
			Delete: &types.Delete{
				TableName: aws.String(s.familyActivitiesTable),
				Key: map[string]types.AttributeValue{
					"PK": &types.AttributeValueMemberS{Value: activity.PK},
					"SK": &types.AttributeValueMemberS{Value: activity.SK},
				},
			},
		})
	}

	// Execute transaction in batches (DynamoDB limit is 100 items per transaction)
	if err := s.executeTransactionBatches(ctx, transactItems); err != nil {
		return nil, fmt.Errorf("failed to execute deletion transaction: %w", err)
	}

	// Calculate total records deleted
	result.TotalRecords = 0
	if result.SubmissionDeleted {
		result.TotalRecords++
	}
	if result.AnalysisDeleted {
		result.TotalRecords++
	}
	if result.ConfigDeleted {
		result.TotalRecords++
	}
	result.TotalRecords += result.ActivitiesDeleted

	return result, nil
}

// queryActivitiesBySource finds all activities associated with a source
func (s *DynamoDBService) queryActivitiesBySource(ctx context.Context, sourceID string) ([]models.FamilyActivity, error) {
	// Query activities table for records with source_id
	result, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(s.familyActivitiesTable),
		FilterExpression: aws.String("source_id = :source_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":source_id": &types.AttributeValueMemberS{Value: sourceID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan activities by source: %w", err)
	}

	var activities []models.FamilyActivity
	err = attributevalue.UnmarshalListOfMaps(result.Items, &activities)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal activities: %w", err)
	}

	return activities, nil
}

// checkRecordExists checks if a record exists in DynamoDB
func (s *DynamoDBService) checkRecordExists(ctx context.Context, tableName, pk, sk string) (bool, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return false, err
	}

	return result.Item != nil, nil
}

// executeTransactionBatches executes transaction items in batches of 100
func (s *DynamoDBService) executeTransactionBatches(ctx context.Context, items []types.TransactWriteItem) error {
	if len(items) == 0 {
		return nil
	}

	batchSize := 100 // DynamoDB transaction limit
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		_, err := s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
			TransactItems: batch,
		})
		if err != nil {
			return fmt.Errorf("failed to execute transaction batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// CreateSourceDeletionEvent logs a source deletion event
func (s *DynamoDBService) CreateSourceDeletionEvent(ctx context.Context, event *models.SourceDeletionEvent) error {
	// Set timestamps and keys
	now := time.Now()
	event.Timestamp = now
	event.PK = models.CreateSourceDeletionEventPK(event.EventID)
	event.SK = models.CreateSourceDeletionEventSK(event.Timestamp)
	event.EventTypeKey = models.GenerateEventTypeKey(event.EventType)

	// Marshal to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("failed to marshal source deletion event: %w", err)
	}

	// Put item
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.adminEventsTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create source deletion event: %w", err)
	}

	return nil
}