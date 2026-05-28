# ADR-0007: IM Delivery — At-Least-Once with Multi-Tier Dedup

- **Status**: Accepted (retroactive)
- **Date**: 2026-05-28
- **Deciders**: Architect / Author

## Context

The IM chat system must answer: "what happens if a message could either be lost or be delivered twice — which do we accept?"

Three classical delivery semantics:

| Semantic | Loss possible? | Duplicate possible? | Cost |
|---|---|---|---|
| **At-most-once** | yes | no | simplest |
| **At-least-once** | no | yes | needs dedup downstream |
| **Exactly-once** | no | no | needs distributed transaction or 2-phase + dedup |

For IM:

- **Lost messages** are visible to users immediately ("I sent it but they didn't get it"). High user-frustration cost.
- **Duplicate messages** can be filtered transparently if dedup is correct. User never sees them.
- **Exactly-once** is operationally expensive (distributed consensus per message) and not actually achievable across mobile network breaks anyway.

## Decision

The IM system is **at-least-once with multi-tier dedup**:

1. Producers (gateway, im-service) may retry sends on uncertain failure
2. Each message carries a deterministic dedup key (typically `client_msg_id`)
3. Dedup happens at two layers:
   - **Server-side**: Redis SET with TTL holds dedup keys; conflict = duplicate, drop
   - **Client-side**: SQLite (mobile clients) or IndexedDB (web) holds the last N delivered IDs; conflict = drop before showing user

Eventual consistency on group membership cache uses TTL + etcd Watch to invalidate stale entries; messages can be briefly delivered to ex-members during the invalidation window — accepted, deduped client-side.

## Why at-least-once over alternatives

### Why not at-most-once

- A single mobile-network blip during ack would drop the message
- For IM, dropped-message rate even at 0.1% is noticeable to users (in a 10K-message-per-day power user, that's 10 lost messages)
- Operationally indistinguishable from "the app is broken" from a user's POV

### Why not exactly-once

- Requires distributed transactions across gateway + storage + delivery — operationally expensive
- Requires every link in the chain (including unreliable mobile networks) to participate — physically impossible
- The "exactly-once" promise on the wire is always a fiction; what you actually get is "at-least-once + idempotent consumer = effectively exactly-once". That's what we built.

## Why two layers of dedup

Server dedup alone doesn't solve everything:

- Server may have ack'd a message and the ack was lost → producer retries → server detects dedup → drops second copy → producer thinks first attempt failed → user re-sends → new message ID, looks like a real new message to the server
- Client-side dedup catches the case where the server's dedup ID and the client's "I just sent this" memory diverge

Client dedup alone doesn't solve everything:

- Without server dedup, a single message could fan out to multiple recipients with multiple delivery attempts → duplicate amplification at the recipient

Two layers, each catching the failure mode the other misses.

## Why TTL on dedup keys, not forever

- Dedup window only needs to cover the realistic retry window (typically minutes, generously seconds-to-low-minutes)
- Storing forever bloats Redis indefinitely
- TTL = 1 hour by default. Messages older than the window can theoretically duplicate but practically the producer would have given up by then.

## Why eventual consistency on group membership

When a user is added/removed from a group:

- Strict consistency option: wait for group membership cache to be updated everywhere before allowing message sends. Adds latency to every send.
- Eventual consistency option: cache is invalidated via etcd Watch with TTL fallback; brief window where a message can be delivered to a recently-removed member. Recipient drops it client-side.

For IM, eventual is fine. Briefly delivering one message to someone who left a group 200ms ago is much less bad than slowing every group message by an etcd round-trip.

## Consequences

✅ Producers can retry freely on uncertain failures without amplifying user-visible duplication
✅ Network blips don't drop messages
✅ Group membership changes don't slow the send hot path
✅ Two independent dedup layers protect against single-layer bugs

❌ Dedup logic must be correct on both server and client — bugs cause either visible duplicates (client dedup miss) or visible loss (over-aggressive server dedup)
❌ Server-side dedup TTL is a tunable; too short = retries slip through, too long = Redis bloat
❌ Group membership eventual consistency means a window where ex-members can receive one stale message — explicitly part of the contract, not a bug
❌ Dedup ID generation on client must be reliable — current scheme is `userId:timestamp:random_suffix`; if a client's clock or RNG is broken, dedup degrades

## Validation

- Property-based tests on dedup logic: random concurrent sends with various retry patterns produce no duplicates at recipient
- Chaos test: kill a gateway instance mid-message → producer retries → recipient sees exactly one copy
- Group membership churn test: rapidly add/remove user from group while messages flow → no duplicate or lost messages once cache settles

## Related

- [ADR-0004](0004-stateless-ws-gateway-with-etcd.md) — the routing tier this delivery semantic depends on
- `docs/architecture/IM_CHAT_SYSTEM.md` — full IM system design including the slow-path (offline queue) which has its own delivery semantics
