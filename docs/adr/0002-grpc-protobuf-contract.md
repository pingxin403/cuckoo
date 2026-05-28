# ADR-0002: gRPC + Protobuf as Service Contract

- **Status**: Accepted (retroactive)
- **Date**: 2026-05-28
- **Deciders**: Architect / Author

## Context

In a polyglot monorepo (see [ADR-0001](0001-monorepo-polyglot.md)), services need a contract that:

- Is language-agnostic (Go, Java, TS clients all consume it)
- Versions explicitly (breaking change ≠ silent breakage)
- Has acceptable browser support (the React frontend is a first-class client)

Three contract models were on the table:

1. **REST + JSON** — easiest browser story, weakest schema discipline
2. **GraphQL** — flexible client queries, but adds N3+1 + complex caching for a small service set
3. **gRPC + Protobuf** — strongest schema discipline, requires gRPC-Web for browser

## Decision

Use **gRPC + Protobuf** for inter-service communication. Browser clients use **gRPC-Web** translated at the gateway tier (see [ADR-0003](0003-higress-over-plain-envoy.md)).

Protobuf definitions live in `proto/` at the monorepo root. A `make proto` target regenerates Go + Java + TypeScript stubs in one shot.

## Why gRPC over REST/JSON

| Concern | REST/JSON | gRPC/Protobuf (this) |
|---|---|---|
| Schema enforcement | OpenAPI optional, often drifts | Compile-time; Protobuf is the source of truth |
| Cross-language client gen | per-language toolchain, often manual | one `protoc` invocation, all languages |
| Wire size | verbose JSON | binary, smaller |
| Streaming | Server-Sent Events, ad-hoc | first-class bidirectional streams |
| Browser story | trivial | requires gRPC-Web (gateway translates) |
| Debuggability | curl-friendly | grpcurl exists but slightly less friendly |

For a project where the React frontend is non-negotiable, the gRPC-Web gateway tax is real. It's paid in [ADR-0003](0003-higress-over-plain-envoy.md) — Higress translates gRPC-Web ↔ gRPC at the edge so service code never thinks about it.

## Why Protobuf over alternatives

| Alternative | Why rejected |
|---|---|
| Avro | Schema registry adds operational overhead; weaker IDE tooling |
| Thrift | Lower momentum, fewer 2026-era code generators |
| FlatBuffers | Optimization for read-without-parse use cases we don't have |
| Capnp | Niche; would orphan us in Go/Java ecosystems |

Protobuf's combination of cross-language tooling, code-gen quality, and IDE support (proto3 LSP support is mature) made it the default safe choice.

## Centralized proto generation

The repo uses a centralized proto build:

- `proto/` contains all `.proto` files
- `make proto` regenerates stubs into each consumer's expected location (`apps/<svc>/internal/pb/`, `apps/web/src/proto/` etc.)
- A pre-commit hook rejects manual edits to generated files
- CI re-runs `make proto` on every build and fails if working tree diverges from regenerated state

This is the "single source of truth" pattern. Consumers cannot drift.

## Consequences

✅ Adding/changing a service contract is a Protobuf edit + regen — Go, Java, TS clients all update in one PR
✅ Wire format is small enough that mobile / 4G clients tolerate it well
✅ Streaming primitives are first-class — IM gateway uses bidirectional streaming naturally
✅ IDE jumps from caller to RPC definition work across language boundaries

❌ Browser clients need gRPC-Web translation at the edge — solved at the gateway, costs an extra hop
❌ `protoc` and language plugins must be installed for `make proto` — `make init` automates this
❌ "Just curl it" is slightly harder; `grpcurl` works but is less universal than `curl`
❌ Versioning discipline must be deliberate — `proto/v1/`, `proto/v2/` prefixes; never edit a published version in place
❌ Generated code reviews require careful diff (large generated files) — pre-commit + CI gates the human review effort to source `.proto` only

## Validation

- A breaking proto change (e.g., field rename) fails CI on all consumers in one shot
- Cross-language interop test exists: hello-service (Java) is called from todo-service (Go) and frontend (TS)
- `make proto` is idempotent — running twice produces identical output

## Related

- [ADR-0001](0001-monorepo-polyglot.md) — monorepo is what makes centralized proto practical
- [ADR-0003](0003-higress-over-plain-envoy.md) — gateway absorbs the gRPC-Web translation cost
- `docs/architecture/PROTO_GENERATION.md` (current placeholder; should be expanded with codegen architecture)
