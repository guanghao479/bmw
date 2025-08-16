# GitHub Actions OIDC Setup for AWS

This document provides step-by-step instructions to configure GitHub Actions to authenticate with AWS using OpenID Connect (OIDC) instead of long-lived access keys.

## Prerequisites

- AWS CLI configured with admin access
- GitHub repository with admin permissions
- Basic knowledge of AWS IAM

## Step 1: Create OIDC Identity Provider in AWS

1. Go to AWS IAM Console → Identity providers → Add provider
2. Select "OpenID Connect"
3. Provider URL: `https://token.actions.githubusercontent.com`
4. Audience: `sts.amazonaws.com`
5. Click "Add provider"

Or via AWS CLI:
```bash
aws iam create-open-id-connect-provider \
  --url https://token.actions.githubusercontent.com \
  --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1 \
  --client-id-list sts.amazonaws.com
```

## Step 2: Create IAM Role for GitHub Actions

Create an IAM role with the following trust policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::YOUR_AWS_ACCOUNT_ID:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringLike": {
          "token.actions.githubusercontent.com:sub": "repo:YOUR_GITHUB_USERNAME/bmw:*"
        },
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
```

**Important:** Replace `YOUR_AWS_ACCOUNT_ID` and `YOUR_GITHUB_USERNAME` with your actual values.

### Create Role via AWS CLI

```bash
# Create trust policy file
cat > github-actions-trust-policy.json << EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::YOUR_AWS_ACCOUNT_ID:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringLike": {
          "token.actions.githubusercontent.com:sub": "repo:YOUR_GITHUB_USERNAME/bmw:*"
        },
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
EOF

# Create the role
aws iam create-role \
  --role-name GitHubActions-SeattleFamilyActivities \
  --assume-role-policy-document file://github-actions-trust-policy.json
```

## Step 3: Attach Policies to the Role

The role needs the following policies for this project:

```bash
# Attach CDK and Lambda deployment policies
aws iam attach-role-policy \
  --role-name GitHubActions-SeattleFamilyActivities \
  --policy-arn arn:aws:iam::aws:policy/AWSLambdaFullAccess

aws iam attach-role-policy \
  --role-name GitHubActions-SeattleFamilyActivities \
  --policy-arn arn:aws:iam::aws:policy/AmazonS3FullAccess

aws iam attach-role-policy \
  --role-name GitHubActions-SeattleFamilyActivities \
  --policy-arn arn:aws:iam::aws:policy/CloudWatchFullAccess

aws iam attach-role-policy \
  --role-name GitHubActions-SeattleFamilyActivities \
  --policy-arn arn:aws:iam::aws:policy/AmazonEventBridgeFullAccess

aws iam attach-role-policy \
  --role-name GitHubActions-SeattleFamilyActivities \
  --policy-arn arn:aws:iam::aws:policy/AmazonSNSFullAccess

# For CDK deployments
aws iam attach-role-policy \
  --role-name GitHubActions-SeattleFamilyActivities \
  --policy-arn arn:aws:iam::aws:policy/IAMFullAccess

aws iam attach-role-policy \
  --role-name GitHubActions-SeattleFamilyActivities \
  --policy-arn arn:aws:iam::aws:policy/AWSCloudFormationFullAccess
```

## Step 4: Configure GitHub Repository Secrets

1. Go to your GitHub repository → Settings → Secrets and variables → Actions
2. Remove old secrets (if they exist):
   - `AWS_ACCESS_KEY_ID`
   - `AWS_SECRET_ACCESS_KEY`
3. Add new secret:
   - Name: `AWS_ROLE_TO_ASSUME`
   - Value: `arn:aws:iam::YOUR_AWS_ACCOUNT_ID:role/GitHubActions-SeattleFamilyActivities`

## Step 5: Verify the Setup

The GitHub Actions workflow has been updated to use OIDC. The key changes:

1. Added `permissions` block with `id-token: write` and `contents: read`
2. Updated `aws-actions/configure-aws-credentials@v4` to use `role-to-assume` instead of access keys

## Security Benefits

- **No long-lived credentials:** No access keys stored in GitHub
- **Fine-grained access:** Role can only be assumed by your specific repository
- **Temporary credentials:** Each workflow run gets fresh, short-lived tokens
- **Audit trail:** All role assumptions are logged in CloudTrail

## Troubleshooting

### Common Issues

1. **Role assumption fails:** Check that the trust policy repository path matches exactly
2. **Permission denied:** Ensure all required policies are attached to the role
3. **OIDC provider not found:** Verify the identity provider was created correctly

### Debug Commands

```bash
# Check if OIDC provider exists
aws iam list-open-id-connect-providers

# Verify role trust policy
aws iam get-role --role-name GitHubActions-SeattleFamilyActivities

# List attached policies
aws iam list-attached-role-policies --role-name GitHubActions-SeattleFamilyActivities
```

## Cleanup Old Access Keys

After verifying the OIDC setup works:

1. Go to AWS IAM Console → Users
2. Find the user with the old access keys
3. Delete the access keys under "Security credentials"
4. Optionally delete the user if it was created specifically for GitHub Actions