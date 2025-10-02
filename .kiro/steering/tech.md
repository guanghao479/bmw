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

## Common Commands

### Backend Development
```bash
# Navigate to backend
cd backend

# Install dependencies
go mod tidy

# Run unit tests
go test ./internal/models -v

# Run integration tests (requires FIRECRAWL_API_KEY)
./scripts/run_integration_tests.sh

# Build Lambda functions
go build -o ../testing/bin/admin_api ./cmd/admin_api
go build -o ../testing/bin/scraping_orchestrator ./cmd/scraping_orchestrator
```

### Infrastructure
```bash
# Navigate to infrastructure
cd infrastructure

# Install dependencies
npm install

# Deploy to AWS
npm run deploy

# View differences
npm run diff

# Destroy stack
npm run destroy
```

### Frontend Testing
```bash
# Serve locally (from app directory)
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