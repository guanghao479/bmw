# Delete Source Functionality Design

## Overview

This design implements comprehensive delete source functionality for the Seattle Family Activities admin interface. The solution provides a secure, user-friendly way to permanently remove sources and all associated data from the system, with proper confirmation workflows and error handling.

## Architecture

The delete functionality follows the existing three-tier architecture:

1. **Frontend (Admin Interface)**: Enhanced UI with delete buttons and confirmation dialogs
2. **Backend API (Admin API Lambda)**: New DELETE endpoint for source removal
3. **Data Layer (DynamoDB Service)**: New methods for cascading source deletion

## Components and Interfaces

### Frontend Components

#### Delete Button Component
- **Location**: Added to each source card in the admin interface
- **Styling**: Red button with trash icon, positioned alongside existing action buttons
- **State Management**: Disabled during deletion process to prevent duplicate requests

#### Confirmation Dialog Component
- **Modal Implementation**: Native JavaScript modal using existing CSS patterns (similar to event details modal)
- **Styling**: Tailwind CSS classes following existing design system
- **Content**: Source name, URL, activity count, last scrape date
- **Confirmation Method**: Type source name to confirm OR click "Delete Permanently" button
- **Actions**: Cancel (closes modal) and Delete (proceeds with deletion)
- **DOM Management**: Created/removed dynamically using createElement and appendChild

#### Alert System Enhancement
- **Success Messages**: "Source [name] deleted successfully"
- **Error Messages**: Detailed error information from backend
- **Loading States**: Visual feedback during deletion process

### Backend API Enhancement

#### New DELETE Endpoint
```
DELETE /api/sources/{sourceId}
```

**Request Headers:**
- Content-Type: application/json
- Authorization: Admin privileges required

**Response Format:**
```json
{
  "success": true|false,
  "message": "Source deleted successfully" | "Error message",
  "data": {
    "deleted_records": {
      "submission": true|false,
      "analysis": true|false,
      "config": true|false,
      "activities_count": number
    }
  }
}
```

**Error Responses:**
- 400: Invalid source ID format
- 404: Source not found
- 403: Insufficient permissions
- 500: Internal server error during deletion

### Data Layer Enhancement

#### DynamoDB Service Methods

**DeleteSourceCompletely Method:**
```go
func (s *DynamoDBService) DeleteSourceCompletely(ctx context.Context, sourceID string) (*DeletionResult, error)
```

**Deletion Process:**
1. Validate source exists
2. Query all associated activity records
3. Begin transaction for atomic deletion
4. Delete source submission record
5. Delete source analysis record (if exists)
6. Delete source configuration record (if exists)
7. Delete all associated activity records
8. Commit transaction or rollback on failure

**DeletionResult Structure:**
```go
type DeletionResult struct {
    SourceID         string `json:"source_id"`
    SubmissionDeleted bool   `json:"submission_deleted"`
    AnalysisDeleted   bool   `json:"analysis_deleted"`
    ConfigDeleted     bool   `json:"config_deleted"`
    ActivitiesDeleted int    `json:"activities_deleted"`
    TotalRecords      int    `json:"total_records"`
}
```

## Data Models

### Admin Event Logging
New admin event type for source deletions:

```go
type AdminEvent struct {
    PK           string    `json:"PK"`           // ADMIN_EVENT#{event_id}
    SK           string    `json:"SK"`           // TIMESTAMP#{timestamp}
    EventType    string    `json:"event_type"`  // "source_deleted"
    AdminUser    string    `json:"admin_user"`
    SourceID     string    `json:"source_id"`
    SourceName   string    `json:"source_name"`
    DeletionData DeletionResult `json:"deletion_data"`
    Timestamp    time.Time `json:"timestamp"`
}
```

### Database Queries

#### Find Associated Activities
```sql
-- Query to find all activities from a specific source
SELECT * FROM family-activities 
WHERE source_id = :sourceId
```

#### Cascading Deletion Order
1. Activities (family-activities table)
2. Source Configuration (source-management table, SK=CONFIG)
3. Source Analysis (source-management table, SK=ANALYSIS)  
4. Source Submission (source-management table, SK=SUBMISSION)

## Error Handling

### Frontend Error Handling
- **Network Errors**: Display "Connection failed, please try again"
- **Permission Errors**: Display "Insufficient permissions to delete source"
- **Not Found Errors**: Display "Source not found or already deleted"
- **Server Errors**: Display detailed error message from backend

### Backend Error Handling
- **Validation Errors**: Return 400 with specific validation message
- **Database Errors**: Log error details, return 500 with generic message
- **Transaction Failures**: Automatic rollback, return detailed failure information
- **Partial Deletion**: Log which records were deleted, attempt cleanup

### Rollback Strategy
- Use DynamoDB transactions for atomic operations
- If any deletion fails, rollback all changes
- Log partial deletion attempts for manual cleanup
- Return detailed status of what was/wasn't deleted

## Testing Strategy

### Unit Tests
- **Frontend**: Test confirmation dialog behavior, button states, error display
- **Backend**: Test DELETE endpoint with various scenarios (success, not found, errors)
- **Data Layer**: Test cascading deletion logic, transaction rollback, error handling

### Integration Tests
- **End-to-End**: Complete deletion workflow from UI click to database cleanup
- **Error Scenarios**: Network failures, database errors, permission issues
- **Data Integrity**: Verify no orphaned records remain after deletion

### Test Data Scenarios
1. **Complete Source**: Source with submission, analysis, config, and activities
2. **Partial Source**: Source with only submission (no analysis/config)
3. **Active Source**: Currently active source with recent scraping data
4. **Failed Source**: Source with failed analysis or configuration

## Security Considerations

### Authentication & Authorization
- Verify admin privileges before allowing deletion
- Log all deletion attempts with user identity
- Rate limiting on deletion endpoints to prevent abuse

### Data Protection
- Confirmation dialog prevents accidental deletions
- Transaction-based deletion ensures data consistency
- Audit trail for all deletion operations

### Input Validation
- Validate source ID format and existence
- Sanitize all user inputs in confirmation dialog
- Prevent SQL injection through parameterized queries

## Performance Considerations

### Batch Operations
- Delete activities in batches to avoid timeout
- Use parallel deletion for independent records
- Implement progress tracking for large deletions

### Database Optimization
- Use efficient queries to find associated records
- Minimize transaction scope to reduce lock time
- Index optimization for source lookup operations

### UI Responsiveness
- Async deletion with loading indicators
- Optimistic UI updates where appropriate
- Background refresh after successful deletion