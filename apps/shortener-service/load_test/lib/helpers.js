/**
 * Helper Functions
 * 
 * Common utility functions for load tests.
 */

import { Counter, Rate, Trend } from 'k6/metrics';

/**
 * Create standard custom metrics
 * @returns {Object} Custom metrics
 */
export function createMetrics() {
  return {
    cacheHitsEstimated: new Counter('cache_hits_estimated'),
    errorRate: new Rate('error_rate'),
    latencyTrend: new Trend('request_latency'),
    successfulRedirects: new Counter('successful_redirects'),
    throughput: new Counter('requests_total'),
  };
}

/**
 * Estimate if request was a cache hit based on latency
 * @param {number} duration - Request duration in ms
 * @param {number} threshold - Cache hit threshold in ms (default: 5ms)
 * @returns {boolean} True if likely a cache hit
 */
export function isCacheHit(duration, threshold = 5) {
  return duration < threshold;
}

/**
 * Select item with weighted distribution
 * @param {Array} items - Array of items
 * @param {number} hotPercentage - Percentage of requests to hot items (0-1)
 * @param {number} hotCount - Number of hot items
 * @returns {*} Selected item
 */
export function selectWithWeight(items, hotPercentage = 0.8, hotCount = 5) {
  const isHot = Math.random() < hotPercentage;
  const index = isHot
    ? Math.floor(Math.random() * Math.min(hotCount, items.length))
    : Math.floor(Math.random() * items.length);
  
  return items[index];
}

/**
 * Format summary statistics
 * @param {Object} data - k6 summary data
 * @returns {string} Formatted summary
 */
export function formatSummary(data, testName) {
  const totalRequests = data.metrics.http_reqs?.values?.count || 0;
  const duration = (data.state?.testRunDurationMs || 0) / 1000;
  const avgQPS = duration > 0 ? totalRequests / duration : 0;
  
  const p50 = data.metrics.http_req_duration?.values?.['p(50)'] || 0;
  const p95 = data.metrics.http_req_duration?.values?.['p(95)'] || 0;
  const p99 = data.metrics.http_req_duration?.values?.['p(99)'] || 0;
  
  const failedRate = data.metrics.http_req_failed?.values?.rate || 0;
  
  let summary = '\n';
  summary += '========================================\n';
  summary += `${testName} Results\n`;
  summary += '========================================\n';
  summary += `Total Requests: ${totalRequests.toLocaleString()}\n`;
  summary += `Duration: ${duration.toFixed(1)}s\n`;
  summary += `Average QPS: ${avgQPS.toFixed(0)}\n`;
  summary += `Error Rate: ${(failedRate * 100).toFixed(3)}%\n`;
  summary += '\n';
  summary += 'Latency:\n';
  summary += `  P50: ${p50.toFixed(2)}ms\n`;
  summary += `  P95: ${p95.toFixed(2)}ms\n`;
  summary += `  P99: ${p99.toFixed(2)}ms\n`;
  summary += '\n';
  summary += '========================================\n';
  
  return summary;
}

/**
 * Validate test data
 * @param {Array} data - Test data array
 * @param {string} dataType - Type of data for error message
 * @throws {Error} If data is invalid
 */
export function validateTestData(data, dataType = 'test data') {
  if (!data || !Array.isArray(data) || data.length === 0) {
    throw new Error(`No valid ${dataType} found. Please prepare test data first.`);
  }
}
