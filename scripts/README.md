# Development Scripts

This directory contains all development scripts for the Seattle Family Activities platform.

## Available Scripts

- **`dev`** - Start both backend and frontend servers for full development environment
- **`dev-backend`** - Start only the backend API server (SAM local)
- **`dev-frontend`** - Start only the frontend server (Python HTTP server)

## Usage

### Quick Start (Recommended)
```bash
# From project root
make dev
```

### Individual Services
```bash
# Backend only
make dev-backend

# Frontend only  
make dev-frontend
```

### Direct Script Usage
```bash
# From project root
./scripts/dev
./scripts/dev-backend
./scripts/dev-frontend
```

## Prerequisites

- Go 1.22.5+
- AWS SAM CLI
- Python 3
- AWS credentials configured
- Environment variables set (FIRECRAWL_API_KEY, OPENAI_API_KEY)

## Ports

- **Backend**: http://127.0.0.1:3000
- **Frontend**: http://localhost:8000

## Notes

- All scripts are designed to be run from the project root
- Scripts use relative paths and will navigate to appropriate directories
- Environment variables are validated before starting services
- CDK stack must be synthesized before running backend (`cd infrastructure && cdk synth`)