# Property-Based Testing with Build Tags

## Problem

Property-based tests are extremely valuable but very slow:
- TTL tests wait for actual timeouts (1-2 seconds per iteration)
- Each property runs 100+ iterations
- Total runtime: 5-10 minutes per service

This makes `make test` too slow for rapid development.

## Solution: Go Build Tags

We use Go build tags to separate fast unit tests from slow property tests:

```go
//go:build property
// +build property

package mypackage

func TestProperty_MyInvariant(t *testing.T) {
    // This test only runs with -tags=property
}
```

## Usage

### Fast Development (Default)
```bash
# Runs only unit tests (~2 seconds)
make test APP=im
go test ./...
```

### Full Validation (On-Demand)
```bash
# Runs ALL tests including property tests (~10 minutes)
go test ./... -tags=property -timeout=30m
```

## How It Works

### Without Build Tag (Default)
```bash
$ go test ./...
# Compiles: *_test.go (unit tests)
# Skips: *_property_test.go (property tests)
# Runtime: ~2 seconds ✅
```

### With Build Tag
```bash
$ go test ./... -tags=property
# Compiles: *_test.go + *_property_test.go
# Runtime: ~10 minutes ⏱️
```

## Adding Build Tags to New Files

### Automatic (Recommended)
```bash
./scripts/add-property-tags.sh
```

### Manual
Add these lines at the top of `*_property_test.go` files:

```go
//go:build property
// +build property

package mypackage
```

**Important**: 
- Must be the **first lines** in the file
- Blank line required after build tags
- Both formats required for compatibility

## CI/CD Strategy

### Pull Request (Fast)
```yaml
- name: Run tests
  run: make test  # Unit tests only, ~30 seconds
```

### Nightly Build (Comprehensive)
```yaml
- name: Run all tests
  run: go test ./... -tags=property -timeout=1h
```

### Pre-Release (Full)
```yaml
- name: Full test suite
  run: |
    go test ./... -tags=property -timeout=2h
    make test-coverage
```

## Benefits

✅ **Fast feedback loop**: Unit tests run in seconds  
✅ **Comprehensive validation**: Property tests catch edge cases  
✅ **Flexible**: Choose speed vs thoroughness  
✅ **Standard Go practice**: Uses built-in build tag system  
✅ **No test skipping**: Tests are properly excluded, not skipped  

## Affected Services

All Go services with property-based tests:
- `apps/auth-service`
- `apps/user-service`
- `apps/im-service`
- `apps/todo-service`
- `apps/shortener-service`

## Test File Naming Convention

- `*_test.go` - Unit tests (always run)
- `*_property_test.go` - Property tests (require `-tags=property`)

## Verification

### Check if property tests are excluded:
```bash
cd apps/im-service
go test ./... -v | grep -c "TestProperty"
# Should output: 0
```

### Check if property tests run with tag:
```bash
cd apps/im-service
go test ./... -tags=property -v | grep -c "TestProperty"
# Should output: 8 (or number of property tests)
```

## Troubleshooting

### Tests still slow?
- Check if property test files have build tags
- Run: `./scripts/add-property-tags.sh`

### Property tests not running?
- Make sure you include `-tags=property`
- Verify build tags are at the top of the file

### Build tag format errors?
```go
// ❌ Wrong - missing blank line
//go:build property
package mypackage

// ✅ Correct - has blank line
//go:build property
// +build property

package mypackage
```

## References

- [Go Build Constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [Property-Based Testing](https://pkg.go.dev/pgregory.net/rapid)
- [Testing Best Practices](/TESTING.md)
