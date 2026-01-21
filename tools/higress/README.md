# Higress Configuration for URL Shortener Service

This directory contains Higress ingress configurations for the URL Shortener Service.

## Overview

Higress is a cloud-native API gateway built on Envoy and Istio. These configurations enable:

- **gRPC API routing**: Routes `/api/shortener` to the gRPC service (port 9092)
- **HTTP redirect routing**: Routes short codes to the HTTP redirect handler (port 8080)
- **Rate limiting**: Protects the service from abuse
- **Circuit breaking**: Prevents cascading failures
- **CORS support**: Enables gRPC-Web from browsers

## Files

- `shortener-route.yaml`: Main ingress configuration with routing rules

## Prerequisites

- Kubernetes cluster with Higress installed
- `kubectl` configured to access the cluster
- Shortener service deployed to the cluster

## Quick Start

### 1. Update Domain Names

Edit `shortener-route.yaml` and replace the placeholder domains:

```yaml
# For gRPC API
- host: api.example.com  # Replace with your API domain

# For HTTP redirects
- host: short.example.com  # Replace with your short URL domain
```

### 2. Deploy Configuration

```bash
# Apply the Higress configuration
kubectl apply -f tools/higress/shortener-route.yaml

# Verify ingress is created
kubectl get ingress shortener-service-ingress

# Check ingress details
kubectl describe ingress shortener-service-ingress
```

### 3. Verify Routing

```bash
# Test gRPC API (requires grpcurl)
grpcurl -plaintext -d '{
  "long_url": "https://example.com/test"
}' api.example.com:443 api.v1.ShortenerService/CreateShortLink

# Test HTTP redirect
curl -I http://short.example.com/abc1234
```

## Configuration Details

### Routing Rules

#### gRPC API Routing

```yaml
- path: /api/shortener
  pathType: Prefix
  backend:
    service:
      name: shortener-service
      port:
        number: 9092
```

Routes all requests to `/api/shortener/*` to the gRPC service on port 9092.

#### HTTP Redirect Routing

```yaml
- path: /
  pathType: Prefix
  backend:
    service:
      name: shortener-service
      port:
        number: 8080
```

Routes all other requests to the HTTP redirect handler on port 8080.

### Rate Limiting

The configuration includes rate limiting to prevent abuse:

- **gRPC API**: 100 requests per minute per IP
- **HTTP redirects**: 1000 requests per minute per IP

To adjust limits, edit the `WasmPlugin` configuration:

```yaml
config:
  rules:
  - match:
      path:
        prefix: /api/shortener
    limit:
      requests_per_unit: 100  # Adjust this value
      unit: minute
```

### Circuit Breaking

Circuit breaker settings prevent cascading failures:

- **Max connections**: 100
- **Max pending requests**: 50
- **Consecutive errors before ejection**: 5
- **Ejection time**: 30 seconds

To adjust circuit breaker settings, edit the `DestinationRule`:

```yaml
trafficPolicy:
  connectionPool:
    tcp:
      maxConnections: 100  # Adjust this value
```

### CORS Configuration

CORS is enabled for gRPC-Web support:

```yaml
annotations:
  higress.io/cors-allow-origin: "*"
  higress.io/cors-allow-methods: "GET, POST, PUT, DELETE, OPTIONS"
  higress.io/cors-allow-headers: "content-type,x-grpc-web,x-user-agent,grpc-timeout"
```

For production, restrict `cors-allow-origin` to specific domains:

```yaml
higress.io/cors-allow-origin: "https://yourdomain.com"
```

## Advanced Configuration

### TLS/HTTPS

To enable HTTPS, add a TLS section to the ingress:

```yaml
spec:
  tls:
  - hosts:
    - api.example.com
    - short.example.com
    secretName: shortener-tls-secret
```

Create the TLS secret:

```bash
kubectl create secret tls shortener-tls-secret \
  --cert=path/to/tls.crt \
  --key=path/to/tls.key
```

### Custom Headers

Add custom headers to responses:

```yaml
annotations:
  higress.io/response-header-add: |
    X-Service-Name: shortener-service
    X-Service-Version: 1.0.0
```

### Request Timeout

Adjust request timeouts:

```yaml
annotations:
  higress.io/upstream-timeout: "30s"  # gRPC API timeout
```

### Load Balancing

Configure load balancing strategy:

```yaml
trafficPolicy:
  loadBalancer:
    simple: ROUND_ROBIN  # or LEAST_CONN, RANDOM
```

## Monitoring

### Check Ingress Status

```bash
# Get ingress status
kubectl get ingress shortener-service-ingress -o yaml

# Check ingress events
kubectl get events --field-selector involvedObject.name=shortener-service-ingress
```

### View Higress Logs

```bash
# Get Higress gateway pod
kubectl get pods -n higress-system

# View logs
kubectl logs -n higress-system <higress-gateway-pod> -f
```

### Metrics

Higress exposes Prometheus metrics:

```bash
# Port forward to Higress metrics endpoint
kubectl port-forward -n higress-system svc/higress-gateway 15020:15020

# Access metrics
curl http://localhost:15020/stats/prometheus
```

## Troubleshooting

### Ingress Not Working

1. **Check ingress status**:
   ```bash
   kubectl describe ingress shortener-service-ingress
   ```

2. **Verify service exists**:
   ```bash
   kubectl get svc shortener-service
   ```

3. **Check pod status**:
   ```bash
   kubectl get pods -l app=shortener-service
   ```

4. **View Higress logs**:
   ```bash
   kubectl logs -n higress-system -l app=higress-gateway --tail=100
   ```

### Rate Limiting Not Working

1. **Check WasmPlugin status**:
   ```bash
   kubectl get wasmplugin shortener-rate-limit -o yaml
   ```

2. **Verify plugin is loaded**:
   ```bash
   kubectl logs -n higress-system -l app=higress-gateway | grep rate-limit
   ```

### Circuit Breaker Not Triggering

1. **Check DestinationRule**:
   ```bash
   kubectl get destinationrule shortener-service-circuit-breaker -o yaml
   ```

2. **Monitor connection pool metrics**:
   ```bash
   # Check Envoy stats
   kubectl exec -n higress-system <higress-gateway-pod> -- \
     curl localhost:15000/stats | grep shortener
   ```

## Production Recommendations

1. **Use specific domains**: Replace `example.com` with your actual domains
2. **Enable TLS**: Always use HTTPS in production
3. **Restrict CORS**: Limit `cors-allow-origin` to trusted domains
4. **Adjust rate limits**: Set appropriate limits based on expected traffic
5. **Monitor metrics**: Set up Prometheus and Grafana for monitoring
6. **Set up alerts**: Configure alerts for high error rates and circuit breaker trips
7. **Use multiple replicas**: Deploy multiple instances of the shortener service
8. **Configure autoscaling**: Use HPA (Horizontal Pod Autoscaler) for automatic scaling

## References

- [Higress Documentation](https://higress.io/docs/)
- [Kubernetes Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)
- [Istio Traffic Management](https://istio.io/latest/docs/concepts/traffic-management/)
- [Envoy Proxy](https://www.envoyproxy.io/docs/envoy/latest/)

## Support

For questions or issues:
- Check the [Higress GitHub](https://github.com/alibaba/higress)
- Review the [main README](../../README.md)
- Contact the platform team
