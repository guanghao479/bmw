# AWS OIDC Infrastructure Setup Task

**Status:** In Progress  
**Started:** August 16, 2025  
**Objective:** Replace GitHub Actions AWS access keys with secure OIDC authentication via infrastructure as code

## Background

Currently, the GitHub Actions workflow uses long-lived AWS access keys stored as secrets. This poses security risks:
- Keys don't expire automatically
- Difficult to rotate
- Broad access if compromised
- No audit trail of which workflow used them

OIDC (OpenID Connect) provides a more secure alternative with:
- Temporary credentials (1-hour expiration)
- Repository-scoped access
- Cryptographic proof of origin
- Full CloudTrail audit logging

## Technical Implementation

### Repository Context
- GitHub Repository: `guanghao479/bmw`
- AWS Region: `us-west-2`
- CDK Stack: `SeattleFamilyActivitiesMVPStack`
- Current Stack File: `infrastructure/lib/mvp-stack.ts`

### OIDC Authentication Flow
1. GitHub Actions workflow starts
2. GitHub issues signed JWT token with repository claims
3. AWS validates token signature using GitHub's public key
4. AWS checks trust policy conditions (repository match)
5. AWS STS returns temporary credentials (1-hour validity)
6. Workflow uses temporary credentials for AWS operations

### Trust Policy Security
The IAM role trust policy ensures only workflows from `guanghao479/bmw` can assume the role:

```json
{
  "StringLike": {
    "token.actions.githubusercontent.com:sub": "repo:guanghao479/bmw:*"
  },
  "StringEquals": {
    "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
  }
}
```

## Implementation Steps

### Phase 1: Infrastructure as Code Changes

**File:** `infrastructure/lib/mvp-stack.ts`

**Additions:**
1. OIDC Identity Provider for GitHub Actions
2. IAM Role with GitHub Actions trust policy
3. Required AWS managed policy attachments
4. CloudFormation output for role ARN

**Required Policies:**
- `AWSLambdaFullAccess` - Lambda function management
- `AmazonS3FullAccess` - S3 bucket operations
- `CloudWatchFullAccess` - Monitoring and logging
- `AmazonEventBridgeFullAccess` - Scheduled events
- `AmazonSNSFullAccess` - Alert notifications
- `IAMFullAccess` - CDK role management
- `AWSCloudFormationFullAccess` - CDK deployments

### Phase 2: GitHub Repository Configuration

**Secrets to Remove:**
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`

**Secrets to Add:**
- `AWS_ROLE_TO_ASSUME` (value from CDK output)

**Workflow Changes:**
- Already updated in `.github/workflows/deploy.yml`
- Uses `role-to-assume` instead of access keys
- Added `permissions.id-token: write` for OIDC

### Phase 3: Deployment and Testing

1. Deploy CDK stack: `cdk deploy`
2. Update GitHub secrets with role ARN
3. Test workflow deployment
4. Verify all operations work correctly

## Security Benefits

### Before (Access Keys)
- ❌ Long-lived credentials (never expire)
- ❌ Static secrets stored in GitHub
- ❌ Hard to rotate and audit
- ❌ Broad access if compromised

### After (OIDC)
- ✅ Temporary credentials (1-hour expiration)
- ✅ No secrets stored in GitHub
- ✅ Automatic rotation built-in
- ✅ Repository-scoped access only
- ✅ Full CloudTrail audit trail
- ✅ Cryptographic proof of origin

## Risk Mitigation

**Potential Issues:**
1. OIDC provider creation might fail if it already exists
2. Policy attachments might need adjustment for specific operations
3. Token expiration could affect long-running workflows

**Mitigations:**
1. CDK will handle existing provider gracefully
2. Using AWS managed policies ensures comprehensive access
3. Current workflow completes in ~10 minutes (well under 1-hour limit)

## Rollback Plan

If issues occur during or after deployment:

```bash
# Option 1: Rollback to previous CDK stack version
cd infrastructure
cdk destroy
# Then redeploy previous version

# Option 2: Manual resource cleanup
aws iam delete-role --role-name GitHubActionsDeploymentRole
aws iam delete-open-id-connect-provider --open-id-connect-provider-arn <arn>

# Option 3: Revert GitHub Actions workflow to use access keys
git revert <commit-hash>
```

## Success Criteria

- [ ] CDK deployment completes successfully
- [ ] GitHub Actions workflow authenticates with OIDC
- [ ] Lambda function deploys successfully
- [ ] S3 operations work correctly
- [ ] End-to-end pipeline functions normally
- [ ] No regression in deployment functionality

## Implementation Log

### August 16, 2025 - Initial Setup
- ✅ Created implementation documentation
- ✅ Analyzed current CDK stack structure
- ✅ Planned infrastructure modifications
- ✅ Identified required AWS policies

### August 16, 2025 - CDK Infrastructure Updates
- ✅ Added OIDC Identity Provider for GitHub Actions to CDK stack
- ✅ Added IAM Role `GitHubActions-SeattleFamilyActivities` with trust policy
- ✅ Fixed policy name issue: `AWSLambdaFullAccess` → `AWSLambda_FullAccess`
- ✅ Successfully deployed CDK stack with all OIDC resources
- ✅ Role ARN: `arn:aws:iam::009952409073:role/GitHubActions-SeattleFamilyActivities`

### Next Steps
- Update GitHub repository secrets with new role ARN
- Remove old AWS access key secrets
- Test GitHub Actions workflow with OIDC authentication