# Higress Routing Configuration

## Overview
This document describes the Higress-based routing configuration for the monorepo platform. Higress is a cloud-native API gateway that provides advanced features like gRPC-Web translation, rate limiting, circuit breaking, and more.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Higress Gateway                      │
│  (gRPC-Web ↔ gRPC translation, Rate Limiting, CORS, etc.)  │
└─────────────────────────────────────────────────────────────┘
                              │
                ┌─────────────┼─────────────┬─────────────┐
                │             │             │             │
         ┌──────▼──────┐ ┌───▼────┐ ┌─────▼──────┐ ┌───▼────┐
         │ Web Frontend│ │ Hello  │ │   TODO     │ │Shortener│
         │   (React)   │ │Service │ │  Service   │ │ Service │
         │   Port 80   │ │Port 9090│ │ Port 9091  │ │Port 9092│
         └─────────────┘ └────────┘ └────────────┘ └─────────┘
                                                         │
                                                    ┌────▼────┐
                                                    │  HTTP   │
                                                    │Redirect │
                                                    │Port 8080│
                                                    └─────────┘
```

## Services and Routes

### 1. Web Frontend (React Application)
**Domain**: `app.example.com` (or localhost for development)
**Service**: `web:80`
**Path**: `/`

**Features**:
- Serves the React SPA
- SPA routing support (all paths rewrite to `/`)
- Static asset caching (1 year)
- Security headers (X-Content-Type-Options, X-Frame-Options, X-XSS-Protection)

**Access**:
```bash
# Production
https://app.example.com

# Development
http://localhost/
```

### 2. Hello Service (gRPC)
**Domain**: `api.example.com` (or localhost for development)
**Service**: `hello-service:9090`
**Paths**:
- `/api.v1.HelloService/*` - gRPC service path
- `/api/hello/*` - REST-style path

**Features**:
- gRPC-Web to gRPC translation
- CORS enabled for browser access
- Rate limiting: 100 requests/minute per IP

**Access**:
```bash
# gRPC-Web (from browser)
POST https://api.example.com/api.v1.HelloService/SayHello

# REST-style
POST https://api.example.com/api/hello/say

# gRPC (direct)
grpcurl -d '{"name": "World"}' api.example.com:443 api.v1.HelloService/SayHello
```

### 3. TODO Service (gRPC)
**Domain**: `api.example.com` (or localhost for development)
**Service**: `todo-service:9091`
**Paths**:
- `/api.v1.TodoService/*` - gRPC service path
- `/api/todo/*` - REST-style path

**Features**:
- gRPC-Web to gRPC translation
- CORS enabled for browser access
- Rate limiting: 100 requests/minute per IP

**Access**:
```bash
# gRPC-Web (from browser)
POST https://api.example.com/api.v1.TodoService/ListTodos

# REST-style
GET https://api.example.com/api/todo/list

# gRPC (direct)
grpcurl api.example.com:443 api.v1.TodoService/ListTodos
```

### 4. Shortener Service (gRPC + HTTP)
**API Domain**: `api.example.com`
**Redirect Domain**: `short.example.com`
**Services**:
- `shortener-service:9092` - gRPC API
- `shortener-service:8080` - HTTP redirects

**Paths**:
- `/api.v1.ShortenerService/*` - gRPC service path
- `/api/shortener/*` - REST-style path
- `/s/*` or `short.example.com/*` - Short URL redirects

**Features**:
- gRPC-Web to gRPC translation for API
- HTTP redirects for short URLs
- Rate limiting:
  - API: 100 requests/minute per IP
  - Creation: 10 requests/minute per IP (stricter)
  - Redirects: 1000 requests/minute per IP
- Response caching for redirects (5 minutes)

**Access**:
```bash
# Create short URL (gRPC-Web)
POST https://api.example.com/api.v1.ShortenerService/CreateShortUrl

# Create short URL (REST-style)
POST https://api.example.com/api/shortener/create

# Access short URL
GET https://short.example.com/abc123
# or
GET https://api.example.com/s/abc123
```

## Higress Features

### 1. gRPC-Web Translation
Higress automatically translates gRPC-Web requests from browsers to gRPC for backend services.

**Configuration**:
```yaml
annotations:
  higress.io/backend-protocol: "GRPC"
  higress.io/enable-grpc-web: "true"
```

### 2. CORS Support
Enables cross-origin requests from web browsers.

**Configuration**:
```yaml
annotations:
  higress.io/cors-allow-origin: "*"
  higress.io/cors-allow-methods: "GET, POST, PUT, DELETE, OPTIONS"
  higress.io/cors-allow-headers: "content-type,x-grpc-web,x-user-agent,grpc-timeout,authorization"
  higress.io/cors-expose-headers: "grpc-status,grpc-message,grpc-status-details-bin"
  higress.io/cors-max-age: "86400"
  higress.io/cors-allow-credentials: "true"
```

### 3. Rate Limiting
Protects services from abuse using WasmPlugin.

**Configuration**:
```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: api-rate-limit
spec:
  defaultConfig:
    rules:
    - match:
        path:
          prefix: /api/
      limit:
        requests_per_unit: 100
        unit: minute
        key: remote_addr
```

**Limits**:
- API endpoints: 100 requests/minute per IP
- Shortener creation: 10 requests/minute per IP
- Redirects: 1000 requests/minute per IP

### 4. Circuit Breaking
Prevents cascading failures using DestinationRule.

**Configuration**:
```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: api-services-circuit-breaker
spec:
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http2MaxRequests: 100
    outlierDetection:
      consecutiveErrors: 5
      interval: 30s
      baseEjectionTime: 30s
```

**Behavior**:
- Max 100 concurrent connections per service
- Max 100 concurrent HTTP/2 requests
- Eject unhealthy instances after 5 consecutive errors
- Ejection time: 30 seconds

### 5. Advanced Routing (HttpRoute)
Fine-grained routing control using Higress HttpRoute CRD.

**Features**:
- Path-based routing
- Header manipulation
- Request/response transformation
- Traffic splitting (A/B testing, canary deployments)

## Deployment

### Prerequisites
1. Kubernetes cluster
2. Higress installed (via Helm)
3. DNS configured for domains

### Install Higress
```bash
# Add Higress Helm repository
helm repo add higress https://higress.io/helm-charts
helm repo update

# Install Higress
helm install higress higress/higress \
  -f deploy/k8s/infra/higress-values.yaml \
  --namespace higress-system \
  --create-namespace
```

### Deploy Services
```bash
# Development environment
kubectl apply -k deploy/k8s/overlays/development

# Production environment
kubectl apply -k deploy/k8s/overlays/production
```

### Verify Deployment
```bash
# Check Higress gateway
kubectl get pods -n higress-system

# Check ingress resources
kubectl get ingress -n default

# Check routes
kubectl get httproute -n default

# Check rate limiting
kubectl get wasmplugin -n default

# Check circuit breaker
kubectl get destinationrule -n default
```

## Configuration

### Domain Configuration
Update the following domains in `deploy/k8s/services/higress-routes.yaml`:
- `api.example.com` → Your API domain
- `short.example.com` → Your short URL domain
- `app.example.com` → Your web app domain

### TLS/HTTPS Configuration
To enable HTTPS:

1. Create TLS secret:
```bash
kubectl create secret tls higress-tls \
  --cert=path/to/cert.pem \
  --key=path/to/key.pem \
  -n default
```

2. Update ingress with TLS:
```yaml
spec:
  tls:
  - hosts:
    - api.example.com
    - short.example.com
    - app.example.com
    secretName: higress-tls
```

3. Enable SSL redirect:
```yaml
annotations:
  higress.io/ssl-redirect: "true"
```

### Rate Limit Customization
Adjust rate limits in `higress-routes.yaml`:
```yaml
spec:
  defaultConfig:
    rules:
    - match:
        path:
          prefix: /api/
      limit:
        requests_per_unit: 200  # Increase limit
        unit: minute
```

### Circuit Breaker Tuning
Adjust circuit breaker settings:
```yaml
spec:
  trafficPolicy:
    outlierDetection:
      consecutiveErrors: 10  # More tolerant
      baseEjectionTime: 60s  # Longer ejection
```

## Monitoring

### Higress Metrics
Higress exposes Prometheus metrics on port 15020:
```bash
# Port-forward to access metrics
kubectl port-forward -n higress-system \
  svc/higress-gateway 15020:15020

# Access metrics
curl http://localhost:15020/metrics
```

### Key Metrics
- `higress_request_total` - Total requests
- `higress_request_duration_seconds` - Request latency
- `higress_upstream_rq_total` - Upstream requests
- `higress_upstream_rq_time` - Upstream latency
- `higress_rate_limit_rejected` - Rate limited requests

### Logs
View Higress gateway logs:
```bash
kubectl logs -n higress-system \
  -l app=higress-gateway \
  -f
```

## Troubleshooting

### Issue: 502 Bad Gateway
**Cause**: Backend service not available
**Solution**:
```bash
# Check service status
kubectl get pods -n default
kubectl get svc -n default

# Check service endpoints
kubectl get endpoints -n default
```

### Issue: CORS errors
**Cause**: CORS headers not configured
**Solution**: Verify CORS annotations in ingress:
```bash
kubectl describe ingress api-gateway -n default
```

### Issue: Rate limit too strict
**Cause**: Rate limit exceeded
**Solution**: Adjust rate limits or use authentication for higher limits

### Issue: gRPC-Web not working
**Cause**: gRPC-Web translation not enabled
**Solution**: Verify annotations:
```yaml
higress.io/backend-protocol: "GRPC"
higress.io/enable-grpc-web: "true"
```

## Best Practices

1. **Use separate domains** for different purposes:
   - API: `api.example.com`
   - Short URLs: `short.example.com`
   - Web app: `app.example.com`

2. **Enable TLS** in production for security

3. **Configure rate limiting** to protect against abuse

4. **Use circuit breakers** to prevent cascading failures

5. **Monitor metrics** to track performance and issues

6. **Test routing** in development before deploying to production

7. **Use HttpRoute** for advanced routing needs

8. **Configure proper timeouts** based on service requirements

9. **Enable access logs** for debugging

10. **Use health checks** to ensure service availability

## Related Files
- `deploy/k8s/services/higress-routes.yaml` - Main routing configuration
- `deploy/k8s/infra/higress-values.yaml` - Higress Helm values
- `deploy/k8s/services/web/` - Web service K8s resources
- `deploy/k8s/overlays/development/` - Development environment
- `deploy/k8s/overlays/production/` - Production environment

## References
- [Higress Documentation](https://higress.io/docs/)
- [Higress GitHub](https://github.com/alibaba/higress)
- [gRPC-Web Protocol](https://github.com/grpc/grpc-web)
- [Kubernetes Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)
