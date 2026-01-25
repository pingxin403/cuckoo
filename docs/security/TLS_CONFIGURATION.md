# TLS Configuration Guide for IM Gateway Service

## Overview

This guide explains how to configure TLS 1.3 for WebSocket connections (wss://) in the IM Gateway Service, including certificate management, rotation, and enforcement.

## TLS Requirements

- **Protocol**: TLS 1.3 (minimum TLS 1.2)
- **Cipher Suites**: Strong ciphers only (AES-GCM, ChaCha20-Poly1305)
- **Certificate**: Valid X.509 certificate from trusted CA
- **Key Size**: RSA 2048-bit minimum (4096-bit recommended) or ECDSA P-256
- **Certificate Rotation**: Every 90 days

## Architecture

```
┌─────────────┐
│   Client    │
│  (Browser/  │
│   Mobile)   │
└──────┬──────┘
       │ wss:// (TLS 1.3)
       │
       ▼
┌─────────────┐
│   Nginx/    │
│  Traefik    │
│ (TLS Term.) │
└──────┬──────┘
       │ http:// (internal)
       │
       ▼
┌─────────────┐
│ IM Gateway  │
│  Service    │
└─────────────┘
```

## Certificate Management

### Option 1: Let's Encrypt (Recommended for Production)

**Automatic Certificate Management**:

```yaml
# docker-compose.yml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./certbot/conf:/etc/letsencrypt:ro
      - ./certbot/www:/var/www/certbot:ro
    depends_on:
      - im-gateway-service

  certbot:
    image: certbot/certbot
    volumes:
      - ./certbot/conf:/etc/letsencrypt
      - ./certbot/www:/var/www/certbot
    entrypoint: "/bin/sh -c 'trap exit TERM; while :; do certbot renew; sleep 12h & wait $${!}; done;'"
```

**Initial Certificate Request**:
```bash
docker compose run --rm certbot certonly --webroot \
  --webroot-path=/var/www/certbot \
  --email admin@example.com \
  --agree-tos \
  --no-eff-email \
  -d gateway.example.com
```

### Option 2: Self-Signed Certificate (Development Only)

**Generate Self-Signed Certificate**:
```bash
# Generate private key
openssl genrsa -out server.key 4096

# Generate certificate signing request
openssl req -new -key server.key -out server.csr \
  -subj "/C=US/ST=CA/L=SF/O=Example/CN=localhost"

# Generate self-signed certificate (valid for 365 days)
openssl x509 -req -days 365 -in server.csr \
  -signkey server.key -out server.crt

# Create directory and copy files
mkdir -p deploy/docker/certs
cp server.key deploy/docker/certs/
cp server.crt deploy/docker/certs/
```

### Option 3: Corporate CA Certificate

**Use Corporate Certificate**:
```bash
# Copy certificate and key from corporate CA
cp /path/to/corporate.crt deploy/docker/certs/server.crt
cp /path/to/corporate.key deploy/docker/certs/server.key

# Verify certificate
openssl x509 -in deploy/docker/certs/server.crt -text -noout
```

## Nginx Configuration

### TLS Termination with Nginx

Create `deploy/docker/nginx-tls.conf`:

```nginx
# Upstream to IM Gateway Service
upstream im_gateway {
    server im-gateway-service:8080;
}

# HTTP server - redirect to HTTPS
server {
    listen 80;
    server_name gateway.example.com;

    # Let's Encrypt challenge
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    # Redirect all other traffic to HTTPS
    location / {
        return 301 https://$host$request_uri;
    }
}

# HTTPS server - TLS termination
server {
    listen 443 ssl http2;
    server_name gateway.example.com;

    # TLS Configuration
    ssl_certificate /etc/letsencrypt/live/gateway.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/gateway.example.com/privkey.pem;

    # TLS 1.3 only (or TLS 1.2+ for compatibility)
    ssl_protocols TLSv1.3 TLSv1.2;
    
    # Strong cipher suites
    ssl_ciphers 'TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_prefer_server_ciphers off;

    # OCSP Stapling
    ssl_stapling on;
    ssl_stapling_verify on;
    ssl_trusted_certificate /etc/letsencrypt/live/gateway.example.com/chain.pem;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;

    # WebSocket upgrade
    location /ws {
        proxy_pass http://im_gateway;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket timeouts
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }

    # Health check endpoint (no auth required)
    location /health {
        proxy_pass http://im_gateway;
        access_log off;
    }

    # API endpoints
    location /api {
        proxy_pass http://im_gateway;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Docker Compose Configuration

Update `docker-compose.services.yml`:

```yaml
services:
  nginx:
    image: nginx:alpine
    container_name: nginx-tls
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./nginx-tls.conf:/etc/nginx/nginx.conf:ro
      - ./certbot/conf:/etc/letsencrypt:ro
      - ./certbot/www:/var/www/certbot:ro
    depends_on:
      - im-gateway-service
    networks:
      - monorepo-network
    restart: unless-stopped

  certbot:
    image: certbot/certbot
    container_name: certbot
    volumes:
      - ./certbot/conf:/etc/letsencrypt
      - ./certbot/www:/var/www/certbot
    entrypoint: "/bin/sh -c 'trap exit TERM; while :; do certbot renew; sleep 12h & wait $${!}; done;'"
    networks:
      - monorepo-network

  im-gateway-service:
    # ... existing configuration ...
    environment:
      - TLS_ENABLED=false  # TLS terminated at nginx
      - BEHIND_PROXY=true
      - TRUSTED_PROXIES=nginx-tls
```

## Certificate Rotation

### Automatic Rotation with Let's Encrypt

Certbot automatically renews certificates when they have 30 days or less remaining.

**Manual Renewal**:
```bash
docker compose run --rm certbot renew
docker compose restart nginx
```

### Manual Certificate Rotation

**Rotation Script** (`scripts/rotate-tls-cert.sh`):
```bash
#!/bin/bash
set -e

echo "Starting TLS certificate rotation..."

# Backup old certificate
cp deploy/docker/certs/server.crt deploy/docker/certs/server.crt.old
cp deploy/docker/certs/server.key deploy/docker/certs/server.key.old

# Generate new certificate (or copy from CA)
# ... certificate generation steps ...

# Reload nginx to use new certificate
docker compose exec nginx nginx -s reload

echo "Certificate rotation complete"
```

### Certificate Expiry Monitoring

**Prometheus Alert**:
```yaml
- alert: TLSCertificateExpiringSoon
  expr: |
    (ssl_certificate_expiry_seconds{job="nginx"} - time()) / 86400 < 30
  for: 1h
  labels:
    severity: warning
  annotations:
    summary: "TLS certificate expiring in {{ $value }} days"
    description: "Certificate for {{ $labels.instance }} expires soon"
```

## TLS Enforcement

### Client-Side Enforcement

**JavaScript Client**:
```javascript
// Only allow wss:// connections
const wsUrl = `wss://${window.location.host}/ws`;
const ws = new WebSocket(wsUrl);

// Reject non-TLS connections
if (window.location.protocol !== 'https:') {
  console.error('TLS required - redirecting to HTTPS');
  window.location.protocol = 'https:';
}
```

**Mobile Client (iOS)**:
```swift
// Enforce TLS 1.3
let configuration = URLSessionConfiguration.default
configuration.tlsMinimumSupportedProtocolVersion = .TLSv13

let session = URLSession(configuration: configuration)
```

### Server-Side Enforcement

**Nginx Configuration**:
```nginx
# Reject non-TLS connections
server {
    listen 80;
    server_name gateway.example.com;
    
    # Reject all non-ACME traffic
    location / {
        return 403 "TLS required";
    }
    
    # Allow Let's Encrypt challenges
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }
}
```

## Testing TLS Configuration

### Test TLS Version

```bash
# Test TLS 1.3 support
openssl s_client -connect gateway.example.com:443 -tls1_3

# Verify TLS version in output
# Protocol  : TLSv1.3
```

### Test Cipher Suites

```bash
# Test cipher suite negotiation
nmap --script ssl-enum-ciphers -p 443 gateway.example.com
```

### Test Certificate

```bash
# Verify certificate chain
openssl s_client -connect gateway.example.com:443 -showcerts

# Check certificate expiry
echo | openssl s_client -connect gateway.example.com:443 2>/dev/null | \
  openssl x509 -noout -dates
```

### Test WebSocket over TLS

```bash
# Test wss:// connection
wscat -c wss://gateway.example.com/ws
```

### SSL Labs Test

For production deployments, use SSL Labs:
```
https://www.ssllabs.com/ssltest/analyze.html?d=gateway.example.com
```

Target: A+ rating

## Security Best Practices

### DO:
- Use TLS 1.3 (or minimum TLS 1.2)
- Use strong cipher suites only
- Enable HSTS (HTTP Strict Transport Security)
- Implement certificate pinning for mobile apps
- Monitor certificate expiry
- Rotate certificates every 90 days
- Use certificates from trusted CAs

### DON'T:
- Use self-signed certificates in production
- Allow TLS 1.0 or 1.1
- Use weak cipher suites (RC4, DES, 3DES)
- Ignore certificate expiry warnings
- Hardcode certificates in code
- Share private keys
- Use the same certificate across environments

## Troubleshooting

### Certificate Not Trusted

**Problem**: Browser shows "Certificate not trusted" error

**Solution**:
1. Verify certificate chain is complete
2. Check certificate is from trusted CA
3. For self-signed certs, add to trusted store

### TLS Handshake Failure

**Problem**: Connection fails during TLS handshake

**Solution**:
1. Check TLS version compatibility
2. Verify cipher suite support
3. Check certificate validity dates
4. Verify hostname matches certificate CN/SAN

### Certificate Expired

**Problem**: Certificate has expired

**Solution**:
```bash
# Renew certificate immediately
docker compose run --rm certbot renew --force-renewal
docker compose restart nginx
```

## Monitoring

### Metrics to Monitor

- Certificate expiry date
- TLS handshake success rate
- TLS handshake latency
- TLS version distribution
- Cipher suite distribution

### Grafana Dashboard

Create panels for:
- Days until certificate expiry
- TLS handshake errors
- TLS version usage (should be 100% TLS 1.3)

## Compliance

### PCI DSS Requirements

- TLS 1.2 minimum (TLS 1.3 recommended)
- Strong cryptography
- Certificate from trusted CA
- Regular certificate rotation

### GDPR Requirements

- Encryption in transit (TLS)
- Secure key management
- Audit logging of certificate changes

## Support

For TLS configuration issues:
- Slack: #security-team
- Email: security@example.com
- Documentation: https://wiki.example.com/security/tls
