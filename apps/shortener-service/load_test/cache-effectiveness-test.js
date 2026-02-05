import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const cacheHitRate = new Rate('cache_hit_rate');
const dbQueryCount = new Counter('db_query_count');
const cacheQueryCount = new Counter('cache_query_count');
const responseTime = new Trend('response_time');
const errorRate = new Rate('error_rate');

// Test configuration
export const options = {
  scenarios: {
    // Scenario 1: Cold cache - first access to URLs
    cold_cache: {
      executor: 'constant-vus',
      vus: 50,
      duration: '30s',
      startTime: '0s',
      tags: { scenario: 'cold_cache' },
    },
    // Scenario 2: Warm cache - repeated access to same URLs
    warm_cache: {
      executor: 'constant-vus',
      vus: 100,
      duration: '60s',
      startTime: '35s',
      tags: { scenario: 'warm_cache' },
    },
    // Scenario 3: Hot cache - high concurrency on popular URLs
    hot_cache: {
      executor: 'ramping-vus',
      startVUs: 50,
      stages: [
        { duration: '20s', target: 200 },
        { duration: '40s', target: 200 },
        { duration: '20s', target: 50 },
      ],
      startTime: '100s',
      tags: { scenario: 'hot_cache' },
    },
  },
  thresholds: {
    'http_req_duration{scenario:cold_cache}': ['p(95)<500'],
    'http_req_duration{scenario:warm_cache}': ['p(95)<100'], // Should be much faster with cache
    'http_req_duration{scenario:hot_cache}': ['p(95)<50'],   // Should be very fast with hot cache
    'error_rate': ['rate<0.01'],
    'http_req_failed': ['rate<0.01'],
  },
};

// Base URL for the service
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';
const GRPC_ADDR = __ENV.GRPC_ADDR || 'localhost:9092';

// Test data - URLs to create and access
const TEST_URLS = [
  'https://example.com/page1',
  'https://example.com/page2',
  'https://example.com/page3',
  'https://example.com/page4',
  'https://example.com/page5',
  'https://github.com/kubernetes/kubernetes',
  'https://redis.io/docs/manual/patterns/',
  'https://www.postgresql.org/docs/current/',
  'https://golang.org/doc/effective_go',
  'https://reactjs.org/docs/getting-started.html',
];

// Popular URLs that will be accessed frequently (80/20 rule)
const POPULAR_URLS = TEST_URLS.slice(0, 3);
const REGULAR_URLS = TEST_URLS.slice(3);

// Store created short codes
let shortCodes = [];

export function setup() {
  console.log('Setting up test data...');
  
  // Create short links for all test URLs
  const codes = [];
  
  for (const url of TEST_URLS) {
    const payload = JSON.stringify({
      long_url: url,
      custom_code: '',
      expires_at: '',
    });
    
    const params = {
      headers: {
        'Content-Type': 'application/json',
      },
    };
    
    const res = http.post(`${BASE_URL}/api/v1/shorten`, payload, params);
    
    if (res.status === 200 || res.status === 201) {
      const body = JSON.parse(res.body);
      codes.push(body.short_code);
      console.log(`Created short code: ${body.short_code} for ${url}`);
    } else {
      console.error(`Failed to create short link for ${url}: ${res.status}`);
    }
  }
  
  console.log(`Setup complete. Created ${codes.length} short codes.`);
  return { shortCodes: codes };
}

export default function(data) {
  const scenario = __ENV.SCENARIO || 'warm_cache';
  
  if (!data.shortCodes || data.shortCodes.length === 0) {
    console.error('No short codes available for testing');
    return;
  }
  
  let shortCode;
  
  // Select URL based on scenario
  if (scenario === 'cold_cache') {
    // Cold cache: access URLs randomly
    shortCode = data.shortCodes[Math.floor(Math.random() * data.shortCodes.length)];
  } else if (scenario === 'warm_cache') {
    // Warm cache: 80% popular URLs, 20% regular URLs
    if (Math.random() < 0.8) {
      shortCode = data.shortCodes[Math.floor(Math.random() * 3)]; // Popular URLs
    } else {
      shortCode = data.shortCodes[3 + Math.floor(Math.random() * (data.shortCodes.length - 3))];
    }
  } else {
    // Hot cache: mostly access the most popular URL
    if (Math.random() < 0.9) {
      shortCode = data.shortCodes[0]; // Hottest URL
    } else {
      shortCode = data.shortCodes[Math.floor(Math.random() * data.shortCodes.length)];
    }
  }
  
  // Make redirect request
  const startTime = new Date().getTime();
  const res = http.get(`${BASE_URL}/${shortCode}`, {
    redirects: 0, // Don't follow redirects
    tags: { name: 'redirect' },
  });
  const endTime = new Date().getTime();
  
  // Record response time
  responseTime.add(endTime - startTime);
  
  // Check response
  const success = check(res, {
    'status is 301 or 302': (r) => r.status === 301 || r.status === 302,
    'has location header': (r) => r.headers['Location'] !== undefined,
  });
  
  if (!success) {
    errorRate.add(1);
    console.error(`Request failed: ${res.status} ${res.body}`);
  } else {
    errorRate.add(0);
    
    // Try to detect cache hit from response time
    // Cache hits should be < 10ms, DB queries > 50ms
    if (endTime - startTime < 10) {
      cacheHitRate.add(1);
      cacheQueryCount.add(1);
    } else {
      cacheHitRate.add(0);
      dbQueryCount.add(1);
    }
  }
  
  // Small delay between requests
  sleep(0.1);
}

export function teardown(data) {
  console.log('\n=== Cache Effectiveness Test Results ===');
  console.log(`Total short codes tested: ${data.shortCodes.length}`);
  console.log('\nExpected behavior:');
  console.log('- Cold cache: High DB queries, slower response times');
  console.log('- Warm cache: Mixed cache hits, faster response times');
  console.log('- Hot cache: Very high cache hit rate, very fast response times');
  console.log('\nCheck the metrics above to verify cache effectiveness.');
}
