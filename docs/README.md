# Documentation Index

Welcome to the Monorepo documentation! This index helps you find the information you need quickly.

## 🚀 Getting Started

- [Getting Started Guide](GETTING_STARTED.md) - Quick start guide for new developers
- [Quick Reference](QUICK_REFERENCE.md) - Common commands and workflows

## 🏗️ Architecture

- [System Architecture](architecture/ARCHITECTURE.md) - Overall system design and components
- [Infrastructure Overview](architecture/INFRASTRUCTURE.md) - Infrastructure components and setup
- [Higress Routing Configuration](architecture/HIGRESS_ROUTING_CONFIGURATION.md) - API gateway routing

## 💻 Development

- [Code Quality Standards](development/CODE_QUALITY.md) - Coding standards and best practices
- [Testing Guide](development/TESTING_GUIDE.md) - How to write and run tests
- [Linting Guide](development/LINTING_GUIDE.md) - Linting rules and configuration
- [Create App Guide](development/CREATE_APP_GUIDE.md) - How to create new applications
- [Service Creation Automation](development/SERVICE_CREATION_AUTOMATION.md) - Automated service creation details
- [App Management](development/APP_MANAGEMENT.md) - Managing applications in the monorepo
- [Makefile Guide](development/MAKEFILE_GUIDE.md) - Using the Makefile for builds and tasks

## 🚢 Deployment

- [Deployment Guide](deployment/DEPLOYMENT_GUIDE.md) - Complete deployment guide
- [Deployment Quick Reference](deployment/DEPLOYMENT_QUICK_REFERENCE.md) - Quick deployment commands
- [Production Operations](deployment/PRODUCTION_OPERATIONS.md) - Operating in production

## 🔄 CI/CD

- [Dynamic CI Strategy](ci-cd/DYNAMIC_CI_STRATEGY.md) - CI/CD pipeline design
- [Integration Tests](ci-cd/INTEGRATION_TESTS_IMPLEMENTATION.md) - Integration testing approach
- [Coverage Quick Reference](ci-cd/COVERAGE_QUICK_REFERENCE.md) - Code coverage guidelines

## 📋 Process & Governance

- [Governance Model](process/governance.md) - Project governance and decision-making
- [Communication Guidelines](process/COMMUNICATION.md) - How we communicate
- [Shift-Left Practices](process/SHIFT_LEFT.md) - Quality and security practices

## 📚 OpenSpec

- [App Management System](openspec/openspec-app-management-system.md)
- [Integration Testing](openspec/openspec-integration-testing.md)
- [Monorepo Architecture](openspec/openspec-monorepo-architecture.md)
- [Quality Practices](openspec/openspec-quality-practices.md)

## 📦 Archive

Historical documents and migration guides are archived in the [archive](archive/) directory:

- [Migrations](archive/migrations/) - Migration guides and summaries
- [Completions](archive/completions/) - Task completion summaries
- [Fixes](archive/fixes/) - Bug fix and issue resolution documents
- [App-Specific](archive/app-specific/) - App-specific historical documents

## 🔍 Quick Links

### For New Developers
1. [Getting Started](GETTING_STARTED.md)
2. [Code Quality](development/CODE_QUALITY.md)
3. [Testing Guide](development/TESTING_GUIDE.md)
4. [Quick Reference](QUICK_REFERENCE.md)

### For DevOps
1. [Deployment Guide](deployment/DEPLOYMENT_GUIDE.md)
2. [Infrastructure Overview](architecture/INFRASTRUCTURE.md)
3. [Production Operations](deployment/PRODUCTION_OPERATIONS.md)
4. [Higress Routing](architecture/HIGRESS_ROUTING_CONFIGURATION.md)

### For Architects
1. [System Architecture](architecture/ARCHITECTURE.md)
2. [Infrastructure Overview](architecture/INFRASTRUCTURE.md)
3. [OpenSpec Documentation](openspec/)

## 📝 Documentation Standards

When creating or updating documentation:

1. **Use clear, concise language**
2. **Include code examples** where appropriate
3. **Keep it up-to-date** - update docs when code changes
4. **Link to related docs** for context
5. **Use proper markdown formatting**

## 🤝 Contributing

To contribute to documentation:

1. Follow the existing structure
2. Place new docs in the appropriate directory
3. Update this index when adding new documents
4. Use meaningful file names (kebab-case)
5. Include a clear title and overview

## 📞 Getting Help

If you can't find what you're looking for:

1. Check the [Quick Reference](QUICK_REFERENCE.md)
2. Search the documentation (use your IDE's search)
3. Ask in the team chat
4. Create an issue for missing documentation

## 🗂️ Directory Structure

```
docs/
├── README.md                    # This file
├── GETTING_STARTED.md           # Getting started guide
├── QUICK_REFERENCE.md           # Quick reference
│
├── architecture/                # Architecture documentation
│   ├── ARCHITECTURE.md
│   ├── INFRASTRUCTURE.md
│   └── HIGRESS_ROUTING_CONFIGURATION.md
│
├── development/                 # Development guides
│   ├── CODE_QUALITY.md
│   ├── TESTING_GUIDE.md
│   ├── LINTING_GUIDE.md
│   ├── CREATE_APP_GUIDE.md
│   ├── SERVICE_CREATION_AUTOMATION.md
│   ├── APP_MANAGEMENT.md
│   └── MAKEFILE_GUIDE.md
│
├── deployment/                  # Deployment documentation
│   ├── DEPLOYMENT_GUIDE.md
│   ├── DEPLOYMENT_QUICK_REFERENCE.md
│   └── PRODUCTION_OPERATIONS.md
│
├── ci-cd/                       # CI/CD documentation
│   ├── DYNAMIC_CI_STRATEGY.md
│   ├── INTEGRATION_TESTS_IMPLEMENTATION.md
│   └── COVERAGE_QUICK_REFERENCE.md
│
├── process/                     # Process and governance
│   ├── governance.md
│   ├── COMMUNICATION.md
│   └── SHIFT_LEFT.md
│
├── openspec/                    # OpenSpec documentation
│   ├── openspec-app-management-system.md
│   ├── openspec-integration-testing.md
│   ├── openspec-monorepo-architecture.md
│   └── openspec-quality-practices.md
│
└── archive/                     # Historical documents
    ├── migrations/
    ├── completions/
    ├── fixes/
    └── app-specific/
```

## 📊 Documentation Metrics

- **Active Documents**: 25
- **Archived Documents**: 30+
- **Last Major Cleanup**: 2026-01-22
- **Organization**: By topic and purpose

---

**Last Updated**: 2026-01-22  
**Maintained By**: Platform Team
