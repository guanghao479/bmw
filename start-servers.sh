#!/bin/bash

# Convenience script to start both backend and frontend servers from project root

set -e

echo "🚀 Starting Both Servers"
echo "========================"

# Check if backend directory exists
if [ ! -d "backend" ]; then
    echo "❌ backend directory not found. Please run from project root."
    exit 1
fi

# Navigate to backend directory and start both servers
cd backend
exec ./start-local-servers.sh