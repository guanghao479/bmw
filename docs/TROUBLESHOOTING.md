# Local Development Troubleshooting Guide

This guide covers common issues you might encounter when setting up and running the local development environment.

## Common Issues and Solutions

### 1. SAM CLI Issues

#### Error: "sam: command not found"

**Problem**: AWS SAM CLI is not installed or not in PATH.

**Solution**:
```bash
# Install SAM CLI (macOS)
brew install aws-sam-cli

# Verify installation
sam --version
```

#### Error: "Template file not found"

**Problem**: CDK template hasn't been synthesized.

**Solution**:
```bash
cd infrastructure
npm install
cdk synth
```

#### Error: "Port 3000 is already in use"

**Problem**: Another process is using port 3000.

**Solutions**:
1. **Find and kill the process**:
   ```bash
   lsof -ti:3000 | xargs kill -9
   ```

2. **Use a different port**:
   ```bash
   sam local start-api --port 3001 --template-file ...
   ```
   
   Then update frontend configuration to use port 3001.

#### Error: "Unable to import module 'main'"

**Problem**: Lambda function binary not built or not found.

**Solution**:
```bash
cd backend
go build -o ../testing/bin/admin_api ./cmd/admin_api
go build -o ../testing/bin/scraping_orchestrator ./cmd/scraping_orchestrator
```

### 2. AWS Credentials Issues

#### Error: "Unable to locate credentials"

**Problem**: AWS credentials not configured.

**Solutions**:
1. **Configure AWS CLI**:
   ```bash
   aws configure
   ```

2. **Use AWS profile**:
   ```bash
   export AWS_PROFILE=your-profile-name
   ```

3. **Set environment variables**:
   ```bash
   export AWS_ACCESS_KEY_ID=your-key
   export AWS_SECRET_ACCESS_KEY=your-secret
   export AWS_REGION=us-west-2
   ```

#### Error: "Access Denied" when accessing DynamoDB

**Problem**: AWS credentials don't have DynamoDB permissions.

**Solution**:
- Ensure your AWS user/role has DynamoDB permissions
- Check if tables exist in the correct region (us-west-2)
- Verify table names match the environment configuration

### 3. Environment Variable Issues

#### Error: "FIRECRAWL_API_KEY not set"

**Problem**: Required API keys not configured.

**Solutions**:
1. **Using direnv** (recommended):
   ```bash
   # Create .envrc file
   echo 'export FIRECRAWL_API_KEY="your_key_here"' >> .envrc
   echo 'export OPENAI_API_KEY="your_key_here"' >> .envrc
   direnv allow
   ```

2. **Manual export**:
   ```bash
   export FIRECRAWL_API_KEY="your_key_here"
   export OPENAI_API_KEY="your_key_here"
   ```

#### Error: "direnv: command not found"

**Problem**: direnv not installed.

**Solution**:
```bash
# Install direnv
brew install direnv

# Add to shell configuration
echo 'eval "$(direnv hook zsh)"' >> ~/.zshrc
source ~/.zshrc
```

### 4. Go Build Issues

#### Error: "go: command not found"

**Problem**: Go not installed or not in PATH.

**Solution**:
```bash
# Install Go (macOS)
brew install go

# Verify installation
go version
```

#### Error: "package not found" or module issues

**Problem**: Go dependencies not downloaded.

**Solution**:
```bash
cd backend
go mod tidy
go mod download
```

#### Error: Build fails with compilation errors

**Problem**: Code syntax errors or missing dependencies.

**Solutions**:
1. **Check Go version** (requires 1.22.5+):
   ```bash
   go version
   ```

2. **Run tests to identify issues**:
   ```bash
   go test ./...
   ```

3. **Check for missing imports**:
   ```bash
   go mod tidy
   ```

### 5. Frontend Issues

#### Error: "Failed to fetch" in browser console

**Problem**: Frontend can't connect to backend.

**Solutions**:
1. **Verify backend is running**:
   ```bash
   curl http://127.0.0.1:3000/api/sources
   ```

2. **Check CORS configuration**:
   - SAM CLI should handle CORS automatically
   - Verify frontend is using correct localhost URLs

3. **Check browser network tab**:
   - Look for failed requests
   - Verify request URLs are correct

#### Error: "Address already in use" for frontend server

**Problem**: Port 8000 is already in use.

**Solutions**:
1. **Kill existing process**:
   ```bash
   lsof -ti:8000 | xargs kill -9
   ```

2. **Use different port**:
   ```bash
   cd app
   python3 -m http.server 8001
   ```

#### Error: Python not found

**Problem**: Python not installed or not in PATH.

**Solutions**:
1. **Install Python**:
   ```bash
   # macOS (usually pre-installed)
   python3 --version
   
   # If not available
   brew install python
   ```

2. **Use alternative server**:
   ```bash
   # Node.js
   npx serve app
   
   # PHP
   php -S localhost:8000 -t app
   ```

### 6. CDK Issues

#### Error: "cdk: command not found"

**Problem**: AWS CDK not installed.

**Solution**:
```bash
npm install -g aws-cdk
cdk --version
```

#### Error: "Cannot find module" in CDK

**Problem**: Node.js dependencies not installed.

**Solution**:
```bash
cd infrastructure
npm install
```

#### Error: CDK synthesis fails

**Problem**: CDK code has errors or dependencies missing.

**Solutions**:
1. **Check CDK code syntax**:
   ```bash
   cd infrastructure
   npm run build
   ```

2. **Update CDK version**:
   ```bash
   npm update aws-cdk-lib
   ```

### 7. Network and Connectivity Issues

#### Error: "Connection refused" to localhost

**Problem**: Service not running or wrong port.

**Solutions**:
1. **Check if service is running**:
   ```bash
   # Check SAM local
   curl http://127.0.0.1:3000
   
   # Check frontend
   curl http://localhost:8000
   ```

2. **Check process status**:
   ```bash
   ps aux | grep sam
   ps aux | grep python
   ```

3. **Restart services**:
   ```bash
   # Stop all
   pkill -f "sam local"
   pkill -f "python.*http.server"
   
   # Restart
   ./start-local-backend.sh
   ./start-local-frontend.sh
   ```

#### Error: CORS errors in browser

**Problem**: Cross-origin requests blocked.

**Solutions**:
1. **Verify SAM CLI CORS handling**:
   - SAM CLI should automatically handle CORS for local development
   - Check SAM CLI logs for CORS-related messages

2. **Use same origin**:
   - Access frontend via http://localhost:8000 (not file://)
   - Ensure backend uses 127.0.0.1:3000

### 8. Performance Issues

#### Issue: Slow startup times

**Solutions**:
1. **Pre-build Lambda functions**:
   ```bash
   cd backend
   go build -o ../testing/bin/admin_api ./cmd/admin_api
   go build -o ../testing/bin/scraping_orchestrator ./cmd/scraping_orchestrator
   ```

2. **Use faster storage**:
   - Run from SSD if possible
   - Exclude testing/ directory from antivirus scans

#### Issue: High memory usage

**Solutions**:
1. **Limit SAM CLI memory**:
   ```bash
   sam local start-api --memory 512
   ```

2. **Monitor processes**:
   ```bash
   top -o mem
   ```

## Debugging Tips

### 1. Enable Debug Logging

**Backend**:
```bash
# Set LOG_LEVEL=DEBUG in env.json
export LOG_LEVEL=DEBUG
```

**SAM CLI**:
```bash
sam local start-api --debug
```

### 2. Check Logs

**SAM CLI logs**:
- Displayed in terminal where SAM is running
- Look for Lambda function errors and API Gateway logs

**Frontend logs**:
- Open browser Developer Tools (F12)
- Check Console tab for JavaScript errors
- Check Network tab for API request failures

### 3. Test Individual Components

**Test backend directly**:
```bash
# Test API endpoints
curl -X GET http://127.0.0.1:3000/api/sources
curl -X POST http://127.0.0.1:3000/api/sources -d '{"name":"test","url":"http://example.com"}'
```

**Test frontend without backend**:
- Check if HTML/CSS loads correctly
- Test JavaScript functionality that doesn't require API calls

### 4. Validate Configuration

**Check environment variables**:
```bash
# In backend directory
cat env.json

# Check loaded variables
env | grep -E "(FIRECRAWL|OPENAI|AWS)"
```

**Verify CDK template**:
```bash
cd infrastructure
cat cdk.out/SeattleFamilyActivitiesMVPStack.template.json | jq '.Resources'
```

## Getting Help

If you're still experiencing issues:

1. **Check the logs** for specific error messages
2. **Search the AWS SAM CLI documentation**: https://docs.aws.amazon.com/serverless-application-model/
3. **Review the CDK documentation**: https://docs.aws.amazon.com/cdk/
4. **Check GitHub issues** for similar problems
5. **Create a minimal reproduction** of the issue

## Common Commands Reference

```bash
# Start everything
cd backend && ./start-local-servers.sh

# Start backend only
cd backend && ./start-local-backend.sh

# Start frontend only
./start-local-frontend.sh

# Rebuild and restart backend
cd backend
go build -o ../testing/bin/admin_api ./cmd/admin_api
./start-local-backend.sh

# Re-synthesize CDK
cd infrastructure && cdk synth

# Check running processes
ps aux | grep -E "(sam|python.*http.server)"

# Kill all local servers
pkill -f "sam local"
pkill -f "python.*http.server"
```