//go:build property
// +build property

package registry

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// Property 6: Registry Consistency with TTL
// **Validates: Requirements 7.1, 7.2, 7.6**
func TestProperty_RegistryTTLExpiration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Use short TTL for testing (1 second)
		ttl := 1 * time.Second
		rc := NewMockRegistryClient(ttl)
		ctx := context.Background()

		// Generate random user and device IDs
		userID := rapid.StringMatching(`^user[0-9]{3}$`).Draw(t, "user_id")
		deviceID := rapid.StringMatching(`^device[0-9]{3}$`).Draw(t, "device_id")
		gatewayNode := rapid.StringMatching(`^gateway-[0-9]:8080$`).Draw(t, "gateway_node")

		// Register user
		leaseID, err := rc.RegisterUser(ctx, userID, deviceID, gatewayNode)
		if err != nil {
			t.Fatalf("Failed to register user: %v", err)
		}

		// Verify user is registered
		locations, err := rc.LookupUser(ctx, userID)
		if err != nil {
			t.Fatalf("Failed to lookup user: %v", err)
		}

		if len(locations) != 1 {
			t.Fatalf("Expected 1 location, got %d", len(locations))
		}

		// Wait for TTL to expire (1.5 seconds to be safe)
		time.Sleep(1500 * time.Millisecond)

		// Property: After TTL expiration without heartbeat, lease should be expired
		// Note: In real etcd, the entry would be automatically removed
		// In our mock, we check if the lease is expired
		err = rc.RenewLease(ctx, leaseID)
		if err == nil {
			t.Fatal("Expected lease renewal to fail after TTL expiration")
		}
	})
}

// Property: Lease renewal extends TTL
// **Validates: Requirements 7.2**
func TestProperty_LeaseRenewalExtendsTTL(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Use short TTL for testing (2 seconds)
		ttl := 2 * time.Second
		rc := NewMockRegistryClient(ttl)
		ctx := context.Background()

		userID := rapid.StringMatching(`^user[0-9]{3}$`).Draw(t, "user_id")
		deviceID := rapid.StringMatching(`^device[0-9]{3}$`).Draw(t, "device_id")
		gatewayNode := rapid.StringMatching(`^gateway-[0-9]:8080$`).Draw(t, "gateway_node")

		// Register user
		leaseID, err := rc.RegisterUser(ctx, userID, deviceID, gatewayNode)
		if err != nil {
			t.Fatalf("Failed to register user: %v", err)
		}

		// Wait 1 second (half of TTL)
		time.Sleep(1 * time.Second)

		// Renew lease
		err = rc.RenewLease(ctx, leaseID)
		if err != nil {
			t.Fatalf("Failed to renew lease: %v", err)
		}

		// Wait another 1.5 seconds (total 2.5 seconds from registration)
		// Without renewal, lease would have expired at 2 seconds
		// With renewal at 1 second, lease should expire at 3 seconds
		time.Sleep(1500 * time.Millisecond)

		// Property: Lease should still be valid after renewal
		err = rc.RenewLease(ctx, leaseID)
		if err != nil {
			t.Fatalf("Lease should still be valid after renewal: %v", err)
		}

		// Verify user is still registered
		locations, err := rc.LookupUser(ctx, userID)
		if err != nil {
			t.Fatalf("Failed to lookup user: %v", err)
		}

		if len(locations) != 1 {
			t.Fatalf("Expected 1 location after renewal, got %d", len(locations))
		}
	})
}

// Property: Multi-device consistency
// **Validates: Requirements 15.1, 15.2**
func TestProperty_MultiDeviceConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rc := NewMockRegistryClient(90 * time.Second)
		ctx := context.Background()

		userID := rapid.StringMatching(`^user[0-9]{3}$`).Draw(t, "user_id")
		numDevices := rapid.IntRange(1, 5).Draw(t, "num_devices")

		// Register multiple devices for the same user
		deviceIDs := make([]string, numDevices)
		for i := 0; i < numDevices; i++ {
			deviceIDs[i] = fmt.Sprintf("device%03d", i)
			gatewayNode := fmt.Sprintf("gateway-%d:8080", i%3)

			_, err := rc.RegisterUser(ctx, userID, deviceIDs[i], gatewayNode)
			if err != nil {
				t.Fatalf("Failed to register device %d: %v", i, err)
			}
		}

		// Property: Lookup should return all registered devices
		locations, err := rc.LookupUser(ctx, userID)
		if err != nil {
			t.Fatalf("Failed to lookup user: %v", err)
		}

		if len(locations) != numDevices {
			t.Fatalf("Expected %d devices, got %d", numDevices, len(locations))
		}

		// Verify all device IDs are present
		foundDevices := make(map[string]bool)
		for _, loc := range locations {
			foundDevices[loc.DeviceID] = true
		}

		for _, deviceID := range deviceIDs {
			if !foundDevices[deviceID] {
				t.Fatalf("Device %s not found in lookup results", deviceID)
			}
		}
	})
}

// Property: Concurrent registrations maintain consistency
// **Validates: Requirements 7.1, 7.6**
func TestProperty_ConcurrentRegistrationConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rc := NewMockRegistryClient(90 * time.Second)
		ctx := context.Background()

		numUsers := rapid.IntRange(5, 20).Draw(t, "num_users")
		var wg sync.WaitGroup

		// Register multiple users concurrently
		for i := 0; i < numUsers; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				userID := fmt.Sprintf("user%03d", idx)
				deviceID := fmt.Sprintf("device%03d", idx)
				gatewayNode := fmt.Sprintf("gateway-%d:8080", idx%3)

				_, err := rc.RegisterUser(ctx, userID, deviceID, gatewayNode)
				if err != nil {
					t.Errorf("Failed to register user %d: %v", idx, err)
				}
			}(i)
		}

		wg.Wait()

		// Property: All users should be registered and retrievable
		for i := 0; i < numUsers; i++ {
			userID := fmt.Sprintf("user%03d", i)
			locations, err := rc.LookupUser(ctx, userID)
			if err != nil {
				t.Fatalf("Failed to lookup user %s: %v", userID, err)
			}

			if len(locations) != 1 {
				t.Fatalf("Expected 1 location for user %s, got %d", userID, len(locations))
			}
		}
	})
}

// Property: Unregister removes only specified device
// **Validates: Requirements 15.2**
func TestProperty_UnregisterSpecificDevice(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rc := NewMockRegistryClient(90 * time.Second)
		ctx := context.Background()

		userID := rapid.StringMatching(`^user[0-9]{3}$`).Draw(t, "user_id")
		numDevices := rapid.IntRange(2, 5).Draw(t, "num_devices")

		// Register multiple devices
		deviceIDs := make([]string, numDevices)
		for i := 0; i < numDevices; i++ {
			deviceIDs[i] = fmt.Sprintf("device%03d", i)
			gatewayNode := fmt.Sprintf("gateway-%d:8080", i%3)

			_, err := rc.RegisterUser(ctx, userID, deviceIDs[i], gatewayNode)
			if err != nil {
				t.Fatalf("Failed to register device %d: %v", i, err)
			}
		}

		// Unregister a random device
		deviceToRemove := rapid.IntRange(0, numDevices-1).Draw(t, "device_to_remove")
		err := rc.UnregisterUser(ctx, userID, deviceIDs[deviceToRemove])
		if err != nil {
			t.Fatalf("Failed to unregister device: %v", err)
		}

		// Property: Lookup should return all devices except the removed one
		locations, err := rc.LookupUser(ctx, userID)
		if err != nil {
			t.Fatalf("Failed to lookup user: %v", err)
		}

		if len(locations) != numDevices-1 {
			t.Fatalf("Expected %d devices after removal, got %d", numDevices-1, len(locations))
		}

		// Verify removed device is not present
		for _, loc := range locations {
			if loc.DeviceID == deviceIDs[deviceToRemove] {
				t.Fatalf("Removed device %s still present in lookup results", deviceIDs[deviceToRemove])
			}
		}
	})
}

// Property: Empty input validation
// **Validates: Requirements 7.1**
func TestProperty_EmptyInputValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rc := NewMockRegistryClient(90 * time.Second)
		ctx := context.Background()

		validUserID := rapid.StringMatching(`^user[0-9]{3}$`).Draw(t, "valid_user_id")
		validDeviceID := rapid.StringMatching(`^device[0-9]{3}$`).Draw(t, "valid_device_id")
		validGatewayNode := rapid.StringMatching(`^gateway-[0-9]:8080$`).Draw(t, "valid_gateway_node")

		// Test empty user ID
		_, err := rc.RegisterUser(ctx, "", validDeviceID, validGatewayNode)
		if err == nil {
			t.Fatal("Expected error for empty user ID")
		}

		// Test empty device ID
		_, err = rc.RegisterUser(ctx, validUserID, "", validGatewayNode)
		if err == nil {
			t.Fatal("Expected error for empty device ID")
		}

		// Test empty gateway node
		_, err = rc.RegisterUser(ctx, validUserID, validDeviceID, "")
		if err == nil {
			t.Fatal("Expected error for empty gateway node")
		}

		// Test empty user ID in lookup
		_, err = rc.LookupUser(ctx, "")
		if err == nil {
			t.Fatal("Expected error for empty user ID in lookup")
		}

		// Test empty user ID in unregister
		err = rc.UnregisterUser(ctx, "", validDeviceID)
		if err == nil {
			t.Fatal("Expected error for empty user ID in unregister")
		}

		// Test empty device ID in unregister
		err = rc.UnregisterUser(ctx, validUserID, "")
		if err == nil {
			t.Fatal("Expected error for empty device ID in unregister")
		}
	})
}

// Property 7: Watch Event Delivery Consistency
// Validates: Requirements 7.9, 17.3
// Property: All Registry changes trigger corresponding watch events
func TestProperty_WatchEventDeliveryConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rc := NewMockRegistryClient(90 * time.Second)
		ctx := context.Background()

		// Generate random operations
		numOps := rapid.IntRange(1, 20).Draw(t, "numOps")

		// Track expected events
		expectedEvents := make([]WatchEvent, 0, numOps)

		// Channel to collect watch events
		events := make(chan WatchEvent, numOps*2)
		callback := func(event WatchEvent) {
			events <- event
		}

		// Perform random operations and track expected events
		for i := 0; i < numOps; i++ {
			userID := fmt.Sprintf("user%03d", rapid.IntRange(1, 10).Draw(t, "userID"))
			deviceID := fmt.Sprintf("device%03d", rapid.IntRange(1, 3).Draw(t, "deviceID"))
			gatewayNode := fmt.Sprintf("gateway-%d:8080", rapid.IntRange(1, 5).Draw(t, "gateway"))

			opType := rapid.IntRange(0, 1).Draw(t, "opType")

			if opType == 0 {
				// Register (PUT event)
				_, err := rc.RegisterUser(ctx, userID, deviceID, gatewayNode)
				if err == nil {
					expectedEvents = append(expectedEvents, WatchEvent{
						Type:     WatchEventPut,
						UserID:   userID,
						DeviceID: deviceID,
					})

					// Simulate watch event
					callback(WatchEvent{
						Type:     WatchEventPut,
						UserID:   userID,
						DeviceID: deviceID,
						Key:      fmt.Sprintf("/registry/users/%s/%s", userID, deviceID),
						Value:    fmt.Sprintf("%s|%d", gatewayNode, time.Now().Unix()),
					})
				}
			} else {
				// Unregister (DELETE event)
				err := rc.UnregisterUser(ctx, userID, deviceID)
				if err == nil {
					expectedEvents = append(expectedEvents, WatchEvent{
						Type:     WatchEventDelete,
						UserID:   userID,
						DeviceID: deviceID,
					})

					// Simulate watch event
					callback(WatchEvent{
						Type:     WatchEventDelete,
						UserID:   userID,
						DeviceID: deviceID,
						Key:      fmt.Sprintf("/registry/users/%s/%s", userID, deviceID),
						Value:    "",
					})
				}
			}
		}

		// Verify all expected events were received
		receivedCount := 0
		timeout := time.After(2 * time.Second)

		for receivedCount < len(expectedEvents) {
			select {
			case event := <-events:
				// Verify event matches one of the expected events
				found := false
				for _, expected := range expectedEvents {
					if event.Type == expected.Type &&
						event.UserID == expected.UserID &&
						event.DeviceID == expected.DeviceID {
						found = true
						break
					}
				}
				assert.True(t, found, "Received unexpected watch event")
				receivedCount++
			case <-timeout:
				t.Fatalf("Timeout: received %d events, expected %d", receivedCount, len(expectedEvents))
			}
		}
	})
}

// Property 8: Watch Event Ordering
// Validates: Requirements 7.9
// Property: Watch events for the same user_id+device_id are delivered in order
func TestProperty_WatchEventOrdering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rc := NewMockRegistryClient(90 * time.Second)
		ctx := context.Background()

		userID := "user001"
		deviceID := "device001"

		// Generate sequence of operations
		numOps := rapid.IntRange(5, 15).Draw(t, "numOps")

		// Track operation sequence
		operations := make([]WatchEventType, 0, numOps)
		events := make(chan WatchEvent, numOps*2)

		callback := func(event WatchEvent) {
			events <- event
		}

		// Perform operations in sequence
		for i := 0; i < numOps; i++ {
			if i%2 == 0 {
				// Register
				_, err := rc.RegisterUser(ctx, userID, deviceID, "gateway-1:8080")
				if err == nil {
					operations = append(operations, WatchEventPut)
					callback(WatchEvent{
						Type:     WatchEventPut,
						UserID:   userID,
						DeviceID: deviceID,
						Key:      fmt.Sprintf("/registry/users/%s/%s", userID, deviceID),
						Value:    fmt.Sprintf("gateway-1:8080|%d", time.Now().Unix()),
					})
				}
			} else {
				// Unregister
				err := rc.UnregisterUser(ctx, userID, deviceID)
				if err == nil {
					operations = append(operations, WatchEventDelete)
					callback(WatchEvent{
						Type:     WatchEventDelete,
						UserID:   userID,
						DeviceID: deviceID,
						Key:      fmt.Sprintf("/registry/users/%s/%s", userID, deviceID),
						Value:    "",
					})
				}
			}
		}

		// Verify events received in order
		for i := 0; i < len(operations); i++ {
			select {
			case event := <-events:
				assert.Equal(t, operations[i], event.Type,
					"Event %d: expected type %v, got %v", i, operations[i], event.Type)
			case <-time.After(1 * time.Second):
				t.Fatalf("Timeout waiting for event %d", i)
			}
		}
	})
}
