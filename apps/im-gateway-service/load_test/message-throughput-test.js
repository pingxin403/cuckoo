// k6 Load Test: Message Throughput Test
// Tests message throughput (messages/sec)
// Validates: Requirements 1.1, 17.1

import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics
const messagesSent = new Counter('messages_sent');
const messagesReceived = new Counter('messages_received');
const messageLatency = new Trend('message_latency');
const messageThroughput = new Rate('message_throughput');
const sendErrors = new Counter('send_errors');

// Test configuration
export const options = {
  scenarios: {
    // Scenario: Sustained message throughput
    throughput_test: {
      executor: 'constant-vus',
      vus: 1000,                    // 1000 concurrent users
      duration: '10m',              // Run for 10 minutes
    },
  },
  thresholds: {
    'message_latency': ['p(99)<200'],              // P99 latency < 200ms
    'message_throughput': ['rate>0.99'],           // 99% message success
    'messages_sent': ['count>600000'],             // > 10K msg/sec for 10 min
  },
};

// Environment variables
const GATEWAY_HOST = __ENV.GATEWAY_HOST || 'localhost';
const GATEWAY_PORT = __ENV.GATEWAY_PORT || '8080';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || 'test-token';
const MESSAGE_SIZE = parseInt(__ENV.MESSAGE_SIZE || '1024'); // 1KB default

export default function () {
  const url = `ws://${GATEWAY_HOST}:${GATEWAY_PORT}/ws`;
  const userId = `user_${__VU}`;
  const deviceId = `device_${__VU}`;
  
  const params = {
    headers: {
      'Authorization': `Bearer ${AUTH_TOKEN}`,
    },
    tags: {
      scenario: 'throughput_test',
      user_id: userId,
    },
  };

  const pendingMessages = new Map();
  
  ws.connect(url, params, function (socket) {
    let authenticated = false;

    socket.on('open', function () {
      // Send authentication
      const authMsg = JSON.stringify({
        type: 'auth',
        token: AUTH_TOKEN,
        user_id: userId,
        device_id: deviceId,
      });
      socket.send(authMsg);
    });

    socket.on('message', function (data) {
      const msg = JSON.parse(data);
      
      if (msg.type === 'auth_response') {
        authenticated = msg.success;
        check(msg, {
          'auth successful': (m) => m.success === true,
        });
      }
      
      if (msg.type === 'message' || msg.type === 'ack') {
        messagesReceived.add(1);
        
        // Calculate latency if we have the original timestamp
        if (msg.msg_id && pendingMessages.has(msg.msg_id)) {
          const sentTime = pendingMessages.get(msg.msg_id);
          const latency = Date.now() - sentTime;
          messageLatency.add(latency);
          messageThroughput.add(1);
          pendingMessages.delete(msg.msg_id);
        }
      }
    });

    socket.on('error', function (e) {
      sendErrors.add(1);
      console.error(`WebSocket error: ${e.error()}`);
    });

    // Send messages continuously
    socket.setInterval(function () {
      if (!authenticated) return;

      const msgId = `msg_${userId}_${Date.now()}_${Math.random()}`;
      const content = randomString(MESSAGE_SIZE);
      
      const message = JSON.stringify({
        type: 'send_msg',
        msg_id: msgId,
        recipient_id: `user_${(__VU % 1000) + 1}`, // Round-robin recipients
        content: content,
        timestamp: Date.now(),
      });

      try {
        socket.send(message);
        messagesSent.add(1);
        pendingMessages.set(msgId, Date.now());
      } catch (e) {
        sendErrors.add(1);
        messageThroughput.add(0);
      }
    }, 100); // Send message every 100ms (10 msg/sec per user)

    // Keep connection alive for test duration
    socket.setTimeout(function () {
      socket.close();
    }, 600000); // 10 minutes
  });

  sleep(1);
}

export function handleSummary(data) {
  const totalMessages = data.metrics.messages_sent.values.count;
  const totalTime = data.state.testRunDurationMs / 1000; // Convert to seconds
  const throughput = totalMessages / totalTime;
  
  const summary = {
    'summary.json': JSON.stringify(data),
    'stdout': generateTextSummary(data, throughput),
  };
  
  return summary;
}

function generateTextSummary(data, throughput) {
  let summary = '\n';
  summary += 'Message Throughput Test Summary\n';
  summary += '================================\n\n';
  
  summary += 'Messages:\n';
  summary += `  Sent: ${data.metrics.messages_sent.values.count}\n`;
  summary += `  Received: ${data.metrics.messages_received.values.count}\n`;
  summary += `  Errors: ${data.metrics.send_errors.values.count}\n`;
  summary += `  Throughput: ${throughput.toFixed(2)} msg/sec\n\n`;
  
  if (data.metrics.message_latency) {
    summary += 'Message Latency:\n';
    summary += `  Min: ${data.metrics.message_latency.values.min.toFixed(2)}ms\n`;
    summary += `  Avg: ${data.metrics.message_latency.values.avg.toFixed(2)}ms\n`;
    summary += `  P50: ${data.metrics.message_latency.values['p(50)'].toFixed(2)}ms\n`;
    summary += `  P95: ${data.metrics.message_latency.values['p(95)'].toFixed(2)}ms\n`;
    summary += `  P99: ${data.metrics.message_latency.values['p(99)'].toFixed(2)}ms\n`;
    summary += `  Max: ${data.metrics.message_latency.values.max.toFixed(2)}ms\n\n`;
  }
  
  summary += 'Success Rate:\n';
  summary += `  Message Success: ${(data.metrics.message_throughput.values.rate * 100).toFixed(2)}%\n`;
  
  return summary;
}
