# ADR-0006: OpenTelemetry-First Observability Library

- **Status**: Accepted (retroactive)
- **Date**: 2026-05-28
- **Deciders**: Architect / Author

## Context

Every service in `cuckoo` needs:

- Metrics (request rate, error rate, latency, custom business metrics)
- Logs (structured, traceable, queryable)
- Traces (cross-service, cross-language)

Three instrumentation strategies:

1. **Vendor-specific SDK per service** (e.g., Datadog APM agent, New Relic SDK)
2. **OpenTelemetry SDK + collector pipeline**
3. **No instrumentation, debug from logs only**

For a polyglot stack (Go + Java + TS), the choice has follow-on consequences for trace correlation across language boundaries.

## Decision

Build a thin shared library `libs/observability` that wraps **OpenTelemetry SDK** (Go) for the language-uniform service onboarding API, and emit telemetry to an **OTel Collector** which fans out to backend-specific stores:

```
service ──OTLP──> OTel Collector ──┬──> Prometheus    (metrics)
                                   ├──> Tempo / Jaeger (traces)
                                   └──> Loki / ELK     (logs, structured JSON)
```

Java services use the OTel Java agent for auto-instrumentation; the contract (`traceparent` header, attribute names) matches what `libs/observability` produces in Go.

## Why OTel + collector over vendor SDKs

| Concern | Vendor SDK | OTel + Collector (this) |
|---|---|---|
| Cross-language consistency | each vendor has its own taxonomy | OTel semantic conventions are uniform |
| Vendor lock-in | high — switching = rewrite all instrumentation | low — switch backends by editing collector config |
| Sampling decisions | tied to SDK config per service | centralized at collector (tail-sampling possible) |
| Auto-instrumentation breadth | varies by vendor | OTel covers HTTP, gRPC, DB clients, Kafka, etc. |
| Cost | typically per-host or per-event vendor pricing | pay only for backend storage |
| Open standard | proprietary | CNCF, broad ecosystem |

For a polyglot project where trace continuity across Go ↔ Java is non-negotiable, OTel's cross-language consistency is the deciding factor.

## Why a wrapper library (`libs/observability`) on top of raw OTel SDK

OTel SDK is comprehensive but verbose. Every new service would otherwise repeat:

- Resource attribute setup (`service.name`, `service.version`, `deployment.env`)
- Exporter wiring (OTLP gRPC to collector)
- Sampler configuration
- HTTP/gRPC handler instrumentation
- Tracer/meter providers as singletons

`libs/observability` exposes:

```go
obs.Setup(obs.Config{
    ServiceName: "shortener-service",
    Environment: "prod",
})  // one call at app boot

defer obs.Shutdown()  // flush pending exports

// then use obs.Tracer() / obs.Meter() / obs.Logger() throughout
```

A new service onboards observability in ~5 lines, not 50. Onboarding cost is the difference between "every service is observable" and "the service that should have been observable wasn't."

## Why not just Prometheus + structlog (skipping OTel)

Tempting for a simpler stack. Three reasons we didn't:

1. **Distributed tracing is non-optional in IM**. The chat path goes Client → IM Gateway → IM Service → Storage. Without traces, debugging "why was this message slow" requires correlating timestamps across logs from 3 services.
2. **Cross-language tracing**: hello-service (Java) is called from Go services. Bespoke per-language tracing libraries don't correlate.
3. **Future-proofing**: even if today we only export metrics, having OTel SDK already integrated means adding traces is config, not code.

## Consequences

✅ Onboarding observability for a new service is a one-line `obs.Setup()` call
✅ Switching backends (e.g., Prometheus → Mimir, Tempo → Jaeger) is a collector config edit; service code never changes
✅ Cross-language traces work — a request entering hello-service (Java) and continuing to todo-service (Go) shows as one trace in Tempo
✅ Sampling decisions are centralized; no per-service tuning
✅ Sensitive data scrubbing happens at the collector (one place to maintain the rule)

❌ OTel Collector becomes critical infra — must be deployed, monitored, and have its own alerts
❌ One more thing in `make init` (collector config), one more thing in K8s manifests
❌ Auto-instrumentation can produce noisy spans (every DB call as its own span); some custom-instrumentation guidance is needed for new contributors
❌ OTel semantic-convention versions evolve; pinning is required to avoid drift between services emitting different attribute names

## Cross-reference

This ADR is the local cuckoo-specific instance of a broader pattern. The full treatment of the same observability platform pattern (with deeper ADRs on tail-sampling, structured-logging contract, and three-tier alert layering) lives in the **[observability-platform-showcase](https://github.com/pingxin403/observability-platform-showcase)** repo.

## Validation

- Service trace shown end-to-end across Java → Go boundary in Jaeger UI
- Backend swap exercise: collector reconfigured to dual-export to a second Prometheus; both backends populated with identical data
- New-service smoke check: `templates/go-service/` produces a service with observability already wired

## Related

- [ADR-0001](0001-monorepo-polyglot.md) — `libs/observability` lives in the monorepo and is consumed in-tree
- `docs/architecture/OBSERVABILITY_SYSTEM.md` — concrete architecture
- External: [observability-platform-showcase](https://github.com/pingxin403/observability-platform-showcase)
