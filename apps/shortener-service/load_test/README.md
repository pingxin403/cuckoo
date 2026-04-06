# URL Shortener Load Tests

## Prerequisites

- k6 installed: `brew install k6` or `go install go.k6.io/k6@latest`
- Shortener service running on port 8080 (HTTP) and 9092 (gRPC)

## Running Load Tests

### Redirect Load Test (Sustained)

```bash
k6 run redirect-load-test.js
```

SLAs:
- P99 latency < 10ms for redirects
- P99 latency < 50ms for creation
- Error rate < 1%

### Creation Load Test

```bash
k6 run creation-load-test.js
```

SLAs:
- P99 latency < 50ms for creation
- Error rate < 1%

### Spike Test

```bash
k6 run spike-test.js
```

Tests: 0 → 100K QPS spike
SLAs:
- P99 latency < 100ms
- Error rate < 5%

## Environment Variables

```bash
BASE_URL=http://localhost:8080 k6 run redirect-load-test.js
GRPC_URL=http://localhost:9092 k6 run creation-load-test.js
```

## Expected Results

- Redirect: ~500K+ QPS with warm cache
- Creation: ~10K QPS
- Spike: Service handles 100K burst