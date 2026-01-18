# Protobuf Tools Version Requirements

This document specifies the required versions of protobuf tools to ensure consistency between local development and CI environments.

## Required Versions

### Protobuf Compiler (protoc)
- **Version**: 28.3 (libprotoc 33.1)
- **Download**: https://github.com/protocolbuffers/protobuf/releases/tag/v28.3

### Go Plugins
- **protoc-gen-go**: v1.36.6
  ```bash
  go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
  ```

- **protoc-gen-go-grpc**: v1.5.1
  ```bash
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
  ```

### TypeScript Plugin
- **protoc-gen-ts_proto**: Managed by npm in `apps/web/package.json`
- Version is locked in `apps/web/package-lock.json`

## Verification

Check your installed versions:

```bash
# Check protoc version
protoc --version
# Expected output: libprotoc 33.1

# Check Go plugin versions
protoc-gen-go --version
# Expected output: protoc-gen-go v1.36.6

protoc-gen-go-grpc --version
# Expected output: protoc-gen-go-grpc 1.5.1
```

## Why Version Consistency Matters

Different versions of protobuf tools can generate slightly different code:
- Different comment formatting
- Different function signatures
- Different optimization levels

To ensure that generated code is consistent across all environments (local development, CI, and other developers' machines), we must use the exact same tool versions.

## Updating Versions

If you need to update the protobuf tool versions:

1. Update your local tools to the new versions
2. Update the versions in `.github/workflows/ci.yml` (verify-proto job)
3. Run `make proto` to regenerate all code
4. Commit the updated generated code
5. Update this document with the new version numbers

## CI Configuration

The CI workflow (`.github/workflows/ci.yml`) is configured to use these exact versions in the `verify-proto` job. This ensures that:
- Generated code in git matches what CI expects
- CI will fail if someone commits code generated with different tool versions
- All developers are encouraged to use the same tool versions

## Troubleshooting

### "Generated code is out of date" error in CI

This error means the generated code in git was created with different tool versions than CI uses.

**Solution**:
1. Install the correct tool versions (see above)
2. Run `make proto` to regenerate code
3. Commit the updated generated code

### Different developers have different generated code

This happens when developers use different tool versions.

**Solution**:
- All developers should install the exact versions specified in this document
- Run `make proto` to regenerate code with correct versions
- The last person to commit wins - their generated code becomes the standard
