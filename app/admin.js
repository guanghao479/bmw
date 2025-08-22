// Admin interface for Seattle Family Activities source management

class SourceManagementAdmin {
    constructor() {
        this.apiBaseUrl = this.detectEnvironment();
        this.currentTab = 'submit';
        this.sources = {
            pending: [],
            active: [],
            all: []
        };
        
        this.init();
    }

    // Detect if we're in development or production
    detectEnvironment() {
        const hostname = window.location.hostname;
        if (hostname === 'localhost' || hostname === '127.0.0.1' || hostname.includes('192.168')) {
            // Development - use local mock API or development endpoints
            return 'http://localhost:3000/api';
        } else {
            // Production - use actual AWS API Gateway endpoints
            return 'https://qg8c2jt6se.execute-api.us-west-2.amazonaws.com/prod/api';
        }
    }

    init() {
        this.setupEventListeners();
        this.loadInitialData();
        this.setupFormValidation();
    }

    setupEventListeners() {
        // Tab switching
        document.querySelectorAll('.tab-button').forEach(button => {
            button.addEventListener('click', (e) => {
                this.switchTab(e.target.dataset.tab);
            });
        });

        // Form submission
        const form = document.getElementById('source-submission-form');
        if (form) {
            form.addEventListener('submit', (e) => {
                e.preventDefault();
                this.submitSource();
            });
        }

        // Auto-refresh data every 30 seconds for pending sources
        setInterval(() => {
            if (this.currentTab === 'pending') {
                this.loadPendingSources();
            }
        }, 30000);
    }

    setupFormValidation() {
        const form = document.getElementById('source-submission-form');
        const inputs = form.querySelectorAll('input[required], select[required]');
        
        inputs.forEach(input => {
            input.addEventListener('blur', () => this.validateField(input));
            input.addEventListener('input', () => this.clearFieldError(input));
        });
    }

    validateField(field) {
        const value = field.value.trim();
        let isValid = true;
        let errorMessage = '';

        // Remove existing error styling
        field.classList.remove('error');
        this.removeFieldError(field);

        // Required field validation
        if (field.hasAttribute('required') && !value) {
            isValid = false;
            errorMessage = 'This field is required';
        }

        // URL validation
        if (field.type === 'url' && value) {
            try {
                new URL(value);
            } catch {
                isValid = false;
                errorMessage = 'Please enter a valid URL';
            }
        }

        // Email validation
        if (field.type === 'email' && value) {
            const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
            if (!emailPattern.test(value)) {
                isValid = false;
                errorMessage = 'Please enter a valid email address';
            }
        }

        // Expected content validation (at least one checkbox must be checked)
        if (field.name === 'expected_content') {
            const checkedBoxes = document.querySelectorAll('input[name="expected_content"]:checked');
            if (checkedBoxes.length === 0) {
                isValid = false;
                errorMessage = 'Please select at least one content type';
            }
        }

        if (!isValid) {
            this.showFieldError(field, errorMessage);
        }

        return isValid;
    }

    showFieldError(field, message) {
        field.classList.add('error');
        
        const errorDiv = document.createElement('div');
        errorDiv.className = 'field-error';
        errorDiv.textContent = message;
        errorDiv.style.color = 'var(--error-color)';
        errorDiv.style.fontSize = '0.85rem';
        errorDiv.style.marginTop = '0.25rem';
        
        field.parentNode.appendChild(errorDiv);
    }

    removeFieldError(field) {
        const errorDiv = field.parentNode.querySelector('.field-error');
        if (errorDiv) {
            errorDiv.remove();
        }
    }

    clearFieldError(field) {
        field.classList.remove('error');
        this.removeFieldError(field);
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
            case 'pending':
                this.loadPendingSources();
                break;
            case 'active':
                this.loadActiveSources();
                break;
            case 'analytics':
                this.loadAnalytics();
                break;
        }
    }

    async loadInitialData() {
        // Load sample data for demo purposes
        this.sources.pending = [
            {
                source_id: 'seattle-childrens-theatre',
                source_name: "Seattle Children's Theatre",
                base_url: 'https://sct.org',
                source_type: 'event-organizer',
                status: 'pending_analysis',
                submitted_at: '2025-08-20T10:00:00Z',
                submitted_by: 'founder@seattlefamilyactivities.com',
                expected_content: ['events', 'classes'],
                hint_urls: ['https://sct.org/events', 'https://sct.org/classes']
            },
            {
                source_id: 'pacific-science-center',
                source_name: 'Pacific Science Center',
                base_url: 'https://pacificsciencecenter.org',
                source_type: 'venue',
                status: 'analyzing',
                submitted_at: '2025-08-20T09:30:00Z',
                submitted_by: 'founder@seattlefamilyactivities.com',
                expected_content: ['events', 'classes', 'attractions'],
                hint_urls: ['https://pacificsciencecenter.org/events']
            }
        ];

        this.sources.active = [
            {
                source_id: 'seattle-parks',
                source_name: 'Seattle Parks and Recreation',
                base_url: 'https://seattle.gov/parks',
                source_type: 'program-provider',
                status: 'active',
                activated_at: '2025-08-15T14:00:00Z',
                last_scraped: '2025-08-20T06:00:00Z',
                scraping_frequency: 'daily',
                activities_found: 127,
                success_rate: 98.5
            }
        ];
    }

    async submitSource() {
        const form = document.getElementById('source-submission-form');
        const submitBtn = document.getElementById('submit-btn');
        
        // Validate form
        const isValid = this.validateForm();
        if (!isValid) {
            this.showAlert('error', 'Please fix the errors above before submitting.');
            return;
        }

        // Disable submit button
        submitBtn.disabled = true;
        submitBtn.textContent = 'Submitting...';

        try {
            const formData = this.getFormData();
            
            // Make real API call to submit the source
            const response = await this.makeApiCall('/sources/submit', 'POST', formData);
            
            if (response.success) {
                this.showAlert('success', 'Source submitted successfully! It will be analyzed automatically.');
                form.reset();
                
                // Refresh pending sources to show the newly submitted source
                if (this.currentTab === 'pending') {
                    this.loadPendingSources();
                }
            } else {
                throw new Error(response.error || 'Failed to submit source');
            }
            
        } catch (error) {
            this.showAlert('error', `Failed to submit source: ${error.message}`);
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Submit Source for Analysis';
        }
    }

    validateForm() {
        const form = document.getElementById('source-submission-form');
        const requiredFields = form.querySelectorAll('input[required], select[required]');
        let isValid = true;

        // Clear previous alerts
        this.clearAlerts();

        // Validate each required field
        requiredFields.forEach(field => {
            if (!this.validateField(field)) {
                isValid = false;
            }
        });

        // Validate expected content checkboxes
        const checkedBoxes = form.querySelectorAll('input[name="expected_content"]:checked');
        if (checkedBoxes.length === 0) {
            isValid = false;
            this.showAlert('error', 'Please select at least one expected content type.');
        }

        return isValid;
    }

    getFormData() {
        const form = document.getElementById('source-submission-form');
        const formData = new FormData(form);
        
        const data = {
            source_name: formData.get('source_name'),
            base_url: formData.get('base_url'),
            source_type: formData.get('source_type'),
            priority: formData.get('priority') || 'medium',
            submitted_by: formData.get('submitted_by'),
            notes: formData.get('notes') || '',
            expected_content: [],
            hint_urls: []
        };

        // Get expected content checkboxes
        const expectedContentBoxes = form.querySelectorAll('input[name="expected_content"]:checked');
        data.expected_content = Array.from(expectedContentBoxes).map(box => box.value);

        // Get hint URLs
        const hintUrlInputs = form.querySelectorAll('input[name="hint_urls"]');
        data.hint_urls = Array.from(hintUrlInputs)
            .map(input => input.value.trim())
            .filter(url => url !== '');

        return data;
    }

    async makeApiCall(endpoint, method = 'GET', body = null) {
        const url = `${this.apiBaseUrl}${endpoint}`;
        const options = {
            method: method,
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
            throw error;
        }
    }

    generateSourceId(sourceName) {
        return sourceName.toLowerCase()
            .replace(/[^a-z0-9\s]/g, '')
            .replace(/\s+/g, '-')
            .substring(0, 50);
    }

    async loadPendingSources() {
        const container = document.getElementById('pending-sources');
        container.innerHTML = '<div class="alert alert-info">Loading pending sources...</div>';
        
        try {
            const response = await this.makeApiCall('/sources/pending');
            
            if (response.success) {
                this.sources.pending = response.data || [];
                this.displayPendingSources();
            } else {
                throw new Error(response.error || 'Failed to load pending sources');
            }
            
        } catch (error) {
            container.innerHTML = `<div class="alert alert-error">Failed to load pending sources: ${error.message}</div>`;
        }
    }

    displayPendingSources() {
        const container = document.getElementById('pending-sources');
        
        if (this.sources.pending.length === 0) {
            container.innerHTML = '<div class="alert alert-info">No sources pending analysis.</div>';
            return;
        }

        const sourcesHtml = this.sources.pending.map(source => `
            <div class="source-card">
                <div class="source-header">
                    <div class="source-title">${source.source_name}</div>
                    <span class="status-badge status-${source.status.replace('_', '-')}">${this.formatStatus(source.status)}</span>
                </div>
                <div class="source-meta">
                    <div class="meta-item">
                        <div class="meta-label">Type:</div>
                        ${this.formatSourceType(source.source_type)}
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Submitted:</div>
                        ${this.formatDate(source.submitted_at)}
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Base URL:</div>
                        <a href="${source.base_url}" target="_blank">${source.base_url}</a>
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Expected Content:</div>
                        ${source.expected_content.join(', ')}
                    </div>
                </div>
                ${source.hint_urls && source.hint_urls.length > 0 ? `
                    <div style="margin-top: 1rem;">
                        <div class="meta-label">Hint URLs:</div>
                        ${source.hint_urls.map(url => `<div><a href="${url}" target="_blank">${url}</a></div>`).join('')}
                    </div>
                ` : ''}
            </div>
        `).join('');

        container.innerHTML = sourcesHtml;
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
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
    }

    showAlert(type, message) {
        this.clearAlerts();
        
        const alert = document.createElement('div');
        alert.className = `alert alert-${type}`;
        alert.textContent = message;
        
        const form = document.getElementById('source-submission-form');
        form.insertBefore(alert, form.firstChild);
        
        // Auto-remove success alerts after 5 seconds
        if (type === 'success') {
            setTimeout(() => {
                if (alert.parentNode) {
                    alert.remove();
                }
            }, 5000);
        }
    }

    clearAlerts() {
        document.querySelectorAll('.alert').forEach(alert => {
            if (alert.parentNode && alert.parentNode.id === 'source-submission-form') {
                alert.remove();
            }
        });
    }
}

// Utility functions for managing hint URLs
function addHintUrl() {
    const container = document.getElementById('hint-urls-container');
    const urlGroup = document.createElement('div');
    urlGroup.className = 'url-input-group';
    urlGroup.innerHTML = `
        <input type="url" name="hint_urls" placeholder="https://example.com/page">
        <button type="button" class="btn-remove" onclick="removeHintUrl(this)">Remove</button>
    `;
    container.appendChild(urlGroup);
}

function removeHintUrl(button) {
    const container = document.getElementById('hint-urls-container');
    if (container.children.length > 1) {
        button.parentNode.remove();
    }
}

// Initialize the admin interface when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.adminApp = new SourceManagementAdmin();
});