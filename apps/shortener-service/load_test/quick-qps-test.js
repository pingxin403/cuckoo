/**
 * Quick QPS Test - Fast performance validation
 * 
 * This test provides a quick way to measure actual throughput
 * and validate service performance after changes.
 * 
 * Test Strategy:
 * - Uses HTTP redirect endpoint (GET /:code)
 * - Short duration (30 seconds)
 * - Moderate load to quickly identify issues
 * 
 * Expected Performance (single machine):
 * - QPS: 10K-20K (moderate load)
 * - P99 Latency: < 10ms
 * - Error Rate: < 1%
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const latencyTrend = new Trend('request_latency');
const successfulRedirects = new Counter('successful_redirects');
const cacheHitsEstimated = new Counter('cache_hits_estimated');

// Test configuration - Moderate constant load
export const options = {
  scenarios: {
    constant_load: {
      executor: 'constant-vus',
      vus: 100,
      duration: '30s',
    },
  },
  thresholds: {
    'http_req_duration': ['p(99)<10'], // P99 < 10ms
    'error_rate': ['rate<0.01'], // Error rate < 1%
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
  console.log('Quick QPS Test');
  console.log('========================================');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Test: 100 VUs for 30 seconds`);
  console.log('');
  
  // Verify health endpoint
  const health = http.get(`${BASE_URL}/health`);
  console.log(`Health check: ${health.status}`);
  
  // Verify codes exist
  let validCodes = [];
  console.log('Verifying short codes...');
  for (const code of EXISTING_CODES) {
    const res = http.get(`${BASE_URL}/${code}`, { redirects: 0 });
    if (res.status === 302) {
      validCodes.push(code);
    }
  }
  
  console.log(`Valid codes: ${validCodes.length}/${EXISTING_CODES.length}`);
  console.log('');
  
  if (validCodes.length === 0) {
    throw new Error('No valid short codes found. Run prepare-test-data.sh first.');
  }
  
  return { codes: validCodes };
}

export default function(data) {
  // Pick a random code
  const code = data.codes[Math.floor(Math.random() * data.codes.length)];
  
  // Test redirect (don't follow redirects)
  const res = http.get(`${BASE_URL}/${code}`, {
    redirects: 0,
    tags: { type: 'redirect' },
  });
  
  const success = check(res, {
    'status is 302': (r) => r.status === 302,
    'has location header': (r) => r.headers['Location'] !== undefined,
    'response time < 10ms': (r) => r.timings.duration < 10,
  });
  
  if (res.status === 302) {
    successfulRedirects.add(1);
    
    // Estimate cache hits by latency
    if (res.timings.duration < 5) {
      cacheHitsEstimated.add(1);
    }
  }
  
  errorRate.add(!success);
  latencyTrend.add(res.timings.duration);
  
  // Small sleep to avoid overwhelming the service
  sleep(0.01);
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
  console.log('Quick QPS Test Results');
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
  console.log('Quick QPS test complete.');
}
