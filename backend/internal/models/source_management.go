package models

import (
	"fmt"
	"time"
)

// Source lifecycle stage constants
const (
	SourceStageSubmission = "SUBMISSION"
	SourceStageAnalysis   = "ANALYSIS"
	SourceStageConfig     = "CONFIG"
)

// Source status constants
const (
	SourceStatusPendingAnalysis = "pending_analysis"
	SourceStatusAnalysisComplete = "analysis_complete"
	SourceStatusActive          = "active"
	SourceStatusInactive        = "inactive"
	SourceStatusRejected        = "rejected"
)

// Source priority constants
const (
	SourcePriorityHigh   = "high"
	SourcePriorityMedium = "medium"
	SourcePriorityLow    = "low"
)

// Source type constants
const (
	SourceTypeVenue             = "venue"
	SourceTypeEventOrganizer    = "event-organizer"
	SourceTypeProgramProvider   = "program-provider"
	SourceTypeCommunityCalendar = "community-calendar"
)

// SourceSubmission represents a founder-submitted source for analysis
type SourceSubmission struct {
	// Primary Keys
	PK string `json:"PK" dynamodbav:"PK"` // SOURCE#{source_id}
	SK string `json:"SK" dynamodbav:"SK"` // SUBMISSION

	// Source Basic Information
	SourceID    string   `json:"source_id" dynamodbav:"source_id"`
	SourceName  string   `json:"source_name" dynamodbav:"source_name"`
	BaseURL     string   `json:"base_url" dynamodbav:"base_url"`
	SourceType  string   `json:"source_type" dynamodbav:"source_type"`     // venue, event-organizer, program-provider, community-calendar
	Priority    string   `json:"priority" dynamodbav:"priority"`           // high, medium, low
	ExpectedContent []string `json:"expected_content" dynamodbav:"expected_content"` // events, classes, camps, venues

	// Founder-provided hints
	HintURLs []string `json:"hint_urls" dynamodbav:"hint_urls"` // URLs that might contain activities

	// Submission metadata
	SubmittedBy string    `json:"submitted_by" dynamodbav:"submitted_by"`
	SubmittedAt time.Time `json:"submitted_at" dynamodbav:"submitted_at"`
	Status      string    `json:"status" dynamodbav:"status"` // pending_analysis, analysis_complete, etc.

	// GSI Keys
	StatusKey   string `json:"StatusKey,omitempty" dynamodbav:"StatusKey,omitempty"`     // STATUS#{status}
	PriorityKey string `json:"PriorityKey,omitempty" dynamodbav:"PriorityKey,omitempty"` // PRIORITY#{priority}#{source_id}
}

// SourceAnalysis represents the automated analysis results
type SourceAnalysis struct {
	// Primary Keys
	PK string `json:"PK" dynamodbav:"PK"` // SOURCE#{source_id}
	SK string `json:"SK" dynamodbav:"SK"` // ANALYSIS

	// Analysis metadata
	SourceID            string    `json:"source_id" dynamodbav:"source_id"`
	AnalysisCompletedAt time.Time `json:"analysis_completed_at" dynamodbav:"analysis_completed_at"`
	AnalysisVersion     string    `json:"analysis_version" dynamodbav:"analysis_version"`

	// Discovery results
	DiscoveredPatterns DiscoveryPatterns `json:"discovered_patterns" dynamodbav:"discovered_patterns"`

	// Extraction test results
	ExtractionTestResults ExtractionTestResults `json:"extraction_test_results" dynamodbav:"extraction_test_results"`

	// Generated recommendations
	RecommendedConfig RecommendedSourceConfig `json:"recommended_config" dynamodbav:"recommended_config"`

	// Quality assessment
	OverallQualityScore float64 `json:"overall_quality_score" dynamodbav:"overall_quality_score"`
	Issues             []string `json:"issues" dynamodbav:"issues"`
	Recommendations    []string `json:"recommendations" dynamodbav:"recommendations"`

	// Analysis status
	Status string `json:"status" dynamodbav:"status"` // analysis_complete, failed, etc.
}

// DiscoveryPatterns contains the results of automated content discovery
type DiscoveryPatterns struct {
	// Sitemap and robots.txt analysis
	SitemapFound bool     `json:"sitemap_found" dynamodbav:"sitemap_found"`
	SitemapURL   string   `json:"sitemap_url" dynamodbav:"sitemap_url"`
	RSSFeeds     []string `json:"rss_feeds" dynamodbav:"rss_feeds"`

	// Content page discovery
	ContentPages []ContentPage `json:"content_pages" dynamodbav:"content_pages"`

	// Generated CSS selectors
	DataSelectors DataSelectors `json:"data_selectors" dynamodbav:"data_selectors"`

	// Schema.org structured data
	StructuredDataFound bool                   `json:"structured_data_found" dynamodbav:"structured_data_found"`
	SchemaTypes        []string               `json:"schema_types" dynamodbav:"schema_types"`
	StructuredData     map[string]interface{} `json:"structured_data" dynamodbav:"structured_data"`
}

// ContentPage represents a discovered page with activity content
type ContentPage struct {
	URL        string  `json:"url" dynamodbav:"url"`
	Type       string  `json:"type" dynamodbav:"type"`             // events, programs, venues, classes
	Confidence float64 `json:"confidence" dynamodbav:"confidence"` // 0.0 - 1.0
	Title      string  `json:"title" dynamodbav:"title"`
	Language   string  `json:"language" dynamodbav:"language"`
}

// DataSelectors contains CSS selectors for extracting activity data
type DataSelectors struct {
	Title       string `json:"title" dynamodbav:"title"`
	Date        string `json:"date" dynamodbav:"date"`
	Time        string `json:"time" dynamodbav:"time"`
	Description string `json:"description" dynamodbav:"description"`
	Location    string `json:"location" dynamodbav:"location"`
	Venue       string `json:"venue" dynamodbav:"venue"`
	Price       string `json:"price" dynamodbav:"price"`
	AgeRange    string `json:"age_range" dynamodbav:"age_range"`
	Category    string `json:"category" dynamodbav:"category"`
	RegistrationURL string `json:"registration_url" dynamodbav:"registration_url"`
	ContactInfo     string `json:"contact_info" dynamodbav:"contact_info"`
	Images          string `json:"images" dynamodbav:"images"`
}

// ExtractionTestResults contains results from testing data extraction
type ExtractionTestResults struct {
	TestURL      string                 `json:"test_url" dynamodbav:"test_url"`
	ItemsFound   int                    `json:"items_found" dynamodbav:"items_found"`
	QualityScore float64                `json:"quality_score" dynamodbav:"quality_score"`
	SampleData   []ExtractedActivity    `json:"sample_data" dynamodbav:"sample_data"`
	TestDuration int64                  `json:"test_duration" dynamodbav:"test_duration"` // milliseconds
	Errors       []string               `json:"errors" dynamodbav:"errors"`
	Warnings     []string               `json:"warnings" dynamodbav:"warnings"`
	Metrics      ExtractionMetrics      `json:"metrics" dynamodbav:"metrics"`
}

// ExtractedActivity represents a sample activity extracted during testing
type ExtractedActivity struct {
	Title       string `json:"title" dynamodbav:"title"`
	Date        string `json:"date" dynamodbav:"date"`
	Time        string `json:"time" dynamodbav:"time"`
	Description string `json:"description" dynamodbav:"description"`
	Location    string `json:"location" dynamodbav:"location"`
	Price       string `json:"price" dynamodbav:"price"`
	AgeRange    string `json:"age_range" dynamodbav:"age_range"`
	Category    string `json:"category" dynamodbav:"category"`
}

// ExtractionMetrics contains detailed metrics about extraction quality
type ExtractionMetrics struct {
	TitleCompleteness       float64 `json:"title_completeness" dynamodbav:"title_completeness"`
	DateCompleteness        float64 `json:"date_completeness" dynamodbav:"date_completeness"`
	DescriptionCompleteness float64 `json:"description_completeness" dynamodbav:"description_completeness"`
	LocationCompleteness    float64 `json:"location_completeness" dynamodbav:"location_completeness"`
	PriceCompleteness       float64 `json:"price_completeness" dynamodbav:"price_completeness"`
	OverallCompleteness     float64 `json:"overall_completeness" dynamodbav:"overall_completeness"`
}

// RecommendedSourceConfig contains the system-generated configuration recommendations
type RecommendedSourceConfig struct {
	ScrapingFrequency     string        `json:"scraping_frequency" dynamodbav:"scraping_frequency"`         // daily, weekly, monthly
	RateLimit             RateLimit     `json:"rate_limit" dynamodbav:"rate_limit"`
	EstimatedItemsPerScrape string      `json:"estimated_items_per_scrape" dynamodbav:"estimated_items_per_scrape"`
	EstimatedContentVolatility float64  `json:"estimated_content_volatility" dynamodbav:"estimated_content_volatility"`
	PreferredExtraction   string        `json:"preferred_extraction" dynamodbav:"preferred_extraction"`     // html, rss, api, structured-data
	BestSelectors         DataSelectors `json:"best_selectors" dynamodbav:"best_selectors"`
	TargetURLs           []string      `json:"target_urls" dynamodbav:"target_urls"`
}

// RateLimit defines scraping rate limits
type RateLimit struct {
	RequestsPerMinute     int   `json:"requests_per_minute" dynamodbav:"requests_per_minute"`
	DelayBetweenRequests  int64 `json:"delay_between_requests" dynamodbav:"delay_between_requests"` // milliseconds
	ConcurrentRequests    int   `json:"concurrent_requests" dynamodbav:"concurrent_requests"`
}

// DynamoSourceConfig represents the production configuration for an active source in DynamoDB
type DynamoSourceConfig struct {
	// Primary Keys
	PK string `json:"PK" dynamodbav:"PK"` // SOURCE#{source_id}
	SK string `json:"SK" dynamodbav:"SK"` // CONFIG

	// Source identification
	SourceID   string `json:"source_id" dynamodbav:"source_id"`
	SourceName string `json:"source_name" dynamodbav:"source_name"`
	SourceType string `json:"source_type" dynamodbav:"source_type"`
	BaseURL    string `json:"base_url" dynamodbav:"base_url"`

	// Target URLs and content extraction
	TargetURLs      []string      `json:"target_urls" dynamodbav:"target_urls"`
	ContentSelectors DataSelectors `json:"content_selectors" dynamodbav:"content_selectors"`

	// Scraping configuration
	ScrapingConfig DynamoScrapingConfig `json:"scraping_config" dynamodbav:"scraping_config"`

	// Data quality tracking
	DataQuality DataQuality `json:"data_quality" dynamodbav:"data_quality"`

	// Adaptive frequency management
	AdaptiveFrequency AdaptiveFrequency `json:"adaptive_frequency" dynamodbav:"adaptive_frequency"`

	// Configuration metadata
	Status       string    `json:"status" dynamodbav:"status"`         // active, inactive, suspended
	ActivatedBy  string    `json:"activated_by" dynamodbav:"activated_by"`
	ActivatedAt  time.Time `json:"activated_at" dynamodbav:"activated_at"`
	LastModified time.Time `json:"last_modified" dynamodbav:"last_modified"`

	// GSI Keys
	StatusKey   string `json:"StatusKey,omitempty" dynamodbav:"StatusKey,omitempty"`     // STATUS#{status}
	PriorityKey string `json:"PriorityKey,omitempty" dynamodbav:"PriorityKey,omitempty"` // PRIORITY#{priority}#{source_id}
}

// DynamoScrapingConfig defines how to scrape the source (DynamoDB version)
type DynamoScrapingConfig struct {
	Frequency         string    `json:"frequency" dynamodbav:"frequency"`                   // daily, weekly, monthly
	Priority          string    `json:"priority" dynamodbav:"priority"`                     // high, medium, low
	RateLimit         RateLimit `json:"rate_limit" dynamodbav:"rate_limit"`
	UserAgent         string    `json:"user_agent" dynamodbav:"user_agent"`
	RespectRobotsTxt  bool      `json:"respect_robots_txt" dynamodbav:"respect_robots_txt"`
	Timeout           int       `json:"timeout" dynamodbav:"timeout"`                       // seconds
	MaxRetries        int       `json:"max_retries" dynamodbav:"max_retries"`
	BackoffMultiplier float64   `json:"backoff_multiplier" dynamodbav:"backoff_multiplier"`
}

// DataQuality tracks the quality and reliability of a source
type DataQuality struct {
	ReliabilityScore         float64   `json:"reliability_score" dynamodbav:"reliability_score"`                   // 0.0 - 1.0
	LastSuccessfulScrape     time.Time `json:"last_successful_scrape" dynamodbav:"last_successful_scrape"`
	LastAttemptedScrape      time.Time `json:"last_attempted_scrape" dynamodbav:"last_attempted_scrape"`
	ConsecutiveFailures      int       `json:"consecutive_failures" dynamodbav:"consecutive_failures"`
	TotalSuccessfulScrapes   int       `json:"total_successful_scrapes" dynamodbav:"total_successful_scrapes"`
	TotalFailedScrapes       int       `json:"total_failed_scrapes" dynamodbav:"total_failed_scrapes"`
	AverageItemsPerScrape    float64   `json:"average_items_per_scrape" dynamodbav:"average_items_per_scrape"`
	ExpectedItemsRange       ItemRange `json:"expected_items_range" dynamodbav:"expected_items_range"`
	LastContentHashChange    time.Time `json:"last_content_hash_change" dynamodbav:"last_content_hash_change"`
	ContentVolatilityScore   float64   `json:"content_volatility_score" dynamodbav:"content_volatility_score"`     // 0.0 - 1.0
}

// ItemRange defines the expected range of items per scrape
type ItemRange struct {
	Min int `json:"min" dynamodbav:"min"`
	Max int `json:"max" dynamodbav:"max"`
}

// AdaptiveFrequency manages dynamic frequency adjustment
type AdaptiveFrequency struct {
	BaseFrequency    string    `json:"base_frequency" dynamodbav:"base_frequency"`       // original frequency
	CurrentFrequency string    `json:"current_frequency" dynamodbav:"current_frequency"` // current adjusted frequency
	NextAdjustment   time.Time `json:"next_adjustment" dynamodbav:"next_adjustment"`     // when to next evaluate
	AdjustmentReason string    `json:"adjustment_reason" dynamodbav:"adjustment_reason"` // why frequency was changed
	AdjustmentHistory []FrequencyAdjustment `json:"adjustment_history" dynamodbav:"adjustment_history"`
}

// FrequencyAdjustment tracks history of frequency changes
type FrequencyAdjustment struct {
	Timestamp    time.Time `json:"timestamp" dynamodbav:"timestamp"`
	OldFrequency string    `json:"old_frequency" dynamodbav:"old_frequency"`
	NewFrequency string    `json:"new_frequency" dynamodbav:"new_frequency"`
	Reason       string    `json:"reason" dynamodbav:"reason"`
	TriggerScore float64   `json:"trigger_score" dynamodbav:"trigger_score"`
}

// Helper functions to create primary keys for source management
func CreateSourcePK(sourceID string) string {
	return "SOURCE#" + sourceID
}

func CreateSourceSubmissionSK() string {
	return SourceStageSubmission
}

func CreateSourceAnalysisSK() string {
	return SourceStageAnalysis
}

func CreateSourceConfigSK() string {
	return SourceStageConfig
}

// Helper functions to generate GSI keys for source management
func GenerateSourceStatusKey(status string) string {
	return "STATUS#" + status
}

func GenerateSourcePriorityKey(priority, sourceID string) string {
	return "PRIORITY#" + priority + "#" + sourceID
}

// Validation functions
func (ss *SourceSubmission) Validate() error {
	if ss.SourceName == "" {
		return fmt.Errorf("source_name is required")
	}
	if ss.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	if ss.SourceType == "" {
		return fmt.Errorf("source_type is required")
	}
	if len(ss.ExpectedContent) == 0 {
		return fmt.Errorf("expected_content is required")
	}
	return nil
}

func (sc *DynamoSourceConfig) Validate() error {
	if sc.SourceName == "" {
		return fmt.Errorf("source_name is required")
	}
	if sc.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	if len(sc.TargetURLs) == 0 {
		return fmt.Errorf("target_urls is required")
	}
	if sc.ScrapingConfig.Frequency == "" {
		return fmt.Errorf("scraping frequency is required")
	}
	return nil
}