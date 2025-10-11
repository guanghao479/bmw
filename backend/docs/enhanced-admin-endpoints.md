# Enhanced Admin Endpoints

This document describes the enhancements made to existing admin API endpoints to provide better error information, diagnostic data, and field mapping details.

## Enhanced GET /api/events/pending

The pending events endpoint now includes comprehensive diagnostic information for each event.

### Enhanced Response Format

```json
{
  "success": true,
  "message": "Pending events retrieved successfully",
  "data": [
    {
      "event_id": "12345",
      "source_url": "https://example.com/events",
      "schema_type": "events",
      "status": "pending",
      "extracted_at": "2024-10-10T09:00:00Z",
      "extracted_by_user": "admin@example.com",
      "events_count": 5,
      "conversion_issues": ["Missing address information"],
      "can_approve": true,
      "admin_notes": "Extracted from calendar page",
      
      // NEW: Detailed conversion information
      "conversion_details": {
        "has_conversion_preview": true,
        "conversion_issues_count": 1,
        "conversion_status": "success",
        "confidence_score": 85.0,
        "field_mappings": {
          "title": "title",
          "description": "description",
          "location": "venue",
          "schedule": "date",
          "pricing": "not_found"
        },
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
        },
        "issues_by_type": {
          "missing_data": ["Missing address information"],
          "format_issues": [],
          "other": []
        }
      },
      
      // NEW: Raw data sample for debugging
      "raw_data_sample": {
        "structure": {
          "total_fields": 3,
          "field_types": {
            "array": 1,
            "string": 2
          },
          "array_fields": ["events"],
          "string_fields": ["title", "url"]
        },
        "sample_fields": {
          "events": {
            "type": "array",
            "length": 5,
            "sample": [
              {
                "title": "Kids Art Workshop",
                "date": "2024-12-15"
              }
            ]
          }
        },
        "total_fields": 3
      },
      
      // NEW: Data quality assessment
      "quality_assessment": {
        "overall_score": 85.0,
        "factors": {
          "data_availability": {
            "score": 100,
            "message": "Data successfully extracted"
          },
          "conversion_success": {
            "score": 100,
            "message": "Successfully converted to Activity model"
          },
          "conversion_issues": {
            "score": 80,
            "message": "1 minor conversion issues"
          },
          "events_count": {
            "score": 100,
            "message": "5 events found"
          }
        },
        "recommendations": []
      }
    }
  ]
}
```

### New Fields Explained

- **conversion_details**: Comprehensive information about the conversion process
  - `conversion_status`: "success", "failed", or "unknown"
  - `confidence_score`: Overall confidence in the conversion (0-100)
  - `field_mappings`: Which source fields were used for each Activity field
  - `detailed_mappings`: Enhanced field mapping with validation status
  - `issues_by_type`: Conversion issues categorized by type

- **raw_data_sample**: Sample of the extracted raw data for debugging
  - `structure`: Analysis of the data structure
  - `sample_fields`: Sample values from each field type
  - Limited to prevent large responses

- **quality_assessment**: Overall assessment of data quality
  - `overall_score`: Composite quality score (0-100)
  - `factors`: Individual quality factors with scores
  - `recommendations`: Actionable suggestions for improvement

## Enhanced PUT /api/events/{id}/approve

The event approval endpoint now provides detailed diagnostic information in both success and error responses.

### Enhanced Error Response

When conversion fails:

```json
{
  "success": false,
  "error": "Failed to convert event to activity - see details for more information",
  "data": {
    "conversion_error": "No events found in extracted data",
    "event_id": "12345",
    "source_url": "https://example.com/events",
    "schema_type": "events",
    "diagnostics": {
      "processing_time": "150ms",
      "conversion_issues": [
        {
          "type": "missing_field",
          "field": "events",
          "message": "No events found in extracted data",
          "suggestion": "Check if the raw data contains an events array",
          "severity": "error"
        }
      ],
      "field_mappings": {},
      "confidence_score": 0.0
    }
  }
}
```

When activity generation fails:

```json
{
  "success": false,
  "error": "Could not generate valid activity from event data - see details for diagnostic information",
  "data": {
    "conversion_issues": ["Missing required title field", "Invalid date format"],
    "field_mappings": {
      "title": "not_found",
      "description": "description",
      "location": "venue"
    },
    "confidence_score": 25.0,
    "event_id": "12345",
    "source_url": "https://example.com/events",
    "detailed_mappings": {
      "title": {
        "activity_field": "title",
        "source_field": "not_found",
        "mapping_type": "default",
        "confidence": 0.1,
        "validation_status": "invalid"
      }
    },
    "validation_results": {
      "title": {
        "validation_status": "invalid",
        "confidence": 0.1,
        "mapping_type": "default"
      }
    },
    "suggestions": [
      "Check if the extracted data contains valid event information",
      "Try using a different schema type for extraction",
      "Review the conversion issues for specific problems"
    ]
  }
}
```

### Enhanced Success Response

```json
{
  "success": true,
  "message": "Event approved and published successfully",
  "data": {
    "event_id": "12345",
    "activity_id": "activity-67890",
    "status": "approved",
    "conversion_summary": {
      "confidence_score": 85.0,
      "issues_count": 1,
      "field_mappings_count": 8
    },
    "conversion_details": {
      "processing_time": "150ms",
      "success": true,
      "field_mappings": {
        "title": {
          "activity_field": "title",
          "source_field": "title",
          "mapping_type": "direct",
          "confidence": 0.9
        }
      }
    },
    "warnings": ["Missing address information"]
  }
}
```

## Enhanced PUT /api/events/{id}/reject

The event rejection endpoint now includes diagnostic information to help understand why events are being rejected.

### Enhanced Response

```json
{
  "success": true,
  "message": "Event rejected successfully",
  "data": {
    "event_id": "12345",
    "status": "rejected",
    "rejection_details": {
      "rejected_by": "admin@example.com",
      "rejected_at": "2024-10-10T10:00:00Z",
      "admin_notes": "Poor data quality - missing essential fields",
      "source_url": "https://example.com/events",
      "schema_type": "events"
    },
    "conversion_analysis": {
      "conversion_succeeded": true,
      "confidence_score": 45.0,
      "issues_count": 5,
      "issues": [
        "Missing required title field",
        "Invalid date format",
        "No location information"
      ],
      "field_mappings": {
        "title": "not_found",
        "description": "description",
        "location": "not_found"
      }
    },
    "quality_assessment": {
      "overall_score": 35.0,
      "factors": {
        "data_availability": {
          "score": 100,
          "message": "Data successfully extracted"
        },
        "conversion_success": {
          "score": 100,
          "message": "Successfully converted to Activity model"
        },
        "conversion_issues": {
          "score": 40,
          "message": "5 conversion issues detected"
        },
        "events_count": {
          "score": 100,
          "message": "3 events found"
        }
      },
      "recommendations": [
        "Review conversion issues and improve source data quality"
      ]
    }
  }
}
```

## Benefits of Enhanced Endpoints

1. **Better Debugging**: Detailed diagnostic information helps identify exactly where extraction or conversion fails
2. **Quality Assessment**: Automated quality scoring helps prioritize which events need attention
3. **Field Mapping Visibility**: Clear visibility into which source fields are being used for each Activity field
4. **Actionable Suggestions**: Specific recommendations for improving data quality
5. **Performance Monitoring**: Processing times and confidence scores help monitor system performance
6. **Audit Trail**: Detailed information about why events were approved or rejected

## Backward Compatibility

All enhancements are additive - existing fields remain unchanged, ensuring backward compatibility with existing admin interfaces.