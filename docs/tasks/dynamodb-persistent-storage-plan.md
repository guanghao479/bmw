# DynamoDB Persistent Storage Implementation Plan

## Overview
Add DynamoDB as persistent storage and refine the data schema for Seattle Family Activities platform. This plan implements a 3-table architecture with proper entity separation and intelligent source management.

## Problem Statement
Current system uses S3-only storage which limits:
- Complex queries and relationships
- Real-time data updates
- Scalable source management
- Entity relationship modeling
- Performance optimization

## Solution Architecture

### 3-Table DynamoDB Design

#### Table 1: `family-activities` (Business Data)
**Purpose**: Store venues, events, programs, attractions for end users
**Partition Key**: Entity type + ID (e.g., `VENUE#ifly-seattle`)
**Sort Key**: Metadata type (e.g., `METADATA`, `INSTANCE#2025-03-01`)

**Entity Types**:
- **VENUE**: Physical locations (iFLY Seattle, Remlinger Farm, Pacific Science Center)
- **EVENT**: Time-bound happenings (festivals, workshops, birthday parties)
- **PROGRAM**: Recurring activities (summer camps, weekly classes, sports leagues)  
- **ATTRACTION**: Ongoing venue features (exhibits, playgrounds, permanent installations)

**Global Secondary Indexes**:
1. **location-date-index**: `GEO#{region}#{city}` → `DATE#{date}#TYPE#{entity_type}#{entity_id}`
2. **category-age-index**: `CAT#{category}#{age_group}` → `DATE#{date}#FEATURED#{featured}#{entity_id}`
3. **venue-activity-index**: `VENUE#{venue_id}` → `TYPE#{entity_type}#{start_date}#{entity_id}`
4. **provider-index**: `PROVIDER#{provider_id}` → `TYPE#{entity_type}#STATUS#{status}#{entity_id}`

#### Table 2: `source-management` (Source Configuration)
**Purpose**: Founder-submitted sources and their configurations
**Partition Key**: `SOURCE#{source_id}`
**Sort Key**: Lifecycle stage (`SUBMISSION`, `ANALYSIS`, `CONFIG`)

**Lifecycle**: Submission → Analysis → Approval → Production

**Global Secondary Indexes**:
1. **status-priority-index**: `STATUS#{status}` → `PRIORITY#{priority}#{source_id}`

#### Table 3: `scraping-operations` (Dynamic Scraping State)
**Purpose**: Scheduled tasks, execution status, performance metrics
**Partition Key**: Operation type + ID (`SCHEDULE#{date}`, `EXECUTION#{id}`)
**Sort Key**: Task details or status

**Features**:
- High-frequency writes for task scheduling
- TTL for auto-expiring old data (30-90 days)
- Performance metrics and monitoring

**Global Secondary Indexes**:
1. **next-run-index**: `NEXT_RUN#{timestamp}` → `PRIORITY#{priority}#{source_id}`

### Source Management Workflow

1. **Founder Submission**: Web form for new source discovery
2. **Automated Analysis**: Lambda function using Jina + OpenAI for content structure analysis
3. **Admin Approval**: Review dashboard with quality metrics and sample data
4. **Production Integration**: Seamless activation into scraping pipeline

## Implementation Plan

### Phase 1: Infrastructure & Core Schema (Week 1) - ✅ COMPLETED
**Goals**: Set up DynamoDB tables and basic Go service layers

**Tasks**:
1. ✅ Update CDK stack with 3 DynamoDB table definitions
2. ✅ Create IAM roles and permissions
3. ✅ Implement Go models for all entity types
4. Build basic CRUD operations (IN PROGRESS)
5. Set up CloudWatch monitoring

**Testing**: Verify table creation, basic read/write operations

**Completed Implementation Details**:
- **CDK Infrastructure**: Added 3 DynamoDB tables with 5 GSIs each
  - `seattle-family-activities` (business data)
  - `seattle-source-management` (source configuration)
  - `seattle-scraping-operations` (operational data with TTL)
- **Go Models**: Created comprehensive models for all entity types
  - `family_activities.go`: Venue, Event, Program, Attraction entities
  - `source_management.go`: Source submission, analysis, and configuration
  - `scraping_operations.go`: Task scheduling, execution tracking, metrics
- **IAM Permissions**: Updated Lambda and GitHub Actions roles with DynamoDB access
- **GSI Design**: Implemented efficient query patterns for all use cases

### Phase 2: Source Management System (Week 2)
**Goals**: Implement founder-driven source submission and automated analysis

**Tasks**:
1. Build source analyzer Lambda (Jina + OpenAI integration)
2. Create frontend admin interface for source submission
3. Implement submission → analysis → approval workflow
4. Add source configuration management
5. Test with 2-3 real Seattle sources

**Testing**: Complete source submission workflow, verify analysis quality

### Phase 3: Scraping & Data Population (Week 3)
**Goals**: Build new scraping system and start populating data

**Tasks**:
1. Implement scraping orchestrator using DynamoDB sources
2. Build scraping workers with entity classification
3. Start populating family-activities table with fresh data
4. Add adaptive scheduling and performance monitoring
5. Integrate with existing Lambda infrastructure

**Testing**: Verify data quality, scraping performance, entity classification

### Phase 4: Frontend Integration & Launch (Week 4)
**Goals**: Create new frontend version and launch system

**Tasks**:
1. Update API endpoints to use DynamoDB
2. Implement entity-specific views (venues, events, programs)
3. Add enhanced search and filtering capabilities
4. Create comprehensive documentation
5. Launch new system alongside existing S3 version

**Testing**: Full end-to-end testing, performance validation, user acceptance

## Data Schema Details

### Core Entity Structures

#### VENUE Record
```json
{
  "PK": "VENUE#ifly-seattle",
  "SK": "METADATA",
  "venue_name": "iFLY Seattle",
  "venue_type": "indoor-entertainment",
  "category": "active-sports",
  "subcategory": "adventure-sports",
  "address": "349 7th Ave, Seattle, WA 98104",
  "coordinates": {"lat": 47.6062, "lng": -122.3321},
  "region": "seattle-downtown",
  "amenities": ["parking", "restrooms", "food", "accessibility"],
  "operating_hours": {"monday": "10:00-22:00"},
  "age_groups": [{"category": "elementary", "min_age": 3, "max_age": 99}],
  "pricing": {"type": "paid", "cost": 79.95, "currency": "USD"},
  "provider_id": "PROVIDER#ifly-world",
  "status": "active"
}
```

#### PROGRAM Record (Recurring Classes/Camps)
```json
{
  "PK": "PROGRAM#soccer-tots-spring-2025",
  "SK": "METADATA",
  "program_name": "Soccer Tots Spring Session",
  "program_type": "sports-class",
  "category": "active-sports",
  "subcategory": "soccer",
  "venue_id": "VENUE#magnuson-park",
  "schedule": {
    "type": "recurring",
    "frequency": "weekly",
    "days_of_week": ["saturday"],
    "start_time": "10:00",
    "session_count": 8,
    "start_date": "2025-03-01"
  },
  "age_groups": [{"category": "toddler", "min_age": 18, "max_age": 36}],
  "registration": {"required": true, "method": "online"},
  "provider_id": "PROVIDER#seattle-parks"
}
```

#### Source Configuration
```json
{
  "PK": "SOURCE#seattle-childrens-theatre",
  "SK": "CONFIG",
  "source_name": "Seattle Children's Theatre",
  "base_url": "https://sct.org",
  "target_urls": ["https://sct.org/events", "https://sct.org/classes"],
  "content_selectors": {
    "title": ".event-title h2",
    "date": ".event-date",
    "description": ".event-content p"
  },
  "scraping_config": {
    "frequency": "weekly",
    "priority": "high",
    "rate_limit": {"requests_per_minute": 5}
  },
  "status": "active"
}
```

## Risk Analysis & Mitigations

### Technical Risks
1. **DynamoDB costs higher than expected**
   - Mitigation: Start with on-demand, monitor usage patterns
2. **Source analysis quality varies**
   - Mitigation: Manual review process, quality thresholds
3. **Performance impact on existing system**
   - Mitigation: Parallel operation, gradual transition

### Business Risks
1. **Learning curve for founders on new admin interface**
   - Mitigation: Simple UI design, comprehensive documentation
2. **Data quality concerns with fresh start**
   - Mitigation: Thorough testing, quality validation processes

## Success Metrics
- **Data Quality**: >95% successful entity classification
- **Source Reliability**: >90% successful scrapes for active sources
- **Performance**: <2 seconds for activity searches
- **Cost**: Monthly costs under $50
- **User Experience**: Enhanced functionality vs current system

## Dependencies
- AWS CDK CLI and proper AWS credentials
- Go 1.21+ development environment
- Jina AI and OpenAI API access
- Frontend development capabilities

## Timeline
- **Week 1**: Infrastructure and basic models
- **Week 2**: Source management system
- **Week 3**: Scraping and data population
- **Week 4**: Frontend integration and launch

Total: 4 weeks to full deployment with parallel operation alongside existing S3 system.