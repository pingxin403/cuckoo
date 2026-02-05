/**
 * Spike Test - 0 → 200K QPS in 1 minute
 * 
 * This test validates that the shortener service can handle
 * sudden traffic spikes with Redis optimizations enabled.
 * 
 * Test Strategy:
 * - Uses HTTP redirect endpoint (GET /:code)
 * - Rapid ramp up to simulate traffic spike
 * - Tests system stability and recovery
 * 
 * Target Metrics:
 * - Peak Throughput: 200K QPS
 * - P99 Latency: < 10ms (during spike)
 * - Error Rate: < 1%
 * - System Recovery: < 30s after spike
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const latencyTrend = new Trend('request_latency');
const throughput = new Counter('requests_total');
const cacheHitsEstimated = new Counter('cache_hits_estimated');

// Test configuration
export const options = {
  stages: [
    { duration: '1m', target: 2000 },   // Ramp up to 2000 VUs in 1 minute (spike)
    { duration: '2m', target: 2000 },   // Hold at 2000 VUs for 2 minutes
    { duration: '1m', target: 0 },      // Ramp down to 0 in 1 minute
  ],
  thresholds: {
    'http_req_duration': ['p(99)<10'], // P99 < 10ms during spike
    'error_rate': ['rate<0.01'], // Error rate < 1%
    'http_req_failed': ['rate<0.01'], // Request failure rate < 1%
  },
};

// Base URL
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Use existing short codes from prepare-test-data.sh
const EXISTING_CODES = [
  'ncll0yl', 'LqhWmMl', '8eOIL5Z', '0UCIIQf', 'oslpgO2', 'YGuviUI',
  'test001', 'test002', 'test003', 'test004', 'test005'
];

export function setup() {
  console.log('========================================');
  console.log('Spike Test');
  console.log('========================================');
  console.log(`Target: 0 → 200K QPS in 1 minute`);
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
  
  console.log('Starting spike test...');
  return { codes: validCodes };
}

export default function(data) {
  // Weighted distribution: 95% hit first 5 codes (mostly cache hits during spike)
  const index = Math.random() < 0.95
    ? Math.floor(Math.random() * Math.min(5, data.codes.length))  // 95% hit first 5 codes
    : Math.floor(Math.random() * data.codes.length);               // 5% hit any code
  
  const code = data.codes[index];
  
  // GET request - redirect
  const res = http.get(`${BASE_URL}/${code}`, {
    redirects: 0,
    tags: { type: 'redirect' },
  });
  
  const success = check(res, {
    'status is 302': (r) => r.status === 302,
    'response time < 10ms': (r) => r.timings.duration < 10,
  });
  
  if (res.status === 302 && res.timings.duration < 5) {
    cacheHitsEstimated.add(1);
  }
  
  errorRate.add(!success);
  latencyTrend.add(res.timings.duration);
  throughput.add(1);
  
  // Small sleep to avoid overwhelming the service
  sleep(0.001);
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
  console.log('Spike Test Results');
  console.log('========================================');
  console.log(`Total Requests: ${totalRequests.toLocaleString()}`);
  console.log(`Duration: ${duration.toFixed(1)}s`);
  console.log(`Average QPS: ${avgQPS.toFixed(0)}`);
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
  console.log('Verify system recovery time after spike.');
  
  return {
    'stdout': '',
  };
}

export function teardown() {
  console.log('Spike test complete.');
}
