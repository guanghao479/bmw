# Requirements Document

## Introduction

This feature enables local development of the Seattle Family Activities platform by allowing the backend Go services to run locally as HTTP servers instead of Lambda functions, and configuring the frontend to connect to the local backend. This will replace the current mock API server and provide a complete local development environment.

## Glossary

- **Backend Service**: The Go-based admin API and scraping orchestrator currently deployed as AWS Lambda functions
- **Frontend Application**: The vanilla JavaScript web application served from the app/ directory
- **Mock API Server**: The current Node.js server (mock-api-server.js) that provides fake API responses for development
- **Local Development Environment**: A setup where both backend and frontend run on the developer's local machine
- **HTTP Server Mode**: Running the Go Lambda functions as standard HTTP servers instead of Lambda handlers
- **Environment Detection**: Frontend logic that determines whether to use local or production API endpoints

## Requirements

### Requirement 1

**User Story:** As a developer, I want to run the backend services locally as HTTP servers, so that I can develop and test backend functionality without deploying to AWS.

#### Acceptance Criteria

1. WHEN a developer runs a local development command, THE Backend Service SHALL start as an HTTP server on localhost
2. THE Backend Service SHALL expose all existing API endpoints through HTTP instead of Lambda
3. THE Backend Service SHALL maintain the same request/response format as the Lambda version
4. THE Backend Service SHALL support CORS headers for local frontend development
5. THE Backend Service SHALL use local environment variables for configuration instead of AWS-specific settings

### Requirement 2

**User Story:** As a developer, I want the frontend to automatically connect to the local backend when running locally, so that I can test the complete application stack.

#### Acceptance Criteria

1. WHEN the Frontend Application detects a local development environment, THE Frontend Application SHALL connect to the local Backend Service
2. THE Frontend Application SHALL use localhost URLs for API calls in development mode
3. THE Frontend Application SHALL maintain the same API interface regardless of backend location
4. THE Frontend Application SHALL provide clear feedback when the local backend is unavailable
5. THE Frontend Application SHALL fall back gracefully if the local backend is not running

### Requirement 3

**User Story:** As a developer, I want to deprecate the mock API server, so that I can use real backend logic during development instead of fake responses.

#### Acceptance Criteria

1. THE Mock API Server SHALL be marked as deprecated in documentation
2. THE Mock API Server SHALL be removed from the development workflow
3. THE Backend Service SHALL provide all endpoints currently served by the Mock API Server
4. THE Frontend Application SHALL no longer reference the Mock API Server endpoints
5. THE development documentation SHALL be updated to use the local Backend Service

### Requirement 4

**User Story:** As a developer, I want simple commands to start the local development environment, so that I can quickly begin development work.

#### Acceptance Criteria

1. THE Local Development Environment SHALL provide a single command to start the backend
2. THE Local Development Environment SHALL provide a single command to start the frontend
3. THE Local Development Environment SHALL include documentation for the development setup process
4. THE Local Development Environment SHALL validate required dependencies before starting
5. THE Local Development Environment SHALL provide clear error messages for setup issues

### Requirement 5

**User Story:** As a developer, I want the local backend to use appropriate local services where practical, so that I can develop with minimal AWS dependencies.

#### Acceptance Criteria

1. THE Backend Service SHALL connect to remote DynamoDB tables for data operations when running locally
2. THE Backend Service SHALL use real external API calls (FireCrawl, OpenAI) when API keys are available
3. THE Backend Service SHALL provide configuration options for local vs production service usage
4. THE Backend Service SHALL use AWS credentials from the local development environment for DynamoDB access
5. THE Backend Service SHALL not require S3 services for local development functionality

### Requirement 6

**User Story:** As a developer, I want to remove deprecated S3 code paths, so that the codebase is clean and only contains actively used functionality.

#### Acceptance Criteria

1. THE Backend Service SHALL remove unused S3 service code that is no longer part of the main data flow
2. THE Backend Service SHALL remove S3-related environment variables and configuration from Lambda functions
3. THE Infrastructure SHALL remove S3 bucket and related IAM permissions that are no longer needed
4. THE Backend Service SHALL remove S3 client initialization and related dependencies where not needed
5. THE Backend Service SHALL maintain only essential S3 functionality if any backup features are still required