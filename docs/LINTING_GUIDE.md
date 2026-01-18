# Linting Guide

This guide explains the linting configuration and usage across different application types in the monorepo.

## Overview

The monorepo supports linting for three types of applications:
- **Java** (Spring Boot services)
- **Go** (gRPC services)
- **Node.js** (React/TypeScript applications)

## Running Linters

### Lint All Changed Apps
```bash
make lint
```

### Lint Specific App
```bash
make lint APP=hello-service
make lint APP=todo-service
make lint APP=web
```

### Auto-fix Lint Errors
```bash
# Auto-fix all changed apps
make lint-fix

# Auto-fix specific app
make lint-fix APP=hello-service
make lint-fix APP=todo-service
make lint-fix APP=web
```

**What gets auto-fixed:**
- **Java**: Spotless formatting (imports, whitespace, line endings)
- **Go**: golangci-lint auto-fixable issues + gofmt formatting
- **Node.js**: ESLint auto-fixable issues

**Note**: Some issues (like SpotBugs violations in Java) require manual fixes.

## Java Linting

### Configuration
Java applications can use **Checkstyle** for code style enforcement.

**Status**: Currently disabled by default in templates and existing apps.

**Location**: 
- Configuration: `build.gradle` (commented out)
- Rules: Would be in `config/checkstyle/checkstyle.xml` (not created yet)

**To Enable**:
1. Uncomment the `checkstyle` plugin in `build.gradle`
2. Create a `config/checkstyle/checkstyle.xml` file with your rules
3. Uncomment the checkstyle configuration block at the end of `build.gradle`

**Example**:
```gradle
plugins {
    id 'checkstyle'
}

checkstyle {
    toolVersion = '10.12.5'
    configFile = file("${rootDir}/config/checkstyle/checkstyle.xml")
    ignoreFailures = false
    maxWarnings = 0
}
```

### Current Behavior
When Checkstyle is not configured, `make lint` will show a warning but won't fail:
```
[WARNING] Checkstyle not configured, skipping...
[SUCCESS] Linting passed for hello-service
```

## Go Linting

### Configuration
Go applications use **golangci-lint** for comprehensive code analysis.

**Status**: ✅ Fully configured and working

**Location**: `.golangci.yml` in each Go service directory

**Enabled Linters**:
- `errcheck` - Check for unchecked errors
- `govet` - Vet examines Go source code
- `ineffassign` - Detect ineffectual assignments
- `unused` - Check for unused code
- `misspell` - Check for misspelled words
- `gosec` - Security checks
- `gocyclo` - Cyclomatic complexity
- `unconvert` - Remove unnecessary type conversions

**Disabled Linters**:
- `staticcheck` - Disabled due to false positives in test code
- Formatters (`gofmt`, `goimports`, `gofumpt`) - Use `make format` instead

**Configuration File**:
```yaml
# golangci-lint configuration
version: "2"

run:
  timeout: 5m
  tests: true
  modules-download-mode: readonly

linters:
  disable-all: true
  enable:
    - errcheck
    - govet
    - ineffassign
    - unused
    - misspell
    - gosec
    - gocyclo
    - unconvert
  disable:
    - staticcheck

issues:
  exclude-rules:
    - path: _test\.go$
      linters:
        - gocyclo
        - errcheck
        - gosec
        - staticcheck
    - path: gen/
      linters:
        - all
```

### Installation
```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Or use go install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Node.js Linting

### Configuration
Node.js applications use **ESLint** for TypeScript/JavaScript linting.

**Status**: ✅ Fully configured and working

**Location**: `eslint.config.js` in each Node.js app directory

**Configuration**:
- TypeScript support via `@typescript-eslint`
- React hooks rules
- React refresh rules
- Strict quote and semicolon rules

**Key Rules**:
- Single quotes for strings
- Semicolons required
- Trailing commas in multiline
- No unused variables (except those starting with `_`)

**Auto-fix**:
```bash
cd apps/web
npm run lint -- --fix
```

### ESLint Configuration Example
```javascript
import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  { ignores: ['dist'] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],
      'quotes': ['error', 'single'],
      'semi': ['error', 'always'],
    },
  },
);
```

## Template Configuration

### Go Service Template
**Location**: `templates/go-service/.golangci.yml`

**Status**: ✅ Includes working golangci-lint configuration

**Usage**: Automatically copied when creating a new Go service with `make create`

### Java Service Template
**Location**: `templates/java-service/build.gradle`

**Status**: ⚠️ Checkstyle configuration is commented out

**Usage**: New Java services will have Checkstyle disabled by default. Uncomment to enable.

### Node.js Template
**Status**: ❌ No template exists yet

**Note**: The `create-app.sh` script supports creating Node.js apps, but there's no template directory. Consider creating `templates/node-service/` with ESLint configuration.

## Integration with CI/CD

The linting is integrated into the CI/CD pipeline:

```yaml
# .github/workflows/ci.yml
- name: Lint
  run: make lint
```

This ensures all code changes are linted before merging.

## Troubleshooting

### Java: "Task 'checkstyleMain' not found"
**Cause**: Checkstyle plugin is not enabled in `build.gradle`

**Solution**: This is expected. Either:
1. Enable Checkstyle by uncommenting the plugin and configuration
2. Or ignore the warning (linting will pass with a warning message)

### Go: "can't load config: unsupported version"
**Cause**: Missing `version` field in `.golangci.yml`

**Solution**: Add `version: "2"` at the top of `.golangci.yml`

### Go: "unknown linters: 'gosimple,exportloopref'"
**Cause**: Linter names changed in newer versions of golangci-lint

**Solution**: Use the updated configuration from `templates/go-service/.golangci.yml`

### Node.js: Many quote/semicolon errors
**Cause**: Code doesn't follow ESLint rules

**Solution**: Run auto-fix:
```bash
cd apps/web
npm run lint -- --fix
```

## Best Practices

1. **Run linting locally** before committing:
   ```bash
   make lint
   ```

2. **Fix issues automatically** when possible:
   - Go: Use `make format APP=<app-name>`
   - Node.js: Use `npm run lint -- --fix`

3. **Keep configurations in sync**:
   - When updating linting rules, update both the app and template
   - Test changes with `make lint` before committing

4. **Don't disable linters without reason**:
   - If a linter reports false positives, configure exclusions instead
   - Document why specific linters are disabled

5. **Use pre-commit hooks**:
   - The monorepo includes pre-commit hooks that run linting
   - Install with: `./scripts/install-hooks.sh`

## Summary

| App Type | Linter | Status | Auto-fix | Template |
|----------|--------|--------|----------|----------|
| Java | Checkstyle | ⚠️ Disabled | No | ⚠️ Commented |
| Go | golangci-lint | ✅ Working | Partial | ✅ Included |
| Node.js | ESLint | ✅ Working | Yes | ❌ Missing |

**Recommendations**:
1. Create Node.js template with ESLint configuration
2. Consider enabling Checkstyle for Java services
3. Document team-specific linting rules in this guide
