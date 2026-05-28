# ADR-0003: Higress for Production, Envoy for Local Dev

- **Status**: Accepted (retroactive)
- **Date**: 2026-05-28
- **Deciders**: Architect / Author

## Context

The polyglot gRPC stack ([ADR-0002](0002-grpc-protobuf-contract.md)) needs an edge tier that:

- Translates gRPC-Web ↔ gRPC for browser clients
- Centralizes auth, rate limit, and CORS so service code stays focused on business logic
- Works the same in local dev and Kubernetes production

Three options:

1. **Plain Envoy** in both environments
2. **Higress** in both environments
3. **Envoy locally + Higress in production** — what we picked

## Decision

Use **Higress** as the production gateway (Kubernetes ingress controller), and **plain Envoy** in local dev (Docker Compose). Both speak the same upstream gRPC protocol so service code is identical regardless of environment.

## Why Higress in production

Higress is a cloud-native API gateway built on Envoy + Istio. Choosing it over plain Envoy in production gets us:

| Capability | Plain Envoy | Higress (this) |
|---|---|---|
| gRPC-Web ↔ gRPC translation | filter config + custom YAML | built-in, zero config |
| Rate limiting (per-IP, per-API) | requires Lua filters or external rate-limit service | built-in CRDs |
| Circuit breaking | `cluster.outlier_detection` config | built-in CRDs with sane defaults |
| CORS | filter config | built-in CRDs |
| Kubernetes-native CRDs | no, raw YAML | yes (`HigressGateway`, `HigressVirtualService`) |
| WASM extensibility | requires custom filter dev | extension marketplace |
| Operator / lifecycle | self-managed | managed via Helm + operator |

The cost of replicating these features on plain Envoy (custom filters, rate-limit service, lifecycle automation) easily outweighs Higress's marginal complexity for a single-author project.

## Why plain Envoy locally

For local Docker Compose, we don't need K8s CRDs, operator lifecycle, or rate-limit-as-a-service. Plain Envoy with a static `envoy-local-config.yaml` is:

- Lighter (single container vs. control plane + data plane)
- Easier to debug (one config file, no admission webhooks)
- No K8s dependency for `make dev-up`

The trade-off is a small dev/prod parity gap — local Envoy doesn't enforce rate limits or circuit breaking. Acceptable: those are production-shape concerns, not "does the service's RPC even respond" concerns.

## Why not Higress everywhere

Tried it. Higress requires K8s primitives (CRDs, namespaces, operator). Bringing K8s into local Docker Compose adds 30s+ to `make dev-up` and 2 GB+ of memory pressure. For a 1-author iteration loop, this is the kind of friction that compounds.

Local Envoy reproduces the request shape (gRPC-Web → gRPC translation works locally), and the gateway-specific concerns (rate limit, CB) are validated in staging/CI integration tests rather than every dev cycle.

## Consequences

✅ Production gets enterprise-grade gateway features for free
✅ Local dev stays fast and dependency-light
✅ Service code never branches on environment — same gRPC server, same proto contract

❌ Two configs to maintain (`deploy/docker/envoy-local-config.yaml` + Higress CRDs in `deploy/k8s/`)
❌ Rate-limit and circuit-breaker bugs only surface in staging/CI, not in local dev
❌ Higress is less universally known than plain Envoy — learning curve for new contributors
❌ Higress version upgrades require upstream-tracking (mitigated: Higress follows Envoy semver semantics)

## Validation

- gRPC-Web client (browser) and gRPC client (todo-service) both succeed against the same backend
- Local `docker compose up` brings up gateway in < 10s
- K8s `kubectl apply -f deploy/k8s/higress/` brings up gateway with CRDs configured
- Rate-limit policies in production reject malformed requests at the gateway layer (not the service)

## Related

- [ADR-0002](0002-grpc-protobuf-contract.md) — what the gateway is translating
- `docs/architecture/HIGRESS_ROUTING_CONFIGURATION.md` — concrete routing rules
- `docs/architecture/INFRASTRUCTURE.md` — full local + production infrastructure picture
