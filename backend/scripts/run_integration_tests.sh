#!/bin/bash

# Integration test runner for Seattle Family Activities scraper
# This script runs integration tests for both Jina and OpenAI services

set -e

echo "🧪 Seattle Family Activities - Integration Tests"
echo "================================================="

# Check environment variables
echo "🔍 Checking environment variables..."

if [ -z "$OPENAI_API_KEY" ]; then
    echo "❌ OPENAI_API_KEY is not set"
    echo "   Please set your OpenAI API key:"
    echo "   export OPENAI_API_KEY=your_key_here"
    exit 1
else
    echo "✅ OPENAI_API_KEY is set"
fi

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    exit 1
else
    echo "✅ Go is available: $(go version)"
fi

# Navigate to backend directory
cd "$(dirname "$0")/.."

echo ""
echo "📁 Working directory: $(pwd)"

# Download dependencies
echo "📦 Installing dependencies..."
go mod tidy

# Run unit tests first
echo ""
echo "🧪 Running unit tests..."
go test ./internal/models -v

echo ""
echo "🌐 Running integration tests..."
echo "   Note: These tests make real API calls and may take several minutes"

# Set build tag for integration tests
export INTEGRATION_TESTS=true

# Test categories
declare -a test_categories=(
    "Jina service tests"
    "OpenAI service tests" 
    "Pipeline tests"
)

declare -a test_patterns=(
    "TestJinaClient"
    "TestOpenAIClient"
    "TestPipeline"
)

total_tests=0
passed_tests=0
failed_tests=0

# Run each test category
for i in "${!test_categories[@]}"; do
    category="${test_categories[$i]}"
    pattern="${test_patterns[$i]}"
    
    echo ""
    echo "🔬 Running $category..."
    echo "   Pattern: $pattern"
    
    if go test -tags=integration ./internal/services -run "$pattern" -v -timeout=10m; then
        echo "✅ $category passed"
        ((passed_tests++))
    else
        echo "❌ $category failed"
        ((failed_tests++))
    fi
    ((total_tests++))
done

# Run performance benchmarks
echo ""
echo "⚡ Running performance benchmarks..."
if go test -tags=integration ./internal/services -bench=. -benchtime=3s -timeout=5m; then
    echo "✅ Benchmarks completed"
else
    echo "⚠️  Some benchmarks may have failed (this is not critical)"
fi

# Final summary
echo ""
echo "📊 Integration Test Summary"
echo "=========================="
echo "Total test categories: $total_tests"
echo "Passed: $passed_tests"
echo "Failed: $failed_tests"

if [ $failed_tests -eq 0 ]; then
    echo "🎉 All integration tests passed!"
    echo ""
    echo "💡 Next steps:"
    echo "   1. Review the test output above for performance metrics"
    echo "   2. Check that activities are being extracted correctly from Seattle sources"
    echo "   3. Verify cost estimates are reasonable for production use"
    echo "   4. Consider running the tests again to check consistency"
    exit 0
else
    echo "💥 Some integration tests failed"
    echo ""
    echo "🔧 Troubleshooting:"
    echo "   1. Check your OPENAI_API_KEY is valid and has credits"
    echo "   2. Verify internet connection for external API calls"
    echo "   3. Check if the Seattle source websites are accessible"
    echo "   4. Review error messages above for specific issues"
    exit 1
fi