#!/bin/bash

# Start local backend and frontend servers for browser testing

set -e

echo "🚀 Starting Local Development Servers"
echo "===================================="

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Please run this script from the backend directory"
    exit 1
fi

# Check environment variables
if [ -z "$FIRECRAWL_API_KEY" ] || [ -z "$OPENAI_API_KEY" ]; then
    echo "❌ Environment variables not set. Please ensure .envrc is loaded: direnv allow"
    exit 1
fi

# Build Lambda functions
echo "🔨 Building Lambda functions..."
mkdir -p ../testing/bin
go build -o ../testing/bin/admin_api ./cmd/admin_api
go build -o ../testing/bin/scraping_orchestrator ./cmd/scraping_orchestrator
echo "✅ Lambda functions built"

# Update env.json with current environment variables
echo "📝 Updating environment configuration..."
cat > env.json << EOF
{
  "Parameters": {
    "LOCAL_DEV": "true",
    "FAMILY_ACTIVITIES_TABLE": "seattle-family-activities-dev",
    "SOURCE_MANAGEMENT_TABLE": "seattle-source-management-dev", 
    "SCRAPING_OPERATIONS_TABLE": "seattle-scraping-operations-dev",
    "ADMIN_EVENTS_TABLE": "seattle-admin-events-dev",
    "AWS_REGION": "us-west-2",
    "FIRECRAWL_API_KEY": "$FIRECRAWL_API_KEY",
    "OPENAI_API_KEY": "$OPENAI_API_KEY",
    "LOG_LEVEL": "DEBUG"
  }
}
EOF

echo "✅ Environment configuration updated"

# Check CDK output
if [ ! -d "../infrastructure/cdk.out" ]; then
    echo "❌ CDK output not found. Please deploy CDK stack first:"
    echo "   cd ../infrastructure && npm run deploy"
    exit 1
fi

echo "🚀 Starting SAM local API on http://127.0.0.1:3000..."
echo "🌐 Starting frontend server on http://localhost:8000..."
echo ""
echo "Servers will start in background. Use browser MCP tools for testing."
echo "Press Ctrl+C to stop all servers."

# Start SAM local
cd ../infrastructure
sam local start-api --template-file cdk.out/SeattleFamilyActivitiesMVPStack.template.json \
    --env-vars ../backend/env.json \
    --port 3000 \
    --host 127.0.0.1 &

SAM_PID=$!
cd ../backend

# Start frontend server
cd ../app
python3 -m http.server 8000 &
FRONTEND_PID=$!
cd ../backend

# Cleanup function
cleanup() {
    echo ""
    echo "🧹 Stopping servers..."
    kill $SAM_PID 2>/dev/null || true
    kill $FRONTEND_PID 2>/dev/null || true
    echo "✅ Servers stopped"
}

trap cleanup EXIT

echo "SAM local PID: $SAM_PID"
echo "Frontend PID: $FRONTEND_PID"
echo ""
echo "Waiting for servers to start..."
sleep 5

# Test if servers are running
if curl -s http://127.0.0.1:3000 >/dev/null 2>&1; then
    echo "✅ SAM local API is running on http://127.0.0.1:3000"
else
    echo "⚠️  SAM local may still be starting..."
fi

if curl -s http://localhost:8000 >/dev/null 2>&1; then
    echo "✅ Frontend server is running on http://localhost:8000"
else
    echo "⚠️  Frontend server may still be starting..."
fi

echo ""
echo "🎯 Servers are ready for browser MCP testing!"

# Keep running until interrupted
while true; do
    sleep 1
done