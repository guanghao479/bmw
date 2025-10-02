# Project Structure

## Root Directory Layout

```
├── app/                    # Frontend application (GitHub Pages)
├── backend/                # Go backend services
├── infrastructure/         # AWS CDK infrastructure code
├── docs/                   # Documentation and task specifications
├── testing/                # All testing artifacts (gitignored)
└── .kiro/                  # Kiro configuration and steering
```

## Frontend (`app/`)

- `index.html` - Main application page
- `admin.html` - Admin interface for source management
- `script.js` - Main application JavaScript
- `admin.js` - Admin interface JavaScript
- `styles.css` - Application styles (mobile-first)

## Backend (`backend/`)

### Command Structure (`cmd/`)
- `admin_api/` - Admin API Lambda function
- `scraping_orchestrator/` - Main scraping Lambda function
- `test_*/` - Various testing utilities

### Internal Packages (`internal/`)
- `models/` - Data models and validation
- `services/` - External service integrations (DynamoDB, S3, FireCrawl)

### Key Files
- `go.mod` - Go module definition
- `scripts/run_integration_tests.sh` - Integration test runner

## Infrastructure (`infrastructure/`)

- `lib/mvp-stack.ts` - Main CDK stack definition
- `bin/` - CDK app entry point
- `cdk.json` - CDK configuration
- `package.json` - Node.js dependencies

## Documentation (`docs/`)

- `tasks/` - Detailed implementation specifications
- `GITHUB_PAGES_SETUP.md` - Deployment guide
- `aws-oidc-setup.md` - AWS authentication setup

## Testing (`testing/`)

**Note**: This directory is gitignored and contains all testing artifacts.

- `temp/` - Temporary development files
- `build/` - Compiled binaries
- `data/` - Test responses and sample data
- `bin/` - Executable binaries
- `logs/` - Debug logs and test output

## Code Organization Patterns

### Go Package Structure
- Use `internal/` for private packages
- Separate models from services
- Each Lambda function has its own `cmd/` directory
- Comprehensive test coverage with `*_test.go` files

### DynamoDB Table Design
- 4 main tables: family-activities, source-management, scraping-operations, admin-events
- GSI indexes for efficient querying
- Composite keys (PK/SK) for hierarchical data

### API Structure
- RESTful endpoints under `/api/`
- CORS enabled for GitHub Pages origin
- Consistent JSON response format with success/error fields

### Frontend Architecture
- No build process - vanilla JS/CSS
- Environment-aware configuration
- Offline support with localStorage caching
- Mobile-first responsive design

## File Naming Conventions

- **Go files**: snake_case (e.g., `admin_api.go`)
- **JavaScript**: camelCase (e.g., `loadConfiguration()`)
- **CSS classes**: kebab-case (e.g., `.search-container`)
- **Environment variables**: UPPER_SNAKE_CASE
- **DynamoDB tables**: kebab-case with region suffix

## Development Workflow

1. **Backend changes**: Modify Go code → Run tests → Deploy via CDK
2. **Frontend changes**: Edit HTML/JS/CSS → Push to main → Auto-deploy to GitHub Pages
3. **Infrastructure changes**: Update CDK stack → Deploy via GitHub Actions
4. **Testing**: Use `testing/` directory for all artifacts and temporary files