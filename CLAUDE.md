# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Testing Directory Policy [CRITICAL - ALWAYS FOLLOW]

**ALL testing artifacts, temporary files, build outputs, and test data MUST go in `/testing/` directory.**

### Mandatory Rules:
1. **NEVER create test files outside `/testing/` directory**
2. **ALL temporary files** → `/testing/temp/`
3. **ALL build artifacts** → `/testing/build/`
4. **ALL test responses/data** → `/testing/data/`
5. **ALL compiled binaries** → `/testing/bin/`

### Examples of what goes in `/testing/`:
- Compiled Go binaries (lambda, bootstrap, etc.)
- JSON response files (response.json, latest_check.json, etc.)
- Test output files
- Temporary scripts
- Debug logs
- Sample data files

### How to obey this rule:
- Before creating ANY file, ask: "Is this a test/temp/build artifact?"
- If YES → put it in `/testing/` subdirectory
- Use absolute paths like `/Users/guanghaoding/personal-projects/bmw/testing/build/lambda`
- When running commands, redirect output to `/testing/` when possible

**This prevents repository pollution and maintains clean development environment.**

## Project Overview

This is a Seattle family activities platform that automatically scrapes and curates family-friendly events, activities, and venues from dynamically managed sources. The system consists of:

- **Production System**: AWS Lambda backend (Go) uses FireCrawl for extraction, stores data in DynamoDB, serves via database API and GitHub Pages frontend
- **Demo/Development**: Static frontend in `/app/` directory can run independently with embedded sample data
- **Architecture**: Serverless, database-direct, cost-effective (under $10/month), scalable to 50+ sources

**Live System**:
- Frontend: https://guanghao479.github.io/bmw/
- Data API: Database-hosted JSON served via API Gateway with real-time updates
- Sources: All sources managed dynamically via admin interface (no hardcoded sources)
- Admin Interface: Real-time source management, event approval workflow

## Architecture

### Production System Architecture (Database-Direct)
- **Frontend**: GitHub Pages static site (`/app/`) loading data from database API via REST API
- **Backend**: AWS Lambda functions (Go runtime) in us-west-2 region
- **Data Pipeline**: Admin URL Submission → FireCrawl Extract → DynamoDB Storage → Admin Review → API Serving
- **Infrastructure**: API Gateway, DynamoDB, CloudWatch (monitoring), SNS (alerts)
- **Storage**: DynamoDB for primary data, S3 for backups only
- **Region**: us-west-2 (Oregon) for optimal performance and cost

### NEW ARCHITECTURE BENEFITS:
- **Real-time data**: No S3 upload delays, immediate display after approval
- **Dynamic sources**: All sources managed via admin interface, no hardcoded lists
- **URL deduplication**: Prevents duplicate crawling attempts
- **Auto-source creation**: Successful extractions automatically become active sources
- **Manual re-extraction**: On-demand source updates via admin interface

### Frontend Architecture (Both Demo and Production)
- **Static single-page application**: No server-side rendering, pure client-side
- **Data source flexibility**: Database API (production) → Cache → Embedded JSON (offline fallback)
- **Class-based structure**: Main functionality in `FamilyEventsApp` class
- **Modern CSS**: CSS custom properties for theming and design system
- **Responsive design**: Mobile-first with breakpoints at 480px, 768px, 1200px
- **Auto-refresh**: 30-minute data refresh cycle with manual refresh capability
- **Offline support**: LocalStorage caching with 24-hour expiration
- **Admin Interface**: Real-time source management, event approval workflow, re-extraction capabilities

## Key Components

### Data Sources (Dynamic Management)
**All sources now managed dynamically via admin interface - no hardcoded sources!**

**Source Management Features:**
- **Dynamic addition**: Submit any URL + schema via admin interface
- **URL deduplication**: Automatic checking for duplicate submissions
- **Auto-activation**: Successful extractions automatically become active sources
- **Manual re-extraction**: Trigger on-demand updates for any source
- **Status management**: Active/inactive/paused source states
- **Priority handling**: High/medium/low priority source processing

**Example Sources (user-managed):**
- Seattle's Child, PEPS, ParentMap, Tinybeans, Seattle Fun for Kids, etc.
- Any community calendar or event listing site can be added

### Data Structure & Schema
**Enhanced Backend Schema:** Activities include comprehensive metadata (schedule, pricing, registration, age groups, location details)
**Frontend Legacy Format:** Simplified schema for UI compatibility with three types: `events`, `activities`, `venues`
**Schema Conversion:** Backend→Frontend format conversion via `convertToLegacyFormat()` function

### Backend Services (Go/AWS) - UPDATED ARCHITECTURE
- **FireCrawl Service** (`internal/services/firecrawl.go`): Direct structured data extraction with built-in AI
- **DynamoDB Service** (`internal/services/dynamodb.go`): Primary data storage, source management, admin events
- **S3 Service** (`internal/services/s3.go`): Backup storage only (not main data flow)
- **Admin API** (`cmd/admin_api/main.go`): Handles URL submission, event approval, source management
- **Scraping Orchestrator** (`cmd/scraping_orchestrator/main.go`): Database-driven scraping (no hardcoded sources)

**REMOVED SERVICES:**
- ~~Jina Service~~ (replaced by FireCrawl)
- ~~OpenAI Service~~ (FireCrawl handles AI extraction internally)

### Frontend Architecture (`app/script.js`)
- **FamilyEventsApp class**: Handles all UI interactions and data management
- **Data loading**: Fetch from Database API → Cache in localStorage → Fallback to embedded data
- **Real-time filtering**: Category, search term, and age group filtering
- **Modal system**: Detailed item views with enhanced metadata
- **Auto-refresh system**: Background data updates with user notifications

### Admin Frontend Architecture (`app/admin.js`)
- **Source Management**: Real-time loading of active sources from database
- **URL Submission**: Submit new URLs with deduplication checking
- **Event Approval**: Review and approve/reject pending events
- **Re-extraction**: Trigger manual re-extraction for any source
- **Real-time Updates**: Live data from database APIs

### CSS Design System (`app/styles.css`)
- **CSS custom properties**: Colors, spacing, typography, transitions
- **Component-based classes**: `.card`, `.filter-btn`, `.modal`, responsive grid
- **Glassmorphism effects**: Modern shadows, blur effects, accessibility focus states
- **Mobile-first responsive**: Breakpoints optimized for family browsing patterns

## Development

### Frontend Development
**Running Locally:**
- Open `app/index.html` in browser (no build process required)
- For API integration testing, serve via local server (e.g., `python -m http.server 8000`)
- Admin interface: Open `app/admin.html` for source management and event approval

**Configuration:**
- Environment detection: Automatic dev/prod mode based on hostname
- API endpoints: Configured in `loadConfiguration()` method for database API
- Debug mode: Available in development for detailed logging
- Data source priorities: Database API → Cache → Embedded fallback

### Backend Development (Go + AWS)
**Prerequisites:**
- Go 1.21+, AWS CLI configured, AWS CDK CLI
- Environment variables: `AWS_PROFILE`, `FIRECRAWL_API_KEY`
- DynamoDB tables: `seattle-family-activities`, `seattle-source-management`, `seattle-admin-events`

**Local Testing:**
```bash
cd backend
go test ./...                              # Run all unit tests
go test -tags=integration ./internal/services -run TestFireCrawl  # Test FireCrawl integration
cd ../testing && node run_frontend_tests.js  # Run frontend API integration tests
```

**Infrastructure Management:**
```bash
cd infrastructure
npm install                               # Install CDK dependencies
cdk deploy                               # Deploy to AWS
cdk destroy                              # Clean up resources
```

### Testing Commands
**Frontend Testing:**
- Manual: Open `app/index.html` and verify filtering, search, modal functionality
- Mobile: Test responsive design at 480px, 768px, 1200px breakpoints
- Data loading: Verify Database API fetch, localStorage caching, and fallback behavior
- Admin interface: Test source management, URL submission, event approval workflow
- Run comprehensive tests: `cd testing && node run_frontend_tests.js`

**Backend Testing:**
- `go test ./internal/models/` - Data model validation
- `go test ./internal/services/` - Service integration tests (FireCrawl, DynamoDB)
- `go test ./cmd/admin_api/` - Admin API endpoint tests
- `go test ./cmd/scraping_orchestrator/` - Database-driven orchestration tests
- `cd testing && go run new_architecture_tests.go` - Comprehensive architecture validation
- Integration: Tests real APIs (FireCrawl, DynamoDB) with dynamic sources

### Making Changes
**Frontend Updates:**
- **Styling**: Update CSS custom properties in `app/styles.css` for global changes
- **New features**: Extend `FamilyEventsApp` class methods in `app/script.js`
- **Admin features**: Extend admin interface functionality in `app/admin.js`
- **Data schema**: Modify `convertToLegacyFormat()` for new backend fields

**Backend Updates:**
- **New sources**: Add via admin interface (no code changes needed)
- **Data models**: Update `internal/models/` and regenerate tests
- **API endpoints**: Extend admin API in `cmd/admin_api/main.go`
- **Services**: Extend FireCrawl or DynamoDB services in `internal/services/`

**Infrastructure Updates:**
- **AWS resources**: Modify `infrastructure/lib/mvp-stack.ts`
- **Deployment**: Update GitHub Actions workflows in `.github/workflows/`

## Technical Notes

### Frontend
- **No build tools**: Intentional simplicity - pure HTML/CSS/JS, no package.json required
- **Data flexibility**: Can load from Database API (production) → Cache → Embedded JSON (offline)
- **Dynamic UI**: Modal styles injected when needed, intersection observer for animations
- **Modern JavaScript**: async/await, arrow functions, template literals, fetch API
- **Admin Interface**: Real-time source management, event approval, re-extraction
- **Image handling**: Unsplash URLs with SVG placeholder fallback, lazy loading

### Backend
- **Go modules**: Clean dependency management, AWS SDK v2, comprehensive testing
- **Database-first**: DynamoDB primary storage, real-time data access
- **Error handling**: Graceful degradation, retry logic, detailed CloudWatch logging
- **Cost optimization**: FireCrawl structured extraction, efficient DynamoDB operations
- **Security**: No secrets in code, AWS IAM roles, CORS properly configured
- **Dynamic scaling**: No hardcoded sources, unlimited source addition capability

## AWS Infrastructure

### Services Used
- **Lambda**: Go runtime, 15-minute timeout, 1GB memory, us-west-2 region (2 functions: Admin API + Orchestrator)
- **API Gateway**: REST API for database endpoints, CORS configured for frontend
- **DynamoDB**: Primary data storage (4 tables: activities, source management, operations, admin events)
- **S3**: Backup storage only, CORS configured for legacy compatibility
- **CloudWatch**: Error monitoring, performance metrics, log aggregation
- **SNS**: Email alerts for Lambda failures and data quality issues
- **IAM**: Least-privilege roles for Lambda DynamoDB and API Gateway access

### Cost Structure (Monthly)
- **Lambda**: ~$2-3 (execution time, memory usage)
- **DynamoDB**: ~$1-2 (read/write operations, storage)
- **API Gateway**: ~$0.50-1 (API calls, data transfer)
- **S3**: ~$0.50 (backup storage only)
- **CloudWatch**: ~$1 (logs, metrics, alarms)
- **API calls**: ~$3-5 (FireCrawl extractions)
- **Total**: Under $10/month, scales to 50+ sources

### Performance Metrics
- **Scraping**: 10-20 seconds per source via FireCrawl, unlimited sources
- **Data processing**: Direct structured extraction, no post-processing needed
- **Frontend load**: Sub-2 second initial load via database API, real-time updates
- **Admin workflow**: Instant URL submission, real-time event approval
- **Reliability**: 99%+ uptime via DynamoDB, graceful error handling and fallbacks

## Data Pipeline

### NEW Scraping Process (Database-Direct)
1. **Admin submits URL + schema** via admin interface
2. **URL deduplication check** against existing events and sources
3. **FireCrawl structured extraction** with schema-based AI processing
4. **Database storage** as pending events (AdminEvent table)
5. **Auto-source creation** for successful extractions
6. **Admin review and approval** via admin interface
7. **Real-time frontend updates** via database API (no delays)

### LEGACY Process (for reference)
~~EventBridge trigger → Hardcoded sources → Jina/OpenAI → S3 storage~~

### Data Quality
- **URL deduplication**: Prevents duplicate crawling attempts at submission time
- **Structured extraction**: FireCrawl provides schema-validated JSON output
- **Admin review**: Manual approval process ensures quality before publication
- **Validation**: Schema validation, required field checking
- **Monitoring**: Success rates, extraction quality, source reliability
- **Fallback**: Database redundancy, frontend cache, embedded data for outages

### Source Management (Completely Redesigned)
- **Dynamic addition**: Any URL can be added via admin interface
- **Auto-activation**: Successful extractions automatically become active sources
- **Priority management**: High/medium/low priority via admin interface
- **Re-extraction**: Manual trigger for any source via admin interface
- **Status management**: Active/inactive/paused states
- **Reliability scoring**: Track success rates per source
- **Zero hardcoding**: All sources managed through database
- **Error handling**: Individual source failures don't break entire pipeline

## Deployment

### GitHub Actions Workflows (Updated for Database-Direct)
- **Main deployment** (`.github/workflows/deploy.yml`): CDK + Lambda + API Gateway + GitHub Pages
- **Database testing** (`.github/workflows/test-scraper.yml`): API endpoints, DynamoDB connectivity
- **Automated testing**: API CORS validation, database data quality, accessibility audits
- **Frontend testing**: Database API integration, admin interface functionality
- **Performance monitoring**: Lighthouse integration with quality thresholds
- **Security**: OIDC authentication with AWS (no long-lived access keys)
- **Removed**: JINA_API_KEY, OPENAI_API_KEY references

### Infrastructure as Code
- **AWS CDK**: TypeScript stack definition in `infrastructure/lib/mvp-stack.ts`
- **Deployment**: Automated via GitHub Actions on main branch push
- **Environment management**: Dev/prod configuration via GitHub secrets and OIDC
- **Authentication**: OpenID Connect (OIDC) for secure AWS access without access keys
- **Rollback**: CDK stack versioning and automated rollback capabilities
- **Setup guide**: See `docs/aws-oidc-setup.md` for OIDC configuration instructions

## Monitoring & Alerts

### CloudWatch Integration
- **Lambda metrics**: Duration, errors, cost tracking, memory usage (Admin API + Orchestrator)
- **API Gateway metrics**: Request count, latency, error rates, throttling
- **DynamoDB metrics**: Read/write capacity, throttling, latency
- **Data quality**: Approved events count, pending events, source success rates
- **Frontend metrics**: Page load times, error rates, API response times
- **Cost monitoring**: FireCrawl usage, DynamoDB consumption, AWS service costs

### Error Handling
- **Lambda failures**: SNS email alerts, automatic retries, graceful degradation
- **API Gateway failures**: Frontend fallback to cache then embedded data
- **DynamoDB throttling**: Exponential backoff, retry logic in services
- **FireCrawl API limits**: Rate limiting, queue management, retry logic
- **Database outages**: Frontend cache utilization, embedded data fallback
- **Data corruption**: Validation checks, DynamoDB point-in-time recovery, manual intervention
- **Admin workflow errors**: Clear error messages, graceful handling in admin interface

## Completed Tasks

### Database-Direct Architecture Implementation (September 2025)
- ✅ **Phase 1**: Remove hardcoded sources from orchestrator
- ✅ **Phase 2**: Add URL deduplication logic to admin API
- ✅ **Phase 3**: Create GET /api/events/approved endpoint for main frontend
- ✅ **Phase 4**: Update main frontend to use database API instead of S3
- ✅ **Phase 5**: Remove S3 dependencies from orchestrator
- ✅ **Phase 6**: Connect admin crawling to source management
- ✅ **Phase 7**: Enhance source management tab with real data
- ✅ **Phase 8**: Update backend and frontend tests for new flow
- ✅ **Phase 9**: Update AWS infrastructure configuration
- ✅ **Phase 10**: Update GitHub Actions workflows
- ✅ **Phase 11**: Update CLAUDE.md documentation

### Original MVP Implementation Status
- ✅ **Phase 1-2**: Infrastructure, backend scraping system (24 hours actual vs 12 estimated)
- ✅ **Phase 3-4**: Frontend enhancement, deployment pipeline (completed August 15, 2025)
- ✅ **Production ready**: Live at https://guanghao479.github.io/bmw/ with database-direct architecture

### System Validation (Updated for Database-Direct)
- **Backend tests**: All passing (unit + integration with FireCrawl, DynamoDB)
- **Frontend tests**: 20/20 passing (database API integration, admin interface, fallback mechanisms)
- **Architecture tests**: Comprehensive validation of new database-direct flow
- **Infrastructure**: CDK deployment successful, API Gateway + DynamoDB operational
- **End-to-end**: Complete pipeline validated from URL submission to real-time display

### Technical Notes
- AWS credentials: Check expiration first if issues, refresh token (project uses SSO profile)
- Scaling: Architecture designed for 50+ sources, no hardcoded limitations
- Geographic expansion: Infrastructure not limited to Seattle, ready for global scaling
- Zero hardcoded sources: All source management through admin interface

## Architecture Summary

### BEFORE (Legacy S3-based):
Hardcoded sources → EventBridge → Jina/OpenAI → S3 storage → Frontend

### AFTER (Database-direct):
Admin URL submission → Deduplication → FireCrawl extraction → DynamoDB storage → Admin approval → Database API → Frontend

### Key Benefits Achieved:
- **Real-time data**: No S3 upload delays
- **Dynamic sources**: All sources managed via admin interface
- **URL deduplication**: Prevents duplicate crawling
- **Auto-source creation**: Successful extractions become sources
- **Manual re-extraction**: On-demand source updates
- **Simplified pipeline**: Single API replaces complex multi-step processing
- **Cost efficiency**: Better extraction quality at lower operational complexity
- **Unlimited scaling**: No hardcoded source limitations