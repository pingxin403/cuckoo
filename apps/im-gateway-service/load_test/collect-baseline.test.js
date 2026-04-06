const { describe, it } = require('node:test');
const assert = require('node:assert/strict');

const {
  baselineFromThroughputSummary,
  baselineFromConnectionSummary,
  baselineFromClusterSummary,
  buildBaselineReport,
} = require('./collect-baseline.js');

describe('collect-baseline', () => {
  it('extracts throughput baseline with latency and error rate', () => {
    const summary = {
      state: { testRunDurationMs: 120000 },
      metrics: {
        messages_sent: { values: { count: 12000 } },
        send_errors: { values: { count: 120 } },
        message_latency: { values: { 'p(95)': 140, 'p(99)': 190 } },
      },
    };

    const baseline = baselineFromThroughputSummary(summary);

    assert.equal(baseline.throughputMsgSec, 100);
    assert.equal(baseline.p95Ms, 140);
    assert.equal(baseline.p99Ms, 190);
    assert.equal(baseline.errorRate, 0.01);
  });

  it('extracts connection baseline success and latency', () => {
    const summary = {
      metrics: {
        connection_success: { values: { rate: 0.975 } },
        connection_duration: { values: { 'p(95)': 3200, 'p(99)': 4500 } },
      },
    };

    const baseline = baselineFromConnectionSummary(summary);
    assert.equal(baseline.connectSuccessRate, 0.975);
    assert.equal(baseline.connectP95Ms, 3200);
    assert.equal(baseline.connectP99Ms, 4500);
  });

  it('extracts connection baseline from k6 summary-export shape', () => {
    const summary = {
      metrics: {
        connection_success: { value: 0.98 },
        ws_connecting: { 'p(95)': 210.5, 'p(99)': 350.2 },
      },
    };

    const baseline = baselineFromConnectionSummary(summary);
    assert.equal(baseline.connectSuccessRate, 0.98);
    assert.equal(baseline.connectP95Ms, 210.5);
    assert.equal(baseline.connectP99Ms, 350.2);
  });

  it('extracts cluster baseline with error rate', () => {
    const summary = {
      metrics: {
        cluster_messages: { values: { count: 5000 } },
        connection_errors: { values: { count: 25 } },
        message_latency: { values: { 'p(95)': 160, 'p(99)': 220 } },
      },
    };

    const baseline = baselineFromClusterSummary(summary);
    assert.equal(baseline.p95Ms, 160);
    assert.equal(baseline.p99Ms, 220);
    assert.equal(baseline.errorRate, 0.005);
  });

  it('builds p2 baseline report with required critical paths', () => {
    const report = buildBaselineReport({
      label: 'test-baseline',
      throughputSummary: {
        state: { testRunDurationMs: 60000 },
        metrics: {
          messages_sent: { values: { count: 9000 } },
          send_errors: { values: { count: 90 } },
          message_latency: { values: { 'p(95)': 120, 'p(99)': 180 } },
        },
      },
      connectionSummary: {
        metrics: {
          connection_success: { values: { rate: 0.99 } },
          connection_duration: { values: { 'p(95)': 2500, 'p(99)': 3500 } },
        },
      },
      clusterSummary: {
        metrics: {
          cluster_messages: { values: { count: 4000 } },
          connection_errors: { values: { count: 20 } },
          message_latency: { values: { 'p(95)': 150, 'p(99)': 210 } },
        },
      },
    });

    assert.equal(report.label, 'test-baseline');
    assert.ok(report.baselines.privateMessage);
    assert.ok(report.baselines.groupMessage);
    assert.ok(report.baselines.crossGatewayDelivery);
    assert.ok(report.baselines.readReceiptPath);
    assert.equal(report.baselines.readReceiptPath.source, 'pending-dedicated-read-receipt-load-test');
  });

  it('uses pending marker when cluster baseline is not provided', () => {
    const report = buildBaselineReport({
      label: 'no-cluster',
      throughputSummary: {
        state: { testRunDurationMs: 60000 },
        metrics: {
          messages_sent: { values: { count: 3000 } },
          send_errors: { values: { count: 30 } },
          message_latency: { values: { 'p(95)': 100, 'p(99)': 160 } },
        },
      },
      connectionSummary: null,
      clusterSummary: null,
    });

    assert.equal(
      report.baselines.crossGatewayDelivery.source,
      'pending-multi-gateway-cluster-load-test',
    );
  });

  it('prefers multi-gateway cluster per-path baselines when available', () => {
    const report = buildBaselineReport({
      label: 'multi-gateway',
      throughputSummary: null,
      connectionSummary: {
        metrics: {
          connection_success: { value: 0.97 },
          ws_connecting: { 'p(95)': 18.5, 'p(99)': 31.2 },
        },
      },
      clusterSummary: {
        metrics: {
          cluster_private_messages: { count: 7200 },
          cluster_group_messages: { count: 5400 },
          cluster_read_receipts: { count: 3600 },
          cluster_private_latency: { 'p(95)': 145, 'p(99)': 210 },
          cluster_group_latency: { 'p(95)': 182, 'p(99)': 260 },
          cluster_read_receipt_latency: { 'p(95)': 98, 'p(99)': 155 },
          connection_errors: { count: 108 },
        },
        state: { testRunDurationMs: 120000 },
      },
    });

    assert.equal(report.baselines.privateMessage.source, 'k6:cluster-load-test/private');
    assert.equal(report.baselines.groupMessage.source, 'k6:cluster-load-test/group');
    assert.equal(report.baselines.readReceiptPath.source, 'k6:cluster-load-test/read-receipt');

    assert.equal(report.baselines.privateMessage.throughputMsgSec, 60);
    assert.equal(report.baselines.groupMessage.throughputMsgSec, 45);
    assert.equal(report.baselines.readReceiptPath.throughputMsgSec, 30);

    assert.equal(report.baselines.privateMessage.p95Ms, 145);
    assert.equal(report.baselines.groupMessage.p99Ms, 260);
    assert.equal(report.baselines.readReceiptPath.p99Ms, 155);
  });
});
