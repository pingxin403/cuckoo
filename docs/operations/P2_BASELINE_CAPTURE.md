# P2 Baseline Capture Guide

## Scope

This guide defines baseline capture for OpenSpec change `im-p2-performance-architecture-polish`, task **1.1**.

Baseline dimensions:
- throughput
- P95/P99 latency
- timeout rate (when available)
- error rate

Critical paths:
- private message
- group message
- cross-gateway delivery
- read-receipt path

## How to Capture

From repository root:

```bash
./apps/im-gateway-service/load_test/run-load-tests.sh quick
```

or full baseline:

```bash
./apps/im-gateway-service/load_test/run-load-tests.sh all
```

After successful run, a baseline report is generated automatically:

```bash
apps/im-gateway-service/load_test/results/<timestamp>/p2-baseline-report.json
```

## Report Semantics

`p2-baseline-report.json` includes:
- `baselines.privateMessage`
- `baselines.groupMessage`
- `baselines.crossGatewayDelivery`
- `baselines.readReceiptPath`

If cluster or read-receipt dedicated runs are missing, report marks them as pending placeholders rather than failing generation.

## Acceptance Notes for Task 1.1

Task 1.1 is considered complete when:
1. baseline collector test passes,
2. runner can generate baseline report in quick/all flow,
3. report structure covers all required critical paths,
4. current pending dimensions are explicitly documented.

## Next Steps

- Task 1.2 should add dual/multi-gateway cluster baseline runs and fill `crossGatewayDelivery` with measured values.
- Add dedicated read-receipt load scenario to replace placeholder in `readReceiptPath`.
