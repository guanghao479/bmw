# Technology Stack

## Backend

- **Language**: Go 1.22.5+ with toolchain 1.24.2
- **Runtime**: AWS Lambda with Go runtime
- **Database**: DynamoDB with GSI indexes for querying
- **Storage**: S3 for backups and static assets
- **API**: API Gateway with CORS for admin interface
- **External APIs**: FireCrawl for web scraping, OpenAI for content extraction

## Frontend

- **Framework**: Vanilla JavaScript (no frameworks)
- **Styling**: Tailwind CSS via CDN for utility-first styling and mobile-first responsive design
- **Hosting**: GitHub Pages with automatic deployment
- **Data Loading**: Fetch API with S3/API Gateway endpoints

## Infrastructure

- **IaC**: AWS CDK with TypeScript
- **Deployment**: GitHub Actions with OIDC authentication
- **Monitoring**: CloudWatch with SNS alerts
- **Region**: us-west-2 (Oregon)

## Development Tools

- **Environment**: direnv for environment variables
- **Testing**: Go testing framework with integration tests
- **Build**: Native Go build tools, CDK CLI
- **Local Development**: Makefile targets for starting local servers

## Local Development

### Makefile Targets (Recommended)

Use these Makefile targets for local development and testing:

```bash
# Start both frontend and backend servers simultaneously
make dev

# Start only the backend server (for API testing)
make dev-backend

# Start only the frontend server (for UI development)
make dev-frontend

# Build Lambda functions for testing
make build

# Run backend unit tests
make test

# Run integration tests (requires API keys)
make test-integration

# Deploy infrastructure
make infra-deploy

# Show infrastructure deployment diff
make infra-diff

# Clean build artifacts
make clean
```

### Development Server Details

- **Frontend Server**: Serves the `app/` directory on a local port
- **Backend Server**: Runs a local development server for API testing
- **Combined Mode**: `make dev` starts both servers for full-stack development

## Common Commands

### Backend Development
```bash
# Recommended: Use Makefile targets
make dev-backend          # Start local backend server
make build               # Build Lambda functions
make test                # Run unit tests
make test-integration    # Run integration tests

# Alternative: Manual commands
cd backend
go mod tidy              # Install dependencies
go test ./internal/models -v
./scripts/run_integration_tests.sh
go build -o ../testing/bin/admin_api ./cmd/admin_api
go build -o ../testing/bin/scraping_orchestrator ./cmd/scraping_orchestrator
```

### Infrastructure
```bash
# Recommended: Use Makefile targets
make infra-deploy        # Deploy to AWS
make infra-diff         # View deployment differences

# Alternative: Manual commands
cd infrastructure
npm install             # Install dependencies
npm run deploy          # Deploy to AWS
npm run diff           # View differences
npm run destroy        # Destroy stack
```

### Frontend Development
```bash
# Recommended: Use Makefile target
make dev-frontend

# Alternative: Manual server start (from app directory)
python -m http.server 8000
# or
npx serve .
```

### Tailwind CSS Usage
- **CDN Integration**: Include Tailwind via CDN in HTML head
- **Utility Classes**: Use utility-first approach (e.g., `bg-blue-500`, `text-center`, `md:flex`)
- **Responsive Design**: Mobile-first with `sm:`, `md:`, `lg:`, `xl:` prefixes
- **Custom Components**: Combine utilities for reusable component patterns

## Key Dependencies

### Go Modules
- `github.com/aws/aws-lambda-go` - Lambda runtime
- `github.com/aws/aws-sdk-go-v2` - AWS services
- `github.com/mendableai/firecrawl-go` - Web scraping
- `github.com/google/uuid` - ID generation

### CDK Stack
- `aws-cdk-lib` - Infrastructure constructs
- `@aws-cdk/aws-lambda-go-alpha` - Go Lambda support
- `constructs` - CDK constructs library