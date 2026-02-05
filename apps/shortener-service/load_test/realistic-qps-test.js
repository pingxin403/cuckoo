/**
 * Realistic QPS Test - Measure actual single machine performance
 * 
 * This test uses existing short codes to measure realistic redirect performance
 */

import http from 'k6/http';
import { check } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const latencyTrend = new Trend('request_latency');
const successfulRedirects = new Counter('successful_redirects');
const cacheHits = new Counter('cache_hits_estimated');

// Test configuration - Ramp up to find max QPS
export const options = {
  stages: [
    { duration: '30s', target: 100 },   // Warm up
    { duration: '1m', target: 500 },    // Ramp to 500 VUs
    { duration: '2m', target: 1000 },   // Ramp to 1000 VUs
    { duration: '2m', target: 1000 },   // Hold at 1000 VUs
    { duration: '30s', target: 0 },     // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<20', 'p(99)<50'], // P95 < 20ms, P99 < 50ms
    'error_rate': ['rate<0.05'], // Error rate < 5%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Use the short codes we created earlier
const EXISTING_CODES = ['ncll0yl', 'LqhWmMl', '8eOIL5Z', '0UCIIQf', 'oslpgO2', 'YGuviUI', 'test001', 'test002', 'test003', 'test004', 'test005'];

export function setup() {
  console.log('========================================');
  console.log('Realistic QPS Test');
  console.log('========================================');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Test: Ramp up to 1000 VUs`);
  console.log(`Duration: 6 minutes total`);
  console.log('');
  
  // Verify codes exist
  let validCodes = [];
  for (const code of EXISTING_CODES) {
    const res = http.get(`${BASE_URL}/${code}`, { redirects: 0 });
    if (res.status === 302) {
      validCodes.push(code);
      console.log(`✓ ${code}: Valid (302)`);
    } else {
      console.log(`✗ ${code}: Invalid (${res.status})`);
    }
  }
  
  console.log('');
  console.log(`Valid codes: ${validCodes.length}/${EXISTING_CODES.length}`);
  console.log('');
  
  return { codes: validCodes.length > 0 ? validCodes : EXISTING_CODES };
}

export default function(data) {
  const startTime = Date.now();
  
  // Pick a random code (weighted towards first few for cache hits)
  const index = Math.random() < 0.8 
    ? Math.floor(Math.random() * Math.min(5, data.codes.length))  // 80% hit first 5 codes
    : Math.floor(Math.random() * data.codes.length);               // 20% hit any code
  
  const code = data.codes[index];
  
  // Test redirect (don't follow)
  const res = http.get(`${BASE_URL}/${code}`, {
    redirects: 0,
    tags: { type: 'redirect' },
  });
  
  const success = check(res, {
    'status is 302': (r) => r.status === 302,
    'has location header': (r) => r.headers['Location'] !== undefined,
    'response time < 50ms': (r) => r.timings.duration < 50,
  });
  
  if (res.status === 302) {
    successfulRedirects.add(1);
    
    // Estimate cache hits by latency
    if (res.timings.duration < 5) {
      cacheHits.add(1);
    }
  }
  
  errorRate.add(!success);
  
  const latency = Date.now() - startTime;
  latencyTrend.add(latency);
}

export function handleSummary(data) {
  const totalRequests = data.metrics.http_reqs?.values?.count || 0;
  const duration = (data.state?.testRunDurationMs || 0) / 1000;
  const avgQPS = duration > 0 ? totalRequests / duration : 0;
  const successfulRedirects = data.metrics.successful_redirects?.values?.count || 0;
  const cacheHits = data.metrics.cache_hits_estimated?.values?.count || 0;
  const cacheHitRate = successfulRedirects > 0 ? (cacheHits / successfulRedirects * 100).toFixed(2) : '0.00';
  
  const p50 = data.metrics.http_req_duration?.values?.['p(50)'] || 0;
  const p95 = data.metrics.http_req_duration?.values?.['p(95)'] || 0;
  const p99 = data.metrics.http_req_duration?.values?.['p(99)'] || 0;
  
  console.log('');
  console.log('========================================');
  console.log('Test Results Summary');
  console.log('========================================');
  console.log(`Total Requests: ${totalRequests.toLocaleString()}`);
  console.log(`Duration: ${duration.toFixed(1)}s`);
  console.log(`Average QPS: ${avgQPS.toFixed(0)}`);
  console.log(`Successful Redirects: ${successfulRedirects.toLocaleString()}`);
  console.log(`Estimated Cache Hits: ${cacheHits.toLocaleString()} (${cacheHitRate}%)`);
  console.log('');
  console.log('Latency:');
  console.log(`  P50: ${p50.toFixed(2)}ms`);
  console.log(`  P95: ${p95.toFixed(2)}ms`);
  console.log(`  P99: ${p99.toFixed(2)}ms`);
  console.log('');
  console.log('========================================');
  
  return {
    'stdout': '', // Return empty to use console.log above
  };
}

export function teardown() {
  console.log('Test complete. See summary above.');
}
