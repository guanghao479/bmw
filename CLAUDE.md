# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Process Plan & Execution [VERY IMPORTANT - FOLLOW STRICTLY]

For each task, go through plan phase first, then execute based on the plan. **NEVER skip any step.**

### Plan Phase [MANDATORY CHECKPOINTS]
1. **MUST use TodoWrite tool** to create initial task breakdown
2. **MUST research first** - If task requires external knowledge or packages, use Task tool for research BEFORE planning
3. **MUST create detailed plan** with implementation steps and reasoning
4. **MUST follow TDD approach** - Write down what to test and verify for each step (both automated tests and manual verification)
5. **MUST think MVP** - Don't over-plan, focus on minimal viable implementation
6. **MUST write plan document** - Create docs/tasks/TASK_NAME.md with the complete plan
7. **CHECKPOINT: STOP and ask for user approval** - Do not continue until user approves the plan

### Execution Phase [MANDATORY FOR EACH STEP]
For each implementation step:
1. **MUST mark todo as in_progress** before starting work
2. **MUST follow TDD approach** - Write tests first, then implement
3. **MUST test thoroughly** - Run all unit tests, integration tests, and manual verification
4. **MUST seek approval from user** - Get confirmation before proceeding
5. **MUST update documentation** - Once approved, update both:
   - docs/tasks/TASK_NAME.md - Append detailed descriptions of changes made
   - CLAUDE.md - Update if architecture, processes, or technical details changed
6. **MUST commit and push changes** - With descriptive commit messages
7. **MUST mark todo as completed** - Update todo status immediately after commit
8. **Then proceed to next task**

### After Implementation [FINAL CHECKPOINTS]
1. **MUST verify all todos marked complete** - Check TodoWrite tool shows all tasks done
2. **MUST ask user to review** final implementation
3. **MUST commit and push** final changes with proper commit message
4. **MUST update CLAUDE.md** if any new learnings or process improvements discovered

### Critical Reminders
- **NEVER skip updating documentation** - Both task doc and CLAUDE.md must be updated
- **NEVER proceed without testing** - All tests must pass before seeking approval  
- **NEVER forget TodoWrite tool** - Track progress throughout implementation
- **NEVER commit without user approval** - Always get confirmation first

## Project Overview

This is a Seattle family activities platform that automatically scrapes and curates family-friendly events, activities, and venues from 6+ local Seattle sources. The system consists of:

- **Production System**: AWS Lambda backend (Go) scrapes Seattle websites every 6 hours using Jina AI + OpenAI, stores data in S3, serves via GitHub Pages frontend
- **Demo/Development**: Static frontend in `/app/` directory can run independently with embedded sample data
- **Architecture**: Serverless, cost-effective (under $10/month), scalable to 10x traffic

**Live System**: 
- Frontend: https://guanghao479.github.io/bmw/
- Data API: S3-hosted JSON updated every 6 hours with real Seattle activities
- Sources: Seattle's Child, PEPS, ParentMap, Tinybeans, and more

## Architecture

### Production System Architecture
- **Frontend**: GitHub Pages static site (`/app/`) loading data from S3 via REST API
- **Backend**: AWS Lambda functions (Go runtime) in us-west-2 region
- **Data Pipeline**: Jina AI Reader → OpenAI GPT-4o mini → S3 JSON storage
- **Infrastructure**: EventBridge (6-hour scheduling), CloudWatch (monitoring), SNS (alerts)
- **Storage**: S3 bucket with public read access, CORS configured for GitHub Pages
- **Region**: us-west-2 (Oregon) for optimal performance and cost

### Frontend Architecture (Both Demo and Production)
- **Static single-page application**: No server-side rendering, pure client-side
- **Data source flexibility**: S3 REST API (production) or embedded JSON (demo/offline)
- **Class-based structure**: Main functionality in `FamilyEventsApp` class
- **Modern CSS**: CSS custom properties for theming and design system  
- **Responsive design**: Mobile-first with breakpoints at 480px, 768px, 1200px
- **Auto-refresh**: 30-minute data refresh cycle with manual refresh capability
- **Offline support**: LocalStorage caching with 24-hour expiration

## Key Components

### Data Sources (Production System)
**Seattle Family Activity Sources:**
1. **Seattle's Child** - https://www.seattleschild.com (events, activities)
2. **PEPS** - https://www.peps.org (parent groups, classes)  
3. **ParentMap** - https://www.parentmap.com (comprehensive family events)
4. **Tinybeans Seattle** - https://tinybeans.com/seattle/ (curated activities)
5. **Seattle Fun for Kids** - https://www.seattlefunforkids.com (family activities)
6. **Macaroni Kid West Seattle** - https://westseattle.macaronikid.com (local events)

### Data Structure & Schema
**Enhanced Backend Schema:** Activities include comprehensive metadata (schedule, pricing, registration, age groups, location details)
**Frontend Legacy Format:** Simplified schema for UI compatibility with three types: `events`, `activities`, `venues`
**Schema Conversion:** Backend→Frontend format conversion via `convertToLegacyFormat()` function

### Backend Services (Go/AWS)
- **Jina Service** (`internal/services/jina.go`): Website content extraction with retry logic
- **OpenAI Service** (`internal/services/openai.go`): Activity data extraction using GPT-4o mini
- **S3 Service** (`internal/services/s3.go`): JSON storage, backup management, public URL generation
- **Lambda Orchestrator** (`cmd/lambda/main.go`): Coordinates scraping pipeline with rate limiting

### Frontend Architecture (`app/script.js`)
- **FamilyEventsApp class**: Handles all UI interactions and data management
- **Data loading**: Fetch from S3 → Cache in localStorage → Fallback to embedded data
- **Real-time filtering**: Category, search term, and age group filtering
- **Modal system**: Detailed item views with enhanced metadata
- **Auto-refresh system**: Background data updates with user notifications

### CSS Design System (`app/styles.css`)
- **CSS custom properties**: Colors, spacing, typography, transitions
- **Component-based classes**: `.card`, `.filter-btn`, `.modal`, responsive grid
- **Glassmorphism effects**: Modern shadows, blur effects, accessibility focus states
- **Mobile-first responsive**: Breakpoints optimized for family browsing patterns

## Development

### Frontend Development
**Running Locally:**
- Open `app/index.html` in browser (no build process required)
- For S3 integration testing, serve via local server (e.g., `python -m http.server 8000`)

**Configuration:**
- Environment detection: Automatic dev/prod mode based on hostname
- S3 endpoints: Configured in `loadConfiguration()` method
- Debug mode: Available in development for detailed logging

### Backend Development (Go + AWS)
**Prerequisites:**
- Go 1.21+, AWS CLI configured, AWS CDK CLI
- Environment variables: `AWS_PROFILE`, `JINA_API_KEY`, `OPENAI_API_KEY`

**Local Testing:**
```bash
cd backend
go test ./...                              # Run all unit tests
./scripts/run_integration_tests.sh        # Run integration tests
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
- Data loading: Verify S3 fetch, localStorage caching, and fallback behavior

**Backend Testing:**
- `go test ./internal/models/` - Data model validation
- `go test ./internal/services/` - Service integration tests  
- `go test ./cmd/lambda/` - Lambda orchestration tests
- Integration: Tests real APIs (Jina, OpenAI, S3) with Seattle sources

### Making Changes
**Frontend Updates:**
- **Styling**: Update CSS custom properties in `app/styles.css` for global changes
- **New features**: Extend `FamilyEventsApp` class methods in `app/script.js`
- **Data schema**: Modify `convertToLegacyFormat()` for new backend fields

**Backend Updates:**
- **New sources**: Add to source configuration in `cmd/lambda/main.go`
- **Data models**: Update `internal/models/activity.go` and regenerate tests
- **Services**: Extend Jina, OpenAI, or S3 services in `internal/services/`

**Infrastructure Updates:**
- **AWS resources**: Modify `infrastructure/lib/mvp-stack.ts`
- **Deployment**: Update GitHub Actions workflows in `.github/workflows/`

## Technical Notes

### Frontend
- **No build tools**: Intentional simplicity - pure HTML/CSS/JS, no package.json required
- **Data flexibility**: Can load from S3 (production) or embedded JSON (demo/offline)
- **Dynamic UI**: Modal styles injected when needed, intersection observer for animations
- **Modern JavaScript**: async/await, arrow functions, template literals, fetch API
- **Image handling**: Unsplash URLs with SVG placeholder fallback, lazy loading

### Backend  
- **Go modules**: Clean dependency management, AWS SDK v2, comprehensive testing
- **Error handling**: Graceful degradation, retry logic, detailed CloudWatch logging
- **Cost optimization**: Content truncation, token usage tracking, efficient S3 operations
- **Security**: No secrets in code, AWS IAM roles, CORS properly configured

## AWS Infrastructure

### Services Used
- **Lambda**: Go runtime, 15-minute timeout, 1GB memory, us-west-2 region
- **S3**: Public bucket with CORS, versioned backups, lifecycle policies
- **EventBridge**: 6-hour scheduling with timezone handling
- **CloudWatch**: Error monitoring, performance metrics, log aggregation
- **SNS**: Email alerts for Lambda failures and data quality issues
- **IAM**: Least-privilege roles for Lambda S3 access

### Cost Structure (Monthly)
- **Lambda**: ~$2-3 (execution time, memory usage)
- **S3**: ~$1-2 (storage, requests, data transfer)
- **CloudWatch**: ~$1 (logs, metrics, alarms)
- **API calls**: ~$2-3 (Jina AI, OpenAI tokens)
- **Total**: Under $10/month at current scale

### Performance Metrics
- **Scraping**: 15-25 seconds per source, 6 sources in ~100 seconds total
- **Data processing**: 50% cost reduction via content optimization
- **Frontend load**: Sub-3 second initial load, 30-minute refresh cycle
- **Reliability**: 99%+ uptime via S3, graceful error handling

## Data Pipeline

### Scraping Process
1. **EventBridge trigger** every 6 hours (6 AM, 12 PM, 6 PM, 12 AM PST)
2. **Lambda orchestration** with rate limiting (max 3 concurrent)
3. **Jina AI extraction** from Seattle source websites
4. **OpenAI processing** to standardize data format and extract activities
5. **S3 storage** with latest.json and timestamped backups
6. **Frontend auto-refresh** detects new data and updates UI

### Data Quality
- **Deduplication**: Title + location + date similarity matching
- **Validation**: Schema validation, required field checking
- **Monitoring**: Success rates, extraction quality, source reliability
- **Fallback**: Multiple sources for redundancy, cached data for outages

### Source Management
- **Priority-based processing**: PEPS, Seattle's Child, ParentMap prioritized
- **Reliability scoring**: Track success rates per source
- **Content optimization**: 17k→5k character truncation for cost efficiency
- **Error handling**: Individual source failures don't break entire pipeline

## Deployment

### GitHub Actions Workflows
- **Main deployment** (`.github/workflows/deploy.yml`): CDK + Lambda + GitHub Pages
- **Scraper testing** (`.github/workflows/test-scraper.yml`): Manual trigger for testing
- **Automated testing**: CORS validation, data quality checks, accessibility audits
- **Performance monitoring**: Lighthouse integration with quality thresholds
- **Security**: OIDC authentication with AWS (no long-lived access keys)

### Infrastructure as Code
- **AWS CDK**: TypeScript stack definition in `infrastructure/lib/mvp-stack.ts`
- **Deployment**: Automated via GitHub Actions on main branch push
- **Environment management**: Dev/prod configuration via GitHub secrets and OIDC
- **Authentication**: OpenID Connect (OIDC) for secure AWS access without access keys
- **Rollback**: CDK stack versioning and automated rollback capabilities
- **Setup guide**: See `docs/aws-oidc-setup.md` for OIDC configuration instructions

## Monitoring & Alerts

### CloudWatch Integration
- **Lambda metrics**: Duration, errors, cost tracking, memory usage
- **Data quality**: Activity count, source success rates, extraction accuracy
- **Frontend metrics**: Page load times, error rates, user engagement
- **Cost monitoring**: API usage, AWS service costs, budget alerts

### Error Handling
- **Lambda failures**: SNS email alerts, automatic retries, graceful degradation
- **API quota limits**: Exponential backoff, alternative source fallback
- **S3 outages**: Frontend cache utilization, embedded data fallback
- **Data corruption**: Validation checks, backup restoration, manual intervention

## Completed Tasks

### MVP Implementation Status
- ✅ **Phase 1-2**: Infrastructure, backend scraping system (24 hours actual vs 12 estimated)
- ✅ **Phase 3-4**: Frontend enhancement, deployment pipeline (completed August 15, 2025)
- ✅ **Production ready**: Live at https://guanghao479.github.io/bmw/ with 9+ Seattle activities
- 📝 **Documentation**: See `docs/tasks/seattle-family-activities-mvp.md` for detailed implementation notes

### System Validation
- **Backend tests**: 15/15 passing (unit + integration with real APIs)
- **Frontend tests**: 7/7 passing (S3 integration, mobile responsiveness, offline support)
- **Infrastructure**: CDK deployment successful, all AWS services operational
- **End-to-end**: Complete data pipeline validated from scraping to frontend display