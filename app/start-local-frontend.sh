#!/bin/bash

# Start local frontend development server

set -e

echo "🌐 Starting Local Frontend Development Server"
echo "============================================="

# Check if we're in the right directory
if [ ! -f "index.html" ]; then
    echo "❌ Please run this script from the app directory"
    exit 1
fi

# Check for Python
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 not found. Please install Python 3"
    exit 1
fi

echo "🚀 Starting frontend server on http://localhost:8000..."
echo "📁 Serving files from: $(pwd)"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

# Start frontend server
exec python3 -m http.server 8000