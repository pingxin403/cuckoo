/**
 * Spike Test - 0 → 500K QPS in 1 minute
 * 
 * This test validates that the shortener service can handle
 * sudden traffic spikes with Redis optimizations enabled.
 * 
 * 
 * Target Metrics:
 * - Peak Throughput: 500K QPS
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

// Test configuration
export const options = {
  scenarios: {
    spike_test: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      preAllocatedVUs: 2000,
      maxVUs: 5000,
      stages: [
        { duration: '1m', target: 500000 }, // Ramp up to 500K QPS in 1 minute
        { duration: '2m', target: 500000 }, // Hold at 500K QPS for 2 minutes
        { duration: '1m', target: 0 },      // Ramp down to 0 in 1 minute
      ],
    },
  },
  thresholds: {
    'http_req_duration{type:get}': ['p(99)<10'], // P99 < 10ms during spike
    'error_rate': ['rate<0.01'], // Error rate < 1%
    'http_req_failed': ['rate<0.01'], // Request failure rate < 1%
  },
};

// Base URL
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Pre-generated short codes
const SHORT_CODES = generateShortCodes(1000);

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
  console.log('Starting spike test...');
  console.log(`Target: 0 → 500K QPS in 1 minute`);
  console.log(`Base URL: ${BASE_URL}`);
  
  // Warm up: Create some short links
  console.log('Warming up cache...');
  for (let i = 0; i < 100; i++) {
    const payload = JSON.stringify({
      long_url: `https://example.com/spike/${i}`,
      custom_code: SHORT_CODES[i],
    });
    
    http.post(`${BASE_URL}/api/v1/shorten`, payload, {
      headers: { 'Content-Type': 'application/json' },
    });
  }
  
  console.log('Warmup complete. Starting spike test...');
  return { shortCodes: SHORT_CODES };
}

export default function(data) {
  const startTime = Date.now();
  
  // 90% GET requests (mostly cache hits during spike)
  const isGet = Math.random() < 0.9;
  
  if (isGet) {
    // GET request - should hit cache
    const shortCode = data.shortCodes[Math.floor(Math.random() * 100)];
    const res = http.get(`${BASE_URL}/api/v1/${shortCode}`, {
      tags: { type: 'get' },
    });
    
    const success = check(res, {
      'status is 200 or 404': (r) => r.status === 200 || r.status === 404,
      'response time < 10ms': (r) => r.timings.duration < 10,
    });
    
    errorRate.add(!success);
    
  } else {
    // POST request - create new short link
    const randomId = Math.floor(Math.random() * 1000000);
    const payload = JSON.stringify({
      long_url: `https://example.com/spike/${randomId}`,
    });
    
    const res = http.post(`${BASE_URL}/api/v1/shorten`, payload, {
      headers: { 'Content-Type': 'application/json' },
      tags: { type: 'post' },
    });
    
    const success = check(res, {
      'status is 200 or 201': (r) => r.status === 200 || r.status === 201,
      'response time < 20ms': (r) => r.timings.duration < 20,
    });
    
    errorRate.add(!success);
  }
  
  const latency = Date.now() - startTime;
  latencyTrend.add(latency);
  throughput.add(1);
}

export function teardown(data) {
  console.log('Spike test complete.');
  console.log('Check metrics for results.');
  console.log('Verify system recovery time after spike.');
}
