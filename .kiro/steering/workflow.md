# Development Workflow

## Task Completion Process

After completing any development task, follow this mandatory workflow:

### 1. Commit Changes
```bash
# Stage all changes
git add .

# Create descriptive commit message
git commit -m "feat: [brief description of changes]"
# or
git commit -m "fix: [brief description of fix]"
# or  
git commit -m "docs: [brief description of documentation changes]"
```

### 2. Push to Repository
```bash
# Push to main branch (triggers GitHub Actions)
git push origin main
```

### 3. Monitor GitHub Actions
```bash
# Check the status of the latest workflow run
gh run list --limit 1

# Watch the workflow in real-time (if gh CLI available)
gh run watch

# Or check via web interface
echo "Monitor at: https://github.com/[username]/[repo]/actions"
```

### 4. Handle Deployment Issues

If GitHub Actions fails:

1. **Check the logs**:
   ```bash
   gh run view --log
   ```

2. **Common failure scenarios**:
   - **CDK deployment errors**: Check AWS credentials and permissions
   - **Build failures**: Verify Go compilation and dependencies
   - **Test failures**: Run tests locally first with `go test ./...`
   - **Infrastructure issues**: Check CDK diff before deployment

3. **Fix and retry**:
   ```bash
   # Make necessary fixes
   git add .
   git commit -m "fix: resolve deployment issue - [description]"
   git push origin main
   ```

### 5. Verify Deployment Success

After successful GitHub Actions run:

1. **Frontend changes**: Verify at GitHub Pages URL
2. **Backend changes**: Test API endpoints
3. **Infrastructure changes**: Confirm resources in AWS Console

## Commit Message Conventions

Use conventional commit format:
- `feat:` - New features
- `fix:` - Bug fixes  
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

## Automation Requirements

- **Never skip commits**: Every task completion must result in a commit
- **Always push**: Commits must be pushed to trigger CI/CD
- **Monitor results**: Always verify GitHub Actions success
- **Fix immediately**: Address any deployment failures before moving to next task

## GitHub Actions Workflows

The repository uses these automated workflows:
- **Frontend deployment**: Automatic GitHub Pages deployment on main branch push
- **Backend deployment**: CDK deployment via AWS OIDC authentication
- **Testing**: Automated test runs on pull requests and main branch

## Emergency Procedures

If deployment is completely broken:
1. Revert to last known good commit: `git revert HEAD`
2. Push the revert: `git push origin main`
3. Wait for successful deployment
4. Investigate and fix the original issue in a new commit