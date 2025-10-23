# Browser Testing Procedures

This guide provides comprehensive procedures for testing the Seattle Family Activities platform in a browser environment during local development.

## Overview

Browser testing ensures that:
- Frontend connects properly to the local backend
- All user interfaces work correctly
- API calls function as expected
- Error handling provides good user experience
- The application works across different browsers and devices

## Prerequisites

Before starting browser testing:

1. **Local development environment is running**:
   ```bash
   # Backend running on http://127.0.0.1:3000
   cd backend && ./start-local-backend.sh
   
   # Frontend running on http://localhost:8000
   ./start-local-frontend.sh
   ```

2. **Browser developer tools are available**:
   - Chrome DevTools (F12)
   - Firefox Developer Tools (F12)
   - Safari Web Inspector (Cmd+Option+I)

## Testing Procedures

### 1. Environment Detection Testing

**Purpose**: Verify the frontend correctly detects local development environment.

**Steps**:
1. Open http://localhost:8000 in browser
2. Open Developer Tools (F12)
3. Go to Console tab
4. Look for environment detection messages
5. Verify API endpoint configuration

**Expected Results**:
- Console shows local environment detected
- API calls go to http://127.0.0.1:3000/api
- No CORS errors in console

**Test Script**:
```javascript
// Run in browser console
console.log('Environment:', window.location.hostname);
console.log('API Endpoint:', config?.apiEndpoint);

// Test API connectivity
fetch('http://127.0.0.1:3000/api/sources')
  .then(response => response.json())
  .then(data => console.log('API Response:', data))
  .catch(error => console.error('API Error:', error));
```

### 2. Main Application Testing

**URL**: http://localhost:8000/

#### 2.1 Page Load Testing

**Steps**:
1. Navigate to http://localhost:8000/
2. Wait for page to fully load
3. Check for any console errors
4. Verify all UI elements are visible

**Expected Results**:
- Page loads without errors
- Search interface is visible
- Filter options are available
- No broken images or missing styles

#### 2.2 Data Loading Testing

**Steps**:
1. Open Network tab in Developer Tools
2. Refresh the page
3. Monitor API calls to backend
4. Check response data

**Expected Results**:
- API calls to http://127.0.0.1:3000/api/events
- Successful responses (200 status)
- Event data displayed in UI
- Loading states work correctly

**Test Cases**:
```javascript
// Test data loading
async function testDataLoading() {
    console.log('Testing data loading...');
    
    try {
        const response = await fetch('http://127.0.0.1:3000/api/events');
        const data = await response.json();
        
        console.log('Events loaded:', data.length || 0);
        console.log('Sample event:', data[0]);
        
        return data;
    } catch (error) {
        console.error('Data loading failed:', error);
        return null;
    }
}

testDataLoading();
```

#### 2.3 Search and Filter Testing

**Steps**:
1. Use the search input field
2. Try different search terms
3. Test filter options (activity type, age group, etc.)
4. Verify results update correctly

**Expected Results**:
- Search results filter correctly
- Filters work independently and in combination
- No JavaScript errors during filtering
- Results update without page refresh

### 3. Admin Interface Testing

**URL**: http://localhost:8000/admin.html

#### 3.1 Admin Page Load Testing

**Steps**:
1. Navigate to http://localhost:8000/admin.html
2. Check page loads completely
3. Verify all admin UI elements are present
4. Check console for errors

**Expected Results**:
- Admin interface loads successfully
- Source management section visible
- Add source form is functional
- No console errors

#### 3.2 Source Management Testing

**Steps**:
1. **List Sources**:
   - Check existing sources load
   - Verify source information displays correctly

2. **Add New Source**:
   - Fill out the add source form
   - Submit the form
   - Verify success message
   - Check source appears in list

3. **Edit Source**:
   - Click edit on existing source
   - Modify source information
   - Save changes
   - Verify updates are reflected

4. **Delete Source**:
   - Click delete on a source
   - Confirm deletion
   - Verify source is removed from list

**Test Data**:
```javascript
// Test source data
const testSource = {
    name: "Test Source",
    url: "https://example.com",
    category: "events",
    description: "Test source for browser testing"
};

// Add source via console
async function addTestSource() {
    const response = await fetch('http://127.0.0.1:3000/api/sources', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(testSource)
    });
    
    const result = await response.json();
    console.log('Source added:', result);
    return result;
}
```

#### 3.3 Scraping Testing

**Steps**:
1. Select a source for scraping
2. Click "Scrape Now" button
3. Monitor scraping progress
4. Check results when complete

**Expected Results**:
- Scraping request initiates successfully
- Progress indicators work
- Results are displayed when complete
- Any errors are handled gracefully

### 4. API Endpoint Testing

**Purpose**: Test all API endpoints directly through browser.

#### 4.1 Sources API Testing

```javascript
// Test all source operations
async function testSourcesAPI() {
    const baseUrl = 'http://127.0.0.1:3000/api';
    
    // GET sources
    console.log('Testing GET /api/sources...');
    let response = await fetch(`${baseUrl}/sources`);
    let sources = await response.json();
    console.log('Sources:', sources);
    
    // POST new source
    console.log('Testing POST /api/sources...');
    const newSource = {
        name: 'Browser Test Source',
        url: 'https://example.com/test',
        category: 'events'
    };
    
    response = await fetch(`${baseUrl}/sources`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newSource)
    });
    
    const createdSource = await response.json();
    console.log('Created source:', createdSource);
    
    return createdSource;
}

testSourcesAPI();
```

#### 4.2 Events API Testing

```javascript
// Test events API
async function testEventsAPI() {
    const baseUrl = 'http://127.0.0.1:3000/api';
    
    // GET events
    console.log('Testing GET /api/events...');
    const response = await fetch(`${baseUrl}/events`);
    const events = await response.json();
    
    console.log('Events count:', events.length || 0);
    console.log('Sample event:', events[0]);
    
    return events;
}

testEventsAPI();
```

#### 4.3 Scraping API Testing

```javascript
// Test scraping API
async function testScrapingAPI() {
    const baseUrl = 'http://127.0.0.1:3000/api';
    
    // Trigger scraping
    console.log('Testing POST /api/scrape...');
    const response = await fetch(`${baseUrl}/scrape`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            sourceId: 'test-source-id',
            immediate: true
        })
    });
    
    const result = await response.json();
    console.log('Scraping result:', result);
    
    return result;
}

// Note: Only run if you have test sources configured
// testScrapingAPI();
```

### 5. Error Handling Testing

#### 5.1 Backend Unavailable Testing

**Steps**:
1. Stop the backend server (Ctrl+C in backend terminal)
2. Refresh the frontend page
3. Try to perform actions that require API calls
4. Check error messages and user feedback

**Expected Results**:
- Clear error messages when backend is unavailable
- User-friendly feedback (not technical error codes)
- Application doesn't crash or become unusable
- Retry mechanisms work when backend comes back online

#### 5.2 Network Error Testing

**Steps**:
1. Open Developer Tools
2. Go to Network tab
3. Enable "Offline" mode or throttling
4. Try to use the application
5. Check error handling

**Expected Results**:
- Graceful handling of network errors
- Appropriate user feedback
- Application remains functional for cached data

### 6. Cross-Browser Testing

Test the application in multiple browsers:

#### 6.1 Chrome Testing
- Latest Chrome version
- Test all functionality
- Check console for Chrome-specific issues

#### 6.2 Firefox Testing
- Latest Firefox version
- Verify compatibility
- Check for Firefox-specific console errors

#### 6.3 Safari Testing (macOS)
- Latest Safari version
- Test WebKit-specific behavior
- Check for Safari console warnings

#### 6.4 Mobile Browser Testing
- Use browser developer tools device simulation
- Test responsive design
- Verify touch interactions work

### 7. Performance Testing

#### 7.1 Page Load Performance

**Steps**:
1. Open Developer Tools
2. Go to Performance/Profiler tab
3. Record page load
4. Analyze results

**Metrics to Check**:
- First Contentful Paint (FCP)
- Largest Contentful Paint (LCP)
- Time to Interactive (TTI)
- API response times

#### 7.2 Memory Usage Testing

**Steps**:
1. Open Developer Tools
2. Go to Memory tab
3. Take heap snapshots during usage
4. Check for memory leaks

### 8. End-to-End Workflow Testing

**Purpose**: Test complete user workflows from start to finish.

#### 8.1 Admin Workflow Testing

**Complete Workflow**:
1. **Admin adds new source**:
   - Navigate to admin interface
   - Add source with valid URL
   - Verify source is saved

2. **Admin triggers scraping**:
   - Select the new source
   - Initiate scraping process
   - Monitor progress and completion

3. **Data appears in main app**:
   - Navigate to main application
   - Verify scraped events appear
   - Test search and filtering on new data

#### 8.2 User Workflow Testing

**Complete Workflow**:
1. **User visits main app**:
   - Load main application
   - Browse available events

2. **User searches for activities**:
   - Use search functionality
   - Apply filters
   - View detailed event information

3. **User finds relevant activities**:
   - Verify event details are complete
   - Check that information is accurate and useful

## Testing Checklist

### Pre-Testing Setup
- [ ] Backend server running on http://127.0.0.1:3000
- [ ] Frontend server running on http://localhost:8000
- [ ] Browser developer tools open
- [ ] Network tab monitoring enabled

### Main Application Tests
- [ ] Page loads without errors
- [ ] Environment detection works
- [ ] API calls succeed
- [ ] Event data displays correctly
- [ ] Search functionality works
- [ ] Filters work correctly
- [ ] Responsive design works on mobile

### Admin Interface Tests
- [ ] Admin page loads successfully
- [ ] Source list displays
- [ ] Add source form works
- [ ] Edit source functionality works
- [ ] Delete source works
- [ ] Scraping can be triggered
- [ ] Scraping progress is shown

### API Tests
- [ ] GET /api/sources returns data
- [ ] POST /api/sources creates sources
- [ ] GET /api/events returns events
- [ ] POST /api/scrape works (with valid source)

### Error Handling Tests
- [ ] Backend unavailable handled gracefully
- [ ] Network errors show user-friendly messages
- [ ] Invalid API responses handled
- [ ] Form validation works correctly

### Cross-Browser Tests
- [ ] Chrome compatibility
- [ ] Firefox compatibility
- [ ] Safari compatibility (macOS)
- [ ] Mobile browser simulation

### Performance Tests
- [ ] Page load times acceptable
- [ ] API response times reasonable
- [ ] No memory leaks detected
- [ ] Smooth user interactions

## Automated Testing Scripts

You can run these scripts in the browser console for automated testing:

```javascript
// Comprehensive test suite
async function runBrowserTests() {
    console.log('🧪 Starting browser test suite...');
    
    const tests = [
        testEnvironmentDetection,
        testAPIConnectivity,
        testDataLoading,
        testSourcesAPI,
        testEventsAPI
    ];
    
    const results = [];
    
    for (const test of tests) {
        try {
            console.log(`Running ${test.name}...`);
            const result = await test();
            results.push({ test: test.name, status: 'PASS', result });
            console.log(`✅ ${test.name} passed`);
        } catch (error) {
            results.push({ test: test.name, status: 'FAIL', error: error.message });
            console.error(`❌ ${test.name} failed:`, error);
        }
    }
    
    console.log('🏁 Test suite complete:', results);
    return results;
}

// Run the test suite
runBrowserTests();
```

## Reporting Issues

When reporting browser testing issues, include:

1. **Browser and version**
2. **Steps to reproduce**
3. **Expected vs actual behavior**
4. **Console errors** (copy full error messages)
5. **Network tab information** (failed requests)
6. **Screenshots** if UI issues are present

## Best Practices

1. **Always test with developer tools open**
2. **Check both Console and Network tabs**
3. **Test in multiple browsers**
4. **Verify both success and error scenarios**
5. **Test with realistic data volumes**
6. **Check mobile responsiveness**
7. **Validate all user workflows end-to-end**