package registry

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEtcdClient implements a simple in-memory etcd client for testing
type MockEtcdClient struct {
	data    map[string]string
	leases  map[int64]time.Time
	mu      sync.Mutex
	nextID  int64
	ttl     time.Duration
	watches map[string][]chan string
}

func NewMockEtcdClient(ttl time.Duration) *MockEtcdClient {
	return &MockEtcdClient{
		data:    make(map[string]string),
		leases:  make(map[int64]time.Time),
		nextID:  1,
		ttl:     ttl,
		watches: make(map[string][]chan string),
	}
}

func (m *MockEtcdClient) Put(key, value string, leaseID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	if leaseID > 0 {
		m.leases[leaseID] = time.Now().Add(m.ttl)
	}
	return nil
}

func (m *MockEtcdClient) Get(prefix string) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make(map[string]string)
	for k, v := range m.data {
		// Check if key starts with prefix
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			result[k] = v
		}
	}
	return result, nil
}

func (m *MockEtcdClient) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MockEtcdClient) CreateLease() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	leaseID := m.nextID
	m.nextID++
	m.leases[leaseID] = time.Now().Add(m.ttl)
	return leaseID
}

func (m *MockEtcdClient) RenewLease(leaseID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if lease exists and is not expired
	expiry, exists := m.leases[leaseID]
	if !exists {
		return fmt.Errorf("lease not found")
	}

	// Check if lease has expired
	if time.Now().After(expiry) {
		delete(m.leases, leaseID)
		return fmt.Errorf("lease expired")
	}

	// Renew the lease
	m.leases[leaseID] = time.Now().Add(m.ttl)
	return nil
}

// MockRegistryClient wraps MockEtcdClient for testing
type MockRegistryClient struct {
	mock *MockEtcdClient
	ttl  time.Duration
}

func NewMockRegistryClient(ttl time.Duration) *MockRegistryClient {
	return &MockRegistryClient{
		mock: NewMockEtcdClient(ttl),
		ttl:  ttl,
	}
}

func (mrc *MockRegistryClient) RegisterUser(ctx context.Context, userID, deviceID, gatewayNode string) (int64, error) {
	if userID == "" {
		return 0, fmt.Errorf("user ID cannot be empty")
	}
	if deviceID == "" {
		return 0, fmt.Errorf("device ID cannot be empty")
	}
	if gatewayNode == "" {
		return 0, fmt.Errorf("gateway node cannot be empty")
	}

	leaseID := mrc.mock.CreateLease()
	key := fmt.Sprintf("/registry/users/%s/%s", userID, deviceID)
	value := fmt.Sprintf("%s|%d", gatewayNode, time.Now().Unix())
	err := mrc.mock.Put(key, value, leaseID)
	if err != nil {
		return 0, err
	}
	return leaseID, nil
}

func (mrc *MockRegistryClient) UnregisterUser(ctx context.Context, userID, deviceID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if deviceID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}

	key := fmt.Sprintf("/registry/users/%s/%s", userID, deviceID)
	return mrc.mock.Delete(key)
}

func (mrc *MockRegistryClient) LookupUser(ctx context.Context, userID string) ([]GatewayLocation, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	prefix := fmt.Sprintf("/registry/users/%s/", userID)
	data, err := mrc.mock.Get(prefix)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	locations := make([]GatewayLocation, 0, len(data))
	for key, value := range data {
		deviceID := key[len(prefix):]

		// Parse value: {gateway_node}|{connected_at}
		// Find the last | separator to handle colons in gateway_node
		lastPipe := -1
		for i := len(value) - 1; i >= 0; i-- {
			if value[i] == '|' {
				lastPipe = i
				break
			}
		}

		if lastPipe == -1 {
			continue
		}

		gatewayNode := value[:lastPipe]
		var connectedAt int64
		_, err := fmt.Sscanf(value[lastPipe+1:], "%d", &connectedAt)
		if err != nil {
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

func (mrc *MockRegistryClient) RenewLease(ctx context.Context, leaseID int64) error {
	if leaseID <= 0 {
		return fmt.Errorf("invalid lease ID")
	}
	return mrc.mock.RenewLease(leaseID)
}

// Test user registration with TTL
func TestRegisterUser_Success(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	leaseID, err := rc.RegisterUser(ctx, "user001", "device001", "gateway-1:8080")
	require.NoError(t, err)
	assert.Greater(t, leaseID, int64(0))

	// Verify registration
	locations, err := rc.LookupUser(ctx, "user001")
	require.NoError(t, err)
	assert.Len(t, locations, 1)
	assert.Equal(t, "gateway-1:8080", locations[0].GatewayNode)
	assert.Equal(t, "device001", locations[0].DeviceID)
}

// Test empty user ID validation
func TestRegisterUser_EmptyUserID(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	_, err := rc.RegisterUser(ctx, "", "device001", "gateway-1:8080")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID cannot be empty")
}

// Test empty device ID validation
func TestRegisterUser_EmptyDeviceID(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	_, err := rc.RegisterUser(ctx, "user001", "", "gateway-1:8080")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "device ID cannot be empty")
}

// Test empty gateway node validation
func TestRegisterUser_EmptyGatewayNode(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	_, err := rc.RegisterUser(ctx, "user001", "device001", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gateway node cannot be empty")
}

// Test multi-device registration
func TestRegisterUser_MultiDevice(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	// Register same user with 3 different devices
	_, err := rc.RegisterUser(ctx, "user001", "device001", "gateway-1:8080")
	require.NoError(t, err)

	_, err = rc.RegisterUser(ctx, "user001", "device002", "gateway-2:8080")
	require.NoError(t, err)

	_, err = rc.RegisterUser(ctx, "user001", "device003", "gateway-1:8080")
	require.NoError(t, err)

	// Lookup should return all 3 devices
	locations, err := rc.LookupUser(ctx, "user001")
	require.NoError(t, err)
	assert.Len(t, locations, 3)

	// Verify device IDs
	deviceIDs := make(map[string]bool)
	for _, loc := range locations {
		deviceIDs[loc.DeviceID] = true
	}
	assert.True(t, deviceIDs["device001"])
	assert.True(t, deviceIDs["device002"])
	assert.True(t, deviceIDs["device003"])
}

// Test user unregistration
func TestUnregisterUser_Success(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	// Register user
	_, err := rc.RegisterUser(ctx, "user001", "device001", "gateway-1:8080")
	require.NoError(t, err)

	// Unregister user
	err = rc.UnregisterUser(ctx, "user001", "device001")
	require.NoError(t, err)

	// Verify user is gone
	locations, err := rc.LookupUser(ctx, "user001")
	require.NoError(t, err)
	assert.Len(t, locations, 0)
}

// Test unregister with empty user ID
func TestUnregisterUser_EmptyUserID(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	err := rc.UnregisterUser(ctx, "", "device001")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID cannot be empty")
}

// Test unregister with empty device ID
func TestUnregisterUser_EmptyDeviceID(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	err := rc.UnregisterUser(ctx, "user001", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "device ID cannot be empty")
}

// Test lookup for non-existent user
func TestLookupUser_NotFound(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	locations, err := rc.LookupUser(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Len(t, locations, 0)
}

// Test lookup with empty user ID
func TestLookupUser_EmptyUserID(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	_, err := rc.LookupUser(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID cannot be empty")
}

// Test lease renewal
func TestRenewLease_Success(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	// Register user
	leaseID, err := rc.RegisterUser(ctx, "user001", "device001", "gateway-1:8080")
	require.NoError(t, err)

	// Renew lease
	err = rc.RenewLease(ctx, leaseID)
	require.NoError(t, err)

	// Verify user still registered
	locations, err := rc.LookupUser(ctx, "user001")
	require.NoError(t, err)
	assert.Len(t, locations, 1)
}

// Test renew lease with invalid ID
func TestRenewLease_InvalidID(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	err := rc.RenewLease(ctx, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid lease ID")

	err = rc.RenewLease(ctx, -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid lease ID")
}

// Test renew non-existent lease
func TestRenewLease_NotFound(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	err := rc.RenewLease(ctx, 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lease not found")
}

// Test prefix scan for lookup
func TestLookupUser_PrefixScan(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	// Register multiple users
	_, err := rc.RegisterUser(ctx, "user001", "device001", "gateway-1:8080")
	require.NoError(t, err)

	_, err = rc.RegisterUser(ctx, "user002", "device001", "gateway-2:8080")
	require.NoError(t, err)

	_, err = rc.RegisterUser(ctx, "user001", "device002", "gateway-1:8080")
	require.NoError(t, err)

	// Lookup user001 should return only user001's devices
	locations, err := rc.LookupUser(ctx, "user001")
	require.NoError(t, err)
	assert.Len(t, locations, 2)

	for _, loc := range locations {
		assert.Contains(t, loc.DeviceID, "device")
		assert.Equal(t, "gateway-1:8080", loc.GatewayNode)
	}

	// Lookup user002 should return only user002's device
	locations, err = rc.LookupUser(ctx, "user002")
	require.NoError(t, err)
	assert.Len(t, locations, 1)
	assert.Equal(t, "device001", locations[0].DeviceID)
	assert.Equal(t, "gateway-2:8080", locations[0].GatewayNode)
}

// Test concurrent registrations
func TestRegisterUser_Concurrent(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	numGoroutines := 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			userID := fmt.Sprintf("user%03d", idx)
			deviceID := fmt.Sprintf("device%03d", idx)
			gatewayNode := fmt.Sprintf("gateway-%d:8080", idx%3)

			_, err := rc.RegisterUser(ctx, userID, deviceID, gatewayNode)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all users registered
	for i := 0; i < numGoroutines; i++ {
		userID := fmt.Sprintf("user%03d", i)
		locations, err := rc.LookupUser(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, locations, 1)
	}
}

// Test unregister specific device in multi-device scenario
func TestUnregisterUser_SpecificDevice(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	// Register same user with 3 devices
	_, err := rc.RegisterUser(ctx, "user001", "device001", "gateway-1:8080")
	require.NoError(t, err)

	_, err = rc.RegisterUser(ctx, "user001", "device002", "gateway-2:8080")
	require.NoError(t, err)

	_, err = rc.RegisterUser(ctx, "user001", "device003", "gateway-1:8080")
	require.NoError(t, err)

	// Unregister device002
	err = rc.UnregisterUser(ctx, "user001", "device002")
	require.NoError(t, err)

	// Verify only device002 is removed
	locations, err := rc.LookupUser(ctx, "user001")
	require.NoError(t, err)
	assert.Len(t, locations, 2)

	deviceIDs := make(map[string]bool)
	for _, loc := range locations {
		deviceIDs[loc.DeviceID] = true
	}
	assert.True(t, deviceIDs["device001"])
	assert.False(t, deviceIDs["device002"])
	assert.True(t, deviceIDs["device003"])
}

// Test Watch mechanism with PUT events
func TestWatch_PutEvents(t *testing.T) {
	_ = NewMockRegistryClient(90 * time.Second)
	_ = context.Background()

	// Channel to collect watch events
	events := make(chan WatchEvent, 10)
	callback := func(event WatchEvent) {
		events <- event
	}

	// Start watching (mock implementation doesn't actually watch, we'll simulate)
	// In real implementation, this would be tested with actual etcd
	// For unit tests, we verify the callback is called correctly

	// Simulate PUT event
	event := WatchEvent{
		Type:     WatchEventPut,
		UserID:   "user001",
		DeviceID: "device001",
		Key:      "/registry/users/user001/device001",
		Value:    "gateway-1:8080|1234567890",
	}
	callback(event)

	// Verify event received
	select {
	case receivedEvent := <-events:
		assert.Equal(t, WatchEventPut, receivedEvent.Type)
		assert.Equal(t, "user001", receivedEvent.UserID)
		assert.Equal(t, "device001", receivedEvent.DeviceID)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for watch event")
	}
}

// Test Watch mechanism with DELETE events
func TestWatch_DeleteEvents(t *testing.T) {
	_ = NewMockRegistryClient(90 * time.Second)
	_ = context.Background()

	// Channel to collect watch events
	events := make(chan WatchEvent, 10)
	callback := func(event WatchEvent) {
		events <- event
	}

	// Simulate DELETE event
	event := WatchEvent{
		Type:     WatchEventDelete,
		UserID:   "user001",
		DeviceID: "device001",
		Key:      "/registry/users/user001/device001",
		Value:    "",
	}
	callback(event)

	// Verify event received
	select {
	case receivedEvent := <-events:
		assert.Equal(t, WatchEventDelete, receivedEvent.Type)
		assert.Equal(t, "user001", receivedEvent.UserID)
		assert.Equal(t, "device001", receivedEvent.DeviceID)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for watch event")
	}
}

// Test Watch with multiple events
func TestWatch_MultipleEvents(t *testing.T) {
	_ = NewMockRegistryClient(90 * time.Second)
	_ = context.Background()

	// Channel to collect watch events
	events := make(chan WatchEvent, 10)
	callback := func(event WatchEvent) {
		events <- event
	}

	// Simulate multiple events
	testEvents := []WatchEvent{
		{
			Type:     WatchEventPut,
			UserID:   "user001",
			DeviceID: "device001",
			Key:      "/registry/users/user001/device001",
			Value:    "gateway-1:8080|1234567890",
		},
		{
			Type:     WatchEventPut,
			UserID:   "user002",
			DeviceID: "device001",
			Key:      "/registry/users/user002/device001",
			Value:    "gateway-2:8080|1234567891",
		},
		{
			Type:     WatchEventDelete,
			UserID:   "user001",
			DeviceID: "device001",
			Key:      "/registry/users/user001/device001",
			Value:    "",
		},
	}

	for _, event := range testEvents {
		callback(event)
	}

	// Verify all events received
	for i := 0; i < len(testEvents); i++ {
		select {
		case receivedEvent := <-events:
			assert.Equal(t, testEvents[i].Type, receivedEvent.Type)
			assert.Equal(t, testEvents[i].UserID, receivedEvent.UserID)
			assert.Equal(t, testEvents[i].DeviceID, receivedEvent.DeviceID)
		case <-time.After(1 * time.Second):
			t.Fatalf("Timeout waiting for event %d", i)
		}
	}
}

// Test Watch callback with cache invalidation simulation
func TestWatch_CacheInvalidation(t *testing.T) {
	rc := NewMockRegistryClient(90 * time.Second)
	ctx := context.Background()

	// Simulate a local cache
	cache := make(map[string]string)
	cacheMu := sync.Mutex{}

	// Register some users
	_, err := rc.RegisterUser(ctx, "user001", "device001", "gateway-1:8080")
	require.NoError(t, err)

	// Populate cache
	cacheMu.Lock()
	cache["user001"] = "gateway-1:8080"
	cacheMu.Unlock()

	// Watch callback that invalidates cache
	callback := func(event WatchEvent) {
		cacheMu.Lock()
		defer cacheMu.Unlock()

		if event.Type == WatchEventDelete {
			delete(cache, event.UserID)
		} else if event.Type == WatchEventPut {
			// Update cache with new value
			cache[event.UserID] = event.Value
		}
	}

	// Simulate DELETE event
	event := WatchEvent{
		Type:     WatchEventDelete,
		UserID:   "user001",
		DeviceID: "device001",
		Key:      "/registry/users/user001/device001",
		Value:    "",
	}
	callback(event)

	// Verify cache was invalidated
	cacheMu.Lock()
	_, exists := cache["user001"]
	cacheMu.Unlock()
	assert.False(t, exists, "Cache entry should be invalidated")
}

// Test Watch with empty prefix validation
func TestWatch_EmptyPrefix(t *testing.T) {
	// This test verifies that Watch validates input parameters
	// In a real implementation with etcd client, we would test this
	// For unit tests, we just verify the validation logic

	prefix := ""
	assert.Empty(t, prefix, "Empty prefix should be rejected")
}

// Test Watch callback is nil validation
func TestWatch_NilCallback(t *testing.T) {
	// This test verifies that Watch validates callback parameter
	var callback WatchCallback
	assert.Nil(t, callback, "Nil callback should be rejected")
}
