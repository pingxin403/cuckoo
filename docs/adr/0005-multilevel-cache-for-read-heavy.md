# ADR-0005: Multi-Level Cache + Singleflight for Read-Heavy Service

- **Status**: Accepted (retroactive)
- **Date**: 2026-05-28
- **Deciders**: Architect / Author

## Context

The URL shortener service is the canonical read-heavy workload in `cuckoo`. The redirect path:

```
GET /<short_code>  →  resolve to long URL  →  302 redirect
```

is hit far more often than the create path. The dominant request is "given a 6-character code, return the URL", and it must be sub-10ms to feel instant.

> **Honesty note**: `docs/architecture/URL_SHORTENER_SERVICE.md` cites "500K+ QPS read throughput" and "<10ms P99". These are **design targets**, not measured production figures — `cuckoo` is a single-author reference implementation. Local benchmarks have been run at much smaller scale; the architecture is *designed to* sustain that target on appropriate hardware. The architecture decisions below are valid regardless of which order of magnitude is actually observed; they're the right structural choices for any read-heavy service.

The naive design (one MySQL query per redirect) doesn't scale. Three improvement directions:

1. **Process-local cache** (in-memory map / LRU)
2. **Distributed cache** (Redis)
3. **Both, layered**

## Decision

Use a **three-tier cache stack**:

```
L1 (in-memory LRU, per-process)
   ↓ miss
L2 (Redis, cluster-shared)
   ↓ miss
L3 (MySQL, source of truth)
```

Plus **singleflight** (Go's `golang.org/x/sync/singleflight`) on the L1-miss path: if 1000 goroutines simultaneously miss L1 for the same key, exactly one fetches from L2/L3 and the rest wait on the result.

## Why three tiers, not one

| Tier | Latency | Hit-rate (design target) | Cost |
|---|---|---|---|
| L1 (memory) | < 100 µs | 80%+ for hot keys | RAM-bounded; LRU eviction |
| L2 (Redis) | 1–2 ms | 95% cumulative | network round-trip; shared across all replicas |
| L3 (MySQL) | 5–20 ms | source of truth | slow path |

Each tier earns its keep:

- **Without L1**: every redirect is a network call to Redis. Wastes the cheap CPU-cache-resident bytes that a single redirect needs.
- **Without L2**: each replica has its own L1 miss → cold-start L1 means many MySQL hits when traffic shifts between replicas.
- **Without L3**: cache evictions are unrecoverable.

## Why singleflight

Cache stampede: when a hot key expires from L1, every concurrent goroutine that needs it will independently miss → all of them hit L2 (and possibly L3) → write the same value back to L1. At high concurrency, this can be 100×–1000× redundant fetches per expiry event.

Singleflight: one fetcher wins, others wait. Net result for the same expiry event:

- Without singleflight: 1000 L2 reads, 1000 L1 writes (lock contention)
- With singleflight: 1 L2 read, 1 L1 write, 999 callers receive the same value

This is one of the highest-leverage 10-line additions in the service.

## Why not write-through caching

A common alternative is to have the writer populate L1 + L2 immediately on `POST /shorten`. We don't:

- Most created short codes are never accessed (long-tail distribution)
- Writing to L1/L2 on every create wastes capacity on cold keys
- Lazy-load on first read keeps the cache aligned to actual demand

## Cache invalidation

The shortener has very few invalidation events: codes are append-only with rare deletes. Strategy:

- L1 entry TTL = 5 minutes (bounded staleness; lazy reload from L2/L3)
- L2 entry TTL = 1 hour
- Deletes/updates publish to a Redis pub/sub channel; all replicas drop their L1 entry
- "Stale-while-revalidate" not implemented — for shorteners, returning a stale URL for up to 5 min is acceptable; for a stronger consistency requirement (e.g., expiring promotional links), we'd add explicit invalidation

## Consequences

✅ Hot-key reads served entirely from L1 in single-digit-microsecond range
✅ Singleflight collapses cache stampedes
✅ Cold replicas warm naturally without overwhelming MySQL
✅ Each layer can be tuned independently (LRU size, Redis TTL, MySQL connection pool)

❌ Bounded staleness: changes propagate within 5 min unless invalidated explicitly
❌ Three configuration knobs (L1 size, L2 TTL, L3 connection pool) instead of one
❌ Diagnosing "wrong URL returned" requires checking three layers; we tag responses with `cache_hit_tier=L1|L2|L3` for debugging
❌ Memory pressure on Pods if L1 size is set too large — bound at 100MB by default; monitor RSS

## Validation

- Property-based test (`url_validator_property_test.go`) covers the URL canonicalization that L1 caches
- Cache-hit-rate counter exposed as Prometheus metric `shortener_cache_hits_total{tier}`
- Load test (Locust) exercises the cache tiers and asserts L1 hit rate > 60% on hot keyset

## Related

- `docs/architecture/URL_SHORTENER_SERVICE.md` — full service architecture
- `apps/shortener-service/cache/` — implementation (l1_cache, l2_cache, cache_manager, with property tests)
- [ADR-0006](0006-otel-first-observability.md) — cache metrics flow through the same observability pipeline as everything else
