import { Stack, StackProps, Duration, CfnOutput, RemovalPolicy } from 'aws-cdk-lib';
import { Construct } from 'constructs';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
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

    // DynamoDB Table 1: Family Activities (Business Data)
    const familyActivitiesTable = new dynamodb.Table(this, 'FamilyActivitiesTable', {
      tableName: 'seattle-family-activities',
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: RemovalPolicy.DESTROY, // For MVP - allows easy cleanup
      pointInTimeRecoverySpecification: {
        pointInTimeRecoveryEnabled: true
      },
      encryption: dynamodb.TableEncryption.AWS_MANAGED
    });

    // Add Global Secondary Indexes to Family Activities Table
    familyActivitiesTable.addGlobalSecondaryIndex({
      indexName: 'location-date-index',
      partitionKey: { name: 'LocationKey', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'DateTypeKey', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.INCLUDE,
      nonKeyAttributes: ['venue_name', 'event_name', 'program_name', 'attraction_name', 'category', 'age_groups', 'pricing', 'status']
    });

    familyActivitiesTable.addGlobalSecondaryIndex({
      indexName: 'category-age-index',
      partitionKey: { name: 'CategoryAgeKey', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'DateFeaturedKey', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.INCLUDE,
      nonKeyAttributes: ['schedule', 'pricing', 'location', 'description']
    });

    familyActivitiesTable.addGlobalSecondaryIndex({
      indexName: 'venue-activity-index',
      partitionKey: { name: 'VenueKey', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'TypeDateKey', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.INCLUDE,
      nonKeyAttributes: ['venue_name', 'event_name', 'program_name', 'schedule', 'status']
    });

    familyActivitiesTable.addGlobalSecondaryIndex({
      indexName: 'provider-index',
      partitionKey: { name: 'ProviderKey', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'TypeStatusKey', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.INCLUDE,
      nonKeyAttributes: ['venue_name', 'event_name', 'program_name', 'status', 'updated_at']
    });

    // DynamoDB Table 2: Source Management (Source Configuration)
    const sourceManagementTable = new dynamodb.Table(this, 'SourceManagementTable', {
      tableName: 'seattle-source-management',
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: RemovalPolicy.DESTROY, // For MVP - allows easy cleanup
      pointInTimeRecoverySpecification: {
        pointInTimeRecoveryEnabled: true
      },
      encryption: dynamodb.TableEncryption.AWS_MANAGED
    });

    // Add Global Secondary Index to Source Management Table
    sourceManagementTable.addGlobalSecondaryIndex({
      indexName: 'status-priority-index',
      partitionKey: { name: 'StatusKey', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'PriorityKey', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.INCLUDE,
      nonKeyAttributes: ['source_name', 'base_url', 'scraping_config', 'data_quality', 'status']
    });

    // DynamoDB Table 3: Scraping Operations (Dynamic Scraping State)
    const scrapingOperationsTable = new dynamodb.Table(this, 'ScrapingOperationsTable', {
      tableName: 'seattle-scraping-operations',
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: RemovalPolicy.DESTROY, // For MVP - allows easy cleanup
      timeToLiveAttribute: 'TTL', // Auto-expire old scraping data after 90 days
      pointInTimeRecoverySpecification: {
        pointInTimeRecoveryEnabled: false // Not needed for operational data
      },
      encryption: dynamodb.TableEncryption.AWS_MANAGED
    });

    // DynamoDB Table 4: Admin Events (Admin Crawling Flow)
    const adminEventsTable = new dynamodb.Table(this, 'AdminEventsTable', {
      tableName: 'seattle-admin-events',
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: RemovalPolicy.DESTROY, // For MVP - allows easy cleanup
      pointInTimeRecoverySpecification: {
        pointInTimeRecoveryEnabled: true // Important for admin data
      },
      encryption: dynamodb.TableEncryption.AWS_MANAGED
    });

    // Add Global Secondary Index to Scraping Operations Table
    scrapingOperationsTable.addGlobalSecondaryIndex({
      indexName: 'next-run-index',
      partitionKey: { name: 'NextRunKey', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'PrioritySourceKey', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.INCLUDE,
      nonKeyAttributes: ['source_id', 'scheduled_time', 'task_type', 'status', 'retry_count']
    });

    // Add Global Secondary Index to Admin Events Table
    adminEventsTable.addGlobalSecondaryIndex({
      indexName: 'status-date-index',
      partitionKey: { name: 'StatusKey', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.ALL
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
        }),
        DynamoDBAccess: new iam.PolicyDocument({
          statements: [
            new iam.PolicyStatement({
              effect: iam.Effect.ALLOW,
              actions: [
                'dynamodb:PutItem',
                'dynamodb:GetItem',
                'dynamodb:UpdateItem',
                'dynamodb:DeleteItem',
                'dynamodb:Query',
                'dynamodb:Scan',
                'dynamodb:BatchGetItem',
                'dynamodb:BatchWriteItem'
              ],
              resources: [
                familyActivitiesTable.tableArn,
                sourceManagementTable.tableArn,
                scrapingOperationsTable.tableArn,
                adminEventsTable.tableArn,
                `${familyActivitiesTable.tableArn}/index/*`,
                `${sourceManagementTable.tableArn}/index/*`,
                `${scrapingOperationsTable.tableArn}/index/*`,
                `${adminEventsTable.tableArn}/index/*`
              ]
            })
          ]
        }),
        SQSAccess: new iam.PolicyDocument({
          statements: [
            new iam.PolicyStatement({
              effect: iam.Effect.ALLOW,
              actions: [
                'sqs:SendMessage',
                'sqs:ReceiveMessage',
                'sqs:DeleteMessage',
                'sqs:GetQueueAttributes',
                'sqs:GetQueueUrl'
              ],
              resources: [
                scrapingTaskQueue.queueArn,
                scrapingTaskDLQ.queueArn
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
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonDynamoDBFullAccess'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('CloudWatchFullAccess'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonEventBridgeFullAccess'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonSNSFullAccess'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonSSMFullAccess'),
        // CDK deployment policies
        iam.ManagedPolicy.fromAwsManagedPolicyName('IAMFullAccess'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AWSCloudFormationFullAccess')
      ]
    });

    // Lambda function for scraping orchestration (Go runtime)
    const scrapingOrchestratorFunction = new GoFunction(this, 'ScrapingOrchestratorFunction', {
      entry: '../backend/cmd/scraping_orchestrator',
      functionName: 'seattle-family-activities-scraping-orchestrator',
      timeout: Duration.minutes(15),
      memorySize: 1024,
      role: scraperRole, // Reuse the same role since it has DynamoDB and S3 access
      environment: {
        S3_BUCKET: eventsBucket.bucketName,
        FAMILY_ACTIVITIES_TABLE: familyActivitiesTable.tableName,
        SOURCE_MANAGEMENT_TABLE: sourceManagementTable.tableName,
        SCRAPING_OPERATIONS_TABLE: scrapingOperationsTable.tableName,
        SCRAPING_TASK_QUEUE_URL: scrapingTaskQueue.queueUrl,
        FIRECRAWL_API_KEY: process.env.FIRECRAWL_API_KEY || '',
        LOG_LEVEL: 'INFO'
      },
      description: 'Orchestrates scraping tasks from DynamoDB sources using FireCrawl, replacing hardcoded source list'
    });


    // EventBridge rule for DynamoDB-driven orchestrator (every 15 minutes)
    const orchestratorRule = new events.Rule(this, 'OrchestratorScheduleRule', {
      ruleName: 'SeattleFamilyActivities-OrchestratorSchedule',
      description: 'Triggers scraping orchestrator to check for scheduled tasks',
      schedule: events.Schedule.rate(Duration.minutes(15)),
      targets: [
        new targets.LambdaFunction(scrapingOrchestratorFunction, {
          retryAttempts: 1,
          event: events.RuleTargetInput.fromObject({
            trigger_type: 'scheduled'
          })
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

    // Create a separate IAM role for Admin API Lambda  
    const adminApiRole = new iam.Role(this, 'AdminApiLambdaRole', {
      assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com'),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName('service-role/AWSLambdaBasicExecutionRole')
      ],
      inlinePolicies: {
        DynamoDBAccess: new iam.PolicyDocument({
          statements: [
            new iam.PolicyStatement({
              effect: iam.Effect.ALLOW,
              actions: [
                'dynamodb:PutItem',
                'dynamodb:GetItem',
                'dynamodb:UpdateItem',
                'dynamodb:DeleteItem',
                'dynamodb:Query',
                'dynamodb:Scan',
                'dynamodb:BatchGetItem',
                'dynamodb:BatchWriteItem'
              ],
              resources: [
                familyActivitiesTable.tableArn,
                sourceManagementTable.tableArn,
                scrapingOperationsTable.tableArn,
                adminEventsTable.tableArn,
                `${familyActivitiesTable.tableArn}/index/*`,
                `${sourceManagementTable.tableArn}/index/*`,
                `${scrapingOperationsTable.tableArn}/index/*`,
                `${adminEventsTable.tableArn}/index/*`
              ]
            })
          ]
        }),
        LambdaInvokeAccess: new iam.PolicyDocument({
          statements: [
            new iam.PolicyStatement({
              effect: iam.Effect.ALLOW,
              actions: [
                'lambda:InvokeFunction'
              ],
              resources: [
                scrapingOrchestratorFunction.functionArn
              ]
            })
          ]
        })
      }
    });

    // Admin API Lambda function for UI backend
    const adminApiFunction = new GoFunction(this, 'AdminApiFunction', {
      entry: '../backend/cmd/admin_api',
      functionName: 'seattle-family-activities-admin-api',
      timeout: Duration.minutes(5),
      memorySize: 512,
      role: adminApiRole,
      environment: {
        FAMILY_ACTIVITIES_TABLE: familyActivitiesTable.tableName,
        SOURCE_MANAGEMENT_TABLE: sourceManagementTable.tableName,
        SCRAPING_OPERATIONS_TABLE: scrapingOperationsTable.tableName,
        ADMIN_EVENTS_TABLE: adminEventsTable.tableName,
        ORCHESTRATOR_FUNCTION_NAME: scrapingOrchestratorFunction.functionName,
        FIRECRAWL_API_KEY: process.env.FIRECRAWL_API_KEY || '',
      }
    });

    // API Gateway for Admin UI
    const adminApi = new apigateway.RestApi(this, 'AdminApi', {
      restApiName: 'SeattleFamilyActivities-AdminAPI',
      description: 'Admin API for Seattle Family Activities source management',
      defaultCorsPreflightOptions: {
        allowOrigins: ['https://guanghao479.github.io', 'http://localhost:8000'],
        allowMethods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
        allowHeaders: ['Content-Type', 'X-Amz-Date', 'Authorization', 'X-Api-Key', 'X-Amz-Security-Token'],
      },
      deployOptions: {
        stageName: 'prod'
      }
    });

    // API Gateway Lambda integration
    const adminApiIntegration = new apigateway.LambdaIntegration(adminApiFunction, {
      requestTemplates: { 'application/json': '{ "statusCode": "200" }' },
      proxy: true
    });

    // API routes
    const apiResource = adminApi.root.addResource('api');
    const sourcesResource = apiResource.addResource('sources');
    
    // Sources routes
    sourcesResource.addMethod('POST', adminApiIntegration); // POST /api/sources (with {action: 'submit'} in body)
    sourcesResource.addMethod('GET', adminApiIntegration);  // GET /api/sources?type=pending|active
    
    const sourceResource = sourcesResource.addResource('{id}');
    const analysisResource = sourceResource.addResource('analysis');
    const activateResource = sourceResource.addResource('activate');
    const rejectResource = sourceResource.addResource('reject');
    const detailsResource = sourceResource.addResource('details');
    const triggerResource = sourceResource.addResource('trigger');
    
    analysisResource.addMethod('GET', adminApiIntegration); // GET /api/sources/{id}/analysis
    activateResource.addMethod('PUT', adminApiIntegration); // PUT /api/sources/{id}/activate
    rejectResource.addMethod('PUT', adminApiIntegration);   // PUT /api/sources/{id}/reject
    detailsResource.addMethod('GET', adminApiIntegration);  // GET /api/sources/{id}/details
    triggerResource.addMethod('POST', adminApiIntegration); // POST /api/sources/{id}/trigger
    
    // Analytics route
    const analyticsResource = apiResource.addResource('analytics');
    analyticsResource.addMethod('GET', adminApiIntegration); // GET /api/analytics

    // Submit route for backwards compatibility
    const submitResource = sourcesResource.addResource('submit');
    submitResource.addMethod('POST', adminApiIntegration); // POST /api/sources/submit
    
    // Pending/active routes for backwards compatibility
    const pendingResource = sourcesResource.addResource('pending');
    const activeResource = sourcesResource.addResource('active');
    pendingResource.addMethod('GET', adminApiIntegration); // GET /api/sources/pending
    activeResource.addMethod('GET', adminApiIntegration);  // GET /api/sources/active

    // Admin Crawling Routes
    const crawlResource = apiResource.addResource('crawl');
    const crawlSubmitResource = crawlResource.addResource('submit');
    crawlSubmitResource.addMethod('POST', adminApiIntegration); // POST /api/crawl/submit

    const eventsResource = apiResource.addResource('events');
    const eventsPendingResource = eventsResource.addResource('pending');
    eventsPendingResource.addMethod('GET', adminApiIntegration); // GET /api/events/pending

    const eventResource = eventsResource.addResource('{id}');
    eventResource.addMethod('GET', adminApiIntegration); // GET /api/events/{id}

    const approveResource = eventResource.addResource('approve');
    const rejectEventResource = eventResource.addResource('reject');
    const editResource = eventResource.addResource('edit');

    approveResource.addMethod('PUT', adminApiIntegration); // PUT /api/events/{id}/approve
    rejectEventResource.addMethod('PUT', adminApiIntegration); // PUT /api/events/{id}/reject
    editResource.addMethod('PUT', adminApiIntegration); // PUT /api/events/{id}/edit

    const schemasResource = apiResource.addResource('schemas');
    schemasResource.addMethod('GET', adminApiIntegration); // GET /api/schemas

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



    new CfnOutput(this, 'ScrapingOrchestratorFunctionName', {
      value: scrapingOrchestratorFunction.functionName,
      description: 'Scraping orchestrator Lambda function name for manual invocation',
      exportName: 'SeattleFamilyActivities-ScrapingOrchestratorFunctionName'
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

    new CfnOutput(this, 'FamilyActivitiesTableName', {
      value: familyActivitiesTable.tableName,
      description: 'DynamoDB table name for family activities data',
      exportName: 'SeattleFamilyActivities-FamilyActivitiesTableName'
    });

    new CfnOutput(this, 'SourceManagementTableName', {
      value: sourceManagementTable.tableName,
      description: 'DynamoDB table name for source management',
      exportName: 'SeattleFamilyActivities-SourceManagementTableName'
    });

    new CfnOutput(this, 'ScrapingOperationsTableName', {
      value: scrapingOperationsTable.tableName,
      description: 'DynamoDB table name for scraping operations',
      exportName: 'SeattleFamilyActivities-ScrapingOperationsTableName'
    });

    new CfnOutput(this, 'AdminEventsTableName', {
      value: adminEventsTable.tableName,
      description: 'DynamoDB table name for admin events',
      exportName: 'SeattleFamilyActivities-AdminEventsTableName'
    });

    new CfnOutput(this, 'AdminApiFunctionName', {
      value: adminApiFunction.functionName,
      description: 'Admin API Lambda function name',
      exportName: 'SeattleFamilyActivities-AdminApiFunctionName'
    });

    new CfnOutput(this, 'AdminApiUrl', {
      value: adminApi.url,
      description: 'Admin API Gateway base URL',
      exportName: 'SeattleFamilyActivities-AdminApiUrl'
    });

    new CfnOutput(this, 'AdminApiId', {
      value: adminApi.restApiId,
      description: 'Admin API Gateway ID',
      exportName: 'SeattleFamilyActivities-AdminApiId'
    });

    new CfnOutput(this, 'ScrapingTaskQueueUrl', {
      value: scrapingTaskQueue.queueUrl,
      description: 'SQS queue URL for scraping tasks',
      exportName: 'SeattleFamilyActivities-ScrapingTaskQueueUrl'
    });

    new CfnOutput(this, 'ScrapingTaskQueueArn', {
      value: scrapingTaskQueue.queueArn,
      description: 'SQS queue ARN for scraping tasks',
      exportName: 'SeattleFamilyActivities-ScrapingTaskQueueArn'
    });

    new CfnOutput(this, 'TaskExecutorFunctionName', {
      value: taskExecutorFunction.functionName,
      description: 'Task executor Lambda function name',
      exportName: 'SeattleFamilyActivities-TaskExecutorFunctionName'
    });
  }
}