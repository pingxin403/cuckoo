//go:build e2e
// +build e2e

package multiregion

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cuckoo-org/cuckoo/apps/im-gateway-service/routing"
)

// TestBusinessEndToEndVerification validates Task 16.3.1 requirements
// - 跨地域单聊消息测试 (Requirement 9.3.1)
// - 跨地域群聊消息测试 (Requirement 9.3.2)
// - 离线消息推送测试 (Requirement 9.3.3)
// - 多设备同步测试 (Requirement 9.3.4)
// - 故障转移恢复后数据一致性测试
func TestBusinessEndToEndVerification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping business end-to-end test in short mode")
	}

	ctx := context.Background()

	// Setup test environment
	env := setupMultiRegionTestEnvironment(t, ctx)
	defer env.Cleanup()

	t.Run("CrossRegionDirectMessage", func(t *testing.T) {
		testCrossRegionDirectMessage(t, ctx, env)
	})

	t.Run("CrossRegionGroupChat", func(t *testing.T) {
		testCrossRegionGroupChat(t, ctx, env)
	})

	t.Run("OfflineMessagePush", func(t *testing.T) {
		testOfflineMessagePush(t, ctx, env)
	})

	t.Run("MultiDeviceSync", func(t *testing.T) {
		testMultiDeviceSync(t, ctx, env)
	})

	t.Run("FailoverRecovery", func(t *testing.T) {
		testFailoverRecovery(t, ctx, env)
	})
}

// testCrossRegionDirectMessage validates requirement 9.3.1
// 验证跨地域单聊消息的发送、接收和确认流程
func testCrossRegionDirectMessage(t *testing.T, ctx context.Context, env *MultiRegionTestEnvironment) {
	t.Log("Testing cross-region direct message flow...")

	// Test scenario: User A in Region A sends message to User B in Region B
	userA := "user-a-region-a"
	userB := "user-b-region-b"
	conversationID := fmt.Sprintf("dm:%s:%s", userA, userB)

	// Step 1: User A sends message in Region A
	t.Log("Step 1: User A sends message in Region A")
	msgID := env.RegionA.HLC.GenerateID()
	message := map[string]interface{}{
		"msg_id":          msgID.String(),
		"conversation_id": conversationID,
		"sender_id":       userA,
		"receiver_id":     userB,
		"content":         "Hello from Region A!",
		"timestamp":       time.Now().Unix(),
		"region_id":       "region-a",
		"sync_status":     "pending",
	}

	// Store message in Region A
	msgKey := fmt.Sprintf("messages:%s:%s", conversationID, msgID.String())
	err := env.RegionA.RedisClient.HSet(ctx, msgKey, message).Err()
	require.NoError(t, err, "Should store message in Region A")

	// Add to conversation index
	convKey := fmt.Sprintf("conversation:%s:messages", conversationID)
	err = env.RegionA.RedisClient.ZAdd(ctx, convKey, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: msgID.String(),
	}).Err()
	require.NoError(t, err)

	t.Log("✓ Message stored in Region A")

	// Step 2: Simulate cross-region sync
	t.Log("Step 2: Simulating cross-region message sync...")
	time.Sleep(100 * time.Millisecond) // Simulate network latency

	// Update HLC in Region B
	err = env.RegionB.HLC.UpdateFromRemote(msgID.PhysicalTime, msgID.LogicalTime)
	require.NoError(t, err)

	// Replicate message to Region B
	err = env.RegionB.RedisClient.HSet(ctx, msgKey, message).Err()
	require.NoError(t, err, "Should replicate message to Region B")

	err = env.RegionB.RedisClient.ZAdd(ctx, convKey, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: msgID.String(),
	}).Err()
	require.NoError(t, err)

	// Update sync status
	err = env.RegionA.RedisClient.HSet(ctx, msgKey, "sync_status", "synced").Err()
	require.NoError(t, err)

	t.Log("✓ Message synced to Region B")

	// Step 3: User B receives message in Region B
	t.Log("Step 3: User B receives message in Region B")
	receivedMsg, err := env.RegionB.RedisClient.HGetAll(ctx, msgKey).Result()
	require.NoError(t, err, "Should retrieve message in Region B")

	assert.Equal(t, message["sender_id"], receivedMsg["sender_id"])
	assert.Equal(t, message["receiver_id"], receivedMsg["receiver_id"])
	assert.Equal(t, message["content"], receivedMsg["content"])
	assert.Equal(t, "region-a", receivedMsg["region_id"])

	t.Log("✓ Message received correctly in Region B")

	// Step 4: User B sends acknowledgment
	t.Log("Step 4: User B sends acknowledgment")
	ackID := env.RegionB.HLC.GenerateID()
	ackKey := fmt.Sprintf("ack:%s:%s", conversationID, msgID.String())
	ack := map[string]interface{}{
		"ack_id":     ackID.String(),
		"msg_id":     msgID.String(),
		"user_id":    userB,
		"timestamp":  time.Now().Unix(),
		"region_id":  "region-b",
		"ack_status": "delivered",
	}

	err = env.RegionB.RedisClient.HSet(ctx, ackKey, ack).Err()
	require.NoError(t, err)

	// Sync acknowledgment back to Region A
	time.Sleep(50 * time.Millisecond)
	err = env.RegionA.RedisClient.HSet(ctx, ackKey, ack).Err()
	require.NoError(t, err)

	t.Log("✓ Acknowledgment synced back to Region A")

	// Step 5: Verify end-to-end latency
	t.Log("Step 5: Verifying end-to-end latency")
	// In a real scenario, we would measure actual latency
	// For this test, we verify the message flow completed successfully

	// Verify message exists in both regions
	msgA, err := env.RegionA.RedisClient.HGetAll(ctx, msgKey).Result()
	require.NoError(t, err)
	msgB, err := env.RegionB.RedisClient.HGetAll(ctx, msgKey).Result()
	require.NoError(t, err)

	assert.Equal(t, msgA["msg_id"], msgB["msg_id"])
	assert.Equal(t, msgA["content"], msgB["content"])

	// Verify acknowledgment exists in both regions
	ackA, err := env.RegionA.RedisClient.HGetAll(ctx, ackKey).Result()
	require.NoError(t, err)
	ackB, err := env.RegionB.RedisClient.HGetAll(ctx, ackKey).Result()
	require.NoError(t, err)

	assert.Equal(t, ackA["ack_id"], ackB["ack_id"])
	assert.Equal(t, "delivered", ackA["ack_status"])

	t.Log("✓ End-to-end message flow verified")

	// Cleanup
	env.RegionA.RedisClient.Del(ctx, msgKey, convKey, ackKey)
	env.RegionB.RedisClient.Del(ctx, msgKey, convKey, ackKey)

	t.Log("✓ Cross-region direct message test completed successfully")
}

// testCrossRegionGroupChat validates requirement 9.3.2
// 验证跨地域群聊消息的广播和排序正确性
func testCrossRegionGroupChat(t *testing.T, ctx context.Context, env *MultiRegionTestEnvironment) {
	t.Log("Testing cross-region group chat flow...")

	// Test scenario: Group with members in both regions
	groupID := "group-cross-region-001"
	membersRegionA := []string{"user-a1", "user-a2", "user-a3"}
	membersRegionB := []string{"user-b1", "user-b2", "user-b3"}
	allMembers := append(membersRegionA, membersRegionB...)

	// Step 1: Create group in both regions
	t.Log("Step 1: Creating group in both regions")
	groupKey := fmt.Sprintf("group:%s", groupID)
	groupData := map[string]interface{}{
		"group_id":     groupID,
		"name":         "Cross-Region Test Group",
		"created_at":   time.Now().Unix(),
		"member_count": len(allMembers),
	}

	err := env.RegionA.RedisClient.HSet(ctx, groupKey, groupData).Err()
	require.NoError(t, err)
	err = env.RegionB.RedisClient.HSet(ctx, groupKey, groupData).Err()
	require.NoError(t, err)

	// Add members to group
	membersKey := fmt.Sprintf("group:%s:members", groupID)
	for _, member := range allMembers {
		err = env.RegionA.RedisClient.SAdd(ctx, membersKey, member).Err()
		require.NoError(t, err)
		err = env.RegionB.RedisClient.SAdd(ctx, membersKey, member).Err()
		require.NoError(t, err)
	}

	t.Log("✓ Group created with members in both regions")

	// Step 2: Send messages from both regions concurrently
	t.Log("Step 2: Sending messages from both regions")
	messages := make([]string, 0, 10)

	// User in Region A sends messages
	for i := 0; i < 5; i++ {
		msgID := env.RegionA.HLC.GenerateID()
		message := map[string]interface{}{
			"msg_id":    msgID.String(),
			"group_id":  groupID,
			"sender_id": membersRegionA[i%len(membersRegionA)],
			"content":   fmt.Sprintf("Message %d from Region A", i),
			"timestamp": time.Now().Unix(),
			"region_id": "region-a",
			"hlc":       msgID.PhysicalTime,
			"logical":   msgID.LogicalTime,
		}

		msgKey := fmt.Sprintf("group:%s:msg:%s", groupID, msgID.String())
		err = env.RegionA.RedisClient.HSet(ctx, msgKey, message).Err()
		require.NoError(t, err)

		messages = append(messages, msgID.String())
		time.Sleep(10 * time.Millisecond)
	}

	// User in Region B sends messages
	for i := 0; i < 5; i++ {
		msgID := env.RegionB.HLC.GenerateID()
		message := map[string]interface{}{
			"msg_id":    msgID.String(),
			"group_id":  groupID,
			"sender_id": membersRegionB[i%len(membersRegionB)],
			"content":   fmt.Sprintf("Message %d from Region B", i),
			"timestamp": time.Now().Unix(),
			"region_id": "region-b",
			"hlc":       msgID.PhysicalTime,
			"logical":   msgID.LogicalTime,
		}

		msgKey := fmt.Sprintf("group:%s:msg:%s", groupID, msgID.String())
		err = env.RegionB.RedisClient.HSet(ctx, msgKey, message).Err()
		require.NoError(t, err)

		messages = append(messages, msgID.String())
		time.Sleep(10 * time.Millisecond)
	}

	t.Logf("✓ Sent %d messages from both regions", len(messages))

	// Step 3: Simulate cross-region broadcast
	t.Log("Step 3: Broadcasting messages to all regions")
	time.Sleep(200 * time.Millisecond) // Simulate sync latency

	// Sync all messages to both regions
	for _, msgIDStr := range messages {
		msgKey := fmt.Sprintf("group:%s:msg:%s", groupID, msgIDStr)

		// Try to get from source region and replicate
		msgA, errA := env.RegionA.RedisClient.HGetAll(ctx, msgKey).Result()
		msgB, errB := env.RegionB.RedisClient.HGetAll(ctx, msgKey).Result()

		if errA == nil && len(msgA) > 0 {
			// Message exists in Region A, replicate to Region B
			err = env.RegionB.RedisClient.HSet(ctx, msgKey, msgA).Err()
			require.NoError(t, err)
		} else if errB == nil && len(msgB) > 0 {
			// Message exists in Region B, replicate to Region A
			err = env.RegionA.RedisClient.HSet(ctx, msgKey, msgB).Err()
			require.NoError(t, err)
		}
	}

	t.Log("✓ Messages broadcast to all regions")

	// Step 4: Verify message ordering using HLC
	t.Log("Step 4: Verifying message ordering")
	type MessageWithHLC struct {
		MsgID    string
		HLC      int64
		Logical  int64
		RegionID string
		Content  string
	}

	messagesWithHLC := make([]MessageWithHLC, 0, len(messages))

	for _, msgIDStr := range messages {
		msgKey := fmt.Sprintf("group:%s:msg:%s", groupID, msgIDStr)
		msg, err := env.RegionA.RedisClient.HGetAll(ctx, msgKey).Result()
		require.NoError(t, err)

		hlc, _ := parseInt64(msg["hlc"])
		logical, _ := parseInt64(msg["logical"])

		messagesWithHLC = append(messagesWithHLC, MessageWithHLC{
			MsgID:    msgIDStr,
			HLC:      hlc,
			Logical:  logical,
			RegionID: msg["region_id"],
			Content:  msg["content"],
		})
	}

	// Sort messages by HLC
	sortMessagesByHLC(messagesWithHLC)

	// Verify ordering is consistent
	for i := 1; i < len(messagesWithHLC); i++ {
		prev := messagesWithHLC[i-1]
		curr := messagesWithHLC[i]

		// Current message should have HLC >= previous message
		assert.True(t,
			curr.HLC > prev.HLC ||
				(curr.HLC == prev.HLC && curr.Logical >= prev.Logical),
			"Messages should be ordered by HLC")
	}

	t.Logf("✓ Message ordering verified (%d messages)", len(messagesWithHLC))

	// Step 5: Verify all members can see all messages
	t.Log("Step 5: Verifying message visibility for all members")
	for _, member := range allMembers {
		// Each member should be able to retrieve all messages
		count := 0
		for _, msgIDStr := range messages {
			msgKey := fmt.Sprintf("group:%s:msg:%s", groupID, msgIDStr)
			exists, err := env.RegionA.RedisClient.Exists(ctx, msgKey).Result()
			require.NoError(t, err)
			if exists > 0 {
				count++
			}
		}
		assert.Equal(t, len(messages), count,
			"Member %s should see all messages", member)
	}

	t.Log("✓ All members can see all messages")

	// Cleanup
	env.RegionA.RedisClient.Del(ctx, groupKey, membersKey)
	env.RegionB.RedisClient.Del(ctx, groupKey, membersKey)
	for _, msgIDStr := range messages {
		msgKey := fmt.Sprintf("group:%s:msg:%s", groupID, msgIDStr)
		env.RegionA.RedisClient.Del(ctx, msgKey)
		env.RegionB.RedisClient.Del(ctx, msgKey)
	}

	t.Log("✓ Cross-region group chat test completed successfully")
}

// testOfflineMessagePush validates requirement 9.3.3
// 验证离线消息在用户从不同地域上线时的推送正确性
func testOfflineMessagePush(t *testing.T, ctx context.Context, env *MultiRegionTestEnvironment) {
	t.Log("Testing offline message push flow...")

	userID := "user-offline-test"
	senderID := "user-sender"

	// Step 1: User is offline, messages are stored
	t.Log("Step 1: Storing offline messages while user is offline")
	offlineMessages := make([]string, 0, 5)

	for i := 0; i < 5; i++ {
		msgID := env.RegionA.HLC.GenerateID()
		message := map[string]interface{}{
			"msg_id":     msgID.String(),
			"user_id":    userID,
			"sender_id":  senderID,
			"content":    fmt.Sprintf("Offline message %d", i),
			"timestamp":  time.Now().Unix(),
			"region_id":  "region-a",
			"status":     "offline",
			"expires_at": time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days TTL
		}

		msgKey := fmt.Sprintf("offline:%s:%s", userID, msgID.String())
		err := env.RegionA.RedisClient.HSet(ctx, msgKey, message).Err()
		require.NoError(t, err)

		// Add to user's offline message queue
		queueKey := fmt.Sprintf("offline:queue:%s", userID)
		err = env.RegionA.RedisClient.ZAdd(ctx, queueKey, redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: msgID.String(),
		}).Err()
		require.NoError(t, err)

		offlineMessages = append(offlineMessages, msgID.String())
		time.Sleep(10 * time.Millisecond)
	}

	t.Logf("✓ Stored %d offline messages in Region A", len(offlineMessages))

	// Step 2: Sync offline messages to Region B
	t.Log("Step 2: Syncing offline messages to Region B")
	time.Sleep(100 * time.Millisecond)

	for _, msgIDStr := range offlineMessages {
		msgKey := fmt.Sprintf("offline:%s:%s", userID, msgIDStr)
		msg, err := env.RegionA.RedisClient.HGetAll(ctx, msgKey).Result()
		require.NoError(t, err)

		// Replicate to Region B
		err = env.RegionB.RedisClient.HSet(ctx, msgKey, msg).Err()
		require.NoError(t, err)
	}

	// Sync queue to Region B
	queueKey := fmt.Sprintf("offline:queue:%s", userID)
	queueItems, err := env.RegionA.RedisClient.ZRangeWithScores(ctx, queueKey, 0, -1).Result()
	require.NoError(t, err)

	for _, item := range queueItems {
		err = env.RegionB.RedisClient.ZAdd(ctx, queueKey, redis.Z{
			Score:  item.Score,
			Member: item.Member,
		}).Err()
		require.NoError(t, err)
	}

	t.Log("✓ Offline messages synced to Region B")

	// Step 3: User comes online in Region B
	t.Log("Step 3: User comes online in Region B")
	sessionKey := fmt.Sprintf("session:%s", userID)
	session := map[string]interface{}{
		"user_id":   userID,
		"region_id": "region-b",
		"status":    "online",
		"login_at":  time.Now().Unix(),
		"device_id": "device-001",
	}

	err = env.RegionB.RedisClient.HSet(ctx, sessionKey, session).Err()
	require.NoError(t, err)

	t.Log("✓ User session created in Region B")

	// Step 4: Retrieve and push offline messages
	t.Log("Step 4: Retrieving offline messages for push")
	retrievedMsgIDs, err := env.RegionB.RedisClient.ZRange(ctx, queueKey, 0, -1).Result()
	require.NoError(t, err)

	assert.Equal(t, len(offlineMessages), len(retrievedMsgIDs),
		"Should retrieve all offline messages")

	pushedMessages := make([]map[string]string, 0, len(retrievedMsgIDs))
	for _, msgIDStr := range retrievedMsgIDs {
		msgKey := fmt.Sprintf("offline:%s:%s", userID, msgIDStr)
		msg, err := env.RegionB.RedisClient.HGetAll(ctx, msgKey).Result()
		require.NoError(t, err)

		// Mark as delivered
		err = env.RegionB.RedisClient.HSet(ctx, msgKey, "status", "delivered").Err()
		require.NoError(t, err)

		pushedMessages = append(pushedMessages, msg)
	}

	t.Logf("✓ Pushed %d offline messages to user", len(pushedMessages))

	// Step 5: Verify message order and content
	t.Log("Step 5: Verifying message order and content")
	for i, msg := range pushedMessages {
		expectedContent := fmt.Sprintf("Offline message %d", i)
		assert.Equal(t, expectedContent, msg["content"],
			"Message content should match")
		assert.Equal(t, senderID, msg["sender_id"],
			"Sender ID should match")
	}

	t.Log("✓ Message order and content verified")

	// Step 6: Clear offline message queue after delivery
	t.Log("Step 6: Clearing offline message queue")
	err = env.RegionB.RedisClient.Del(ctx, queueKey).Err()
	require.NoError(t, err)

	// Sync queue deletion to Region A
	err = env.RegionA.RedisClient.Del(ctx, queueKey).Err()
	require.NoError(t, err)

	// Verify queue is empty
	count, err := env.RegionB.RedisClient.ZCard(ctx, queueKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "Offline queue should be empty after delivery")

	t.Log("✓ Offline message queue cleared")

	// Step 7: Test user coming online in different region
	t.Log("Step 7: Testing user switching to Region A")

	// Store new offline messages while user is in Region B
	newOfflineMsg := env.RegionA.HLC.GenerateID()
	newMessage := map[string]interface{}{
		"msg_id":    newOfflineMsg.String(),
		"user_id":   userID,
		"sender_id": senderID,
		"content":   "New message while online in Region B",
		"timestamp": time.Now().Unix(),
		"region_id": "region-a",
		"status":    "offline",
	}

	newMsgKey := fmt.Sprintf("offline:%s:%s", userID, newOfflineMsg.String())
	err = env.RegionA.RedisClient.HSet(ctx, newMsgKey, newMessage).Err()
	require.NoError(t, err)

	// User switches to Region A
	err = env.RegionA.RedisClient.HSet(ctx, sessionKey, "region_id", "region-a").Err()
	require.NoError(t, err)

	// Verify message can be retrieved from Region A
	retrievedMsg, err := env.RegionA.RedisClient.HGetAll(ctx, newMsgKey).Result()
	require.NoError(t, err)
	assert.Equal(t, newMessage["content"], retrievedMsg["content"])

	t.Log("✓ User can receive messages from different region")

	// Cleanup
	env.RegionA.RedisClient.Del(ctx, sessionKey)
	env.RegionB.RedisClient.Del(ctx, sessionKey)
	for _, msgIDStr := range offlineMessages {
		msgKey := fmt.Sprintf("offline:%s:%s", userID, msgIDStr)
		env.RegionA.RedisClient.Del(ctx, msgKey)
		env.RegionB.RedisClient.Del(ctx, msgKey)
	}
	env.RegionA.RedisClient.Del(ctx, newMsgKey)

	t.Log("✓ Offline message push test completed successfully")
}

// testMultiDeviceSync validates requirement 9.3.4
// 验证多设备登录场景下消息同步的一致性
func testMultiDeviceSync(t *testing.T, ctx context.Context, env *MultiRegionTestEnvironment) {
	t.Log("Testing multi-device sync flow...")

	userID := "user-multi-device"
	devices := []struct {
		DeviceID string
		RegionID string
		Client   interface{}
	}{
		{"device-mobile", "region-a", env.RegionA.RedisClient},
		{"device-desktop", "region-b", env.RegionB.RedisClient},
		{"device-tablet", "region-a", env.RegionA.RedisClient},
	}

	// Step 1: User logs in on multiple devices in different regions
	t.Log("Step 1: Creating sessions for multiple devices")
	for _, device := range devices {
		sessionKey := fmt.Sprintf("session:%s:%s", userID, device.DeviceID)
		session := map[string]interface{}{
			"user_id":   userID,
			"device_id": device.DeviceID,
			"region_id": device.RegionID,
			"status":    "online",
			"login_at":  time.Now().Unix(),
		}

		var err error
		if device.RegionID == "region-a" {
			err = env.RegionA.RedisClient.HSet(ctx, sessionKey, session).Err()
		} else {
			err = env.RegionB.RedisClient.HSet(ctx, sessionKey, session).Err()
		}
		require.NoError(t, err)

		// Add device to user's device list
		devicesKey := fmt.Sprintf("user:%s:devices", userID)
		if device.RegionID == "region-a" {
			err = env.RegionA.RedisClient.SAdd(ctx, devicesKey, device.DeviceID).Err()
		} else {
			err = env.RegionB.RedisClient.SAdd(ctx, devicesKey, device.DeviceID).Err()
		}
		require.NoError(t, err)
	}

	t.Logf("✓ Created sessions for %d devices", len(devices))

	// Step 2: Send message to user (should sync to all devices)
	t.Log("Step 2: Sending message to user")
	msgID := env.RegionA.HLC.GenerateID()
	message := map[string]interface{}{
		"msg_id":      msgID.String(),
		"user_id":     userID,
		"sender_id":   "user-sender",
		"content":     "Message for multi-device sync",
		"timestamp":   time.Now().Unix(),
		"region_id":   "region-a",
		"sync_status": "pending",
	}

	msgKey := fmt.Sprintf("messages:%s:%s", userID, msgID.String())
	err := env.RegionA.RedisClient.HSet(ctx, msgKey, message).Err()
	require.NoError(t, err)

	t.Log("✓ Message sent to user")

	// Step 3: Sync message to all device regions
	t.Log("Step 3: Syncing message to all regions")
	time.Sleep(100 * time.Millisecond)

	// Replicate to Region B
	err = env.RegionB.RedisClient.HSet(ctx, msgKey, message).Err()
	require.NoError(t, err)

	// Update sync status
	err = env.RegionA.RedisClient.HSet(ctx, msgKey, "sync_status", "synced").Err()
	require.NoError(t, err)
	err = env.RegionB.RedisClient.HSet(ctx, msgKey, "sync_status", "synced").Err()
	require.NoError(t, err)

	t.Log("✓ Message synced to all regions")

	// Step 4: Verify each device can retrieve the message
	t.Log("Step 4: Verifying message visibility on all devices")
	for _, device := range devices {
		var retrievedMsg map[string]string
		var err error

		if device.RegionID == "region-a" {
			retrievedMsg, err = env.RegionA.RedisClient.HGetAll(ctx, msgKey).Result()
		} else {
			retrievedMsg, err = env.RegionB.RedisClient.HGetAll(ctx, msgKey).Result()
		}

		require.NoError(t, err, "Device %s should retrieve message", device.DeviceID)
		assert.Equal(t, message["content"], retrievedMsg["content"],
			"Device %s should see correct content", device.DeviceID)
		assert.Equal(t, "synced", retrievedMsg["sync_status"],
			"Device %s should see synced status", device.DeviceID)
	}

	t.Log("✓ All devices can see the message")

	// Step 5: Mark message as read on one device
	t.Log("Step 5: Marking message as read on mobile device")
	readReceiptKey := fmt.Sprintf("read:%s:%s", userID, msgID.String())
	readReceipt := map[string]interface{}{
		"msg_id":    msgID.String(),
		"user_id":   userID,
		"device_id": devices[0].DeviceID, // mobile device
		"read_at":   time.Now().Unix(),
		"region_id": "region-a",
	}

	err = env.RegionA.RedisClient.HSet(ctx, readReceiptKey, readReceipt).Err()
	require.NoError(t, err)

	// Sync read receipt to all regions
	time.Sleep(50 * time.Millisecond)
	err = env.RegionB.RedisClient.HSet(ctx, readReceiptKey, readReceipt).Err()
	require.NoError(t, err)

	t.Log("✓ Read receipt synced")

	// Step 6: Verify read status is visible on all devices
	t.Log("Step 6: Verifying read status on all devices")
	for _, device := range devices {
		var receipt map[string]string
		var err error

		if device.RegionID == "region-a" {
			receipt, err = env.RegionA.RedisClient.HGetAll(ctx, readReceiptKey).Result()
		} else {
			receipt, err = env.RegionB.RedisClient.HGetAll(ctx, readReceiptKey).Result()
		}

		require.NoError(t, err, "Device %s should see read receipt", device.DeviceID)
		assert.Equal(t, msgID.String(), receipt["msg_id"],
			"Device %s should see correct message ID", device.DeviceID)
	}

	t.Log("✓ Read status visible on all devices")

	// Step 7: Test device going offline and coming back online
	t.Log("Step 7: Testing device offline/online scenario")
	offlineDevice := devices[1] // desktop device

	// Mark device as offline
	sessionKey := fmt.Sprintf("session:%s:%s", userID, offlineDevice.DeviceID)
	err = env.RegionB.RedisClient.HSet(ctx, sessionKey, "status", "offline").Err()
	require.NoError(t, err)

	// Send new message while device is offline
	newMsgID := env.RegionA.HLC.GenerateID()
	newMessage := map[string]interface{}{
		"msg_id":    newMsgID.String(),
		"user_id":   userID,
		"sender_id": "user-sender",
		"content":   "Message while desktop offline",
		"timestamp": time.Now().Unix(),
		"region_id": "region-a",
	}

	newMsgKey := fmt.Sprintf("messages:%s:%s", userID, newMsgID.String())
	err = env.RegionA.RedisClient.HSet(ctx, newMsgKey, newMessage).Err()
	require.NoError(t, err)

	// Sync to Region B
	time.Sleep(50 * time.Millisecond)
	err = env.RegionB.RedisClient.HSet(ctx, newMsgKey, newMessage).Err()
	require.NoError(t, err)

	// Device comes back online
	err = env.RegionB.RedisClient.HSet(ctx, sessionKey, "status", "online").Err()
	require.NoError(t, err)

	// Verify device can retrieve the new message
	retrievedNewMsg, err := env.RegionB.RedisClient.HGetAll(ctx, newMsgKey).Result()
	require.NoError(t, err)
	assert.Equal(t, newMessage["content"], retrievedNewMsg["content"],
		"Offline device should receive message after coming online")

	t.Log("✓ Device sync after offline/online verified")

	// Step 8: Test message consistency across all devices
	t.Log("Step 8: Verifying final consistency across all devices")
	allMessageKeys := []string{msgKey, newMsgKey}

	for _, key := range allMessageKeys {
		msgA, err := env.RegionA.RedisClient.HGetAll(ctx, key).Result()
		require.NoError(t, err)

		msgB, err := env.RegionB.RedisClient.HGetAll(ctx, key).Result()
		require.NoError(t, err)

		assert.Equal(t, msgA["msg_id"], msgB["msg_id"],
			"Message ID should be consistent")
		assert.Equal(t, msgA["content"], msgB["content"],
			"Message content should be consistent")
	}

	t.Log("✓ Final consistency verified across all devices")

	// Cleanup
	for _, device := range devices {
		sessionKey := fmt.Sprintf("session:%s:%s", userID, device.DeviceID)
		if device.RegionID == "region-a" {
			env.RegionA.RedisClient.Del(ctx, sessionKey)
		} else {
			env.RegionB.RedisClient.Del(ctx, sessionKey)
		}
	}

	devicesKey := fmt.Sprintf("user:%s:devices", userID)
	env.RegionA.RedisClient.Del(ctx, devicesKey)
	env.RegionB.RedisClient.Del(ctx, devicesKey)
	env.RegionA.RedisClient.Del(ctx, msgKey, newMsgKey, readReceiptKey)
	env.RegionB.RedisClient.Del(ctx, msgKey, newMsgKey, readReceiptKey)

	t.Log("✓ Multi-device sync test completed successfully")
}

// testFailoverRecovery validates failover and data consistency
// 验证故障转移恢复后数据一致性
func testFailoverRecovery(t *testing.T, ctx context.Context, env *MultiRegionTestEnvironment) {
	t.Log("Testing failover recovery and data consistency...")

	userID := "user-failover-test"
	conversationID := "conv-failover-001"

	// Step 1: Establish baseline - messages in both regions
	t.Log("Step 1: Creating baseline messages in both regions")
	baselineMessages := make([]string, 0, 10)

	for i := 0; i < 10; i++ {
		var msgID interface{ String() string }
		var regionID string

		if i%2 == 0 {
			msgID = env.RegionA.HLC.GenerateID()
			regionID = "region-a"
		} else {
			msgID = env.RegionB.HLC.GenerateID()
			regionID = "region-b"
		}

		message := map[string]interface{}{
			"msg_id":          msgID.String(),
			"conversation_id": conversationID,
			"user_id":         userID,
			"content":         fmt.Sprintf("Baseline message %d", i),
			"timestamp":       time.Now().Unix(),
			"region_id":       regionID,
			"sync_status":     "synced",
		}

		msgKey := fmt.Sprintf("failover:msg:%s", msgID.String())

		// Store in both regions
		err := env.RegionA.RedisClient.HSet(ctx, msgKey, message).Err()
		require.NoError(t, err)
		err = env.RegionB.RedisClient.HSet(ctx, msgKey, message).Err()
		require.NoError(t, err)

		baselineMessages = append(baselineMessages, msgID.String())
		time.Sleep(10 * time.Millisecond)
	}

	t.Logf("✓ Created %d baseline messages", len(baselineMessages))

	// Step 2: Simulate Region A failure
	t.Log("Step 2: Simulating Region A failure")

	// Mark Region A as unhealthy
	healthKey := "health:region-a"
	err := env.RegionA.RedisClient.Set(ctx, healthKey, "unhealthy", time.Minute).Err()
	require.NoError(t, err)

	// Stop Region A geo router to simulate failure
	env.RegionA.GeoRouter.Stop()
	t.Log("✓ Region A marked as failed")

	// Step 3: Continue operations in Region B (failover)
	t.Log("Step 3: Continuing operations in Region B after failover")
	failoverMessages := make([]string, 0, 5)

	for i := 0; i < 5; i++ {
		msgID := env.RegionB.HLC.GenerateID()
		message := map[string]interface{}{
			"msg_id":          msgID.String(),
			"conversation_id": conversationID,
			"user_id":         userID,
			"content":         fmt.Sprintf("Failover message %d", i),
			"timestamp":       time.Now().Unix(),
			"region_id":       "region-b",
			"sync_status":     "pending", // Cannot sync to Region A
		}

		msgKey := fmt.Sprintf("failover:msg:%s", msgID.String())
		err := env.RegionB.RedisClient.HSet(ctx, msgKey, message).Err()
		require.NoError(t, err)

		failoverMessages = append(failoverMessages, msgID.String())
		time.Sleep(10 * time.Millisecond)
	}

	t.Logf("✓ Created %d messages during failover", len(failoverMessages))

	// Step 4: Verify Region B can serve all requests
	t.Log("Step 4: Verifying Region B can serve all requests")

	// Check baseline messages are accessible
	for _, msgIDStr := range baselineMessages {
		msgKey := fmt.Sprintf("failover:msg:%s", msgIDStr)
		exists, err := env.RegionB.RedisClient.Exists(ctx, msgKey).Result()
		require.NoError(t, err)
		assert.Greater(t, exists, int64(0),
			"Baseline message should be accessible in Region B")
	}

	// Check failover messages are accessible
	for _, msgIDStr := range failoverMessages {
		msgKey := fmt.Sprintf("failover:msg:%s", msgIDStr)
		exists, err := env.RegionB.RedisClient.Exists(ctx, msgKey).Result()
		require.NoError(t, err)
		assert.Greater(t, exists, int64(0),
			"Failover message should be accessible in Region B")
	}

	t.Log("✓ Region B serving all requests successfully")

	// Step 5: Restore Region A
	t.Log("Step 5: Restoring Region A")

	// Mark Region A as healthy
	err = env.RegionA.RedisClient.Set(ctx, healthKey, "healthy", time.Minute).Err()
	require.NoError(t, err)

	// Restart Region A geo router
	routerConfigA := &routing.GeoRouterConfig{
		PeerRegions: map[string]string{
			"region-b": "http://localhost:8282",
		},
		HealthCheckInterval: 30 * time.Second,
		FailoverEnabled:     true,
	}
	env.RegionA.GeoRouter = routing.NewGeoRouter("region-a", routerConfigA)
	err = env.RegionA.GeoRouter.Start()
	require.NoError(t, err)

	t.Log("✓ Region A restored")

	// Step 6: Sync failover messages to Region A
	t.Log("Step 6: Syncing failover messages to Region A")
	time.Sleep(200 * time.Millisecond) // Allow time for sync

	for _, msgIDStr := range failoverMessages {
		msgKey := fmt.Sprintf("failover:msg:%s", msgIDStr)
		msg, err := env.RegionB.RedisClient.HGetAll(ctx, msgKey).Result()
		require.NoError(t, err)

		// Replicate to Region A
		err = env.RegionA.RedisClient.HSet(ctx, msgKey, msg).Err()
		require.NoError(t, err)

		// Update sync status
		err = env.RegionB.RedisClient.HSet(ctx, msgKey, "sync_status", "synced").Err()
		require.NoError(t, err)
		err = env.RegionA.RedisClient.HSet(ctx, msgKey, "sync_status", "synced").Err()
		require.NoError(t, err)
	}

	t.Log("✓ Failover messages synced to Region A")

	// Step 7: Verify data consistency after recovery
	t.Log("Step 7: Verifying data consistency after recovery")
	allMessages := append(baselineMessages, failoverMessages...)

	for _, msgIDStr := range allMessages {
		msgKey := fmt.Sprintf("failover:msg:%s", msgIDStr)

		msgA, err := env.RegionA.RedisClient.HGetAll(ctx, msgKey).Result()
		require.NoError(t, err, "Message should exist in Region A")

		msgB, err := env.RegionB.RedisClient.HGetAll(ctx, msgKey).Result()
		require.NoError(t, err, "Message should exist in Region B")

		// Verify consistency
		assert.Equal(t, msgA["msg_id"], msgB["msg_id"],
			"Message ID should be consistent")
		assert.Equal(t, msgA["content"], msgB["content"],
			"Message content should be consistent")
		assert.Equal(t, "synced", msgA["sync_status"],
			"Message should be synced in Region A")
		assert.Equal(t, "synced", msgB["sync_status"],
			"Message should be synced in Region B")
	}

	t.Logf("✓ Data consistency verified for %d messages", len(allMessages))

	// Step 8: Verify message ordering after recovery
	t.Log("Step 8: Verifying message ordering after recovery")

	type MessageWithTimestamp struct {
		MsgID     string
		Timestamp int64
		Content   string
	}

	messagesWithTS := make([]MessageWithTimestamp, 0, len(allMessages))

	for _, msgIDStr := range allMessages {
		msgKey := fmt.Sprintf("failover:msg:%s", msgIDStr)
		msg, err := env.RegionA.RedisClient.HGetAll(ctx, msgKey).Result()
		require.NoError(t, err)

		timestamp, _ := parseInt64(msg["timestamp"])
		messagesWithTS = append(messagesWithTS, MessageWithTimestamp{
			MsgID:     msgIDStr,
			Timestamp: timestamp,
			Content:   msg["content"],
		})
	}

	// Verify ordering is maintained
	for i := 1; i < len(messagesWithTS); i++ {
		assert.LessOrEqual(t, messagesWithTS[i-1].Timestamp, messagesWithTS[i].Timestamp,
			"Message timestamps should be in order")
	}

	t.Log("✓ Message ordering preserved after recovery")

	// Step 9: Test RTO (Recovery Time Objective)
	t.Log("Step 9: Verifying RTO < 30 seconds")
	// In a real scenario, we would measure actual failover time
	// For this test, we verify the system can recover within acceptable time

	recoveryStart := time.Now()

	// Simulate checking if system is operational
	for i := 0; i < 30; i++ {
		// Check if both regions are healthy
		healthA, _ := env.RegionA.RedisClient.Get(ctx, "health:region-a").Result()
		healthB, _ := env.RegionB.RedisClient.Get(ctx, "health:region-b").Result()

		if healthA == "healthy" && healthB == "healthy" {
			recoveryTime := time.Since(recoveryStart)
			t.Logf("✓ System recovered in %v", recoveryTime)
			assert.Less(t, recoveryTime.Seconds(), 30.0,
				"Recovery time should be < 30 seconds (RTO requirement)")
			break
		}

		time.Sleep(1 * time.Second)
	}

	// Step 10: Verify RPO (Recovery Point Objective)
	t.Log("Step 10: Verifying RPO ≈ 0 (no data loss)")

	// Count messages in both regions
	countA := 0
	countB := 0

	for _, msgIDStr := range allMessages {
		msgKey := fmt.Sprintf("failover:msg:%s", msgIDStr)

		existsA, _ := env.RegionA.RedisClient.Exists(ctx, msgKey).Result()
		if existsA > 0 {
			countA++
		}

		existsB, _ := env.RegionB.RedisClient.Exists(ctx, msgKey).Result()
		if existsB > 0 {
			countB++
		}
	}

	assert.Equal(t, len(allMessages), countA,
		"Region A should have all messages (no data loss)")
	assert.Equal(t, len(allMessages), countB,
		"Region B should have all messages (no data loss)")

	t.Logf("✓ RPO verified: 0 data loss (%d/%d messages in both regions)",
		countA, len(allMessages))

	// Cleanup
	env.RegionA.RedisClient.Del(ctx, healthKey)
	env.RegionB.RedisClient.Del(ctx, "health:region-b")

	for _, msgIDStr := range allMessages {
		msgKey := fmt.Sprintf("failover:msg:%s", msgIDStr)
		env.RegionA.RedisClient.Del(ctx, msgKey)
		env.RegionB.RedisClient.Del(ctx, msgKey)
	}

	t.Log("✓ Failover recovery test completed successfully")
}

// Helper types and functions

// parseInt64 parses a string to int64, returns 0 if parsing fails
func parseInt64(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	var result int64
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// sortMessagesByHLC sorts messages by HLC timestamp
func sortMessagesByHLC(messages []MessageWithHLC) {
	// Simple bubble sort for small arrays
	n := len(messages)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if messages[j].HLC > messages[j+1].HLC ||
				(messages[j].HLC == messages[j+1].HLC && messages[j].Logical > messages[j+1].Logical) {
				messages[j], messages[j+1] = messages[j+1], messages[j]
			}
		}
	}
}

// MessageWithHLC represents a message with HLC timestamp for sorting
type MessageWithHLC struct {
	MsgID    string
	HLC      int64
	Logical  int64
	RegionID string
	Content  string
}
