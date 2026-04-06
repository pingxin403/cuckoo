import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const GRPC_URL = __ENV.GRPC_URL || 'http://localhost:9092';

// Custom metrics
const errorRate = new Rate('errors');
const redirectLatency = new Trend('redirect_latency');
const createLatency = new Trend('create_latency');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 100 },   // Ramp up to 100 users
    { duration: '1m', target: 100 },    // Sustained 100 users
    { duration: '30s', target: 500 },   // Ramp up to 500 users
    { duration: '2m', target: 500 },    // Sustained 500 users
    { duration: '30s', target: 1000 },  // Spike to 1000 users
    { duration: '1m', target: 1000 },   // Sustained 1000 users
    { duration: '30s', target: 0 },     // Ramp down
  ],
  thresholds: {
    'redirect_latency': ['p(99)<10'],  // P99 < 10ms
    'create_latency': ['p(99)<50'],     // P99 < 50ms
    'errors': ['rate<0.01'],            // < 1% error rate
  },
};

export default function () {
  // Test 1: Redirect (most common operation)
  const redirectRes = http.get(`${BASE_URL}/abc1234`, {
    tags: { name: 'redirect' },
  });
  
  redirectLatency.add(redirectRes.timings.duration);
  
  const redirectOk = check(redirectRes, {
    'redirect status is 302 or 404': (r) => [302, 404].includes(r.status),
  });
  
  errorRate.add(!redirectOk);
  
  // Test 2: Create short link (less frequent)
  const createPayload = JSON.stringify({
    long_url: `https://example.com/page-${Math.random()}`,
  });
  
  const createRes = http.post(`${GRPC_URL}/api.v1.ShortenerService/CreateShortLink`, createPayload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'create' },
  });
  
  createLatency.add(createRes.timings.duration);
  
  const createOk = check(createRes, {
    'create returns 200 or gRPC': (r) => r.status >= 200 && r.status < 500,
  });
  
  errorRate.add(!createOk);
  
  // Test 3: Health check
  const healthRes = http.get(`${BASE_URL}/health`, {
    tags: { name: 'health' },
  });
  
  check(healthRes, {
    'health returns 200': (r) => r.status === 200,
  });
  
  sleep(1);
}