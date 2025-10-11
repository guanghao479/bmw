# Implementation Plan

- [x] 1. Add comprehensive logging and diagnostics to the extraction pipeline
  - Add detailed logging to FireCrawl response parsing in `backend/internal/services/firecrawl.go`
  - Log raw markdown content, extraction attempts, and structured data output
  - Add logging to schema conversion service to track data flow and conversion attempts
  - Create diagnostic information structure to capture extraction metadata
  - _Requirements: 2.1, 2.2, 2.3_

- [x] 2. Enhance FireCrawl markdown parsing for better event detection
  - [x] 2.1 Improve `parseParentMapActivities()` function to extract real event data from markdown
    - Parse markdown headers and content blocks to identify individual events
    - Extract event titles, dates, times, and descriptions from markdown structure
    - Implement pattern recognition for common event attribute formats
    - _Requirements: 1.1, 3.1, 4.1, 4.2_

  - [x] 2.2 Create robust `extractEventsFromMarkdown()` implementation
    - Implement regex patterns for date/time extraction (MM/DD/YYYY, "January 1", etc.)
    - Add location parsing from markdown content (venue names, addresses)
    - Create price/cost extraction patterns ("$10", "Free", "Donation")
    - Add age group detection patterns ("ages 3-5", "toddlers", "all ages")
    - _Requirements: 1.1, 3.1, 3.3_

  - [x] 2.3 Add data structure validation before returning structured data
    - Validate that extracted events have required fields (title, location)
    - Check data types and formats for consistency
    - Add confidence scoring based on extraction quality
    - _Requirements: 1.2, 3.2_

- [-] 3. Improve schema conversion service error handling and validation
  - [x] 3.1 Enhance `extractEventsFromRawData()` with better error reporting
    - Add detailed logging when events array is empty or malformed
    - Log the actual data structure received vs expected structure
    - Provide specific error messages about missing or invalid data fields
    - _Requirements: 2.1, 2.2, 2.4_

  - [x] 3.2 Add fallback strategies for malformed data in `convertSingleEvent()`
    - Implement multiple field name attempts for each required field
    - Add data type conversion and sanitization for extracted values
    - Create default value strategies when required fields are missing
    - _Requirements: 1.1, 3.2, 3.3_

  - [x] 3.3 Implement comprehensive field mapping and validation
    - Track which source fields were used for each Activity field
    - Add validation for extracted dates, times, and other structured data
    - Create detailed conversion issue reporting with suggestions
    - _Requirements: 2.3, 2.4_

- [ ] 4. Create diagnostic and debugging tools for troubleshooting
  - [x] 4.1 Add admin API endpoint for extraction debugging
    - Create `/api/debug/extract` endpoint that shows detailed extraction process
    - Return raw markdown, parsed events, and conversion results side-by-side
    - Include validation issues and suggestions for improvement
    - _Requirements: 2.1, 2.2_

  - [x] 4.2 Enhance existing admin endpoints with better error information
    - Update `/api/events/pending` to show conversion issues and raw data samples
    - Add detailed error messages to event approval/rejection responses
    - Include extraction confidence scores and field mapping information
    - _Requirements: 2.3, 2.4_

- [x] 4.3 Write focused unit tests for core extraction functionality
  - Test markdown parsing with real ParentMap and Remlinger Farms content samples
  - Test schema conversion with key malformed data scenarios
  - Test basic error handling for missing events and conversion failures
  - _Requirements: 1.1, 1.2, 3.1, 3.2_

- [x] 5. Fix specific issues with ParentMap and Remlinger Farms extraction
  - [x] 5.1 Analyze and fix ParentMap calendar parsing
    - Examine actual ParentMap markdown structure and identify event patterns
    - Implement specific parsing logic for ParentMap event format
    - Test extraction with real ParentMap URLs and validate results
    - _Requirements: 4.1, 4.4_

  - [x] 5.2 Analyze and fix Remlinger Farms event parsing
    - Examine actual Remlinger Farms markdown structure and identify event patterns
    - Implement specific parsing logic for Remlinger Farms event format
    - Test extraction with real Remlinger Farms URLs and validate results
    - _Requirements: 4.2, 4.4_

  - [x] 5.3 Implement source-specific parsing strategies
    - Add domain-based parsing strategy selection in FireCrawl service
    - Create configurable parsing rules for different website structures
    - Add fallback to generic parsing when source-specific parsing fails
    - _Requirements: 3.1, 3.3_

- [ ] 6. Validate and test the complete extraction pipeline
  - [ ] 6.1 Test end-to-end extraction with problematic sources
    - Run extraction tests against ParentMap and Remlinger Farms URLs
    - Verify that events are successfully found AND converted to Activity models
    - Validate that converted activities appear correctly in the admin interface
    - _Requirements: 1.1, 1.3, 4.1, 4.2_

  - [ ] 6.2 Implement monitoring and quality metrics
    - Add success rate tracking for extraction and conversion processes
    - Create alerts for conversion failure patterns
    - Add dashboard metrics for extraction quality and performance
    - _Requirements: 1.4, 2.1, 2.2_