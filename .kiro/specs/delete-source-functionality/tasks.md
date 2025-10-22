# Implementation Plan

- [x] 1. Implement backend delete source functionality
  - Create DeleteSourceCompletely method in DynamoDB service with transaction-based cascading deletion
  - Add DeletionResult struct to track what records were deleted
  - Implement proper error handling and rollback logic
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

- [x] 1.1 Add DeletionResult model and helper functions
  - Define DeletionResult struct in models package
  - Add helper functions for source record queries
  - Create admin event logging structures for deletion tracking
  - _Requirements: 3.1, 4.5_

- [x] 1.2 Implement cascading deletion logic in DynamoDB service
  - Add DeleteSourceCompletely method with transaction support
  - Query and delete all associated activity records
  - Delete source submission, analysis, and configuration records
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

- [x] 1.3 Write unit tests for deletion service methods
  - Test successful deletion scenarios with complete and partial sources
  - Test error handling and rollback behavior
  - Test transaction failure scenarios
  - _Requirements: 3.5_

- [ ] 2. Add DELETE API endpoint to admin API
  - Create new DELETE /api/sources/{sourceId} endpoint handler
  - Add request validation and authentication checks
  - Implement proper error responses and status codes
  - _Requirements: 4.1, 4.2, 4.4_

- [ ] 2.1 Implement handleDeleteSource function
  - Parse and validate source ID from URL path
  - Call DynamoDB service deletion method
  - Format response with deletion results
  - _Requirements: 4.1, 4.4_

- [ ] 2.2 Add admin event logging for deletions
  - Log deletion attempts and results to admin events table
  - Include administrator identity and timestamp
  - Record detailed deletion statistics
  - _Requirements: 4.1, 4.5_

- [ ] 2.3 Write integration tests for delete API endpoint
  - Test successful deletion with various source types
  - Test error scenarios (not found, permission denied, server errors)
  - Test admin event logging functionality
  - _Requirements: 4.2, 4.4_

- [ ] 3. Enhance admin interface with delete functionality
  - Add delete buttons to source cards in the admin interface
  - Implement confirmation dialog with source details
  - Add proper error handling and success messaging
  - _Requirements: 1.1, 1.3, 1.4, 1.5, 2.1, 2.2, 2.4_

- [ ] 3.1 Add delete buttons to source management UI
  - Add delete button to each source card alongside existing action buttons
  - Style button with red color and trash icon using Tailwind CSS
  - Wire up click handlers to trigger confirmation dialog
  - _Requirements: 1.1, 1.5_

- [ ] 3.2 Implement confirmation dialog component
  - Create modal overlay using native JavaScript and existing CSS patterns
  - Display source details (name, URL, activity count, last scrape date)
  - Add confirmation input (type source name) and action buttons
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [ ] 3.3 Add delete API integration to admin interface
  - Implement deleteSource method in SourceManagementAdmin class
  - Add proper loading states and button disabling during deletion
  - Handle API responses and display appropriate success/error messages
  - _Requirements: 1.2, 1.3, 1.4_

- [ ] 3.4 Enhance alert system for deletion feedback
  - Update showAlert method to handle deletion-specific messages
  - Add auto-refresh of source list after successful deletion
  - Implement proper error message display for various failure scenarios
  - _Requirements: 1.3, 1.4_

- [ ] 3.5 Write frontend tests for delete functionality
  - Test confirmation dialog behavior and validation
  - Test button states and loading indicators
  - Test error message display and success workflows
  - _Requirements: 1.1, 1.3, 1.4_

- [ ] 4. Update routing and integrate delete endpoint
  - Add DELETE route handling to admin API main function
  - Update API documentation with new delete endpoint
  - Test end-to-end deletion workflow
  - _Requirements: 4.2, 4.4_

- [ ] 4.1 Add DELETE route to admin API Lambda
  - Update main.go router to handle DELETE /api/sources/{sourceId}
  - Add proper HTTP method and path matching
  - Wire up to handleDeleteSource function
  - _Requirements: 4.2_

- [ ] 4.2 Test complete delete workflow end-to-end
  - Test deletion of sources with various data states
  - Verify all associated records are properly removed
  - Confirm UI updates correctly after successful deletion
  - _Requirements: 1.2, 3.4, 3.5_