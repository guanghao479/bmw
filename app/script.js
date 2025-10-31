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

        // No data available - use mock data for testing
        console.log('No real data available, loading mock data for testing...');
        const mockData = this.getMockData();
        this.processData(mockData);
        this.showDataStatus('Using mock data for testing', 'info');
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

    // Get mock data for testing the modernized cards
    getMockData() {
        const today = new Date();
        const tomorrow = new Date(today);
        tomorrow.setDate(today.getDate() + 1);
        const nextWeek = new Date(today);
        nextWeek.setDate(today.getDate() + 7);

        return {
            activities: [
                {
                    id: 'mock-1',
                    title: 'Seattle Children\'s Museum Interactive Exhibits',
                    description: 'Hands-on learning experiences for kids of all ages with rotating exhibits, art studios, and STEM activities.',
                    type: 'venue',
                    category: 'educational-stem',
                    featured: true,
                    images: [{
                        url: 'https://images.unsplash.com/photo-1503454537195-1dcabb73ffb9?w=800&h=600&fit=crop&auto=format&q=80',
                        altText: 'Children playing at interactive museum exhibit'
                    }],
                    schedule: {
                        type: 'recurring',
                        startDate: today.toISOString().split('T')[0],
                        times: [{ startTime: '10:00 AM', endTime: '5:00 PM' }]
                    },
                    location: {
                        name: 'Seattle Children\'s Museum',
                        address: '305 Harrison St, Seattle, WA 98109',
                        neighborhood: 'Seattle Center'
                    },
                    pricing: {
                        type: 'paid',
                        cost: 15,
                        currency: 'USD',
                        description: 'General admission'
                    },
                    ageGroups: [{ description: '0-10 years', category: 'toddler-elementary' }],
                    registration: { required: false }
                },
                {
                    id: 'mock-2',
                    title: 'Woodland Park Zoo Animal Adventures',
                    description: 'Explore wildlife from around the world and participate in educational programs and animal encounters.',
                    type: 'activity',
                    category: 'educational-stem',
                    featured: false,
                    images: [{
                        url: 'https://images.unsplash.com/photo-1564349683136-77e08dba1ef7?w=800&h=600&fit=crop&auto=format&q=80',
                        altText: 'Family watching animals at zoo'
                    }],
                    schedule: {
                        type: 'recurring',
                        startDate: today.toISOString().split('T')[0],
                        times: [{ startTime: '9:30 AM', endTime: '4:00 PM' }]
                    },
                    location: {
                        name: 'Woodland Park Zoo',
                        address: '5500 Phinney Ave N, Seattle, WA 98103',
                        neighborhood: 'Phinney Ridge'
                    },
                    pricing: {
                        type: 'paid',
                        cost: 22,
                        currency: 'USD',
                        description: 'Adult admission'
                    },
                    ageGroups: [{ description: 'All ages', category: 'all-ages' }],
                    registration: { required: false }
                },
                {
                    id: 'mock-3',
                    title: 'Pike Place Market Family Food Tour',
                    description: 'Discover the flavors of Seattle with a family-friendly walking tour through the famous Pike Place Market.',
                    type: 'event',
                    category: 'entertainment-events',
                    featured: false,
                    images: [{
                        url: 'https://images.unsplash.com/photo-1555396273-367ea4eb4db5?w=800&h=600&fit=crop&auto=format&q=80',
                        altText: 'Family walking through Pike Place Market'
                    }],
                    schedule: {
                        type: 'one-time',
                        startDate: tomorrow.toISOString().split('T')[0],
                        times: [{ startTime: '11:00 AM', endTime: '1:00 PM' }]
                    },
                    location: {
                        name: 'Pike Place Market',
                        address: '85 Pike St, Seattle, WA 98101',
                        neighborhood: 'Downtown'
                    },
                    pricing: {
                        type: 'paid',
                        cost: 35,
                        currency: 'USD',
                        description: 'Per person (includes tastings)'
                    },
                    ageGroups: [{ description: '6+ years', category: 'elementary-teen' }],
                    registration: { required: true, status: 'open' }
                },
                {
                    id: 'mock-4',
                    title: 'Discovery Park Nature Walk',
                    description: 'Free guided nature walk through Seattle\'s largest park with opportunities to spot local wildlife.',
                    type: 'activity',
                    category: 'free-community',
                    featured: true,
                    images: [{
                        url: 'https://images.unsplash.com/photo-1441974231531-c6227db76b6e?w=800&h=600&fit=crop&auto=format&q=80',
                        altText: 'Family hiking on forest trail'
                    }],
                    schedule: {
                        type: 'recurring',
                        startDate: nextWeek.toISOString().split('T')[0],
                        times: [{ startTime: '10:00 AM', endTime: '11:30 AM' }]
                    },
                    location: {
                        name: 'Discovery Park',
                        address: '3801 Discovery Park Blvd, Seattle, WA 98199',
                        neighborhood: 'Magnolia'
                    },
                    pricing: {
                        type: 'free',
                        description: 'Free for all participants'
                    },
                    ageGroups: [{ description: 'All ages', category: 'all-ages' }],
                    registration: { required: false }
                },
                {
                    id: 'mock-5',
                    title: 'Seattle Art Museum Family Workshop',
                    description: 'Creative art-making workshop inspired by current exhibitions, designed for families to create together.',
                    type: 'event',
                    category: 'arts-creativity',
                    featured: false,
                    images: [{
                        url: 'https://images.unsplash.com/photo-1513475382585-d06e58bcb0e0?w=800&h=600&fit=crop&auto=format&q=80',
                        altText: 'Children creating art in museum workshop'
                    }],
                    schedule: {
                        type: 'one-time',
                        startDate: nextWeek.toISOString().split('T')[0],
                        times: [{ startTime: '2:00 PM', endTime: '4:00 PM' }]
                    },
                    location: {
                        name: 'Seattle Art Museum',
                        address: '1300 1st Ave, Seattle, WA 98101',
                        neighborhood: 'Downtown'
                    },
                    pricing: {
                        type: 'paid',
                        cost: 12,
                        currency: 'USD',
                        description: 'Per participant (materials included)'
                    },
                    ageGroups: [{ description: '4-12 years with adult', category: 'preschool-elementary' }],
                    registration: { required: true, status: 'open' }
                },
                {
                    id: 'mock-6',
                    title: 'Green Lake Family Bike Ride',
                    description: 'Scenic bike ride around Green Lake with bike rentals available. Perfect for families with children.',
                    type: 'activity',
                    category: 'active-sports',
                    featured: false,
                    images: [{
                        url: 'https://images.unsplash.com/photo-1558618666-fcd25c85cd64?w=800&h=600&fit=crop&auto=format&q=80',
                        altText: 'Family biking around lake'
                    }],
                    schedule: {
                        type: 'recurring',
                        startDate: today.toISOString().split('T')[0],
                        times: [{ startTime: '9:00 AM', endTime: '12:00 PM' }]
                    },
                    location: {
                        name: 'Green Lake Park',
                        address: '7201 E Green Lake Dr N, Seattle, WA 98115',
                        neighborhood: 'Green Lake'
                    },
                    pricing: {
                        type: 'free',
                        description: 'Free (bike rentals extra)'
                    },
                    ageGroups: [{ description: '5+ years', category: 'elementary-teen' }],
                    registration: { required: false }
                }
            ],
            metadata: {
                lastUpdated: new Date().toISOString(),
                total: 6,
                source: 'mock_data'
            }
        };
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
                // Remove active styling from all buttons
                filterButtons.forEach(button => {
                    // Remove active class
                    button.classList.remove('active');
                    button.setAttribute('aria-pressed', 'false');
                    
                    // Reset to inactive Tailwind classes
                    button.className = 'px-4 py-2 bg-white text-gray-700 border border-gray-300 rounded-full text-sm font-medium hover:bg-gray-50 hover:border-blue-500 hover:text-blue-600 active:bg-gray-100 transition-all duration-200 ease-in-out transform hover:-translate-y-0.5 hover:shadow-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 filter-btn';
                });
                
                // Add active styling to clicked button
                e.target.classList.add('active');
                e.target.setAttribute('aria-pressed', 'true');
                
                // Apply active Tailwind classes
                e.target.className = 'px-4 py-2 bg-blue-600 text-white rounded-full text-sm font-medium hover:bg-blue-700 active:bg-blue-800 transition-all duration-200 ease-in-out transform hover:-translate-y-0.5 hover:shadow-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 filter-btn active';
                
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
        const categoryClass = this.getCategoryTailwindClasses(item.category);
        const isFeatured = item.featured;
        
        // Enhanced classes for featured cards
        const cardClasses = isFeatured 
            ? "card bg-white rounded-xl shadow-lg hover:shadow-2xl transition-all duration-300 overflow-hidden group cursor-pointer transform hover:-translate-y-1 border border-blue-100"
            : "card bg-white rounded-xl shadow-md hover:shadow-xl transition-all duration-300 overflow-hidden group cursor-pointer transform hover:-translate-y-1";
            
        const imageClasses = isFeatured
            ? "w-full h-56 object-cover group-hover:scale-110 transition-transform duration-500"
            : "w-full h-48 object-cover group-hover:scale-105 transition-transform duration-300";
            
        const titleClasses = isFeatured
            ? "text-xl font-bold text-gray-900 mb-3 line-clamp-2 leading-tight"
            : "text-lg font-semibold text-gray-900 mb-2 line-clamp-2 leading-tight";
            
        const descriptionClasses = isFeatured
            ? "text-gray-600 text-base mb-4 line-clamp-3 leading-relaxed"
            : "text-gray-600 text-sm mb-4 line-clamp-2 leading-relaxed";
        
        return `
            <div class="${cardClasses}" 
                 data-id="${item.id}" role="button" tabindex="0" aria-label="View details for ${item.title}">
                <div class="relative overflow-hidden">
                    <img src="${item.image}" 
                         alt="${item.title} activity" 
                         class="${imageClasses}" 
                         loading="lazy" 
                         onerror="this.src='data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAwIiBoZWlnaHQ9IjMwMCIgdmlld0JveD0iMCAwIDQwMCAzMDAiIGZpbGw9Im5vbmUiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+CjxyZWN0IHdpZHRoPSI0MDAiIGhlaWdodD0iMzAwIiBmaWxsPSIjRjVGNUY1Ii8+CjxwYXRoIGQ9Ik0xNzUgMTI1SDE0MFYxNzVIMTc1VjE1MEgyMjVWMTc1SDI2MFYxMjVIMjI1VjEwMEgxNzVWMTI1WiIgZmlsbD0iIzk5OTk5OSIvPgo8L3N2Zz4K'; this.onerror=null;">
                    ${isFeatured ? '<div class="absolute top-3 left-3 bg-gradient-to-r from-blue-500 to-purple-600 text-white px-2 py-1 rounded-full text-xs font-semibold">Featured</div>' : ''}
                </div>
                <div class="p-6">
                    <span class="inline-flex items-center gap-1 px-3 py-1.5 ${categoryClass} text-xs font-semibold rounded-full mb-3 transition-all duration-200 group-hover:scale-105 shadow-sm">
                        ${this.formatCategory(item.category)}
                    </span>
                    <h3 class="${titleClasses} group-hover:text-blue-600 transition-colors duration-200">
                        ${item.title}
                    </h3>
                    <p class="${descriptionClasses}">
                        ${item.description}
                    </p>
                    <div class="flex justify-between items-start pt-4 border-t border-gray-100">
                        <div class="space-y-2 flex-1">
                            <div class="flex items-center gap-1 text-xs font-medium text-gray-600">
                                <span class="text-blue-500">📅</span>
                                <span>${this.formatDate(this.getActivityDate(item))}</span>
                                <span class="text-gray-400">•</span>
                                <span>${item.time}</span>
                            </div>
                            <div class="flex items-center gap-1 text-xs text-gray-600">
                                <span class="text-green-500">📍</span>
                                <span class="truncate">${item.location}</span>
                            </div>
                            ${item.age_range ? `
                            <div class="flex items-center gap-1 text-xs text-gray-600">
                                <span class="text-purple-500">👶</span>
                                <span>${item.age_range}</span>
                            </div>` : ''}
                        </div>
                        <div class="ml-4 text-right">
                            <div class="font-bold text-lg text-blue-600 group-hover:text-blue-700 transition-colors duration-200">
                                ${item.price}
                            </div>
                            ${item.price !== 'Free' ? '<div class="text-xs text-gray-500">per person</div>' : ''}
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    // Format category for display with modern icons
    formatCategory(category) {
        const categoryMap = {
            'event': '🎪 Event',
            'activity': '🎯 Activity',
            'venue': '🏢 Venue'
        };
        return categoryMap[category] || '📋 ' + category;
    }

    // Get Tailwind classes for category badges with modern color schemes
    getCategoryTailwindClasses(category) {
        const categoryClasses = {
            'event': 'bg-gradient-to-r from-pink-100 to-rose-100 text-pink-800 border border-pink-200',
            'activity': 'bg-gradient-to-r from-emerald-100 to-teal-100 text-emerald-800 border border-emerald-200', 
            'venue': 'bg-gradient-to-r from-amber-100 to-orange-100 text-amber-800 border border-amber-200'
        };
        return categoryClasses[category] || 'bg-gradient-to-r from-blue-100 to-indigo-100 text-blue-800 border border-blue-200';
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
        const mainContainer = document.querySelector('.max-w-7xl');
        if (mainContainer) {
            mainContainer.style.display = 'block';
        }
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
        const mainContainer = document.querySelector('.max-w-7xl');
        if (mainContainer) {
            mainContainer.style.display = 'none';
        }
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
            <!-- Modern Detail Header -->
            <div class="space-y-6 mb-8">
                <!-- Hero Image Section -->
                <div class="relative w-full h-64 md:h-80 lg:h-96 rounded-2xl overflow-hidden bg-gray-100 shadow-lg">
                    <img src="${item.image}" alt="${item.title}" 
                         class="w-full h-full object-cover transition-transform duration-300 hover:scale-105" 
                         onerror="this.src='data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAwIiBoZWlnaHQ9IjMwMCIgdmlld0JveD0iMCAwIDQwMCAzMDAiIGZpbGw9Im5vbmUiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+CjxyZWN0IHdpZHRoPSI0MDAiIGhlaWdodD0iMzAwIiBmaWxsPSIjRjVGNUY1Ii8+CjxwYXRoIGQ9Ik0xNzUgMTI1SDE0MFYxNzVIMTc1VjE1MEgyMjVWMTc1SDI2MFYxMjVIMjI1VjEwMEgxNzVWMTI1WiIgZmlsbD0iIzk5OTk5OSIvPgo8L3N2Zz4K'; this.onerror=null;">
                    ${this.renderImageGallery(originalActivity)}
                    
                    <!-- Category Badge Overlay -->
                    <div class="absolute top-4 left-4">
                        <span class="inline-flex items-center gap-1.5 px-3 py-1.5 ${this.getCategoryTailwindClasses(item.category)} text-sm font-semibold rounded-full shadow-lg backdrop-blur-sm">
                            ${this.formatCategory(item.category)}
                        </span>
                    </div>
                    
                    <!-- Featured Badge if applicable -->
                    ${item.featured ? `
                        <div class="absolute top-4 right-4">
                            <span class="inline-flex items-center gap-1 px-3 py-1.5 bg-gradient-to-r from-yellow-400 to-orange-500 text-white text-sm font-semibold rounded-full shadow-lg">
                                <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                                    <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                                </svg>
                                Featured
                            </span>
                        </div>
                    ` : ''}
                </div>
                
                <!-- Title and Description Section -->
                <div class="space-y-4">
                    <h1 class="text-3xl md:text-4xl lg:text-5xl font-bold text-gray-900 leading-tight" id="detailTitle">
                        ${item.title}
                    </h1>
                    <p class="text-lg md:text-xl text-gray-600 leading-relaxed max-w-4xl">
                        ${item.description}
                    </p>
                </div>
                
                <!-- Quick Info Bar -->
                <div class="flex flex-wrap items-center gap-4 p-4 bg-gradient-to-r from-blue-50 to-purple-50 rounded-xl border border-blue-100">
                    <div class="flex items-center gap-2 text-sm font-medium text-gray-700">
                        <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
                        </svg>
                        <span>${this.formatDate(this.getActivityDate(item))}</span>
                    </div>
                    <div class="flex items-center gap-2 text-sm font-medium text-gray-700">
                        <svg class="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"></path>
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"></path>
                        </svg>
                        <span>${item.location}</span>
                    </div>
                    <div class="flex items-center gap-2 text-sm font-medium text-gray-700">
                        <svg class="w-5 h-5 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                        <span>${item.time}</span>
                    </div>
                    <div class="ml-auto">
                        <span class="inline-flex items-center px-4 py-2 bg-white rounded-lg shadow-sm border border-gray-200 text-lg font-bold text-blue-600">
                            ${item.price}
                        </span>
                    </div>
                </div>
            </div>
            
            ${this.renderScheduleSection(originalActivity || item)}
            ${this.renderLocationSection(originalActivity || item)}
            ${this.renderPricingSection(originalActivity || item)}
            ${this.renderRegistrationSection(originalActivity || item)}
            ${this.renderProviderSection(originalActivity)}
            ${this.renderAdditionalInfo(originalActivity || item)}
            
            <!-- Floating Action Buttons -->
            <div class="fixed bottom-6 right-6 flex flex-col gap-3 z-50">
                <button onclick="navigator.share ? navigator.share({title: '${item.title.replace(/'/g, "\\'")}', text: '${item.description.replace(/'/g, "\\'").substring(0, 100)}...', url: window.location.href}) : this.style.display='none'" 
                        class="w-14 h-14 bg-gradient-to-r from-blue-500 to-purple-600 text-white rounded-full shadow-lg hover:shadow-xl transform hover:scale-110 transition-all duration-200 flex items-center justify-center group focus:outline-none focus:ring-4 focus:ring-blue-300 focus:ring-opacity-50"
                        title="Share this activity">
                    <svg class="w-6 h-6 group-hover:scale-110 transition-transform duration-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.367 2.684 3 3 0 00-5.367-2.684z"></path>
                    </svg>
                </button>
                
                <button onclick="window.scrollTo({top: 0, behavior: 'smooth'})" 
                        class="w-12 h-12 bg-white text-gray-600 rounded-full shadow-lg hover:shadow-xl hover:text-gray-900 transform hover:scale-110 transition-all duration-200 flex items-center justify-center border border-gray-200 focus:outline-none focus:ring-4 focus:ring-gray-300 focus:ring-opacity-50"
                        title="Back to top">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18"></path>
                    </svg>
                </button>
            </div>
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
                 class="w-16 h-12 md:w-20 md:h-14 object-cover rounded-lg cursor-pointer opacity-70 hover:opacity-100 transition-all duration-200 border-2 border-transparent hover:border-white shadow-md flex-shrink-0" 
                 onclick="this.closest('div').querySelector('img').src='${img.url}'; this.closest('div').querySelectorAll('img').forEach(i => i.classList.remove('border-white', 'opacity-100')); this.classList.add('border-white', 'opacity-100');">`
        ).join('');
        
        return `
            <div class="absolute bottom-4 left-4 right-4">
                <div class="flex gap-2 overflow-x-auto pb-2 scrollbar-hide">
                    ${thumbnails}
                </div>
            </div>
        `;
    }
    
    // Render schedule section
    renderScheduleSection(item) {
        const schedule = item.schedule || {};
        const times = schedule.times || [];
        
        let scheduleContent = `
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                <div class="bg-gradient-to-br from-blue-50 to-blue-100 p-4 rounded-xl border border-blue-200">
                    <div class="flex items-center gap-2 mb-2">
                        <svg class="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
                        </svg>
                        <span class="text-sm font-medium text-blue-700 uppercase tracking-wide">Date</span>
                    </div>
                    <div class="text-lg font-semibold text-gray-900">${item.date || schedule.startDate || 'TBD'}</div>
                </div>
                <div class="bg-gradient-to-br from-green-50 to-green-100 p-4 rounded-xl border border-green-200">
                    <div class="flex items-center gap-2 mb-2">
                        <svg class="w-5 h-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                        <span class="text-sm font-medium text-green-700 uppercase tracking-wide">Time</span>
                    </div>
                    <div class="text-lg font-semibold text-gray-900">${item.time || this.formatTime(schedule)}</div>
                </div>
        `;
        
        if (schedule.type) {
            scheduleContent += `
                <div class="bg-gradient-to-br from-purple-50 to-purple-100 p-4 rounded-xl border border-purple-200">
                    <div class="flex items-center gap-2 mb-2">
                        <svg class="w-5 h-5 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16"></path>
                        </svg>
                        <span class="text-sm font-medium text-purple-700 uppercase tracking-wide">Schedule Type</span>
                    </div>
                    <div class="text-lg font-semibold text-gray-900">${this.formatScheduleType(schedule.type)}</div>
                </div>
            `;
        }
        
        if (schedule.duration) {
            scheduleContent += `
                <div class="bg-gradient-to-br from-amber-50 to-amber-100 p-4 rounded-xl border border-amber-200">
                    <div class="flex items-center gap-2 mb-2">
                        <svg class="w-5 h-5 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                        <span class="text-sm font-medium text-amber-700 uppercase tracking-wide">Duration</span>
                    </div>
                    <div class="text-lg font-semibold text-gray-900">${schedule.duration}</div>
                </div>
            `;
        }
        
        scheduleContent += `</div>`;
        
        // Add time slots if available
        if (times.length > 1) {
            const timeSlots = times.map(time => 
                `<div class="bg-white p-3 rounded-lg border border-gray-200 shadow-sm">
                    <div class="font-medium text-gray-900">${time.startTime} - ${time.endTime}</div>
                    ${time.ageGroup ? `<div class="text-sm text-gray-600 mt-1">${time.ageGroup}</div>` : ''}
                </div>`
            ).join('');
            
            scheduleContent += `
                <div class="mt-6">
                    <h4 class="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
                        <svg class="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                        Available Time Slots
                    </h4>
                    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                        ${timeSlots}
                    </div>
                </div>
            `;
        }
        
        return `
            <div class="bg-white rounded-2xl shadow-lg border border-gray-100 p-6 mb-6">
                <h3 class="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-3">
                    <div class="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center">
                        <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
                        </svg>
                    </div>
                    Schedule & Timing
                </h3>
                ${scheduleContent}
            </div>
        `;
    }
    
    // Render location section
    renderLocationSection(item) {
        const location = item.location || {};
        
        let locationCards = `
            <div class="bg-gradient-to-br from-green-50 to-emerald-100 p-6 rounded-xl border border-green-200">
                <div class="flex items-center gap-3 mb-3">
                    <svg class="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"></path>
                    </svg>
                    <span class="text-sm font-semibold text-green-700 uppercase tracking-wide">Venue</span>
                </div>
                <div class="text-xl font-bold text-gray-900">${location.name || item.location || 'TBD'}</div>
            </div>
        `;
        
        if (location.address) {
            locationCards += `
                <div class="bg-gradient-to-br from-blue-50 to-cyan-100 p-6 rounded-xl border border-blue-200">
                    <div class="flex items-center gap-3 mb-3">
                        <svg class="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"></path>
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"></path>
                        </svg>
                        <span class="text-sm font-semibold text-blue-700 uppercase tracking-wide">Address</span>
                    </div>
                    <div class="text-lg font-semibold text-gray-900">${location.address}</div>
                    <button class="mt-3 inline-flex items-center gap-2 text-blue-600 hover:text-blue-700 font-medium text-sm transition-colors duration-200" onclick="window.open('https://maps.google.com/?q=${encodeURIComponent(location.address)}', '_blank')">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path>
                        </svg>
                        View on Maps
                    </button>
                </div>
            `;
        }
        
        if (location.neighborhood) {
            locationCards += `
                <div class="bg-gradient-to-br from-purple-50 to-pink-100 p-6 rounded-xl border border-purple-200">
                    <div class="flex items-center gap-3 mb-3">
                        <svg class="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                        <span class="text-sm font-semibold text-purple-700 uppercase tracking-wide">Neighborhood</span>
                    </div>
                    <div class="text-lg font-semibold text-gray-900">${location.neighborhood}</div>
                </div>
            `;
        }
        
        // Additional info cards
        let additionalInfo = '';
        if (location.parking) {
            additionalInfo += `
                <div class="bg-gradient-to-br from-amber-50 to-yellow-100 p-4 rounded-lg border border-amber-200">
                    <div class="flex items-center gap-2 mb-2">
                        <svg class="w-5 h-5 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7v8a2 2 0 002 2h6M8 7V5a2 2 0 012-2h4.586a1 1 0 01.707.293l4.414 4.414a1 1 0 01.293.707V15a2 2 0 01-2 2h-2M8 7H6a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2v-2"></path>
                        </svg>
                        <span class="text-sm font-medium text-amber-700 uppercase tracking-wide">Parking</span>
                    </div>
                    <div class="text-sm font-medium text-gray-900">${location.parking}</div>
                </div>
            `;
        }
        
        if (location.accessibility) {
            additionalInfo += `
                <div class="bg-gradient-to-br from-teal-50 to-cyan-100 p-4 rounded-lg border border-teal-200">
                    <div class="flex items-center gap-2 mb-2">
                        <svg class="w-5 h-5 text-teal-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"></path>
                        </svg>
                        <span class="text-sm font-medium text-teal-700 uppercase tracking-wide">Accessibility</span>
                    </div>
                    <div class="text-sm font-medium text-gray-900">${location.accessibility}</div>
                </div>
            `;
        }
        
        return `
            <div class="bg-white rounded-2xl shadow-lg border border-gray-100 p-6 mb-6">
                <h3 class="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-3">
                    <div class="w-10 h-10 bg-gradient-to-br from-green-500 to-emerald-600 rounded-xl flex items-center justify-center">
                        <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"></path>
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"></path>
                        </svg>
                    </div>
                    Location Details
                </h3>
                <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
                    ${locationCards}
                </div>
                ${additionalInfo ? `
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                        ${additionalInfo}
                    </div>
                ` : ''}
            </div>
        `;
    }
    
    // Render pricing section
    renderPricingSection(item) {
        const pricing = item.pricing || {};
        const isFree = item.price === 'Free' || pricing.type === 'free';
        
        let pricingContent = `
            <div class="text-center mb-6">
                <div class="inline-flex items-center justify-center w-20 h-20 ${isFree ? 'bg-gradient-to-br from-green-500 to-emerald-600' : 'bg-gradient-to-br from-blue-500 to-purple-600'} rounded-2xl mb-4">
                    <svg class="w-10 h-10 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        ${isFree ? 
                            '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1"></path>' :
                            '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1"></path>'
                        }
                    </svg>
                </div>
                <div class="text-4xl md:text-5xl font-bold ${isFree ? 'text-green-600' : 'text-blue-600'} mb-2">
                    ${item.price || this.formatPrice(pricing)}
                </div>
                ${!isFree ? '<div class="text-gray-600 text-lg">per person</div>' : ''}
            </div>
        `;
        
        // Additional pricing details
        let detailsCards = '';
        
        if (pricing.description) {
            detailsCards += `
                <div class="bg-gradient-to-br from-blue-50 to-indigo-100 p-4 rounded-xl border border-blue-200">
                    <div class="flex items-center gap-2 mb-2">
                        <svg class="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                        <span class="text-sm font-medium text-blue-700 uppercase tracking-wide">Details</span>
                    </div>
                    <div class="text-sm font-medium text-gray-900">${pricing.description}</div>
                </div>
            `;
        }
        
        if (pricing.includesSupplies) {
            detailsCards += `
                <div class="bg-gradient-to-br from-green-50 to-emerald-100 p-4 rounded-xl border border-green-200">
                    <div class="flex items-center gap-2 mb-2">
                        <svg class="w-5 h-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                        <span class="text-sm font-medium text-green-700 uppercase tracking-wide">Included</span>
                    </div>
                    <div class="text-sm font-medium text-gray-900">All supplies included</div>
                </div>
            `;
        }
        
        if (detailsCards) {
            pricingContent += `
                <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                    ${detailsCards}
                </div>
            `;
        }
        
        // Add discounts if available
        if (pricing.discounts && pricing.discounts.length > 0) {
            const discounts = pricing.discounts.map(discount => 
                `<div class="bg-gradient-to-r from-yellow-100 to-orange-100 border border-yellow-200 rounded-lg p-3">
                    <div class="flex items-center gap-2">
                        <svg class="w-5 h-5 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"></path>
                        </svg>
                        <span class="font-medium text-yellow-800">${discount.description || discount.type}</span>
                    </div>
                </div>`
            ).join('');
            
            pricingContent += `
                <div class="space-y-3">
                    <h4 class="text-lg font-semibold text-gray-900 flex items-center gap-2">
                        <svg class="w-5 h-5 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"></path>
                        </svg>
                        Available Discounts
                    </h4>
                    <div class="space-y-2">
                        ${discounts}
                    </div>
                </div>
            `;
        }
        
        return `
            <div class="bg-white rounded-2xl shadow-lg border border-gray-100 p-6 mb-6">
                <h3 class="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-3">
                    <div class="w-10 h-10 ${isFree ? 'bg-gradient-to-br from-green-500 to-emerald-600' : 'bg-gradient-to-br from-blue-500 to-purple-600'} rounded-xl flex items-center justify-center">
                        <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1"></path>
                        </svg>
                    </div>
                    Pricing Information
                </h3>
                ${pricingContent}
            </div>
        `;
    }
    
    // Render registration section
    renderRegistrationSection(item) {
        const registration = item.registration || {};
        const isRequired = registration.required !== false;
        const status = registration.status || 'open';
        
        // Status styling
        const statusConfig = {
            'open': {
                bg: 'from-green-100 to-emerald-100',
                border: 'border-green-200',
                text: 'text-green-800',
                icon: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z'
            },
            'waitlist': {
                bg: 'from-yellow-100 to-amber-100',
                border: 'border-yellow-200',
                text: 'text-yellow-800',
                icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z'
            },
            'closed': {
                bg: 'from-red-100 to-rose-100',
                border: 'border-red-200',
                text: 'text-red-800',
                icon: 'M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z'
            },
            'sold-out': {
                bg: 'from-gray-100 to-slate-100',
                border: 'border-gray-200',
                text: 'text-gray-800',
                icon: 'M6 18L18 6M6 6l12 12'
            }
        };
        
        const currentStatus = statusConfig[status] || statusConfig['open'];
        
        let registrationContent = `
            <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
                <div class="bg-gradient-to-br from-blue-50 to-indigo-100 p-6 rounded-xl border border-blue-200">
                    <div class="flex items-center gap-3 mb-3">
                        <svg class="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                        </svg>
                        <span class="text-sm font-semibold text-blue-700 uppercase tracking-wide">Registration</span>
                    </div>
                    <div class="text-xl font-bold text-gray-900">
                        ${isRequired ? 'Required' : 'Not Required'}
                    </div>
                </div>
                
                <div class="bg-gradient-to-br ${currentStatus.bg} p-6 rounded-xl border ${currentStatus.border}">
                    <div class="flex items-center gap-3 mb-3">
                        <svg class="w-6 h-6 ${currentStatus.text}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="${currentStatus.icon}"></path>
                        </svg>
                        <span class="text-sm font-semibold ${currentStatus.text} uppercase tracking-wide">Status</span>
                    </div>
                    <div class="text-xl font-bold text-gray-900">
                        ${this.formatRegistrationStatus(status)}
                    </div>
                </div>
        `;
        
        if (registration.deadline) {
            registrationContent += `
                <div class="bg-gradient-to-br from-purple-50 to-pink-100 p-6 rounded-xl border border-purple-200">
                    <div class="flex items-center gap-3 mb-3">
                        <svg class="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
                        </svg>
                        <span class="text-sm font-semibold text-purple-700 uppercase tracking-wide">Deadline</span>
                    </div>
                    <div class="text-xl font-bold text-gray-900">${registration.deadline}</div>
                </div>
            `;
        }
        
        registrationContent += `</div>`;
        
        // Add action buttons and contact methods
        if (registration.phone || registration.email || registration.url) {
            let actionButtons = '';
            
            if (registration.url && status === 'open') {
                actionButtons += `
                    <a href="${registration.url}" target="_blank" 
                       class="inline-flex items-center justify-center gap-3 px-8 py-4 bg-gradient-to-r from-blue-600 to-purple-600 text-white font-semibold rounded-xl shadow-lg hover:shadow-xl transform hover:-translate-y-1 transition-all duration-200 focus:outline-none focus:ring-4 focus:ring-blue-300 focus:ring-opacity-50 group">
                        <svg class="w-6 h-6 group-hover:scale-110 transition-transform duration-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9v-9m0 9c-1.657 0-3-4.03-3-9s1.343-9 3-9m0 9c1.657 0 3-4.03 3-9s-1.343 9-3 9"></path>
                        </svg>
                        Register Online
                        <svg class="w-5 h-5 group-hover:translate-x-1 transition-transform duration-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path>
                        </svg>
                    </a>
                `;
            } else if (registration.url && status === 'waitlist') {
                actionButtons += `
                    <a href="${registration.url}" target="_blank" 
                       class="inline-flex items-center justify-center gap-3 px-8 py-4 bg-gradient-to-r from-yellow-500 to-orange-500 text-white font-semibold rounded-xl shadow-lg hover:shadow-xl transform hover:-translate-y-1 transition-all duration-200 focus:outline-none focus:ring-4 focus:ring-yellow-300 focus:ring-opacity-50 group">
                        <svg class="w-6 h-6 group-hover:scale-110 transition-transform duration-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                        Join Waitlist
                        <svg class="w-5 h-5 group-hover:translate-x-1 transition-transform duration-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path>
                        </svg>
                    </a>
                `;
            }
            
            // Contact buttons
            let contactButtons = '';
            if (registration.phone) {
                contactButtons += `
                    <a href="tel:${registration.phone}" 
                       class="inline-flex items-center justify-center gap-2 px-6 py-3 bg-white text-gray-700 border-2 border-gray-200 font-medium rounded-xl hover:border-blue-300 hover:text-blue-600 hover:bg-blue-50 transition-all duration-200 focus:outline-none focus:ring-4 focus:ring-blue-100 group">
                        <svg class="w-5 h-5 group-hover:scale-110 transition-transform duration-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z"></path>
                        </svg>
                        ${registration.phone}
                    </a>
                `;
            }
            
            if (registration.email) {
                contactButtons += `
                    <a href="mailto:${registration.email}" 
                       class="inline-flex items-center justify-center gap-2 px-6 py-3 bg-white text-gray-700 border-2 border-gray-200 font-medium rounded-xl hover:border-green-300 hover:text-green-600 hover:bg-green-50 transition-all duration-200 focus:outline-none focus:ring-4 focus:ring-green-100 group">
                        <svg class="w-5 h-5 group-hover:scale-110 transition-transform duration-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"></path>
                        </svg>
                        Email Us
                    </a>
                `;
            }
            
            registrationContent += `
                <div class="space-y-4">
                    ${actionButtons ? `
                        <div class="text-center">
                            ${actionButtons}
                        </div>
                    ` : ''}
                    ${contactButtons ? `
                        <div class="flex flex-wrap justify-center gap-3">
                            ${contactButtons}
                        </div>
                    ` : ''}
                </div>
            `;
        }
        
        return `
            <div class="bg-white rounded-2xl shadow-lg border border-gray-100 p-6 mb-6">
                <h3 class="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-3">
                    <div class="w-10 h-10 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-xl flex items-center justify-center">
                        <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                        </svg>
                    </div>
                    Registration & Contact
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
            <div class="bg-white rounded-2xl shadow-lg border border-gray-100 p-6 mb-6">
                <h3 class="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-3">
                    <div class="w-10 h-10 bg-gradient-to-br from-teal-500 to-cyan-600 rounded-xl flex items-center justify-center">
                        <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"></path>
                        </svg>
                    </div>
                    About the Provider
                </h3>
                
                <div class="flex items-start gap-6 p-6 bg-gradient-to-br from-gray-50 to-blue-50 rounded-xl border border-gray-200">
                    <div class="flex-shrink-0">
                        <div class="w-16 h-16 bg-gradient-to-br from-teal-500 to-cyan-600 rounded-2xl flex items-center justify-center shadow-lg">
                            <span class="text-2xl font-bold text-white">
                                ${provider.name.charAt(0).toUpperCase()}
                            </span>
                        </div>
                    </div>
                    
                    <div class="flex-1 min-w-0">
                        <h4 class="text-xl font-bold text-gray-900 mb-2">${provider.name}</h4>
                        <p class="text-gray-600 leading-relaxed mb-4">${provider.description || provider.type}</p>
                        
                        ${provider.website ? `
                            <a href="${provider.website}" target="_blank" 
                               class="inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-teal-600 to-cyan-600 text-white font-medium rounded-lg hover:from-teal-700 hover:to-cyan-700 transform hover:-translate-y-0.5 transition-all duration-200 shadow-md hover:shadow-lg focus:outline-none focus:ring-4 focus:ring-teal-300 focus:ring-opacity-50 group">
                                <svg class="w-5 h-5 group-hover:scale-110 transition-transform duration-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9v-9m0 9c-1.657 0-3-4.03-3-9s1.343-9 3-9m0 9c1.657 0 3-4.03 3-9s-1.343 9-3 9"></path>
                                </svg>
                                Visit Website
                                <svg class="w-4 h-4 group-hover:translate-x-1 transition-transform duration-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path>
                                </svg>
                            </a>
                        ` : ''}
                    </div>
                </div>
            </div>
        `;
    }
    
    // Render additional information
    renderAdditionalInfo(item) {
        let sections = [];
        
        // Age groups
        if (item.age_range || (item.ageGroups && item.ageGroups.length > 0)) {
            const ageGroups = item.ageGroups ? 
                item.ageGroups.map(ag => ag.description || ag.category).join(', ') : 
                item.age_range;
                
            const ageGroupBadges = ageGroups.split(',').map(age => 
                `<span class="inline-flex items-center px-3 py-1.5 bg-gradient-to-r from-purple-100 to-pink-100 text-purple-800 text-sm font-medium rounded-full border border-purple-200">
                    <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"></path>
                    </svg>
                    ${age.trim()}
                </span>`
            ).join('');
                
            sections.push(`
                <div class="bg-white rounded-2xl shadow-lg border border-gray-100 p-6">
                    <h3 class="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-3">
                        <div class="w-10 h-10 bg-gradient-to-br from-purple-500 to-pink-600 rounded-xl flex items-center justify-center">
                            <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"></path>
                            </svg>
                        </div>
                        Age Groups
                    </h3>
                    <div class="flex flex-wrap gap-3">
                        ${ageGroupBadges}
                    </div>
                </div>
            `);
        }
        
        // Tags
        if (item.tags && item.tags.length > 0) {
            const tagBadges = item.tags.map(tag => 
                `<span class="inline-flex items-center px-3 py-1.5 bg-gradient-to-r from-blue-100 to-cyan-100 text-blue-800 text-sm font-medium rounded-full border border-blue-200">
                    <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"></path>
                    </svg>
                    ${tag}
                </span>`
            ).join('');
            
            sections.push(`
                <div class="bg-white rounded-2xl shadow-lg border border-gray-100 p-6">
                    <h3 class="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-3">
                        <div class="w-10 h-10 bg-gradient-to-br from-blue-500 to-cyan-600 rounded-xl flex items-center justify-center">
                            <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"></path>
                            </svg>
                        </div>
                        Tags
                    </h3>
                    <div class="flex flex-wrap gap-3">
                        ${tagBadges}
                    </div>
                </div>
            `);
        }
        
        return sections.length > 0 ? `
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
                ${sections.join('')}
            </div>
        ` : '';
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
            'open': 'Open for Registration',
            'waitlist': 'Join Waitlist',
            'closed': 'Registration Closed',
            'sold-out': 'Sold Out'
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
            const isSelected = tab.date === this.selectedDate;
            const hasActivities = tab.count > 0 || tab.date === 'all';
            
            // Base classes for all tabs
            let baseClasses = 'flex-shrink-0 whitespace-nowrap flex items-center gap-1 px-3 py-2 rounded-lg text-sm font-medium cursor-pointer transition-all duration-200 ease-in-out relative focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2';
            
            // Determine styling based on state
            let stateClasses = '';
            
            if (!hasActivities && tab.date !== 'all') {
                // No activities - disabled state
                stateClasses = 'opacity-60 cursor-default bg-gray-50 text-gray-400 border border-gray-100';
            } else if (isSelected) {
                if (tab.isToday) {
                    // Today + selected
                    stateClasses = 'bg-gradient-to-r from-blue-600 to-purple-600 text-white border border-blue-600 font-semibold shadow-md';
                } else if (tab.isWeekend) {
                    // Weekend + selected
                    stateClasses = 'bg-gradient-to-r from-indigo-600 to-cyan-600 text-white border border-indigo-600 font-semibold shadow-md';
                } else {
                    // Regular selected
                    stateClasses = 'bg-blue-600 text-white border border-blue-600 font-semibold shadow-md';
                }
            } else {
                if (tab.isToday) {
                    // Today but not selected
                    stateClasses = 'bg-gradient-to-r from-blue-100 to-purple-100 text-blue-800 border border-blue-200 hover:from-blue-200 hover:to-purple-200 hover:text-blue-900 hover:-translate-y-0.5 hover:shadow-sm';
                } else if (tab.isWeekend) {
                    // Weekend but not selected
                    stateClasses = 'bg-gradient-to-r from-indigo-50 to-cyan-50 text-indigo-700 border border-indigo-200 hover:from-indigo-100 hover:to-cyan-100 hover:text-indigo-800 hover:-translate-y-0.5 hover:shadow-sm';
                } else {
                    // Regular weekday
                    stateClasses = 'bg-white text-gray-700 border border-gray-200 hover:bg-gray-50 hover:text-gray-900 hover:-translate-y-0.5 hover:shadow-sm';
                }
            }
            
            const finalClasses = `${baseClasses} ${stateClasses}`;
            
            const countText = tab.count > 0 ? `${tab.count} activities` : 'no activities';
            const ariaLabel = `${tab.label}, ${countText}${isSelected ? ', selected' : ''}`;
            
            // Count badge styling
            let countBadgeClasses = 'min-w-4 h-4 text-xs font-semibold rounded-full flex items-center justify-center leading-none';
            
            if (!hasActivities && tab.date !== 'all') {
                countBadgeClasses += ' bg-gray-200 text-gray-400';
            } else if (isSelected) {
                countBadgeClasses += ' bg-white bg-opacity-25 text-white';
            } else if (tab.isToday) {
                countBadgeClasses += ' bg-blue-200 text-blue-800';
            } else if (tab.isWeekend) {
                countBadgeClasses += ' bg-indigo-200 text-indigo-700';
            } else {
                countBadgeClasses += ' bg-gray-200 text-gray-600';
            }
            
            return `
                <button class="${finalClasses}" 
                        data-date="${tab.date}"
                        role="tab"
                        aria-selected="${isSelected}"
                        aria-label="${ariaLabel}"
                        ${!hasActivities && tab.date !== 'all' ? 'disabled' : ''}>
                    <span>${tab.label}</span>
                    ${tab.count > 0 ? `<span class="${countBadgeClasses}" aria-hidden="true">${tab.count}</span>` : ''}
                </button>
            `;
        }).join('');
        
        // Add click event listeners to new tabs
        dateTabsContainer.querySelectorAll('button[data-date]:not([disabled])').forEach(tab => {
            tab.addEventListener('click', (e) => {
                const date = e.target.closest('button').dataset.date;
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