# Registry Service

The Registry Service manages user-to-gateway mappings using etcd as the distributed key-value store. It supports multi-device sessions and provides TTL-based automatic cleanup.

## Features

- **User Registration**: Register user connections with device_id and gateway_node
- **Multi-Device Support**: Multiple devices per user with separate Registry entries
- **TTL Management**: 90-second TTL with automatic expiration
- **Lease Renewal**: Heartbeat mechanism to keep connections alive
- **Prefix Scan**: Efficient lookup of all devices for a user
- **Thread-Safe**: Concurrent-safe operations using etcd's atomic guarantees

## Architecture

### Key Format

```
/registry/users/{user_id}/{device_id} -> {gateway_node}|{connected_at}
```

Example:
```
/registry/users/user001/device001 -> gateway-1.cluster.local:8080|1704067200
/registry/users/user001/device002 -> gateway-2.cluster.local:8080|1704067230
```

### Data Flow

1. **Connection**: Gateway Node registers user with `RegisterUser(userID, deviceID, gatewayNode)`
2. **Heartbeat**: Gateway Node renews lease every 30 seconds with `RenewLease(leaseID)`
3. **Routing**: Message Router looks up user with `LookupUser(userID)` to find all devices
4. **Disconnection**: Gateway Node unregisters user with `UnregisterUser(userID, deviceID)`
5. **Expiration**: If no heartbeat for 90 seconds, etcd automatically removes the entry

## Usage

### Creating a Registry Client

```go
import (
    "time"
    "github.com/pingxin403/cuckoo/apps/im-service/registry"
)

// Create client with etcd endpoints
endpoints := []string{
    "etcd-1.cluster.local:2379",
    "etcd-2.cluster.local:2379",
    "etcd-3.cluster.local:2379",
}
ttl := 90 * time.Second

client, err := registry.NewRegistryClient(endpoints, ttl)
if err != nil {
    log.Fatalf("Failed to create registry client: %v", err)
}
defer client.Close()
```

### Registering a User

```go
ctx := context.Background()
userID := "user123"
deviceID := "550e8400-e29b-41d4-a716-446655440000" // UUID v4
gatewayNode := "gateway-3.cluster.local:8080"

leaseID, err := client.RegisterUser(ctx, userID, deviceID, gatewayNode)
if err != nil {
    log.Fatalf("Failed to register user: %v", err)
}

log.Printf("Registered user %s on device %s with lease %d", userID, deviceID, leaseID)
```

### Renewing Lease (Heartbeat)

```go
// Renew every 30 seconds
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for range ticker.C {
    err := client.RenewLease(ctx, leaseID)
    if err != nil {
        log.Printf("Failed to renew lease: %v", err)
        // Reconnect logic here
        break
    }
}
```

### Looking Up a User

```go
locations, err := client.LookupUser(ctx, "user123")
if err != nil {
    log.Fatalf("Failed to lookup user: %v", err)
}

if len(locations) == 0 {
    log.Println("User is offline")
} else {
    for _, loc := range locations {
        log.Printf("Device %s connected to %s at %d", 
            loc.DeviceID, loc.GatewayNode, loc.ConnectedAt)
    }
}
```

### Unregistering a User

```go
err := client.UnregisterUser(ctx, "user123", "device001")
if err != nil {
    log.Fatalf("Failed to unregister user: %v", err)
}
```

## Configuration

### etcd Cluster Setup

The Registry requires an etcd cluster with 3 or 5 nodes for high availability:

```yaml
registry:
  endpoints:
    - etcd-1.cluster.local:2379
    - etcd-2.cluster.local:2379
    - etcd-3.cluster.local:2379
  ttl: 90s
  lease_renewal_interval: 30s
  dial_timeout: 5s
  request_timeout: 3s
```

### TTL and Heartbeat

- **TTL**: 90 seconds (configurable)
- **Heartbeat Interval**: 30 seconds (recommended: TTL / 3)
- **Grace Period**: 30 seconds (TTL - 2 * heartbeat_interval)

## Multi-Device Support

The Registry supports multiple devices per user:

```go
// User connects from phone
leaseID1, _ := client.RegisterUser(ctx, "user123", "device-phone", "gateway-1:8080")

// Same user connects from PC
leaseID2, _ := client.RegisterUser(ctx, "user123", "device-pc", "gateway-2:8080")

// Lookup returns both devices
locations, _ := client.LookupUser(ctx, "user123")
// locations = [
//   {GatewayNode: "gateway-1:8080", DeviceID: "device-phone", ConnectedAt: ...},
//   {GatewayNode: "gateway-2:8080", DeviceID: "device-pc", ConnectedAt: ...}
// ]
```

## Testing

### Unit Tests

The package includes comprehensive unit tests using a mock etcd client:

```bash
go test -v ./registry/...
```

Tests cover:
- User registration with TTL
- Multi-device registration
- Lease renewal
- Lookup with prefix scan
- Concurrent operations
- Input validation

### Property-Based Tests

Property-based tests verify correctness properties:

```bash
go test -v ./registry/... -run TestProperty
```

Properties tested:
1. **Registry TTL Expiration**: Leases expire after TTL without heartbeat
2. **Lease Renewal Extends TTL**: Renewal extends the expiration time
3. **Multi-Device Consistency**: All devices are retrievable
4. **Concurrent Registration**: Thread-safe operations
5. **Unregister Specific Device**: Only specified device is removed
6. **Empty Input Validation**: Proper error handling

## Requirements Validated

- **Requirement 7.1**: Registry maintains user_id to gateway_node mapping with 90-second TTL
- **Requirement 7.2**: Gateway nodes renew Registry lease every 30 seconds
- **Requirement 7.6**: Use etcd cluster with 3 or 5 nodes for high availability
- **Requirement 15.1**: Support multiple concurrent connections for the same user_id
- **Requirement 15.2**: Registry maintains mappings of user_id to multiple gateway_node entries with device_id

## Performance Considerations

- **Lookup Latency**: O(1) for single device, O(n) for n devices (prefix scan)
- **Memory**: ~100 bytes per registration
- **Throughput**: Limited by etcd cluster (10K+ ops/sec typical)
- **Scalability**: Horizontal scaling via etcd cluster

## Error Handling

The client returns errors for:
- Empty user_id, device_id, or gateway_node
- etcd connection failures
- Lease not found (expired or invalid)
- Network timeouts

Always check errors and implement retry logic with exponential backoff.

## Watch Mechanism for Cache Invalidation

**Status**: ✅ Implemented (Task 6.2)

The Watch mechanism enables real-time cache invalidation when Registry entries change, implementing Requirements 7.9 and 17.3.

### Usage Example

```go
// Create local cache
cache := make(map[string]string)
cacheMu := sync.Mutex{}

// Define watch callback for cache invalidation
callback := func(event registry.WatchEvent) {
    cacheMu.Lock()
    defer cacheMu.Unlock()
    
    switch event.Type {
    case registry.WatchEventPut:
        // Update cache with new mapping
        cache[event.UserID] = event.Value
        log.Printf("Cache updated: user=%s device=%s", event.UserID, event.DeviceID)
        
    case registry.WatchEventDelete:
        // Invalidate cache entry
        delete(cache, event.UserID)
        log.Printf("Cache invalidated: user=%s device=%s", event.UserID, event.DeviceID)
    }
}

// Start watching for changes
ctx := context.Background()
err := client.Watch(ctx, "/registry/users/", callback)
if err != nil {
    log.Fatal(err)
}

// Watch runs in background with automatic reconnection
// Stop watching when done
client.StopWatch("/registry/users/")
```

### Features

- **Automatic Reconnection**: Watch loop automatically reconnects on connection failures
- **Event Types**: Supports PUT (registration) and DELETE (unregistration) events
- **Concurrent Safe**: Multiple watchers can run simultaneously
- **Graceful Shutdown**: All watchers are stopped when client is closed

## Future Enhancements

- **Metrics**: Add Prometheus metrics for registration/lookup latency
- **Metrics**: Add Prometheus metrics for registration/lookup latency
- **Health Checks**: Periodic health checks for etcd cluster
- **Connection Pooling**: Optimize etcd client connection pooling
