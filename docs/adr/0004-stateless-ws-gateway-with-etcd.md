# ADR-0004: Stateless WebSocket Gateway with etcd-Backed Registry

- **Status**: Accepted (retroactive)
- **Date**: 2026-05-28
- **Deciders**: Architect / Author

## Context

The IM chat system has a tier of long-lived WebSocket connections from clients (web, mobile). The IM gateway service terminates these connections. Two questions:

1. How do other services route a message to "user X's current connection" without knowing which gateway instance holds it?
2. How does the gateway tier scale horizontally without sticky-session pain?

> **Honesty note**: `docs/architecture/IM_CHAT_SYSTEM.md` describes the system as supporting "千万级并发用户 / 10M concurrent users". This is a **design target** for the architectural pattern, not a measured throughput in this single-author repo. The IM tier has been load-tested locally to a much smaller scale; the design *is meant to* scale to that order of magnitude given enough hardware, but no claim is made that it has been observed doing so here.

Three options:

1. **Sticky sessions at the load balancer** — bind a user to one gateway instance forever
2. **Sharded gateway** — hash user_id to gateway instance N
3. **Stateless gateway + central registry** — every gateway instance is identical; a registry tells the rest of the system which instance currently holds a connection

## Decision

The IM gateway tier is **stateless**. All connection state — *which gateway instance holds user X's WebSocket right now* — is stored in an **etcd-backed registry** (`im-service` reads from this registry to route messages).

```
Client → LB → any IM Gateway instance
                    │
                    ├── on connect: registry.put(user_id → instance_id, lease)
                    │
                    └── on disconnect / lease expiry: entry GC'd

im-service wants to deliver msg → registry.get(user_id) → forward to that instance
                                                        OR fall back to slow path (offline queue)
```

## Why stateless + registry over sticky / sharded

| Concern | Sticky LB | Sharded | Stateless+registry (this) |
|---|---|---|---|
| Add/remove gateway instance | drains affected users | reshards range | registry entries TTL out, no reshuffling |
| Failure semantics | LB notices dead, drops connection (clients reconnect) | reshard on detection | etcd lease expires; clients reconnect to any instance |
| Cross-instance message delivery | proxy via shared queue | proxy via shared queue | direct: lookup instance, point-to-point |
| Operational complexity | low | medium (requires shard map and reshard tooling) | medium (requires etcd) |
| Failure radius of one bad instance | all that instance's users | all in that shard | all that instance's users (plus brief reconnect storm) |

The deciding factor: stateless+registry **decouples scaling from connection lifetime**. New instances start serving immediately; dead instances' users reconnect to *any* instance and the registry self-heals via lease expiry. No reshard coordination, no LB session-affinity tuning.

## Why etcd specifically

etcd was chosen over alternatives:

| Registry choice | Why not / why |
|---|---|
| Redis | possible, but lease semantics are weaker (TTL eviction is best-effort, not consensus); we already use Redis for caching, mixing roles muddies failure-mode analysis |
| Consul | similar feature set to etcd; etcd integrates more cleanly with the K8s ecosystem we're already in |
| ZooKeeper | older, heavier-weight; team familiarity in 2026 favors etcd |
| Custom (PostgreSQL + heartbeat) | reinventing for no benefit |
| **etcd** (this) | strong consistency, lease primitive, K8s-native, well-understood failure modes |

## Consequences

✅ Gateway scales horizontally with no coordination — start instance, it accepts connections, registry takes care of the rest
✅ Instance failure is recovery-by-reconnect, not recovery-by-resharding
✅ `im-service` (message router) has a simple routing function: registry lookup → either Fast Path (online) or Slow Path (queue for offline)
✅ Decouples connection density from scaling decisions

❌ etcd cluster is a hard dependency — if etcd is down, no new connections can register and no routing works (mitigated: etcd 3-node cluster is robust; runbook exists)
❌ Registry write per connection — at 10M-concurrent design target, registry write rate is ~connection-churn-rate; etcd watches scale but require capacity planning
❌ At-least-once message delivery via Fast Path can race with the disconnect→reconnect window — handled by [ADR-0007](0007-at-least-once-with-dedup.md) dedup
❌ Lease tuning matters — too short = phantom unavailability, too long = stale routing to dead instances; current default 30s with heartbeat at 10s

## Validation (local-scale)

- Spin up 3 IM gateway instances; connect 100 clients spread across all 3; kill instance A; verify clients reconnect to B/C and registry GCs A's entries within 30s
- Send messages from `im-service` to users on different instances; verify each lands at the right WebSocket
- Disconnect an etcd node; verify registry remains available (quorum-based reads)

## Validation (design-target scale, not done in this repo)

To genuinely validate the 10M-concurrent design target would require a load environment we don't have. If a real deployment reached even 1M concurrent, expected validation:

- etcd cluster handles connection-churn rate without quorum stalls
- Routing P99 stays under target (the design implies < 200ms; not measured here)
- Lease GC keeps pace with disconnect rate

## Related

- [ADR-0007](0007-at-least-once-with-dedup.md) — delivery semantics and dedup that depend on this routing tier
- `docs/architecture/IM_CHAT_SYSTEM.md` — full IM architecture
