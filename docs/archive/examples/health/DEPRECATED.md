# Deprecated: Multi-Region Health Check Example

**Status**: DEPRECATED  
**Date**: 2026-02-02  
**Replaced By**: `libs/health` - Standardized Health Check Library

## Deprecation Notice

This example implementation has been deprecated and replaced by the standardized health check library at `libs/health`.

## Migration Path

All services have been migrated to use the new health check library:
- ✅ shortener-service
- ✅ todo-service
- ✅ auth-service
- ✅ user-service
- ✅ im-service
- ✅ im-gateway-service

## New Library Features

The new `libs/health` library provides:
- Standardized liveness and readiness probes
- Built-in checks for common dependencies (Database, Redis, Kafka, HTTP, gRPC)
- Circuit breaker integration
- Auto-recovery mechanisms
- Comprehensive observability (metrics, logging, tracing)
- Anti-flapping logic
- Graceful shutdown support

## Documentation

For the new health check system, see:
- Library README: `libs/health/README.md`
- Integration guides: `apps/*/HEALTH_CHECK_INTEGRATION.md`
- Operational runbook: `docs/operations/health-check-runbook.md`
- Training materials: `docs/training/health-checks/`

## Why This Was Deprecated

This example was created as a proof-of-concept for multi-region health checking. The standardized library provides:
1. Better performance (< 200ms health checks vs ~500ms)
2. More comprehensive testing (73.7% coverage)
3. Production-ready features (circuit breakers, auto-recovery)
4. Consistent implementation across all services
5. Better observability integration

## Historical Context

This example was part of the multi-region demo and served its purpose well for demonstrating health check concepts. The lessons learned from this implementation informed the design of the standardized library.

## Archive Location

This code has been moved to `docs/archive/examples/health/` for historical reference only. It should not be used in new implementations.
