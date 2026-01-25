// k6 Load Test: WebSocket Connection Load Test
// Tests 100K concurrent connections per Gateway node
// Validates: Requirements 6.1, 9.1

import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';

// Custom metrics
const connectionErrors = new Counter('connection_errors');
const connectionDuration = new Trend('connection_duration');
const messageLatency = new Trend('message_latency');
const connectionSuccess = new Rate('connection_success');

// Test configuration
export const options = {
  scenarios: {
    // Scenario 1: Ramp up to 100K connections
    ramp_up: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5m', target: 10000 },   // Ramp to 10K in 5 min
        { duration: '5m', target: 50000 },   // Ramp to 50K in 5 min
        { duration: '5m', target: 100000 },  // Ramp to 100K in 5 min
        { duration: '30m', target: 100000 }, // Hold 100K for 30 min
        { duration: '5m', target: 0 },       // Ramp down in 5 min
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    'connection_success': ['rate>0.95'],           // 95% connection success rate
    'connection_duration': ['p(95)<5000'],         // 95% connections under 5s
    'message_latency': ['p(99)<200'],              // P99 latency < 200ms
    'ws_connecting': ['p(95)<5000'],               // WebSocket connect time
    'ws_session_duration': ['p(95)>1800000'],      // Sessions last > 30 min
  },
};

// Environment variables
const GATEWAY_HOST = __ENV.GATEWAY_HOST || 'localhost';
const GATEWAY_PORT = __ENV.GATEWAY_PORT || '8080';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || 'test-token';

export default function () {
  const url = `ws://${GATEWAY_HOST}:${GATEWAY_PORT}/ws`;
  const params = {
    headers: {
      'Authorization': `Bearer ${AUTH_TOKEN}`,
    },
    tags: {
      scenario: 'connection_load',
    },
  };

  const startTime = Date.now();
  
  const res = ws.connect(url, params, function (socket) {
    const connectDuration = Date.now() - startTime;
    connectionDuration.add(connectDuration);
    connectionSuccess.add(1);

    // Send authentication message
    socket.on('open', function () {
      const authMsg = JSON.stringify({
        type: 'auth',
        token: AUTH_TOKEN,
        user_id: `user_${__VU}`,
        device_id: `device_${__VU}`,
      });
      socket.send(authMsg);
    });

    // Handle incoming messages
    socket.on('message', function (data) {
      const msg = JSON.parse(data);
      
      if (msg.type === 'auth_response') {
        check(msg, {
          'auth successful': (m) => m.success === true,
        });
      }
      
      if (msg.type === 'message') {
        const latency = Date.now() - msg.timestamp;
        messageLatency.add(latency);
      }
    });

    // Handle errors
    socket.on('error', function (e) {
      connectionErrors.add(1);
      console.error(`WebSocket error: ${e.error()}`);
    });

    // Send heartbeat every 30 seconds
    socket.setInterval(function () {
      const heartbeat = JSON.stringify({
        type: 'heartbeat',
        timestamp: Date.now(),
      });
      socket.send(heartbeat);
    }, 30000);

    // Keep connection alive for test duration
    socket.setTimeout(function () {
      socket.close();
    }, 1800000); // 30 minutes
  });

  check(res, {
    'connection established': (r) => r && r.status === 101,
  });

  if (!res || res.status !== 101) {
    connectionErrors.add(1);
    connectionSuccess.add(0);
  }
}

export function handleSummary(data) {
  return {
    'summary.json': JSON.stringify(data),
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;
  
  let summary = '\n';
  summary += `${indent}Connection Load Test Summary\n`;
  summary += `${indent}============================\n\n`;
  
  // Connection metrics
  summary += `${indent}Connections:\n`;
  summary += `${indent}  Total Attempts: ${data.metrics.connection_success.values.count}\n`;
  summary += `${indent}  Success Rate: ${(data.metrics.connection_success.values.rate * 100).toFixed(2)}%\n`;
  summary += `${indent}  Errors: ${data.metrics.connection_errors.values.count}\n\n`;
  
  // Duration metrics
  summary += `${indent}Connection Duration:\n`;
  summary += `${indent}  Min: ${data.metrics.connection_duration.values.min.toFixed(2)}ms\n`;
  summary += `${indent}  Avg: ${data.metrics.connection_duration.values.avg.toFixed(2)}ms\n`;
  summary += `${indent}  P95: ${data.metrics.connection_duration.values['p(95)'].toFixed(2)}ms\n`;
  summary += `${indent}  Max: ${data.metrics.connection_duration.values.max.toFixed(2)}ms\n\n`;
  
  // Message latency
  if (data.metrics.message_latency) {
    summary += `${indent}Message Latency:\n`;
    summary += `${indent}  P50: ${data.metrics.message_latency.values['p(50)'].toFixed(2)}ms\n`;
    summary += `${indent}  P95: ${data.metrics.message_latency.values['p(95)'].toFixed(2)}ms\n`;
    summary += `${indent}  P99: ${data.metrics.message_latency.values['p(99)'].toFixed(2)}ms\n\n`;
  }
  
  return summary;
}
