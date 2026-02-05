/**
 * API Client for Shortener Service
 * 
 * Encapsulates HTTP requests to the shortener service API.
 * Provides a clean interface for test scenarios.
 */

import http from 'k6/http';
import { check } from 'k6';

export class ShortenerAPIClient {
  constructor(baseUrl) {
    this.baseUrl = baseUrl;
    this.headers = {
      'Content-Type': 'application/json',
    };
  }

  /**
   * Get redirect for a short code
   * @param {string} code - Short code
   * @param {Object} options - Additional request options
   * @returns {Object} Response object with data and response
   */
  getRedirect(code, options = {}) {
    const url = `${this.baseUrl}/${code}`;
    const res = http.get(url, {
      redirects: 0,
      tags: { name: 'redirect', ...options.tags },
      ...options,
    });
    
    const success = check(res, {
      'redirect status is 302': (r) => r.status === 302,
      'has location header': (r) => r.headers['Location'] !== undefined,
    });
    
    return {
      success,
      location: res.headers['Location'],
      response: res,
    };
  }

  /**
   * Create a short link
   * @param {string} longUrl - Long URL to shorten
   * @param {string} customCode - Optional custom short code
   * @param {Object} options - Additional request options
   * @returns {Object} Response object with data and response
   */
  createLink(longUrl, customCode = null, options = {}) {
    const url = `${this.baseUrl}/api/v1/links`;
    const payload = {
      long_url: longUrl,
      ...(customCode && { custom_code: customCode }),
    };
    
    const res = http.post(url, JSON.stringify(payload), {
      headers: this.headers,
      tags: { name: 'create_link', ...options.tags },
      ...options,
    });
    
    let data = null;
    try {
      data = res.json();
    } catch (e) {
      // Response is not JSON
    }
    
    const success = check(res, {
      'create status is 201': (r) => r.status === 201,
      'has short_code': (r) => data && data.short_code !== undefined,
    });
    
    return {
      success,
      data,
      response: res,
    };
  }

  /**
   * Health check
   * @returns {Object} Response object
   */
  healthCheck() {
    const url = `${this.baseUrl}/health`;
    const res = http.get(url, {
      tags: { name: 'health_check' },
    });
    
    const success = check(res, {
      'health status is 200': (r) => r.status === 200,
    });
    
    return {
      success,
      response: res,
    };
  }
}
