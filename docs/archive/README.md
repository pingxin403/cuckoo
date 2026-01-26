# Documentation Archive

This directory contains historical documentation that is no longer actively maintained but preserved for reference.

## What's Archived Here

### Migrations (`migrations/`)
Documentation related to past migration efforts:
- Deployment structure migrations
- Metadata system migrations
- Infrastructure migrations (Helm, etcd, etc.)
- Proto generation strategy migrations

These documents capture the reasoning and process behind major architectural changes.

### Completions (`completions/`)
Task completion summaries and milestone documents:
- Feature implementation summaries
- Phase completion reports
- Architecture improvement summaries
- Deployment refactoring completions

These documents provide historical context for completed work.

### Fixes (`fixes/`)
Bug fix and issue resolution documentation:
- CI/CD pipeline fixes
- Test infrastructure fixes
- Linting and code quality fixes
- Security and configuration fixes

These documents explain how specific issues were resolved.

### App-Specific (`app-specific/`)
Historical documents specific to individual applications:
- Gateway setup summaries
- Integration test fixes
- MVP completion reports
- Task status summaries

These documents capture app-specific milestones and fixes.

### Other Historical Documents
- `ARCHITECTURE_SCALABILITY_ANALYSIS.md` - Early scalability analysis
- `CHECKLIST.md` - Old project checklist
- `CONFIG_DOCUMENTATION_CLEANUP.md` - Configuration documentation cleanup report
- `DOCUMENTATION_CLEANUP_ROUND3.md` - Third round documentation cleanup report
- `LOCAL_SETUP_VERIFICATION.md` - Initial setup verification
- `MAKEFILE_PROTO_OPTIMIZATION.md` - Makefile optimization history
- `PRE_COMMIT_HOOK_IMPROVEMENTS.md` - Pre-commit hook evolution
- `PROTO_GENERATION_STRATEGY.md` - Proto generation strategy history
- `PROTO_HYBRID_STRATEGY.md` - Proto hybrid approach documentation
- `PROTO_TOOLS_VERSION.md` - Proto tools version requirements

### Proposals (`proposals/`)
Archived proposals that have been implemented:
- `OBSERVABILITY_LIBRARY_PROPOSAL.md` - Observability library design (implemented in `libs/observability/`)

### Configuration (`CONFIG_MIGRATION_GUIDE.md`)
Configuration migration guide - all services have been migrated to the unified configuration library

## Why Archive?

These documents are archived rather than deleted because they:

1. **Provide Historical Context**: Explain why certain decisions were made
2. **Document Evolution**: Show how the project evolved over time
3. **Preserve Knowledge**: Capture lessons learned and problem-solving approaches
4. **Reference Material**: May be useful when facing similar issues in the future

## When to Use Archived Docs

Refer to archived documentation when:

- Understanding the history of a particular feature or decision
- Investigating why something was implemented a certain way
- Learning from past migration experiences
- Troubleshooting similar issues that were previously resolved

## Active Documentation

For current, actively maintained documentation, see:

- [Documentation Index](../README.md) - Main documentation hub
- [Architecture](../architecture/) - Current system architecture
- [Development](../development/) - Development guides and practices
- [Deployment](../deployment/) - Deployment procedures
- [CI/CD](../ci-cd/) - CI/CD pipeline documentation
- [Process](../process/) - Team processes and governance

## Archive Organization

```
archive/
├── README.md                    # This file
├── migrations/                  # Migration documentation
│   ├── DEPLOY_MIGRATION.md
│   ├── K8S_INFRA_HELM_MIGRATION.md
│   ├── METADATA_MIGRATION.md
│   ├── METADATA_MIGRATION_SUMMARY.md
│   └── MIGRATION_TO_UNIFIED_PROTO.md
├── completions/                 # Completion summaries
│   ├── APP_MANAGEMENT_SUMMARY.md
│   ├── ARCHITECTURE_IMPROVEMENTS_SUMMARY.md
│   ├── DEPLOYMENT_REFACTORING_PHASE1_COMPLETE.md
│   ├── DEPLOYMENT_SUMMARY.md
│   ├── IM_TASK_1_COMPLETION.md
│   ├── INIT_SUMMARY.md
│   ├── K8S_CLEANUP_AND_SHORTENER_ADDITION.md
│   ├── MAKEFILE_AND_K8S_OPTIMIZATION_COMPLETE.md
│   ├── MAKEFILE_COMPLETE_OPTIMIZATION.md
│   ├── MAKEFILE_OPTIMIZATION_SUMMARY.md
│   ├── TASK_6_COMPLETION_SUMMARY.md
│   ├── TASK_6_DOCKER_SIMPLIFICATION_COMPLETE.md
│   └── WEB_SERVICE_AND_HIGRESS_COMPLETE.md
├── fixes/                       # Fix documentation
│   ├── CI_COVERAGE_FIX.md
│   ├── CI_FIX_COMPLETE_SUMMARY.md
│   ├── CI_FIX_QUICK_REFERENCE.md
│   ├── CI_FIX_SUMMARY.md
│   ├── CI_ISSUES_SUMMARY.md
│   ├── CI_SECURITY_K8S_FIX.md
│   ├── CI_SHORTENER_FIX.md
│   ├── LINT_FIX_GUIDE.md
│   ├── LINT_FIX_SUMMARY.md
│   └── TEST_FIX_SUMMARY.md
├── app-specific/                # App-specific historical docs
│   ├── GATEWAY_SETUP_SUMMARY.md
│   ├── INTEGRATION_TEST_FIX.md
│   ├── INTEGRATION_TEST_SUMMARY.md
│   ├── MVP_COMPLETION_SUMMARY.md
│   └── TASK_STATUS_SUMMARY.md
├── proposals/                   # Implemented proposals
│   └── OBSERVABILITY_LIBRARY_PROPOSAL.md
├── CONFIG_DOCUMENTATION_CLEANUP.md  # Configuration cleanup report
├── CONFIG_MIGRATION_GUIDE.md    # Configuration migration guide (all services migrated)
├── DOCUMENTATION_CLEANUP_ROUND3.md  # Third round cleanup report
└── [other historical files]     # Miscellaneous historical docs
```

## Maintenance

This archive is **read-only**. Documents here should not be updated unless:

- Correcting factual errors
- Adding clarifying notes
- Improving formatting for readability

For new documentation, always create files in the appropriate active directory, not in the archive.

---

**Last Updated**: 2026-01-26  
**Archive Created**: 2026-01-22
