#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { SeattleFamilyActivitiesMVPStack } from '../lib/mvp-stack';

const app = new cdk.App();

// Use AWS profile from environment variable
const awsProfile = process.env.AWS_PROFILE || 'default';
const awsRegion = process.env.AWS_REGION || 'us-west-2';

// Get account ID from current AWS credentials/profile
const account = process.env.CDK_DEFAULT_ACCOUNT || app.node.tryGetContext('account');

new SeattleFamilyActivitiesMVPStack(app, 'SeattleFamilyActivitiesMVPStack', {
  env: {
    account: account,
    region: awsRegion
  },
  
  // Stack description
  description: 'Seattle Family Activities MVP - Ultra-minimal serverless architecture with S3 + Lambda + EventBridge',
  
  // Tags for resource management
  tags: {
    Project: 'SeattleFamilyActivities',
    Environment: 'MVP',
    Architecture: 'Serverless',
    Region: 'us-west-2'
  }
});