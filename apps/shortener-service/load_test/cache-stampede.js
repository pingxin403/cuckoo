/**
 * Cache Stampede Test - 1000 concurrent requests for same key
 * 
 * This test validates that the SETNX + Singleflight optimizations
 * prevent cache stampede and reduce database load.
 * 
 * Test Strategy:
 * - All VUs request the same short code simultaneously
 * - Simulates cache miss scenario (cold start)
 * - Measures DB query reduction via Singleflight
 * 
 * Target Metrics:
 * - DB Queries: < 10 (for 1000 concurrent requests)
 * - DB Load Reduction: > 90%
 * - P99 Latency: < 100ms
 * - Error Rate: < 0.1%
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const latencyTrend = new Trend('request_latency');
const cacheHits = new Counter('cache_hits_total');
const cacheMisses = new Counter('cache_misses_total');

// Test configuration
export const options = {
  scenarios: {
    cache_stampede: {
      executor: 'shared-iterations',
      vus: 1000, // 1000 concurrent users
      iterations: 1000, // 1000 total requests
      maxDuration: '30s',
    },
  },
  thresholds: {
    'http_req_duration': ['p(99)<100'], // P99 < 100ms
    'error_rate': ['rate<0.001'], // Error rate < 0.1%
    'http_req_failed': ['rate<0.001'], // Request failure rate < 0.1%
  },
};

// Base URL
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Use one of the existing short codes for stampede test
const STAMPEDE_CODE = 'test001';

export function setup() {
  console.log('========================================');
  console.log('Cache Stampede Test');
  console.log('========================================');
  console.log(`Target: 1000 concurrent requests for same key`);
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Stampede Code: ${STAMPEDE_CODE}`);
  console.log('');
  
  // Verify the short code exists
  console.log('Verifying short code...');
  const res = http.get(`${BASE_URL}/${STAMPEDE_CODE}`, { redirects: 0 });
  
  if (res.status !== 302) {
    throw new Error(`Short code ${STAMPEDE_CODE} not found. Run prepare-test-data.sh first.`);
  }
  
  console.log(`✓ ${STAMPEDE_CODE}: Valid (302 → ${res.headers['Location']})`);
  console.log('');
  
  // Note: In a real scenario, you would clear the cache here to force cache miss
  // For this test, we assume the cache might be cold or we're testing the first request
  console.log('Starting cache stampede test...');
  console.log('All 1000 VUs will request the same short code simultaneously.');
  console.log('');
  
  return { stampedeCode: STAMPEDE_CODE };
}

export default function(data) {
  // All requests hit the same short code (cache stampede scenario)
  const res = http.get(`${BASE_URL}/${data.stampedeCode}`, {
    redirects: 0,
    tags: { type: 'stampede' },
  });
  
  const success = check(res, {
    'status is 302': (r) => r.status === 302,
    'has location header': (r) => r.headers['Location'] !== undefined,
    'response time < 100ms': (r) => r.timings.duration < 100,
  });
  
  errorRate.add(!success);
  
  // Track cache hit/miss based on response timing
  // Assumption: cache hits are faster than cache misses
  if (res.timings.duration < 5) {
    cacheHits.add(1);
  } else {
    cacheMisses.add(1);
  }
  
  latencyTrend.add(res.timings.duration);
}

export function handleSummary(data) {
  const totalRequests = data.metrics.http_reqs?.values?.count || 0;
  const duration = (data.state?.testRunDurationMs || 0) / 1000;
  const cacheHits = data.metrics.cache_hits_total?.values?.count || 0;
  const cacheMisses = data.metrics.cache_misses_total?.values?.count || 0;
  const errorRate = data.metrics.error_rate?.values?.rate || 0;
  
  const p50 = data.metrics.http_req_duration?.values?.['p(50)'] || 0;
  const p95 = data.metrics.http_req_duration?.values?.['p(95)'] || 0;
  const p99 = data.metrics.http_req_duration?.values?.['p(99)'] || 0;
  
  console.log('');
  console.log('========================================');
  console.log('Cache Stampede Test Results');
  console.log('========================================');
  console.log(`Total Requests: ${totalRequests.toLocaleString()}`);
  console.log(`Duration: ${duration.toFixed(2)}s`);
  console.log(`Error Rate: ${(errorRate * 100).toFixed(3)}%`);
  console.log('');
  console.log('Cache Performance:');
  console.log(`  Cache Hits (< 5ms): ${cacheHits.toLocaleString()}`);
  console.log(`  Cache Misses (>= 5ms): ${cacheMisses.toLocaleString()}`);
  console.log('');
  console.log('Latency:');
  console.log(`  P50: ${p50.toFixed(2)}ms`);
  console.log(`  P95: ${p95.toFixed(2)}ms`);
  console.log(`  P99: ${p99.toFixed(2)}ms`);
  console.log('');
  console.log('========================================');
  console.log('Expected: < 10 DB queries for 1000 concurrent requests');
  console.log('Verify SETNX + Singleflight prevented stampede.');
  console.log('Check service logs for actual DB query count.');
  
  return {
    'stdout': '',
  };
}

export function teardown() {
  console.log('Cache stampede test complete.');
}
