---
inclusion: manual
---

# AI-Friendly Implementation Reference

## Quick Component Reference

### Glassmorphic Components

#### .glass-card
```css
.glass-card {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.25) 0%, rgba(255, 255, 255, 0.1) 100%);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: var(--radius-2xl);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1), inset 0 1px 0 rgba(255, 255, 255, 0.4);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
```

#### .glass-modal-backdrop
```css
.glass-modal-backdrop {
  background: rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  z-index: 1000;
}
```

#### .search-container-glass
```css
.search-container-glass {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.4) 0%, rgba(255, 255, 255, 0.2) 100%);
  backdrop-filter: blur(25px);
  -webkit-backdrop-filter: blur(25px);
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: var(--radius-2xl);
  padding: var(--space-5);
}
```

### Neumorphic Components

#### .neuro-button
```css
.neuro-button {
  background: var(--neuro-base);
  border: none;
  border-radius: var(--radius-xl);
  padding: var(--space-3) var(--space-6);
  min-height: var(--touch-target-min);
  box-shadow: 6px 6px 12px var(--neuro-shadow-dark), -6px -6px 12px var(--neuro-shadow-light);
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}
```##
## .neuro-input
```css
.neuro-input {
  background: var(--neuro-base);
  border: none;
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  font-size: var(--font-size-base);
  min-height: var(--touch-target-min);
  box-shadow: inset 4px 4px 8px var(--neuro-shadow-dark), inset -4px -4px 8px var(--neuro-shadow-light);
}
```

#### .date-tabs-neuro-container
```css
.date-tabs-neuro-container {
  background: var(--neuro-light);
  border-radius: var(--radius-2xl);
  padding: var(--space-3);
  box-shadow: inset 8px 8px 16px var(--neuro-shadow-dark), inset -8px -8px 16px var(--neuro-shadow-light);
}
```

### Hybrid Components

#### .activity-card-hybrid
```css
.activity-card-hybrid {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.3) 0%, rgba(255, 255, 255, 0.15) 100%);
  backdrop-filter: blur(15px);
  -webkit-backdrop-filter: blur(15px);
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: var(--radius-2xl);
  min-height: var(--touch-target-comfortable);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}
```

## CSS Custom Properties (Design Tokens)

### Colors
```css
/* Glassmorphic Colors */
--glass-primary: rgba(99, 102, 241, 0.15);
--glass-neutral: rgba(255, 255, 255, 0.25);
--glass-backdrop: rgba(255, 255, 255, 0.8);
--glass-border: rgba(255, 255, 255, 0.2);

/* Neumorphic Colors */
--neuro-light: #f8fafc;
--neuro-base: #f1f5f9;
--neuro-dark: #e2e8f0;
--neuro-shadow-light: rgba(255, 255, 255, 0.9);
--neuro-shadow-dark: rgba(148, 163, 184, 0.4);

/* Brand Colors */
--primary: #6366f1;
--primary-light: #818cf8;
--primary-dark: #4f46e5;
--secondary: #ec4899;
--accent: #14b8a6;
```

### Spacing & Sizing
```css
/* Touch Targets */
--touch-target-min: 44px;
--touch-target-comfortable: 48px;

/* Spacing Scale */
--space-1: 4px; --space-2: 8px; --space-3: 12px; --space-4: 16px;
--space-5: 20px; --space-6: 24px; --space-8: 32px; --space-10: 40px;

/* Border Radius */
--radius-sm: 4px; --radius-md: 8px; --radius-lg: 12px;
--radius-xl: 16px; --radius-2xl: 24px; --radius-full: 9999px;

/* Breakpoints */
--breakpoint-sm: 480px; --breakpoint-md: 768px;
--breakpoint-lg: 1024px; --breakpoint-xl: 1400px;
```

## Component Patterns

### Pattern 1: Glass Container with Neuro Controls
```html
<div class="glass-card p-6 rounded-2xl">
  <h3 class="text-lg font-semibold mb-4">Form Title</h3>
  <div class="space-y-4">
    <input type="text" class="neuro-input w-full" placeholder="Input field">
    <button class="neuro-button neuro-button--primary w-full">Submit</button>
  </div>
</div>
```

### Pattern 2: Modal Dialog
```html
<div class="glass-modal-backdrop fixed inset-0 z-50 flex items-center justify-center p-4">
  <div class="glass-modal-content w-full max-w-2xl">
    <div class="p-6">
      <h2 class="text-xl font-bold mb-4">Modal Title</h2>
      <div class="flex justify-end gap-3">
        <button class="neuro-button neuro-button--secondary">Cancel</button>
        <button class="neuro-button neuro-button--primary">Confirm</button>
      </div>
    </div>
  </div>
</div>
```

### Pattern 3: Activity Card Grid
```html
<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
  <div class="activity-card-hybrid cursor-pointer" tabindex="0">
    <div class="activity-card-hybrid__content p-4">
      <h3 class="font-semibold mb-2">Activity Title</h3>
      <p class="text-sm text-gray-600 mb-3">Description</p>
      <button class="activity-card-hybrid__action-button">Learn More</button>
    </div>
  </div>
</div>
```

### Pattern 4: Search Interface
```html
<div class="search-container-glass mb-6">
  <div class="flex gap-3">
    <input type="text" class="search-input-glass flex-1" placeholder="Search...">
    <button class="neuro-button neuro-button--primary">Search</button>
  </div>
</div>
```

### Pattern 5: Date Navigation
```html
<div class="date-tabs-neuro-container">
  <div class="date-tabs-neuro flex gap-2 overflow-x-auto">
    <button class="date-tab-neuro active">Today <span class="count">5</span></button>
    <button class="date-tab-neuro">Tomorrow <span class="count">3</span></button>
    <button class="date-tab-neuro weekend">Sat <span class="count">8</span></button>
  </div>
</div>
```

## Accessibility Requirements

### Mandatory ARIA Attributes
```html
<!-- Buttons -->
<button class="neuro-button" aria-label="Descriptive action">Button</button>

<!-- Form Controls -->
<input class="neuro-input" aria-describedby="help-text" aria-label="Field name">
<div id="help-text">Helper text</div>

<!-- Modals -->
<div class="glass-modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="modal-title">
  <h2 id="modal-title">Modal Title</h2>
</div>

<!-- Cards -->
<div class="activity-card-hybrid" tabindex="0" role="button" aria-label="View activity details">
```

### Required Focus Indicators
```css
.glass-card:focus-visible,
.neuro-button:focus-visible,
.neuro-input:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}
```

### Screen Reader Support
```html
<span class="sr-only">Screen reader only text</span>
<button aria-label="Close dialog"><span aria-hidden="true">×</span></button>
```

## Responsive Design Patterns

### Mobile-First Grid
```html
<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
  <div class="glass-card p-4 sm:p-6">Content</div>
</div>
```

### Responsive Typography
```html
<h1 class="text-2xl sm:text-3xl lg:text-4xl font-bold">Title</h1>
<p class="text-sm sm:text-base lg:text-lg">Body text</p>
```

### Touch-Optimized Controls
```html
<button class="neuro-button px-4 py-3 sm:px-6 sm:py-2 text-base sm:text-sm">
  Mobile-friendly button
</button>
```

## Browser Fallbacks

### Backdrop Filter Fallback
```css
.glass-card {
  background: rgba(255, 255, 255, 0.9); /* Fallback */
}

@supports (backdrop-filter: blur(20px)) {
  .glass-card {
    backdrop-filter: blur(20px);
    background: linear-gradient(135deg, rgba(255, 255, 255, 0.25), rgba(255, 255, 255, 0.1));
  }
}
```

### High Contrast Mode
```css
@media (prefers-contrast: high) {
  .glass-card, .activity-card-hybrid {
    background: var(--bg-primary);
    border: 2px solid var(--text-primary);
    backdrop-filter: none;
  }
  .neuro-button {
    background: var(--bg-primary);
    border: 2px solid var(--text-primary);
    box-shadow: none;
  }
}
```

### Reduced Motion
```css
@media (prefers-reduced-motion: reduce) {
  .glass-card, .neuro-button, .activity-card-hybrid {
    transition: none;
    animation: none;
  }
  .glass-card:hover, .activity-card-hybrid:hover {
    transform: none;
  }
}
```

### Touch Device Optimization
```css
@media (hover: none) and (pointer: coarse) {
  .glass-card:hover, .neuro-button:hover {
    transform: none;
  }
  .neuro-button, .date-tab-neuro {
    min-height: var(--touch-target-min);
    min-width: var(--touch-target-min);
  }
}
```

## Common Implementation Issues

### Issue 1: Backdrop Filter Not Working
**Problem**: Glassmorphic effects not appearing
**Solution**: 
```css
/* Add webkit prefix and fallback */
.glass-card {
  background: rgba(255, 255, 255, 0.9); /* Fallback */
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px); /* Safari */
}
```

### Issue 2: Touch Targets Too Small
**Problem**: Buttons difficult to tap on mobile
**Solution**:
```css
.neuro-button {
  min-height: 44px; /* Minimum touch target */
  min-width: 44px;
  padding: 12px 24px; /* Adequate padding */
}
```

### Issue 3: Poor Contrast in Glass Elements
**Problem**: Text hard to read on glassmorphic backgrounds
**Solution**:
```css
.glass-card {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.4), rgba(255, 255, 255, 0.2));
  /* Increase opacity for better contrast */
}
```

### Issue 4: Neumorphic Shadows Not Visible
**Problem**: Shadows too subtle or wrong colors
**Solution**:
```css
.neuro-button {
  box-shadow: 
    6px 6px 12px rgba(148, 163, 184, 0.4), /* Darker shadow */
    -6px -6px 12px rgba(255, 255, 255, 0.9); /* Lighter highlight */
}
```

### Issue 5: Animation Performance Issues
**Problem**: Laggy animations on mobile devices
**Solution**:
```css
.glass-card {
  will-change: transform; /* Optimize for animations */
  transform: translateZ(0); /* Force hardware acceleration */
}
```

## Quick Implementation Checklist

### For Glass Components:
- [ ] Add backdrop-filter with webkit prefix
- [ ] Include fallback background color
- [ ] Use rgba colors with appropriate opacity
- [ ] Add subtle border with low opacity
- [ ] Include smooth transitions
- [ ] Test in Safari and Firefox

### For Neuro Components:
- [ ] Use inset shadows for inputs
- [ ] Use outset shadows for buttons
- [ ] Ensure minimum 44px touch targets
- [ ] Add hover and active states
- [ ] Include focus indicators
- [ ] Test shadow visibility

### For Hybrid Components:
- [ ] Combine glass base with neuro controls
- [ ] Maintain visual hierarchy
- [ ] Ensure accessibility compliance
- [ ] Test responsive behavior
- [ ] Verify touch interactions
- [ ] Check keyboard navigation

### For All Components:
- [ ] Add ARIA labels where needed
- [ ] Include focus indicators
- [ ] Test with screen readers
- [ ] Verify color contrast ratios
- [ ] Test on mobile devices
- [ ] Check reduced motion support
- [ ] Validate HTML semantics

## Performance Optimization

### CSS Optimization
```css
/* Use will-change sparingly */
.glass-card:hover {
  will-change: transform;
}

/* Remove will-change after animation */
.glass-card {
  will-change: auto;
}

/* Use transform instead of changing layout properties */
.neuro-button:hover {
  transform: translateY(-1px); /* Better than changing top/bottom */
}
```

### JavaScript Optimization
```javascript
// Use passive event listeners for scroll
element.addEventListener('scroll', handler, { passive: true });

// Debounce resize events
const debouncedResize = debounce(() => {
  // Handle resize
}, 100);
window.addEventListener('resize', debouncedResize);
```

This reference provides AI assistants with structured, actionable information for implementing the design system components correctly and efficiently.