# Product Overview

## Seattle Family Activities Platform

A serverless web application that aggregates and displays family-friendly activities, events, and venues in the Seattle metro area. The platform scrapes content from local websites and presents it through a clean, mobile-responsive interface.

## Core Features

- **Activity Discovery**: Browse events, classes, camps, performances, and free activities
- **Smart Filtering**: Filter by activity type, category, age group, and date
- **Data Management**: Manual data refresh with admin controls (automated feeds planned for future phases)
- **Mobile-First**: Responsive design optimized for families on-the-go
- **Admin Interface**: Source management and content moderation tools

## Target Audience

Families with children in the Seattle metro area looking for local activities, events, and educational opportunities.

## Architecture

- **Frontend**: Static GitHub Pages site with vanilla JavaScript
- **Backend**: AWS Lambda functions (Go) with DynamoDB storage
- **Data Sources**: Local Seattle family websites scraped via FireCrawl API
- **Infrastructure**: AWS CDK for deployment automation