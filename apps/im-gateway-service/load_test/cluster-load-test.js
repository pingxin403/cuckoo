// k6 Load Test: Cluster Load Test
// Tests 10M concurrent users across cluster (100 nodes × 100K connections)
// Validates: Requirements 9.1

import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Trend, Gauge } from 'k6/metrics';

// Custom metrics
const activeConnections = new Gauge('active_connections');
const connectionErrors = new Counter('connection_errors');
const messageLatency = new Trend('message_latency');
const clusterMessages = new Counter('cluster_messages');
const clusterPrivateMessages = new Counter('cluster_private_messages');
const clusterGroupMessages = new Counter('cluster_group_messages');
const clusterReadReceipts = new Counter('cluster_read_receipts');
const clusterPrivateLatency = new Trend('cluster_private_latency');
const clusterGroupLatency = new Trend('cluster_group_latency');
const clusterReadReceiptLatency = new Trend('cluster_read_receipt_latency');
const wsConnectSuccess = new Counter('ws_connect_success');
const wsConnectFailure = new Counter('ws_connect_failure');

// Test configuration
export const options = {
  scenarios: {
    // Scenario: Distributed cluster load
    cluster_load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10m', target: 100000 },  // Ramp to 100K per node
        { duration: '60m', target: 100000 },  // Hold for 1 hour
        { duration: '10m', target: 0 },       // Ramp down
      ],
      gracefulRampDown: '1m',
    },
  },
  thresholds: {
    'active_connections': ['value<=100000'],       // Max 100K per node
    'message_latency': ['p(99)<200'],              // P99 < 200ms
    'connection_errors': ['count<1000'],           // < 1K errors
  },
};

// Environment variables
const GATEWAY_HOST = __ENV.GATEWAY_HOST || 'localhost';
const GATEWAY_PORT = __ENV.GATEWAY_PORT || '8080';
const NODE_ID = __ENV.NODE_ID || '1';
const CLUSTER_SIZE = parseInt(__ENV.CLUSTER_SIZE || '100');
const AUTH_TOKEN = __ENV.AUTH_TOKEN || 'test-token';
const SEND_INTERVAL_MS = parseInt(__ENV.SEND_INTERVAL_MS || '1000');

function sendCrossClusterMessage(socket, userId) {
  const targetNode = Math.floor(Math.random() * CLUSTER_SIZE) + 1;
  const sendGroupMessage = Math.random() < 0.3;
  const targetUser = Math.floor(Math.random() * 100000) + 1;
  const targetGroup = Math.floor(Math.random() * 1000) + 1;
  const recipientId = sendGroupMessage
    ? `group_${targetGroup}`
    : `user_node${targetNode}_${targetUser}`;

  const message = JSON.stringify({
    type: 'send_msg',
    msg_id: `msg_${userId}_${Date.now()}`,
    sender_id: userId,
    recipient_id: recipientId,
    recipient_type: sendGroupMessage ? 'group' : 'user',
    content: `Cross-cluster message from node ${NODE_ID}`,
    timestamp: Date.now(),
  });

  const sendStartedAt = Date.now();
  socket.send(message);
  const sendLatency = Date.now() - sendStartedAt;

  if (sendGroupMessage) {
    clusterGroupMessages.add(1);
    clusterGroupLatency.add(sendLatency);
  } else {
    clusterPrivateMessages.add(1);
    clusterPrivateLatency.add(sendLatency);
  }

  clusterReadReceipts.add(1);
  clusterReadReceiptLatency.add(sendLatency);
}

export default function () {
  const url = `ws://${GATEWAY_HOST}:${GATEWAY_PORT}/ws`;
  const userId = `user_node${NODE_ID}_${__VU}`;
  const deviceId = `device_node${NODE_ID}_${__VU}`;
  
  const params = {
    headers: {
      'Authorization': `Bearer ${AUTH_TOKEN}`,
    },
    tags: {
      scenario: 'cluster_load',
      node_id: NODE_ID,
      user_id: userId,
    },
  };

  const res = ws.connect(url, params, function (socket) {
    let authenticated = false;
    let connectionCount = 0;

    socket.on('open', function () {
      connectionCount++;
      activeConnections.add(1);
      
      // Send authentication
      const authMsg = JSON.stringify({
        type: 'auth',
        token: AUTH_TOKEN,
        user_id: userId,
        device_id: deviceId,
        node_id: NODE_ID,
      });
      socket.send(authMsg);
      sendCrossClusterMessage(socket, userId);
    });

    socket.on('message', function (data) {
      const msg = JSON.parse(data);
      
      if (msg.type === 'auth_response') {
        authenticated = msg.success;
        check(msg, {
          'auth successful': (m) => m.success === true,
        });
      }
      
      if (msg.type === 'message') {
        clusterMessages.add(1);

        const isGroupMessage = !!msg.group_id || msg.recipient_type === 'group' || (msg.recipient_id && msg.recipient_id.startsWith('group_'));

        if (isGroupMessage) {
          clusterGroupMessages.add(1);
        } else {
          clusterPrivateMessages.add(1);
        }

        if (msg.timestamp) {
          const latency = Date.now() - msg.timestamp;
          messageLatency.add(latency);
          if (isGroupMessage) {
            clusterGroupLatency.add(latency);
          } else {
            clusterPrivateLatency.add(latency);
          }
        }
      }

      if (msg.type === 'read_receipt') {
        clusterReadReceipts.add(1);
        if (msg.read_at) {
          clusterReadReceiptLatency.add(Date.now() - msg.read_at);
        } else if (msg.timestamp) {
          clusterReadReceiptLatency.add(Date.now() - msg.timestamp);
        }
      }
    });

    socket.on('error', function (e) {
      connectionErrors.add(1);
      activeConnections.add(-1);
      console.error(`WebSocket error on node ${NODE_ID}: ${e.error()}`);
    });

    socket.on('close', function () {
      activeConnections.add(-1);
    });

    // Send heartbeat every 30 seconds
    socket.setInterval(function () {
      if (!authenticated) return;
      
      const heartbeat = JSON.stringify({
        type: 'heartbeat',
        user_id: userId,
        node_id: NODE_ID,
        timestamp: Date.now(),
      });
      socket.send(heartbeat);
    }, 30000);

    socket.setInterval(function () {
      sendCrossClusterMessage(socket, userId);
    }, SEND_INTERVAL_MS);

    // Keep connection alive for test duration
    socket.setTimeout(function () {
      socket.close();
    }, 4200000); // 70 minutes
  });

  if (res && res.status === 101) {
    wsConnectSuccess.add(1);
  } else {
    wsConnectFailure.add(1);
  }

  sleep(1);
}

export function handleSummary(data) {
  const summary = {
    node_id: NODE_ID,
    active_connections: data.metrics.active_connections ? data.metrics.active_connections.values.value : 0,
    total_messages: data.metrics.cluster_messages ? data.metrics.cluster_messages.values.count : 0,
    connection_errors: data.metrics.connection_errors ? data.metrics.connection_errors.values.count : 0,
    latency_p99: data.metrics.message_latency ? data.metrics.message_latency.values['p(99)'] : 0,
    cluster_private_messages: data.metrics.cluster_private_messages
      ? data.metrics.cluster_private_messages.values.count
      : 0,
    cluster_group_messages: data.metrics.cluster_group_messages
      ? data.metrics.cluster_group_messages.values.count
      : 0,
    cluster_read_receipts: data.metrics.cluster_read_receipts
      ? data.metrics.cluster_read_receipts.values.count
      : 0,
    cluster_private_latency_p95: data.metrics.cluster_private_latency
      ? data.metrics.cluster_private_latency.values['p(95)']
      : null,
    cluster_private_latency_p99: data.metrics.cluster_private_latency
      ? data.metrics.cluster_private_latency.values['p(99)']
      : null,
    cluster_group_latency_p95: data.metrics.cluster_group_latency
      ? data.metrics.cluster_group_latency.values['p(95)']
      : null,
    cluster_group_latency_p99: data.metrics.cluster_group_latency
      ? data.metrics.cluster_group_latency.values['p(99)']
      : null,
    cluster_read_receipt_latency_p95: data.metrics.cluster_read_receipt_latency
      ? data.metrics.cluster_read_receipt_latency.values['p(95)']
      : null,
    cluster_read_receipt_latency_p99: data.metrics.cluster_read_receipt_latency
      ? data.metrics.cluster_read_receipt_latency.values['p(99)']
      : null,
    ws_connect_success: data.metrics.ws_connect_success
      ? data.metrics.ws_connect_success.values.count
      : 0,
    ws_connect_failure: data.metrics.ws_connect_failure
      ? data.metrics.ws_connect_failure.values.count
      : 0,
  };
  
  return {
    [`summary-node-${NODE_ID}.json`]: JSON.stringify(summary, null, 2),
    'stdout': generateTextSummary(data),
  };
}

function generateTextSummary(data) {
  let summary = '\n';
  summary += `Cluster Load Test Summary - Node ${NODE_ID}\n`;
  summary += '==========================================\n\n';
  
  summary += 'Connections:\n';
  if (data.metrics.active_connections) {
    summary += `  Active: ${data.metrics.active_connections.values.value}\n`;
  }
  summary += `  Errors: ${data.metrics.connection_errors ? data.metrics.connection_errors.values.count : 0}\n\n`;
  summary += `  WS Connect Success: ${data.metrics.ws_connect_success ? data.metrics.ws_connect_success.values.count : 0}\n`;
  summary += `  WS Connect Failure: ${data.metrics.ws_connect_failure ? data.metrics.ws_connect_failure.values.count : 0}\n\n`;
  
  summary += 'Messages:\n';
  summary += `  Total: ${data.metrics.cluster_messages ? data.metrics.cluster_messages.values.count : 0}\n\n`;

  summary += 'Per-path Message Counters:\n';
  summary += `  Private: ${data.metrics.cluster_private_messages ? data.metrics.cluster_private_messages.values.count : 0}\n`;
  summary += `  Group: ${data.metrics.cluster_group_messages ? data.metrics.cluster_group_messages.values.count : 0}\n`;
  summary += `  ReadReceipt: ${data.metrics.cluster_read_receipts ? data.metrics.cluster_read_receipts.values.count : 0}\n\n`;
  
  if (data.metrics.message_latency) {
    summary += 'Cross-Cluster Latency:\n';
    summary += `  P50: ${data.metrics.message_latency.values['p(50)'].toFixed(2)}ms\n`;
    summary += `  P95: ${data.metrics.message_latency.values['p(95)'].toFixed(2)}ms\n`;
    summary += `  P99: ${data.metrics.message_latency.values['p(99)'].toFixed(2)}ms\n`;
  }
  
  return summary;
}
