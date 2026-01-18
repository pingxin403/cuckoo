# Protobuf Tools Version Requirements

This document specifies the required versions of protobuf tools to ensure consistency between local development and CI environments.

## Version Management

All tool versions are centrally managed in the `.tool-versions` file at the project root. This ensures:
- ✅ Consistent versions across all environments
- ✅ Single source of truth for version requirements
- ✅ Easy version updates (change in one place)
- ✅ Automatic version checking via `make check-versions`

## Required Versions

Current versions are defined in `.tool-versions`:

```bash
# View current versions
cat .tool-versions
```

### Quick Setup

```bash
# Initialize environment with correct versions
make init

# Verify your tool versions
make check-versions
```

## Manual Installation

If `make init` doesn't work for your environment, install tools manually:

### Protobuf Compiler (protoc)

Check `.tool-versions` for the required version, then:

**macOS**:
```bash
# Install specific version
PROTOC_VERSION=33.1  # Check .tool-versions for current version
PROTOC_ZIP=protoc-${PROTOC_VERSION}-osx-x86_64.zip
curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}
sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
sudo unzip -o $PROTOC_ZIP -d /usr/local 'include/*'
rm -f $PROTOC_ZIP
```

**Linux**:
```bash
# Install specific version
PROTOC_VERSION=33.1  # Check .tool-versions for current version
PROTOC_ZIP=protoc-${PROTOC_VERSION}-linux-x86_64.zip
curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}
sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
sudo unzip -o $PROTOC_ZIP -d /usr/local 'include/*'
rm -f $PROTOC_ZIP
```

### Go Plugins

```bash
# Load versions from .tool-versions
source .tool-versions

# Install specific versions
go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}
```

### TypeScript Plugin
- Managed by npm in `apps/web/package.json`
- Version is locked in `apps/web/package-lock.json`
- Install with: `cd apps/web && npm ci`

## Verification

Check your installed versions:

```bash
# Automated check
make check-versions

# Manual check
protoc --version
protoc-gen-go --version
protoc-gen-go-grpc --version
go version
node --version
java -version
```

## Why Version Consistency Matters

Different versions of protobuf tools can generate slightly different code:
- Different comment formatting
- Different function signatures  
- Different optimization levels
- Different error messages

To ensure that generated code is consistent across all environments (local development, CI, and other developers' machines), we must use the exact same tool versions.

## Updating Versions

If you need to update the protobuf tool versions:

1. **Update `.tool-versions`** with new version numbers
2. **Update your local tools**:
   ```bash
   make init  # Reinstall with new versions
   ```
3. **Regenerate code**:
   ```bash
   make proto
   ```
4. **Verify changes**:
   ```bash
   make check-versions
   git diff  # Review generated code changes
   ```
5. **Commit everything**:
   ```bash
   git add .tool-versions apps/*/gen/ apps/*/src/gen/
   git commit -m "chore: update protobuf tools to vX.Y.Z"
   ```
6. **Update this documentation** if needed

## CI Configuration

The CI workflow (`.github/workflows/ci.yml`) automatically reads versions from `.tool-versions` in the `verify-proto` job. This ensures that:
- Generated code in git matches what CI expects
- CI will fail if someone commits code generated with different tool versions
- No manual version synchronization needed between CI and `.tool-versions`

## Troubleshooting

### "Generated code is out of date" error in CI

This error means the generated code in git was created with different tool versions than specified in `.tool-versions`.

**Solution**:
```bash
# Install correct versions
make init

# Regenerate code
make proto

# Verify versions match
make check-versions

# Commit updated code
git add apps/*/gen/ apps/*/src/gen/
git commit -m "fix: regenerate proto code with correct tool versions"
```

### Different developers have different generated code

This happens when developers use different tool versions.

**Solution**:
```bash
# Everyone should run:
make init           # Install correct versions
make check-versions # Verify versions
make proto          # Regenerate code

# The last person to commit wins
```

### Version check fails

If `make check-versions` reports version mismatches:

1. **Critical tools** (protoc, protoc-gen-go, protoc-gen-go-grpc): Must match exactly
   ```bash
   make init  # Reinstall correct versions
   ```

2. **Runtime tools** (Go, Node, Java): Must meet minimum version
   - Update your runtime if below minimum
   - Warnings are acceptable if above minimum

## See Also

- [Getting Started Guide](GETTING_STARTED.md) - Full setup instructions
- [CI/CD Pipeline](.github/workflows/ci.yml) - CI configuration
- `.tool-versions` - Version definitions
