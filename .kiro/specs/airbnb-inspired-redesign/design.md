# Design Document

## Overview

This design document outlines the Airbnb-inspired redesign of the Seattle Family Activities Platform header and filter system. The design focuses on creating a modern, consistent header layout with three distinct sections: logo/branding (left), filter navigation (center), and action buttons (right). The design will be implemented using Tailwind CSS and maintain full responsiveness across all device sizes.

## Architecture

### Header Layout Structure

The header will follow a three-column layout pattern inspired by Airbnb's design with a two-row filter system:

```
┌─────────────────────────────────────────────────────────────┐
│ [Logo/Brand]    [Two-Row Filter Navigation]    [Actions]   │
│                 ┌─ Top Row: Category Filters ─┐            │
│                 │ All | Events | Activities   │            │
│                 ├─ Bottom Row: Specific Filters ─┤         │
│                 │ Search | Dates | Price       │            │
│                 └─────────────────────────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

### Component Hierarchy

1. **Header Container**: Main wrapper with consistent padding and responsive behavior
2. **Left Section**: Logo and platform branding
3. **Center Section**: Two-row filter navigation with category filters (top) and category-specific filters (bottom)
4. **Right Section**: Action buttons (admin, menu, etc.)

## Components and Interfaces

### Header Component

**Purpose**: Consistent navigation header across all pages

**Layout**:
- Fixed height header with white background and subtle shadow
- Three-section flexbox layout with proper spacing
- Responsive behavior that adapts to mobile, tablet, and desktop

**Visual Design**:
- Clean white background with subtle border/shadow
- Consistent padding and margins using Tailwind spacing scale
- Modern typography with proper font weights and sizes

### Logo Section (Left)

**Purpose**: Platform branding and identity

**Elements**:
- Platform name/logo with modern typography
- Consistent branding colors
- Optional tagline for larger screens

**Responsive Behavior**:
- Full branding on desktop
- Condensed logo on mobile
- Maintains consistent left alignment

### Filter Navigation (Center)

**Purpose**: Comprehensive activity filtering with two-row layout similar to Airbnb's experience pages

**Design Pattern**: Two-row filter system with category selection (top row) and category-specific filters (bottom row)

**Two-Row Structure**:
- **Top Row**: Category filter buttons (All, Events, Activities, Venues) - always visible
- **Bottom Row**: Category-specific filters that change based on selected category
- Both rows support horizontal scrolling with touch support
- Smooth transitions when bottom row content changes

**Core Elements**:
- **Top Row Elements**: Category filter buttons with pill-shaped design
- **Bottom Row Elements**: Dynamic filters including search, date, price, age group, etc.
- Scroll indicators or fade effects at edges for both rows
- Consistent visual design between both rows

**Two-Row Filter Behavior**:
- **Top Row (Category Filters)**: Always visible, maintains selected state
- **Bottom Row (Category-Specific Filters)**: Content changes based on selected category
- **Filter Expansion**: When search or date filters expand, they expand within the bottom row space
- **Category Persistence**: Top row selection remains active during bottom row interactions
- **Smooth Transitions**: Animated content changes when switching categories

**Category-Specific Bottom Row Content**:
- **All Category**: Search, Dates, Age Group, Price filters
- **Events Category**: Search, Dates, Event Type, Age Group filters  
- **Activities Category**: Search, Dates, Activity Type, Duration filters
- **Venues Category**: Search, Location, Amenities, Age Group filters

**Visual States**:
- **Inactive**: Light background, dark text, subtle border
- **Active**: Dark background, white text, no border
- **Expanded**: Full-width input or picker interface with close button
- **Hover**: Subtle background color change and elevation
- **Focus**: Clear focus ring for accessibility

**Search Filter Component (Bottom Row)**:
- Collapsed: "Search" pill button with search icon in bottom row
- Expanded: Full-width search input within bottom row space, category-specific placeholder text
- Real-time filtering as user types within selected category context
- Clear button to reset search while maintaining category selection
- Close button to return to other bottom row filters
- Top row (category filters) remains visible during search expansion

**Date Filter Component (Bottom Row)**:
- Collapsed: "Dates" pill button in bottom row showing selected range or "Any date"
- Expanded: Date picker interface within bottom row space (calendar or date range selector)
- Quick date options (Today, This Weekend, This Week)
- Clear button to reset date filter
- Close button to return to other bottom row filters
- Top row (category filters) remains visible during date expansion

**Responsive Behavior**:
- Both rows support horizontal scroll on all screen sizes
- Touch-friendly button sizing (minimum 44px height) for both rows
- Expanded filters adapt to available screen width within bottom row
- Mobile-optimized date picker and search input
- Consistent spacing and alignment between top and bottom rows
- Stack rows vertically on very small screens if needed

### Action Buttons (Right)

**Purpose**: User actions and navigation

**Elements**:
- Admin dashboard link
- Potential user menu or settings
- Search toggle for mobile

**Visual Design**:
- Consistent button styling with filter buttons
- Icon-based design for space efficiency
- Clear hover and focus states

## Data Models

### Filter State Management

```javascript
{
  // Top Row (Category Filters)
  activeCategory: string, // 'all', 'events', 'activities', 'venues'
  categoryFilters: [
    { id: 'all', label: 'All', count: number },
    { id: 'events', label: 'Events', count: number },
    { id: 'activities', label: 'Activities', count: number },
    { id: 'venues', label: 'Venues', count: number }
  ],
  
  // Bottom Row (Category-Specific Filters)
  bottomRowFilters: {
    all: ['search', 'dates', 'ageGroup', 'price'],
    events: ['search', 'dates', 'eventType', 'ageGroup'],
    activities: ['search', 'dates', 'activityType', 'duration'],
    venues: ['search', 'location', 'amenities', 'ageGroup']
  },
  
  // Filter States
  searchState: {
    isExpanded: boolean,
    query: string,
    placeholder: string // Dynamic based on active category
  },
  dateState: {
    isExpanded: boolean,
    startDate: Date | null,
    endDate: Date | null,
    displayText: string
  },
  
  // UI State
  expandedBottomFilter: 'none' | 'search' | 'dates' | 'other',
  topRowScrollPosition: number,
  bottomRowScrollPosition: number
}
```

### Header Configuration

```javascript
{
  branding: {
    title: 'Seattle Family Activities',
    subtitle: 'Discover amazing activities',
    logo: string // optional
  },
  showFilters: boolean, // true for index, false for detail
  actionButtons: [
    { type: 'admin', icon: '⚙️', href: 'admin.html' },
    { type: 'back', icon: '←', action: 'goBack' } // for detail pages
  ]
}
```

## Error Handling

### Filter Navigation Errors

- **No filters available**: Show placeholder message
- **Filter loading failure**: Display retry option
- **Scroll mechanism failure**: Fallback to standard overflow scroll

### Responsive Layout Errors

- **Insufficient space**: Graceful degradation to mobile layout
- **Font loading failure**: Fallback to system fonts
- **CSS loading failure**: Maintain functional layout with browser defaults

## Testing Strategy

### Visual Testing

1. **Cross-browser compatibility**: Test header layout in Chrome, Firefox, Safari, Edge
2. **Responsive design**: Verify layout at breakpoints (320px, 768px, 1024px, 1440px)
3. **Filter interaction**: Test scrolling, clicking, and keyboard navigation
4. **State management**: Verify active filter persistence and visual feedback

### Accessibility Testing

1. **Keyboard navigation**: Ensure all interactive elements are keyboard accessible
2. **Screen reader compatibility**: Test with ARIA labels and semantic HTML
3. **Focus management**: Verify clear focus indicators and logical tab order
4. **Color contrast**: Ensure all text meets WCAG AA standards

### Performance Testing

1. **Layout shift**: Minimize CLS during header rendering
2. **Scroll performance**: Ensure smooth filter navigation scrolling
3. **Touch responsiveness**: Test filter button interactions on mobile devices

## Implementation Details

### Tailwind CSS Classes

**Header Container with Two-Row Filter**:
```html
<header class="sticky top-0 z-50 bg-white border-b border-gray-200 shadow-sm">
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
    <div class="flex items-center justify-between h-20"> <!-- Increased height for two rows -->
      <!-- Left: Logo -->
      <div class="flex-shrink-0">
        <!-- Logo content -->
      </div>
      
      <!-- Center: Two-Row Filter Navigation -->
      <div class="flex-1 max-w-2xl mx-8">
        <!-- Top Row: Category Filters -->
        <div class="flex items-center space-x-2 mb-2 overflow-x-auto scrollbar-hide">
          <!-- Category filter buttons -->
        </div>
        
        <!-- Bottom Row: Category-Specific Filters -->
        <div class="flex items-center space-x-2 overflow-x-auto scrollbar-hide">
          <!-- Dynamic category-specific filters -->
        </div>
      </div>
      
      <!-- Right: Action Buttons -->
      <div class="flex-shrink-0">
        <!-- Action buttons -->
      </div>
    </div>
  </div>
</header>
```

**Filter Button Styling (Both Rows)**:
```html
<!-- Top Row Category Filter Button -->
<button class="inline-flex items-center px-4 py-2 rounded-full text-sm font-medium transition-all duration-200 
               bg-gray-100 text-gray-700 hover:bg-gray-200 
               data-[active]:bg-gray-900 data-[active]:text-white
               focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2
               whitespace-nowrap">
  Category Name
</button>

<!-- Bottom Row Specific Filter Button -->
<button class="inline-flex items-center px-3 py-1.5 rounded-full text-xs font-medium transition-all duration-200 
               bg-gray-50 text-gray-600 hover:bg-gray-100 border border-gray-200
               data-[active]:bg-blue-50 data-[active]:text-blue-700 data-[active]:border-blue-200
               focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
               whitespace-nowrap">
  Filter Name
</button>
```

**Expanded Search Filter (Bottom Row)**:
```html
<!-- Expanded search takes full width of bottom row -->
<div class="flex items-center w-full bg-white border border-gray-300 rounded-full shadow-sm">
  <input type="text" 
         class="flex-1 px-4 py-1.5 text-xs bg-transparent border-0 rounded-l-full focus:outline-none focus:ring-0"
         placeholder="Search events..."> <!-- Dynamic placeholder based on category -->
  <button class="px-3 py-1.5 text-gray-400 hover:text-gray-600">
    <svg class="w-3 h-3"><!-- search icon --></svg>
  </button>
  <button class="px-3 py-1.5 text-gray-400 hover:text-gray-600 border-l border-gray-200">
    <svg class="w-3 h-3"><!-- close icon --></svg>
  </button>
</div>
```

**Expanded Date Filter (Bottom Row)**:
```html
<!-- Expanded date picker takes full width of bottom row -->
<div class="flex items-center space-x-2 bg-white border border-gray-300 rounded-full shadow-sm px-4 py-1.5">
  <span class="text-xs text-gray-700">Start</span>
  <input type="date" class="text-xs border-0 bg-transparent focus:outline-none">
  <span class="text-gray-300">—</span>
  <span class="text-xs text-gray-700">End</span>
  <input type="date" class="text-xs border-0 bg-transparent focus:outline-none">
  <button class="ml-2 text-gray-400 hover:text-gray-600">
    <svg class="w-3 h-3"><!-- close icon --></svg>
  </button>
</div>
```

### JavaScript Interactions

**Two-Row Filter Scrolling**:
- Implement smooth horizontal scrolling with touch support for both rows
- Add scroll indicators or fade effects at container edges for each row
- Maintain independent scroll positions for top and bottom rows
- Handle scroll behavior during bottom row content changes
- Sync scroll behavior when switching between categories

**Two-Row Filter Logic**:
- Track active category in top row (All, Events, Activities, Venues)
- Update bottom row content based on selected category
- Track which bottom row filter is currently expanded (none, search, dates, etc.)
- Animate transition between collapsed and expanded states within bottom row
- Handle click outside to collapse expanded bottom row filters
- Manage focus states during expansion/collapse while keeping top row accessible

**State Management**:
- Update URL parameters when filters change
- Persist filter state including active category, search query and date range
- Provide visual feedback for active category filters
- Sync filter state with activity display logic
- Handle real-time search filtering within selected category context
- Maintain category filter selection when using search or date filters
- Manage date range validation and formatting

**Search Functionality**:
- Implement debounced search to avoid excessive API calls
- Filter activities by title, description, and venue name based on selected category
- Scope search results to active category filter (Events, Activities, Venues, or All)
- Highlight search terms in results
- Provide search suggestions or autocomplete

**Date Filtering**:
- Support single date and date range selection
- Provide quick date options (Today, Tomorrow, This Weekend)
- Apply date filtering within the context of selected category filter
- Handle date validation and range constraints
- Format date display for different locales

### Category-Specific Filtering Logic

**Two-Row Filter Interaction Behavior**:
- **Top Row Selection**: When category is selected, bottom row updates with relevant filters
- **All Category**: Bottom row shows general filters (Search, Dates, Age Group, Price)
- **Events Category**: Bottom row shows event-specific filters (Search, Dates, Event Type, Age Group)
- **Activities Category**: Bottom row shows activity-specific filters (Search, Dates, Activity Type, Duration)
- **Venues Category**: Bottom row shows venue-specific filters (Search, Location, Amenities, Age Group)
- **Category Persistence**: Top row selection remains active during bottom row interactions
- **Dynamic Placeholders**: Search placeholder updates based on selected category

**Two-Row Content Filtering Pipeline**:
1. User selects category in top row (All/Events/Activities/Venues)
2. Bottom row updates with category-specific filters
3. User applies filters in bottom row (search, dates, etc.)
4. Apply category filter first, then bottom row filters within category context
5. Update display with filtered results
6. Maintain both top and bottom row filter states in URL and local storage

### Mobile Optimizations

- Maintain two-row structure on mobile with adjusted spacing
- Implement touch-friendly filter button sizing for both rows (minimum 44px touch targets)
- Consider stacking rows vertically on very small screens (< 320px)
- Optimize scroll performance for touch devices on both rows
- Ensure adequate spacing between top and bottom rows for touch interaction
- Implement momentum scrolling and scroll snap for better mobile experience
- Consider reducing bottom row filter button size slightly to fit more options