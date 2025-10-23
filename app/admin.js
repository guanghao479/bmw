// Admin interface for Seattle Family Activities source management

class SourceManagementAdmin {
    constructor() {
        this.apiBaseUrl = this.detectEnvironment();
        this.currentTab = 'sources';
        this.sources = {
            active: [],
            all: []
        };
        this.schemas = {};
        this.pendingEvents = [];

        this.init();
    }

    // Detect if we're in development or production
    detectEnvironment() {
        const hostname = window.location.hostname;
        const isLocal = hostname === 'localhost' || hostname === '127.0.0.1';

        if (isLocal) {
            // Local development - use SAM CLI local API Gateway endpoint
            return 'http://127.0.0.1:3000/api';
        } else if (hostname.includes('192.168') || hostname.includes('github.dev')) {
            // Other development environments - use development endpoints
            return 'http://localhost:3000/api';
        } else {
            // Production - use actual AWS API Gateway endpoints
            return 'https://qg8c2jt6se.execute-api.us-west-2.amazonaws.com/prod/api';
        }
    }

    init() {
        this.setupEventListeners();
        this.loadInitialData();
    }

    setupEventListeners() {
        // Tab switching
        document.querySelectorAll('.tab-button').forEach(button => {
            button.addEventListener('click', (e) => {
                this.switchTab(e.target.dataset.tab);
            });
        });


        // Event crawling form
        const crawlingForm = document.getElementById('crawling-form');
        if (crawlingForm) {
            crawlingForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.submitCrawlRequest();
            });
        }

        // Schema type selector
        const schemaSelect = document.getElementById('schema-type');
        if (schemaSelect) {
            schemaSelect.addEventListener('change', (e) => {
                this.handleSchemaChange(e.target.value);
            });
        }

        // Auto-refresh data every 30 seconds for events
        setInterval(() => {
            if (this.currentTab === 'crawling') {
                this.loadPendingEvents();
            }
        }, 30000);
    }






    switchTab(tabName) {
        // Update active tab button
        document.querySelectorAll('.tab-button').forEach(btn => {
            btn.classList.remove('active');
        });
        document.querySelector(`[data-tab="${tabName}"]`).classList.add('active');

        // Show/hide tab content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(`${tabName}-tab`).classList.add('active');

        this.currentTab = tabName;

        // Load data for the active tab
        switch (tabName) {
            case 'sources':
                this.loadSourceManagement();
                break;
            case 'crawling':
                this.loadSchemas();
                this.loadPendingEvents();
                break;
        }
    }

    async loadInitialData() {
        // Load initial data on startup
        await this.loadSourceManagement();
        await this.loadSchemas();
    }

    // Load real source management data from API
    async loadSourceManagement() {
        try {
            const response = await this.makeApiCall('/sources/active');
            if (response.success) {
                this.sources.active = response.data?.sources || [];
                this.displaySourceManagement();
            } else {
                console.warn('Failed to load sources:', response.error);
                // Fallback to empty data when API fails
                this.sources.active = [
                    {
                        source_id: 'demo-source',
                        source_name: 'Demo Source',
                        base_url: 'https://example.com',
                        source_type: 'auto-discovered',
                        status: 'active',
                        submitted_at: new Date().toISOString(),
                        updated_at: new Date().toISOString(),
                        success_rate: 95.0,
                        activities_found: 42
                    }
                ];
                this.displaySourceManagement();
            }
        } catch (error) {
            console.error('Error loading sources:', error);
            this.sources.active = [];
            this.displaySourceManagement();
        }
    }

    // Load extraction schemas from API
    async loadSchemas() {
        try {
            const response = await this.makeApiCall('/schemas');
            if (response.success) {
                this.schemas = response.data || {};
            } else {
                console.warn('Failed to load schemas:', response.error);
            }
        } catch (error) {
            console.error('Error loading schemas:', error);
            this.schemas = {};
        }
    }

    // Load pending events for review
    async loadPendingEvents() {
        try {
            const response = await this.makeApiCall('/events/pending');
            if (response.success) {
                this.pendingEvents = response.data || [];
                this.displayPendingEvents();
            } else {
                console.warn('Failed to load pending events:', response.error);
            }
        } catch (error) {
            console.error('Error loading pending events:', error);
            this.pendingEvents = [];
            this.displayPendingEvents();
        }
    }

    // Display source management data in the admin interface
    displaySourceManagement() {
        const container = document.getElementById('active-sources');
        if (!container) {
            console.warn('Sources container not found');
            return;
        }

        if (this.sources.active.length === 0) {
            container.innerHTML = '<div class="alert alert-info">No active sources configured yet. Submit URLs for crawling to create sources automatically.</div>';
            return;
        }

        const sourcesHtml = this.sources.active.map(source => `
            <div class="source-card" style="position: relative;">
                <div class="source-header">
                    <div class="source-title">${source.source_name}</div>
                    <div style="display: flex; gap: 0.5rem; align-items: center;">
                        <span class="status-badge status-complete">Active</span>
                        <button class="btn-secondary" onclick="adminApp.triggerReExtraction('${source.source_id}', '${source.base_url}')"
                                style="font-size: var(--font-size-xs); padding: var(--space-2) var(--space-3);">
                            Re-extract
                        </button>
                        <button onclick="adminApp.showDeleteConfirmation('${source.source_id}', '${source.source_name}', '${source.base_url}', ${source.activities_found || 0}, '${source.last_scraped || ''}')" 
                                class="btn-danger"
                                style="font-size: var(--font-size-xs); padding: var(--space-2) var(--space-3); display: flex; align-items: center; gap: var(--space-1);">
                            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                <polyline points="3,6 5,6 21,6"></polyline>
                                <path d="m19,6v14a2,2 0 0,1 -2,2H7a2,2 0 0,1 -2,-2V6m3,0V4a2,2 0 0,1 2,-2h4a2,2 0 0,1 2,2v2"></path>
                                <line x1="10" y1="11" x2="10" y2="17"></line>
                                <line x1="14" y1="11" x2="14" y2="17"></line>
                            </svg>
                            Delete
                        </button>
                    </div>
                </div>

                <!-- Performance Metrics Row -->
                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: var(--space-2); margin: var(--space-4) 0; padding: var(--space-3); 
                           background: rgba(255, 255, 255, 0.1); 
                           backdrop-filter: blur(5px); 
                           -webkit-backdrop-filter: blur(5px);
                           border: 1px solid rgba(255, 255, 255, 0.15);
                           border-radius: var(--radius-lg);">
                    <div style="text-align: center;">
                        <div style="font-size: var(--font-size-xs); color: var(--text-secondary); margin-bottom: var(--space-1); font-weight: var(--font-weight-medium);">Success Rate</div>
                        <div style="font-weight: var(--font-weight-bold); color: #10b981; font-size: var(--font-size-lg);">${source.success_rate || 100}%</div>
                    </div>
                    <div style="text-align: center;">
                        <div style="font-size: var(--font-size-xs); color: var(--text-secondary); margin-bottom: var(--space-1); font-weight: var(--font-weight-medium);">Activities</div>
                        <div style="font-weight: var(--font-weight-bold); color: var(--primary); font-size: var(--font-size-lg);">${source.activities_found || 0}</div>
                    </div>
                    <div style="text-align: center;">
                        <div style="font-size: var(--font-size-xs); color: var(--text-secondary); margin-bottom: var(--space-1); font-weight: var(--font-weight-medium);">Last Scraped</div>
                        <div style="font-weight: var(--font-weight-bold); font-size: var(--font-size-sm); color: var(--text-primary);">${this.formatDate(source.last_scraped) || 'Never'}</div>
                    </div>
                    <div style="text-align: center;">
                        <div style="font-size: var(--font-size-xs); color: var(--text-secondary); margin-bottom: var(--space-1); font-weight: var(--font-weight-medium);">Frequency</div>
                        <div style="font-weight: var(--font-weight-bold); color: var(--text-secondary); font-size: var(--font-size-sm);">${source.scraping_frequency || 'Manual'}</div>
                    </div>
                </div>

                <!-- Source Details -->
                <div class="source-meta">
                    <div class="meta-item">
                        <div class="meta-label">Type:</div>
                        ${this.formatSourceType(source.source_type)}
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Priority:</div>
                        ${this.formatPriority(source.priority)}
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Base URL:</div>
                        <a href="${source.base_url}" target="_blank">${source.base_url}</a>
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Submitted by:</div>
                        ${source.submitted_by}
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Expected Content:</div>
                        ${source.expected_content ? source.expected_content.join(', ') : 'events'}
                    </div>
                </div>
            </div>
        `).join('');

        container.innerHTML = sourcesHtml;
    }

    // Trigger re-extraction for an existing source
    async triggerReExtraction(sourceId, sourceUrl) {
        if (!confirm(`Re-extract events from ${sourceUrl}? This will create new pending events for review.`)) {
            return;
        }

        try {
            // Use the same crawl submission API but for re-extraction
            const requestData = {
                url: sourceUrl,
                schema_type: 'events', // Default to events schema
                extracted_by_user: 'admin-re-extraction',
                admin_notes: `Re-extraction from existing source: ${sourceId}`
            };

            const response = await this.makeApiCall('/crawl/submit', 'POST', requestData);

            if (response.success) {
                this.showAlert(
                    `Successfully re-extracted ${response.data.events_count} events! ` +
                    `Check the pending events tab for review.`,
                    'success'
                );
                // Refresh pending events if we're on that tab
                if (this.currentTab === 'crawling') {
                    this.loadPendingEvents();
                }
            } else {
                this.showAlert(`Re-extraction failed: ${response.error}`, 'error');
            }
        } catch (error) {
            this.showAlert(`Re-extraction failed: ${error.message}`, 'error');
        }
    }




    async makeApiCall(endpoint, method = 'GET', body = null) {
        const url = `${this.apiBaseUrl}${endpoint}`;
        const isLocal = window.location.hostname === 'localhost' ||
            window.location.hostname === '127.0.0.1';

        const options = {
            method: method,
            mode: 'cors',
            credentials: isLocal ? 'omit' : 'same-origin',
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            }
        };

        if (body && method !== 'GET') {
            options.body = JSON.stringify(body);
        }

        try {
            const response = await fetch(url, options);
            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || `HTTP ${response.status}: ${response.statusText}`);
            }

            return data;
        } catch (error) {
            console.error('API call failed:', error);

            // Provide specific error message for local development
            const isLocal = window.location.hostname === 'localhost' ||
                window.location.hostname === '127.0.0.1';

            if (isLocal && (error.name === 'TypeError' || error.message.includes('Failed to fetch'))) {
                throw new Error('Local backend is not running. Please start the SAM local API server with: sam local start-api -t ../infrastructure/cdk.out/SeattleFamilyActivitiesMVPStack.template.json --env-vars env.json --port 3000');
            }

            throw error;
        }
    }

    generateSourceId(sourceName) {
        return sourceName.toLowerCase()
            .replace(/[^a-z0-9\s]/g, '')
            .replace(/\s+/g, '-')
            .substring(0, 50);
    }



    async loadActiveSources() {
        const container = document.getElementById('active-sources');
        container.innerHTML = '<div class="alert alert-info">Loading active sources...</div>';

        try {
            const response = await this.makeApiCall('/sources/active');

            if (response.success) {
                this.sources.active = response.data || [];
                this.displayActiveSources();
            } else {
                throw new Error(response.error || 'Failed to load active sources');
            }

        } catch (error) {
            container.innerHTML = `<div class="alert alert-error">Failed to load active sources: ${error.message}</div>`;
        }
    }

    displayActiveSources() {
        const container = document.getElementById('active-sources');

        if (this.sources.active.length === 0) {
            container.innerHTML = '<div class="alert alert-info">No active sources configured yet.</div>';
            return;
        }

        const sourcesHtml = this.sources.active.map(source => `
            <div class="source-card">
                <div class="source-header">
                    <div class="source-title">${source.source_name}</div>
                    <span class="status-badge status-complete">Active</span>
                </div>
                <div class="source-meta">
                    <div class="meta-item">
                        <div class="meta-label">Type:</div>
                        ${this.formatSourceType(source.source_type)}
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Frequency:</div>
                        ${source.scraping_frequency}
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Last Scraped:</div>
                        ${this.formatDate(source.last_scraped)}
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Success Rate:</div>
                        ${source.success_rate}%
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Activities Found:</div>
                        ${source.activities_found}
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Base URL:</div>
                        <a href="${source.base_url}" target="_blank">${source.base_url}</a>
                    </div>
                </div>
            </div>
        `).join('');

        container.innerHTML = sourcesHtml;
    }

    async loadSourceManagement() {
        // Load analytics overview
        const analyticsContainer = document.getElementById('analytics-overview');
        analyticsContainer.innerHTML = '<div class="alert alert-info">Loading analytics...</div>';

        // Load active sources
        const sourcesContainer = document.getElementById('active-sources');
        sourcesContainer.innerHTML = '<div class="alert alert-info">Loading active sources...</div>';

        try {
            // Fetch both analytics and active sources data
            const [analyticsResponse, sourcesResponse] = await Promise.all([
                this.makeApiCall('/analytics'),
                this.makeApiCall('/sources/active')
            ]);

            // Display analytics overview
            if (analyticsResponse.success) {
                const analytics = analyticsResponse.data;
                this.displayAnalyticsOverview(analytics);
            } else {
                // Show default analytics when API fails
                this.displayAnalyticsOverview({
                    total_sources_submitted: 0,
                    sources_active: 0,
                    success_rate: '0%',
                    total_activities: 0
                });
            }

            // Display active sources with enhanced data
            if (sourcesResponse.success) {
                this.sources.active = sourcesResponse.data || [];
                this.displayEnhancedActiveSources();
            } else {
                // Ensure sources array is empty when API fails
                this.sources.active = [];
                this.displayEnhancedActiveSources();
            }

        } catch (error) {
            // Show default analytics when network error occurs
            this.displayAnalyticsOverview({
                total_sources_submitted: 0,
                sources_active: 0,
                success_rate: '0%',
                total_activities: 0
            });

            // Ensure sources array is empty when network error occurs
            this.sources.active = [];
            this.displayEnhancedActiveSources();
        }
    }

    displayAnalyticsOverview(analytics) {
        const container = document.getElementById('analytics-overview');
        const analyticsHtml = `
            <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: var(--space-4); margin-bottom: 2rem;">
                <div class="source-card" style="text-align: center; padding: var(--space-4);">
                    <h4 style="margin: 0 0 var(--space-2) 0; color: var(--text-secondary); font-weight: var(--font-weight-semibold); font-size: var(--font-size-sm);">Sources Submitted</h4>
                    <div style="font-size: var(--font-size-4xl); font-weight: var(--font-weight-bold); color: var(--primary);">${analytics.total_sources_submitted || 0}</div>
                    <div style="font-size: var(--font-size-xs); color: var(--text-muted); margin-top: var(--space-1);">All time</div>
                </div>
                <div class="source-card" style="text-align: center; padding: var(--space-4);">
                    <h4 style="margin: 0 0 var(--space-2) 0; color: var(--text-secondary); font-weight: var(--font-weight-semibold); font-size: var(--font-size-sm);">Active & Scraping</h4>
                    <div style="font-size: var(--font-size-4xl); font-weight: var(--font-weight-bold); color: #10b981;">${analytics.sources_active || 0}</div>
                    <div style="font-size: var(--font-size-xs); color: var(--text-muted); margin-top: var(--space-1);">Currently running</div>
                </div>
                <div class="source-card" style="text-align: center; padding: var(--space-4);">
                    <h4 style="margin: 0 0 var(--space-2) 0; color: var(--text-secondary); font-weight: var(--font-weight-semibold); font-size: var(--font-size-sm);">Success Rate</h4>
                    <div style="font-size: var(--font-size-4xl); font-weight: var(--font-weight-bold); color: #10b981;">${analytics.success_rate || '0%'}</div>
                    <div style="font-size: var(--font-size-xs); color: var(--text-muted); margin-top: var(--space-1);">Last 30 days</div>
                </div>
                <div class="source-card" style="text-align: center; padding: var(--space-4);">
                    <h4 style="margin: 0 0 var(--space-2) 0; color: var(--text-secondary); font-weight: var(--font-weight-semibold); font-size: var(--font-size-sm);">Activities Found</h4>
                    <div style="font-size: var(--font-size-4xl); font-weight: var(--font-weight-bold); color: var(--primary);">${analytics.total_activities || 0}</div>
                    <div style="font-size: var(--font-size-xs); color: var(--text-muted); margin-top: var(--space-1);">Total scraped</div>
                </div>
            </div>
        `;
        container.innerHTML = analyticsHtml;
    }

    displayEnhancedActiveSources() {
        const container = document.getElementById('active-sources');

        if (this.sources.active.length === 0) {
            container.innerHTML = '<div class="alert alert-info">No sources are currently active and scraping. Sources need to be submitted, analyzed, and activated before they appear here. Use the "Event Crawling" tab to submit new sources for analysis.</div>';
            return;
        }

        const sourcesHtml = this.sources.active.map(source => `
            <div class="source-card" style="position: relative;">
                <div class="source-header">
                    <div class="source-title">${source.source_name}</div>
                    <div style="display: flex; gap: 0.5rem; align-items: center;">
                        <span class="status-badge status-complete">Active</span>
                        <button class="btn-secondary" onclick="adminApp.triggerManualScrape('${source.source_id}')" 
                                style="font-size: var(--font-size-xs); padding: var(--space-2) var(--space-3);">
                            Scrape Now
                        </button>
                    </div>
                </div>
                
                <!-- Performance Metrics Row -->
                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: var(--space-2); margin: var(--space-4) 0; padding: var(--space-3); 
                           background: rgba(255, 255, 255, 0.1); 
                           backdrop-filter: blur(5px); 
                           -webkit-backdrop-filter: blur(5px);
                           border: 1px solid rgba(255, 255, 255, 0.15);
                           border-radius: var(--radius-lg);">
                    <div style="text-align: center;">
                        <div style="font-size: var(--font-size-xs); color: var(--text-secondary); margin-bottom: var(--space-1); font-weight: var(--font-weight-medium);">Success Rate</div>
                        <div style="font-weight: var(--font-weight-bold); color: #10b981; font-size: var(--font-size-lg);">${source.success_rate || 0}%</div>
                    </div>
                    <div style="text-align: center;">
                        <div style="font-size: var(--font-size-xs); color: var(--text-secondary); margin-bottom: var(--space-1); font-weight: var(--font-weight-medium);">Activities</div>
                        <div style="font-weight: var(--font-weight-bold); color: var(--primary); font-size: var(--font-size-lg);">${source.activities_found || 0}</div>
                    </div>
                    <div style="text-align: center;">
                        <div style="font-size: var(--font-size-xs); color: var(--text-secondary); margin-bottom: var(--space-1); font-weight: var(--font-weight-medium);">Last Scraped</div>
                        <div style="font-weight: var(--font-weight-bold); font-size: var(--font-size-sm); color: var(--text-primary);">${this.formatDate(source.last_scraped) || 'Never'}</div>
                    </div>
                    <div style="text-align: center;">
                        <div style="font-size: var(--font-size-xs); color: var(--text-secondary); margin-bottom: var(--space-1); font-weight: var(--font-weight-medium);">Status</div>
                        <div style="font-weight: var(--font-weight-bold); color: ${this.getStatusColor(source.scraping_status)}; font-size: var(--font-size-sm);">${source.scraping_status || 'Ready'}</div>
                    </div>
                </div>
                
                <!-- Source Details -->
                <div class="source-meta">
                    <div class="meta-item">
                        <div class="meta-label">Type:</div>
                        ${this.formatSourceType(source.source_type)}
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Frequency:</div>
                        ${source.scraping_frequency || 'Daily'}
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Base URL:</div>
                        <a href="${source.base_url}" target="_blank" style="color: var(--primary-color); text-decoration: none;">${this.truncateUrl(source.base_url)}</a>
                    </div>
                </div>
                
                <!-- Action Buttons -->
                <div style="margin-top: 1rem; display: flex; gap: 0.75rem; justify-content: flex-end; flex-wrap: wrap;">
                    <button onclick="adminApp.showSourceDetails('${source.source_id}')" 
                            class="btn-secondary"
                            style="font-size: var(--font-size-sm);">
                        View Details
                    </button>
                    <button onclick="adminApp.toggleSourceStatus('${source.source_id}')" 
                            class="btn-secondary"
                            style="background: linear-gradient(145deg, #f59e0b, #d97706); color: var(--text-inverse); font-size: var(--font-size-sm);">
                        Pause
                    </button>
                    <button onclick="adminApp.showDeleteConfirmation('${source.source_id}', '${source.source_name}', '${source.base_url}', ${source.activities_found || 0}, '${source.last_scraped || ''}')" 
                            class="btn-danger"
                            style="font-size: var(--font-size-sm); display: flex; align-items: center; gap: var(--space-1);">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <polyline points="3,6 5,6 21,6"></polyline>
                            <path d="m19,6v14a2,2 0 0,1 -2,2H7a2,2 0 0,1 -2,-2V6m3,0V4a2,2 0 0,1 2,-2h4a2,2 0 0,1 2,2v2"></path>
                            <line x1="10" y1="11" x2="10" y2="17"></line>
                            <line x1="14" y1="11" x2="14" y2="17"></line>
                        </svg>
                        Delete
                    </button>
                </div>
            </div>
        `).join('');

        container.innerHTML = sourcesHtml;
    }

    getStatusColor(status) {
        const colors = {
            'ready': '#10b981',
            'running': '#f59e0b',
            'completed': '#10b981',
            'failed': '#ef4444',
            'paused': '#6b7280'
        };
        return colors[status?.toLowerCase()] || '#6b7280';
    }

    truncateUrl(url) {
        if (url.length > 40) {
            return url.substring(0, 40) + '...';
        }
        return url;
    }

    async triggerManualScrape(sourceId) {
        try {
            const response = await this.makeApiCall(`/sources/${sourceId}/trigger`, 'POST');
            if (response.success) {
                this.showAlert('success', 'Manual scrape triggered successfully! Check back in a few minutes for results.');
                // Refresh the source management view
                this.loadSourceManagement();
            } else {
                throw new Error(response.error || 'Failed to trigger scrape');
            }
        } catch (error) {
            this.showAlert('error', `Failed to trigger manual scrape: ${error.message}`);
        }
    }

    showSourceDetails(sourceId) {
        // Placeholder for source details modal
        this.showAlert('info', `Source details view for ${sourceId} - Coming soon!`);
    }

    toggleSourceStatus(sourceId) {
        // Placeholder for source pause/resume
        this.showAlert('info', `Source status toggle for ${sourceId} - Coming soon!`);
    }

    async loadAnalytics() {
        const container = document.getElementById('analytics-content');
        container.innerHTML = '<div class="alert alert-info">Loading analytics...</div>';

        try {
            const response = await this.makeApiCall('/analytics');

            if (response.success) {
                const analytics = response.data;
                const analyticsHtml = `
                    <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 1rem; margin-bottom: 2rem;">
                        <div class="source-card">
                            <h4>Total Sources</h4>
                            <div style="font-size: 2rem; font-weight: bold; color: var(--primary-color);">${analytics.total_sources_submitted || 0}</div>
                        </div>
                        <div class="source-card">
                            <h4>Pending Analysis</h4>
                            <div style="font-size: 2rem; font-weight: bold; color: #f59e0b;">${analytics.sources_pending_analysis || 0}</div>
                        </div>
                        <div class="source-card">
                            <h4>Active Sources</h4>
                            <div style="font-size: 2rem; font-weight: bold; color: #10b981;">${analytics.sources_active || 0}</div>
                        </div>
                        <div class="source-card">
                            <h4>Rejected Sources</h4>
                            <div style="font-size: 2rem; font-weight: bold; color: #ef4444;">${analytics.sources_rejected || 0}</div>
                        </div>
                        <div class="source-card">
                            <h4>Success Rate</h4>
                            <div style="font-size: 2rem; font-weight: bold; color: #10b981;">${analytics.success_rate || '0%'}</div>
                        </div>
                        <div class="source-card">
                            <h4>Avg Analysis Time</h4>
                            <div style="font-size: 2rem; font-weight: bold; color: var(--primary-color);">${analytics.avg_analysis_time || 'N/A'}</div>
                        </div>
                    </div>
                    
                    <div class="alert alert-info">
                        Detailed analytics dashboard coming soon with source performance metrics, content quality scores, and scraping efficiency reports.
                    </div>
                `;

                container.innerHTML = analyticsHtml;
            } else {
                throw new Error(response.error || 'Failed to load analytics');
            }

        } catch (error) {
            container.innerHTML = `<div class="alert alert-error">Failed to load analytics: ${error.message}</div>`;
        }
    }

    formatStatus(status) {
        const statusMap = {
            'pending_analysis': 'Pending Analysis',
            'analyzing': 'Analyzing',
            'analysis_complete': 'Analysis Complete',
            'active': 'Active',
            'rejected': 'Rejected'
        };
        return statusMap[status] || status;
    }

    formatSourceType(type) {
        const typeMap = {
            'venue': 'Venue',
            'event-organizer': 'Event Organizer',
            'program-provider': 'Program Provider',
            'community-calendar': 'Community Calendar'
        };
        return typeMap[type] || type;
    }

    formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }



    // Event Crawling Methods

    async loadSchemas() {
        try {
            const response = await fetch(`${this.apiBaseUrl}/schemas`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            if (response.ok) {
                const result = await response.json();
                this.schemas = result.data;
            } else {
                console.error('Failed to load schemas:', response.statusText);
            }
        } catch (error) {
            console.error('Error loading schemas:', error);
        }
    }

    handleSchemaChange(schemaType) {
        const schemaPreview = document.getElementById('schema-preview');
        const schemaPreviewContent = document.getElementById('schema-preview-content');
        const customSchemaGroup = document.getElementById('custom-schema-group');

        if (schemaType === 'custom') {
            customSchemaGroup.style.display = 'block';
            schemaPreview.style.display = 'none';
        } else if (schemaType && this.schemas[schemaType]) {
            customSchemaGroup.style.display = 'none';
            schemaPreview.style.display = 'block';

            const schema = this.schemas[schemaType];
            schemaPreviewContent.innerHTML = `
                <strong>${schema.name}</strong><br>
                <em>${schema.description}</em><br><br>
                <strong>Fields to extract:</strong><br>
                ${this.formatSchemaFields(schema.schema)}
            `;
        } else {
            customSchemaGroup.style.display = 'none';
            schemaPreview.style.display = 'none';
        }
    }

    formatSchemaFields(schema) {
        if (!schema.properties) return 'No fields defined';

        let fieldsHtml = '';
        for (const [key, value] of Object.entries(schema.properties)) {
            if (value.type === 'array' && value.items && value.items.properties) {
                fieldsHtml += `<strong>${key}[]:</strong><br>`;
                for (const [itemKey, itemValue] of Object.entries(value.items.properties)) {
                    fieldsHtml += `&nbsp;&nbsp;• ${itemKey} (${itemValue.type})<br>`;
                }
            } else {
                fieldsHtml += `• ${key} (${value.type})<br>`;
            }
        }
        return fieldsHtml;
    }

    async submitCrawlRequest() {
        const form = document.getElementById('crawling-form');
        const submitBtn = document.getElementById('crawl-submit-btn');

        // Clear previous alerts
        document.querySelectorAll('.alert').forEach(alert => {
            if (alert.parentNode && alert.parentNode.classList.contains('form-section')) {
                alert.remove();
            }
        });

        // Validate form
        const formData = new FormData(form);
        const url = formData.get('url');
        const schemaType = formData.get('schema_type');
        const extractedByUser = formData.get('extracted_by_user');

        if (!url || !schemaType || !extractedByUser) {
            this.showAlert('Please fill in all required fields.', 'error');
            return;
        }

        // Prepare request data
        const requestData = {
            url: url,
            schema_type: schemaType,
            extracted_by_user: extractedByUser,
            admin_notes: formData.get('admin_notes') || ''
        };

        // Add custom schema if selected
        if (schemaType === 'custom') {
            const customSchemaText = formData.get('custom_schema');
            if (!customSchemaText) {
                this.showAlert('Custom schema is required when using custom schema type.', 'error');
                return;
            }

            try {
                requestData.custom_schema = JSON.parse(customSchemaText);
            } catch (error) {
                this.showAlert('Invalid JSON in custom schema. Please check the format.', 'error');
                return;
            }
        }

        // Disable submit button and show loading
        submitBtn.disabled = true;
        submitBtn.textContent = 'Extracting...';

        try {
            const response = await fetch(`${this.apiBaseUrl}/crawl/submit`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestData)
            });

            const result = await response.json();

            if (response.ok && result.success) {
                this.showAlert(
                    `Successfully extracted ${result.data.events_count} events! ` +
                    `Processing time: ${result.data.processing_time}. ` +
                    `Credits used: ${result.data.credits_used}`,
                    'success'
                );
                form.reset();
                this.handleSchemaChange(''); // Reset schema preview
                this.loadPendingEvents(); // Refresh pending events
            } else {
                this.showAlert(result.error || 'Failed to extract events from URL.', 'error');
            }
        } catch (error) {
            console.error('Error submitting crawl request:', error);
            this.showAlert('Network error. Please try again.', 'error');
        } finally {
            // Re-enable submit button
            submitBtn.disabled = false;
            submitBtn.textContent = 'Extract Events';
        }
    }

    async loadPendingEvents() {
        try {
            const response = await fetch(`${this.apiBaseUrl}/events/pending?limit=25`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            if (response.ok) {
                const result = await response.json();
                this.pendingEvents = result.data || [];
                this.renderPendingEvents();
            } else {
                console.error('Failed to load pending events:', response.statusText);
                this.renderPendingEventsError();
            }
        } catch (error) {
            console.error('Error loading pending events:', error);
            this.renderPendingEventsError();
        }
    }

    renderPendingEvents() {
        const container = document.getElementById('pending-events-container');
        const countElement = document.getElementById('pending-count');

        if (this.pendingEvents.length === 0) {
            container.innerHTML = '<div class="alert alert-info">No pending events found.</div>';
            countElement.textContent = '0 pending events';
            return;
        }

        countElement.textContent = `${this.pendingEvents.length} pending events`;

        const eventsHtml = this.pendingEvents.map(event => this.renderEventCard(event)).join('');
        container.innerHTML = eventsHtml;
    }

    renderEventCard(event) {
        const canApprove = event.can_approve;
        const hasIssues = event.conversion_issues && event.conversion_issues.length > 0;

        return `
            <div class="source-card" style="margin-bottom: 1rem;">
                <div class="source-header">
                    <div class="source-title">${this.escapeHtml(event.source_url)}</div>
                    <div style="display: flex; gap: 0.5rem;">
                        <span class="status-badge status-${event.status}">${event.status}</span>
                        <span class="status-badge">${event.schema_type}</span>
                    </div>
                </div>

                <div class="source-meta">
                    <div class="meta-item">
                        <span class="meta-label">Events Found:</span> ${event.events_count}
                    </div>
                    <div class="meta-item">
                        <span class="meta-label">Extracted By:</span> ${this.escapeHtml(event.extracted_by_user)}
                    </div>
                    <div class="meta-item">
                        <span class="meta-label">Extracted At:</span> ${new Date(event.extracted_at).toLocaleString()}
                    </div>
                    <div class="meta-item">
                        <span class="meta-label">Status:</span> ${event.status}
                    </div>
                </div>

                ${hasIssues ? `
                    <div class="alert alert-error" style="margin-top: 1rem;">
                        <strong>Conversion Issues:</strong><br>
                        ${event.conversion_issues.map(issue => `• ${this.escapeHtml(issue)}`).join('<br>')}
                    </div>
                ` : ''}

                ${event.admin_notes ? `
                    <div style="margin-top: 1rem;">
                        <strong>Notes:</strong> ${this.escapeHtml(event.admin_notes)}
                    </div>
                ` : ''}

                <div style="display: flex; gap: 0.75rem; margin-top: 1rem; flex-wrap: wrap;">
                    <button class="btn-secondary" onclick="adminApp.viewEventDetails('${event.event_id}')">
                        View Details
                    </button>
                    <button class="btn-primary"
                            onclick="adminApp.approveEvent('${event.event_id}')"
                            ${!canApprove ? 'disabled' : ''}
                            title="${!canApprove ? 'Fix conversion issues before approving' : 'Approve and publish event'}">
                        Approve
                    </button>
                    <button class="btn-secondary" onclick="adminApp.editEvent('${event.event_id}')">
                        Edit
                    </button>
                    <button class="btn-danger" onclick="adminApp.rejectEvent('${event.event_id}')">
                        Reject
                    </button>
                </div>
            </div>
        `;
    }

    renderPendingEventsError() {
        const container = document.getElementById('pending-events-container');
        const countElement = document.getElementById('pending-count');

        container.innerHTML = '<div class="alert alert-error">Failed to load pending events. Please try again.</div>';
        countElement.textContent = 'Error loading';
    }

    async viewEventDetails(eventId) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/events/${eventId}`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            if (response.ok) {
                const result = await response.json();
                this.showEventModal(result.data);
            } else {
                this.showAlert('Failed to load event details.', 'error');
            }
        } catch (error) {
            console.error('Error loading event details:', error);
            this.showAlert('Error loading event details.', 'error');
        }
    }

    showEventModal(eventData) {
        // Create glassmorphic modal overlay
        const modal = document.createElement('div');
        modal.className = 'modal-overlay';
        // Styles are now handled by CSS classes in admin.html

        const modalContent = document.createElement('div');
        modalContent.style.cssText = `
            padding: 2rem;
            max-width: 800px;
            max-height: 80vh;
            overflow-y: auto;
            margin: 1rem;
        `;

        modalContent.innerHTML = `
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem;">
                <h3 style="margin: 0; color: var(--text-primary); font-weight: var(--font-weight-bold);">Event Details</h3>
                <button onclick="this.closest('.modal-overlay').remove()" 
                        class="btn-secondary" 
                        style="background: none; border: none; font-size: 1.5rem; cursor: pointer; color: var(--text-secondary); padding: 0.5rem; border-radius: var(--radius-md); min-width: auto; min-height: auto;"
                        aria-label="Close modal">&times;</button>
            </div>

            <div style="margin-bottom: 1rem;">
                <strong style="color: var(--text-primary);">Source URL:</strong> 
                <span style="color: var(--text-secondary);">${this.escapeHtml(eventData.source_url)}</span>
            </div>

            <div style="margin-bottom: 1rem;">
                <strong style="color: var(--text-primary);">Schema Type:</strong> 
                <span style="color: var(--text-secondary);">${eventData.schema_type}</span>
            </div>

            <div style="margin-bottom: 1rem;">
                <strong style="color: var(--text-primary);">Raw Extracted Data:</strong>
                <pre style="background: linear-gradient(135deg, rgba(248, 250, 252, 0.8), rgba(241, 245, 249, 0.6)); 
                           backdrop-filter: blur(10px); 
                           -webkit-backdrop-filter: blur(10px);
                           border: 1px solid rgba(255, 255, 255, 0.3);
                           padding: 1rem; 
                           border-radius: var(--radius-lg); 
                           overflow-x: auto; 
                           font-size: var(--font-size-sm);
                           color: var(--text-secondary);
                           font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;">${JSON.stringify(eventData.raw_extracted_data, null, 2)}</pre>
            </div>

            ${eventData.conversion_preview ? `
                <div style="margin-bottom: 1rem;">
                    <strong style="color: var(--text-primary);">Conversion Preview:</strong>
                    <pre style="background: linear-gradient(135deg, rgba(16, 185, 129, 0.1), rgba(5, 150, 105, 0.05)); 
                               backdrop-filter: blur(10px); 
                               -webkit-backdrop-filter: blur(10px);
                               border: 1px solid rgba(16, 185, 129, 0.2);
                               padding: 1rem; 
                               border-radius: var(--radius-lg); 
                               overflow-x: auto; 
                               font-size: var(--font-size-sm);
                               color: #065f46;
                               font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;">${JSON.stringify(eventData.conversion_preview, null, 2)}</pre>
                </div>
            ` : ''}

            <div style="display: flex; gap: 0.75rem; margin-top: 1.5rem; justify-content: flex-end;">
                <button class="btn-secondary" onclick="this.closest('.modal-overlay').remove();">
                    Close
                </button>
                <button class="btn-primary" onclick="adminApp.approveEvent('${eventData.event_id}'); this.closest('.modal-overlay').remove();">
                    Approve Event
                </button>
            </div>
        `;

        modal.appendChild(modalContent);
        document.body.appendChild(modal);

        // Close modal when clicking outside
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.remove();
            }
        });
    }

    async approveEvent(eventId) {
        if (!confirm('Are you sure you want to approve this event? It will be published to the frontend.')) {
            return;
        }

        try {
            const response = await fetch(`${this.apiBaseUrl}/events/${eventId}/approve`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    reviewed_by: 'admin',
                    admin_notes: 'Approved via admin interface'
                })
            });

            const result = await response.json();

            if (response.ok && result.success) {
                this.showAlert('Event approved and published successfully!', 'success');
                this.loadPendingEvents(); // Refresh the list
            } else {
                this.showAlert(result.error || 'Failed to approve event.', 'error');
            }
        } catch (error) {
            console.error('Error approving event:', error);
            this.showAlert('Error approving event.', 'error');
        }
    }

    async rejectEvent(eventId) {
        const reason = prompt('Please provide a reason for rejecting this event:');
        if (!reason) return;

        try {
            const response = await fetch(`${this.apiBaseUrl}/events/${eventId}/reject`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    reviewed_by: 'admin',
                    admin_notes: reason
                })
            });

            const result = await response.json();

            if (response.ok && result.success) {
                this.showAlert('Event rejected successfully.', 'success');
                this.loadPendingEvents(); // Refresh the list
            } else {
                this.showAlert(result.error || 'Failed to reject event.', 'error');
            }
        } catch (error) {
            console.error('Error rejecting event:', error);
            this.showAlert('Error rejecting event.', 'error');
        }
    }

    async editEvent(eventId) {
        // For now, just show a simple alert - full editing would require a complex modal
        alert('Event editing interface coming soon! For now, please reject the event and re-submit with correct data.');
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    showDeleteConfirmation(sourceId, sourceName, sourceUrl, activitiesCount, lastScraped) {
        // Create glassmorphic modal overlay
        const modal = document.createElement('div');
        modal.className = 'modal-overlay';
        // Styles are now handled by CSS classes in admin.html

        const modalContent = document.createElement('div');
        modalContent.style.cssText = `
            padding: 2rem;
            max-width: 500px;
            width: 90%;
            margin: 1rem;
        `;

        const lastScrapedText = lastScraped ? this.formatDate(lastScraped) : 'Never';

        modalContent.innerHTML = `
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem;">
                <h3 style="margin: 0; color: #dc2626; display: flex; align-items: center; gap: 0.5rem; font-weight: var(--font-weight-bold);">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <circle cx="12" cy="12" r="10"></circle>
                        <line x1="15" y1="9" x2="9" y2="15"></line>
                        <line x1="9" y1="9" x2="15" y2="15"></line>
                    </svg>
                    Delete Source
                </h3>
                <button onclick="this.closest('.modal-overlay').remove()" 
                        class="btn-secondary"
                        style="background: none; border: none; font-size: 1.5rem; cursor: pointer; color: var(--text-secondary); padding: 0.5rem; border-radius: var(--radius-md); min-width: auto; min-height: auto;"
                        aria-label="Close modal">&times;</button>
            </div>

            <div class="alert alert-error" style="margin-bottom: 1.5rem;">
                <strong>Warning:</strong> This action cannot be undone. All data associated with this source will be permanently deleted.
            </div>

            <div style="margin-bottom: 1.5rem;">
                <h4 style="margin: 0 0 1rem 0; color: var(--text-primary); font-weight: var(--font-weight-semibold);">Source Details</h4>
                <div style="background: linear-gradient(135deg, rgba(248, 250, 252, 0.8), rgba(241, 245, 249, 0.6)); 
                           backdrop-filter: blur(10px); 
                           -webkit-backdrop-filter: blur(10px);
                           border: 1px solid rgba(239, 68, 68, 0.2);
                           border-left: 4px solid #dc2626;
                           padding: 1rem; 
                           border-radius: var(--radius-lg);">
                    <div style="margin-bottom: 0.75rem;">
                        <strong style="color: var(--text-primary);">Name:</strong> 
                        <span style="color: var(--text-secondary);">${this.escapeHtml(sourceName)}</span>
                    </div>
                    <div style="margin-bottom: 0.75rem;">
                        <strong style="color: var(--text-primary);">URL:</strong> 
                        <span style="color: var(--text-secondary);">${this.escapeHtml(sourceUrl)}</span>
                    </div>
                    <div style="margin-bottom: 0.75rem;">
                        <strong style="color: var(--text-primary);">Activities Found:</strong> 
                        <span style="color: var(--text-secondary);">${activitiesCount}</span>
                    </div>
                    <div>
                        <strong style="color: var(--text-primary);">Last Scraped:</strong> 
                        <span style="color: var(--text-secondary);">${lastScrapedText}</span>
                    </div>
                </div>
            </div>

            <div style="margin-bottom: 1.5rem;">
                <label for="delete-confirmation-input" style="display: block; margin-bottom: 0.5rem; font-weight: var(--font-weight-semibold); color: var(--text-primary);">
                    Type the source name to confirm deletion:
                </label>
                <input type="text" 
                       id="delete-confirmation-input" 
                       placeholder="Enter source name exactly as shown above"
                       class="neuro-input"
                       style="width: 100%; box-sizing: border-box;">
                <div id="delete-confirmation-error" style="color: #dc2626; font-size: var(--font-size-xs); font-weight: var(--font-weight-medium); margin-top: var(--space-1); display: none;">
                    Source name does not match. Please type it exactly as shown above.
                </div>
            </div>

            <div style="display: flex; gap: 0.75rem; justify-content: flex-end;">
                <button onclick="this.closest('.modal-overlay').remove()" 
                        class="btn-secondary">
                    Cancel
                </button>
                <button id="delete-confirm-button"
                        onclick="adminApp.confirmDelete('${sourceId}', '${this.escapeHtml(sourceName)}', this.closest('.modal-overlay'))"
                        disabled
                        class="btn-danger"
                        style="opacity: 0.5;">
                    Delete Permanently
                </button>
            </div>
        `;

        modal.appendChild(modalContent);
        document.body.appendChild(modal);

        // Set up confirmation input validation
        const confirmationInput = modal.querySelector('#delete-confirmation-input');
        const confirmButton = modal.querySelector('#delete-confirm-button');
        const errorDiv = modal.querySelector('#delete-confirmation-error');

        confirmationInput.addEventListener('input', (e) => {
            const inputValue = e.target.value.trim();
            const isValid = inputValue === sourceName;

            confirmButton.disabled = !isValid;
            confirmButton.style.opacity = isValid ? '1' : '0.5';
            confirmButton.style.cursor = isValid ? 'pointer' : 'not-allowed';

            if (inputValue && !isValid) {
                errorDiv.style.display = 'block';
            } else {
                errorDiv.style.display = 'none';
            }
        });

        // Close modal when clicking outside
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.remove();
            }
        });

        // Focus the input
        setTimeout(() => {
            confirmationInput.focus();
        }, 100);
    }

    async confirmDelete(sourceId, sourceName, modal) {
        const confirmButton = modal.querySelector('#delete-confirm-button');
        const originalText = confirmButton.textContent;

        // Show loading state
        confirmButton.disabled = true;
        confirmButton.textContent = 'Deleting...';
        confirmButton.style.opacity = '0.7';

        try {
            await this.deleteSource(sourceId, sourceName);
            modal.remove();
        } catch (error) {
            // Re-enable button on error
            confirmButton.disabled = false;
            confirmButton.textContent = originalText;
            confirmButton.style.opacity = '1';
        }
    }

    async deleteSource(sourceId, sourceName) {
        try {
            const response = await this.makeApiCall(`/sources/${sourceId}`, 'DELETE');

            if (response.success) {
                this.showSourceAlert(
                    `Source "${sourceName}" deleted successfully! All associated data has been removed.`,
                    'success'
                );

                // Refresh the source list after successful deletion
                await this.loadSourceManagement();
            } else {
                throw new Error(response.error || 'Failed to delete source');
            }
        } catch (error) {
            console.error('Error deleting source:', error);
            this.showSourceAlert(
                `Failed to delete source "${sourceName}": ${error.message}`,
                'error'
            );
            throw error;
        }
    }

    showSourceAlert(message, type = 'info') {
        // Remove existing alerts in sources tab
        document.querySelectorAll('#sources-tab .alert').forEach(alert => alert.remove());

        const alertDiv = document.createElement('div');
        alertDiv.className = `alert alert-${type}`;
        alertDiv.textContent = message;

        // Insert at the top of the sources tab
        const sourcesTab = document.getElementById('sources-tab');
        const firstFormSection = sourcesTab.querySelector('.form-section');
        if (firstFormSection) {
            firstFormSection.insertBefore(alertDiv, firstFormSection.firstChild);
        }

        // Auto-remove success alerts after 5 seconds
        if (type === 'success') {
            setTimeout(() => {
                if (alertDiv.parentNode) {
                    alertDiv.remove();
                }
            }, 5000);
        }

        // Scroll to top to show the alert
        sourcesTab.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }

    showAlert(message, type = 'info') {
        // Remove existing alerts in crawling tab
        document.querySelectorAll('#crawling-tab .alert').forEach(alert => alert.remove());

        const alertDiv = document.createElement('div');
        alertDiv.className = `alert alert-${type}`;
        alertDiv.textContent = message;

        // Insert at the top of the first form section in crawling tab
        const firstFormSection = document.querySelector('#crawling-tab .form-section');
        if (firstFormSection) {
            firstFormSection.insertBefore(alertDiv, firstFormSection.firstChild);
        }

        // Auto-remove success alerts after 5 seconds
        if (type === 'success') {
            setTimeout(() => {
                alertDiv.remove();
            }, 5000);
        }
    }

    // Test environment detection (for debugging)
    testEnvironmentDetection() {
        console.log('Admin Environment Detection Test:');
        console.log('- Hostname:', window.location.hostname);
        console.log('- API Base URL:', this.apiBaseUrl);

        const isLocal = window.location.hostname === 'localhost' ||
            window.location.hostname === '127.0.0.1';
        console.log('- Is Local:', isLocal);

        return {
            hostname: window.location.hostname,
            apiBaseUrl: this.apiBaseUrl,
            isLocal: isLocal
        };
    }

    // Test local backend connection
    async testLocalBackendConnection() {
        const isLocal = window.location.hostname === 'localhost' ||
            window.location.hostname === '127.0.0.1';

        if (!isLocal) {
            console.log('Not in local development mode');
            return false;
        }

        try {
            console.log('Testing admin connection to local backend...');
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
                this.showAlert('Local backend connection successful', 'success');
                return true;
            } else {
                console.log('❌ Local backend responded with error:', response.status);
                this.showAlert(`Local backend error: ${response.status}`, 'error');
                return false;
            }
        } catch (error) {
            console.log('❌ Local backend connection failed:', error.message);
            this.showAlert('Local backend unavailable. Please start SAM local API server.', 'error');
            return false;
        }
    }
}


// Global function for refreshing pending events (called from HTML)
function loadPendingEvents() {
    if (window.adminApp) {
        window.adminApp.loadPendingEvents();
    }
}

// Global functions for testing (accessible from browser console)
window.testAdminEnvironment = () => {
    if (window.adminApp) {
        return window.adminApp.testEnvironmentDetection();
    }
    console.log('Admin app not initialized yet');
    return null;
};

window.testAdminLocalBackend = async () => {
    if (window.adminApp) {
        return await window.adminApp.testLocalBackendConnection();
    }
    console.log('Admin app not initialized yet');
    return false;
};

// Initialize the admin interface when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.adminApp = new SourceManagementAdmin();
});