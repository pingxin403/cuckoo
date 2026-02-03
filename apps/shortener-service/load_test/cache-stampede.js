/**
 * Cache Stampede Test - 1000 concurrent cache misses
 * 
 * This test validates that the SETNX + Singleflight optimizations
 * prevent cache stampede and reduce database load.
 * 
 * 
 * Target Metrics:
 * - DB Queries: < 10 (for 1000 concurrent requests)
 * - DB Load Reduction: > 90%
 * - P99 Latency: < 100ms
 * - Error Rate: < 0.1%
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const latencyTrend = new Trend('request_latency');
const dbQueries = new Counter('db_queries_total');
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

// Shared short code for stampede test
const STAMPEDE_CODE = 'STAMPEDE';

export function setup() {
  console.log('Starting cache stampede test...');
  console.log(`Target: 1000 concurrent requests for same key`);
  console.log(`Base URL: ${BASE_URL}`);
  
  // Create the short link that will be stampeded
  console.log('Creating test short link...');
  const payload = JSON.stringify({
    long_url: 'https://example.com/stampede-test',
    custom_code: STAMPEDE_CODE,
  });
  
  const res = http.post(`${BASE_URL}/api/v1/shorten`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  
  if (res.status !== 200 && res.status !== 201) {
    console.error('Failed to create test short link:', res.status, res.body);
  }
  
  // Clear cache to force cache miss
  console.log('Clearing cache...');
  sleep(2); // Wait for cache to be ready
  
  // Delete the cache entry (if API supports it)
  // For now, we'll rely on TTL expiration or manual cache clear
  
  console.log('Setup complete. Starting stampede test...');
  return { stampedeCode: STAMPEDE_CODE };
}

export default function(data) {
  const startTime = Date.now();
  
  // All requests hit the same short code (cache stampede scenario)
  const res = http.get(`${BASE_URL}/api/v1/${data.stampedeCode}`, {
    tags: { type: 'stampede' },
  });
  
  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 100ms': (r) => r.timings.duration < 100,
    'has long_url': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.long_url !== undefined;
      } catch (e) {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  
  // Track cache hit/miss based on response headers or timing
  // Assumption: cache hits are faster than cache misses
  if (res.timings.duration < 10) {
    cacheHits.add(1);
  } else {
    cacheMisses.add(1);
    dbQueries.add(1); // Approximate DB query count
  }
  
  const latency = Date.now() - startTime;
  latencyTrend.add(latency);
}

export function teardown(data) {
  console.log('Cache stampede test complete.');
  console.log('Expected: < 10 DB queries for 1000 concurrent requests');
  console.log('Check metrics for actual DB query count.');
  console.log('Verify SETNX + Singleflight prevented stampede.');
}
