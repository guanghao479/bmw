# Seattle Family Activities MVP Implementation Plan

## Overview
Implement an ultra-minimal MVP for a family activities platform that scrapes Seattle-area family activity websites using Jina + OpenAI and serves data via S3 JSON files to a GitHub Pages frontend.

## Architecture Summary
- **Frontend**: GitHub Pages static site in `/app/` directory
- **Backend**: AWS Lambda (us-west-2) with Go runtime
- **Storage**: S3 bucket with public JSON files
- **Scraping**: Jina AI Reader + OpenAI GPT-4o mini
- **Scheduling**: EventBridge every 6 hours
- **No database**: Pure file-based storage
- **No admin portal**: AWS Console for manual operations

## Data Schema Design

### Target Seattle Sources
1. Seattle's Child - https://www.seattleschild.com
2. Tinybeans Seattle - https://tinybeans.com/seattle/
3. Seattle Fun for Kids - https://www.seattlefunforkids.com
4. Macaroni Kid West Seattle - https://westseattle.macaronikid.com
5. ParentMap - https://www.parentmap.com
6. PEPS - https://www.peps.org/
7. Eventbrite - Seattle family events
8. Meetup - Seattle family groups

### Standardized Terminology

**Activity Types:**
- `class` - Ongoing instructional activities (music lessons, art classes)
- `camp` - Multi-day intensive programs (summer camps, day camps)
- `event` - One-time or special occasions (festivals, performances)
- `performance` - Shows and entertainment (theater, concerts)
- `free-activity` - No-cost community activities (library programs)

**Categories:**
- `arts-creativity` - Art, music, dance, theater, crafts
- `active-sports` - Physical activities, sports, outdoor adventures
- `educational-stem` - Learning activities, science, technology, museums
- `entertainment-events` - Festivals, performances, cultural events
- `camps-programs` - Structured multi-day or multi-week programs
- `free-community` - No-cost activities, library programs

**Age Groups:**
- `infant` (0-12 months), `toddler` (1-3 years), `preschool` (3-5 years)
- `elementary` (6-10 years), `tween` (11-14 years), `teen` (15-18 years)
- `adult` (18+ years), `all-ages` (No restrictions)

### Enhanced JSON Schema

```json
{
  "metadata": {
    "lastUpdated": "2024-08-13T10:30:00Z",
    "totalActivities": 125,
    "sources": ["seattleschild.com", "tinybeans.com/seattle", "..."],
    "nextUpdate": "2024-08-13T16:30:00Z",
    "version": "1.0.0",
    "region": "us-west-2",
    "coverage": "Seattle Metro Area"
  },
  "activities": [
    {
      "id": "act_a1b2c3d4",
      "title": "Toddler Music & Movement Class",
      "description": "Interactive music class for toddlers...",
      "type": "class",
      "category": "arts-creativity",
      "subcategory": "music",
      "schedule": {
        "type": "recurring",
        "startDate": "2024-09-01",
        "endDate": "2024-12-15",
        "frequency": "weekly",
        "daysOfWeek": ["tuesday", "thursday"],
        "times": [{"startTime": "10:00", "endTime": "10:45"}],
        "duration": "45 minutes",
        "sessions": 16
      },
      "ageGroups": [{
        "category": "toddler",
        "minAge": 18,
        "maxAge": 36,
        "unit": "months",
        "description": "18 months - 3 years"
      }],
      "location": {
        "name": "Seattle Music Academy",
        "address": "123 Pine Street, Seattle, WA 98101",
        "neighborhood": "Capitol Hill",
        "city": "Seattle",
        "region": "Seattle Metro",
        "venueType": "indoor"
      },
      "pricing": {
        "type": "paid",
        "cost": 180,
        "currency": "USD",
        "unit": "session",
        "description": "$180 for 16-week session"
      },
      "registration": {
        "required": true,
        "method": "online",
        "url": "https://example.com/register",
        "status": "open"
      },
      "tags": ["music", "movement", "toddler", "parent-child"],
      "provider": {
        "name": "Seattle Music Academy",
        "type": "business",
        "verified": true
      },
      "source": {
        "url": "https://www.seattleschild.com/music-classes",
        "domain": "seattleschild.com",
        "scrapedAt": "2024-08-13T08:15:00Z"
      },
      "featured": false,
      "status": "active"
    }
  ]
}
```

## Implementation Tasks

### Phase 1: Repository & Infrastructure Setup

**Task 1.1: Repository Restructuring**
- Move existing `index.html`, `script.js`, `styles.css` to `/app/` directory
- Create root `index.html` with redirect to `/app/`
- Update GitHub Pages source configuration

**Task 1.2: AWS CDK Infrastructure (us-west-2)**
- Create `infrastructure/` directory with CDK stack
- S3 bucket: `family-events-mvp-data-usw2` with public read access
- Lambda function: EventScraper with 15min timeout, 1GB memory
- EventBridge: Schedule every 6 hours
- CloudWatch: Email alerts for Lambda failures
- Estimated time: 2 hours

**Task 1.3: Go Project Structure**
- Create `backend/` directory with Go modules
- Implement enhanced data models matching JSON schema
- Set up internal packages: models, services (jina, openai, s3)
- Estimated time: 1 hour

### Phase 2: Scraping Implementation

**Task 2.1: Jina AI Client**
- HTTP client for Jina Reader API
- Error handling and retries
- Content extraction and cleaning
- Estimated time: 1 hour

**Task 2.2: Enhanced OpenAI Integration**
- GPT-4o mini integration for activity extraction
- Custom prompt template for Seattle activities
- JSON parsing and validation
- Activity ID generation and deduplication
- Estimated time: 2 hours

**Task 2.3: S3 Service**
- Upload activities JSON to S3 with proper headers
- Create daily snapshots for backup
- Handle upload errors and retries
- Estimated time: 1 hour

**Task 2.4: Main Lambda Function**
- Orchestrate scraping from 8 Seattle sources
- Parallel processing with error handling
- Data deduplication and validation
- Upload final JSON to S3
- Estimated time: 1 hour

### Phase 3: Frontend Enhancement

**Task 3.1: Enhanced Frontend Script**
- Load data from S3 instead of embedded JSON
- Support new activity schema with enhanced filtering
- Auto-refresh every 30 minutes
- Offline fallback to cached data
- Enhanced error handling and user notifications
- Estimated time: 2 hours

**Task 3.2: Configuration**
- S3 endpoint configuration
- Environment detection (dev vs prod)
- Cache and timeout settings
- Estimated time: 30 minutes

### Phase 4: Deployment & Testing

**Task 4.1: GitHub Actions**
- CDK deployment workflow
- Lambda function build and deployment
- Frontend deployment to GitHub Pages
- Estimated time: 1 hour

**Task 4.2: End-to-End Testing**
- Test S3 CORS configuration
- Verify Lambda execution with Seattle sources
- Test frontend data loading and error handling
- Mobile responsiveness validation
- Estimated time: 1 hour

## Technical Implementation Details

### Go Data Models
```go
type Activity struct {
    ID          string     `json:"id"`
    Title       string     `json:"title"`
    Description string     `json:"description"`
    Type        string     `json:"type"`        // class|camp|event|performance|free-activity
    Category    string     `json:"category"`    // arts-creativity|active-sports|etc
    Subcategory string     `json:"subcategory"`
    Schedule    Schedule   `json:"schedule"`
    AgeGroups   []AgeGroup `json:"ageGroups"`
    Location    Location   `json:"location"`
    Pricing     Pricing    `json:"pricing"`
    Registration Registration `json:"registration"`
    Tags        []string   `json:"tags"`
    Provider    Provider   `json:"provider"`
    Source      Source     `json:"source"`
    Featured    bool       `json:"featured"`
    Status      string     `json:"status"`
}
```

### OpenAI Extraction Prompt
- Extract activities from Seattle family websites
- Standardize to consistent schema
- Focus on family-friendly content only
- Validate data types and required fields

### Frontend Enhancements
- Load from S3: `https://family-events-mvp-data-usw2.s3.us-west-2.amazonaws.com/events/events.json`
- Enhanced filtering by activity type, category, age group
- Real-time data refresh with user notifications
- Graceful degradation for offline usage

## Success Metrics

**Data Quality:**
- Successfully scrape 50+ activities from 5+ Seattle sources
- 90%+ data extraction accuracy with complete fields
- Consistent categorization across all sources

**Technical Performance:**
- Sub-3 second initial load time
- 99%+ uptime with S3 reliability
- Graceful error handling for all failure scenarios

**Cost Efficiency:**
- Total monthly cost under $10
- Scalable to 10x traffic without significant cost increase

## Risk Mitigation

**Technical Risks:**
- S3 outage → Browser caching + embedded data fallback
- Lambda timeout → Batch processing with retries
- API rate limits → Exponential backoff
- JSON file growth → Monitor size, implement pagination

**Data Quality Risks:**
- Source website changes → Multiple sources for redundancy
- Extraction accuracy → Manual validation during MVP
- Duplicate content → ID-based deduplication

**Business Risks:**
- Legal compliance → Only public data, respect robots.txt
- Data freshness → 6-hour update cycle with manual triggers

## Next Steps After Plan Approval

1. Create directory structure and move files
2. Set up AWS CDK infrastructure
3. Implement Go backend with enhanced data models
4. Update frontend for S3 integration
5. Deploy and test end-to-end functionality

**Estimated Total Development Time: 12-16 hours over 1.5 days**