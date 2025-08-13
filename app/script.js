// Family Events App - Dynamic Content Loading and Interaction
class FamilyEventsApp {
    constructor() {
        this.allData = [];
        this.currentFilter = 'all';
        this.searchTerm = '';
        
        this.init();
    }

    async init() {
        this.showLoading();
        this.loadData();
        this.setupEventListeners();
        this.renderContent();
        this.hideLoading();
    }

    // Load embedded data
    loadData() {
        const data = {
            "events": [
                {
                    "id": 1,
                    "title": "Summer Family Festival",
                    "description": "Join us for a day filled with live music, food trucks, face painting, and family-friendly activities in the heart of downtown.",
                    "category": "event",
                    "image": "https://images.unsplash.com/photo-1533174072545-7a4b6ad7a6c3?w=400&h=300&fit=crop",
                    "date": "2025-08-15",
                    "time": "10:00 AM - 6:00 PM",
                    "location": "Central Park",
                    "price": "Free",
                    "age_range": "All ages",
                    "featured": true
                },
                {
                    "id": 2,
                    "title": "Kids Movie Night Under Stars",
                    "description": "Bring your blankets and enjoy a classic family movie under the stars. Popcorn and refreshments available.",
                    "category": "event",
                    "image": "https://images.unsplash.com/photo-1440404653325-ab127d49abc1?w=400&h=300&fit=crop",
                    "date": "2025-08-20",
                    "time": "7:30 PM - 10:00 PM",
                    "location": "Riverside Park Amphitheater",
                    "price": "$5 per family",
                    "age_range": "3-12 years",
                    "featured": false
                },
                {
                    "id": 3,
                    "title": "Science Discovery Day",
                    "description": "Interactive science experiments, planetarium shows, and hands-on learning experiences for curious minds.",
                    "category": "event",
                    "image": "https://images.unsplash.com/photo-1581833971358-2c8b550f87b3?w=400&h=300&fit=crop",
                    "date": "2025-08-25",
                    "time": "9:00 AM - 4:00 PM",
                    "location": "City Science Museum",
                    "price": "$12 adults, $8 children",
                    "age_range": "5-16 years",
                    "featured": true
                },
                {
                    "id": 4,
                    "title": "Art & Craft Fair",
                    "description": "Local artisans showcase their work with interactive workshops for children and families.",
                    "category": "event",
                    "image": "https://images.unsplash.com/photo-1513475382585-d06e58bcb0e0?w=400&h=300&fit=crop",
                    "date": "2025-09-01",
                    "time": "10:00 AM - 5:00 PM",
                    "location": "Town Square",
                    "price": "Free entry, workshop fees vary",
                    "age_range": "All ages",
                    "featured": false
                }
            ],
            "activities": [
                {
                    "id": 5,
                    "title": "Swimming Lessons",
                    "description": "Professional swimming instruction for beginners to advanced swimmers. Small class sizes and certified instructors.",
                    "category": "activity",
                    "image": "https://images.unsplash.com/photo-1530549387789-4c1017266635?w=400&h=300&fit=crop",
                    "date": "Mondays & Wednesdays",
                    "time": "4:00 PM - 5:00 PM",
                    "location": "Community Pool",
                    "price": "$80/month",
                    "age_range": "4-16 years",
                    "featured": false
                },
                {
                    "id": 6,
                    "title": "Kids Yoga Classes",
                    "description": "Fun and engaging yoga sessions designed specifically for children. Improves flexibility, focus, and mindfulness.",
                    "category": "activity",
                    "image": "https://images.unsplash.com/photo-1544367567-0f2fcb009e0b?w=400&h=300&fit=crop",
                    "date": "Saturdays",
                    "time": "9:00 AM - 10:00 AM",
                    "location": "Harmony Wellness Center",
                    "price": "$15 per session",
                    "age_range": "5-12 years",
                    "featured": true
                },
                {
                    "id": 7,
                    "title": "STEM Robotics Club",
                    "description": "Build and program robots while learning coding, engineering, and problem-solving skills.",
                    "category": "activity",
                    "image": "https://images.unsplash.com/photo-1485827404703-89b55fcc595e?w=400&h=300&fit=crop",
                    "date": "Thursdays",
                    "time": "3:30 PM - 5:30 PM",
                    "location": "Tech Learning Center",
                    "price": "$120/month",
                    "age_range": "8-14 years",
                    "featured": false
                },
                {
                    "id": 8,
                    "title": "Dance Classes for Kids",
                    "description": "Learn various dance styles including hip-hop, ballet, and contemporary. Great for building confidence and coordination.",
                    "category": "activity",
                    "image": "https://images.unsplash.com/photo-1547036967-23d11aacaee0?w=400&h=300&fit=crop",
                    "date": "Tuesdays & Fridays",
                    "time": "5:00 PM - 6:00 PM",
                    "location": "Movement Studio",
                    "price": "$90/month",
                    "age_range": "6-16 years",
                    "featured": false
                }
            ],
            "venues": [
                {
                    "id": 9,
                    "title": "Adventure Playground",
                    "description": "Large outdoor playground with climbing structures, slides, swings, and a splash pad. Perfect for active families.",
                    "category": "venue",
                    "image": "https://images.unsplash.com/photo-1578662996442-48f60103fc96?w=400&h=300&fit=crop",
                    "date": "Daily",
                    "time": "6:00 AM - 8:00 PM",
                    "location": "Maple Avenue Park",
                    "price": "Free",
                    "age_range": "2-12 years",
                    "featured": true
                },
                {
                    "id": 10,
                    "title": "Children's Library",
                    "description": "Extensive collection of children's books, storytelling sessions, and quiet reading areas. Free wifi and computers available.",
                    "category": "venue",
                    "image": "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=400&h=300&fit=crop",
                    "date": "Monday - Saturday",
                    "time": "9:00 AM - 8:00 PM",
                    "location": "Downtown Library Branch",
                    "price": "Free",
                    "age_range": "All ages",
                    "featured": false
                },
                {
                    "id": 11,
                    "title": "Indoor Trampoline Park",
                    "description": "Safe, supervised indoor trampoline facilities with foam pits, dodgeball courts, and special toddler areas.",
                    "category": "venue",
                    "image": "https://images.unsplash.com/photo-1571019613454-1cb2f99b2d8b?w=400&h=300&fit=crop",
                    "date": "Daily",
                    "time": "9:00 AM - 9:00 PM",
                    "location": "Bounce Zone",
                    "price": "$18 per hour",
                    "age_range": "3+ years",
                    "featured": false
                },
                {
                    "id": 12,
                    "title": "Family Bowling Center",
                    "description": "Modern bowling facility with bumper lanes for kids, arcade games, and party packages. Snack bar on-site.",
                    "category": "venue",
                    "image": "https://images.unsplash.com/photo-1578662996442-48f60103fc96?w=400&h=300&fit=crop",
                    "date": "Daily",
                    "time": "10:00 AM - 11:00 PM",
                    "location": "Strike Zone Bowling",
                    "price": "$12 per game",
                    "age_range": "All ages",
                    "featured": false
                },
                {
                    "id": 13,
                    "title": "Nature Discovery Center",
                    "description": "Interactive exhibits about local wildlife, nature trails, and educational programs. Perfect for outdoor learning.",
                    "category": "venue",
                    "image": "https://images.unsplash.com/photo-1441974231531-c6227db76b6e?w=400&h=300&fit=crop",
                    "date": "Tuesday - Sunday",
                    "time": "9:00 AM - 5:00 PM",
                    "location": "Greenwood Nature Preserve",
                    "price": "$8 adults, $5 children",
                    "age_range": "All ages",
                    "featured": true
                }
            ]
        };
        
        // Combine all data types into a single array
        this.allData = [
            ...data.events,
            ...data.activities,
            ...data.venues
        ];
    }

    // Setup event listeners for interactivity
    setupEventListeners() {
        // Search input
        const searchInput = document.getElementById('searchInput');
        searchInput.addEventListener('input', (e) => {
            this.searchTerm = e.target.value.toLowerCase();
            this.renderContent();
        });

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