/**
 * Redirect QPS Test - Test actual redirect performance
 * 
 * This test measures QPS for the main redirect operation
 * with real short codes.
 */

import http from 'k6/http';
import { check } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const latencyTrend = new Trend('request_latency');
const successfulRedirects = new Counter('successful_redirects');
const cacheHits = new Counter('cache_hits');

// Test configuration - High constant load
export const options = {
  scenarios: {
    high_load: {
      executor: 'constant-vus',
      vus: 500,
      duration: '60s',
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<10', 'p(99)<20'], // P95 < 10ms, P99 < 20ms
    'error_rate': ['rate<0.01'], // Error rate < 1%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test codes that exist
const TEST_CODES = ['test001', 'test002', 'test003', 'test004', 'test005'];

export function setup() {
  console.log('Redirect QPS Test');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Test: 500 VUs for 60 seconds`);
  console.log(`Testing redirect performance with real short codes`);
  
  // Verify test codes exist
  for (const code of TEST_CODES) {
    const res = http.get(`${BASE_URL}/${code}`, { redirects: 0 });
    console.log(`Test code ${code}: ${res.status}`);
  }
  
  return { codes: TEST_CODES };
}

export default function(data) {
  const startTime = Date.now();
  
  // Pick a random test code
  const code = data.codes[Math.floor(Math.random() * data.codes.length)];
  
  // Test redirect (don't follow)
  const res = http.get(`${BASE_URL}/${code}`, {
    redirects: 0,
    tags: { type: 'redirect', code: code },
  });
  
  const success = check(res, {
    'status is 302': (r) => r.status === 302,
    'has location header': (r) => r.headers['Location'] !== undefined,
    'response time < 20ms': (r) => r.timings.duration < 20,
  });
  
  if (res.status === 302) {
    successfulRedirects.add(1);
    
    // Fast response likely means cache hit
    if (res.timings.duration < 5) {
      cacheHits.add(1);
    }
  }
  
  errorRate.add(!success);
  
  const latency = Date.now() - startTime;
  latencyTrend.add(latency);
}

export function teardown() {
  console.log('Redirect QPS test complete.');
  console.log('Check metrics for:');
  console.log('- http_reqs: Total QPS achieved');
  console.log('- successful_redirects: Successful redirect count');
  console.log('- cache_hits: Estimated cache hits (< 5ms)');
  console.log('- http_req_duration p(99): 99th percentile latency');
}
