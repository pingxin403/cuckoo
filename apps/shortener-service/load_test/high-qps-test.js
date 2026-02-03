/**
 * High QPS Test - Test maximum throughput
 * 
 * This test gradually increases load to find the maximum QPS
 * the service can handle.
 */

import http from 'k6/http';
import { check } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const latencyTrend = new Trend('request_latency');
const successfulRequests = new Counter('successful_requests');

// Test configuration - Ramp up to high QPS
export const options = {
  stages: [
    { duration: '10s', target: 100 },   // Ramp up to 100 VUs
    { duration: '20s', target: 500 },   // Ramp up to 500 VUs
    { duration: '30s', target: 1000 },  // Ramp up to 1000 VUs
    { duration: '30s', target: 1000 },  // Hold at 1000 VUs
    { duration: '10s', target: 0 },     // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<20', 'p(99)<50'], // P95 < 20ms, P99 < 50ms
    'error_rate': ['rate<0.05'], // Error rate < 5%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
  console.log('High QPS Test');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Test: Ramp up to 1000 VUs over 100 seconds`);
  
  const health = http.get(`${BASE_URL}/health`);
  console.log(`Health check: ${health.status}`);
  
  return {};
}

export default function() {
  const startTime = Date.now();
  
  // Test health endpoint (lightweight operation)
  const res = http.get(`${BASE_URL}/health`, {
    tags: { type: 'health' },
  });
  
  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 50ms': (r) => r.timings.duration < 50,
  });
  
  if (success) {
    successfulRequests.add(1);
  }
  
  errorRate.add(!success);
  
  const latency = Date.now() - startTime;
  latencyTrend.add(latency);
}

export function teardown() {
  console.log('High QPS test complete.');
  console.log('Check the summary for peak QPS achieved.');
}
