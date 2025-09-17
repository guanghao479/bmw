# Seattle Family Activities MVP Infrastructure

AWS CDK infrastructure for the Seattle Family Activities MVP platform.

## Architecture

- **S3 Bucket**: Public read access for JSON event data
- **Lambda Function**: Go runtime for FireCrawl extraction
- **EventBridge**: Scheduled scraping every 6 hours
- **CloudWatch**: Monitoring and alerting
- **SNS**: Email notifications for failures

## Prerequisites

1. AWS CLI configured with appropriate credentials
2. Node.js 18+ installed
3. AWS CDK CLI installed: `npm install -g aws-cdk`

## AWS Profile Configuration

The CDK will use the AWS profile specified in the `AWS_PROFILE` environment variable.

### Option 1: Using .envrc (recommended with direnv)
If using direnv, the `.envrc` file in the project root already contains:
```bash
export AWS_PROFILE=bmw
export AWS_REGION=us-west-2
```

### Option 2: Manual environment variables
Set these environment variables before deployment:
```bash
export AWS_PROFILE=your-aws-profile-name
export AWS_REGION=us-west-2
export FIRECRAWL_API_KEY="your-firecrawl-api-key"
export ADMIN_EMAIL="your-email@example.com"  # Optional for alerts
```

### Option 3: Default AWS credentials
If no `AWS_PROFILE` is set, CDK will use the default AWS credentials.

## Deployment

1. Install dependencies:
   ```bash
   npm install
   ```

2. Bootstrap CDK (first time only):
   ```bash
   cdk bootstrap aws://YOUR-ACCOUNT-ID/us-west-2
   ```

3. Deploy the stack:
   ```bash
   npm run deploy
   ```

## Useful Commands

- `npm run build` - Compile TypeScript to JavaScript
- `npm run deploy` - Deploy the CDK stack
- `npm run diff` - Compare deployed stack with current state
- `npm run synth` - Emit the synthesized CloudFormation template
- `npm run destroy` - Delete the stack (use with caution)

## Manual Operations

### Trigger scraping manually:
```bash
aws lambda invoke --function-name SeattleFamilyActivities-EventScraper --region us-west-2 output.json
```

### Check scraping logs:
```bash
aws logs tail /aws/lambda/SeattleFamilyActivities-EventScraper --follow --region us-west-2
```

### View current events data:
```bash
curl https://seattle-family-activities-mvp-data-usw2.s3.us-west-2.amazonaws.com/events/events.json
```

## Monitoring

The stack includes CloudWatch alarms for:
- Lambda function errors
- Lambda function timeouts (>12 minutes)

Alerts are sent to the configured email address via SNS.

## Cost Optimization

This infrastructure is designed for minimal cost:
- S3 with lifecycle rules for automatic cleanup
- Lambda with appropriate timeout and memory settings
- EventBridge scheduled events (free tier)
- CloudWatch basic monitoring (free tier)

Expected monthly cost: $5-15 depending on usage.