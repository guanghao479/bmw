# GitHub Pages Deployment Setup

This document provides step-by-step instructions for setting up GitHub Pages deployment for the Seattle Family Activities MVP.

## Prerequisites

- GitHub repository: `guanghao479/bmw`
- GitHub Actions workflow already created (`.github/workflows/deploy.yml`)
- AWS credentials configured in repository secrets

## Setup Instructions

### Step 1: Configure GitHub Pages in Repository Settings

1. **Navigate to Repository Settings**:
   ```
   https://github.com/guanghao479/bmw/settings
   ```

2. **Go to Pages Section**:
   - Click on "Pages" in the left sidebar under "Code and automation"

3. **Configure Source**:
   - Under "Source", select **"GitHub Actions"**
   - ⚠️ **Important**: Do NOT select "Deploy from a branch" - this will conflict with our workflow

4. **Verify Configuration**:
   - You should see: "Your site is ready to be published at `https://guanghao479.github.io/bmw/`"
   - The status will show "Ready" or "Building" after the first workflow run

### Step 2: Configure Repository Permissions

1. **Actions Permissions**:
   - Go to Settings → Actions → General
   - Under "Workflow permissions":
     - Select ✅ **"Read and write permissions"**
     - Check ✅ **"Allow GitHub Actions to create and approve pull requests"**

2. **Environment Protection** (Optional but Recommended):
   - Go to Settings → Environments
   - If `github-pages` environment exists, click on it
   - Add protection rules:
     - ✅ Required reviewers (add yourself)
     - ✅ Wait timer: 0 minutes
     - ✅ Restrict pushes to protected branches

### Step 3: Verify AWS Secrets Configuration

The deployment workflow requires these secrets to be configured in repository settings:

1. **Go to Repository Secrets**:
   ```
   Settings → Secrets and variables → Actions
   ```

2. **Required Secrets**:
   - `AWS_ACCESS_KEY_ID`
   - `AWS_SECRET_ACCESS_KEY`
   - `AWS_ACCOUNT_ID`
   - `OPENAI_API_KEY` (for Lambda function)
   - `JINA_API_KEY` (for Lambda function)

3. **Verify Secrets Exist**:
   - All secrets should be listed under "Repository secrets"
   - Values are hidden but names should be visible

### Step 4: Trigger First Deployment

1. **Manual Trigger** (Recommended for first deployment):
   - Go to Actions tab in repository
   - Click on "Deploy Seattle Family Activities MVP" workflow
   - Click "Run workflow" → Select "main" branch → Click "Run workflow"

2. **Automatic Trigger**:
   - Push any change to the `main` branch
   - The workflow will automatically trigger

### Step 5: Monitor Deployment

1. **Check Workflow Status**:
   - Go to Actions tab: `https://github.com/guanghao479/bmw/actions`
   - Look for "Deploy Seattle Family Activities MVP" workflow runs
   - Click on a run to see detailed logs

2. **Verify Deployment Steps**:
   - ✅ `deploy-backend`: CDK deployment and Lambda function update
   - ✅ `deploy-frontend`: GitHub Pages artifact upload and deployment
   - ✅ `test-e2e`: End-to-end testing of S3 and frontend

3. **Check GitHub Pages Status**:
   - Go back to Settings → Pages
   - You should see: "Your site is live at `https://guanghao479.github.io/bmw/`"
   - Click the link to verify the site loads

## Deployment Workflow Details

### Frontend Deployment Process

The GitHub Actions workflow automatically:

1. **Builds Frontend**:
   - Validates JavaScript syntax (`node -c app/script.js`)
   - Validates HTML structure
   - Tests frontend functionality

2. **Creates Pages Artifact**:
   - Packages the `/app/` directory contents
   - Uploads as GitHub Pages artifact

3. **Deploys to GitHub Pages**:
   - Uses official `actions/deploy-pages@v4` action
   - Deploys artifact to GitHub Pages hosting

4. **Tests Deployment**:
   - Verifies site accessibility
   - Runs Lighthouse performance audit
   - Tests S3 data loading

### Workflow Triggers

The deployment runs on:

- ✅ **Push to main branch**: Automatic full deployment
- ✅ **Manual workflow dispatch**: Can select backend/frontend deployment
- ✅ **Pull requests**: Runs tests but doesn't deploy

### Environment-Specific Configuration

The frontend automatically detects environment:

- **Development** (`localhost`): 
  - 5-minute data refresh
  - Debug logging enabled
  - Development-specific configuration

- **Production** (`guanghao479.github.io`):
  - 30-minute data refresh  
  - Production optimizations
  - Error tracking enabled

## Troubleshooting

### Common Issues

1. **"Source" shows "Deploy from a branch"**:
   - ❌ Problem: Wrong source type selected
   - ✅ Solution: Change to "GitHub Actions" in Settings → Pages

2. **Workflow fails with permissions error**:
   - ❌ Problem: Insufficient permissions
   - ✅ Solution: Enable "Read and write permissions" in Settings → Actions

3. **Site shows 404 or old content**:
   - ❌ Problem: Deployment not complete or cached
   - ✅ Solution: Wait 5-10 minutes, clear browser cache, check workflow logs

4. **Frontend shows "Sample data loaded"**:
   - ❌ Problem: S3 endpoint not accessible or Lambda not running
   - ✅ Solution: Check AWS deployment, verify S3 CORS configuration

5. **Workflow fails with AWS errors**:
   - ❌ Problem: AWS credentials expired or incorrect
   - ✅ Solution: Update repository secrets with fresh AWS credentials

### Verification Steps

1. **Check Frontend URL**: https://guanghao479.github.io/bmw/
2. **Check Data API**: https://seattle-family-activities-mvp-data-usw2.s3.us-west-2.amazonaws.com/activities/latest.json
3. **Check Workflow Logs**: https://github.com/guanghao479/bmw/actions
4. **Check Pages Status**: https://github.com/guanghao479/bmw/settings/pages

## Manual Deployment (Alternative)

If GitHub Actions fails, you can manually deploy:

1. **Local Setup**:
   ```bash
   git clone https://github.com/guanghao479/bmw.git
   cd bmw
   ```

2. **Deploy to GitHub Pages**:
   ```bash
   # Enable GitHub Pages with branch source
   git checkout -b gh-pages
   cp -r app/* .
   git add .
   git commit -m "Manual GitHub Pages deployment"
   git push origin gh-pages
   ```

3. **Configure Source**:
   - Go to Settings → Pages
   - Select "Deploy from a branch"
   - Choose `gh-pages` branch, `/ (root)` folder

## Next Steps

After successful deployment:

1. **Test the Live Site**: Visit https://guanghao479.github.io/bmw/
2. **Verify Real Data Loading**: Check that Seattle activities load (not sample data)
3. **Test Mobile Responsiveness**: Check site on mobile devices
4. **Monitor Performance**: Check browser developer tools for load times
5. **Set Up Monitoring**: Configure uptime monitoring if desired

## Support

If you encounter issues:

1. Check the troubleshooting section above
2. Review workflow logs in the Actions tab
3. Verify all prerequisites are met
4. Check AWS and GitHub service status pages