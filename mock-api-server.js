#!/usr/bin/env node

const http = require('http');
const url = require('url');

const PORT = 3000;

// Mock data
const mockAnalytics = {
  total_sources_submitted: 12,
  sources_active: 0,
  success_rate: '0%',
  total_activities: 0
};

const mockActiveSources = [];

const mockSchemas = {
  events: {
    name: "Events Schema",
    description: "Extract events with title, date, location, and price",
    schema: {
      type: "object",
      properties: {
        items: {
          type: "array",
          items: {
            type: "object",
            properties: {
              title: { type: "string" },
              date: { type: "string" },
              location: { type: "string" },
              price: { type: "string" }
            }
          }
        }
      }
    }
  },
  activities: {
    name: "Activities Schema", 
    description: "Extract activities with name, age groups, and duration",
    schema: {
      type: "object",
      properties: {
        items: {
          type: "array",
          items: {
            type: "object",
            properties: {
              name: { type: "string" },
              age_groups: { type: "string" },
              duration: { type: "string" }
            }
          }
        }
      }
    }
  }
};

const mockPendingEvents = [];

function handleRequest(req, res) {
  // Set CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token');
  res.setHeader('Access-Control-Allow-Methods', 'GET,POST,PUT,DELETE,OPTIONS');
  res.setHeader('Content-Type', 'application/json');

  // Handle preflight OPTIONS request
  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }

  const parsedUrl = url.parse(req.url, true);
  const path = parsedUrl.pathname;
  const method = req.method;

  console.log(`${method} ${path}`);

  let response = { success: false, error: 'Not found' };
  let statusCode = 404;

  try {
    switch (true) {
      case method === 'GET' && path === '/api/analytics':
        response = {
          success: true,
          message: 'Analytics retrieved successfully',
          data: mockAnalytics
        };
        statusCode = 200;
        break;

      case method === 'GET' && path === '/api/sources/active':
        response = {
          success: true,
          message: 'Active sources retrieved successfully',
          data: mockActiveSources
        };
        statusCode = 200;
        break;

      case method === 'GET' && path === '/api/schemas':
        response = {
          success: true,
          message: 'Schemas retrieved successfully',
          data: mockSchemas
        };
        statusCode = 200;
        break;

      case method === 'GET' && path === '/api/events/pending':
        response = {
          success: true,
          message: 'Pending events retrieved successfully',
          data: mockPendingEvents
        };
        statusCode = 200;
        break;

      case method === 'POST' && path === '/api/crawl/submit':
        // Mock crawl submission
        response = {
          success: true,
          message: 'Crawl submitted successfully',
          data: {
            events_count: 5,
            processing_time: '2.3s',
            credits_used: 1
          }
        };
        statusCode = 200;
        break;

      default:
        response = {
          success: false,
          error: `Endpoint not implemented: ${method} ${path}`
        };
        statusCode = 404;
    }
  } catch (error) {
    console.error('Error handling request:', error);
    response = {
      success: false,
      error: 'Internal server error'
    };
    statusCode = 500;
  }

  res.writeHead(statusCode);
  res.end(JSON.stringify(response));
}

const server = http.createServer(handleRequest);

server.listen(PORT, () => {
  console.log(`Mock API server running on http://localhost:${PORT}`);
  console.log('Available endpoints:');
  console.log('  GET  /api/analytics');
  console.log('  GET  /api/sources/active');
  console.log('  GET  /api/schemas');
  console.log('  GET  /api/events/pending');
  console.log('  POST /api/crawl/submit');
});