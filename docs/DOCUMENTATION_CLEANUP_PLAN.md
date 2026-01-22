# Documentation Cleanup Plan

## Overview
This document outlines the plan to clean up, consolidate, and organize documentation across the monorepo.

## Current State Analysis

### Total Documents
- **docs/**: 62 files
- **Other locations**: ~20 files
- **Total**: ~82 documentation files

### Issues Identified
1. **Redundancy**: Multiple SUMMARY/COMPLETE files covering similar topics
2. **Outdated**: Migration and fix documents that are no longer relevant
3. **Scattered**: Documentation spread across multiple locations
4. **Inconsistent**: No clear organization or naming convention

## Cleanup Strategy

### 1. Core Documentation (KEEP)
Essential reference documents that should be maintained:

**Architecture & Design**:
- `docs/ARCHITECTURE.md` - System architecture overview
- `docs/GETTING_STARTED.md` - Getting started guide
- `docs/QUICK_REFERENCE.md` - Quick command reference

**Development Guides**:
- `docs/CODE_QUALITY.md` - Code quality standards
- `docs/TESTING_GUIDE.md` - Testing guidelines
- `docs/LINTING_GUIDE.md` - Linting standards
- `docs/CREATE_APP_GUIDE.md` - How to create new apps
- `docs/APP_MANAGEMENT.md` - App management guide

**Deployment**:
- `docs/DEPLOYMENT_QUICK_REFERENCE.md` - Quick deployment commands
- `docs/HIGRESS_ROUTING_CONFIGURATION.md` - Higress routing guide
- `docs/INFRASTRUCTURE.md` - Infrastructure overview
- `docs/PRODUCTION_OPERATIONS.md` - Production operations

**Process & Governance**:
- `docs/governance.md` - Governance model
- `docs/COMMUNICATION.md` - Communication guidelines
- `docs/SHIFT_LEFT.md` - Shift-left practices

**OpenSpec**:
- `docs/openspec-*.md` - OpenSpec documentation (4 files)

### 2. Consolidate (MERGE)
Multiple documents covering similar topics should be merged:

#### A. Deployment Documentation
**Merge into**: `docs/DEPLOYMENT_GUIDE.md` (new consolidated guide)
**Source files** (DELETE after merge):
- `docs/DOCKER_DEPLOYMENT.md`
- `docs/KUBERNETES_DEPLOYMENT.md`
- `docs/DEPLOYMENT_SUMMARY.md`
- `docs/DOCKER_COMPOSE_SIMPLIFICATION.md`
- `deploy/DEPLOYMENT_GUIDE.md` (move to docs/)

#### B. CI/CD Documentation
**Merge into**: `docs/CI_CD_GUIDE.md` (new consolidated guide)
**Source files** (DELETE after merge):
- `docs/DYNAMIC_CI_STRATEGY.md`
- `docs/INTEGRATION_TESTS_IMPLEMENTATION.md`
- `docs/COVERAGE_QUICK_REFERENCE.md`
- `docs/COVERAGE_STRATEGY_SUMMARY.md`

#### C. Makefile Documentation
**Merge into**: `docs/MAKEFILE_GUIDE.md` (new consolidated guide)
**Source files** (DELETE after merge):
- `docs/MAKEFILE_OPTIMIZATION_SUMMARY.md`
- `docs/MAKEFILE_PROTO_OPTIMIZATION.md`
- `docs/PROTO_GENERATION_STRATEGY.md`
- `docs/PROTO_HYBRID_STRATEGY.md`
- `docs/PROTO_TOOLS_VERSION.md`

### 3. Archive (MOVE to docs/archive/)
Historical documents that may be useful for reference but are no longer active:

**Migration Documents**:
- `docs/DEPLOY_MIGRATION.md`
- `docs/METADATA_MIGRATION.md`
- `docs/METADATA_MIGRATION_SUMMARY.md`
- `docs/MIGRATION_TO_UNIFIED_PROTO.md`
- `docs/K8S_INFRA_HELM_MIGRATION.md`

**Completion/Summary Documents**:
- `docs/DEPLOYMENT_REFACTORING_PHASE1_COMPLETE.md`
- `docs/TASK_6_COMPLETION_SUMMARY.md`
- `docs/TASK_6_DOCKER_SIMPLIFICATION_COMPLETE.md`
- `docs/MAKEFILE_AND_K8S_OPTIMIZATION_COMPLETE.md`
- `docs/MAKEFILE_COMPLETE_OPTIMIZATION.md`
- `docs/K8S_CLEANUP_AND_SHORTENER_ADDITION.md`
- `docs/WEB_SERVICE_AND_HIGRESS_COMPLETE.md`
- `docs/APP_MANAGEMENT_SUMMARY.md`
- `docs/ARCHITECTURE_IMPROVEMENTS_SUMMARY.md`
- `docs/IM_TASK_1_COMPLETION.md`

**Fix Documents**:
- `docs/CI_COVERAGE_FIX.md`
- `docs/CI_FIX_COMPLETE_SUMMARY.md`
- `docs/CI_FIX_QUICK_REFERENCE.md`
- `docs/CI_FIX_SUMMARY.md`
- `docs/CI_ISSUES_SUMMARY.md`
- `docs/CI_SECURITY_K8S_FIX.md`
- `docs/CI_SHORTENER_FIX.md`
- `docs/LINT_FIX_GUIDE.md`
- `docs/LINT_FIX_SUMMARY.md`
- `docs/TEST_FIX_SUMMARY.md`

**Other Historical**:
- `docs/INIT_SUMMARY.md`
- `docs/LOCAL_SETUP_VERIFICATION.md`
- `docs/PRE_COMMIT_HOOK_IMPROVEMENTS.md`
- `docs/SHORT_NAMES_REFERENCE.md` (merge into QUICK_REFERENCE.md)
- `docs/CHECKLIST.md` (outdated)
- `docs/ARCHITECTURE_SCALABILITY_ANALYSIS.md` (merge into ARCHITECTURE.md)

### 4. Delete (REMOVE completely)
Temporary or redundant documents with no historical value:

**App-Specific Summaries** (move to app directories if needed):
- `apps/shortener-service/GATEWAY_SETUP_SUMMARY.md`
- `apps/shortener-service/INTEGRATION_TEST_SUMMARY.md`
- `apps/shortener-service/MVP_COMPLETION_SUMMARY.md`
- `apps/shortener-service/TASK_STATUS_SUMMARY.md`
- `apps/hello-service/INTEGRATION_TEST_FIX.md`

**Redundant**:
- `deploy/REFACTORING_SUMMARY.md` (covered in other docs)
- `deploy/k8s/infra/etcd/etcd-README.md` (deprecated, already marked)

## New Documentation Structure

```
docs/
├── README.md                          # Documentation index
├── GETTING_STARTED.md                 # Quick start guide
├── QUICK_REFERENCE.md                 # Command quick reference
│
├── architecture/
│   ├── ARCHITECTURE.md                # System architecture
│   ├── INFRASTRUCTURE.md              # Infrastructure overview
│   └── HIGRESS_ROUTING_CONFIGURATION.md
│
├── development/
│   ├── CODE_QUALITY.md                # Code standards
│   ├── TESTING_GUIDE.md               # Testing guidelines
│   ├── LINTING_GUIDE.md               # Linting standards
│   ├── CREATE_APP_GUIDE.md            # Creating new apps
│   ├── APP_MANAGEMENT.md              # Managing apps
│   └── MAKEFILE_GUIDE.md              # Makefile usage (NEW)
│
├── deployment/
│   ├── DEPLOYMENT_GUIDE.md            # Complete deployment guide (NEW)
│   ├── DEPLOYMENT_QUICK_REFERENCE.md  # Quick commands
│   └── PRODUCTION_OPERATIONS.md       # Production ops
│
├── ci-cd/
│   └── CI_CD_GUIDE.md                 # CI/CD guide (NEW)
│
├── process/
│   ├── governance.md                  # Governance
│   ├── COMMUNICATION.md               # Communication
│   └── SHIFT_LEFT.md                  # Shift-left practices
│
├── openspec/
│   ├── openspec-app-management-system.md
│   ├── openspec-integration-testing.md
│   ├── openspec-monorepo-architecture.md
│   └── openspec-quality-practices.md
│
└── archive/                           # Historical documents
    ├── migrations/
    ├── completions/
    └── fixes/
```

## Implementation Steps

### Phase 1: Create Archive Directory
```bash
mkdir -p docs/archive/{migrations,completions,fixes}
```

### Phase 2: Move Historical Documents
Move migration, completion, and fix documents to archive.

### Phase 3: Create Consolidated Guides
1. Create `docs/DEPLOYMENT_GUIDE.md`
2. Create `docs/CI_CD_GUIDE.md`
3. Create `docs/MAKEFILE_GUIDE.md`

### Phase 4: Delete Redundant Documents
Remove temporary and redundant files.

### Phase 5: Reorganize Structure
Create subdirectories and move files to new structure.

### Phase 6: Update References
Update all references to moved/deleted documents in:
- README.md
- Other documentation files
- Code comments
- CI/CD configurations

### Phase 7: Create Documentation Index
Create `docs/README.md` as the main documentation index.

## Benefits

1. **Clarity**: Clear organization by topic
2. **Maintainability**: Easier to find and update documentation
3. **Reduced Redundancy**: Single source of truth for each topic
4. **Better Onboarding**: New developers can find information quickly
5. **Historical Context**: Archive preserves history without cluttering main docs

## Metrics

### Before Cleanup
- Total files: ~82
- Redundant files: ~30
- Outdated files: ~25
- Well-organized: ~27

### After Cleanup (Target)
- Active files: ~25
- Archived files: ~30
- Deleted files: ~27
- Organization: 100%

## Timeline

- **Phase 1-2**: 30 minutes (archive setup and moves)
- **Phase 3**: 2 hours (create consolidated guides)
- **Phase 4**: 30 minutes (delete redundant files)
- **Phase 5**: 1 hour (reorganize structure)
- **Phase 6**: 1 hour (update references)
- **Phase 7**: 30 minutes (create index)

**Total**: ~5.5 hours

## Approval Required

This cleanup plan should be reviewed and approved before execution to ensure:
1. No critical information is lost
2. Historical context is preserved appropriately
3. Team members are aware of the changes
4. References are properly updated

## Next Steps

1. Review this plan
2. Get team approval
3. Execute phases sequentially
4. Verify all references are updated
5. Communicate changes to team
