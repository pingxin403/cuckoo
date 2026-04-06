import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

const createLatency = new Trend('create_latency');
const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '10s', target: 100 },
    { duration: '30s', target: 500 },
    { duration: '10s', target: 1000 },
    { duration: '20s', target: 10000 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    'create_latency': ['p(99)<50'],
    'errors': ['rate<0.01'],
  },
};

export default function () {
  const payload = JSON.stringify({
    long_url: `https://example.com/very/long/url/path/${Math.random()}/${Date.now()}`,
  });

  const res = http.post(`${BASE_URL}/api/v1/shortener`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  createLatency.add(res.timings.duration);

  const ok = check(res, {
    'create succeeds': (r) => r.status >= 200 && r.status < 300,
    'returns short_code': (r) => r.json('short_code') !== undefined,
  });

  errorRate.add(!ok);
  sleep(0.1);
}