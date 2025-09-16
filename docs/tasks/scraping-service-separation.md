# FireCrawl Extract Service Integration - Implementation Plan

## Executive Summary

This document outlines the plan to replace the existing monolithic scraping system with FireCrawl's extract API, which returns structured activity data directly. This eliminates the need for both Jina and OpenAI processing, significantly simplifying the architecture.

## Current Architecture Analysis

### Current System (Monolithic)
- **Lambda Function** (`cmd/lambda/main.go`): Contains both scraping orchestration AND business logic
- **Jina Service** (`internal/services/jina.go`): Direct content extraction from websites
- **OpenAI Service** (`internal/services/openai.go`): Activity extraction from scraped content
- **S3 Service** (`internal/services/s3.go`): Data storage and backup
- **Infrastructure** (`infrastructure/`): AWS resources including Lambda, DynamoDB, S3, EventBridge

### Key Insight: FireCrawl Extract
- **Structured extraction**: Define activity schema, get JSON objects back
- **No content processing**: FireCrawl handles AI extraction internally
- **Direct business objects**: Activities, venues, events in proper format
- **Single API call**: Replaces entire Jina + OpenAI pipeline

### Current Problems
1. **Complexity**: Multi-step pipeline (Jina → OpenAI → Processing)
2. **Maintenance**: Complex content processing and AI prompt management
3. **Cost**: Multiple API calls (Jina + OpenAI) for each source
4. **Reliability**: Multiple failure points in extraction pipeline
5. **Development overhead**: Managing multiple external service integrations

## New Architecture Design

### New Architecture (FireCrawl Extract)

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend (GitHub Pages)                  │
│                           app/index.html                        │
└───────────────────────────┬─────────────────────────────────────┘
                            │ HTTP/REST
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Backend API Service                          │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │   API Gateway   │  │  Lambda/Handler │  │   DynamoDB      │ │
│  │                 │  │                 │  │                 │ │
│  │ - Activities    │  │ - Business Logic│  │ - Family Data   │ │
│  │ - Venues        │  │ - Data Validation│ │ - Source Config │ │
│  │ - Events        │  │ - API Endpoints │  │ - Activities    │ │
│  │ - Admin         │  │ - Scheduling    │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────┬─────────────────────────────────────┘
                              │ HTTP/REST API (Direct)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    FireCrawl Extract API                        │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │   Web Scraping  │  │  AI Extraction  │  │  Structured     │ │
│  │                 │  │                 │  │  JSON Output    │ │
│  │ - Content Fetch │  │ - Schema-based  │  │ - Activities    │ │
│  │ - JS Rendering  │  │ - AI Processing │  │ - Venues        │ │
│  │ - Rate Limiting │  │ - Data Cleaning │  │ - Events        │ │
│  │ - Error Handling│  │ - Quality Check │  │ - Validation    │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Service Responsibilities

#### Backend API Service (Core Business Logic)
- **Data Management**: CRUD operations for activities, venues, events
- **Business Logic**: Filtering, search, recommendations, admin functions
- **API Endpoints**: REST APIs for frontend consumption
- **Data Storage**: DynamoDB for primary data, S3 for static assets
- **Scheduling**: Trigger FireCrawl extract jobs on schedule
- **Data Validation**: Simple validation of structured data from FireCrawl
- **Monitoring**: System health, performance metrics

#### FireCrawl Extract API (External Service)
- **Web Scraping**: Handle all content extraction and JS rendering
- **AI Processing**: Built-in AI extraction using schema definitions
- **Structured Output**: Return properly formatted JSON activities
- **Rate Limiting**: Handle source website limits automatically
- **Error Handling**: Built-in retry logic and failure recovery
- **Quality Control**: Built-in data validation and cleaning

## FireCrawl Extract API Integration

### Direct FireCrawl API Usage

#### Extract Request

**POST** `https://api.firecrawl.dev/v1/extract`
```json
{
  "url": "https://www.parentmap.com/calendar?date=2025-09-15",
  "schema": {
    "type": "object",
    "properties": {
      "activities": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "title": {"type": "string"},
            "location": {
              "type": "object",
              "properties": {
                "name": {"type": "string"},
                "address": {"type": "string"},
                "coordinates": {"type": "string"}
              }
            },
            "schedule": {
              "type": "object",
              "properties": {
                "start_date": {"type": "string"},
                "start_time": {"type": "string"},
                "end_time": {"type": "string"}
              }
            },
            "age_groups": {"type": "array", "items": {"type": "string"}},
            "pricing": {"type": "string"},
            "description": {"type": "string"},
            "registration_url": {"type": "string"}
          }
        }
      }
    }
  }
}
```

#### Extract Response

**Response from FireCrawl**
```json
{
  "success": true,
  "data": {
    "activities": [
      {
        "title": "Family Storytime at Central Library",
        "location": {
          "name": "Seattle Central Library",
          "address": "1000 4th Ave, Seattle, WA 98104",
          "coordinates": "47.6062,-122.3321"
        },
        "schedule": {
          "start_date": "2025-09-15",
          "start_time": "10:00",
          "end_time": "11:00"
        },
        "age_groups": ["toddlers", "preschoolers"],
        "pricing": "Free",
        "description": "Join us for an interactive storytime featuring books, songs, and activities for young children.",
        "registration_url": "https://www.spl.org/events/family-storytime"
      }
    ]
  },
  "metadata": {
    "url": "https://www.parentmap.com/calendar?date=2025-09-15",
    "extract_time": "2025-09-15T10:04:30Z",
    "credits_used": 50
  }
}
```

#### Backend Processing

**Simplified Backend Flow**
```go
// Direct FireCrawl integration in Lambda
func (h *Handler) ProcessSource(source Source) error {
    // 1. Call FireCrawl extract API
    extractReq := &FireCrawlExtractRequest{
        URL: source.URL,
        Schema: getActivityExtractionSchema(),
    }

    response, err := h.firecrawlClient.Extract(extractReq)
    if err != nil {
        return err
    }

    // 2. Simple validation
    if err := h.validateActivities(response.Data.Activities); err != nil {
        return err
    }

    // 3. Store directly in DynamoDB
    return h.storeActivities(response.Data.Activities, source)
}
```

## Implementation Plan

### Branch Strategy
- **Work on main branch** for development
- **Create feature branch** `feature/firecrawl-extract` to preserve current code
- **Simple rollback** via `git checkout feature/firecrawl-extract`

### Week 1: FireCrawl Extract Integration
1. **Set up FireCrawl API client** with extract endpoint
2. **Define activity extraction schema** for structured output
3. **Create direct integration** in Lambda function
4. **Test extract API** with sample sources
5. **Validate structured data output**

### Week 2: Backend Cleanup & Refactoring
1. **Remove Jina service** (`internal/services/jina.go`) completely
2. **Remove OpenAI service** (`internal/services/openai.go`) completely
3. **Update Lambda function** to use FireCrawl extract directly
4. **Simplify data flow**: extract → validate → store
5. **Remove related infrastructure** (API keys, IAM permissions)

### Week 3: Testing & Deployment
1. **Integration testing** with real sources
2. **Validate data quality** and structure consistency
3. **Performance testing** and optimization
4. **Deploy simplified architecture**
5. **Monitor and adjust** extraction quality

## Technical Implementation Details

### Complete Removal List

#### Backend Code (Complete Files)
- `internal/services/jina.go` (entire file)
- `internal/services/openai.go` (entire file)
- All content processing and parsing logic
- OpenAI/Jina imports and dependencies throughout codebase

#### Infrastructure Code
- Lambda environment variables: `JINA_API_KEY`, `OPENAI_API_KEY`
- IAM permissions for external API access
- OpenAI/Jina CloudWatch metrics and cost tracking
- Token usage and quality scoring infrastructure

#### Data Models
- OpenAI response structures
- Content processing models
- AI validation and retry logic

### FireCrawl Extract Integration

```go
// Simplified Lambda structure
type Handler struct {
    firecrawlClient *firecrawl.Client
    dynamoClient    *dynamodb.Client
    s3Client        *s3.Client
}

// Direct extraction
func (h *Handler) ProcessSource(source Source) error {
    // 1. Call FireCrawl extract with schema
    extractReq := &firecrawl.ExtractRequest{
        URL: source.URL,
        Schema: getActivityExtractionSchema(),
    }

    response, err := h.firecrawlClient.Extract(extractReq)
    if err != nil {
        return fmt.Errorf("extract failed: %w", err)
    }

    // 2. Simple validation
    activities := response.Data.Activities
    if len(activities) == 0 {
        return fmt.Errorf("no activities extracted")
    }

    // 3. Store directly
    return h.storeActivities(activities, source)
}
```

### Activity Extraction Schema

```go
// Schema definition for FireCrawl extract
func getActivityExtractionSchema() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "activities": map[string]interface{}{
                "type": "array",
                "items": map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "title": map[string]interface{}{"type": "string"},
                        "location": map[string]interface{}{
                            "type": "object",
                            "properties": map[string]interface{}{
                                "name": map[string]interface{}{"type": "string"},
                                "address": map[string]interface{}{"type": "string"},
                                "coordinates": map[string]interface{}{"type": "string"},
                            },
                        },
                        "schedule": map[string]interface{}{
                            "type": "object",
                            "properties": map[string]interface{}{
                                "start_date": map[string]interface{}{"type": "string"},
                                "start_time": map[string]interface{}{"type": "string"},
                                "end_time": map[string]interface{}{"type": "string"},
                            },
                        },
                        "age_groups": map[string]interface{}{
                            "type": "array",
                            "items": map[string]interface{}{"type": "string"},
                        },
                        "pricing": map[string]interface{}{"type": "string"},
                        "description": map[string]interface{}{"type": "string"},
                        "registration_url": map[string]interface{}{"type": "string"},
                    },
                },
            },
        },
    }
}
```

## Benefits of FireCrawl Extract Approach

### Massive Simplification
- **Single API call** replaces complex Jina → OpenAI pipeline
- **No content processing** - get structured data directly
- **Better data quality** - FireCrawl's AI extraction likely superior
- **Faster processing** - no multi-step content → AI → parsing workflow

### Cost & Maintenance Benefits
- **Lower operational costs** - single service vs multiple APIs
- **Reduced complexity** - remove ~2000+ lines of scraping code
- **Easier debugging** - single point of failure vs complex pipeline
- **Better reliability** - fewer external dependencies

### Development Benefits
- **Faster implementation** - 3 weeks vs 5+ weeks
- **Simpler testing** - test extraction schema vs complex prompts
- **Easy scaling** - FireCrawl handles rate limiting and infrastructure
- **Future flexibility** - can switch to self-hosted FireCrawl later

## Files to Remove/Modify

### Complete File Removal
- `internal/services/jina.go` (entire file - ~300+ lines)
- `internal/services/openai.go` (entire file - ~500+ lines)
- Related test files and integration tests

### Infrastructure Updates
- Remove `JINA_API_KEY` and `OPENAI_API_KEY` from Lambda environment
- Add `FIRECRAWL_API_KEY` environment variable
- Remove OpenAI/Jina IAM permissions
- Simplify CloudWatch metrics (remove token tracking)

### Lambda Function Changes
- Replace scraping orchestration with direct FireCrawl calls
- Remove complex content processing logic
- Simplify error handling (single API vs multiple)
- Update data models to match FireCrawl output

## Risk Assessment

### Low Risk (Simplified Architecture)
- **Single dependency** - only FireCrawl vs multiple services
- **Proven service** - FireCrawl is established and reliable
- **Simple rollback** - git checkout to restore original code
- **No data migration** - activities remain in DynamoDB

### Minimal Risks
- **FireCrawl costs** - pay-per-extract pricing
- **API rate limits** - handled by FireCrawl automatically
- **Schema changes** - may need to adjust extraction schema

## Success Metrics

### Performance
- **Extraction time** - target < 30 seconds per source (vs 2+ minutes)
- **Data quality** - same or better activity extraction accuracy
- **System uptime** - > 99.5% (improved reliability)

### Functionality
- **Structured output** - properly formatted activities without post-processing
- **Error handling** - simpler error recovery with single API
- **Cost efficiency** - lower monthly costs vs current Jina + OpenAI

### Development
- **Code reduction** - remove 800+ lines of complex scraping code
- **Faster deployment** - simpler Lambda with fewer dependencies
- **Easier maintenance** - single external API vs multiple services

## Next Steps

1. **Get approval** for this FireCrawl extract approach
2. **Create feature branch** to preserve current scraping code
3. **Begin Week 1** - FireCrawl API integration and schema definition
4. **Week 2** - Remove all Jina/OpenAI code and infrastructure
5. **Week 3** - Testing and deployment of simplified architecture

---

**Document Version**: 2.0 (Updated for FireCrawl Extract)
**Last Updated**: 2025-09-16
**Next Review**: After implementation completion