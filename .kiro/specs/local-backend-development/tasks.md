# Implementation Plan

- [x] 1. Set up CDK + SAM integration for local development
  - Create local environment configuration file (env.json) with development table names and LOCAL_DEV flag
  - Test CDK synthesis workflow to generate CloudFormation template
  - Verify SAM CLI can read CDK-generated template and start local API
  - Test basic SAM local start-api functionality with CDK template
  - _Requirements: 1.1, 1.2, 1.3, 4.1_

- [x] 2. Remove deprecated S3 infrastructure and code
  - Remove S3 bucket and related resources from CDK stack (mvp-stack.ts)
  - Remove S3-related IAM permissions and policies from Lambda roles
  - Remove S3-related environment variables from Lambda function configuration
  - Remove S3Client service code and related imports from backend
  - Update CDK outputs to remove S3-related exports
  - _Requirements: 6.1, 6.2, 6.3, 6.4_

- [x] 3. Configure frontend for local backend connection
  - Update loadConfiguration() in app/script.js to detect local environment and use SAM local endpoints
  - Update detectEnvironment() in app/admin.js to use SAM local API URLs
  - Configure CORS handling for local development in frontend
  - Add error handling and user feedback when local backend is unavailable
  - Test frontend environment detection logic
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [x] 4. Test complete local development workflow
  - Test CDK synthesis and SAM CLI integration end-to-end
  - Verify DynamoDB connection works with local AWS credentials
  - Test external API integration (FireCrawl, OpenAI) in local mode
  - Verify all Lambda functions start correctly with SAM local
  - Test environment variable overrides work correctly
  - _Requirements: 1.4, 5.1, 5.2, 5.3_

- [x] 5. Browser testing and validation
  - Start local backend with SAM CLI and verify API endpoints respond
  - Start local frontend and test connection to SAM local backend
  - Test main frontend (app/index.html) loads data from local backend
  - Test admin interface (app/admin.html) functionality with local backend
  - Test all API endpoints work correctly through browser requests
  - Verify CORS configuration allows frontend-backend communication
  - Test error handling when local backend is unavailable
  - _Requirements: 2.4, 4.2, 4.3_

- [x] 6. Create development documentation and scripts
  - Create startup script for local backend development (start-local-backend.sh)
  - Create startup script for local frontend development (start-local-frontend.sh)
  - Document complete local development setup process
  - Create troubleshooting guide for common SAM local issues
  - Document browser testing procedures
  - _Requirements: 4.1, 4.2, 4.4, 4.5_

- [x] 7. Deprecate mock API server
  - Remove mock-api-server.js file
  - Update any documentation references to mock API server
  - Update frontend configuration to remove mock API server endpoints
  - Verify no remaining references to mock API server in codebase
  - Update development workflow documentation
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

- [x] 8. End-to-end workflow testing
  - Test complete user workflow: admin submits source → data in DynamoDB → appears in main frontend ✅
  - Verify data persistence across SAM local restarts ✅
  - Test multiple concurrent browser sessions with local backend ✅
  - Use browser mcp tool ✅
  - _Requirements: 2.1, 2.2, 2.3, 5.4_