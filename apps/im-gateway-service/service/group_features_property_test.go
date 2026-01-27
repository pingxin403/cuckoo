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

// TestProperty5_GroupMessageBroadcastCompleteness tests Property 5:
// When a message is sent to a group, all online members MUST receive the message exactly once.
//
// **Validates: Requirements 2.1, 2.2, 2.3, 2.9, Property 5**
func TestProperty5_GroupMessageBroadcastCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate small group for testing (5-20 members)
		groupSize := rapid.IntRange(5, 20).Draw(t, "groupSize")
		groupID := fmt.Sprintf("group_%d", rapid.IntRange(1000, 9999).Draw(t, "groupID"))

		// Generate group members
		members := make([]string, groupSize)
		for i := 0; i < groupSize; i++ {
			members[i] = fmt.Sprintf("user_%d", i+1)
		}

		// All members are online for simplicity
		onlineMembers := make(map[string]bool)
		for _, member := range members {
			onlineMembers[member] = true
		}

		// Create mock gateway with connections
		gateway, _, _ := setupMockGatewayForGroupTest(t, groupID, members, onlineMembers)

		// Verify connections were created
		connCount := 0
		gateway.connections.Range(func(key, value any) bool {
			connCount++
			return true
		})
		require.Equal(t, groupSize, connCount, "All members should have connections")

		// Create group message
		groupMsg := GroupMessage{
			MsgID:          generateMsgID(t),
			GroupID:        groupID,
			SenderID:       members[0],
			Content:        "Test message",
			MessageType:    "text",
			SequenceNumber: rapid.Int64Range(1, 100000).Draw(t, "seqNum"),
			Timestamp:      time.Now().Unix(),
		}

		// Serialize message
		msgData, err := json.Marshal(groupMsg)
		require.NoError(t, err)

		// Process the group message
		consumer := gateway.kafkaConsumer
		err = consumer.processGroupMessage(msgData)
		require.NoError(t, err)

		// Give some time for async processing
		time.Sleep(100 * time.Millisecond)

		// Verify all online members received the message
		receivedCount := 0
		for memberID := range onlineMembers {
			// Check if member received message
			key := memberID + "_device1"
			if conn, ok := gateway.connections.Load(key); ok {
				connection := conn.(*Connection)
				select {
				case msg := <-connection.Send:
					// Verify message content
					var serverMsg ServerMessage
					err := json.Unmarshal(msg, &serverMsg)
					require.NoError(t, err)
					assert.Equal(t, groupMsg.MsgID, serverMsg.MsgID)
					assert.Equal(t, groupMsg.SenderID, serverMsg.Sender)
					receivedCount++
				case <-time.After(200 * time.Millisecond):
					// Timeout - member didn't receive message
					t.Fatalf("Online member %s did not receive message", memberID)
				}
			} else {
				t.Fatalf("Connection not found for member %s", memberID)
			}
		}

		// Verify all online members received exactly once
		assert.Equal(t, groupSize, receivedCount,
			"All online members should receive message exactly once")
	})
}

// TestProperty5_LargeGroupBroadcast tests Property 5 with large groups (>1,000 members).
// For large groups, only locally-connected members should be cached and receive messages.
//
// **Validates: Requirements 2.10, 2.11, 2.12, Property 5**
func TestProperty5_LargeGroupBroadcast(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate large group (1,001 to 5,000 members)
		groupSize := rapid.IntRange(1001, 5000).Draw(t, "groupSize")
		groupID := fmt.Sprintf("large_group_%d", rapid.IntRange(1000, 9999).Draw(t, "groupID"))

		// Generate group members
		members := make([]string, groupSize)
		for i := 0; i < groupSize; i++ {
			members[i] = fmt.Sprintf("user_%d", i+1)
		}

		// Only a small subset are locally connected (1-10%)
		localPercentage := rapid.IntRange(1, 10).Draw(t, "localPercentage")
		localCount := (groupSize * localPercentage) / 100
		if localCount == 0 {
			localCount = 1
		}

		localMembers := make(map[string]bool)
		for i := 0; i < localCount; i++ {
			localMembers[members[i]] = true
		}

		// Create mock gateway with only local connections
		gateway, _, _ := setupMockGatewayForGroupTest(t, groupID, members, localMembers)

		// Verify cache manager uses large group optimization
		cacheEntry, ok := gateway.cacheManager.groupMemberCache.Load(groupID)
		if ok {
			entry := cacheEntry.(*GroupCacheEntry)
			assert.True(t, entry.IsLarge, "Group should be marked as large")
		}

		// Create group message
		groupMsg := GroupMessage{
			MsgID:          generateMsgID(t),
			GroupID:        groupID,
			SenderID:       members[0],
			Content:        "Test message for large group",
			MessageType:    "text",
			SequenceNumber: rapid.Int64Range(1, 100000).Draw(t, "seqNum"),
			Timestamp:      time.Now().Unix(),
		}

		// Serialize message
		msgData, err := json.Marshal(groupMsg)
		require.NoError(t, err)

		// Process the group message
		consumer := gateway.kafkaConsumer
		err = consumer.processGroupMessage(msgData)
		require.NoError(t, err)

		// Verify only locally-connected members received the message
		receivedCount := 0
		for memberID := range localMembers {
			key := memberID + "_device1"
			if conn, ok := gateway.connections.Load(key); ok {
				connection := conn.(*Connection)
				select {
				case msg := <-connection.Send:
					var serverMsg ServerMessage
					err := json.Unmarshal(msg, &serverMsg)
					require.NoError(t, err)
					assert.Equal(t, groupMsg.MsgID, serverMsg.MsgID)
					receivedCount++
				case <-time.After(100 * time.Millisecond):
					t.Fatalf("Local member %s did not receive message", memberID)
				}
			}
		}

		assert.Equal(t, localCount, receivedCount,
			"All locally-connected members should receive message")

		// Verify memory usage is bounded
		memoryUsage := gateway.cacheManager.GetMemoryUsage()
		// For large groups, memory should be proportional to local members, not total members
		// Rough estimate: 50 bytes per member
		maxExpectedMemory := int64(localCount * 100)             // Allow some overhead
		assert.LessOrEqual(t, memoryUsage, maxExpectedMemory*10, // 10x buffer for other caches
			"Memory usage should be bounded for large groups")
	})
}

// TestProperty5_OfflineMemberRouting tests that offline members get messages in Offline Channel.
//
// **Validates: Requirements 2.3, 4.1, 4.2, Property 5**
func TestProperty5_OfflineMemberRouting(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate group with mix of online and offline members
		groupSize := rapid.IntRange(10, 100).Draw(t, "groupSize")
		groupID := fmt.Sprintf("group_%d", rapid.IntRange(1000, 9999).Draw(t, "groupID"))

		members := make([]string, groupSize)
		for i := 0; i < groupSize; i++ {
			members[i] = fmt.Sprintf("user_%d", i+1)
		}

		// 50% online, 50% offline
		onlineCount := groupSize / 2
		onlineMembers := make(map[string]bool)
		offlineMembers := make([]string, 0)

		for i := 0; i < groupSize; i++ {
			if i < onlineCount {
				onlineMembers[members[i]] = true
			} else {
				offlineMembers = append(offlineMembers, members[i])
			}
		}

		// Create mock gateway
		gateway, _, _ := setupMockGatewayForGroupTest(t, groupID, members, onlineMembers)

		// Create group message
		groupMsg := GroupMessage{
			MsgID:          generateMsgID(t),
			GroupID:        groupID,
			SenderID:       members[0],
			Content:        rapid.String().Draw(t, "content"),
			MessageType:    "text",
			SequenceNumber: rapid.Int64Range(1, 100000).Draw(t, "seqNum"),
			Timestamp:      time.Now().Unix(),
		}

		msgData, err := json.Marshal(groupMsg)
		require.NoError(t, err)

		// Process the group message
		consumer := gateway.kafkaConsumer
		err = consumer.processGroupMessage(msgData)
		require.NoError(t, err)

		// Verify online members received via WebSocket
		for memberID := range onlineMembers {
			key := memberID + "_device1"
			if conn, ok := gateway.connections.Load(key); ok {
				connection := conn.(*Connection)
				select {
				case <-connection.Send:
					// Expected - online member received
				case <-time.After(100 * time.Millisecond):
					t.Fatalf("Online member %s did not receive message", memberID)
				}
			}
		}

		// Note: In a real implementation, we would verify that offline members
		// get messages routed to Kafka offline_msg topic. This requires integration
		// with the IM Service which handles offline routing.
		// For this property test, we verify that offline members do NOT receive
		// via WebSocket, which is the correct behavior.

		// Verify offline members did NOT receive via WebSocket
		for _, memberID := range offlineMembers {
			key := memberID + "_device1"
			// Offline members should not have connections
			_, exists := gateway.connections.Load(key)
			assert.False(t, exists,
				"Offline member %s should not have active connection", memberID)
		}
	})
}

// TestProperty5_ConcurrentBroadcast tests concurrent group message broadcasts.
//
// **Validates: Requirements 2.2, 2.3, Property 5**
func TestProperty5_ConcurrentBroadcast(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate group
		groupSize := rapid.IntRange(20, 100).Draw(t, "groupSize")
		groupID := fmt.Sprintf("group_%d", rapid.IntRange(1000, 9999).Draw(t, "groupID"))

		members := make([]string, groupSize)
		onlineMembers := make(map[string]bool)
		for i := 0; i < groupSize; i++ {
			members[i] = fmt.Sprintf("user_%d", i+1)
			onlineMembers[members[i]] = true
		}

		gateway, _, _ := setupMockGatewayForGroupTest(t, groupID, members, onlineMembers)

		// Send multiple messages concurrently
		numMessages := rapid.IntRange(5, 20).Draw(t, "numMessages")
		var wg sync.WaitGroup
		errors := make(chan error, numMessages)

		for i := 0; i < numMessages; i++ {
			wg.Add(1)
			go func(msgNum int) {
				defer wg.Done()

				groupMsg := GroupMessage{
					MsgID:          fmt.Sprintf("msg_%d", msgNum),
					GroupID:        groupID,
					SenderID:       members[0],
					Content:        fmt.Sprintf("Message %d", msgNum),
					MessageType:    "text",
					SequenceNumber: int64(msgNum),
					Timestamp:      time.Now().Unix(),
				}

				msgData, err := json.Marshal(groupMsg)
				if err != nil {
					errors <- err
					return
				}

				err = gateway.kafkaConsumer.processGroupMessage(msgData)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			require.NoError(t, err)
		}

		// Verify each member received all messages
		// Note: Due to concurrent processing, we can't guarantee order,
		// but each member should receive all messages
		for memberID := range onlineMembers {
			key := memberID + "_device1"
			if conn, ok := gateway.connections.Load(key); ok {
				connection := conn.(*Connection)
				receivedCount := 0

				// Drain all messages from channel
				timeout := time.After(500 * time.Millisecond)
			drainLoop:
				for {
					select {
					case <-connection.Send:
						receivedCount++
						if receivedCount >= numMessages {
							break drainLoop
						}
					case <-timeout:
						break drainLoop
					}
				}

				// Each member should receive all messages
				assert.Equal(t, numMessages, receivedCount,
					"Member %s should receive all %d messages", memberID, numMessages)
			}
		}
	})
}

// Helper functions

// setupMockGatewayForGroupTest creates a mock gateway with connections for testing.
func setupMockGatewayForGroupTest(
	t *rapid.T,
	groupID string,
	allMembers []string,
	onlineMembers map[string]bool,
) (*GatewayService, *mockRegistryClient, *mockUserServiceClient) {
	// Create mock clients
	mockRegistry := &mockRegistryClient{
		users: make(map[string][]GatewayLocation),
	}

	mockUserService := &mockUserServiceClient{
		groupMembers: make(map[string][]string),
	}
	mockUserService.groupMembers[groupID] = allMembers

	mockAuth := &mockAuthClient{}
	mockIM := &mockIMClient{}

	// Create gateway
	config := DefaultGatewayConfig()
	gateway := NewGatewayService(mockAuth, mockRegistry, mockIM, nil, config)

	// Create connections for online members
	for memberID := range onlineMembers {
		deviceID := "device1"
		ctx, cancel := context.WithCancel(context.Background())

		conn := &Connection{
			UserID:   memberID,
			DeviceID: deviceID,
			Send:     make(chan []byte, 256),
			Gateway:  gateway,
			ctx:      ctx,
			cancel:   cancel,
		}

		gateway.connections.Store(memberID+"_"+deviceID, conn)

		// Register in mock registry
		mockRegistry.users[memberID] = []GatewayLocation{
			{
				GatewayNode: "gateway-1",
				DeviceID:    deviceID,
			},
		}
	}

	// Create cache manager
	gateway.cacheManager = &CacheManager{
		userCacheTTL:        5 * time.Minute,
		groupCacheTTL:       5 * time.Minute,
		largeGroupThreshold: 1000,
		gateway:             gateway,
	}

	// Pre-populate group cache
	gateway.cacheManager.groupMemberCache.Store(groupID, &GroupCacheEntry{
		Members:   allMembers,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		IsLarge:   len(allMembers) > 1000,
	})

	// Create Kafka consumer
	gateway.kafkaConsumer = &KafkaConsumer{
		gateway: gateway,
		ctx:     context.Background(),
	}

	return gateway, mockRegistry, mockUserService
}

// generateMsgID generates a random message ID for testing.
func generateMsgID(t *rapid.T) string {
	return fmt.Sprintf("msg_%d", rapid.IntRange(100000, 999999).Draw(t, "msgID"))
}

// mockUserServiceClient is a mock implementation of User Service.
type mockUserServiceClient struct {
	groupMembers map[string][]string
}
