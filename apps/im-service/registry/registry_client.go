package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// WatchEventType represents the type of watch event
type WatchEventType int

const (
	WatchEventPut WatchEventType = iota
	WatchEventDelete
)

// WatchEvent represents a change in the Registry
type WatchEvent struct {
	Type     WatchEventType
	UserID   string
	DeviceID string
	Key      string
	Value    string
}

// WatchCallback is called when a Registry change is detected
type WatchCallback func(event WatchEvent)

// RegistryClient manages user-to-gateway mappings in etcd
type RegistryClient struct {
	client    *clientv3.Client
	ttl       time.Duration
	watchers  map[string]context.CancelFunc
	watcherMu sync.Mutex
}

// GatewayLocation represents a user's connection location
type GatewayLocation struct {
	GatewayNode string
	DeviceID    string
	ConnectedAt int64
}

// NewRegistryClient creates a new Registry client with etcd backend
func NewRegistryClient(endpoints []string, ttl time.Duration) (*RegistryClient, error) {
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("etcd endpoints cannot be empty")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("TTL must be positive")
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &RegistryClient{
		client:   client,
		ttl:      ttl,
		watchers: make(map[string]context.CancelFunc),
	}, nil
}

// MaxDevicesPerUser is the maximum number of devices allowed per user
const MaxDevicesPerUser = 5

// RegisterUser registers a user's connection to a gateway node with TTL
// Key format: /registry/users/{user_id}/{device_id}
// Validates: Requirements 15.10 (max 5 devices per user)
func (rc *RegistryClient) RegisterUser(ctx context.Context, userID, deviceID, gatewayNode string) (int64, error) {
	if userID == "" {
		return 0, fmt.Errorf("user ID cannot be empty")
	}
	if deviceID == "" {
		return 0, fmt.Errorf("device ID cannot be empty")
	}
	if gatewayNode == "" {
		return 0, fmt.Errorf("gateway node cannot be empty")
	}

	// Check current device count for this user
	// Validates: Requirement 15.10 (enforce max 5 devices per user)
	prefix := fmt.Sprintf("/registry/users/%s/", userID)
	resp, err := rc.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		return 0, fmt.Errorf("failed to check device count: %w", err)
	}

	// Check if this is a new device (not already registered)
	key := fmt.Sprintf("/registry/users/%s/%s", userID, deviceID)
	existingResp, err := rc.client.Get(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("failed to check existing device: %w", err)
	}

	isNewDevice := len(existingResp.Kvs) == 0

	// If this is a new device and we're at the limit, reject
	if isNewDevice && resp.Count >= MaxDevicesPerUser {
		return 0, fmt.Errorf("maximum number of devices (%d) reached for user", MaxDevicesPerUser)
	}

	// Create lease with TTL
	leaseResp, err := rc.client.Grant(ctx, int64(rc.ttl.Seconds()))
	if err != nil {
		return 0, fmt.Errorf("failed to create lease: %w", err)
	}

	// Build value
	value := fmt.Sprintf("%s|%d", gatewayNode, time.Now().Unix())

	// Put with lease
	_, err = rc.client.Put(ctx, key, value, clientv3.WithLease(leaseResp.ID))
	if err != nil {
		return 0, fmt.Errorf("failed to register user: %w", err)
	}

	return int64(leaseResp.ID), nil
}

// UnregisterUser removes a user's registration from the Registry
func (rc *RegistryClient) UnregisterUser(ctx context.Context, userID, deviceID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if deviceID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}

	key := fmt.Sprintf("/registry/users/%s/%s", userID, deviceID)
	_, err := rc.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to unregister user: %w", err)
	}

	return nil
}

// LookupUser returns all gateway locations for a user (supports multi-device)
func (rc *RegistryClient) LookupUser(ctx context.Context, userID string) ([]GatewayLocation, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	// Prefix scan: /registry/users/{user_id}/
	prefix := fmt.Sprintf("/registry/users/%s/", userID)
	resp, err := rc.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil // User not found (offline)
	}

	locations := make([]GatewayLocation, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		// Parse key to extract device_id
		// Key format: /registry/users/{user_id}/{device_id}
		key := string(kv.Key)
		deviceID := key[len(prefix):]

		// Parse value: {gateway_node}|{connected_at}
		value := string(kv.Value)
		var gatewayNode string
		var connectedAt int64
		_, err := fmt.Sscanf(value, "%s|%d", &gatewayNode, &connectedAt)
		if err != nil {
			// Skip malformed entries
			continue
		}

		locations = append(locations, GatewayLocation{
			GatewayNode: gatewayNode,
			DeviceID:    deviceID,
			ConnectedAt: connectedAt,
		})
	}

	return locations, nil
}

// RenewLease renews a lease to keep the registration alive
func (rc *RegistryClient) RenewLease(ctx context.Context, leaseID int64) error {
	if leaseID <= 0 {
		return fmt.Errorf("invalid lease ID")
	}

	_, err := rc.client.KeepAliveOnce(ctx, clientv3.LeaseID(leaseID))
	if err != nil {
		return fmt.Errorf("failed to renew lease: %w", err)
	}

	return nil
}

// Close closes the etcd client connection
func (rc *RegistryClient) Close() error {
	// Stop all watchers
	rc.watcherMu.Lock()
	for _, cancel := range rc.watchers {
		cancel()
	}
	rc.watchers = make(map[string]context.CancelFunc)
	rc.watcherMu.Unlock()

	if rc.client != nil {
		return rc.client.Close()
	}
	return nil
}

// Watch starts watching for changes on a specific prefix
// Validates: Requirements 7.9, 17.3
func (rc *RegistryClient) Watch(ctx context.Context, prefix string, callback WatchCallback) error {
	if prefix == "" {
		return fmt.Errorf("watch prefix cannot be empty")
	}
	if callback == nil {
		return fmt.Errorf("watch callback cannot be nil")
	}

	// Create cancellable context for this watcher
	watchCtx, cancel := context.WithCancel(ctx)

	// Store cancel function
	rc.watcherMu.Lock()
	rc.watchers[prefix] = cancel
	rc.watcherMu.Unlock()

	// Start watching in a goroutine
	go rc.watchLoop(watchCtx, prefix, callback)

	return nil
}

// watchLoop handles the watch loop with automatic reconnection
func (rc *RegistryClient) watchLoop(ctx context.Context, prefix string, callback WatchCallback) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Start watching
			watchChan := rc.client.Watch(ctx, prefix, clientv3.WithPrefix())

			// Process watch events
			for watchResp := range watchChan {
				if watchResp.Err() != nil {
					// Watch connection failed, will retry after delay
					time.Sleep(1 * time.Second)
					break
				}

				// Process events
				for _, event := range watchResp.Events {
					watchEvent := rc.parseWatchEvent(event)
					if watchEvent != nil {
						callback(*watchEvent)
					}
				}
			}

			// If we get here, watch channel closed
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return
			default:
				// Reconnect after delay
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// parseWatchEvent converts etcd event to WatchEvent
func (rc *RegistryClient) parseWatchEvent(event *clientv3.Event) *WatchEvent {
	key := string(event.Kv.Key)

	// Parse key: /registry/users/{user_id}/{device_id}
	// Extract user_id and device_id
	userID, deviceID := rc.parseRegistryKey(key)
	if userID == "" {
		return nil
	}

	var eventType WatchEventType
	switch event.Type {
	case clientv3.EventTypePut:
		eventType = WatchEventPut
	case clientv3.EventTypeDelete:
		eventType = WatchEventDelete
	default:
		return nil
	}

	return &WatchEvent{
		Type:     eventType,
		UserID:   userID,
		DeviceID: deviceID,
		Key:      key,
		Value:    string(event.Kv.Value),
	}
}

// parseRegistryKey extracts user_id and device_id from registry key
func (rc *RegistryClient) parseRegistryKey(key string) (userID, deviceID string) {
	// Key format: /registry/users/{user_id}/{device_id}
	prefix := "/registry/users/"
	if len(key) <= len(prefix) {
		return "", ""
	}

	remainder := key[len(prefix):]

	// Find the separator between user_id and device_id
	for i := 0; i < len(remainder); i++ {
		if remainder[i] == '/' {
			userID = remainder[:i]
			if i+1 < len(remainder) {
				deviceID = remainder[i+1:]
			}
			return userID, deviceID
		}
	}

	return "", ""
}

// StopWatch stops watching a specific prefix
func (rc *RegistryClient) StopWatch(prefix string) {
	rc.watcherMu.Lock()
	defer rc.watcherMu.Unlock()

	if cancel, ok := rc.watchers[prefix]; ok {
		cancel()
		delete(rc.watchers, prefix)
	}
}
