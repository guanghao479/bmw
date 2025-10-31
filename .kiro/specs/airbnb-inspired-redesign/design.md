# Design Document

## Overview

This design document outlines the Airbnb-inspired redesign of the Seattle Family Activities Platform header and filter system. The design focuses on creating a modern, consistent header layout with three distinct sections: logo/branding (left), filter navigation (center), and action buttons (right). The design will be implemented using Tailwind CSS and maintain full responsiveness across all device sizes.

## Architecture

### Header Layout Structure

The header will follow a three-column layout pattern inspired by Airbnb's design:

```
┌─────────────────────────────────────────────────────────────┐
│ [Logo/Brand]    [Filter Navigation]    [Action Buttons]    │
└─────────────────────────────────────────────────────────────┘
```

### Component Hierarchy

1. **Header Container**: Main wrapper with consistent padding and responsive behavior
2. **Left Section**: Logo and platform branding
3. **Center Section**: Horizontal scrollable filter navigation
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

**Purpose**: Category-based activity filtering

**Design Pattern**: Horizontal scrollable pill buttons similar to Airbnb

**Elements**:
- Pill-shaped filter buttons with rounded corners
- Clear active/inactive states with color and background changes
- Smooth horizontal scrolling with touch support
- Scroll indicators or fade effects at edges

**Visual States**:
- **Inactive**: Light background, dark text, subtle border
- **Active**: Dark background, white text, no border
- **Hover**: Subtle background color change and elevation
- **Focus**: Clear focus ring for accessibility

**Responsive Behavior**:
- Horizontal scroll on all screen sizes
- Touch-friendly button sizing (minimum 44px height)
- Proper spacing between buttons

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
  activeFilter: string,
  availableFilters: [
    { id: 'all', label: 'All', count: number },
    { id: 'events', label: 'Events', count: number },
    { id: 'activities', label: 'Activities', count: number },
    { id: 'venues', label: 'Venues', count: number }
  ],
  scrollPosition: number
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

**Header Container**:
```html
<header class="sticky top-0 z-50 bg-white border-b border-gray-200 shadow-sm">
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
    <div class="flex items-center justify-between h-16">
      <!-- Three sections here -->
    </div>
  </div>
</header>
```

**Filter Button Styling**:
```html
<button class="inline-flex items-center px-4 py-2 rounded-full text-sm font-medium transition-all duration-200 
               bg-gray-100 text-gray-700 hover:bg-gray-200 
               data-[active]:bg-gray-900 data-[active]:text-white
               focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2">
  Filter Name
</button>
```

### JavaScript Interactions

**Filter Scrolling**:
- Implement smooth horizontal scrolling with touch support
- Add scroll indicators or fade effects at container edges
- Maintain scroll position during filter changes

**State Management**:
- Update URL parameters when filters change
- Persist filter state across page navigation
- Provide visual feedback for active filters

### Mobile Optimizations

- Reduce header height on mobile for more content space
- Implement touch-friendly filter button sizing
- Consider collapsible filter section for very small screens
- Optimize scroll performance for touch devices