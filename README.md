# Seattle Family Activities Platform

A serverless web application that aggregates and displays family-friendly activities, events, and venues in the Seattle metro area.

## Quick Start

### Local Development

1. **Setup local development environment**:
   ```bash
   # Install dependencies and configure environment
   # See docs/LOCAL_DEVELOPMENT_SETUP.md for detailed instructions
   ```

2. **Start backend and frontend**:
   ```bash
   # Start backend (AWS SAM CLI)
   cd backend
   ./start-local-backend.sh
   
   # Start frontend (in another terminal)
   ./start-local-frontend.sh
   ```

3. **Access the application**:
   - Main app: http://localhost:8000
   - Admin interface: http://localhost:8000/admin.html
   - Backend API: http://127.0.0.1:3000/api

## Documentation

- **[Local Development Setup](docs/LOCAL_DEVELOPMENT_SETUP.md)** - Complete guide to setting up local development
- **[Troubleshooting Guide](docs/TROUBLESHOOTING.md)** - Solutions for common development issues
- **[Browser Testing Procedures](docs/BROWSER_TESTING.md)** - Comprehensive browser testing guide

## Architecture

- **Frontend**: Vanilla JavaScript with Tailwind CSS (GitHub Pages)
- **Backend**: Go Lambda functions with DynamoDB (AWS)
- **Infrastructure**: AWS CDK with TypeScript
- **Local Development**: AWS SAM CLI for local Lambda execution

## Project Structure

```
├── app/                    # Frontend application
├── backend/                # Go backend services  
├── infrastructure/         # AWS CDK infrastructure
├── docs/                   # Documentation
└── .kiro/                  # Kiro configuration
```

## Development Workflow

1. **Make changes** to frontend (app/) or backend (backend/)
2. **Test locally** using the local development environment
3. **Run tests**: `cd backend && go test ./...`
4. **Commit and push** to trigger automated deployment

## Getting Started

For detailed setup instructions, see [Local Development Setup](docs/LOCAL_DEVELOPMENT_SETUP.md).