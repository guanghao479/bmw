// Simple Router for SPA Navigation
class Router {
    constructor(app) {
        this.app = app;
        this.routes = {
            '': () => this.app.showListView(),
            'activity': (id) => this.app.showDetailView(id)
        };
        
        // Listen for hash changes
        window.addEventListener('hashchange', () => this.handleRoute());
        window.addEventListener('load', () => this.handleRoute());
    }
    
    handleRoute() {
        const hash = window.location.hash.slice(1); // Remove #
        const [route, param] = hash.split('/');
        
        if (this.routes[route]) {
            this.routes[route](param);
        } else {
            this.routes['']();
        }
    }
    
    navigate(path) {
        window.location.hash = path;
    }
    
    back() {
        window.history.back();
    }
}

// Family Events App - Dynamic Content Loading and Interaction
class FamilyEventsApp {
    constructor() {
        this.allData = [];
        this.currentFilter = 'all';
        this.searchTerm = '';
        this.lastUpdated = null;
        this.refreshInterval = null;
        this.config = this.loadConfiguration();
        this.router = new Router(this);
        this.currentView = 'list'; // 'list' or 'detail'
        this.selectedDate = 'all'; // 'all' or specific date (YYYY-MM-DD)
        this.dateTabs = [];
        this.currentTabIndex = 0;
        
        this.init();
    }

    // Load configuration based on environment
    loadConfiguration() {
        const isLocal = window.location.hostname === 'localhost' || 
                       window.location.hostname === '127.0.0.1';
        const isDevelopment = isLocal || window.location.hostname.includes('github.dev');
        
        const baseConfig = {
            refreshIntervalMs: 30 * 60 * 1000, // 30 minutes
            retryAttempts: 3,
            retryDelay: 1000,
            cacheKey: 'familyEvents_cached_data',
            cacheTimestamp: 'familyEvents_cache_timestamp',
            maxCacheAge: 24 * 60 * 60 * 1000, // 24 hours
            environment: isDevelopment ? 'development' : 'production'
        };

        // Local development with SAM CLI
        if (isLocal) {
            return {
                ...baseConfig,
                // SAM CLI default local API Gateway endpoint
                apiEndpoint: 'http://127.0.0.1:3000/api/events/approved',
                refreshIntervalMs: 5 * 60 * 1000, // 5 minutes for development
                debugMode: true,
                samLocal: true,
                environment: 'local'
            };
        }
        // Other development environments (GitHub Codespaces, etc.)
        else if (isDevelopment) {
            return {
                ...baseConfig,
                apiEndpoint: 'https://your-api-gateway-url.amazonaws.com/api/events/approved', // TODO: Update with actual API Gateway URL
                refreshIntervalMs: 5 * 60 * 1000, // 5 minutes for development
                debugMode: true
            };
        } 
        // Production
        else {
            return {
                ...baseConfig,
                apiEndpoint: 'https://your-api-gateway-url.amazonaws.com/api/events/approved', // TODO: Update with actual API Gateway URL
                debugMode: false
            };
        }
    }

    async init() {
        this.showLoading();
        await this.loadData();
        this.setupEventListeners();
        this.renderContent();
        this.setupAutoRefresh();
        this.hideLoading();
    }

    // Load data from API with offline fallback
    async loadData() {
        try {
            // Try to fetch fresh data from API
            const freshData = await this.fetchFromAPI();
            if (freshData) {
                this.processData(freshData);
                this.cacheData(freshData);
                const count = this.allData.length;
                this.showDataStatus(`Fresh data loaded: ${count} activities (${this.config.environment})`, 'success');
                return;
            }
        } catch (error) {
            console.warn('Failed to fetch fresh data:', error);
            
            // Provide specific error message for local development
            if (this.config.samLocal) {
                this.showDataStatus('Local backend unavailable - using cached data', 'warning');
            } else {
                this.showDataStatus(`Using cached data (${this.config.environment})`, 'warning');
            }
        }

        // Fall back to cached data
        const cachedData = this.getCachedData();
        if (cachedData) {
            this.processData(cachedData);
            this.showDataStatus('Loaded from cache', 'info');
            return;
        }

        // No data available - show environment-specific error message
        if (this.config.samLocal) {
            this.showError('Local backend is not running. Please start the SAM local API server with: sam local start-api -t ../infrastructure/cdk.out/SeattleFamilyActivitiesMVPStack.template.json --env-vars env.json --port 3000');
            this.showDataStatus('Local backend unavailable', 'error');
        } else {
            this.showError('Unable to load family activities from our database. Please check your internet connection and try refreshing the page.');
            this.showDataStatus('Failed to load API data', 'error');
        }
    }

    // Fetch data from API endpoint
    async fetchFromAPI() {
        if (this.config.debugMode) {
            console.log(`Fetching data from API: ${this.config.apiEndpoint}`);
        }

        for (let attempt = 1; attempt <= this.config.retryAttempts; attempt++) {
            try {
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), 10000); // 10s timeout

                // Build query parameters for the API
                const params = new URLSearchParams({
                    limit: '200' // Request more activities for better user experience
                });

                // Add updated_since parameter if we have cached data
                const lastUpdateTimestamp = this.getLastUpdateTimestamp();
                if (lastUpdateTimestamp) {
                    params.append('updated_since', lastUpdateTimestamp);
                }

                const apiUrl = `${this.config.apiEndpoint}?${params}`;

                const response = await fetch(apiUrl, {
                    signal: controller.signal,
                    mode: this.config.samLocal ? 'cors' : 'cors',
                    credentials: this.config.samLocal ? 'omit' : 'same-origin',
                    headers: {
                        'Cache-Control': 'no-cache',
                        'Accept': 'application/json'
                    }
                });

                clearTimeout(timeoutId);

                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                }

                const apiResponse = await response.json();

                // Validate API response structure
                if (!apiResponse.success) {
                    throw new Error(`API error: ${apiResponse.error || 'Unknown error'}`);
                }

                const activities = apiResponse.data?.activities || [];
                if (!Array.isArray(activities)) {
                    throw new Error('Invalid data structure from API');
                }

                if (this.config.debugMode) {
                    console.log(`API fetch successful: ${activities.length} activities, last updated: ${apiResponse.data?.meta?.last_updated}`);
                }

                // Update last update timestamp from API metadata
                if (apiResponse.data?.meta?.last_updated) {
                    this.setLastUpdateTimestamp(apiResponse.data.meta.last_updated);
                }

                // Convert API response to the expected format
                return {
                    activities: activities,
                    metadata: {
                        lastUpdated: apiResponse.data?.meta?.last_updated || new Date().toISOString(),
                        total: apiResponse.data?.meta?.total || activities.length,
                        source: 'database_api'
                    }
                };
            } catch (error) {
                if (this.config.debugMode) {
                    console.warn(`API fetch attempt ${attempt}/${this.config.retryAttempts} failed:`, error);
                }

                if (attempt < this.config.retryAttempts) {
                    await new Promise(resolve => setTimeout(resolve, this.config.retryDelay * attempt));
                }
            }
        }

        return null;
    }

    // Helper functions for timestamp management
    getLastUpdateTimestamp() {
        return localStorage.getItem('familyEvents_last_updated');
    }

    setLastUpdateTimestamp(timestamp) {
        localStorage.setItem('familyEvents_last_updated', timestamp);
    }

    // Process data from API (new schema) to legacy format for compatibility
    processData(data) {
        if (!data.activities) {
            console.error('No activities in data:', data);
            return;
        }

        this.lastUpdated = data.metadata?.lastUpdated || new Date().toISOString();

        // Store original data for detail page access
        this.originalData = data;

        // Convert API response activities to legacy format for existing UI compatibility
        this.allData = data.activities.map(activity => this.convertToLegacyFormat(activity));
        
        // Update date tabs when data changes
        if (this.dateTabs && this.dateTabs.length > 0) {
            this.updateDateTabsDisplay();
        }
    }

    // Convert new activity schema to legacy format
    convertToLegacyFormat(activity) {
        return {
            id: activity.id,
            title: activity.title,
            description: activity.description,
            category: this.mapCategoryToLegacy(activity.type, activity.category),
            image: this.getActivityImage(activity),
            date: this.formatSchedule(activity.schedule),
            time: this.formatTime(activity.schedule),
            location: activity.location?.name || activity.location?.address || 'Location TBD',
            price: this.formatPrice(activity.pricing),
            age_range: this.formatAgeRange(activity.ageGroups),
            featured: activity.featured || false,
            detail_url: activity.detailURL || activity.registration?.url || null
        };
    }

    // Map new schema types/categories to legacy categories
    mapCategoryToLegacy(type, category) {
        const typeMap = {
            'class': 'activity',
            'camp': 'activity', 
            'event': 'event',
            'performance': 'event',
            'free-activity': 'activity'
        };
        return typeMap[type] || 'activity';
    }

    // Generate appropriate Unsplash image URL based on category
    generateImageUrl(category, subcategory) {
        const imageMap = {
            'arts-creativity': 'photo-1513475382585-d06e58bcb0e0',
            'active-sports': 'photo-1530549387789-4c1017266635',
            'educational-stem': 'photo-1581833971358-2c8b550f87b3',
            'entertainment-events': 'photo-1533174072545-7a4b6ad7a6c3',
            'camps-programs': 'photo-1578662996442-48f60103fc96',
            'free-community': 'photo-1507003211169-0a1dd7228f2d'
        };
        
        const imageId = imageMap[category] || imageMap['entertainment-events'];
        return `https://images.unsplash.com/${imageId}?w=800&h=600&fit=crop&auto=format&q=80`;
    }

    // Get activity image - use real image if available, fallback to generated
    getActivityImage(activity) {
        // Check if activity has real images from scraping
        if (activity.images && activity.images.length > 0) {
            // Use the first available image
            const realImage = activity.images[0];
            if (realImage && realImage.url && realImage.url.startsWith('http')) {
                return realImage.url;
            }
        }
        
        // Fallback to category-based Unsplash image
        return this.generateImageUrl(activity.category, activity.subcategory);
    }

    // Format schedule for display
    formatSchedule(schedule) {
        if (!schedule) return 'TBD';
        
        if (schedule.type === 'recurring' && schedule.daysOfWeek) {
            return schedule.daysOfWeek.map(day => 
                day.charAt(0).toUpperCase() + day.slice(1)
            ).join(', ');
        }
        
        if (schedule.startDate) {
            return schedule.startDate;
        }
        
        return 'TBD';
    }

    // Format time for display
    formatTime(schedule) {
        if (!schedule || !schedule.times || !schedule.times[0]) {
            return 'TBD';
        }
        
        const time = schedule.times[0];
        if (time.startTime && time.endTime) {
            return `${time.startTime} - ${time.endTime}`;
        }
        
        return time.startTime || 'TBD';
    }

    // Format pricing for display
    formatPrice(pricing) {
        if (!pricing || pricing.type === 'free') {
            return 'Free';
        }
        
        if (pricing.cost && pricing.currency) {
            const symbol = pricing.currency === 'USD' ? '$' : pricing.currency;
            return `${symbol}${pricing.cost}${pricing.unit ? `/${pricing.unit}` : ''}`;
        }
        
        return pricing.description || 'Price varies';
    }

    // Format age range for display
    formatAgeRange(ageGroups) {
        if (!ageGroups || ageGroups.length === 0) {
            return 'All ages';
        }
        
        const ageGroup = ageGroups[0];
        if (ageGroup.description) {
            return ageGroup.description;
        }
        
        if (ageGroup.minAge && ageGroup.maxAge) {
            const unit = ageGroup.unit === 'months' ? 'mo' : 'yr';
            return `${ageGroup.minAge}-${ageGroup.maxAge} ${unit}`;
        }
        
        return ageGroup.category || 'All ages';
    }

    // Cache data in localStorage
    cacheData(data) {
        try {
            localStorage.setItem(this.config.cacheKey, JSON.stringify(data));
            localStorage.setItem(this.config.cacheTimestamp, Date.now().toString());
        } catch (error) {
            console.warn('Failed to cache data:', error);
        }
    }

    // Get cached data from localStorage
    getCachedData() {
        try {
            const cached = localStorage.getItem(this.config.cacheKey);
            const timestamp = localStorage.getItem(this.config.cacheTimestamp);
            
            if (cached && timestamp) {
                const age = Date.now() - parseInt(timestamp);
                
                if (age < this.config.maxCacheAge) {
                    if (this.config.debugMode) {
                        console.log(`Using cached data (age: ${Math.round(age / 1000 / 60)} minutes)`);
                    }
                    return JSON.parse(cached);
                }
            }
        } catch (error) {
            console.warn('Failed to get cached data:', error);
        }
        
        return null;
    }


    // Setup auto-refresh functionality
    setupAutoRefresh() {
        // Clear any existing interval
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
        }
        
        // Set up new refresh interval
        this.refreshInterval = setInterval(async () => {
            console.log('Auto-refreshing data...');
            await this.refreshData();
        }, this.config.refreshIntervalMs);
        
        // Listen for visibility change to refresh when page becomes visible
        document.addEventListener('visibilitychange', () => {
            if (!document.hidden) {
                this.refreshData();
            }
        });
    }

    // Refresh data manually
    async refreshData() {
        try {
            const freshData = await this.fetchFromS3();
            if (freshData) {
                const oldCount = this.allData.length;
                this.processData(freshData);
                this.cacheData(freshData);
                this.renderContent();
                
                const newCount = this.allData.length;
                if (newCount !== oldCount) {
                    this.showDataStatus(`Updated! ${newCount} activities available`, 'success');
                }
            }
        } catch (error) {
            console.warn('Auto-refresh failed:', error);
        }
    }

    // Show data status message to user
    showDataStatus(message, type = 'info') {
        // Create or update status element
        let statusElement = document.getElementById('data-status');
        if (!statusElement) {
            statusElement = document.createElement('div');
            statusElement.id = 'data-status';
            statusElement.style.cssText = `
                position: fixed;
                top: 20px;
                right: 20px;
                padding: 12px 16px;
                border-radius: 8px;
                font-size: 14px;
                font-weight: 500;
                z-index: 1000;
                opacity: 0;
                transition: opacity 0.3s ease;
                max-width: 300px;
            `;
            document.body.appendChild(statusElement);
        }
        
        // Set colors based on type
        const colors = {
            success: { bg: '#d4edda', text: '#155724', border: '#c3e6cb' },
            warning: { bg: '#fff3cd', text: '#856404', border: '#ffeaa7' },
            error: { bg: '#f8d7da', text: '#721c24', border: '#f5c6cb' },
            info: { bg: '#d1ecf1', text: '#0c5460', border: '#bee5eb' }
        };
        
        const color = colors[type] || colors.info;
        statusElement.style.backgroundColor = color.bg;
        statusElement.style.color = color.text;
        statusElement.style.border = `1px solid ${color.border}`;
        statusElement.textContent = message;
        
        // Show and auto-hide
        statusElement.style.opacity = '1';
        setTimeout(() => {
            statusElement.style.opacity = '0';
        }, 5000);
    }

    // Setup event listeners for interactivity
    setupEventListeners() {
        // Search input
        const searchInput = document.getElementById('searchInput');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.searchTerm = e.target.value.toLowerCase();
                this.renderContent();
            });
        }

        // Filter buttons
        const filterButtons = document.querySelectorAll('.filter-btn');
        filterButtons.forEach(btn => {
            btn.addEventListener('click', (e) => {
                // Remove active class and set aria-pressed to false for all buttons
                filterButtons.forEach(button => {
                    button.classList.remove('active');
                    button.setAttribute('aria-pressed', 'false');
                });
                // Add active class and set aria-pressed to true for clicked button
                e.target.classList.add('active');
                e.target.setAttribute('aria-pressed', 'true');
                
                this.currentFilter = e.target.dataset.filter;
                this.renderContent();
            });
        });

        // Card click interactions
        document.addEventListener('click', (e) => {
            if (e.target.closest('.card')) {
                const card = e.target.closest('.card');
                this.handleCardClick(card);
            }
        });

        // Card keyboard interactions
        document.addEventListener('keydown', (e) => {
            if (e.target.closest('.card') && (e.key === 'Enter' || e.key === ' ')) {
                e.preventDefault();
                const card = e.target.closest('.card');
                this.handleCardClick(card);
            }
        });

        // Breadcrumb back button
        const breadcrumbBack = document.getElementById('breadcrumbBack');
        if (breadcrumbBack) {
            breadcrumbBack.addEventListener('click', () => {
                this.router.navigate('');
            });
        }

        // Add manual refresh button
        this.addRefreshButton();
        
        // Setup date tabs
        this.setupDateTabs();
    }

    // Add manual refresh button
    addRefreshButton() {
        // Check if refresh button already exists
        if (document.getElementById('refresh-btn')) return;

        const refreshBtn = document.createElement('button');
        refreshBtn.id = 'refresh-btn';
        refreshBtn.innerHTML = '🔄 Refresh';
        refreshBtn.style.cssText = `
            position: fixed;
            top: 80px;
            right: 20px;
            background: rgba(255, 255, 255, 0.9);
            border: 1px solid #ddd;
            border-radius: 8px;
            padding: 8px 16px;
            font-size: 14px;
            cursor: pointer;
            z-index: 999;
            backdrop-filter: blur(10px);
            transition: all 0.2s ease;
        `;

        refreshBtn.addEventListener('click', async () => {
            refreshBtn.disabled = true;
            refreshBtn.innerHTML = '🔄 Refreshing...';
            
            await this.refreshData();
            
            refreshBtn.disabled = false;
            refreshBtn.innerHTML = '🔄 Refresh';
        });

        refreshBtn.addEventListener('mouseenter', () => {
            refreshBtn.style.background = 'rgba(255, 255, 255, 1)';
            refreshBtn.style.transform = 'translateY(-1px)';
        });

        refreshBtn.addEventListener('mouseleave', () => {
            refreshBtn.style.background = 'rgba(255, 255, 255, 0.9)';
            refreshBtn.style.transform = 'translateY(0)';
        });

        document.body.appendChild(refreshBtn);
    }

    // Filter and search data
    getFilteredData() {
        return this.allData.filter(item => {
            // Filter by category
            const matchesFilter = this.currentFilter === 'all' || 
                                item.category === this.currentFilter.slice(0, -1); // Remove 's' from 'events', etc.

            // Filter by search term
            const matchesSearch = this.searchTerm === '' ||
                                item.title.toLowerCase().includes(this.searchTerm) ||
                                item.description.toLowerCase().includes(this.searchTerm) ||
                                item.location.toLowerCase().includes(this.searchTerm);
            
            // Filter by selected date
            const activityDate = this.getActivityDate(item);
            const matchesDate = this.selectedDate === 'all' || activityDate === this.selectedDate;
            
            // Debug logging for date filtering
            if (this.config.debugMode && this.selectedDate !== 'all' && activityDate) {
                console.log(`Filtering: activity "${item.title}" has date "${activityDate}", selected "${this.selectedDate}", matches: ${matchesDate}`);
            }

            return matchesFilter && matchesSearch && matchesDate;
        });
    }

    // Render all content
    renderContent() {
        const filteredData = this.getFilteredData();
        this.renderMainContent(filteredData);
    }

    // Render main content
    renderMainContent(items) {
        const contentGrid = document.getElementById('contentGrid');
        
        if (items.length === 0) {
            contentGrid.innerHTML = `
                <div class="no-results">
                    <h3>No items found</h3>
                    <p>Try adjusting your search or filter criteria.</p>
                </div>
            `;
            return;
        }

        contentGrid.innerHTML = items
            .map(item => this.createCardHTML(item, false))
            .join('');
    }

    // Create HTML for a single card
    createCardHTML(item) {
        const categoryClass = `category-${item.category}`;
        
        return `
            <div class="card" data-id="${item.id}" role="button" tabindex="0" aria-label="View details for ${item.title}">
                <img src="${item.image}" alt="${item.title} activity" class="card-image" loading="lazy" onerror="this.src='data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAwIiBoZWlnaHQ9IjMwMCIgdmlld0JveD0iMCAwIDQwMCAzMDAiIGZpbGw9Im5vbmUiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+CjxyZWN0IHdpZHRoPSI0MDAiIGhlaWdodD0iMzAwIiBmaWxsPSIjRjVGNUY1Ii8+CjxwYXRoIGQ9Ik0xNzUgMTI1SDE0MFYxNzVIMTc1VjE1MEgyMjVWMTc1SDI2MFYxMjVIMjI1VjEwMEgxNzVWMTI1WiIgZmlsbD0iIzk5OTk5OSIvPgo8L3N2Zz4K'; this.onerror=null;">
                <div class="card-content">
                    <span class="card-category ${categoryClass}">${this.formatCategory(item.category)}</span>
                    <h3 class="card-title">${item.title}</h3>
                    <p class="card-description">${item.description}</p>
                    <div class="card-meta">
                        <div>
                            <div class="card-date">${this.formatDate(this.getActivityDate(item))} • ${item.time}</div>
                            <div class="card-location">📍 ${item.location}</div>
                            ${item.age_range ? `<div class="card-age">👶 ${item.age_range}</div>` : ''}
                        </div>
                        <div class="card-price">${item.price}</div>
                    </div>
                </div>
            </div>
        `;
    }

    // Format category for display
    formatCategory(category) {
        const categoryMap = {
            'event': '🎉 Event',
            'activity': '⚽ Activity',
            'venue': '🏛️ Venue'
        };
        return categoryMap[category] || category;
    }

    // Date utility functions - centralized and DRY
    parseDate(dateString) {
        // Parse any date string into a Date object safely (timezone-aware)
        if (!dateString || dateString === 'TBD') {
            return null;
        }
        
        try {
            // Handle YYYY-MM-DD format safely to avoid timezone issues
            if (/^\d{4}-\d{2}-\d{2}$/.test(dateString)) {
                const [year, month, day] = dateString.split('-');
                return new Date(parseInt(year), parseInt(month) - 1, parseInt(day));
            }
            
            // Fallback for other date formats
            return new Date(dateString + 'T12:00:00'); // Add noon to avoid timezone shifts
        } catch {
            return null;
        }
    }
    
    // Format date for card display
    formatDate(dateString) {
        if (!dateString) {
            return 'TBD';
        }
        
        // Handle recurring dates
        if (dateString.includes('day')) {
            return dateString;
        }
        
        const date = this.parseDate(dateString);
        if (!date) {
            return dateString;
        }
        
        return date.toLocaleDateString('en-US', {
            weekday: 'short',
            month: 'short',
            day: 'numeric'
        });
    }
    
    // Format date for tab labels
    formatDateTabLabel(dateString, daysFromToday) {
        const date = this.parseDate(dateString);
        if (!date) {
            return dateString;
        }
        
        if (daysFromToday < 7) {
            // This week: "Mon 18", "Tue 19"
            return date.toLocaleDateString('en-US', { weekday: 'short', day: 'numeric' });
        } else {
            // Future weeks: "Nov 25", "Dec 2"
            return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
        }
    }

    // Handle card click interactions
    handleCardClick(card) {
        const itemId = card.dataset.id;
        this.router.navigate(`activity/${itemId}`);
    }

    // Show list view
    showListView() {
        this.currentView = 'list';
        document.querySelector('.container').style.display = 'block';
        document.getElementById('detailPage').classList.remove('show');
        setTimeout(() => {
            document.getElementById('detailPage').style.display = 'none';
        }, 300);
    }
    
    // Show detail view for specific activity
    showDetailView(activityId) {
        const item = this.allData.find(i => i.id == activityId);
        
        if (!item) {
            console.error('Activity not found:', activityId);
            this.router.navigate('');
            return;
        }
        
        this.currentView = 'detail';
        this.renderDetailPage(item);
        
        // Hide list view and show detail page
        document.querySelector('.container').style.display = 'none';
        const detailPage = document.getElementById('detailPage');
        detailPage.style.display = 'block';
        setTimeout(() => detailPage.classList.add('show'), 10);
    }
    
    // Render the detail page content
    renderDetailPage(item) {
        const detailContent = document.getElementById('detailContent');
        
        // Get original activity data from backend if available
        const originalActivity = this.getOriginalActivityData(item.id);
        
        detailContent.innerHTML = `
            <div class="detail-header">
                <div class="detail-image-container">
                    <img src="${item.image}" alt="${item.title}" class="detail-image" 
                         onerror="this.src='data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAwIiBoZWlnaHQ9IjMwMCIgdmlld0JveD0iMCAwIDQwMCAzMDAiIGZpbGw9Im5vbmUiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+CjxyZWN0IHdpZHRoPSI0MDAiIGhlaWdodD0iMzAwIiBmaWxsPSIjRjVGNUY1Ii8+CjxwYXRoIGQ9Ik0xNzUgMTI1SDE0MFYxNzVIMTc1VjE1MEgyMjVWMTc1SDI2MFYxMjVIMjI1VjEwMEgxNzVWMTI1WiIgZmlsbD0iIzk5OTk5OSIvPgo8L3N2Zz4K'; this.onerror=null;">
                    ${this.renderImageGallery(originalActivity)}
                </div>
                
                <div class="detail-category">${this.formatCategory(item.category)}</div>
                <h1 class="detail-title">${item.title}</h1>
                <p class="detail-description">${item.description}</p>
            </div>
            
            ${this.renderScheduleSection(originalActivity || item)}
            ${this.renderLocationSection(originalActivity || item)}
            ${this.renderPricingSection(originalActivity || item)}
            ${this.renderRegistrationSection(originalActivity || item)}
            ${this.renderProviderSection(originalActivity)}
            ${this.renderAdditionalInfo(originalActivity || item)}
        `;
    }
    
    // Get original activity data from the backend if available
    getOriginalActivityData(itemId) {
        // Try to find the original activity data from the backend
        if (this.originalData && this.originalData.activities) {
            return this.originalData.activities.find(a => a.id == itemId);
        }
        return null;
    }
    
    // Render image gallery if multiple images available
    renderImageGallery(originalActivity) {
        if (!originalActivity || !originalActivity.images || originalActivity.images.length <= 1) {
            return '';
        }
        
        const thumbnails = originalActivity.images.slice(1).map((img, index) => 
            `<img src="${img.url}" alt="${img.altText || ''}" 
                 class="detail-image-thumbnail" 
                 onclick="this.closest('.detail-image-container').querySelector('.detail-image').src='${img.url}'">`
        ).join('');
        
        return `<div class="detail-image-gallery">${thumbnails}</div>`;
    }
    
    // Render schedule section
    renderScheduleSection(item) {
        const schedule = item.schedule || {};
        const times = schedule.times || [];
        
        let scheduleContent = `
            <div class="detail-info-grid">
                <div class="detail-info-item">
                    <div class="detail-info-label">Date</div>
                    <div class="detail-info-value">${item.date || schedule.startDate || 'TBD'}</div>
                </div>
                <div class="detail-info-item">
                    <div class="detail-info-label">Time</div>
                    <div class="detail-info-value">${item.time || this.formatTime(schedule)}</div>
                </div>
        `;
        
        if (schedule.type) {
            scheduleContent += `
                <div class="detail-info-item">
                    <div class="detail-info-label">Schedule Type</div>
                    <div class="detail-info-value">${this.formatScheduleType(schedule.type)}</div>
                </div>
            `;
        }
        
        if (schedule.duration) {
            scheduleContent += `
                <div class="detail-info-item">
                    <div class="detail-info-label">Duration</div>
                    <div class="detail-info-value">${schedule.duration}</div>
                </div>
            `;
        }
        
        scheduleContent += `</div>`;
        
        // Add time slots if available
        if (times.length > 1) {
            const timeSlots = times.map(time => 
                `<div class="schedule-time">
                    ${time.startTime} - ${time.endTime}
                    ${time.ageGroup ? `<br><small>${time.ageGroup}</small>` : ''}
                </div>`
            ).join('');
            
            scheduleContent += `
                <div class="schedule-times">
                    ${timeSlots}
                </div>
            `;
        }
        
        return `
            <div class="detail-section">
                <h3 class="detail-section-title">
                    📅 Schedule & Timing
                </h3>
                ${scheduleContent}
            </div>
        `;
    }
    
    // Render location section
    renderLocationSection(item) {
        const location = item.location || {};
        
        return `
            <div class="detail-section">
                <h3 class="detail-section-title">
                    📍 Location
                </h3>
                <div class="detail-info-grid">
                    <div class="detail-info-item">
                        <div class="detail-info-label">Venue</div>
                        <div class="detail-info-value">${location.name || item.location || 'TBD'}</div>
                    </div>
                    ${location.address ? `
                    <div class="detail-info-item">
                        <div class="detail-info-label">Address</div>
                        <div class="detail-info-value">${location.address}</div>
                    </div>
                    ` : ''}
                    ${location.neighborhood ? `
                    <div class="detail-info-item">
                        <div class="detail-info-label">Neighborhood</div>
                        <div class="detail-info-value">${location.neighborhood}</div>
                    </div>
                    ` : ''}
                    ${location.parking ? `
                    <div class="detail-info-item">
                        <div class="detail-info-label">Parking</div>
                        <div class="detail-info-value">${location.parking}</div>
                    </div>
                    ` : ''}
                    ${location.accessibility ? `
                    <div class="detail-info-item">
                        <div class="detail-info-label">Accessibility</div>
                        <div class="detail-info-value">${location.accessibility}</div>
                    </div>
                    ` : ''}
                </div>
            </div>
        `;
    }
    
    // Render pricing section
    renderPricingSection(item) {
        const pricing = item.pricing || {};
        
        let pricingContent = `
            <div class="pricing-details">
                <div class="price-main">${item.price || this.formatPrice(pricing)}</div>
        `;
        
        if (pricing.description) {
            pricingContent += `<div class="price-description">${pricing.description}</div>`;
        }
        
        if (pricing.includesSupplies) {
            pricingContent += `<div class="price-description">✅ Supplies included</div>`;
        }
        
        // Add discounts if available
        if (pricing.discounts && pricing.discounts.length > 0) {
            const discounts = pricing.discounts.map(discount => 
                `<span class="discount-item">${discount.description || discount.type}</span>`
            ).join('');
            
            pricingContent += `
                <div class="discounts-list">
                    <strong>Available Discounts:</strong><br>
                    ${discounts}
                </div>
            `;
        }
        
        pricingContent += `</div>`;
        
        return `
            <div class="detail-section">
                <h3 class="detail-section-title">
                    💰 Pricing
                </h3>
                ${pricingContent}
            </div>
        `;
    }
    
    // Render registration section
    renderRegistrationSection(item) {
        const registration = item.registration || {};
        
        let registrationContent = `
            <div class="detail-info-grid">
                <div class="detail-info-item">
                    <div class="detail-info-label">Registration</div>
                    <div class="detail-info-value">
                        ${registration.required !== false ? 'Required' : 'Not Required'}
                    </div>
                </div>
        `;
        
        if (registration.status) {
            registrationContent += `
                <div class="detail-info-item">
                    <div class="detail-info-label">Status</div>
                    <div class="detail-info-value">
                        <span class="registration-status ${registration.status}">
                            ${this.formatRegistrationStatus(registration.status)}
                        </span>
                    </div>
                </div>
            `;
        }
        
        if (registration.deadline) {
            registrationContent += `
                <div class="detail-info-item">
                    <div class="detail-info-label">Deadline</div>
                    <div class="detail-info-value">${registration.deadline}</div>
                </div>
            `;
        }
        
        registrationContent += `</div>`;
        
        // Add contact methods
        if (registration.phone || registration.email || registration.url) {
            registrationContent += `
                <div class="contact-methods">
                    ${registration.url ? `
                        <a href="${registration.url}" target="_blank" class="contact-method">
                            🌐 Register Online
                        </a>
                    ` : ''}
                    ${registration.phone ? `
                        <a href="tel:${registration.phone}" class="contact-method">
                            📞 ${registration.phone}
                        </a>
                    ` : ''}
                    ${registration.email ? `
                        <a href="mailto:${registration.email}" class="contact-method">
                            ✉️ ${registration.email}
                        </a>
                    ` : ''}
                </div>
            `;
        }
        
        return `
            <div class="detail-section">
                <h3 class="detail-section-title">
                    📝 Registration
                </h3>
                ${registrationContent}
            </div>
        `;
    }
    
    // Render provider section
    renderProviderSection(originalActivity) {
        if (!originalActivity || !originalActivity.provider) {
            return '';
        }
        
        const provider = originalActivity.provider;
        
        return `
            <div class="detail-section">
                <h3 class="detail-section-title">
                    🏢 About the Provider
                </h3>
                <div class="provider-info">
                    <div class="provider-logo">
                        ${provider.name.charAt(0).toUpperCase()}
                    </div>
                    <div class="provider-details">
                        <h4>${provider.name}</h4>
                        <p>${provider.description || provider.type}</p>
                        ${provider.website ? `
                            <a href="${provider.website}" target="_blank" class="contact-method">
                                🌐 Visit Website
                            </a>
                        ` : ''}
                    </div>
                </div>
            </div>
        `;
    }
    
    // Render additional information
    renderAdditionalInfo(item) {
        let content = '';
        
        // Age groups
        if (item.age_range || (item.ageGroups && item.ageGroups.length > 0)) {
            const ageGroups = item.ageGroups ? 
                item.ageGroups.map(ag => ag.description || ag.category).join(', ') : 
                item.age_range;
                
            content += `
                <div class="detail-section">
                    <h3 class="detail-section-title">
                        👶 Age Groups
                    </h3>
                    <div class="age-groups">
                        ${ageGroups.split(',').map(age => `<span class="age-group">${age.trim()}</span>`).join('')}
                    </div>
                </div>
            `;
        }
        
        // Tags
        if (item.tags && item.tags.length > 0) {
            content += `
                <div class="detail-section">
                    <h3 class="detail-section-title">
                        🏷️ Tags
                    </h3>
                    <div class="tags-container">
                        ${item.tags.map(tag => `<span class="tag">${tag}</span>`).join('')}
                    </div>
                </div>
            `;
        }
        
        return content;
    }
    
    // Helper methods for formatting
    formatScheduleType(type) {
        const typeMap = {
            'one-time': 'One-time Event',
            'recurring': 'Recurring',
            'multi-day': 'Multi-day',
            'ongoing': 'Ongoing'
        };
        return typeMap[type] || type;
    }
    
    formatRegistrationStatus(status) {
        const statusMap = {
            'open': '✅ Open',
            'waitlist': '⏳ Waitlist',
            'closed': '❌ Closed',
            'sold-out': '🎫 Sold Out'
        };
        return statusMap[status] || status;
    }
    
    // Setup date tabs functionality
    setupDateTabs() {
        this.generateDateTabs();
        this.setupDateTabsEventListeners();
        this.updateDateTabsDisplay();
    }
    
    // Generate date tabs for the next 30 days
    generateDateTabs() {
        // Create today using local date to avoid timezone issues
        const today = new Date();
        today.setHours(0, 0, 0, 0); // Normalize to start of day
        this.dateTabs = [];
        
        // Add "All Dates" tab
        this.dateTabs.push({
            date: 'all',
            label: 'All Dates',
            isToday: false,
            isWeekend: false,
            count: 0
        });
        
        // Generate tabs for next 30 days
        for (let i = 0; i < 30; i++) {
            const date = new Date(today);
            date.setDate(today.getDate() + i);
            
            // Use direct date component access to avoid timezone issues
            const year = date.getFullYear();
            const month = String(date.getMonth() + 1).padStart(2, '0');
            const day = String(date.getDate()).padStart(2, '0');
            const dateString = `${year}-${month}-${day}`;
            
            // Debug logging for date generation
            if (this.config.debugMode && i < 5) {
                console.log(`Date tab ${i}: ${dateString} (${date.toDateString()})`);
            }
            
            const dayOfWeek = date.getDay(); // 0 = Sunday, 6 = Saturday
            const isWeekend = dayOfWeek === 0 || dayOfWeek === 6;
            const isToday = i === 0;
            const isTomorrow = i === 1;
            
            let label;
            if (isToday) {
                label = 'Today';
            } else if (isTomorrow) {
                label = 'Tomorrow';
            } else {
                label = this.formatDateTabLabel(dateString, i);
            }
            
            this.dateTabs.push({
                date: dateString,
                label: label,
                isToday: isToday,
                isWeekend: isWeekend,
                count: 0
            });
        }
        
        // Update activity counts for each date
        this.updateDateTabCounts();
    }
    
    // Update activity counts for each date tab
    updateDateTabCounts() {
        this.dateTabs.forEach(tab => {
            if (tab.date === 'all') {
                tab.count = this.allData.length;
            } else {
                const activitiesForDate = this.allData.filter(item => {
                    const itemDate = this.getActivityDate(item);
                    return itemDate === tab.date;
                });
                tab.count = activitiesForDate.length;
                
                // Debug logging
                if (this.config.debugMode && activitiesForDate.length > 0) {
                    console.log(`Date tab ${tab.date} (${tab.label}): ${tab.count} activities`);
                    activitiesForDate.forEach(activity => {
                        console.log(`  - ${activity.title} (date: ${this.getActivityDate(activity)})`);
                    });
                }
            }
        });
    }
    
    // Get today's date string in YYYY-MM-DD format (local timezone)
    getTodayDateString() {
        const today = new Date();
        const year = today.getFullYear();
        const month = String(today.getMonth() + 1).padStart(2, '0');
        const day = String(today.getDate()).padStart(2, '0');
        return `${year}-${month}-${day}`;
    }
    
    // Get activity date in YYYY-MM-DD format
    getActivityDate(item) {
        // Try to get date from original activity data
        const originalActivity = this.getOriginalActivityData(item.id);
        if (originalActivity && originalActivity.schedule && originalActivity.schedule.startDate) {
            // Backend dates are already in YYYY-MM-DD format, return as-is
            return originalActivity.schedule.startDate;
        }
        
        // Fallback to parsing from formatted date in legacy data
        if (item.date && item.date !== 'TBD' && !item.date.includes('day')) {
            // If it's already in YYYY-MM-DD format, return as-is
            if (/^\d{4}-\d{2}-\d{2}$/.test(item.date)) {
                return item.date;
            }
            
            // Use centralized date parsing
            const parsedDate = this.parseDate(item.date);
            if (parsedDate) {
                const year = parsedDate.getFullYear();
                const month = String(parsedDate.getMonth() + 1).padStart(2, '0');
                const day = String(parsedDate.getDate()).padStart(2, '0');
                return `${year}-${month}-${day}`;
            }
        }
        
        return null; // No valid date found
    }
    
    // Setup event listeners for date tabs
    setupDateTabsEventListeners() {
        // Tab click handlers will be added when tabs are rendered
        
        // Navigation arrows
        const prevBtn = document.getElementById('datePrevBtn');
        const nextBtn = document.getElementById('dateNextBtn');
        
        if (prevBtn) {
            prevBtn.addEventListener('click', () => this.scrollDateTabs('prev'));
        }
        
        if (nextBtn) {
            nextBtn.addEventListener('click', () => this.scrollDateTabs('next'));
        }
    }
    
    // Update date tabs display
    updateDateTabsDisplay() {
        const dateTabsContainer = document.getElementById('dateTabs');
        if (!dateTabsContainer) return;
        
        this.updateDateTabCounts();
        
        dateTabsContainer.innerHTML = this.dateTabs.map(tab => {
            const classes = ['date-tab'];
            const isSelected = tab.date === this.selectedDate;
            
            if (isSelected) classes.push('active');
            if (tab.isToday) classes.push('today');
            if (tab.isWeekend) classes.push('weekend');
            if (tab.count === 0 && tab.date !== 'all') classes.push('no-activities');
            
            const countText = tab.count > 0 ? `${tab.count} activities` : 'no activities';
            const ariaLabel = `${tab.label}, ${countText}${isSelected ? ', selected' : ''}`;
            
            return `
                <button class="${classes.join(' ')}" 
                        data-date="${tab.date}"
                        role="tab"
                        aria-selected="${isSelected}"
                        aria-label="${ariaLabel}">
                    ${tab.label}
                    ${tab.count > 0 ? `<span class="count" aria-hidden="true">${tab.count}</span>` : ''}
                </button>
            `;
        }).join('');
        
        // Add click event listeners to new tabs
        dateTabsContainer.querySelectorAll('.date-tab').forEach(tab => {
            tab.addEventListener('click', (e) => {
                const date = e.target.closest('.date-tab').dataset.date;
                this.selectDateTab(date);
            });
        });
    }
    
    // Select a date tab
    selectDateTab(date) {
        this.selectedDate = date;
        this.updateDateTabsDisplay();
        this.renderContent();
        
        // Scroll selected tab into view
        const selectedTab = document.querySelector(`[data-date="${date}"]`);
        if (selectedTab) {
            selectedTab.scrollIntoView({ 
                behavior: 'smooth', 
                block: 'nearest', 
                inline: 'center' 
            });
        }
    }
    
    // Scroll date tabs horizontally
    scrollDateTabs(direction) {
        const scrollContainer = document.getElementById('dateTabsScroll');
        if (!scrollContainer) return;
        
        const scrollAmount = 200; // pixels
        const currentScroll = scrollContainer.scrollLeft;
        
        if (direction === 'prev') {
            scrollContainer.scrollTo({
                left: currentScroll - scrollAmount,
                behavior: 'smooth'
            });
        } else {
            scrollContainer.scrollTo({
                left: currentScroll + scrollAmount,
                behavior: 'smooth'
            });
        }
    }

    // Add modal styles
    addModalStyles() {
        const styles = document.createElement('style');
        styles.id = 'modal-styles';
        styles.textContent = `
            .modal-overlay {
                position: fixed;
                top: 0;
                left: 0;
                right: 0;
                bottom: 0;
                background: rgba(0, 0, 0, 0.8);
                display: flex;
                align-items: center;
                justify-content: center;
                z-index: 1000;
                opacity: 0;
                transition: opacity 0.3s ease;
                backdrop-filter: blur(10px);
            }
            
            .modal-overlay.show {
                opacity: 1;
            }
            
            .modal-content {
                background: rgba(255, 255, 255, 0.95);
                border-radius: 20px;
                max-width: 500px;
                max-height: 80vh;
                overflow-y: auto;
                position: relative;
                backdrop-filter: blur(20px);
                border: 1px solid rgba(255, 255, 255, 0.2);
                box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            }
            
            .modal-close {
                position: absolute;
                top: 15px;
                right: 20px;
                background: none;
                border: none;
                font-size: 2rem;
                cursor: pointer;
                color: #666;
                z-index: 1001;
            }
            
            .modal-image {
                width: 100%;
                height: 200px;
                object-fit: cover;
                border-radius: 20px 20px 0 0;
            }
            
            .modal-info {
                padding: 24px;
            }
            
            .modal-info h2 {
                margin-bottom: 12px;
                color: #333;
            }
            
            .modal-description {
                margin-bottom: 20px;
                color: #666;
                line-height: 1.6;
            }
            
            .modal-details p {
                margin-bottom: 8px;
                color: #555;
            }
            
            .modal-cta {
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: white;
                border: none;
                padding: 12px 24px;
                border-radius: 12px;
                font-weight: 600;
                cursor: pointer;
                transition: transform 0.2s ease;
                margin-top: 16px;
            }
            
            .modal-cta:hover {
                transform: translateY(-2px);
            }
        `;
        document.head.appendChild(styles);
    }

    // Show loading state
    showLoading() {
        document.getElementById('loading').classList.add('show');
    }

    // Hide loading state
    hideLoading() {
        document.getElementById('loading').classList.remove('show');
    }

    // Show error message
    showError(message) {
        const contentGrid = document.getElementById('contentGrid');
        contentGrid.innerHTML = `
            <div class="error-message">
                <h3>Oops! Something went wrong</h3>
                <p>${message}</p>
                <button onclick="location.reload()">Try Again</button>
            </div>
        `;
    }

    // Test environment detection (for debugging)
    testEnvironmentDetection() {
        console.log('Environment Detection Test:');
        console.log('- Hostname:', window.location.hostname);
        console.log('- Configuration:', this.config);
        console.log('- API Endpoint:', this.config.apiEndpoint);
        console.log('- Environment:', this.config.environment);
        console.log('- SAM Local:', this.config.samLocal);
        console.log('- Debug Mode:', this.config.debugMode);
        
        return {
            hostname: window.location.hostname,
            config: this.config
        };
    }

    // Test local backend connection
    async testLocalBackendConnection() {
        if (!this.config.samLocal) {
            console.log('Not in local development mode');
            return false;
        }

        try {
            console.log('Testing connection to local backend...');
            const response = await fetch('http://127.0.0.1:3000/api/health', {
                method: 'GET',
                mode: 'cors',
                credentials: 'omit',
                headers: {
                    'Accept': 'application/json'
                }
            });

            if (response.ok) {
                console.log('✅ Local backend is running and accessible');
                this.showDataStatus('Local backend connection successful', 'success');
                return true;
            } else {
                console.log('❌ Local backend responded with error:', response.status);
                this.showDataStatus(`Local backend error: ${response.status}`, 'error');
                return false;
            }
        } catch (error) {
            console.log('❌ Local backend connection failed:', error.message);
            this.showDataStatus('Local backend unavailable', 'error');
            return false;
        }
    }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.familyApp = new FamilyEventsApp();
});

// Global functions for testing (accessible from browser console)
window.testFrontendEnvironment = () => {
    if (window.familyApp) {
        return window.familyApp.testEnvironmentDetection();
    }
    console.log('App not initialized yet');
    return null;
};

window.testLocalBackend = async () => {
    if (window.familyApp) {
        return await window.familyApp.testLocalBackendConnection();
    }
    console.log('App not initialized yet');
    return false;
};

// Add some additional interactive features
document.addEventListener('DOMContentLoaded', () => {
    // Add scroll animations
    const observerOptions = {
        threshold: 0.1,
        rootMargin: '0px 0px -50px 0px'
    };

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.style.opacity = '1';
                entry.target.style.transform = 'translateY(0)';
            }
        });
    }, observerOptions);

    // Observe all cards for animation
    setTimeout(() => {
        document.querySelectorAll('.card').forEach(card => {
            card.style.opacity = '0';
            card.style.transform = 'translateY(20px)';
            card.style.transition = 'opacity 0.6s ease, transform 0.6s ease';
            observer.observe(card);
        });
    }, 100);
});