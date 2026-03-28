# IM P0 Delivery & Security Closure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Complete P0 IM/IM-Gateway production closure: gateway wiring, cross-gateway delivery/read-receipts, ACK tracking, origin allowlist, and IM service group/status TODOs.

**Architecture:** Keep current architecture (Gateway + IM Service + Registry + Kafka) and close only missing runtime links. Implement minimally invasive changes in existing services, prioritize compatibility, and validate through focused unit/integration tests before docs sync.

**Tech Stack:** Go 1.24, gRPC, Gorilla WebSocket, Redis, Kafka (segmentio/kafka-go), etcd registry, testify.

---

## Prerequisites

### Task 0: Isolated workspace and baseline check

**Files:**
- Modify: none
- Test: none

**Step 1: Create a dedicated worktree**

Run: `git worktree add ../cuckoo-im-p0 -b feat/im-p0-closure`

**Step 2: Enter new worktree and verify clean state**

Run: `git status`
Expected: clean working tree on `feat/im-p0-closure`

**Step 3: Record current failing/passing baseline (IM Service)**

Run: `go test ./...`
Workdir: `apps/im-service`

**Step 4: Record current failing/passing baseline (IM Gateway)**

Run: `go test ./...`
Workdir: `apps/im-gateway-service`

**Step 5: Commit baseline notes (optional markdown note)**

Run: `git add . && git commit -m "chore: capture im p0 baseline status"`

---

### Task 1: IM Service group message Kafka publish closure

**Files:**
- Modify: `apps/im-service/service/im_service.go` (RouteGroupMessage)
- Test: `apps/im-service/service/im_service_test.go`

**Step 1: Write failing test for group publish event path**

Add test case in `im_service_test.go` asserting `RouteGroupMessage` publishes an event for offline processing when Kafka producer exists.

**Step 2: Run single test to verify fail**

Run: `go test ./service -run TestRouteGroupMessage -v`
Workdir: `apps/im-service`
Expected: FAIL on missing publish behavior.

**Step 3: Implement minimal group publish logic**

In `RouteGroupMessage`, publish an offline/group event through existing producer abstraction and keep response contract compatible.

**Step 4: Re-run test and package tests**

Run: `go test ./service -run TestRouteGroupMessage -v && go test ./service`
Workdir: `apps/im-service`
Expected: PASS.

**Step 5: Commit**

Run: `git add apps/im-service/service/im_service.go apps/im-service/service/im_service_test.go && git commit -m "feat(im-service): close group message kafka publish path"`

---

### Task 2: IM Service GetMessageStatus minimal implementation

**Files:**
- Modify: `apps/im-service/service/im_service.go` (`GetMessageStatus`)
- Test: `apps/im-service/service/im_service_test.go`

**Step 1: Write failing tests for status query outcomes**

Cover at least: valid msg id status response, invalid input validation, fallback/default status.

**Step 2: Run targeted tests to verify fail**

Run: `go test ./service -run TestGetMessageStatus -v`
Workdir: `apps/im-service`
Expected: FAIL (currently TODO behavior).

**Step 3: Implement minimal status mapping**

Implement status retrieval/mapping with existing structures; avoid schema changes.

**Step 4: Re-run service tests**

Run: `go test ./service`
Workdir: `apps/im-service`
Expected: PASS.

**Step 5: Commit**

Run: `git add apps/im-service/service/im_service.go apps/im-service/service/im_service_test.go && git commit -m "feat(im-service): implement minimal message status query"`

---

### Task 3: IM Service structured logging in critical routing paths

**Files:**
- Modify: `apps/im-service/service/im_service.go`
- Test: `apps/im-service/service/im_service_test.go` (behavioral assertions only, no brittle log text matching)

**Step 1: Add failing behavior test where useful (optional lightweight)**

Add test ensuring success/fallback/failure branches remain behaviorally unchanged after logging.

**Step 2: Run targeted tests**

Run: `go test ./service -run TestRoutePrivateMessage|TestRouteGroupMessage -v`
Workdir: `apps/im-service`

**Step 3: Add structured logs**

Add contextual fields (`msg_id`, `conversation_id`, `delivery_path`, `error`) in retry/fallback/failure branches, without changing response semantics.

**Step 4: Run IM service full tests**

Run: `go test ./...`
Workdir: `apps/im-service`

**Step 5: Commit**

Run: `git add apps/im-service/service/im_service.go apps/im-service/service/im_service_test.go && git commit -m "chore(im-service): add structured logs for routing critical paths"`

---

### Task 4: IM Gateway startup wiring and lifecycle closure

**Files:**
- Modify: `apps/im-gateway-service/main.go`
- Modify: `apps/im-gateway-service/service/gateway_service.go` (only if constructor/start signatures need minimal extension)
- Test: `apps/im-gateway-service/integration_test/service_dependency_test.go`

**Step 1: Write/extend failing test for startup path**

Add/extend test to assert gateway starts internal components with configured Kafka and can shut down cleanly.

**Step 2: Run targeted integration test**

Run: `go test ./integration_test -run TestServiceStartup -v`
Workdir: `apps/im-gateway-service`
Expected: FAIL if wiring is incomplete.

**Step 3: Implement wiring**

Replace placeholder nil-client comments with concrete initialization and start call chain (`gateway.Start(kafkaConfig)`), keeping graceful shutdown intact.

**Step 4: Re-run gateway tests**

Run: `go test ./...`
Workdir: `apps/im-gateway-service`
Expected: PASS.

**Step 5: Commit**

Run: `git add apps/im-gateway-service/main.go apps/im-gateway-service/service/gateway_service.go apps/im-gateway-service/integration_test/service_dependency_test.go && git commit -m "feat(im-gateway): complete startup wiring and lifecycle"`

---

### Task 5: Cross-gateway message and read-receipt delivery

**Files:**
- Modify: `apps/im-gateway-service/service/push_service.go`
- Modify: `apps/im-gateway-service/service/gateway_service.go` (if forwarding helper needed)
- Test: `apps/im-gateway-service/service/push_service_test.go`
- Test: `apps/im-gateway-service/service/read_receipt_integration_test.go`

**Step 1: Add failing tests for remote-node branches**

Add cases where registry returns remote gateway node; assert delivery attempts use remote forwarding path for message + read receipt.

**Step 2: Run targeted tests to verify fail**

Run: `go test ./service -run TestPushService -v`
Workdir: `apps/im-gateway-service`

**Step 3: Implement minimal remote forwarding path**

Replace TODO branches that mark remote devices as failed; forward via gateway inter-node mechanism and preserve response aggregation.

**Step 4: Run service + integration tests**

Run: `go test ./service ./integration_test -v`
Workdir: `apps/im-gateway-service`

**Step 5: Commit**

Run: `git add apps/im-gateway-service/service/push_service.go apps/im-gateway-service/service/gateway_service.go apps/im-gateway-service/service/push_service_test.go apps/im-gateway-service/service/read_receipt_integration_test.go && git commit -m "feat(im-gateway): implement cross-gateway delivery for message and read receipts"`

---

### Task 6: ACK receive/associate/timeout closure

**Files:**
- Modify: `apps/im-gateway-service/service/gateway_service.go` (`handleAck` and related tracking structures)
- Test: `apps/im-gateway-service/service/gateway_service_test.go`
- Test: `apps/im-gateway-service/metrics/metrics_test.go` (if timeout counters updated)

**Step 1: Write failing tests for ACK behaviors**

Cover ack accepted, unknown ack id, timeout expiry path, and status update side effects.

**Step 2: Run ack-specific tests**

Run: `go test ./service -run TestConnection.*Ack|Test.*ACK -v`
Workdir: `apps/im-gateway-service`

**Step 3: Implement ACK state machine minimal closure**

Add pending map/timeout handling and callback/status update hooks; keep lock scope minimal.

**Step 4: Re-run tests**

Run: `go test ./service ./metrics -v`
Workdir: `apps/im-gateway-service`

**Step 5: Commit**

Run: `git add apps/im-gateway-service/service/gateway_service.go apps/im-gateway-service/service/gateway_service_test.go apps/im-gateway-service/metrics/metrics_test.go && git commit -m "feat(im-gateway): close ack tracking and timeout handling"`

---

### Task 7: Origin allowlist security baseline

**Files:**
- Modify: `apps/im-gateway-service/service/gateway_service.go` (upgrader CheckOrigin)
- Modify: `apps/im-gateway-service/config/config.go` (origin-related config fields/defaults)
- Modify: `apps/im-gateway-service/main.go` (inject origin config)
- Test: `apps/im-gateway-service/service/gateway_service_test.go`

**Step 1: Add failing origin policy tests**

Add tests for allowed origin, denied origin, empty origin handling under both allow/disallow-empty configs.

**Step 2: Run targeted tests**

Run: `go test ./service -run TestHandleWebSocket.*Origin -v`
Workdir: `apps/im-gateway-service`

**Step 3: Implement configurable origin checker**

Implement allowlist + empty-origin policy with default deny and log-friendly rejection reasons.

**Step 4: Run full gateway tests**

Run: `go test ./...`
Workdir: `apps/im-gateway-service`

**Step 5: Commit**

Run: `git add apps/im-gateway-service/service/gateway_service.go apps/im-gateway-service/config/config.go apps/im-gateway-service/main.go apps/im-gateway-service/service/gateway_service_test.go && git commit -m "feat(im-gateway): enforce configurable websocket origin allowlist"`

---

### Task 8: Docs sync and P0 verification report

**Files:**
- Modify: `apps/im-service/README.md`
- Modify: `apps/im-gateway-service/README.md`
- Create: `docs/reports/2026-03-28-im-p0-verification.md`

**Step 1: Capture actual implemented behavior**

List delivered P0 capabilities, known limitations, rollback points.

**Step 2: Update readmes to match code truth**

Remove outdated TODO claims if completed; keep remaining TODOs explicit.

**Step 3: Write verification report**

Include commands run, summary of test results, risk list, and rollback instructions.

**Step 4: Run final verification suites**

Run:
- `go test ./...` (workdir `apps/im-service`)
- `go test ./...` (workdir `apps/im-gateway-service`)

**Step 5: Commit**

Run: `git add apps/im-service/README.md apps/im-gateway-service/README.md docs/reports/2026-03-28-im-p0-verification.md && git commit -m "docs(im): sync p0 capability status and verification report"`

---

## Final acceptance checklist

- [ ] All TODOs in P0 scope removed or replaced with explicit intentional deferred notes.
- [ ] IM Service + IM Gateway tests pass in full.
- [ ] Cross-gateway delivery/read-receipt paths validated by tests.
- [ ] ACK timeout path observable and tested.
- [ ] Origin policy defaults to safe behavior and supports migration toggles.
- [ ] Readme/docs reflect real implementation state.
