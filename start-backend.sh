#!/bin/bash

# Convenience script to start backend from project root

set -e

echo "🚀 Starting Backend Server"
echo "=========================="

# Check if backend directory exists
if [ ! -d "backend" ]; then
    echo "❌ backend directory not found. Please run from project root."
    exit 1
fi

# Navigate to backend directory and start backend
cd backend
exec ./start-local-backend.sh