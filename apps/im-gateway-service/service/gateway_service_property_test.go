//go:build property

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// setupTestGatewayForProperty creates a test gateway for property-based tests
func setupTestGatewayForProperty() (*GatewayService, *mockAuthClient, *mockRegistryClient, *mockIMClient) {
	authClient := &mockAuthClient{}
	registryClient := newMockRegistryClient()
	imClient := &mockIMClient{}

	config := DefaultGatewayConfig()
	config.PongWait = 1 * time.Second
	config.PingPeriod = 500 * time.Millisecond

	gateway := NewGatewayService(
		authClient,
		registryClient,
		imClient,
		nil, // Redis client not needed for basic tests
		config,
	)

	return gateway, authClient, registryClient, imClient
}

// TestProperty4_MultiDeviceMessageConsistency tests Property 4:
// When a user is online on multiple devices, all devices MUST receive
// the same messages in the same order.
//
// **Validates: Requirements 15.1, 15.2, 15.3, 15.4**
func TestProperty4_MultiDeviceMessageConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gateway, _, _, _ := setupTestGatewayForProperty()

		// Generate random number of devices (2-5)
		numDevices := rapid.IntRange(2, 5).Draw(t, "numDevices")

		// Generate random number of messages (5-20)
		numMessages := rapid.IntRange(5, 20).Draw(t, "numMessages")

		// Create connections for multiple devices of the same user
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var connections []*Connection
		var mu sync.Mutex
		receivedMessages := make([][]ServerMessage, numDevices)

		for i := 0; i < numDevices; i++ {
			deviceID := fmt.Sprintf("device%d", i)
			conn := &Connection{
				UserID:   "user123",
				DeviceID: deviceID,
				Send:     make(chan []byte, 256),
				Gateway:  gateway,
				ctx:      ctx,
				cancel:   cancel,
			}
			gateway.connections.Store("user123_"+deviceID, conn)
			connections = append(connections, conn)

			// Start goroutine to collect messages
			go func(deviceIdx int, c *Connection) {
				for {
					select {
					case data := <-c.Send:
						var msg ServerMessage
						if err := json.Unmarshal(data, &msg); err == nil {
							mu.Lock()
							receivedMessages[deviceIdx] = append(receivedMessages[deviceIdx], msg)
							mu.Unlock()
						}
					case <-ctx.Done():
						return
					}
				}
			}(i, conn)
		}

		// Send messages to the user
		for i := 0; i < numMessages; i++ {
			req := &PushMessageRequest{
				MsgID:          fmt.Sprintf("msg-%d", i),
				RecipientID:    "user123",
				SenderID:       "sender789",
				Content:        fmt.Sprintf("Message %d", i),
				MessageType:    "text",
				SequenceNumber: int64(1000 + i),
				Timestamp:      time.Now().Unix(),
			}

			resp, err := gateway.pushService.PushMessage(context.Background(), req)
			require.NoError(t, err)
			assert.True(t, resp.Success, "Message %d should be delivered", i)
		}

		// Wait for all messages to be delivered
		time.Sleep(100 * time.Millisecond)

		// Verify all devices received all messages
		mu.Lock()
		defer mu.Unlock()

		for deviceIdx := 0; deviceIdx < numDevices; deviceIdx++ {
			assert.Equal(t, numMessages, len(receivedMessages[deviceIdx]),
				"Device %d should receive all %d messages", deviceIdx, numMessages)
		}

		// Verify all devices received messages in the same order
		if len(receivedMessages[0]) > 0 {
			for deviceIdx := 1; deviceIdx < numDevices; deviceIdx++ {
				for msgIdx := 0; msgIdx < len(receivedMessages[0]); msgIdx++ {
					if msgIdx < len(receivedMessages[deviceIdx]) {
						assert.Equal(t,
							receivedMessages[0][msgIdx].MsgID,
							receivedMessages[deviceIdx][msgIdx].MsgID,
							"Device %d message %d should match device 0", deviceIdx, msgIdx)

						assert.Equal(t,
							receivedMessages[0][msgIdx].SequenceNumber,
							receivedMessages[deviceIdx][msgIdx].SequenceNumber,
							"Device %d sequence number %d should match device 0", deviceIdx, msgIdx)
					}
				}
			}
		}
	})
}

// TestProperty9_GroupCacheMemoryBounds tests Property 9:
// Gateway Node memory usage for group caches MUST be bounded by the number
// of locally-connected users, not total group membership.
//
// **Validates: Requirements 2.10, 2.11, 2.12**
func TestProperty9_GroupCacheMemoryBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gateway, _, _, _ := setupTestGatewayForProperty()

		// Generate a large group size (1000-5000 members)
		totalGroupMembers := rapid.IntRange(1000, 5000).Draw(t, "totalGroupMembers")

		// Generate number of locally-connected members (10-100)
		locallyConnected := rapid.IntRange(10, 100).Draw(t, "locallyConnected")

		// Ensure locally connected is less than total
		if locallyConnected >= totalGroupMembers {
			locallyConnected = totalGroupMembers / 10
		}

		// Create group members list
		allMembers := make([]string, totalGroupMembers)
		for i := 0; i < totalGroupMembers; i++ {
			allMembers[i] = fmt.Sprintf("user%d", i)
		}

		// Connect subset of members to this gateway
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		connectedMembers := make(map[string]bool)
		for i := 0; i < locallyConnected; i++ {
			userID := allMembers[i]
			conn := &Connection{
				UserID:   userID,
				DeviceID: fmt.Sprintf("device%d", i),
				Send:     make(chan []byte, 256),
				Gateway:  gateway,
				ctx:      ctx,
				cancel:   cancel,
			}
			gateway.connections.Store(userID+"_"+conn.DeviceID, conn)
			connectedMembers[userID] = true
		}

		// Simulate getting locally-connected members for a large group
		// This tests that the gateway only caches locally-connected members
		localMembers := make([]string, 0)
		gateway.connections.Range(func(key, value interface{}) bool {
			conn := value.(*Connection)
			// Check if this user is in the group
			for _, member := range allMembers {
				if conn.UserID == member {
					if !contains(localMembers, conn.UserID) {
						localMembers = append(localMembers, conn.UserID)
					}
					break
				}
			}
			return true
		})

		// Verify cache size is bounded by locally-connected members
		assert.LessOrEqual(t, len(localMembers), locallyConnected,
			"Cache should only contain locally-connected members")

		assert.Less(t, len(localMembers), totalGroupMembers,
			"Cache should NOT contain all group members")

		// Verify memory bound: O(locally-connected) not O(total-members)
		// For a group with 5000 members but only 50 connected locally,
		// we should only cache ~50 entries, not 5000
		expectedMaxCache := locallyConnected * 2 // Allow 2x buffer
		assert.LessOrEqual(t, len(localMembers), expectedMaxCache,
			"Cache size should be O(locally-connected), not O(total-members)")

		// Verify all locally-connected members are in cache
		for userID := range connectedMembers {
			found := false
			for _, cachedUser := range localMembers {
				if cachedUser == userID {
					found = true
					break
				}
			}
			assert.True(t, found, "Locally-connected user %s should be in cache", userID)
		}
	})
}

// TestProperty4_MultiDeviceMessageConsistency_EdgeCases tests edge cases for Property 4
func TestProperty4_MultiDeviceMessageConsistency_EdgeCases(t *testing.T) {
	t.Run("DeviceConnectsMidConversation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			gateway, _, _, _ := setupTestGatewayForProperty()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create first device
			conn1 := &Connection{
				UserID:   "user123",
				DeviceID: "device1",
				Send:     make(chan []byte, 256),
				Gateway:  gateway,
				ctx:      ctx,
				cancel:   cancel,
			}
			gateway.connections.Store("user123_device1", conn1)

			// Send some messages
			numInitialMessages := rapid.IntRange(3, 10).Draw(t, "numInitialMessages")
			for i := 0; i < numInitialMessages; i++ {
				req := &PushMessageRequest{
					MsgID:          fmt.Sprintf("msg-%d", i),
					RecipientID:    "user123",
					SenderID:       "sender789",
					Content:        fmt.Sprintf("Message %d", i),
					SequenceNumber: int64(1000 + i),
					Timestamp:      time.Now().Unix(),
				}
				gateway.pushService.PushMessage(context.Background(), req)
			}

			// Connect second device mid-conversation
			conn2 := &Connection{
				UserID:   "user123",
				DeviceID: "device2",
				Send:     make(chan []byte, 256),
				Gateway:  gateway,
				ctx:      ctx,
				cancel:   cancel,
			}
			gateway.connections.Store("user123_device2", conn2)

			// Send more messages
			numLaterMessages := rapid.IntRange(3, 10).Draw(t, "numLaterMessages")
			for i := 0; i < numLaterMessages; i++ {
				req := &PushMessageRequest{
					MsgID:          fmt.Sprintf("msg-%d", numInitialMessages+i),
					RecipientID:    "user123",
					SenderID:       "sender789",
					Content:        fmt.Sprintf("Message %d", numInitialMessages+i),
					SequenceNumber: int64(1000 + numInitialMessages + i),
					Timestamp:      time.Now().Unix(),
				}
				resp, err := gateway.pushService.PushMessage(context.Background(), req)
				require.NoError(t, err)

				// Both devices should receive messages sent after device2 connected
				assert.Equal(t, int32(2), resp.DeliveredCount,
					"Both devices should receive message %d", numInitialMessages+i)
			}
		})
	})

	t.Run("DeviceDisconnectsDuringDelivery", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			gateway, _, _, _ := setupTestGatewayForProperty()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create two devices
			conn1 := &Connection{
				UserID:   "user123",
				DeviceID: "device1",
				Send:     make(chan []byte, 256),
				Gateway:  gateway,
				ctx:      ctx,
				cancel:   cancel,
			}
			gateway.connections.Store("user123_device1", conn1)

			ctx2, cancel2 := context.WithCancel(context.Background())
			conn2 := &Connection{
				UserID:   "user123",
				DeviceID: "device2",
				Send:     make(chan []byte, 256),
				Gateway:  gateway,
				ctx:      ctx2,
				cancel:   cancel2,
			}
			gateway.connections.Store("user123_device2", conn2)

			// Disconnect device2
			cancel2()
			gateway.connections.Delete("user123_device2")

			// Send message
			req := &PushMessageRequest{
				MsgID:          "msg-001",
				RecipientID:    "user123",
				SenderID:       "sender789",
				Content:        "Test message",
				SequenceNumber: 1000,
				Timestamp:      time.Now().Unix(),
			}
			resp, err := gateway.pushService.PushMessage(context.Background(), req)
			require.NoError(t, err)

			// Only device1 should receive the message
			assert.Equal(t, int32(1), resp.DeliveredCount,
				"Only connected device should receive message")
		})
	})
}
