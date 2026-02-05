/**
 * Redirect Performance Test
 * 
 * Tests the redirect endpoint performance with configurable
 * environment and workload settings.
 * 
 * Usage:
 *   k6 run tests/redirect-test.js
 *   k6 run -e ENVIRONMENT=staging -e WORKLOAD=load tests/redirect-test.js
 *   k6 run -e ENVIRONMENT=prod -e WORKLOAD=stress tests/redirect-test.js
 */

import { getEnvironment } from '../config/environments.js';
import { getWorkload } from '../config/workloads.js';
import { getThresholds } from '../config/thresholds.js';
import { ShortenerAPIClient } from '../lib/api-client.js';
import {
  createMetrics,
  isCacheHit,
  selectWithWeight,
  formatSummary,
  validateTestData,
} from '../lib/helpers.js';

// Get configuration
const env = getEnvironment();
const workload = getWorkload();
const thresholds = getThresholds('redirect');

// Test configuration
export const options = {
  scenarios: {
    redirect_test: workload,
  },
  thresholds,
};

// Custom metrics
const metrics = createMetrics();

// Test data - use existing short codes
const EXISTING_CODES = [
  'ncll0yl', 'LqhWmMl', '8eOIL5Z', '0UCIIQf', 'oslpgO2', 'YGuviUI',
  'test001', 'test002', 'test003', 'test004', 'test005'
];

export function setup() {
  console.log('========================================');
  console.log('Redirect Performance Test');
  console.log('========================================');
  console.log(`Environment: ${__ENV.ENVIRONMENT || 'local'}`);
  console.log(`Workload: ${__ENV.WORKLOAD || 'smoke'}`);
  console.log(`Base URL: ${env.baseUrl}`);
  console.log('');
  
  // Verify test codes exist
  const client = new ShortenerAPIClient(env.baseUrl);
  const validCodes = [];
  
  console.log('Verifying short codes...');
  for (const code of EXISTING_CODES) {
    const result = client.getRedirect(code);
    if (result.success) {
      validCodes.push(code);
      console.log(`✓ ${code}: Valid (302 → ${result.location})`);
    } else {
      console.log(`✗ ${code}: Invalid (${result.response.status})`);
    }
  }
  
  console.log('');
  console.log(`Valid codes: ${validCodes.length}/${EXISTING_CODES.length}`);
  console.log('');
  
  validateTestData(validCodes, 'short codes');
  
  console.log('Starting test...');
  return { codes: validCodes };
}

export default function(data) {
  const client = new ShortenerAPIClient(env.baseUrl);
  
  // Weighted distribution: 80% hit first 5 codes (simulate hot data)
  const code = selectWithWeight(data.codes, 0.8, 5);
  
  // Execute redirect request
  const result = client.getRedirect(code);
  
  // Track metrics
  if (result.success) {
    metrics.successfulRedirects.add(1);
    
    // Estimate cache hits by latency
    if (isCacheHit(result.response.timings.duration)) {
      metrics.cacheHitsEstimated.add(1);
    }
  }
  
  metrics.errorRate.add(!result.success);
  metrics.latencyTrend.add(result.response.timings.duration);
  metrics.throughput.add(1);
}

export function handleSummary(data) {
  const summary = formatSummary(data, 'Redirect Performance Test');
  
  // Add cache performance
  const totalRequests = data.metrics.http_reqs?.values?.count || 0;
  const cacheHits = data.metrics.cache_hits_estimated?.values?.count || 0;
  const cacheHitRate = totalRequests > 0 
    ? (cacheHits / totalRequests * 100).toFixed(2) 
    : '0.00';
  
  console.log(summary);
  console.log('Cache Performance:');
  console.log(`  Estimated Cache Hits: ${cacheHits.toLocaleString()} (${cacheHitRate}%)`);
  console.log('');
  
  return {
    'stdout': '',
  };
}

export function teardown() {
  console.log('Test complete.');
}
