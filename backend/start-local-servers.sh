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

# Start backend using the dedicated backend script
echo "🔧 Starting backend server..."
./start-local-backend.sh &
BACKEND_PID=$!

# Wait for backend to start
echo "⏳ Waiting for backend to initialize..."
sleep 8

# Check CDK output
if [ ! -d "../infrastructure/cdk.out" ]; then
    echo "❌ CDK output not found. Please deploy CDK stack first:"
    echo "   cd ../infrastructure && npm run deploy"
    exit 1
fi

echo "🌐 Starting frontend server..."
echo ""
echo "Servers will start in background. Use browser MCP tools for testing."
echo "Press Ctrl+C to stop all servers."

# Start frontend server using dedicated script
cd ../app
./start-local-frontend.sh &
FRONTEND_PID=$!
cd ../backend

# Cleanup function
cleanup() {
    echo ""
    echo "🧹 Stopping servers..."
    kill $BACKEND_PID 2>/dev/null || true
    kill $FRONTEND_PID 2>/dev/null || true
    echo "✅ Servers stopped"
}

trap cleanup EXIT

echo "Backend PID: $BACKEND_PID"
echo "Frontend PID: $FRONTEND_PID"
echo ""
echo "Waiting for frontend server to start..."
sleep 3

# Test if servers are running
if curl -s http://127.0.0.1:3000 >/dev/null 2>&1; then
    echo "✅ Backend API is running on http://127.0.0.1:3000"
else
    echo "⚠️  Backend may still be starting..."
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