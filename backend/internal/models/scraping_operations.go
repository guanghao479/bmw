package models

import (
	"fmt"
	"time"
)

// Scraping task type constants
const (
	TaskTypeFullScrape    = "full_scrape"
	TaskTypeIncremental   = "incremental"
	TaskTypeValidation    = "validation"
	TaskTypeDiscovery     = "discovery"
)

// Scraping task status constants
const (
	TaskStatusScheduled   = "scheduled"
	TaskStatusInProgress  = "in_progress"
	TaskStatusCompleted   = "completed"
	TaskStatusFailed      = "failed"
	TaskStatusCancelled   = "cancelled"
	TaskStatusRetrying    = "retrying"
)

// Scraping task priority constants
const (
	TaskPriorityHigh   = "high"
	TaskPriorityMedium = "medium"
	TaskPriorityLow    = "low"
)

// ScrapingTask represents a scheduled scraping task
type ScrapingTask struct {
	// Primary Keys
	PK string `json:"PK" dynamodbav:"PK"` // SCHEDULE#{date} or TASK#{task_id}
	SK string `json:"SK" dynamodbav:"SK"` // TASK#{priority}#{source_id}#{task_id}

	// Task identification
	TaskID   string `json:"task_id" dynamodbav:"task_id"`
	SourceID string `json:"source_id" dynamodbav:"source_id"`

	// Task configuration
	TaskType     string    `json:"task_type" dynamodbav:"task_type"`           // full_scrape, incremental, validation, discovery
	Priority     string    `json:"priority" dynamodbav:"priority"`             // high, medium, low
	ScheduledTime time.Time `json:"scheduled_time" dynamodbav:"scheduled_time"`
	TargetURLs   []string  `json:"target_urls" dynamodbav:"target_urls"`
	
	// Execution configuration
	ExtractionRules DataSelectors `json:"extraction_rules" dynamodbav:"extraction_rules"`
	RateLimits      RateLimit     `json:"rate_limits" dynamodbav:"rate_limits"`
	Timeout         int           `json:"timeout" dynamodbav:"timeout"` // seconds
	MaxRetries      int           `json:"max_retries" dynamodbav:"max_retries"`
	
	// Task status
	Status           string    `json:"status" dynamodbav:"status"`                       // scheduled, in_progress, completed, failed
	RetryCount       int       `json:"retry_count" dynamodbav:"retry_count"`
	LastRetryAt      time.Time `json:"last_retry_at" dynamodbav:"last_retry_at"`
	EstimatedDuration int64    `json:"estimated_duration" dynamodbav:"estimated_duration"` // seconds
	
	// Dependencies and prerequisites
	Dependencies []string `json:"dependencies" dynamodbav:"dependencies"` // other task IDs that must complete first
	
	// Timestamps
	CreatedAt     time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updated_at"`
	TTL           int64     `json:"TTL" dynamodbav:"TTL"` // auto-expire timestamp
	
	// GSI Keys
	NextRunKey        string `json:"NextRunKey,omitempty" dynamodbav:"NextRunKey,omitempty"`               // NEXT_RUN#{timestamp}
	PrioritySourceKey string `json:"PrioritySourceKey,omitempty" dynamodbav:"PrioritySourceKey,omitempty"` // PRIORITY#{priority}#{source_id}
}

// ScrapingExecution represents an individual execution of a scraping task
type ScrapingExecution struct {
	// Primary Keys
	PK string `json:"PK" dynamodbav:"PK"` // EXECUTION#{execution_id}
	SK string `json:"SK" dynamodbav:"SK"` // STATUS, RESULT, or METRICS

	// Execution identification
	ExecutionID string `json:"execution_id" dynamodbav:"execution_id"`
	TaskID      string `json:"task_id" dynamodbav:"task_id"`
	SourceID    string `json:"source_id" dynamodbav:"source_id"`
	
	// Execution details
	StartedAt    time.Time `json:"started_at" dynamodbav:"started_at"`
	CompletedAt  time.Time `json:"completed_at" dynamodbav:"completed_at"`
	Duration     int64     `json:"duration" dynamodbav:"duration"` // milliseconds
	Status       string    `json:"status" dynamodbav:"status"`     // running, completed, failed
	
	// Results summary
	ItemsExtracted  int      `json:"items_extracted" dynamodbav:"items_extracted"`
	ItemsProcessed  int      `json:"items_processed" dynamodbav:"items_processed"`
	ItemsStored     int      `json:"items_stored" dynamodbav:"items_stored"`
	ErrorCount      int      `json:"error_count" dynamodbav:"error_count"`
	WarningCount    int      `json:"warning_count" dynamodbav:"warning_count"`
	
	// Performance metrics
	Metrics ExecutionMetrics `json:"metrics" dynamodbav:"metrics"`
	
	// Error information
	Errors   []ExecutionError `json:"errors" dynamodbav:"errors"`
	Warnings []ExecutionError `json:"warnings" dynamodbav:"warnings"`
	
	// TTL for auto-expiration
	TTL int64 `json:"TTL" dynamodbav:"TTL"`
}

// ExecutionMetrics contains detailed performance metrics
type ExecutionMetrics struct {
	// Network metrics
	RequestCount        int     `json:"request_count" dynamodbav:"request_count"`
	SuccessfulRequests  int     `json:"successful_requests" dynamodbav:"successful_requests"`
	FailedRequests      int     `json:"failed_requests" dynamodbav:"failed_requests"`
	AverageResponseTime int64   `json:"average_response_time" dynamodbav:"average_response_time"` // milliseconds
	TotalBytes          int64   `json:"total_bytes" dynamodbav:"total_bytes"`
	
	// Processing metrics
	ParsingTime         int64   `json:"parsing_time" dynamodbav:"parsing_time"`         // milliseconds
	ExtractionTime      int64   `json:"extraction_time" dynamodbav:"extraction_time"`   // milliseconds
	ValidationTime      int64   `json:"validation_time" dynamodbav:"validation_time"`   // milliseconds
	StorageTime         int64   `json:"storage_time" dynamodbav:"storage_time"`         // milliseconds
	
	// Quality metrics
	ExtractionSuccess   float64 `json:"extraction_success" dynamodbav:"extraction_success"`     // percentage
	DataCompleteness    float64 `json:"data_completeness" dynamodbav:"data_completeness"`       // percentage
	DuplicateRate       float64 `json:"duplicate_rate" dynamodbav:"duplicate_rate"`             // percentage
	
	// Resource usage
	MemoryUsageMB       float64 `json:"memory_usage_mb" dynamodbav:"memory_usage_mb"`
	CPUUsagePercent     float64 `json:"cpu_usage_percent" dynamodbav:"cpu_usage_percent"`
}

// ExecutionError represents an error or warning during execution
type ExecutionError struct {
	Type        string    `json:"type" dynamodbav:"type"`               // error, warning
	Code        string    `json:"code" dynamodbav:"code"`               // error code
	Message     string    `json:"message" dynamodbav:"message"`         // error message
	URL         string    `json:"url" dynamodbav:"url"`                 // URL where error occurred
	Timestamp   time.Time `json:"timestamp" dynamodbav:"timestamp"`
	Recoverable bool      `json:"recoverable" dynamodbav:"recoverable"` // whether error is recoverable
	Context     map[string]interface{} `json:"context" dynamodbav:"context"` // additional context
}

// SourceMetrics represents aggregated metrics for a source over time
type SourceMetrics struct {
	// Primary Keys
	PK string `json:"PK" dynamodbav:"PK"` // SOURCE#{source_id}
	SK string `json:"SK" dynamodbav:"SK"` // METRICS#{date} or RUN#{timestamp}

	// Source identification
	SourceID    string `json:"source_id" dynamodbav:"source_id"`
	MetricsDate string `json:"metrics_date" dynamodbav:"metrics_date"` // YYYY-MM-DD for daily aggregation
	
	// Aggregated execution metrics
	TotalRuns         int     `json:"total_runs" dynamodbav:"total_runs"`
	SuccessfulRuns    int     `json:"successful_runs" dynamodbav:"successful_runs"`
	FailedRuns        int     `json:"failed_runs" dynamodbav:"failed_runs"`
	AverageDuration   int64   `json:"average_duration" dynamodbav:"average_duration"`   // milliseconds
	TotalItemsFound   int     `json:"total_items_found" dynamodbav:"total_items_found"`
	AverageItemsFound float64 `json:"average_items_found" dynamodbav:"average_items_found"`
	
	// Quality metrics
	SuccessRate         float64 `json:"success_rate" dynamodbav:"success_rate"`                 // percentage
	DataQualityScore    float64 `json:"data_quality_score" dynamodbav:"data_quality_score"`     // 0.0 - 1.0
	ContentStabilityScore float64 `json:"content_stability_score" dynamodbav:"content_stability_score"` // 0.0 - 1.0
	
	// Performance trends
	ResponseTimeTrend   string  `json:"response_time_trend" dynamodbav:"response_time_trend"`     // improving, stable, degrading
	VolumeChangeTrend   string  `json:"volume_change_trend" dynamodbav:"volume_change_trend"`     // increasing, stable, decreasing
	QualityTrend        string  `json:"quality_trend" dynamodbav:"quality_trend"`                 // improving, stable, degrading
	
	// Timestamp and TTL
	UpdatedAt time.Time `json:"updated_at" dynamodbav:"updated_at"`
	TTL       int64     `json:"TTL" dynamodbav:"TTL"`
}

// DynamoScrapingRun represents the results of an individual scraping run in DynamoDB
type DynamoScrapingRun struct {
	// Primary Keys  
	PK string `json:"PK" dynamodbav:"PK"` // SOURCE#{source_id}
	SK string `json:"SK" dynamodbav:"SK"` // RUN#{timestamp}

	// Run identification
	RunID       string    `json:"run_id" dynamodbav:"run_id"`
	SourceID    string    `json:"source_id" dynamodbav:"source_id"`
	ExecutionID string    `json:"execution_id" dynamodbav:"execution_id"`
	Timestamp   time.Time `json:"timestamp" dynamodbav:"timestamp"`
	
	// Run configuration
	TaskType    string   `json:"task_type" dynamodbav:"task_type"`
	TargetURLs  []string `json:"target_urls" dynamodbav:"target_urls"`
	UserAgent   string   `json:"user_agent" dynamodbav:"user_agent"`
	
	// Results
	Status          string                 `json:"status" dynamodbav:"status"`           // success, partial, failed
	ItemsFound      int                    `json:"items_found" dynamodbav:"items_found"`
	ItemsProcessed  int                    `json:"items_processed" dynamodbav:"items_processed"`
	ItemsStored     int                    `json:"items_stored" dynamodbav:"items_stored"`
	NewItems        int                    `json:"new_items" dynamodbav:"new_items"`
	UpdatedItems    int                    `json:"updated_items" dynamodbav:"updated_items"`
	DuplicateItems  int                    `json:"duplicate_items" dynamodbav:"duplicate_items"`
	ErrorItems      int                    `json:"error_items" dynamodbav:"error_items"`
	
	// Content analysis
	ContentHash     string  `json:"content_hash" dynamodbav:"content_hash"`         // hash of scraped content
	ContentSize     int64   `json:"content_size" dynamodbav:"content_size"`         // bytes
	ContentChanged  bool    `json:"content_changed" dynamodbav:"content_changed"`   // vs previous run
	ChangePercent   float64 `json:"change_percent" dynamodbav:"change_percent"`     // percentage of content changed
	
	// Performance
	Duration        int64                  `json:"duration" dynamodbav:"duration"`         // milliseconds
	RequestCount    int                    `json:"request_count" dynamodbav:"request_count"`
	BytesDownloaded int64                  `json:"bytes_downloaded" dynamodbav:"bytes_downloaded"`
	ErrorMessages   []string               `json:"error_messages" dynamodbav:"error_messages"`
	
	// Quality assessment
	QualityScore    float64                `json:"quality_score" dynamodbav:"quality_score"`     // 0.0 - 1.0
	QualityDetails  map[string]float64     `json:"quality_details" dynamodbav:"quality_details"` // detailed quality breakdown
	
	// Sample data for validation
	SampleItems     []interface{}          `json:"sample_items" dynamodbav:"sample_items"`       // sample extracted items
	
	// TTL for auto-expiration
	TTL int64 `json:"TTL" dynamodbav:"TTL"`
}

// ScheduledTaskQueue represents the queue of upcoming scraping tasks
type ScheduledTaskQueue struct {
	// Primary Keys
	PK string `json:"PK" dynamodbav:"PK"` // QUEUE#{date}
	SK string `json:"SK" dynamodbav:"SK"` // PRIORITY#{priority}#{scheduled_time}

	// Queue metadata
	QueueDate      string    `json:"queue_date" dynamodbav:"queue_date"`           // YYYY-MM-DD
	ScheduledTime  time.Time `json:"scheduled_time" dynamodbav:"scheduled_time"`
	Priority       string    `json:"priority" dynamodbav:"priority"`
	
	// Tasks in this queue slot
	TaskCount      int                    `json:"task_count" dynamodbav:"task_count"`
	Tasks          []QueuedTask           `json:"tasks" dynamodbav:"tasks"`
	
	// Resource estimates
	EstimatedDuration    int64   `json:"estimated_duration" dynamodbav:"estimated_duration"`       // total estimated time
	EstimatedRequests    int     `json:"estimated_requests" dynamodbav:"estimated_requests"`
	EstimatedDataVolume  int64   `json:"estimated_data_volume" dynamodbav:"estimated_data_volume"` // bytes
	
	// Queue status
	Status         string    `json:"status" dynamodbav:"status"`                   // pending, processing, completed
	ProcessedAt    time.Time `json:"processed_at" dynamodbav:"processed_at"`
	
	// TTL
	TTL int64 `json:"TTL" dynamodbav:"TTL"`
}

// QueuedTask represents a task in the scheduled queue
type QueuedTask struct {
	TaskID              string   `json:"task_id" dynamodbav:"task_id"`
	SourceID            string   `json:"source_id" dynamodbav:"source_id"`
	TaskType            string   `json:"task_type" dynamodbav:"task_type"`
	Priority            string   `json:"priority" dynamodbav:"priority"`
	EstimatedDuration   int64    `json:"estimated_duration" dynamodbav:"estimated_duration"`
	Dependencies        []string `json:"dependencies" dynamodbav:"dependencies"`
	ReadyToExecute      bool     `json:"ready_to_execute" dynamodbav:"ready_to_execute"`
}

// Helper functions to create primary keys for scraping operations
func CreateSchedulePK(date string) string {
	return "SCHEDULE#" + date
}

func CreateTaskPK(taskID string) string {
	return "TASK#" + taskID
}

func CreateExecutionPK(executionID string) string {
	return "EXECUTION#" + executionID
}

func CreateQueuePK(date string) string {
	return "QUEUE#" + date
}

func CreateTaskSK(priority, sourceID, taskID string) string {
	return "TASK#" + priority + "#" + sourceID + "#" + taskID
}

func CreateRunSK(timestamp time.Time) string {
	return "RUN#" + timestamp.Format("2006-01-02T15:04:05Z")
}

func CreateMetricsSK(date string) string {
	return "METRICS#" + date
}

// Helper functions to generate GSI keys for scraping operations
func GenerateNextRunKey(scheduledTime time.Time) string {
	return "NEXT_RUN#" + scheduledTime.Format("2006-01-02T15:04:05Z")
}

func GenerateTaskPrioritySourceKey(priority, sourceID string) string {
	return "PRIORITY#" + priority + "#" + sourceID
}

// Helper functions to calculate TTL (Time To Live) for auto-expiration
func CalculateTaskTTL(createdAt time.Time, retentionDays int) int64 {
	return createdAt.AddDate(0, 0, retentionDays).Unix()
}

func CalculateExecutionTTL(completedAt time.Time, retentionDays int) int64 {
	return completedAt.AddDate(0, 0, retentionDays).Unix()
}

func CalculateMetricsTTL(metricsDate time.Time, retentionDays int) int64 {
	return metricsDate.AddDate(0, 0, retentionDays).Unix()
}

// Validation functions
func (st *ScrapingTask) Validate() error {
	if st.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}
	if st.SourceID == "" {
		return fmt.Errorf("source_id is required")
	}
	if st.TaskType == "" {
		return fmt.Errorf("task_type is required")
	}
	if len(st.TargetURLs) == 0 {
		return fmt.Errorf("target_urls is required")
	}
	return nil
}

func (se *ScrapingExecution) Validate() error {
	if se.ExecutionID == "" {
		return fmt.Errorf("execution_id is required")
	}
	if se.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}
	if se.SourceID == "" {
		return fmt.Errorf("source_id is required")
	}
	return nil
}

// Status transition validation
func (st *ScrapingTask) CanTransitionTo(newStatus string) bool {
	switch st.Status {
	case TaskStatusScheduled:
		return newStatus == TaskStatusInProgress || newStatus == TaskStatusCancelled
	case TaskStatusInProgress:
		return newStatus == TaskStatusCompleted || newStatus == TaskStatusFailed || newStatus == TaskStatusRetrying
	case TaskStatusFailed:
		return newStatus == TaskStatusRetrying || newStatus == TaskStatusCancelled
	case TaskStatusRetrying:
		return newStatus == TaskStatusInProgress || newStatus == TaskStatusCancelled
	case TaskStatusCompleted, TaskStatusCancelled:
		return false // terminal states
	default:
		return false
	}
}

// CalculateTTL calculates TTL timestamp for auto-expiring data
func CalculateTTL(duration time.Duration) int64 {
	return time.Now().Add(duration).Unix()
}

// GeneratePrioritySourceKey generates GSI key for priority and source lookup
func GeneratePrioritySourceKey(priority, sourceID, taskID string) string {
	return "PRIORITY#" + priority + "#" + sourceID + "#" + taskID
}