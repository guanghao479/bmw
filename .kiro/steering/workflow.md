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

## Visual Testing Guidelines

When implementing frontend changes or features that affect the user interface, perform visual testing using local development servers:

### CRITICAL: Always Use Makefile Targets
- **NEVER** use `python -m http.server` or `npx serve` directly
- **ALWAYS** use `make dev-frontend`, `make dev-backend`, or `make dev`
- The Makefile targets provide proper environment setup, validation, and process management

### When to Perform Visual Testing

- **UI/UX changes**: Any modifications to HTML, CSS, or JavaScript that affect visual appearance
- **Responsive design**: Changes that impact mobile, tablet, or desktop layouts
- **Interactive features**: New buttons, forms, navigation, or user interactions
- **Cross-browser compatibility**: Ensuring consistent behavior across different browsers
- **Admin interface changes**: Updates to admin.html or admin.js functionality

### Visual Testing Process

1. **Start Development Servers**:
   ```bash
   # ALWAYS use Makefile targets - NEVER use python -m http.server directly
   
   # Start both frontend and backend for full functionality testing
   make dev
   
   # Or start individually if needed
   make dev-frontend  # Frontend only (port 8000) - uses proper script with validation
   make dev-backend   # Backend only (for API testing) - uses AWS SAM CLI
   ```

2. **Test Scenarios**:
   - **Main Application**: Navigate to `http://localhost:8000` to test the main interface
   - **Admin Interface**: Navigate to `http://localhost:8000/admin.html` to test admin functionality
   - **Mobile Responsiveness**: Use browser dev tools to test different screen sizes
   - **Cross-browser Testing**: Test in Chrome, Firefox, Safari, and Edge when possible

3. **Validation Checklist**:
   - [ ] Layout renders correctly on desktop (1920x1080)
   - [ ] Mobile layout works properly (375x667 iPhone SE)
   - [ ] Tablet layout functions correctly (768x1024 iPad)
   - [ ] All interactive elements respond appropriately
   - [ ] Forms submit and validate correctly
   - [ ] Navigation works as expected
   - [ ] Loading states display properly
   - [ ] Error states are handled gracefully

4. **Stop Development Servers**:
   ```bash
   # Stop servers when testing is complete
   # Use Ctrl+C in the terminal where make dev is running
   # The scripts handle proper cleanup of both frontend and backend processes
   ```

### Integration with Development Workflow

- **Before committing**: Always perform visual testing for UI-related changes
- **After backend changes**: Test admin interface functionality if API changes affect it
- **Before deployment**: Final visual verification using local servers
- **Post-deployment**: Verify changes on GitHub Pages URL

## Emergency Procedures

If deployment is completely broken:
1. Revert to last known good commit: `git revert HEAD`
2. Push the revert: `git push origin main`
3. Wait for successful deployment
4. Investigate and fix the original issue in a new commit