package models

import "time"

// ScrapingJob represents a single scraping operation
type ScrapingJob struct {
	ID          string    `json:"id"`
	SourceURL   string    `json:"sourceUrl"`
	Domain      string    `json:"domain"`
	Status      string    `json:"status"`      // pending|running|completed|failed
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
	Duration    int64     `json:"duration,omitempty"` // duration in milliseconds
	
	// Results
	ActivitiesFound int      `json:"activitiesFound"`
	ActivitiesNew   int      `json:"activitiesNew"`
	ActivitiesError int      `json:"activitiesError"`
	ErrorMessage    string   `json:"errorMessage,omitempty"`
	ErrorDetails    []string `json:"errorDetails,omitempty"`
	
	// Processing details
	ContentLength   int    `json:"contentLength,omitempty"`   // length of scraped content
	ProcessingTime  int64  `json:"processingTime,omitempty"`  // OpenAI processing time in ms
	TokensUsed      int    `json:"tokensUsed,omitempty"`      // OpenAI tokens consumed
	
	// Metadata
	UserAgent       string `json:"userAgent,omitempty"`
	ScrapingVersion string `json:"scrapingVersion"`
}

// ScrapingRun represents a complete scraping operation across all sources
type ScrapingRun struct {
	ID              string        `json:"id"`
	StartedAt       time.Time     `json:"startedAt"`
	CompletedAt     time.Time     `json:"completedAt,omitempty"`
	Duration        int64         `json:"duration,omitempty"` // total duration in milliseconds
	Status          string        `json:"status"`             // running|completed|failed|partial
	
	// Aggregated results
	TotalSources       int `json:"totalSources"`
	SuccessfulSources  int `json:"successfulSources"`
	FailedSources      int `json:"failedSources"`
	TotalActivities    int `json:"totalActivities"`
	NewActivities      int `json:"newActivities"`
	UpdatedActivities  int `json:"updatedActivities"`
	DuplicatesRemoved  int `json:"duplicatesRemoved"`
	
	// Individual jobs
	Jobs []ScrapingJob `json:"jobs"`
	
	// Error summary
	ErrorSummary string   `json:"errorSummary,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
	
	// Cost tracking
	TotalTokensUsed int     `json:"totalTokensUsed"`
	EstimatedCost   float64 `json:"estimatedCost"` // estimated cost in USD
	
	// Metadata
	TriggerType     string `json:"triggerType"`     // scheduled|manual|webhook
	ScrapingVersion string `json:"scrapingVersion"`
	LambdaRequestId string `json:"lambdaRequestId,omitempty"`
}

// ScrapingStatus represents the current status for admin monitoring
type ScrapingStatus struct {
	CurrentRun      *ScrapingRun `json:"currentRun,omitempty"`
	LastRun         *ScrapingRun `json:"lastRun,omitempty"`
	LastSuccessRun  *ScrapingRun `json:"lastSuccessRun,omitempty"`
	NextScheduledRun time.Time    `json:"nextScheduledRun"`
	
	// Summary statistics
	TotalRuns           int     `json:"totalRuns"`
	SuccessfulRuns      int     `json:"successfulRuns"`
	FailedRuns          int     `json:"failedRuns"`
	AverageRunDuration  int64   `json:"averageRunDuration"`  // in milliseconds
	TotalActivitiesEver int     `json:"totalActivitiesEver"`
	MonthlyTokensUsed   int     `json:"monthlyTokensUsed"`
	MonthlyEstimatedCost float64 `json:"monthlyEstimatedCost"`
	
	// Health indicators
	SystemHealth    string    `json:"systemHealth"`    // healthy|degraded|failing
	LastHealthCheck time.Time `json:"lastHealthCheck"`
	ActiveSources   []string  `json:"activeSources"`
	FailingSources  []string  `json:"failingSources"`
	
	// Updated timestamp
	UpdatedAt time.Time `json:"updatedAt"`
}

// SourceConfig represents configuration for a scraping source
type SourceConfig struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Domain      string            `json:"domain"`
	Enabled     bool              `json:"enabled"`
	Priority    int               `json:"priority"`    // 1-10, higher = more important
	Frequency   string            `json:"frequency"`   // how often to scrape this source
	Selector    string            `json:"selector,omitempty"` // CSS selector if needed
	Headers     map[string]string `json:"headers,omitempty"`  // custom headers
	Timeout     int               `json:"timeout"`     // timeout in seconds
	RetryCount  int               `json:"retryCount"`
	LastScraped time.Time         `json:"lastScraped"`
	SuccessRate float64           `json:"successRate"` // percentage
	Notes       string            `json:"notes,omitempty"`
}

// ScrapingConfig represents the overall configuration
type ScrapingConfig struct {
	Version         string         `json:"version"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	Sources         []SourceConfig `json:"sources"`
	GlobalTimeout   int            `json:"globalTimeout"`   // overall timeout in seconds
	ConcurrentLimit int            `json:"concurrentLimit"` // max concurrent scraping jobs
	UserAgent       string         `json:"userAgent"`
	
	// OpenAI configuration
	OpenAIModel       string  `json:"openaiModel"`
	OpenAITemperature float64 `json:"openaiTemperature"`
	OpenAIMaxTokens   int     `json:"openaiMaxTokens"`
	
	// Jina configuration
	JinaTimeout int  `json:"jinaTimeout"`
	JinaEnabled bool `json:"jinaEnabled"`
	
	// Quality controls
	MinContentLength  int     `json:"minContentLength"`  // minimum content length to process
	DuplicateThreshold float64 `json:"duplicateThreshold"` // similarity threshold for duplicates
	
	// Rate limiting
	RequestDelay    int `json:"requestDelay"`    // delay between requests in milliseconds
	BurstLimit      int `json:"burstLimit"`      // max requests per burst
	BurstWindow     int `json:"burstWindow"`     // burst window in seconds
}

// Scraping job status constants
const (
	ScrapingStatusPending   = "pending"
	ScrapingStatusRunning   = "running"
	ScrapingStatusCompleted = "completed"
	ScrapingStatusFailed    = "failed"
	ScrapingStatusPartial   = "partial"
)

// System health constants
const (
	HealthStatusHealthy  = "healthy"
	HealthStatusDegraded = "degraded"
	HealthStatusFailing  = "failing"
)

// Trigger type constants
const (
	TriggerTypeScheduled = "scheduled"
	TriggerTypeManual    = "manual"
	TriggerTypeWebhook   = "webhook"
)