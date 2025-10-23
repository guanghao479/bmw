#!/bin/bash

# Convenience script to start frontend from project root

set -e

echo "🚀 Starting Frontend Server"
echo "==========================="

# Check if app directory exists
if [ ! -d "app" ]; then
    echo "❌ app directory not found. Please run from project root."
    exit 1
fi

# Navigate to app directory and start frontend
cd app
exec ./start-local-frontend.sh