# Alerting System Quick Start Guide

## Prerequisites

- Docker and Docker Compose installed
- Observability stack running (Prometheus, Grafana, OTel Collector)
- IM Gateway Service running with metrics enabled

## Step 1: Start Alertmanager

```bash
cd deploy/docker

# Start the observability stack with Alertmanager
docker compose -f docker-compose.observability.yml up -d

# Verify Alertmanager is running
docker ps | grep alertmanager
curl http://localhost:9093/-/healthy
```

## Step 2: Verify Alert Rules Loaded

```bash
# Check Prometheus has loaded the alert rules
curl http://localhost:9090/api/v1/rules | jq '.data.groups[].rules[].name'

# Expected output:
# "HighMessageDeliveryLatency"
# "HighMessageLossRate"
# "HighAckTimeoutRate"
# ... (more alerts)
```

## Step 3: Access Alerting UIs

- **Prometheus Alerts**: http://localhost:9090/alerts
- **Alertmanager**: http://localhost:9093
- **Grafana Dashboards**: http://localhost:3000

## Step 4: Test Alert Firing (Optional)

### Test High Latency Alert

```bash
# Simulate high latency by querying a slow endpoint
# (This requires implementing a test endpoint in the service)
```


### View Active Alerts

```bash
# List all active alerts
curl http://localhost:9090/api/v1/alerts | jq '.data.alerts[] | {name: .labels.alertname, state: .state}'

# View alerts in Alertmanager
curl http://localhost:9093/api/v2/alerts | jq '.'
```

## Step 5: Configure Notification Channels

### Slack Configuration

1. Create a Slack webhook: https://api.slack.com/messaging/webhooks
2. Edit `alertmanager-config.yml`:
   ```yaml
   global:
     slack_api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
   ```
3. Uncomment Slack configurations in receivers
4. Restart Alertmanager:
   ```bash
   docker compose -f docker-compose.observability.yml restart alertmanager
   ```

### PagerDuty Configuration

1. Create a PagerDuty integration key
2. Edit `alertmanager-config.yml` and add your service key
3. Uncomment PagerDuty configurations
4. Restart Alertmanager

## Step 6: Verify Alert Routing

```bash
# Send a test alert to Alertmanager
curl -X POST http://localhost:9093/api/v1/alerts -H "Content-Type: application/json" -d '[
  {
    "labels": {
      "alertname": "TestAlert",
      "severity": "warning",
      "service": "im-gateway-service"
    },
    "annotations": {
      "summary": "This is a test alert",
      "description": "Testing alert routing"
    }
  }
]'

# Check if alert appears in Alertmanager UI
open http://localhost:9093
```

## Troubleshooting

### Alerts Not Firing

1. Check Prometheus is scraping metrics:
   ```bash
   curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .labels.job, health: .health}'
   ```

2. Check alert rule evaluation:
   ```bash
   curl http://localhost:9090/api/v1/rules | jq '.data.groups[].rules[] | select(.state != "inactive")'
   ```

### Notifications Not Sending

1. Check Alertmanager logs:
   ```bash
   docker logs alertmanager
   ```

2. Verify webhook/integration configuration
3. Test webhook manually with curl

## Next Steps

- Review [ALERTING_GUIDE.md](./ALERTING_GUIDE.md) for detailed alert documentation
- Configure production notification channels (Slack, PagerDuty, Email)
- Set up alert silences for maintenance windows
- Create runbooks for each alert type
- Monitor alert health metrics in Grafana

## Useful Commands

```bash
# Reload Prometheus configuration
curl -X POST http://localhost:9090/-/reload

# Reload Alertmanager configuration
curl -X POST http://localhost:9093/-/reload

# Silence an alert for 2 hours
amtool silence add alertname="HighMessageDeliveryLatency" --duration=2h --comment="Maintenance"

# List active silences
amtool silence query

# View Alertmanager status
amtool check-config alertmanager-config.yml
```

## Documentation

- [Prometheus Alerting](https://prometheus.io/docs/alerting/latest/overview/)
- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [Alert Rule Best Practices](https://prometheus.io/docs/practices/alerting/)
