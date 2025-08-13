# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Plan & Execution
### Before starting work
- Always in plan mode to make a plan
- After get the plan, make sure you Write the plan to docs/tasks/TASK_NAME.md.
- The plan should be a detailed implementation plan and the reasoning behind them, as well as tasks broken down.
- If the task require external knowledge or certain package, also research to get latest knowledge (Use Task tool for research)
- Don't over plan it, always think MVP.
- Once you write the plan, firstly ask me to review it. Do not continue until I approve the plan.

### While implementing
- You should update the plan as you work.
- After you complete tasks in the plan, you should update and append detailed descriptions of the changes you made, so following tasks can be easily hand over to other engineers.

### After Implementation
- You should ask user to review then proceed to commit and push the changes with git

## Project Overview

This is a static family events web application that displays local events, activities, and venues in a modern, responsive interface. The application is built with vanilla HTML, CSS, and JavaScript without any build tools or frameworks.

## Architecture

- **Static single-page application**: No server-side rendering or backend API
- **Data embedding**: Event data is embedded directly in the JavaScript file (`script.js`) rather than loaded from `data.json`
- **Class-based structure**: Main functionality is encapsulated in the `FamilyEventsApp` class
- **Modern CSS**: Uses CSS custom properties (variables) for theming and design system
- **Responsive design**: Mobile-first approach with breakpoints at 480px, 768px, and 1200px

## Key Components

### Data Structure
Events are categorized into three types: `events`, `activities`, and `venues`. Each item has:
- Basic info: id, title, description, category
- Display: image URL (Unsplash), featured status
- Details: date, time, location, price, age_range

### JavaScript Architecture
- `FamilyEventsApp` class handles all interactions
- Data is embedded in the `loadData()` method (not loaded from external JSON)
- Real-time filtering by category and search term
- Modal system for detailed item views
- Intersection Observer for scroll animations

### CSS Design System
- CSS custom properties for colors, spacing, typography
- Component-based class naming (`.card`, `.filter-btn`, etc.)
- Glassmorphism effects and modern shadows
- Accessibility focus states included

## Development

### Running the Application
Open `index.html` in a web browser - no build process required.

### Making Changes
- **Content updates**: Modify the embedded data object in `script.js:21-197`
- **Styling**: Update CSS custom properties in `styles.css:2-90` for global changes
- **New features**: Extend the `FamilyEventsApp` class methods

### Image Handling
- Uses Unsplash URLs with automatic fallback to SVG placeholder
- Fallback implemented via `onerror` attribute in `script.js:309`
- Images are lazy-loaded with `loading="lazy"`

## Technical Notes

- No package.json or build tools - this is intentional for simplicity
- Data is duplicated between `data.json` and `script.js` - the JavaScript version is used
- Modal styles are injected dynamically when first needed
- Uses modern JavaScript features (async/await, arrow functions, template literals)