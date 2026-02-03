/**
 * Sustained Load Test - 100K QPS for 10 minutes
 * 
 * This test validates that the shortener service can handle
 * sustained high load with Redis optimizations enabled.
 * 
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
const cacheHitRate = new Rate('cache_hit_rate');
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
    'http_req_duration{type:get}': ['p(99)<5'], // P99 < 5ms
    'cache_hit_rate': ['rate>0.95'], // Cache hit rate > 95%
    'error_rate': ['rate<0.001'], // Error rate < 0.1%
    'http_req_failed': ['rate<0.001'], // Request failure rate < 0.1%
  },
};

// Base URL - can be overridden via environment variable
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Pre-generated short codes for testing (simulate realistic traffic)
const SHORT_CODES = generateShortCodes(10000);

function generateShortCodes(count) {
  const codes = [];
  const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  for (let i = 0; i < count; i++) {
    let code = '';
    for (let j = 0; j < 7; j++) {
      code += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    codes.push(code);
  }
  return codes;
}

export function setup() {
  console.log('Starting sustained load test...');
  console.log(`Target: 100K QPS for 10 minutes`);
  console.log(`Base URL: ${BASE_URL}`);
  
  // Warm up: Create some short links
  console.log('Warming up cache...');
  for (let i = 0; i < 100; i++) {
    const payload = JSON.stringify({
      long_url: `https://example.com/page/${i}`,
      custom_code: SHORT_CODES[i],
    });
    
    http.post(`${BASE_URL}/api/v1/shorten`, payload, {
      headers: { 'Content-Type': 'application/json' },
    });
  }
  
  console.log('Warmup complete. Starting load test...');
  return { shortCodes: SHORT_CODES };
}

export default function(data) {
  const startTime = Date.now();
  
  // 80% GET requests (cache hits), 20% POST requests (cache misses)
  const isGet = Math.random() < 0.8;
  
  if (isGet) {
    // GET request - should hit cache
    const shortCode = data.shortCodes[Math.floor(Math.random() * 100)]; // Use first 100 codes for high hit rate
    const res = http.get(`${BASE_URL}/api/v1/${shortCode}`, {
      tags: { type: 'get' },
    });
    
    const success = check(res, {
      'status is 200 or 404': (r) => r.status === 200 || r.status === 404,
      'response time < 5ms': (r) => r.timings.duration < 5,
    });
    
    // Track cache hit (200 = hit, 404 = miss)
    cacheHitRate.add(res.status === 200);
    errorRate.add(!success);
    
  } else {
    // POST request - create new short link
    const randomId = Math.floor(Math.random() * 1000000);
    const payload = JSON.stringify({
      long_url: `https://example.com/page/${randomId}`,
    });
    
    const res = http.post(`${BASE_URL}/api/v1/shorten`, payload, {
      headers: { 'Content-Type': 'application/json' },
      tags: { type: 'post' },
    });
    
    const success = check(res, {
      'status is 200 or 201': (r) => r.status === 200 || r.status === 201,
      'response time < 10ms': (r) => r.timings.duration < 10,
    });
    
    errorRate.add(!success);
  }
  
  const latency = Date.now() - startTime;
  latencyTrend.add(latency);
  throughput.add(1);
}

export function teardown(data) {
  console.log('Sustained load test complete.');
  console.log('Check metrics for results.');
}
