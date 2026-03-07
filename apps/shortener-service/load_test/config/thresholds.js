/**
 * Threshold Configuration
 * 
 * Defines pass/fail criteria for tests.
 * Usage: import { getThresholds } from './config/thresholds.js';
 */

export const CommonThresholds = {
  // HTTP request failure rate < 1%
  http_req_failed: ['rate<0.01'],
  
  // Request duration thresholds
  http_req_duration: ['p(95)<200', 'p(99)<500'],
};

export const StrictThresholds = {
  // HTTP request failure rate < 0.1%
  http_req_failed: ['rate<0.001'],
  
  // Stricter latency requirements
  http_req_duration: ['p(95)<100', 'p(99)<200', 'p(99.9)<500'],
};

export const RelaxedThresholds = {
  // HTTP request failure rate < 5%
  http_req_failed: ['rate<0.05'],
  
  // More lenient latency requirements
  http_req_duration: ['p(95)<500', 'p(99)<1000'],
};

// Service-specific thresholds
export const RedirectThresholds = {
  http_req_failed: ['rate<0.01'],
  http_req_duration: ['p(95)<10', 'p(99)<20'],
  
  // Custom metrics
  cache_hits_estimated: ['count>0'],
  successful_redirects: ['count>0'],
};

export const CreateLinkThresholds = {
  http_req_failed: ['rate<0.01'],
  http_req_duration: ['p(95)<100', 'p(99)<200'],
};

/**
 * Get threshold configuration
 * @param {string} type - Threshold type (common, strict, relaxed, redirect, create)
 * @returns {Object} Threshold configuration
 */
export function getThresholds(type = 'common') {
  const thresholds = {
    common: CommonThresholds,
    strict: StrictThresholds,
    relaxed: RelaxedThresholds,
    redirect: RedirectThresholds,
    create: CreateLinkThresholds,
  };
  
  return thresholds[type] || CommonThresholds;
}
