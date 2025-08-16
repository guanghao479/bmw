import { Stack, StackProps, Duration, CfnOutput, RemovalPolicy } from 'aws-cdk-lib';
import { Construct } from 'constructs';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import * as sns from 'aws-cdk-lib/aws-sns';
import * as snsSubscriptions from 'aws-cdk-lib/aws-sns-subscriptions';
import * as cloudwatchActions from 'aws-cdk-lib/aws-cloudwatch-actions';
import { GoFunction } from '@aws-cdk/aws-lambda-go-alpha';

export class SeattleFamilyActivitiesMVPStack extends Stack {
  constructor(scope: Construct, id: string, props?: StackProps) {
    super(scope, id, props);

    // S3 bucket for events data with public read access
    const eventsBucket = new s3.Bucket(this, 'EventsDataBucket', {
      bucketName: 'seattle-family-activities-mvp-data-usw2',
      publicReadAccess: true,
      blockPublicAccess: new s3.BlockPublicAccess({
        blockPublicAcls: false,
        blockPublicPolicy: false,
        ignorePublicAcls: false,
        restrictPublicBuckets: false
      }),
      cors: [
        {
          allowedOrigins: ['https://guanghao479.github.io', 'http://localhost:*'],
          allowedMethods: [s3.HttpMethods.GET, s3.HttpMethods.HEAD],
          allowedHeaders: ['*'],
          exposedHeaders: ['ETag'],
          maxAge: 3600
        }
      ],
      lifecycleRules: [
        {
          id: 'DeleteOldSnapshots',
          enabled: true,
          prefix: 'events/snapshots/',
          expiration: Duration.days(30) // Keep daily snapshots for 30 days
        }
      ],
      removalPolicy: RemovalPolicy.DESTROY // For MVP - allows easy cleanup
    });

    // IAM role for Lambda function
    const scraperRole = new iam.Role(this, 'ScraperLambdaRole', {
      assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com'),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName('service-role/AWSLambdaBasicExecutionRole')
      ],
      inlinePolicies: {
        S3Access: new iam.PolicyDocument({
          statements: [
            new iam.PolicyStatement({
              effect: iam.Effect.ALLOW,
              actions: [
                's3:PutObject',
                's3:PutObjectAcl',
                's3:GetObject',
                's3:ListBucket'
              ],
              resources: [
                eventsBucket.bucketArn,
                `${eventsBucket.bucketArn}/*`
              ]
            })
          ]
        })
      }
    });

    // OIDC Identity Provider for GitHub Actions
    const githubOidcProvider = new iam.OpenIdConnectProvider(this, 'GitHubOidcProvider', {
      url: 'https://token.actions.githubusercontent.com',
      clientIds: ['sts.amazonaws.com'],
      thumbprints: ['6938fd4d98bab03faadb97b34396831e3780aea1']
    });

    // IAM Role for GitHub Actions with OIDC trust policy
    const githubActionsRole = new iam.Role(this, 'GitHubActionsDeploymentRole', {
      roleName: 'GitHubActions-SeattleFamilyActivities',
      assumedBy: new iam.WebIdentityPrincipal(
        githubOidcProvider.openIdConnectProviderArn,
        {
          'StringLike': {
            'token.actions.githubusercontent.com:sub': 'repo:guanghao479/bmw:*'
          },
          'StringEquals': {
            'token.actions.githubusercontent.com:aud': 'sts.amazonaws.com'
          }
        }
      ),
      description: 'IAM role for GitHub Actions to deploy Seattle Family Activities infrastructure',
      maxSessionDuration: Duration.hours(1),
      managedPolicies: [
        // Core AWS service policies required for CDK deployment
        iam.ManagedPolicy.fromAwsManagedPolicyName('AWSLambda_FullAccess'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonS3FullAccess'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('CloudWatchFullAccess'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonEventBridgeFullAccess'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonSNSFullAccess'),
        // CDK deployment policies
        iam.ManagedPolicy.fromAwsManagedPolicyName('IAMFullAccess'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AWSCloudFormationFullAccess')
      ]
    });

    // Lambda function for scraping (Go runtime)
    const scraperFunction = new GoFunction(this, 'EventScraperFunction', {
      entry: '../backend/cmd/lambda',
      functionName: 'seattle-family-activities-scraper',
      timeout: Duration.minutes(15),
      memorySize: 1024,
      role: scraperRole,
      environment: {
        S3_BUCKET: eventsBucket.bucketName,
        OPENAI_API_KEY: process.env.OPENAI_API_KEY || '',
        // AWS_REGION is automatically set by Lambda runtime
        LOG_LEVEL: 'INFO'
      },
      description: 'Scrapes Seattle family activity websites using Jina + OpenAI and stores results in S3'
    });

    // EventBridge rule for scheduled scraping (every 6 hours)
    const scrapingRule = new events.Rule(this, 'ScrapingScheduleRule', {
      ruleName: 'SeattleFamilyActivities-ScrapingSchedule',
      description: 'Triggers family activities scraping every 6 hours',
      schedule: events.Schedule.rate(Duration.hours(6)),
      targets: [
        new targets.LambdaFunction(scraperFunction, {
          retryAttempts: 2
        })
      ]
    });

    // SNS topic for alerts
    const alertTopic = new sns.Topic(this, 'ScrapingAlertsTopic', {
      topicName: 'SeattleFamilyActivities-Alerts',
      displayName: 'Seattle Family Activities Alerts'
    });

    // Email subscription for alerts (if email provided)
    if (process.env.ADMIN_EMAIL) {
      alertTopic.addSubscription(
        new snsSubscriptions.EmailSubscription(process.env.ADMIN_EMAIL)
      );
    }

    // CloudWatch alarm for Lambda failures
    const lambdaErrorAlarm = new cloudwatch.Alarm(this, 'ScraperLambdaErrorAlarm', {
      alarmName: 'SeattleFamilyActivities-ScraperErrors',
      alarmDescription: 'Alert when event scraper Lambda function fails',
      metric: scraperFunction.metricErrors({
        period: Duration.minutes(5),
        statistic: 'Sum'
      }),
      threshold: 1,
      evaluationPeriods: 1,
      treatMissingData: cloudwatch.TreatMissingData.NOT_BREACHING
    });

    // Add SNS action to the alarm
    lambdaErrorAlarm.addAlarmAction(
      new cloudwatchActions.SnsAction(alertTopic)
    );

    // CloudWatch alarm for Lambda duration (timeout warning)
    const lambdaDurationAlarm = new cloudwatch.Alarm(this, 'ScraperLambdaDurationAlarm', {
      alarmName: 'SeattleFamilyActivities-ScraperDuration',
      alarmDescription: 'Alert when event scraper takes longer than 12 minutes',
      metric: scraperFunction.metricDuration({
        period: Duration.minutes(5),
        statistic: 'Maximum'
      }),
      threshold: Duration.minutes(12).toMilliseconds(),
      evaluationPeriods: 1,
      treatMissingData: cloudwatch.TreatMissingData.NOT_BREACHING
    });

    lambdaDurationAlarm.addAlarmAction(
      new cloudwatchActions.SnsAction(alertTopic)
    );

    // Outputs for reference
    new CfnOutput(this, 'S3BucketName', {
      value: eventsBucket.bucketName,
      description: 'S3 bucket name for events data',
      exportName: 'SeattleFamilyActivities-S3BucketName'
    });

    new CfnOutput(this, 'S3BucketURL', {
      value: `https://${eventsBucket.bucketName}.s3.us-west-2.amazonaws.com`,
      description: 'S3 bucket URL for frontend configuration',
      exportName: 'SeattleFamilyActivities-S3BucketURL'
    });

    new CfnOutput(this, 'LambdaFunctionName', {
      value: scraperFunction.functionName,
      description: 'Lambda function name for manual invocation',
      exportName: 'SeattleFamilyActivities-LambdaFunctionName'
    });

    new CfnOutput(this, 'EventsDataURL', {
      value: `https://${eventsBucket.bucketName}.s3.us-west-2.amazonaws.com/events/latest.json`,
      description: 'Direct URL to latest events JSON file',
      exportName: 'SeattleFamilyActivities-EventsDataURL'
    });

    new CfnOutput(this, 'GitHubActionsRoleArn', {
      value: githubActionsRole.roleArn,
      description: 'IAM Role ARN for GitHub Actions OIDC authentication',
      exportName: 'SeattleFamilyActivities-GitHubActionsRoleArn'
    });
  }
}