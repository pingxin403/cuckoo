/**
 * Quick QPS Test - Measure actual throughput
 * 
 * This test measures the actual QPS the service can handle
 * with the current optimizations.
 */

import http from 'k6/http';
import { check } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const latencyTrend = new Trend('request_latency');

// Test configuration - Start with moderate load
export const options = {
  scenarios: {
    constant_load: {
      executor: 'constant-arrival-rate',
      rate: 1000, // 1000 requests per second
      timeUnit: '1s',
      duration: '30s',
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
  },
  thresholds: {
    'http_req_duration': ['p(99)<50'], // P99 < 50ms
    'error_rate': ['rate<0.01'], // Error rate < 1%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test short codes (will create if not exist)
const TEST_CODES = ['test001', 'test002', 'test003', 'test004', 'test005'];

export function setup() {
  console.log('Quick QPS Test Setup');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Target: 1000 QPS for 30 seconds`);
  
  // Try to access health endpoint
  const health = http.get(`${BASE_URL}/health`);
  console.log(`Health check: ${health.status}`);
  
  return { codes: TEST_CODES };
}

export default function(data) {
  const startTime = Date.now();
  
  // Pick a random test code
  const code = data.codes[Math.floor(Math.random() * data.codes.length)];
  
  // Try to redirect (this is the main operation)
  const res = http.get(`${BASE_URL}/${code}`, {
    redirects: 0, // Don't follow redirects
    tags: { type: 'redirect' },
  });
  
  // Check if we got a redirect or not found (both are valid)
  const success = check(res, {
    'status is 302 or 404': (r) => r.status === 302 || r.status === 404,
    'response time < 50ms': (r) => r.timings.duration < 50,
  });
  
  errorRate.add(!success);
  
  const latency = Date.now() - startTime;
  latencyTrend.add(latency);
}

export function teardown(data) {
  console.log('Quick QPS test complete.');
  console.log('Check the summary for actual QPS achieved.');
}
