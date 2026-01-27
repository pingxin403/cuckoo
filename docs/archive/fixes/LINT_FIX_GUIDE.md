# Lint Fix Guide

This guide explains how to use the `make lint-fix` command to automatically fix linting errors in your code.

## Overview

The `make lint-fix` command automatically fixes common code quality issues across all application types:

- **Java**: Spotless formatting (imports, whitespace, line endings)
- **Go**: golangci-lint auto-fixable issues + gofmt formatting
- **Node.js**: ESLint auto-fixable issues

## Usage

### Fix All Changed Apps

```bash
make lint-fix
```

This will automatically detect which apps have changed (using git) and fix their lint errors.

### Fix Specific App

```bash
make lint-fix APP=hello-service
make lint-fix APP=todo-service
make lint-fix APP=web
```

## What Gets Fixed

### Java Applications

**Spotless** automatically fixes:
- Import order and organization
- Trailing whitespace
- Line endings (ensures files end with newline)
- Code formatting according to Google Java Format

**Example**:
```bash
make lint-fix APP=hello-service
```

**Output**:
```
[INFO] Auto-fixing lint errors for hello-service (java)...
[SUCCESS] Spotless formatting applied
[INFO] Note: SpotBugs issues may require manual fixes
[SUCCESS] Lint errors auto-fixed for hello-service
```

**Note**: SpotBugs violations (like potential bugs or security issues) typically require manual fixes.

### Go Applications

**golangci-lint** with `--fix` flag automatically fixes:
- Unused imports
- Formatting issues
- Simple code improvements
- Some inefficient assignments

**gofmt** and **goimports** (fallback) fix:
- Code formatting
- Import organization

**Example**:
```bash
make lint-fix APP=todo-service
```

**Output**:
```
[INFO] Auto-fixing lint errors for todo-service (go)...
0 issues.
[SUCCESS] Lint errors auto-fixed for todo-service
```

### Node.js Applications

**ESLint** with `--fix` flag automatically fixes:
- Missing semicolons
- Quote style (single vs double)
- Indentation
- Trailing commas
- Unused imports
- And many more auto-fixable rules

**Example**:
```bash
make lint-fix APP=web
```

**Output**:
```
[INFO] Auto-fixing lint errors for web (node)...
[SUCCESS] Lint errors auto-fixed for web
```

## Workflow

### Recommended Development Workflow

1. **Write code** as usual
2. **Run lint-fix** before committing:
   ```bash
   make lint-fix
   ```
3. **Verify fixes** with lint check:
   ```bash
   make lint
   ```
4. **Manually fix** any remaining issues that couldn't be auto-fixed
5. **Commit** your changes

### Pre-commit Hook Integration

The pre-commit hook automatically runs linters on changed files. To ensure smooth commits:

```bash
# Before committing
make lint-fix

# Then commit
git add .
git commit -m "Your commit message"
```

## Manual Fixes Required

Some issues cannot be auto-fixed and require manual intervention:

### Java (SpotBugs Issues)

- **Potential bugs**: Null pointer dereferences, resource leaks
- **Security issues**: SQL injection, XSS vulnerabilities
- **Performance issues**: Inefficient algorithms, unnecessary object creation

**How to fix**: Review the SpotBugs report at `apps/hello-service/build/reports/spotbugs/main/spotbugs.html`

### Go (Complex Issues)

- **Logic errors**: Incorrect error handling, race conditions
- **Complexity issues**: Functions that are too complex
- **Security issues**: Potential vulnerabilities

**How to fix**: Review the golangci-lint output and refactor code as needed

### Node.js (Complex Rules)

- **React Hooks dependencies**: Missing dependencies in useEffect/useCallback
- **Type issues**: TypeScript type errors
- **Logic errors**: Unreachable code, incorrect comparisons

**How to fix**: Review the ESLint output and update code accordingly

## Troubleshooting

### Lint-fix Doesn't Fix All Issues

This is expected. Auto-fix only handles formatting and simple issues. Complex problems require manual fixes.

**Solution**: Run `make lint` to see remaining issues and fix them manually.

### Changes Not Applied

Make sure you have write permissions to the files and that the files are not read-only.

**Solution**: Check file permissions and ensure you're not in a read-only directory.

### Conflicts with Manual Formatting

If you've manually formatted code differently, lint-fix will override your formatting.

**Solution**: Let the automated tools handle formatting consistently. Don't fight the formatter.

### Git Shows Many Changes After Lint-fix

This is normal, especially on first run. The tools are standardizing formatting across the codebase.

**Solution**: Review the changes, ensure they're formatting-only, and commit them.

## Best Practices

1. **Run lint-fix regularly**: Before committing, before pushing, after merging
2. **Don't disable auto-fix**: It maintains consistency across the team
3. **Commit formatting separately**: If lint-fix makes many changes, commit them separately from logic changes
4. **Review auto-fixes**: Quickly review what was changed to ensure it's correct
5. **Fix remaining issues**: Don't ignore issues that can't be auto-fixed

## Integration with CI/CD

The CI/CD pipeline runs `make lint` (not `make lint-fix`) to verify code quality. This means:

- **Locally**: Use `make lint-fix` to fix issues before pushing
- **CI/CD**: Runs `make lint` to verify no issues remain
- **Pull Requests**: Will fail if lint issues are found

**Workflow**:
```bash
# Before pushing
make lint-fix
make lint  # Verify all issues are fixed
git add .
git commit -m "Fix lint issues"
git push
```

## Examples

### Example 1: Fix Java Formatting

```bash
# Check current issues
make lint APP=hello-service

# Auto-fix
make lint-fix APP=hello-service

# Verify fixes
make lint APP=hello-service
```

### Example 2: Fix All Changed Apps

```bash
# Make some changes to multiple apps
vim apps/hello-service/src/main/java/...
vim apps/todo-service/service/...
vim apps/web/src/...

# Fix all changed apps
make lint-fix

# Verify
make lint
```

### Example 3: Pre-commit Workflow

```bash
# Make changes
vim apps/hello-service/...

# Fix lint issues
make lint-fix APP=hello-service

# Run tests
make test APP=hello-service

# Commit
git add apps/hello-service
git commit -m "Add new feature"
```

## Related Commands

- `make lint` - Check for lint errors without fixing
- `make format` - Format code (similar to lint-fix but focuses on formatting)
- `make lint APP=<name>` - Check specific app
- `make format APP=<name>` - Format specific app

## Additional Resources

- [Linting Guide](LINTING_GUIDE.md) - Detailed linting configuration
- [Code Quality Guide](CODE_QUALITY.md) - Code quality standards
- [App Management Guide](APP_MANAGEMENT.md) - App management commands
