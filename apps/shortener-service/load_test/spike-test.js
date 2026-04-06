import http from 'k6/http';
import { check } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

const latency = new Trend('spike_latency');
const errorRate = new Rate('spike_errors');

export const options = {
  stages: [
    { duration: '5s', target: 0 },
    { duration: '2s', target: 100000 },
    { duration: '5s', target: 100000 },
    { duration: '2s', target: 0 },
  ],
  thresholds: {
    'spike_latency': ['p(99)<100'],
    'spike_errors': ['rate<0.05'],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/abc1234`);
  latency.add(res.timings.duration);

  const ok = check(res, {
    'responds': (r) => r.status > 0,
  });

  errorRate.add(!ok);
}