// Family Events App - Dynamic Content Loading and Interaction
class FamilyEventsApp {
    constructor() {
        this.allData = [];
        this.currentFilter = 'all';
        this.searchTerm = '';
        this.lastUpdated = null;
        this.refreshInterval = null;
        this.config = this.loadConfiguration();
        
        this.init();
    }

    // Load configuration based on environment
    loadConfiguration() {
        const isDevelopment = window.location.hostname === 'localhost' || 
                             window.location.hostname === '127.0.0.1' ||
                             window.location.hostname.includes('github.dev');
        
        const baseConfig = {
            refreshIntervalMs: 30 * 60 * 1000, // 30 minutes
            retryAttempts: 3,
            retryDelay: 1000,
            cacheKey: 'familyEvents_cached_data',
            cacheTimestamp: 'familyEvents_cache_timestamp',
            maxCacheAge: 24 * 60 * 60 * 1000, // 24 hours
            environment: isDevelopment ? 'development' : 'production'
        };

        // Environment-specific configurations
        if (isDevelopment) {
            return {
                ...baseConfig,
                s3Endpoint: 'https://seattle-family-activities-mvp-data-usw2.s3.us-west-2.amazonaws.com/activities/latest.json',
                refreshIntervalMs: 5 * 60 * 1000, // 5 minutes for development
                debugMode: true
            };
        } else {
            return {
                ...baseConfig,
                s3Endpoint: 'https://seattle-family-activities-mvp-data-usw2.s3.us-west-2.amazonaws.com/activities/latest.json',
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

    // Load data from S3 with offline fallback
    async loadData() {
        try {
            // Try to fetch fresh data from S3
            const freshData = await this.fetchFromS3();
            if (freshData) {
                this.processData(freshData);
                this.cacheData(freshData);
                const count = this.allData.length;
                this.showDataStatus(`Fresh data loaded: ${count} activities (${this.config.environment})`, 'success');
                return;
            }
        } catch (error) {
            console.warn('Failed to fetch fresh data:', error);
            this.showDataStatus(`Using cached data (${this.config.environment})`, 'warning');
        }

        // Fall back to cached data
        const cachedData = this.getCachedData();
        if (cachedData) {
            this.processData(cachedData);
            this.showDataStatus('Loaded from cache', 'info');
            return;
        }

        // Final fallback to embedded sample data
        this.loadSampleData();
        this.showDataStatus('Sample data loaded', 'warning');
    }

    // Fetch data from S3 endpoint
    async fetchFromS3() {
        if (this.config.debugMode) {
            console.log(`Fetching data from S3: ${this.config.s3Endpoint}`);
        }

        for (let attempt = 1; attempt <= this.config.retryAttempts; attempt++) {
            try {
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), 10000); // 10s timeout

                const response = await fetch(this.config.s3Endpoint, {
                    signal: controller.signal,
                    headers: {
                        'Cache-Control': 'no-cache'
                    }
                });

                clearTimeout(timeoutId);

                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                }

                const data = await response.json();
                
                // Validate data structure
                if (!data.activities || !Array.isArray(data.activities)) {
                    throw new Error('Invalid data structure from S3');
                }

                if (this.config.debugMode) {
                    console.log(`S3 fetch successful: ${data.activities.length} activities, last updated: ${data.metadata?.lastUpdated}`);
                }

                return data;
            } catch (error) {
                if (this.config.debugMode) {
                    console.warn(`S3 fetch attempt ${attempt}/${this.config.retryAttempts} failed:`, error);
                }
                
                if (attempt < this.config.retryAttempts) {
                    await new Promise(resolve => setTimeout(resolve, this.config.retryDelay * attempt));
                }
            }
        }
        
        return null;
    }

    // Process data from S3 (new schema) to legacy format for compatibility
    processData(data) {
        if (!data.activities) {
            console.error('No activities in data:', data);
            return;
        }

        this.lastUpdated = data.metadata?.lastUpdated || new Date().toISOString();
        
        // Convert new schema activities to legacy format for existing UI compatibility
        this.allData = data.activities.map(activity => this.convertToLegacyFormat(activity));
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
            featured: activity.featured || false
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
        return `https://images.unsplash.com/${imageId}?w=400&h=300&fit=crop`;
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

    // Load sample data as final fallback
    loadSampleData() {
        const sampleData = {
            activities: [
                {
                    id: 'sample_1',
                    title: 'Sample Family Event',
                    description: 'This is sample data. The app will load real Seattle activities when connected.',
                    type: 'event',
                    category: 'entertainment-events',
                    schedule: { type: 'one-time', startDate: '2025-08-15', times: [{ startTime: '10:00', endTime: '16:00' }] },
                    location: { name: 'Sample Location', address: 'Seattle, WA' },
                    pricing: { type: 'free' },
                    ageGroups: [{ description: 'All ages' }],
                    featured: true
                }
            ]
        };
        
        this.processData(sampleData);
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
                // Remove active class from all buttons
                filterButtons.forEach(button => button.classList.remove('active'));
                // Add active class to clicked button
                e.target.classList.add('active');
                
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

        // Add manual refresh button
        this.addRefreshButton();
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

            return matchesFilter && matchesSearch;
        });
    }

    // Render all content
    renderContent() {
        const filteredData = this.getFilteredData();
        
        // Separate featured and regular items
        const featuredItems = filteredData.filter(item => item.featured);
        const regularItems = filteredData.filter(item => !item.featured);

        this.renderFeaturedSection(featuredItems);
        this.renderMainContent(regularItems);
    }

    // Render featured section
    renderFeaturedSection(featuredItems) {
        const featuredGrid = document.getElementById('featuredGrid');
        
        if (featuredItems.length === 0) {
            featuredGrid.innerHTML = '<p>No featured items match your criteria.</p>';
            return;
        }

        featuredGrid.innerHTML = featuredItems
            .slice(0, 4) // Limit to 4 featured items
            .map(item => this.createCardHTML(item, true))
            .join('');
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
    createCardHTML(item, isFeatured = false) {
        const categoryClass = `category-${item.category}`;
        const featuredClass = isFeatured ? 'featured' : '';
        
        return `
            <div class="card ${featuredClass}" data-id="${item.id}">
                <img src="${item.image}" alt="${item.title}" class="card-image" loading="lazy" onerror="this.src='data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAwIiBoZWlnaHQ9IjMwMCIgdmlld0JveD0iMCAwIDQwMCAzMDAiIGZpbGw9Im5vbmUiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+CjxyZWN0IHdpZHRoPSI0MDAiIGhlaWdodD0iMzAwIiBmaWxsPSIjRjVGNUY1Ii8+CjxwYXRoIGQ9Ik0xNzUgMTI1SDE0MFYxNzVIMTc1VjE1MEgyMjVWMTc1SDI2MFYxMjVIMjI1VjEwMEgxNzVWMTI1WiIgZmlsbD0iIzk5OTk5OSIvPgo8L3N2Zz4K'; this.onerror=null;">
                <div class="card-content">
                    <span class="card-category ${categoryClass}">${this.formatCategory(item.category)}</span>
                    <h3 class="card-title">${item.title}</h3>
                    <p class="card-description">${item.description}</p>
                    <div class="card-meta">
                        <div>
                            <div class="card-date">${this.formatDate(item.date)} • ${item.time}</div>
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

    // Format date for display
    formatDate(dateString) {
        // Handle recurring dates
        if (dateString.includes('day')) {
            return dateString;
        }
        
        try {
            const date = new Date(dateString);
            return date.toLocaleDateString('en-US', {
                weekday: 'short',
                month: 'short',
                day: 'numeric'
            });
        } catch {
            return dateString;
        }
    }

    // Handle card click interactions
    handleCardClick(card) {
        const itemId = parseInt(card.dataset.id);
        const item = this.allData.find(i => i.id === itemId);
        
        if (item) {
            this.showItemDetails(item);
        }
    }

    // Show detailed view of an item (mock implementation)
    showItemDetails(item) {
        // Create a modal or detailed view
        const modal = document.createElement('div');
        modal.className = 'modal-overlay';
        modal.innerHTML = `
            <div class="modal-content">
                <button class="modal-close">&times;</button>
                <img src="${item.image}" alt="${item.title}" class="modal-image">
                <div class="modal-info">
                    <h2>${item.title}</h2>
                    <p class="modal-description">${item.description}</p>
                    <div class="modal-details">
                        <p><strong>📅 When:</strong> ${item.date} at ${item.time}</p>
                        <p><strong>📍 Where:</strong> ${item.location}</p>
                        <p><strong>💰 Price:</strong> ${item.price}</p>
                        ${item.age_range ? `<p><strong>👶 Age Range:</strong> ${item.age_range}</p>` : ''}
                    </div>
                    <button class="modal-cta">Learn More</button>
                </div>
            </div>
        `;

        document.body.appendChild(modal);

        // Add modal styles if not already present
        if (!document.querySelector('#modal-styles')) {
            this.addModalStyles();
        }

        // Close modal functionality
        modal.addEventListener('click', (e) => {
            if (e.target === modal || e.target.classList.contains('modal-close')) {
                modal.remove();
            }
        });

        // Animate modal in
        setTimeout(() => modal.classList.add('show'), 10);
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
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new FamilyEventsApp();
});

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