# Debug API Endpoint

## POST /api/debug/extract

This endpoint provides detailed debugging information for the extraction and conversion pipeline. It performs the same extraction as the regular `/api/crawl/submit` endpoint but returns comprehensive diagnostic information without storing the results.

### Request

```json
{
  "url": "https://example.com/events",
  "schema_type": "events",
  "custom_schema": {
    // Optional custom schema for extraction
  }
}
```

### Parameters

- `url` (required): The URL to extract data from
- `schema_type` (optional): The schema type to use ("events", "activities", "venues", "custom"). Defaults to "events"
- `custom_schema` (optional): Custom schema definition when schema_type is "custom"

### Response

The response includes detailed information about each step of the extraction and conversion process:

```json
{
  "success": true,
  "message": "Debug extraction completed",
  "data": {
    "extraction": {
      "url": "https://example.com/events",
      "schema_type": "events",
      "success": true,
      "events_count": 5,
      "credits_used": 1,
      "processing_time": "2.5s",
      "schema_used": { /* schema object */ }
    },
    "raw_data": {
      "markdown_length": 15420,
      "markdown_sample": "# Events\n\n## Art Workshop\nDate: 2024-12-15...",
      "structured_data": {
        "events": [
          {
            "title": "Kids Art Workshop",
            "date": "2024-12-15",
            "location": "Community Center"
          }
        ]
      }
    },
    "conversion": {
      "success": true,
      "activity": { /* converted Activity object */ },
      "issues": ["Missing address information"],
      "field_mappings": {
        "title": "title",
        "description": "description",
        "location": "location",
        "schedule": "date",
        "pricing": "not_found"
      },
      "confidence_score": 85.0,
      "detailed_mappings": {
        "title": {
          "activity_field": "title",
          "source_field": "title",
          "mapping_type": "direct",
          "confidence": 0.9,
          "validation_status": "valid"
        }
      },
      "validation_results": {
        "title": {
          "validation_status": "valid",
          "confidence": 0.9,
          "mapping_type": "direct"
        }
      }
    },
    "extraction_diagnostics": {
      "url": "https://example.com/events",
      "start_time": "2024-10-10T09:00:00Z",
      "end_time": "2024-10-10T09:00:02Z",
      "processing_time": "2.5s",
      "raw_markdown_length": 15420,
      "raw_markdown_sample": "# Events\n\n## Art Workshop...",
      "extraction_attempts": [
        {
          "method": "markdown_parsing",
          "success": true,
          "events_found": 5,
          "issues": []
        }
      ],
      "validation_issues": [
        {
          "field": "date",
          "message": "Date format validation passed",
          "severity": "info"
        }
      ],
      "credits_used": 1,
      "success": true
    },
    "conversion_diagnostics": {
      "admin_event_id": "debug-12345",
      "source_url": "https://example.com/events",
      "schema_type": "events",
      "processing_time": "150ms",
      "success": true,
      "confidence_score": 85.0,
      "field_mappings": { /* detailed field mapping info */ },
      "conversion_issues": [
        {
          "type": "missing_field",
          "field": "location.address",
          "message": "No address information found",
          "suggestion": "Include address field in source data",
          "severity": "warning"
        }
      ]
    },
    "suggestions": [
      "Successfully extracted events - consider this schema configuration for production",
      "Missing or generated fields: pricing - check if source contains this information"
    ]
  }
}
```

### Use Cases

1. **Troubleshooting Failed Extractions**: See exactly where the extraction pipeline fails
2. **Schema Optimization**: Test different schema types to find the best fit for a source
3. **Data Quality Assessment**: Review field mappings and validation results
4. **Performance Analysis**: Monitor extraction and conversion times
5. **Development and Testing**: Validate changes to extraction logic

### Key Features

- **No Data Storage**: Results are not saved to the database
- **Comprehensive Diagnostics**: Includes both extraction and conversion diagnostics
- **Field Mapping Details**: Shows which source fields were used for each Activity field
- **Validation Results**: Detailed validation information with confidence scores
- **Actionable Suggestions**: Specific recommendations for improving extraction quality
- **Raw Data Access**: View both the original markdown and structured data

### Error Responses

If the extraction fails, the response will include error details:

```json
{
  "success": false,
  "error": "Failed to extract data from URL: connection timeout"
}
```

### Authentication

This endpoint requires the same authentication as other admin API endpoints.