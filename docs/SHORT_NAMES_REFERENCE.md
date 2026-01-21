# Short Names Quick Reference

## Available Services and Short Names

| Full Name | Short Name | Type | Port | Command Example |
|-----------|------------|------|------|-----------------|
| hello-service | `hello` | java | 9090 | `make test APP=hello` |
| todo-service | `todo` | go | 9091 | `make test APP=todo` |
| shortener-service | `shortener` | go | 9092 | `make test APP=shortener` |
| web | `web` | node | 5173 | `make test APP=web` |

## Common Commands with Short Names

### Testing
```bash
make test APP=shortener    # Test shortener service
make test APP=hello        # Test hello service
make test APP=todo         # Test todo service
make test APP=web          # Test web frontend
```

### Building
```bash
make build APP=shortener   # Build shortener service
make build APP=hello       # Build hello service
make build APP=todo        # Build todo service
```

### Linting
```bash
make lint APP=shortener    # Lint shortener service
make lint APP=hello        # Lint hello service
make lint-fix APP=todo     # Auto-fix lint issues in todo service
```

### Formatting
```bash
make format APP=shortener  # Format shortener service code
make format APP=hello      # Format hello service code
```

### Docker
```bash
make docker-build APP=shortener  # Build Docker image for shortener
make docker-build APP=hello      # Build Docker image for hello
```

### Running Locally
```bash
make run APP=shortener     # Run shortener service locally
make run APP=hello         # Run hello service locally
make run APP=todo          # Run todo service locally
```

## Tips

1. **Use short names for convenience**: `APP=shortener` instead of `APP=shortener-service`
2. **Full names still work**: Both `APP=shortener` and `APP=shortener-service` are valid
3. **List all apps**: Run `make list-apps` to see all available services
4. **Auto-detection**: Omit `APP=` to operate on changed apps only

## Adding Short Names to New Services

When creating a new service, the short name is automatically generated:

```bash
./scripts/create-app.sh go my-awesome-service --port 9093
# Automatically creates short_name: my-awesome
```

Or manually add to `metadata.yaml`:

```yaml
spec:
  name: my-service
  short_name: myservice  # Add this line
  type: go
  port: 9093
```

## Troubleshooting

### "Unknown app" error
```bash
[ERROR] Unknown app: myapp
[INFO] Available apps: hello-service shortener-service todo-service web
[INFO] Short names: hello shortener todo web
```

**Solution**: Check the short name is correct or use the full service name.

### Short name not working
1. Verify `metadata.yaml` has `short_name` field
2. Run `make verify-auto-detection` to check configuration
3. Check for typos in the short name

## See Also

- [Metadata Migration Guide](./METADATA_MIGRATION.md) - Full migration documentation
- [App Management Guide](./APP_MANAGEMENT.md) - Complete app management reference
- [Create App Guide](./CREATE_APP_GUIDE.md) - Creating new services
