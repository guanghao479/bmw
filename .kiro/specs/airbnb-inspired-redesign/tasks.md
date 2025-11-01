# Implementation Plan

- [x] 1. Implement new header structure and layout
  - Create three-section header layout with flexbox using Tailwind CSS
  - Implement sticky header with proper z-index and background
  - Add responsive container with proper padding and max-width
  - **Visual Verification**: Use Chrome browser MCP to take screenshot and verify header layout structure
  - _Requirements: 1.1, 1.4, 1.5_

- [x] 2. Create logo and branding section (left)
  - [x] 2.1 Implement logo/branding area in header left section
    - Add platform title with modern typography
    - Position branding elements with proper spacing
    - Implement responsive text sizing for different screen sizes
    - **Visual Verification**: Take screenshot to verify logo positioning and typography
    - _Requirements: 1.2_

  - [x] 2.2 Add responsive branding behavior
    - Show full branding on desktop screens
    - Implement condensed version for mobile devices
    - Maintain consistent left alignment across breakpoints
    - **Visual Verification**: Test responsive behavior at different screen sizes using browser resize
    - _Requirements: 1.5_

- [-] 3. Implement two-row Airbnb-style filter navigation (center)
  - [x] 3.1 Create two-row filter navigation structure
    - Build two-row container with top row for category filters and bottom row for category-specific filters
    - Implement proper spacing and alignment between rows
    - Add horizontal scrolling support for both rows independently
    - Ensure proper responsive behavior for two-row layout
    - **Visual Verification**: Take screenshot and verify two-row structure with proper spacing
    - _Requirements: 2.1, 2.3_

  - [x] 3.2 Implement top row category filter buttons
    - Create pill-shaped category filter buttons (All, Events, Activities, Venues) in top row using Tailwind CSS
    - Implement active/inactive visual states with prominent styling for selected category
    - Add hover and focus states with proper transitions
    - Ensure minimum 44px touch target size for accessibility
    - Position buttons in top row with horizontal scrolling support
    - **Visual Verification**: Take screenshot of top row category filters and test hover/active states
    - _Requirements: 2.2, 2.4_

  - [x] 3.3 Create dynamic bottom row filter system
    - Implement bottom row that changes content based on selected top row category
    - Create category-specific filter sets (All: search/dates/age/price, Events: search/dates/type/age, etc.)
    - Add smooth transition animations when bottom row content changes
    - Ensure proper spacing and alignment with top row
    - **Visual Verification**: Test category selection and verify bottom row content updates correctly
    - _Requirements: 2.3, 5.1, 5.2, 5.3, 5.4_

  - [x] 3.4 Implement expandable search filter in bottom row
    - Create search filter button in bottom row that expands to show search input
    - Implement expansion behavior within bottom row space while keeping top row visible
    - Add smooth transition animation between collapsed and expanded states
    - Include close button to return to other bottom row filters
    - Ensure search expansion doesn't affect top row category filters
    - **Visual Verification**: Test search filter expansion within bottom row and verify top row remains visible
    - _Requirements: 3.1, 3.2, 3.3_

  - [x] 3.5 Add search functionality with category-specific behavior
    - Implement search input with dynamic placeholder text based on selected top row category
    - Add real-time activity filtering based on search query within selected category context
    - Include debounced search to optimize performance
    - Connect search results to existing activity display logic with category filtering
    - Ensure search operates within the context of selected category filter
    - **Visual Verification**: Test search functionality across different categories and verify category-specific filtering
    - _Requirements: 3.4, 3.5, 5.1, 5.2, 5.3, 5.4_

  - [x] 3.6 Implement expandable date filter in bottom row
    - Create date filter button in bottom row that expands to show date selection interface
    - Implement expansion behavior within bottom row space while keeping top row visible
    - Add date range picker with start and end date inputs
    - Include quick date options (Today, This Weekend, etc.)
    - Ensure date expansion doesn't affect top row category filters
    - **Visual Verification**: Test date filter expansion within bottom row and verify top row remains visible
    - _Requirements: 4.1, 4.2, 4.3_

  - [x] 3.7 Add date filtering functionality with category context
    - Implement date range validation and formatting
    - Add activity filtering based on selected date criteria within selected category context
    - Display selected date range in collapsed date filter button in bottom row
    - Connect date filtering to existing activity display logic with category filtering
    - Ensure date filtering operates within the context of selected category filter
    - **Visual Verification**: Test date filtering across different categories and verify category-specific results update
    - _Requirements: 4.4, 4.5, 5.4, 5.5_

  - [x] 3.8 Implement comprehensive two-row filter state management
    - Implement JavaScript logic for managing top row category filters and bottom row specific filters
    - Handle bottom row content updates when top row category selection changes
    - Manage bottom row filter expansion states (none, search, date expanded) while maintaining top row visibility
    - Update visual states when filters are clicked or changed in either row
    - Maintain filter state persistence across interactions including both row contexts
    - Ensure top row category filter remains active during bottom row filter usage
    - **Visual Verification**: Test all filter combinations across both rows and verify comprehensive state management
    - _Requirements: 2.4, 2.5, 3.4, 4.5, 5.5_

- [ ] 4. Create action buttons section (right)
  - [ ] 4.1 Implement action buttons area
    - Position action buttons in header right section
    - Style admin dashboard link with consistent design
    - Add proper spacing and alignment for button group
    - **Visual Verification**: Take screenshot to verify action button positioning and styling
    - _Requirements: 1.3_

  - [ ] 4.2 Add responsive action button behavior
    - Ensure buttons remain accessible on all screen sizes
    - Implement proper touch targets for mobile
    - Add hover and focus states for better UX
    - **Visual Verification**: Test responsive behavior and button interactions at different screen sizes
    - _Requirements: 1.5_

- [ ] 5. Update detail page header consistency
  - [ ] 5.1 Apply new header structure to detail page
    - Implement same three-section header layout on detail page
    - Ensure consistent branding and styling across pages
    - Add back navigation button in appropriate section
    - **Visual Verification**: Navigate to detail page and take screenshot to verify header consistency
    - _Requirements: 1.1_

  - [ ] 5.2 Adapt header for detail page context
    - Hide or modify two-row filter section for detail page context
    - Ensure proper navigation flow between index and detail pages
    - Maintain header consistency while adapting to page needs
    - Consider showing simplified navigation or breadcrumbs in filter area
    - **Visual Verification**: Test navigation between index and detail pages, verify header adaptation
    - _Requirements: 1.1_

- [ ] 6. Implement responsive design and mobile optimizations
  - [ ] 6.1 Add mobile-specific two-row header optimizations
    - Optimize header height for mobile screens while accommodating two-row filter layout
    - Ensure all interactive elements in both rows meet touch target requirements (minimum 44px)
    - Test and refine horizontal scrolling for both top and bottom rows on touch devices
    - Implement proper spacing between rows for touch interaction
    - Consider stacking rows vertically on very small screens if needed
    - **Visual Verification**: Resize browser to mobile width and take screenshots to verify two-row mobile optimizations
    - _Requirements: 6.1, 6.2_

  - [ ] 6.2 Test and refine two-row responsive behavior
    - Verify two-row header layout at all major breakpoints (320px, 768px, 1024px, 1440px)
    - Ensure proper spacing and alignment between rows across screen sizes
    - Test both row scrolling, bottom row content changes, and filter expansion on various devices
    - Verify category-specific bottom row updates work properly at all breakpoints
    - **Visual Verification**: Test and screenshot two-row layout at each breakpoint (320px, 768px, 1024px, 1440px)
    - _Requirements: 6.3, 6.4, 6.5_

- [ ] 7. Add accessibility and performance enhancements
  - [ ] 7.1 Implement accessibility improvements for two-row layout
    - Add proper ARIA labels for all filter navigation elements in both rows (category, search, date)
    - Ensure keyboard navigation works for both top and bottom row elements including expanded filters
    - Implement proper focus management between rows and during bottom row content changes
    - Test screen reader compatibility for two-row header structure and filter interactions
    - Verify color contrast meets WCAG AA standards for all filter states in both rows
    - **Visual Verification**: Test keyboard navigation across both rows and take screenshots of focus states
    - _Requirements: 1.1, 2.5, 3.4, 4.5_

  - [ ] 7.2 Optimize performance and interactions for two-row layout
    - Implement smooth transitions and micro-animations for bottom row content changes and filter expansion/collapse
    - Optimize scroll performance for both top and bottom row navigation
    - Add debounced search to prevent excessive filtering operations
    - Test and optimize for minimal layout shift during filter state changes and row transitions
    - Ensure smooth category switching with bottom row content updates
    - **Visual Verification**: Record or observe smooth transitions and animations for both rows in browser
    - _Requirements: 2.3, 2.4, 3.2, 4.2_