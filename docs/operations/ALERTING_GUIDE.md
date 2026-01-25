# IM Gateway Service Alerting Guide

## Overview

This guide explains the alerting system for the IM Gateway Service, including alert rules, severity levels, notification channels, and response procedures.

## Alert Severity Levels

### Critical
- **Response Time**: Immediate (< 5 minutes)
- **Examples**: Service down, circuit breaker triggered, high message loss
- **Notification**: PagerDuty + Slack + Email
- **Escalation**: Automatic after 15 minutes

### P1 (Priority 1)
- **Response Time**: < 15 minutes
- **Examples**: High P99 latency (> 500ms for 5 minutes)
- **Notification**: PagerDuty + Slack
- **Escalation**: Automatic after 30 minutes

### P2 (Priority 2)
- **Response Time**: < 1 hour
- **Examples**: High ACK timeout rate (> 5%)
- **Notification**: Slack
- **Escalation**: Manual escalation if unresolved after 2 hours

### Warning
- **Response Time**: < 4 hours
- **Examples**: Low cache hit rate, high connection error rate
- **Notification**: Slack
- **Escalation**: Review during business hours

## Alert Rules

### 1. HighMessageDeliveryLatency (P1)

**Trigger**: P99 message delivery latency > 500ms for 5 minutes

**Impact**: Users experience slow message delivery

**Possible Causes**:
- Network congestion
- Database slow queries
- High CPU/memory usage
- Downstream service degradation

**Response**:
1. Check Grafana dashboard: http://grafana:3000/d/im-gateway-messages
2. Review service logs for errors
3. Check CPU/memory usage: `docker stats im-gateway-service`
4. Check database connection pool: Query `SHOW PROCESSLIST` in MySQL
5. Check Redis latency: `redis-cli --latency`
6. Consider scaling out gateway nodes if sustained high load

**Runbook**: https://wiki.example.com/runbooks/im-gateway/high-latency

---

### 2. HighMessageLossRate (Critical - Circuit Breaker)

**Trigger**: Message loss rate > 0.01% for 2 minutes

**Impact**: Messages are being lost, violating SLO

**Possible Causes**:
- Database connection failures
- Kafka broker failures
- Redis connection failures
- Memory exhaustion

**Response**:
1. **IMMEDIATE**: Trigger circuit breaker to prevent further message loss
2. Check all infrastructure health:
   - MySQL: `docker logs mysql`
   - Redis: `docker logs redis`
   - Kafka: `docker logs kafka`
3. Check service logs: `docker logs im-gateway-service`
4. Check memory usage: `docker stats`
5. If database is down, messages should route to offline channel
6. Once infrastructure is healthy, disable circuit breaker

**Runbook**: https://wiki.example.com/runbooks/im-gateway/message-loss

---

### 3. HighAckTimeoutRate (P2)

**Trigger**: ACK timeout rate > 5% for 5 minutes

**Impact**: Many messages not being acknowledged by clients

**Possible Causes**:
- Client network issues
- Client application crashes
- WebSocket connection instability
- ACK timeout too aggressive (5s)

**Response**:
1. Check Grafana dashboard: http://grafana:3000/d/im-gateway-messages
2. Review client-side logs if available
3. Check WebSocket connection stability
4. Review ACK timeout configuration
5. Consider increasing ACK timeout if network conditions are poor

**Runbook**: https://wiki.example.com/runbooks/im-gateway/ack-timeout

---

### 4. HighConnectionErrorRate (Warning)

**Trigger**: Connection error rate > 10% for 5 minutes

**Impact**: Many clients unable to establish connections

**Possible Causes**:
- Authentication service down
- Invalid JWT tokens
- Network issues
- Rate limiting

**Response**:
1. Check auth-service health: `curl http://auth-service:9095/health`
2. Review connection error logs
3. Check rate limiting configuration
4. Verify JWT token generation is working
5. Check network connectivity

**Runbook**: https://wiki.example.com/runbooks/im-gateway/connection-errors

---

### 5. TooManyActiveConnections (Warning)

**Trigger**: Active connections > 100,000 for 5 minutes

**Impact**: Gateway node approaching capacity limit

**Possible Causes**:
- Organic traffic growth
- Connection leak (clients not disconnecting)
- DDoS attack

**Response**:
1. Check connection growth trend in Grafana
2. Verify connections are legitimate (not DDoS)
3. Check for connection leaks (stale connections)
4. **Action**: Scale out gateway nodes
   ```bash
   kubectl scale deployment im-gateway-service --replicas=5
   ```
5. Monitor connection distribution across nodes

**Runbook**: https://wiki.example.com/runbooks/im-gateway/scale-out

---

### 6. HighOfflineQueueBacklog (Warning)

**Trigger**: Offline queue size > 10,000 messages for 10 minutes

**Impact**: Offline messages accumulating, potential memory pressure

**Possible Causes**:
- Offline worker processing too slow
- Database write bottleneck
- Kafka consumer lag

**Response**:
1. Check offline worker logs: `docker logs im-service`
2. Check Kafka consumer lag: `kafka-consumer-groups.sh --describe --group offline-worker`
3. Check database write performance
4. Consider scaling offline worker replicas
5. Monitor memory usage

**Runbook**: https://wiki.example.com/runbooks/im-gateway/offline-backlog

---

### 7. LowCacheHitRate (Warning)

**Trigger**: Cache hit rate < 70% for 10 minutes

**Impact**: Increased load on Registry (etcd) and User Service

**Possible Causes**:
- Cache TTL too short
- High cache eviction rate
- Cache warming not effective
- Traffic pattern change

**Response**:
1. Check cache statistics in Grafana
2. Review cache TTL configuration (default: 5 minutes)
3. Check cache memory limits
4. Consider increasing cache size or TTL
5. Review cache warming strategy

**Runbook**: https://wiki.example.com/runbooks/im-gateway/cache-performance

---

### 8. HighMessageDuplicationRate (Warning)

**Trigger**: Message duplication rate > 1% for 10 minutes

**Impact**: Users seeing duplicate messages

**Possible Causes**:
- Redis deduplication service issues
- Network retries
- Client-side retries
- Race conditions

**Response**:
1. Check Redis health: `redis-cli ping`
2. Review deduplication service logs
3. Check Redis memory usage
4. Verify deduplication TTL (7 days)
5. Review retry logic in clients

**Runbook**: https://wiki.example.com/runbooks/im-gateway/duplication

---

### 9. IMGatewayServiceDown (Critical)

**Trigger**: Service not responding to health checks for 1 minute

**Impact**: Service unavailable, all connections lost

**Possible Causes**:
- Service crash
- OOM kill
- Deadlock
- Infrastructure failure

**Response**:
1. **IMMEDIATE**: Check service status: `docker ps | grep im-gateway`
2. Check service logs: `docker logs im-gateway-service --tail 100`
3. Check resource usage: `docker stats im-gateway-service`
4. Restart service if crashed: `docker restart im-gateway-service`
5. If persistent, check for:
   - Memory leaks
   - Deadlocks
   - Dependency failures (etcd, Redis, Kafka)

**Runbook**: https://wiki.example.com/runbooks/im-gateway/service-down

---

## Notification Channels

### Slack

**Configuration**:
1. Create Slack webhook URL: https://api.slack.com/messaging/webhooks
2. Update `alertmanager-config.yml`:
   ```yaml
   slack_api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
   ```
3. Uncomment Slack configurations in receivers
4. Restart Alertmanager

**Channels**:
- `#alerts-critical`: Critical and P1 alerts
- `#alerts-circuit-breaker`: Circuit breaker alerts
- `#alerts-p2`: P2 alerts
- `#alerts-warnings`: Warning alerts

### PagerDuty

**Configuration**:
1. Create PagerDuty service integration
2. Get service key from PagerDuty
3. Update `alertmanager-config.yml`:
   ```yaml
   pagerduty_url: 'https://events.pagerduty.com/v2/enqueue'
   ```
4. Add service key to critical-alerts receiver
5. Restart Alertmanager

**Escalation Policy**:
- Primary on-call: Immediate notification
- Secondary on-call: After 15 minutes
- Manager: After 30 minutes

### Email

**Configuration**:
1. Configure SMTP settings in `alertmanager-config.yml`
2. Add email addresses to receivers
3. Restart Alertmanager

## Testing Alerts

### Manual Alert Testing

**Test High Latency Alert**:
```bash
# Simulate high latency by adding artificial delay
curl -X POST http://im-gateway-service:8080/test/latency?delay=600ms
```

**Test Message Loss Alert**:
```bash
# Simulate message loss by injecting failures
curl -X POST http://im-gateway-service:8080/test/failure-rate?rate=0.02
```

**Test Service Down Alert**:
```bash
# Stop the service
docker stop im-gateway-service

# Wait 1 minute for alert to fire

# Restart the service
docker start im-gateway-service
```

### Alert Validation

1. Check Prometheus alerts: http://prometheus:9090/alerts
2. Check Alertmanager: http://alertmanager:9093
3. Verify notifications in Slack/PagerDuty/Email
4. Verify alert resolves when condition clears

## Alert Tuning

### Adjusting Thresholds

Edit `prometheus-alerts.yml` and adjust thresholds:

```yaml
# Example: Increase P99 latency threshold to 1 second
- alert: HighMessageDeliveryLatency
  expr: histogram_quantile(0.99, rate(im_gateway_message_delivery_latency_seconds_bucket[5m])) > 1.0
```

### Adjusting Evaluation Intervals

Edit `prometheus-alerts.yml` and adjust `for` duration:

```yaml
# Example: Increase evaluation period to 10 minutes
- alert: HighMessageDeliveryLatency
  expr: histogram_quantile(0.99, rate(im_gateway_message_delivery_latency_seconds_bucket[5m])) > 0.5
  for: 10m  # Changed from 5m
```

### Silencing Alerts

**Temporary Silence** (during maintenance):
```bash
# Silence all alerts for im-gateway-service for 2 hours
amtool silence add alertname=~".+" service="im-gateway-service" --duration=2h --comment="Maintenance window"
```

**Permanent Silence** (disable alert):
```yaml
# Comment out the alert rule in prometheus-alerts.yml
# - alert: HighMessageDuplicationRate
#   expr: ...
```

## Monitoring Alert Health

### Prometheus Metrics

- `prometheus_rule_evaluations_total`: Total rule evaluations
- `prometheus_rule_evaluation_failures_total`: Failed evaluations
- `prometheus_notifications_sent_total`: Notifications sent
- `prometheus_notifications_errors_total`: Notification errors

### Alertmanager Metrics

- `alertmanager_alerts`: Active alerts
- `alertmanager_alerts_received_total`: Alerts received
- `alertmanager_notifications_total`: Notifications sent
- `alertmanager_notifications_failed_total`: Failed notifications

## Runbook Links

All runbooks are available at: https://wiki.example.com/runbooks/im-gateway/

- [High Latency](https://wiki.example.com/runbooks/im-gateway/high-latency)
- [Message Loss](https://wiki.example.com/runbooks/im-gateway/message-loss)
- [ACK Timeout](https://wiki.example.com/runbooks/im-gateway/ack-timeout)
- [Connection Errors](https://wiki.example.com/runbooks/im-gateway/connection-errors)
- [Scale Out](https://wiki.example.com/runbooks/im-gateway/scale-out)
- [Offline Backlog](https://wiki.example.com/runbooks/im-gateway/offline-backlog)
- [Cache Performance](https://wiki.example.com/runbooks/im-gateway/cache-performance)
- [Duplication](https://wiki.example.com/runbooks/im-gateway/duplication)
- [Service Down](https://wiki.example.com/runbooks/im-gateway/service-down)

## SLO Tracking

**Service Level Objectives**:
- Availability: 99.95% (21.6 minutes downtime per month)
- P99 Latency: < 200ms
- Message Loss Rate: < 0.01%

**Error Budget**:
- Monthly error budget: 21.6 minutes
- Alert when 50% of error budget consumed
- Circuit breaker when 80% of error budget consumed

## Support

For questions or issues with alerting:
- Slack: #observability-support
- Email: observability-team@example.com
- On-call: PagerDuty escalation policy
