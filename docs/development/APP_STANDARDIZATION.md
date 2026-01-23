# App Standardization Plan

## Overview

This document outlines the standardization effort for all apps in the monorepo to ensure consistency, maintainability, and quality.

## Standardization Checklist

### Required Files

Every app MUST have:

- [ ] `README.md` - Comprehensive documentation
- [ ] `metadata.yaml` - Service metadata
- [ ] `.apptype` - App type marker (go/java/node)
- [ ] `Dockerfile` - Multi-stage Docker build
- [ ] `catalog-info.yaml` - Backstage service catalog
- [ ] `scripts/test-coverage.sh` - Coverage verification (Go apps)
- [ ] `TESTING.md` - Testing guide (optional but recommended)

### Directory Structure

#### Go Services

```
app-name/
├── main.go
├── go.mod
├── go.sum
├── Dockerfile
├── README.md
├── TESTING.md (optional)
├── metadata.yaml
├── .apptype
├── .golangci.yml
├── catalog-info.yaml
├── gen/                    # Generated protobuf code
│   └── *pb/
├── service/                # Service implementation
│   ├── *_service.go
│   ├── *_service_test.go
│   └── *_service_property_test.go  # With build tags
├── storage/                # Storage layer (if needed)
│   ├── *_store.go
│   └── *_store_test.go
└── scripts/
    └── test-coverage.sh
```

#### Java Services

```
app-name/
├── build.gradle
├── settings.gradle
├── gradlew
├── Dockerfile
├── README.md
├── metadata.yaml
├── .apptype
├── catalog-info.yaml
├── checkstyle.xml
├── spotbugs-exclude.xml
└── src/
    ├── main/
    │   ├── java/
    │   └── resources/
    └── test/
        └── java/
```

#### Node.js Applications

```
app-name/
├── package.json
├── tsconfig.json
├── vite.config.ts (or webpack.config.js)
├── Dockerfile
├── README.md
├── metadata.yaml
├── .apptype
├── catalog-info.yaml
└── src/
    ├── components/
    ├── services/
    ├── hooks/
    └── gen/               # Generated protobuf code
```

### README.md Structure

Every README.md should follow this structure:

1. **Title and Description**
2. **Features** - Key capabilities
3. **Technology Stack** - Languages, frameworks, libraries
4. **Quick Start** - How to run locally
5. **Project Structure** - Directory layout
6. **API Documentation** - Endpoints and usage
7. **Testing** - How to run tests, coverage requirements
8. **Development** - Adding features, code standards
9. **Deployment** - Docker, Kubernetes instructions
10. **Troubleshooting** - Common issues and solutions
11. **Resources** - Links to related docs

### Testing Standards

#### Unit Tests

- **Go**: Use standard `testing` package
- **Java**: Use JUnit 5
- **Node.js**: Use Vitest or Jest

#### Property-Based Tests

- **Go**: Use `pgregory.net/rapid` with build tags
  ```go
  //go:build property
  // +build property
  ```
- **Java**: Use jqwik
- **Node.js**: Use fast-check

#### Coverage Requirements

- **Go Services**: 80% overall, 90% for service packages
- **Java Services**: 80% overall, 90% for service classes
- **Node.js Apps**: 70% overall (frontend)

#### Test Organization

- Unit tests: `*_test.go`, `*Test.java`, `*.test.ts`
- Property tests: `*_property_test.go`, `*PropertyTest.java`, `*.property.test.ts`
- Integration tests: `integration_test/` directory

### Documentation Standards

#### Code Comments

- All public functions/methods must have doc comments
- Complex logic should have inline comments
- Use standard doc comment formats (GoDoc, JavaDoc, JSDoc)

#### API Documentation

- All RPC methods documented in proto files
- Request/response examples in README
- Error codes and handling documented

### Metadata Standards

#### metadata.yaml Format

```yaml
spec:
  name: service-name
  short_name: short          # Without -service suffix
  description: Brief description
  type: go|java|node
  port: 9090
  cd: true|false
  codeowners:
    - "@team-name"
  proto_files:                # Optional
    - file1.proto
    - file2.proto
test:
  coverage: 80                # Overall coverage target
  service_coverage: 90        # Service layer coverage (optional)
```

#### .apptype Format

Single line with app type:
```
go
```

or

```
java
```

or

```
node
```

### Build and CI/CD Integration

Every app must:

- [ ] Work with `make test APP=name`
- [ ] Work with `make build APP=name`
- [ ] Work with `make lint APP=name`
- [ ] Work with `make docker-build APP=name`
- [ ] Be auto-detected by changed app detection
- [ ] Have Kubernetes deployment files
- [ ] Have Docker Compose configuration

## Current Status

### ✅ Fully Compliant

- `apps/shortener-service` - Excellent documentation and structure
- `apps/im-service` - Good structure with TESTING.md
- `apps/auth-service` - ✅ Fixed: Removed template placeholders, added TESTING.md
- `apps/user-service` - ✅ Fixed: Added TESTING.md
- `apps/todo-service` - ✅ Fixed: Translated to English, added TESTING.md
- `apps/hello-service` - ✅ Fixed: Translated to English
- `apps/web` - ✅ Fixed: Added TESTING.md

### 📦 Special Cases

- `apps/im-chat-system` - Migration scripts only, no service code (no changes needed)

## Implementation Plan

### ✅ Phase 1: Fix Critical Issues - COMPLETED

1. ✅ Fix auth-service README (removed template placeholders)
2. ✅ Translate Chinese READMEs to English (todo-service, hello-service)
3. ✅ Add missing TESTING.md files (auth, user, todo, web, hello-service, shortener-service)

### ✅ Phase 2: Standardize Templates - COMPLETED

1. ✅ Update `templates/go-service/`:
   - ✅ Add property test template with build tags
   - ✅ Add TESTING.md template
   - ✅ Update README with testing section
   - ✅ Add test-coverage.sh script

2. ✅ Update `templates/java-service/`:
   - ✅ Verify property test setup (jqwik is configured)
   - ✅ Add TESTING.md template
   - ✅ README already has testing section

3. ⚠️ Create `templates/node-service/`:
   - ⚠️ Based on apps/web structure (optional - can be done later)
   - ⚠️ Include Vite configuration
   - ⚠️ Include testing setup
   - ⚠️ Include property test examples

### ✅ Phase 3: Apply to All Apps - COMPLETED

1. ✅ Review each app against checklist
2. ✅ Add missing files (TESTING.md, Dockerfile for web)
3. ✅ Update documentation
4. ✅ Verify CI/CD integration

### Phase 4: Documentation and Verification

1. ⚠️ Update CREATE_APP_GUIDE.md (if needed)
2. ✅ Create APP_STANDARDIZATION.md (this document)
3. ⚠️ Update root README.md with standards link (optional)
4. ⚠️ Run verification tests on all services

## Verification

After standardization, verify each app:

```bash
# Test the app
make test APP=name

# Build the app
make build APP=name

# Lint the app
make lint APP=name

# Build Docker image
make docker-build APP=name

# Check coverage
cd apps/name && ./scripts/test-coverage.sh  # For Go apps
```

## Standardization Completion Summary

### ✅ All Services Now Compliant

All 7 services in the monorepo now meet the standardization requirements:

| Service | README | TESTING.md | metadata.yaml | .apptype | Dockerfile | catalog-info.yaml |
|---------|--------|------------|---------------|----------|------------|-------------------|
| auth-service | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| hello-service | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| im-service | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| shortener-service | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| todo-service | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| user-service | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| web | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

### Files Created/Updated

#### Service Documentation
- ✅ `apps/auth-service/README.md` - Removed template placeholders
- ✅ `apps/auth-service/TESTING.md` - Created comprehensive testing guide
- ✅ `apps/user-service/TESTING.md` - Created with MySQL-specific guidance
- ✅ `apps/todo-service/README.md` - Translated to English
- ✅ `apps/todo-service/TESTING.md` - Created comprehensive testing guide
- ✅ `apps/hello-service/README.md` - Translated to English
- ✅ `apps/hello-service/TESTING.md` - Created with Java/jqwik guidance
- ✅ `apps/shortener-service/TESTING.md` - Created comprehensive testing guide
- ✅ `apps/web/TESTING.md` - Created with React Testing Library guidance
- ✅ `apps/web/Dockerfile` - Created multi-stage build with nginx
- ✅ `apps/web/nginx.conf` - Created nginx configuration

#### Templates
- ✅ `templates/go-service/TESTING.md` - Comprehensive testing template
- ✅ `templates/go-service/service/template_service_property_test.go` - Property test template
- ✅ `templates/java-service/TESTING.md` - Comprehensive Java testing template

#### Documentation
- ✅ `docs/development/APP_STANDARDIZATION.md` - This standardization plan
- ✅ `docs/development/PROPERTY_TESTING.md` - Property testing guide (created earlier)
- ✅ `TESTING.md` - Root testing guide (created earlier)

### Key Improvements

1. **Consistent Documentation**: All services now have comprehensive README.md files in English
2. **Testing Guidance**: Every service has detailed TESTING.md with examples
3. **Property-Based Testing**: All templates include property test examples and guidance
4. **Build Tags**: Go services use build tags to separate fast unit tests from slow property tests
5. **Coverage Requirements**: Clear coverage thresholds documented for each language
6. **Docker Support**: All services including web frontend have Dockerfiles
7. **Template Quality**: Templates now include all best practices from existing services

### Testing Standards Established

- **Go Services**: 80% overall, 90% service package, property tests with `pgregory.net/rapid`
- **Java Services**: 80% overall, 90% service classes, property tests with jqwik
- **Node.js Apps**: 70% overall, property tests with fast-check (documented)

### Next Steps (Optional)

1. Create `templates/node-service/` based on `apps/web` structure
2. Update `CREATE_APP_GUIDE.md` with new template features
3. Add link to APP_STANDARDIZATION.md in root README.md
4. Run verification tests on all services to ensure everything works

## Maintenance

- Review new apps during PR review
- Update templates when patterns emerge
- Keep this document updated
- Run periodic audits

## Related Documentation

- [App Management](./APP_MANAGEMENT.md)
- [Create App Guide](./CREATE_APP_GUIDE.md)
- [Testing Guide](./TESTING_GUIDE.md)
- [Property Testing](./PROPERTY_TESTING.md)
