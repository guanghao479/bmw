#!/bin/bash

# Start local backend development server using AWS SAM CLI
# This script starts the backend Lambda functions as HTTP servers for local development

set -e

echo "🚀 Starting Local Backend Development Server"
echo "==========================================="

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Please run this script from the backend directory"
    exit 1
fi

# Check for required tools
if ! command -v sam &> /dev/null; then
    echo "❌ AWS SAM CLI not found. Please install it first:"
    echo "   https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html"
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ Go not found. Please install Go 1.22.5 or later"
    exit 1
fi

# Check environment variables
if [ -z "$FIRECRAWL_API_KEY" ] || [ -z "$OPENAI_API_KEY" ]; then
    echo "❌ Environment variables not set. Please ensure .envrc is loaded:"
    echo "   direnv allow"
    echo ""
    echo "Or set them manually:"
    echo "   export FIRECRAWL_API_KEY=your_key_here"
    echo "   export OPENAI_API_KEY=your_key_here"
    exit 1
fi

# Check AWS credentials
if ! aws sts get-caller-identity &> /dev/null; then
    echo "❌ AWS credentials not configured. Please run:"
    echo "   aws configure"
    echo "   or set AWS_PROFILE environment variable"
    exit 1
fi

# Build Lambda functions
echo "🔨 Building Lambda functions..."
mkdir -p ../testing/bin
go build -o ../testing/bin/admin_api ./cmd/admin_api
go build -o ../testing/bin/scraping_orchestrator ./cmd/scraping_orchestrator
echo "✅ Lambda functions built successfully"

# Create/update environment configuration
echo "📝 Creating environment configuration..."
cat > env.json << EOF
{
  "Parameters": {
    "LOCAL_DEV": "true",
    "FAMILY_ACTIVITIES_TABLE": "seattle-family-activities-dev",
    "SOURCE_MANAGEMENT_TABLE": "seattle-source-management-dev", 
    "SCRAPING_OPERATIONS_TABLE": "seattle-scraping-operations-dev",
    "ADMIN_EVENTS_TABLE": "seattle-admin-events-dev",
    "AWS_REGION": "us-west-2",
    "FIRECRAWL_API_KEY": "\${FIRECRAWL_API_KEY}",
    "OPENAI_API_KEY": "\${OPENAI_API_KEY}",
    "LOG_LEVEL": "DEBUG"
  }
}
EOF
echo "✅ Environment configuration created (using environment variable references)"

# Check CDK output exists
if [ ! -d "../infrastructure/cdk.out" ]; then
    echo "❌ CDK output not found. Please synthesize CDK stack first:"
    echo "   cd ../infrastructure"
    echo "   npm install"
    echo "   cdk synth"
    exit 1
fi

# Check if template file exists
TEMPLATE_FILE="../infrastructure/cdk.out/SeattleFamilyActivitiesMVPStack.template.json"
if [ ! -f "$TEMPLATE_FILE" ]; then
    echo "❌ CDK template not found at: $TEMPLATE_FILE"
    echo "Please run: cd ../infrastructure && cdk synth"
    exit 1
fi

echo "🚀 Starting SAM local API server..."
echo "📍 Backend will be available at: http://127.0.0.1:3000"
echo "📋 API endpoints:"
echo "   - GET  /api/sources"
echo "   - POST /api/sources"
echo "   - GET  /api/events"
echo "   - POST /api/scrape"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

# Start SAM local API
cd ../infrastructure
exec sam local start-api \
    --template-file cdk.out/SeattleFamilyActivitiesMVPStack.template.json \
    --env-vars ../backend/env.json \
    --port 3000 \
    --host 127.0.0.1