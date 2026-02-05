/**
 * Sustained Load Test - 100K QPS for 10 minutes
 * 
 * This test validates that the shortener service can handle
 * sustained high load with Redis optimizations enabled.
 * 
 * Test Strategy:
 * - Uses HTTP redirect endpoint (GET /:code)
 * - 100% read operations (redirect requests)
 * - Uses existing short codes with weighted distribution
 * - Measures sustained performance over 10 minutes
 * 
 * Target Metrics:
 * - Throughput: 100K QPS sustained
 * - P99 Latency: < 5ms
 * - Cache Hit Rate: > 95%
 * - Error Rate: < 0.1%
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const cacheHitsEstimated = new Counter('cache_hits_estimated');
const errorRate = new Rate('error_rate');
const latencyTrend = new Trend('request_latency');
const throughput = new Counter('requests_total');

// Test configuration
export const options = {
  scenarios: {
    sustained_load: {
      executor: 'constant-arrival-rate',
      rate: 100000, // 100K requests per second
      timeUnit: '1s',
      duration: '10m', // 10 minutes
      preAllocatedVUs: 1000, // Pre-allocate VUs
      maxVUs: 2000, // Maximum VUs if needed
    },
  },
  thresholds: {
    'http_req_duration': ['p(99)<5'], // P99 < 5ms
    'error_rate': ['rate<0.001'], // Error rate < 0.1%
    'http_req_failed': ['rate<0.001'], // Request failure rate < 0.1%
  },
};

// Base URL - can be overridden via environment variable
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Use existing short codes from prepare-test-data.sh
const EXISTING_CODES = [
  'ncll0yl', 'LqhWmMl', '8eOIL5Z', '0UCIIQf', 'oslpgO2', 'YGuviUI',
  'test001', 'test002', 'test003', 'test004', 'test005'
];

export function setup() {
  console.log('========================================');
  console.log('Sustained Load Test');
  console.log('========================================');
  console.log(`Target: 100K QPS for 10 minutes`);
  console.log(`Base URL: ${BASE_URL}`);
  console.log('');
  
  // Verify codes exist
  console.log('Verifying short codes...');
  let validCodes = [];
  for (const code of EXISTING_CODES) {
    const res = http.get(`${BASE_URL}/${code}`, { redirects: 0 });
    if (res.status === 302) {
      validCodes.push(code);
      console.log(`✓ ${code}: Valid`);
    }
  }
  
  console.log('');
  console.log(`Valid codes: ${validCodes.length}/${EXISTING_CODES.length}`);
  console.log('');
  
  if (validCodes.length === 0) {
    throw new Error('No valid short codes found. Run prepare-test-data.sh first.');
  }
  
  console.log('Starting sustained load test...');
  return { codes: validCodes };
}

export default function(data) {
  // Weighted distribution: 90% hit first 5 codes (high cache hit rate)
  const index = Math.random() < 0.9
    ? Math.floor(Math.random() * Math.min(5, data.codes.length))  // 90% hit first 5 codes
    : Math.floor(Math.random() * data.codes.length);               // 10% hit any code
  
  const code = data.codes[index];
  
  // GET request - redirect
  const res = http.get(`${BASE_URL}/${code}`, {
    redirects: 0,
    tags: { type: 'redirect' },
  });
  
  const success = check(res, {
    'status is 302': (r) => r.status === 302,
    'response time < 5ms': (r) => r.timings.duration < 5,
  });
  
  // Track cache hit (< 2ms likely L1 cache hit)
  if (res.timings.duration < 2) {
    cacheHitsEstimated.add(1);
  }
  
  errorRate.add(!success);
  latencyTrend.add(res.timings.duration);
  throughput.add(1);
}

export function handleSummary(data) {
  const totalRequests = data.metrics.http_reqs?.values?.count || 0;
  const duration = (data.state?.testRunDurationMs || 0) / 1000;
  const avgQPS = duration > 0 ? totalRequests / duration : 0;
  const cacheHits = data.metrics.cache_hits_estimated?.values?.count || 0;
  const cacheHitRate = totalRequests > 0 ? (cacheHits / totalRequests * 100).toFixed(2) : '0.00';
  const errorRate = data.metrics.error_rate?.values?.rate || 0;
  
  const p50 = data.metrics.http_req_duration?.values?.['p(50)'] || 0;
  const p95 = data.metrics.http_req_duration?.values?.['p(95)'] || 0;
  const p99 = data.metrics.http_req_duration?.values?.['p(99)'] || 0;
  
  console.log('');
  console.log('========================================');
  console.log('Sustained Load Test Results');
  console.log('========================================');
  console.log(`Total Requests: ${totalRequests.toLocaleString()}`);
  console.log(`Duration: ${duration.toFixed(1)}s`);
  console.log(`Average QPS: ${avgQPS.toFixed(0)}`);
  console.log(`Error Rate: ${(errorRate * 100).toFixed(3)}%`);
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
  console.log('Sustained load test complete.');
}
