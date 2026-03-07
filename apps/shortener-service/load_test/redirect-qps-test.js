/**
 * Redirect QPS Test - Test actual redirect performance
 * 
 * This test measures QPS for the main redirect operation with real short codes.
 * Uses HTTP redirect endpoint (GET /:code) which returns 302 status.
 * 
 * Test Strategy:
 * - Uses existing short codes from prepare-test-data.sh
 * - Weighted distribution: 80% hit first 5 codes (high cache hit rate)
 * - Measures actual redirect performance with realistic traffic pattern
 * 
 * Expected Performance (single machine):
 * - QPS: 150K-180K (with high cache hit rate)
 * - P99 Latency: < 5ms (L1 cache hits)
 * - Cache Hit Rate: > 90%
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const latencyTrend = new Trend('request_latency');
const successfulRedirects = new Counter('successful_redirects');
const cacheHitsEstimated = new Counter('cache_hits_estimated');

// Test configuration - Ramp up to find sustainable QPS
export const options = {
  stages: [
    { duration: '30s', target: 100 },   // Warm up
    { duration: '1m', target: 500 },    // Ramp to 500 VUs
    { duration: '2m', target: 1000 },   // Ramp to 1000 VUs
    { duration: '2m', target: 1000 },   // Hold at 1000 VUs
    { duration: '30s', target: 0 },     // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<10', 'p(99)<20'], // P95 < 10ms, P99 < 20ms
    'error_rate': ['rate<0.01'], // Error rate < 1%
    'http_req_failed': ['rate<0.01'], // Request failure rate < 1%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Use existing short codes from prepare-test-data.sh
const EXISTING_CODES = [
  'ncll0yl', 'LqhWmMl', '8eOIL5Z', '0UCIIQf', 'oslpgO2', 'YGuviUI',
  'test001', 'test002', 'test003', 'test004', 'test005'
];

export function setup() {
  console.log('========================================');
  console.log('Redirect QPS Test');
  console.log('========================================');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Test: Ramp up to 1000 VUs over 6 minutes`);
  console.log(`Testing HTTP redirect performance (GET /:code)`);
  console.log('');
  
  // Verify test codes exist
  let validCodes = [];
  console.log('Verifying short codes...');
  for (const code of EXISTING_CODES) {
    const res = http.get(`${BASE_URL}/${code}`, { redirects: 0 });
    if (res.status === 302) {
      validCodes.push(code);
      console.log(`✓ ${code}: Valid (302 → ${res.headers['Location']})`);
    } else {
      console.log(`✗ ${code}: Invalid (${res.status})`);
    }
  }
  
  console.log('');
  console.log(`Valid codes: ${validCodes.length}/${EXISTING_CODES.length}`);
  console.log('');
  
  if (validCodes.length === 0) {
    throw new Error('No valid short codes found. Run prepare-test-data.sh first.');
  }
  
  return { codes: validCodes };
}

export default function(data) {
  // Weighted distribution: 80% hit first 5 codes (simulate hot data)
  const index = Math.random() < 0.8 
    ? Math.floor(Math.random() * Math.min(5, data.codes.length))  // 80% hit first 5 codes
    : Math.floor(Math.random() * data.codes.length);               // 20% hit any code
  
  const code = data.codes[index];
  
  // Test redirect (don't follow redirects)
  const res = http.get(`${BASE_URL}/${code}`, {
    redirects: 0,
    tags: { type: 'redirect' },
  });
  
  const success = check(res, {
    'status is 302': (r) => r.status === 302,
    'has location header': (r) => r.headers['Location'] !== undefined,
    'response time < 20ms': (r) => r.timings.duration < 20,
  });
  
  if (res.status === 302) {
    successfulRedirects.add(1);
    
    // Estimate cache hits by latency (< 5ms likely L1/L2 cache hit)
    if (res.timings.duration < 5) {
      cacheHitsEstimated.add(1);
    }
  }
  
  errorRate.add(!success);
  latencyTrend.add(res.timings.duration);
  
  // Small sleep to avoid overwhelming the service
  sleep(0.001);
}

export function handleSummary(data) {
  const totalRequests = data.metrics.http_reqs?.values?.count || 0;
  const duration = (data.state?.testRunDurationMs || 0) / 1000;
  const avgQPS = duration > 0 ? totalRequests / duration : 0;
  const successfulRedirects = data.metrics.successful_redirects?.values?.count || 0;
  const cacheHits = data.metrics.cache_hits_estimated?.values?.count || 0;
  const cacheHitRate = successfulRedirects > 0 ? (cacheHits / successfulRedirects * 100).toFixed(2) : '0.00';
  const errorRate = data.metrics.error_rate?.values?.rate || 0;
  
  const p50 = data.metrics.http_req_duration?.values?.['p(50)'] || 0;
  const p95 = data.metrics.http_req_duration?.values?.['p(95)'] || 0;
  const p99 = data.metrics.http_req_duration?.values?.['p(99)'] || 0;
  
  console.log('');
  console.log('========================================');
  console.log('Redirect QPS Test Results');
  console.log('========================================');
  console.log(`Total Requests: ${totalRequests.toLocaleString()}`);
  console.log(`Duration: ${duration.toFixed(1)}s`);
  console.log(`Average QPS: ${avgQPS.toFixed(0)}`);
  console.log(`Successful Redirects: ${successfulRedirects.toLocaleString()}`);
  console.log(`Error Rate: ${(errorRate * 100).toFixed(2)}%`);
  console.log('');
  console.log('Cache Performance:');
  console.log(`  Estimated Cache Hits: ${cacheHits.toLocaleString()} (${cacheHitRate}%)`);
  console.log('');
  console.log('Latency:');
  console.log(`  P50: ${p50.toFixed(2)}ms`);
  console.log(`  P95: ${p95.toFixed(2)}ms`);
  console.log(`  P99: ${p99.toFixed(2)}ms`);
  console.log('');
  console.log('========================================');
  
  return {
    'stdout': '',
  };
}

export function teardown() {
  console.log('Redirect QPS test complete.');
}
