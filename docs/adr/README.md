# Architecture Decision Records

This directory captures the **non-obvious decisions** behind `cuckoo`'s architecture. Each ADR explains a single decision, why it was made, what alternatives were rejected, and what the consequences are. ADRs are not tutorials — for "how does X work", see `docs/architecture/`.

The ADRs here were **retroactively extracted** in 2026-05 from the existing `docs/architecture/` and `docs/ci-cd/` documentation, which described the *what* but left the *why* implicit. Status of each is "Accepted retroactive" — the decisions are already in production-equivalent form in the codebase; the ADRs make them explicit.

For style, see also the [`cuckoo-echo` ADRs](https://github.com/pingxin403/cuckoo-echo) (private; 9 ADRs in a similar format), which serve as the reference template.

---

## Index

| ADR | Title | Date | Status |
|---|---|---|---|
| [0001](0001-monorepo-polyglot.md) | Monorepo with Polyglot Stack | 2026-05-28 | Accepted (retroactive) |
| [0002](0002-grpc-protobuf-contract.md) | gRPC + Protobuf as Service Contract | 2026-05-28 | Accepted (retroactive) |
| [0003](0003-higress-over-plain-envoy.md) | Higress for Production, Envoy for Local Dev | 2026-05-28 | Accepted (retroactive) |
| [0004](0004-stateless-ws-gateway-with-etcd.md) | Stateless WebSocket Gateway with etcd-Backed Registry | 2026-05-28 | Accepted (retroactive) |
| [0005](0005-multilevel-cache-for-read-heavy.md) | Multi-Level Cache + Singleflight for Read-Heavy Service | 2026-05-28 | Accepted (retroactive) |
| [0006](0006-otel-first-observability.md) | OpenTelemetry-First Observability Library | 2026-05-28 | Accepted (retroactive) |
| [0007](0007-at-least-once-with-dedup.md) | IM Delivery: At-Least-Once with Multi-Tier Dedup | 2026-05-28 | Accepted (retroactive) |

---

## On honesty

Several source architecture docs cite ambitious throughput numbers (e.g. "千万级并发", "500K+ QPS"). In `cuckoo`, **these are design targets, not measured production numbers** — `cuckoo` is a single-author reference implementation that has never carried real production traffic. ADRs in this directory frame those numbers as design targets explicitly. If a number lacks the "design target" qualifier, it is from a reproducible local measurement and the ADR says so.

## Adding a new ADR

1. Pick the next sequential number
2. Use the template structure: Context → Decision → Why this over alternatives → Consequences → Validation → Related
3. Keep it 80–150 lines; longer means it's becoming an architecture doc, which belongs in `docs/architecture/`
4. Add the entry to the index above
