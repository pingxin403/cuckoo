# Code Quality and Linting

This document describes the code quality tools and linting configurations used in the Monorepo Hello/TODO Services project.

## Overview

We use language-specific linters and formatters to maintain code quality and consistency across the codebase:

- **Java**: Checkstyle + SpotBugs
- **Go**: golangci-lint
- **TypeScript**: ESLint + Prettier

## Quick Start

### Install Git Hooks

To automatically run linters before each commit:

```bash
./scripts/install-hooks.sh
```

### Run All Linters

```bash
make lint
```

### Auto-fix Lint Errors

```bash
make lint-fix
```

### Run All Formatters

```bash
make format
```

## Java (Hello Service)

### Tools

- **Checkstyle**: Enforces coding standards and style guidelines
- **SpotBugs**: Static analysis tool for finding bugs

### Configuration Files

- `apps/hello-service/config/checkstyle/checkstyle.xml` - Checkstyle rules
- `apps/hello-service/config/spotbugs/spotbugs-exclude.xml` - SpotBugs exclusions

### Usage

```bash
# Run Checkstyle
cd apps/hello-service
./gradlew checkstyleMain checkstyleTest

# Run SpotBugs
./gradlew spotbugsMain spotbugsTest

# Run all quality checks
./gradlew check

# Auto-fix formatting issues with Spotless
./gradlew spotlessApply

# Or from root
make lint APP=hello-service
make lint-fix APP=hello-service
```

### Reports

After running the checks, reports are available at:
- Checkstyle: `apps/hello-service/build/reports/checkstyle/`
- SpotBugs: `apps/hello-service/build/reports/spotbugs/`

### Common Issues

**Line too long**: Maximum line length is 120 characters. Break long lines into multiple lines.

**Missing Javadoc**: Public classes and methods should have Javadoc comments.

**Unused imports**: Remove unused import statements.

## Go (TODO Service)

### Tools

- **golangci-lint**: Fast Go linters runner with many linters included
- **gofmt**: Official Go formatter
- **goimports**: Manages import statements

### Configuration

- `apps/todo-service/.golangci.yml` - golangci-lint configuration

### Installation

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Or using Go
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Usage

```bash
# Run golangci-lint
cd apps/todo-service
golangci-lint run ./...

# Auto-fix issues
golangci-lint run --fix ./...

# Format code
gofmt -w .
goimports -w .

# Or from root
make lint APP=todo-service
make lint-fix APP=todo-service
make format APP=todo-service
```

### Enabled Linters

- **errcheck**: Check for unchecked errors
- **gosimple**: Simplify code
- **govet**: Vet examines Go source code
- **staticcheck**: Static analysis
- **unused**: Check for unused code
- **gofmt**: Check formatting
- **goimports**: Check imports
- **gosec**: Security checks
- **gocritic**: Opinionated linter
- And many more (see `.golangci.yml`)

### Common Issues

**Unchecked errors**: Always check error return values.

**Unused variables**: Remove or use variables prefixed with `_`.

**Exported functions without comments**: Add comments to exported functions.

## TypeScript (Web Application)

### Tools

- **ESLint**: Linter for JavaScript and TypeScript
- **Prettier**: Opinionated code formatter

### Configuration

- `apps/web/eslint.config.js` - ESLint configuration
- `apps/web/.prettierrc` - Prettier configuration
- `apps/web/.prettierignore` - Prettier ignore patterns

### Usage

```bash
# Run ESLint
cd apps/web
npm run lint

# Fix ESLint issues automatically
npm run lint -- --fix

# Format with Prettier
npm run format

# Check formatting
npm run format:check

# Or from root
make lint APP=web
make lint-fix APP=web
make format APP=web
```

### ESLint Rules

Key rules enforced:
- No unused variables (except those prefixed with `_`)
- Prefer `const` over `let`
- No `var` declarations
- Always use `===` instead of `==`
- Semicolons required
- Single quotes for strings
- React Hooks rules

### Prettier Configuration

- **Print width**: 100 characters
- **Tab width**: 2 spaces
- **Semicolons**: Required
- **Quotes**: Single quotes
- **Trailing commas**: Always (multiline)

### Common Issues

**Missing semicolons**: Add semicolons at the end of statements.

**Inconsistent quotes**: Use single quotes for strings.

**Unused imports**: Remove unused import statements.

**React Hooks dependencies**: Ensure all dependencies are listed in useEffect/useCallback/useMemo.

## CI/CD Integration

All linters are integrated into the CI/CD pipeline (`.github/workflows/ci.yml`):

1. **Protobuf verification**: Ensures generated code is up to date
2. **Java linting**: Runs Checkstyle and SpotBugs
3. **Go linting**: Runs golangci-lint
4. **TypeScript linting**: Runs ESLint and Prettier checks

Pull requests will fail if any linting issues are found.

## Pre-commit Hooks

The pre-commit hook (`.githooks/pre-commit`) automatically runs linters on changed files before each commit.

### Install Hooks

```bash
./scripts/install-hooks.sh
```

### Bypass Hooks

If you need to commit without running hooks (not recommended):

```bash
git commit --no-verify
```

## IDE Integration

### Visual Studio Code

Install these extensions for automatic linting and formatting:

**Java**:
- Language Support for Java by Red Hat
- Checkstyle for Java
- SpotBugs

**Go**:
- Go (by Go Team at Google)
- golangci-lint

**TypeScript**:
- ESLint
- Prettier - Code formatter

### IntelliJ IDEA / GoLand / WebStorm

These IDEs have built-in support for:
- Checkstyle (Java)
- SpotBugs (Java)
- golangci-lint (Go)
- ESLint (TypeScript)
- Prettier (TypeScript)

Configure them to use the project's configuration files.

## Best Practices

1. **Run linters locally** before pushing code
2. **Use auto-fix** when available: `make lint-fix` or `make lint-fix APP=<app-name>`
3. **Fix issues immediately** rather than accumulating technical debt
4. **Don't disable rules** without good reason and team discussion
5. **Keep configurations in sync** across the team
6. **Update tools regularly** to get latest bug fixes and improvements

## Troubleshooting

### Checkstyle Fails on Generated Code

Generated Protobuf code is excluded from Checkstyle checks. If you see issues, ensure the generated code is in the correct directory (`src/main/java-gen`).

### golangci-lint is Slow

golangci-lint can be slow on first run. Use `--fast` flag for quicker checks during development:

```bash
golangci-lint run --fast ./...
```

### ESLint Conflicts with Prettier

Our ESLint configuration is compatible with Prettier. If you see conflicts, ensure you're using the latest versions of both tools.

### Pre-commit Hook Fails

If the pre-commit hook fails:

1. Read the error message carefully
2. Fix the issues reported
3. Stage the fixed files: `git add <files>`
4. Commit again

To temporarily bypass (not recommended):
```bash
git commit --no-verify
```

## Additional Resources

- [Checkstyle Documentation](https://checkstyle.org/)
- [SpotBugs Documentation](https://spotbugs.github.io/)
- [golangci-lint Documentation](https://golangci-lint.run/)
- [ESLint Documentation](https://eslint.org/)
- [Prettier Documentation](https://prettier.io/)
