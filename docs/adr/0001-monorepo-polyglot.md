# ADR-0001: Monorepo with Polyglot Stack

- **Status**: Accepted (retroactive)
- **Date**: 2026-05-28
- **Deciders**: Architect / Author

## Context

`cuckoo` houses 7+ services across three languages:

- Go (auth, user, IM, IM gateway, shortener, todo)
- Java (hello-service, Spring Boot)
- TypeScript (web frontend)

Plus shared `libs/` (Go observability), `proto/` (Protobuf API contracts), `deploy/`, `docs/`, and unified `Makefile` driving everything.

Two structural alternatives existed at the start:

1. **Per-language repos** (cuckoo-go, cuckoo-java, cuckoo-web) — common in Java-heavy enterprises
2. **Per-service repos** — N services × N CI pipelines × N release cycles
3. **Monorepo** — one `git clone` gets the world

## Decision

Use a **single monorepo** containing all services, shared libraries, infrastructure manifests, and documentation. Polyglot is intentional — pick the right tool per service rather than enforce uniformity.

## Why monorepo + polyglot over alternatives

| Concern | Per-repo | Monorepo (this) |
|---|---|---|
| Cross-service refactor (proto change touches 5 services) | N PRs across N repos, painful to land atomically | 1 PR, atomic |
| New service onboarding | Set up CI/lint/format from scratch each time | Inherit `Makefile` + `templates/{go,java}-service/` |
| Shared library evolution | Publish + version-bump in N consumers | In-tree change |
| Independent release cadence | Easy | Requires explicit per-service tagging (we use it: see ADR-0002 contract versioning) |
| CI cost on small PR | Cheap by default | Requires incremental detection (see `docs/ci-cd/DYNAMIC_CI_STRATEGY.md`) |

The dominant pain point was cross-service refactors. Per-repo adds friction proportional to architectural change rate; for a 1-author project iterating actively, that friction was unacceptable.

## Why polyglot

- **IM gateway, shortener, auth** — Go fits high-concurrency low-latency goroutine workloads
- **hello-service** (Java) — proves the gRPC contract is genuinely cross-language, not a Go-specific shortcut
- **web** — TypeScript + React for actual UI

Operational cost of polyglot (3 language toolchains, 3 sets of testing idioms) is paid once and amortized across services.

## Consequences

✅ Cross-service changes are atomic
✅ `make` from any service directory works the same way
✅ `proto/` is the source of truth for contracts; one regen step rebuilds Go + Java + TS clients
✅ Templates (`templates/go-service/`, `templates/java-service/`) make new-service onboarding mechanical

❌ CI must distinguish "this service changed" from "the world changed" — solved by [ADR-0002](0002-grpc-protobuf-contract.md) for contracts and `docs/ci-cd/DYNAMIC_CI_STRATEGY.md` for builds
❌ A single git history grows fast — partly mitigated by `docs/archive/` for stale design notes
❌ Tooling installation (Go + Java + Node) must be done once on every dev machine — `make init` automates it
❌ Cross-language gRPC stubs are regenerated centrally — small breakage risk if proto repo and stub state drift; build pipeline regenerates on every CI run to detect

## When to split

- A service crosses an organizational boundary (different team owns it with different release cadence)
- A service has security/compliance requirements that demand isolated CI (e.g., handling payment tokens with audit-only repo access)
- The repo grows past ~500k LOC where IDE / build performance suffers

None of these are true today.

## Validation

- New service onboarding measured in hours (use template), not days
- Cross-service breaking proto change ships as one PR with all dependent service updates
- `make build` from monorepo root builds the whole world; service-local `make build` builds one

## Related

- [ADR-0002](0002-grpc-protobuf-contract.md) — what makes cross-language contracts cheap
- [ADR-0003](0003-higress-over-plain-envoy.md) — gateway choices that depend on the proto/gRPC stack
- `docs/ci-cd/DYNAMIC_CI_STRATEGY.md` — how monorepo CI avoids "rebuild everything" cost
