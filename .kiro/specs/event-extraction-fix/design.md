# Design Document

## Overview

The event extraction system is experiencing conversion failures where events are successfully found during the FireCrawl scraping phase but fail to convert into the expected Activity data structure. The issue appears to be in the data transformation pipeline between the raw FireCrawl response and the schema conversion service.

Based on analysis of the codebase, the problem lies in the mismatch between what FireCrawl returns (markdown content) and what the schema conversion service expects (structured JSON data with events arrays). The current implementation creates mock structured data from markdown but this conversion is incomplete and unreliable.

## Architecture

### Current Flow (Problematic)
```
FireCrawl API → Markdown Content → Mock Structured Data → Schema Conversion → Activity Model
```

### Proposed Fixed Flow
```
FireCrawl API → Enhanced Markdown Parser → Validated Structured Data → Improved Schema Conversion → Activity Model
```

## Components and Interfaces

### 1. Enhanced FireCrawl Response Parser

**Location**: `backend/internal/services/firecrawl.go`

**Current Issues**:
- `parseParentMapActivities()` creates minimal mock data
- `extractEventsFromMarkdown()` uses basic header detection
- No validation of extracted data structure
- Missing field mapping for common event attributes

**Proposed Enhancements**:
- Implement robust markdown parsing for event detection
- Add pattern recognition for dates, times, locations, and prices
- Create comprehensive field extraction with fallback strategies
- Add validation layer before returning structured data

### 2. Schema Conversion Service Improvements

**Location**: `backend/internal/services/schema_conversion.go`

**Current Issues**:
- `extractEventsFromRawData()` expects perfect JSON structure
- Limited error handling when events array is empty
- No debugging information about data structure mismatches

**Proposed Enhancements**:
- Add robust data structure validation with detailed error reporting
- Implement fallback extraction strategies for malformed data
- Add comprehensive logging for debugging conversion failures
- Create data structure normalization layer

### 3. Admin Event Processing Pipeline

**Location**: `backend/cmd/admin_api/main.go`

**Current Issues**:
- Limited error reporting in event processing endpoints
- No intermediate validation steps
- Missing diagnostic information for troubleshooting

**Proposed Enhancements**:
- Add detailed error logging with data samples
- Implement validation checkpoints throughout the pipeline
- Create diagnostic endpoints for troubleshooting extraction issues
- Add data quality metrics and reporting

## Data Models

### Enhanced Extraction Response Structure

```go
type EnhancedFireCrawlResponse struct {
    Success         bool                   `json:"success"`
    RawMarkdown     string                 `json:"raw_markdown"`
    ExtractedEvents []ExtractedEvent       `json:"extracted_events"`
    ExtractionMeta  ExtractionMetadata     `json:"extraction_metadata"`
    ValidationIssues []ValidationIssue     `json:"validation_issues"`
}

type ExtractedEvent struct {
    RawData         map[string]interface{} `json:"raw_data"`
    ConfidenceScore float64                `json:"confidence_score"`
    FieldSources    map[string]string      `json:"field_sources"`
    ExtractionMethod string                `json:"extraction_method"`
}

type ValidationIssue struct {
    Severity    string `json:"severity"`    // error|warning|info
    Field       string `json:"field"`
    Message     string `json:"message"`
    Suggestion  string `json:"suggestion"`
}
```

### Conversion Result Enhancement

```go
type EnhancedConversionResult struct {
    Activity        *models.Activity       `json:"activity"`
    Issues          []ConversionIssue      `json:"issues"`
    FieldMappings   map[string]string      `json:"field_mappings"`
    ConfidenceScore float64                `json:"confidence_score"`
    RawDataSample   map[string]interface{} `json:"raw_data_sample"`
    ConversionPath  []string               `json:"conversion_path"`
}

type ConversionIssue struct {
    Type        string `json:"type"`        // missing_field|invalid_format|low_confidence
    Field       string `json:"field"`
    Message     string `json:"message"`
    Suggestion  string `json:"suggestion"`
    RawValue    string `json:"raw_value"`
}
```

## Error Handling

### 1. Extraction Phase Error Handling

- **Markdown Parsing Failures**: Log raw markdown sample and parsing attempts
- **Pattern Recognition Issues**: Provide detailed regex match information
- **Data Structure Validation**: Report expected vs actual structure with examples

### 2. Conversion Phase Error Handling

- **Missing Events Array**: Analyze raw data structure and suggest corrections
- **Field Mapping Failures**: Log attempted field names and available fields
- **Validation Errors**: Provide specific field-level error messages with suggestions

### 3. Diagnostic Information

- **Data Flow Tracking**: Log data at each transformation step
- **Performance Metrics**: Track extraction success rates by source
- **Quality Scoring**: Implement confidence scoring for extracted data

## Testing Strategy

### 1. Unit Tests for Enhanced Components

- **Markdown Parser Tests**: Test with real ParentMap and Remlinger Farms content
- **Schema Conversion Tests**: Test with various malformed data structures
- **Validation Tests**: Test error detection and reporting accuracy

### 2. Integration Tests

- **End-to-End Extraction**: Test complete pipeline with problematic sources
- **Error Recovery Tests**: Test graceful handling of conversion failures
- **Data Quality Tests**: Validate extracted data meets minimum quality standards

### 3. Debugging Tools

- **Extraction Debugger**: Admin endpoint to test extraction with detailed logging
- **Conversion Simulator**: Tool to test conversion with sample data
- **Data Inspector**: Interface to examine raw vs converted data side-by-side

## Implementation Phases

### Phase 1: Enhanced Logging and Diagnostics
- Add comprehensive logging throughout the extraction pipeline
- Create diagnostic endpoints for troubleshooting
- Implement data structure validation with detailed error reporting

### Phase 2: Improved Markdown Parsing
- Enhance FireCrawl response parsing for better event detection
- Implement robust pattern recognition for event attributes
- Add validation layer for extracted structured data

### Phase 3: Schema Conversion Improvements
- Improve error handling in schema conversion service
- Add fallback strategies for malformed data
- Implement confidence scoring and quality metrics

### Phase 4: Testing and Validation
- Create comprehensive test suite with real-world data
- Implement automated quality checks
- Add monitoring and alerting for conversion failures