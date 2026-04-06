#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

function parseArgs(argv) {
  const args = {
    throughput: '',
    connection: '',
    cluster: '',
    output: '',
    label: 'p2-baseline',
  };

  for (let i = 2; i < argv.length; i += 1) {
    const key = argv[i];
    const value = argv[i + 1];
    if (!value || value.startsWith('--')) {
      continue;
    }
    if (key === '--throughput') args.throughput = value;
    if (key === '--connection') args.connection = value;
    if (key === '--cluster') args.cluster = value;
    if (key === '--output') args.output = value;
    if (key === '--label') args.label = value;
  }

  return args;
}

function num(v, fallback = 0) {
  return typeof v === 'number' && Number.isFinite(v) ? v : fallback;
}

function readJSON(filePath) {
  if (!filePath) return null;
  if (!fs.existsSync(filePath)) return null;
  return JSON.parse(fs.readFileSync(filePath, 'utf-8'));
}

function safeDiv(a, b) {
  if (!b) return 0;
  return a / b;
}

function pickLatency(values) {
  if (!values) {
    return { p95Ms: null, p99Ms: null };
  }
  return {
    p95Ms: num(values['p(95)'], 0),
    p99Ms: num(values['p(99)'], 0),
  };
}

function metricCount(summary, metricName) {
  if (!summary || !summary.metrics) return 0;
  const metric = summary.metrics[metricName];
  if (!metric) return 0;
  if (typeof metric.count === 'number') return num(metric.count, 0);
  if (metric.values && typeof metric.values.count === 'number') {
    return num(metric.values.count, 0);
  }
  return 0;
}

function metricLatency(summary, metricName) {
  if (!summary || !summary.metrics) {
    return { p95Ms: null, p99Ms: null };
  }
  const metric = summary.metrics[metricName];
  if (!metric) {
    return { p95Ms: null, p99Ms: null };
  }
  return pickLatency(metric.values || metric);
}

function baselineFromThroughputSummary(summary) {
  if (!summary || !summary.metrics) return null;
  const sent = num(summary.metrics.messages_sent?.values?.count, 0)
    || num(summary.metrics.messages_sent?.count, 0);
  const sendErrors = num(summary.metrics.send_errors?.values?.count, 0);
  const durationSec = num(summary.state?.testRunDurationMs, 0) / 1000;
  const latency = pickLatency(summary.metrics.message_latency?.values);

  return {
    throughputMsgSec: Number(safeDiv(sent, durationSec).toFixed(2)),
    p95Ms: latency.p95Ms,
    p99Ms: latency.p99Ms,
    timeoutRate: null,
    errorRate: Number(safeDiv(sendErrors, Math.max(sent, 1)).toFixed(6)),
    source: 'k6:message-throughput-test',
  };
}

function baselineFromConnectionSummary(summary) {
  if (!summary || !summary.metrics) return null;
  const successRate = num(summary.metrics.connection_success?.values?.rate, 0)
    || num(summary.metrics.connection_success?.value, 0);
  const latencyValues = summary.metrics.connection_duration?.values
    || summary.metrics.ws_connecting;
  const latency = pickLatency(latencyValues);
  return {
    connectSuccessRate: Number(successRate.toFixed(6)),
    connectP95Ms: latency.p95Ms,
    connectP99Ms: latency.p99Ms,
    source: 'k6:connection-load-test',
  };
}

function baselineFromClusterSummary(summary) {
  if (!summary || !summary.metrics) return null;
  const total = metricCount(summary, 'cluster_messages');
  const errors = metricCount(summary, 'connection_errors');
  const latency = metricLatency(summary, 'message_latency');
  return {
    throughputMsgSec: null,
    p95Ms: latency.p95Ms,
    p99Ms: latency.p99Ms,
    timeoutRate: null,
    errorRate: Number(safeDiv(errors, Math.max(total, 1)).toFixed(6)),
    source: 'k6:cluster-load-test',
  };
}

function baselineFromClusterPerPath(summary) {
  if (!summary || !summary.metrics) return null;

  const durationSec = num(summary.state?.testRunDurationMs, 0) / 1000;
  const totalErrors = metricCount(summary, 'connection_errors');

  const privateCount = metricCount(summary, 'cluster_private_messages');
  const groupCount = metricCount(summary, 'cluster_group_messages');
  const readReceiptCount = metricCount(summary, 'cluster_read_receipts');

  const total = privateCount + groupCount + readReceiptCount;
  const errorRate = Number(safeDiv(totalErrors, Math.max(total, 1)).toFixed(6));

  const privateLatency = metricLatency(summary, 'cluster_private_latency');
  const groupLatency = metricLatency(summary, 'cluster_group_latency');
  const readReceiptLatency = metricLatency(summary, 'cluster_read_receipt_latency');

  if (total === 0) {
    return null;
  }

  return {
    privateMessage: {
      throughputMsgSec: Number(safeDiv(privateCount, durationSec).toFixed(2)),
      p95Ms: privateLatency.p95Ms,
      p99Ms: privateLatency.p99Ms,
      timeoutRate: null,
      errorRate,
      source: 'k6:cluster-load-test/private',
    },
    groupMessage: {
      throughputMsgSec: Number(safeDiv(groupCount, durationSec).toFixed(2)),
      p95Ms: groupLatency.p95Ms,
      p99Ms: groupLatency.p99Ms,
      timeoutRate: null,
      errorRate,
      source: 'k6:cluster-load-test/group',
    },
    readReceiptPath: {
      throughputMsgSec: Number(safeDiv(readReceiptCount, durationSec).toFixed(2)),
      p95Ms: readReceiptLatency.p95Ms,
      p99Ms: readReceiptLatency.p99Ms,
      timeoutRate: null,
      errorRate,
      source: 'k6:cluster-load-test/read-receipt',
    },
  };
}

function buildBaselineReport({ throughputSummary, connectionSummary, clusterSummary, label }) {
  const throughputBaseline = baselineFromThroughputSummary(throughputSummary);
  const connectionBaseline = baselineFromConnectionSummary(connectionSummary);
  const clusterBaseline = baselineFromClusterSummary(clusterSummary);
  const perPathCluster = baselineFromClusterPerPath(clusterSummary);
  const crossGatewayBaseline = clusterBaseline || {
    throughputMsgSec: null,
    p95Ms: null,
    p99Ms: null,
    timeoutRate: null,
    errorRate: null,
    source: 'pending-multi-gateway-cluster-load-test',
  };

  return {
    generatedAt: new Date().toISOString(),
    label,
    baselines: {
      privateMessage: perPathCluster?.privateMessage || throughputBaseline,
      groupMessage: perPathCluster?.groupMessage || (throughputBaseline
        ? {
            ...throughputBaseline,
            source: `${throughputBaseline.source} (proxy)`
          }
        : null),
      crossGatewayDelivery: crossGatewayBaseline,
      readReceiptPath: perPathCluster?.readReceiptPath || {
        throughputMsgSec: null,
        p95Ms: null,
        p99Ms: null,
        timeoutRate: null,
        errorRate: null,
        source: 'pending-dedicated-read-receipt-load-test',
      },
    },
    context: {
      connection: connectionBaseline,
    },
    notes: [
      'Task 1.1 baseline focuses on reproducible capture for throughput/latency/error dimensions.',
      'Read-receipt baseline requires dedicated scenario and live gateway environment.',
    ],
  };
}

function writeOutput(report, outputPath) {
  const payload = JSON.stringify(report, null, 2);
  if (!outputPath) {
    process.stdout.write(`${payload}\n`);
    return;
  }
  const dir = path.dirname(outputPath);
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(outputPath, payload, 'utf-8');
}

function main() {
  const args = parseArgs(process.argv);
  const throughputSummary = readJSON(args.throughput);
  const connectionSummary = readJSON(args.connection);
  const clusterSummary = readJSON(args.cluster);

  const report = buildBaselineReport({
    throughputSummary,
    connectionSummary,
    clusterSummary,
    label: args.label,
  });

  writeOutput(report, args.output);
}

if (require.main === module) {
  main();
}

module.exports = {
  baselineFromThroughputSummary,
  baselineFromConnectionSummary,
  baselineFromClusterSummary,
  baselineFromClusterPerPath,
  buildBaselineReport,
};
