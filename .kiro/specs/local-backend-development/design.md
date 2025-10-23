# Design Document

## Overview

This design enables local development of the Seattle Family Activities platform by creating HTTP server wrappers around the existing Lambda functions and updating the frontend to connect to local backend services. The solution maintains the existing Lambda code while adding a local development mode that runs the same business logic as HTTP servers.

## Architecture

### High-Level Architecture

```
┌─────────────────┐    HTTP     ┌─────────────────┐    AWS SDK    ┌─────────────────┐
│   Frontend      │────────────▶│   AWS SAM CLI   │──────────────▶│   Remote AWS    │
│   (localhost)   │             │  Local API GW   │               │   Services      │
└─────────────────┘             └─────────────────┘               └─────────────────┘
                                         │                                 │
                                         │                                 │
                                    ┌────▼────┐                      ┌────▼────┐
                                    │ Lambda  │                      │DynamoDB │
                                    │Functions│                      │   S3    │
                                    │(Local)  │                      │FireCrawl│
                                    └─────────┘                      └─────────┘
```

### Local Development Flow

1. **AWS SAM CLI**: Uses `sam local start-api` to run Lambda functions locally with API Gateway simulation
2. **SAM Template**: Defines API Gateway routes and Lambda function mappings
3. **Frontend Detection**: Automatically detects local environment and uses SAM's localhost URLs
4. **AWS Services**: Uses remote DynamoDB and S3 with local AWS credentials
5. **External APIs**: Uses real FireCrawl and OpenAI APIs when keys are available

## Components and Interfaces

### 1. CDK + SAM Integration (Zero Duplication)

**Purpose**: Use existing CDK stack for local development via SAM CLI - no separate templates needed

**Implementation**:
- **No separate SAM template required** - SAM CLI reads CDK-generated CloudFormation
- Use `cdk synth` to generate CloudFormation template from existing CDK stack
- Use `sam local start-api` with the synthesized template for local development
- **Zero duplication** - CDK remains single source of truth for all infrastructure

**Workflow**:
```bash
# 1. Synthesize CDK stack to CloudFormation
cd infrastructure
cdk synth

# 2. Use SAM CLI with CDK-generated template
cd ../backend
sam local start-api -t ../infrastructure/cdk.out/SeattleFamilyActivitiesMVPStack.template.json
```

**Benefits**:
- **No duplication**: Single CDK stack defines everything
- **Official AWS pattern**: CDK documentation recommends this approach
- **Automatic sync**: Any CDK changes automatically available to SAM after `cdk synth`
- **Same infrastructure**: Local development uses identical infrastructure definition as production

### 2. Local Development Configuration

**Purpose**: Configure SAM CLI to use development environment variables and local services

**Implementation**:
- Create `env.json` for local environment variable overrides
- Configure development table names to avoid production conflicts
- Set up local file storage paths
- Configure external API keys and endpoints

**Configuration Files**:

`backend/env.json` (overrides CDK environment variables for local development):
```json
{
  "Parameters": {
    "LOCAL_DEV": "true",
    "FAMILY_ACTIVITIES_TABLE": "seattle-family-activities-dev",
    "SOURCE_MANAGEMENT_TABLE": "seattle-source-management-dev", 
    "SCRAPING_OPERATIONS_TABLE": "seattle-scraping-operations-dev",
    "ADMIN_EVENTS_TABLE": "seattle-admin-events-dev",
    "AWS_REGION": "us-west-2",
    "FIRECRAWL_API_KEY": "your_key_here",
    "OPENAI_API_KEY": "your_key_here"
  }
}
```

**SAM CLI Command**:
```bash
# Use CDK-generated template with local environment overrides
sam local start-api \
  -t ../infrastructure/cdk.out/SeattleFamilyActivitiesMVPStack.template.json \
  --env-vars env.json \
  --port 3000
```

### 3. Frontend Environment Detection

**Purpose**: Automatically connect to SAM local API when running locally

**Implementation**:
- Update `loadConfiguration()` in `app/script.js` and `app/admin.js`
- Detect localhost environment and use SAM's default local API URLs
- Provide fallback and error handling for unavailable local backend
- Maintain same API interface regardless of backend location

**Updated Configuration**:
```javascript
loadConfiguration() {
    const isLocal = window.location.hostname === 'localhost' || 
                   window.location.hostname === '127.0.0.1';
    
    if (isLocal) {
        return {
            // SAM CLI default local API Gateway endpoint
            apiEndpoint: 'http://127.0.0.1:3000/api',
            adminApiEndpoint: 'http://127.0.0.1:3000/api',
            environment: 'local',
            samLocal: true
        };
    }
    // ... production config
}
```

### 4. S3 Code Cleanup

**Current State**: S3 infrastructure and code exists but is no longer part of main data flow

**Cleanup Approach**:
- **Remove S3 bucket and related IAM permissions** from CDK stack since it's not used
- **Remove S3Client and related services** from backend code
- **Remove S3-related environment variables** from Lambda function configuration
- **Clean up imports and dependencies** related to S3 services
- **Verify no S3 operations** are called in current data flow paths

**Benefits**:
- Simplified infrastructure with fewer unused resources
- Cleaner codebase without deprecated functionality
- Reduced AWS costs by removing unused S3 bucket
- Faster local development without S3 dependencies
- Clearer architecture focused on DynamoDB → API → Frontend flow

## Data Models

### SAM Local Request/Response Flow

**SAM CLI Handles Conversion**:
- SAM CLI automatically converts HTTP requests to Lambda events
- No custom conversion code needed - uses AWS standard patterns
- Maintains exact same event structure as production API Gateway
- Handles CORS, headers, and routing automatically

### Local Development Data Flow

```
HTTP Request → SAM CLI → Lambda Event → Business Logic → Lambda Response → SAM CLI → HTTP Response
     ↓            ↓           ↓              ↓              ↓            ↓           ↓
[Frontend]   [Auto Convert] [Standard]  [Existing Code] [Standard]  [Auto Convert] [Frontend]
```

### Environment Variable Mapping

**Local Development Variables**:
```bash
# Local development flag
LOCAL_DEV=true

# DynamoDB tables (using dev environment)
FAMILY_ACTIVITIES_TABLE=seattle-family-activities-dev
SOURCE_MANAGEMENT_TABLE=seattle-source-management-dev
SCRAPING_OPERATIONS_TABLE=seattle-scraping-operations-dev
ADMIN_EVENTS_TABLE=seattle-admin-events-dev

# AWS configuration
AWS_REGION=us-west-2
AWS_PROFILE=default

# External APIs
FIRECRAWL_API_KEY=your_key_here
OPENAI_API_KEY=your_key_here

# S3 not needed - main data flow is DynamoDB -> API -> Frontend
```

## Error Handling

### Local Backend Unavailable
- Frontend detects connection failures to local backend
- Provides clear error messages to developer
- Suggests starting local backend server
- Does not fall back to production to avoid confusion

### AWS Credential Issues
- Detect missing or invalid AWS credentials
- Provide clear setup instructions
- Support multiple credential methods (profile, environment variables, IAM roles)
- Graceful degradation for optional services

### Port Conflicts
- Detect if configured port is already in use
- Suggest alternative ports
- Allow port configuration via environment variables
- Provide clear error messages for network issues

## Testing Strategy

### Unit Testing
- Test CDK synthesis and SAM CLI integration
- Test environment variable overrides for local development
- Test S3 code removal doesn't break functionality
- Test DynamoDB connection with local credentials

### Integration Testing
- Test complete local development workflow (CDK synth → SAM local start-api)
- Test frontend connection to SAM local backend
- Test AWS service integration (DynamoDB) with local backend
- Test error scenarios and fallback behavior
- Test external API integration (FireCrawl, OpenAI) in local mode

### Browser Testing
- **Test main frontend** (`app/index.html`) connects to local backend and loads data
- **Test admin interface** (`app/admin.html`) connects to local backend for all functionality
- **Test all API endpoints** work correctly through browser (sources, events, analytics)
- **Test CORS configuration** allows local frontend to connect to SAM local API
- **Test error handling** in browser when local backend is unavailable
- **Test data flow** from admin interface through to main frontend display

### End-to-End Workflow Testing
- Start local backend with SAM CLI
- Start local frontend with simple HTTP server
- Submit new source through admin interface
- Verify data appears in DynamoDB
- Verify approved events appear in main frontend
- Test complete user workflow in browser environment

## Implementation Phases

### Phase 1: CDK + SAM Integration Setup
1. Create local environment configuration (env.json) for development overrides
2. Test CDK synthesis and SAM CLI integration workflow
3. Verify SAM can read CDK-generated CloudFormation template
4. Test basic `sam local start-api` functionality with CDK template

### Phase 2: S3 Code Cleanup
1. Remove S3 bucket and related resources from CDK stack
2. Remove S3Client service and related code from backend
3. Remove S3-related environment variables from Lambda configuration
4. Clean up imports and dependencies related to S3
5. Verify all functionality works without S3 dependencies

### Phase 3: Frontend Integration
1. Update frontend environment detection for SAM local endpoints
2. Modify API endpoint configuration in both main and admin apps
3. Test complete frontend-backend communication
4. Handle connection errors and provide developer feedback

### Phase 4: Browser Testing and Developer Experience
1. Test complete workflow in browser environment (both main and admin frontends)
2. Verify all API endpoints work correctly through browser requests
3. Test error handling and user experience when backend is unavailable
4. Create startup scripts and development commands
5. Add comprehensive setup documentation with browser testing steps
6. Update development workflow documentation to reflect real backend usage
7. Create troubleshooting guide for common SAM local and browser connectivity issues

## Security Considerations

### Local Development Security
- SAM CLI binds to localhost by default (configurable)
- CORS handled automatically by SAM CLI API Gateway simulation
- No authentication required for local development
- Clear separation between local and production environments using different table names

### Infrastructure Separation
- **CDK as Single Source**: CDK stack defines all infrastructure for both local and production
- **SAM for Local Runtime**: SAM CLI provides local Lambda/API Gateway runtime using CDK template
- **No Duplication**: SAM reads CDK-generated CloudFormation - no separate template needed
- **Environment Isolation**: Local development uses different table names via environment variable overrides

### AWS Credentials
- Use least-privilege AWS credentials for local development
- Support IAM roles and profiles for DynamoDB access
- Never commit credentials to version control
- Provide clear credential setup documentation

### Data Isolation
- Use separate DynamoDB table names for local development (e.g., `-dev` suffix)
- Prevent accidental production data modification
- Clear data cleanup procedures
- No S3 dependencies for local development after cleanup