#!/bin/bash

# Start local frontend development server
# This script serves the frontend application for local development

set -e

echo "🌐 Starting Local Frontend Development Server"
echo "============================================"

# Check if we're in the right directory (project root)
if [ ! -d "app" ]; then
    echo "❌ Please run this script from the project root directory"
    echo "   The 'app' directory should be present"
    exit 1
fi

# Check for Python (most common)
if command -v python3 &> /dev/null; then
    PYTHON_CMD="python3"
elif command -v python &> /dev/null; then
    PYTHON_CMD="python"
else
    echo "❌ Python not found. Please install Python or use an alternative:"
    echo ""
    echo "Alternative options:"
    echo "   - Node.js: npx serve app"
    echo "   - PHP: php -S localhost:8000 -t app"
    echo "   - Ruby: ruby -run -e httpd app -p 8000"
    exit 1
fi

# Change to app directory
cd app

echo "🚀 Starting frontend server..."
echo "📍 Frontend will be available at: http://localhost:8000"
echo "📋 Available pages:"
echo "   - Main app: http://localhost:8000/"
echo "   - Admin:    http://localhost:8000/admin.html"
echo ""
echo "💡 Make sure the backend is running at http://127.0.0.1:3000"
echo "   Run: ./backend/start-local-backend.sh"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

# Start the server
exec $PYTHON_CMD -m http.server 8000