# Integration Tests

This document describes the integration tests for the Seattle Family Activities scraper, which test the complete pipeline from web content extraction to structured activity data.

## Overview

The integration tests verify:

1. **Jina AI Reader** - Web content extraction from real Seattle family activity websites
2. **OpenAI GPT-4o-mini** - Activity extraction and structuring from web content
3. **Complete Pipeline** - End-to-end testing of the Jina + OpenAI workflow
4. **Error Handling** - Proper error handling and recovery
5. **Performance** - Response times, cost tracking, and concurrent processing

## Setup

### Prerequisites

1. **Go 1.21+** installed
2. **OpenAI API Key** with credits available
3. **Internet connection** for external API calls

### Environment Setup

1. Copy the environment template:
   ```bash
   cp .env.example .env
   ```

2. Add your OpenAI API key to `.env`:
   ```bash
   OPENAI_API_KEY=your_actual_api_key_here
   ```

3. Export the environment variable:
   ```bash
   export OPENAI_API_KEY=your_actual_api_key_here
   ```

## Running Tests

### Quick Start

Run all integration tests with the provided script:

```bash
./scripts/run_integration_tests.sh
```

### Manual Test Execution

#### Unit Tests First
```bash
go test ./internal/models -v
```

#### Jina Integration Tests
```bash
go test -tags=integration ./internal/services -run TestJinaClient -v
```

#### OpenAI Integration Tests
```bash
go test -tags=integration ./internal/services -run TestOpenAIClient -v
```

#### Pipeline Integration Tests
```bash
go test -tags=integration ./internal/services -run TestPipeline -v
```

#### Performance Benchmarks
```bash
go test -tags=integration ./internal/services -bench=. -benchtime=3s
```

### Skip Integration Tests

If you need to skip integration tests (e.g., in CI without API keys):

```bash
export SKIP_INTEGRATION_TESTS=true
go test ./internal/services -v
```

## Test Categories

### 1. Jina AI Reader Tests (`jina_integration_test.go`)

Tests the web content extraction service:

- **Real API Calls** - Tests actual Jina AI Reader API calls
- **Seattle Sources** - Tests extraction from Seattle's Child, Tinybeans, ParentMap
- **Error Handling** - Invalid URLs, timeouts, malformed responses
- **Performance** - Response times and content quality
- **Concurrent Processing** - Multiple simultaneous requests

Key tests:
- `TestJinaClient_RealAPICall` - Basic API functionality
- `TestJinaClient_SeattleChildWebsite` - Real Seattle source
- `TestJinaClient_ErrorHandling` - Error scenarios
- `TestJinaClient_Performance` - Speed and reliability

### 2. OpenAI Integration Tests (`openai_integration_test.go`)

Tests the activity extraction service:

- **Activity Extraction** - Structured data extraction from web content
- **Seattle Validation** - Ensures activities are Seattle-area focused
- **Data Quality** - Validates extracted activity schemas
- **Cost Tracking** - Token usage and cost estimation
- **Configuration** - Model settings and parameters

Key tests:
- `TestOpenAIClient_ExtractActivities_MusicAcademy` - Test content extraction
- `TestOpenAIClient_ValidateExtractionResponse` - Data validation
- `TestOpenAIClient_RealSeattleWebsite_Integration` - Real website processing
- `TestOpenAIClient_PerformanceTracking` - Metrics and costs

### 3. Pipeline Integration Tests (`pipeline_integration_test.go`)

Tests the complete end-to-end workflow:

- **Full Pipeline** - Jina extraction → OpenAI processing → Validation
- **Multiple Sources** - Testing across different Seattle websites
- **Error Recovery** - Handling failures at each pipeline stage
- **Concurrent Processing** - Parallel processing of multiple sources
- **Quality Assurance** - Comprehensive data quality checks

Key tests:
- `TestPipeline_EndToEnd_SeattleSources` - Complete workflow
- `TestPipeline_ErrorRecovery` - Failure handling
- `TestPipeline_Performance_Concurrent` - Parallel processing
- `TestPipeline_QualityAssurance` - Data quality validation

## Test Data Sources

The integration tests use real Seattle family activity websites:

1. **Seattle's Child** - `seattleschild.com`
   - Weekend activity guides
   - Event listings and reviews

2. **ParentMap** - `parentmap.com`
   - Calendar of family events
   - Activity recommendations

3. **Tinybeans Seattle** - `tinybeans.com/seattle`
   - Local family activities
   - Age-specific recommendations

4. **West Seattle Macaroni KID** - `westseattle.macaronikid.com`
   - Neighborhood-specific events
   - Community activities

## Performance Expectations

### Response Times
- **Jina extraction**: < 10 seconds per URL
- **OpenAI processing**: < 30 seconds per content batch
- **Complete pipeline**: < 45 seconds per source

### Cost Estimates
- **OpenAI GPT-4o-mini**: ~$0.0003 per 1K tokens
- **Typical extraction**: 1000-3000 tokens per source
- **Expected cost**: $0.0003-$0.001 per source

### Quality Metrics
- **Content extraction**: > 500 characters from real websites
- **Activity extraction**: 1-10 activities per source (varies by content)
- **Validation**: 90%+ of extracted activities should pass validation

## Troubleshooting

### Common Issues

#### 1. OpenAI API Key Issues
```
Error: OPENAI_API_KEY environment variable is required
```
**Solution**: Set your OpenAI API key in environment variables

#### 2. API Rate Limits
```
Error: rate limit exceeded
```
**Solution**: Wait and retry, or reduce concurrent requests

#### 3. Network Timeouts
```
Error: context deadline exceeded
```
**Solution**: Check internet connection, increase timeout values

#### 4. Content Extraction Failures
```
Error: content too short (X chars), might be an error page
```
**Solution**: Website may be blocking requests, try different user agent

### Debugging Tips

1. **Increase verbosity**: Add `-v` flag to test commands
2. **Check individual tests**: Run specific test functions
3. **Verify API access**: Test API keys with simple requests
4. **Monitor costs**: Track token usage and costs during testing

## CI/CD Integration

For automated testing in CI/CD pipelines:

1. **Skip integration tests** when API keys aren't available:
   ```bash
   export SKIP_INTEGRATION_TESTS=true
   ```

2. **Set reasonable timeouts** for CI environments:
   ```bash
   go test -timeout=10m -tags=integration
   ```

3. **Monitor costs** in production CI runs to avoid unexpected charges

## Contributing

When adding new integration tests:

1. **Use build tags**: Add `//go:build integration` to test files
2. **Check environment**: Skip tests when `SKIP_INTEGRATION_TESTS=true`
3. **Handle errors gracefully**: Don't fail tests due to network issues
4. **Document expectations**: Add performance and cost expectations
5. **Test real scenarios**: Use actual Seattle websites when possible

## Security Notes

- **API Keys**: Never commit API keys to version control
- **Rate Limits**: Respect API rate limits to avoid account suspension
- **Cost Control**: Monitor API usage to prevent unexpected charges
- **Error Handling**: Don't log sensitive information in test output