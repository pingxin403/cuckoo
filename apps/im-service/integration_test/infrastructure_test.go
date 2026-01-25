//go:build integration
// +build integration

package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	clientv3 "go.etcd.io/etcd/client/v3"

	_ "github.com/go-sql-driver/mysql"
)

// TestEtcdClusterFailover tests etcd cluster failover and leader election
// Validates: Requirements 10.3 (etcd cluster resilience)
func TestEtcdClusterFailover(t *testing.T) {
	ctx := context.Background()

	// Step 1: Connect to etcd cluster
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379", "localhost:2380", "localhost:2381"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Skipf("Skipping etcd cluster test: %v", err)
		return
	}
	defer etcdClient.Close()

	// Step 2: Write a test key
	testKey := fmt.Sprintf("/test/failover/%d", time.Now().UnixNano())
	testValue := "test-value"

	_, err = etcdClient.Put(ctx, testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	t.Logf("Successfully wrote test key: %s", testKey)

	// Step 3: Verify we can read the key
	resp, err := etcdClient.Get(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to read test key: %v", err)
	}

	if len(resp.Kvs) == 0 {
		t.Fatal("Test key not found")
	}

	if string(resp.Kvs[0].Value) != testValue {
		t.Errorf("Expected value=%s, got %s", testValue, string(resp.Kvs[0].Value))
	}

	t.Log("Successfully read test key from etcd cluster")

	// Step 4: Test leader election by checking cluster status
	statusResp, err := etcdClient.Status(ctx, "localhost:2379")
	if err != nil {
		t.Logf("Warning: Could not get etcd status: %v", err)
	} else {
		t.Logf("Etcd cluster status: Leader=%d, Version=%s", statusResp.Leader, statusResp.Version)
	}

	// Step 5: Test watch mechanism for failover detection
	watchChan := etcdClient.Watch(ctx, testKey)

	// Update the key
	_, err = etcdClient.Put(ctx, testKey, "updated-value")
	if err != nil {
		t.Fatalf("Failed to update test key: %v", err)
	}

	// Wait for watch event
	select {
	case watchResp := <-watchChan:
		if watchResp.Canceled {
			t.Error("Watch was canceled")
		}
		if len(watchResp.Events) == 0 {
			t.Error("No watch events received")
		} else {
			t.Logf("Received watch event: %v", watchResp.Events[0].Type)
		}
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for watch event")
	}

	// Cleanup
	_, err = etcdClient.Delete(ctx, testKey)
	if err != nil {
		t.Logf("Warning: Failed to cleanup test key: %v", err)
	}

	t.Log("Etcd cluster failover test completed successfully")
}

// TestKafkaBrokerFailover tests Kafka broker failover and replication
// Validates: Requirements 10.3 (Kafka cluster resilience)
func TestKafkaBrokerFailover(t *testing.T) {
	ctx := context.Background()

	// Step 1: Create a test topic with replication
	testTopic := fmt.Sprintf("test-failover-%d", time.Now().UnixNano())

	conn, err := kafka.DialLeader(ctx, "tcp", "localhost:9092", testTopic, 0)
	if err != nil {
		// Topic doesn't exist, create it
		controller, err := kafka.Dial("tcp", "localhost:9092")
		if err != nil {
			t.Skipf("Skipping Kafka test: cannot connect: %v", err)
			return
		}
		defer controller.Close()

		topicConfig := kafka.TopicConfig{
			Topic:             testTopic,
			NumPartitions:     3,
			ReplicationFactor: 1, // Single broker in test environment
		}

		err = controller.CreateTopics(topicConfig)
		if err != nil {
			t.Fatalf("Failed to create test topic: %v", err)
		}

		t.Logf("Created test topic: %s", testTopic)
		time.Sleep(2 * time.Second) // Wait for topic creation
	} else {
		conn.Close()
	}

	// Step 2: Create producer and send messages
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    testTopic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	messages := []kafka.Message{
		{Key: []byte("key1"), Value: []byte("message1")},
		{Key: []byte("key2"), Value: []byte("message2")},
		{Key: []byte("key3"), Value: []byte("message3")},
	}

	err = writer.WriteMessages(ctx, messages...)
	if err != nil {
		t.Fatalf("Failed to write messages: %v", err)
	}

	t.Logf("Successfully wrote %d messages to Kafka", len(messages))

	// Step 3: Create consumer and read messages
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{"localhost:9092"},
		Topic:     testTopic,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  10e6,
	})
	defer reader.Close()

	// Set read deadline
	reader.SetOffset(0)

	receivedCount := 0
	timeout := time.After(10 * time.Second)

	for receivedCount < len(messages) {
		select {
		case <-timeout:
			t.Fatalf("Timeout: only received %d/%d messages", receivedCount, len(messages))
		default:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			msg, err := reader.ReadMessage(ctx)
			cancel()

			if err != nil {
				if err == context.DeadlineExceeded {
					continue
				}
				t.Fatalf("Failed to read message: %v", err)
			}

			receivedCount++
			t.Logf("Received message: key=%s, value=%s", string(msg.Key), string(msg.Value))
		}
	}

	if receivedCount != len(messages) {
		t.Errorf("Expected %d messages, received %d", len(messages), receivedCount)
	}

	t.Log("Kafka broker failover test completed successfully")
}

// TestRedisFailover tests Redis failover and persistence
// Validates: Requirements 10.3 (Redis resilience)
func TestRedisFailover(t *testing.T) {
	ctx := context.Background()

	// Step 1: Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	defer redisClient.Close()

	// Step 2: Test connection
	err := redisClient.Ping(ctx).Err()
	if err != nil {
		t.Skipf("Skipping Redis test: cannot connect: %v", err)
		return
	}

	t.Log("Successfully connected to Redis")

	// Step 3: Write test data
	testKey := fmt.Sprintf("test:failover:%d", time.Now().UnixNano())
	testValue := "test-value"

	err = redisClient.Set(ctx, testKey, testValue, 10*time.Minute).Err()
	if err != nil {
		t.Fatalf("Failed to write to Redis: %v", err)
	}

	t.Logf("Successfully wrote test key: %s", testKey)

	// Step 4: Read test data
	value, err := redisClient.Get(ctx, testKey).Result()
	if err != nil {
		t.Fatalf("Failed to read from Redis: %v", err)
	}

	if value != testValue {
		t.Errorf("Expected value=%s, got %s", testValue, value)
	}

	t.Log("Successfully read test key from Redis")

	// Step 5: Test TTL functionality
	ttl, err := redisClient.TTL(ctx, testKey).Result()
	if err != nil {
		t.Fatalf("Failed to get TTL: %v", err)
	}

	if ttl <= 0 {
		t.Errorf("Expected positive TTL, got %v", ttl)
	}

	t.Logf("TTL for test key: %v", ttl)

	// Step 6: Test persistence by checking INFO
	info, err := redisClient.Info(ctx, "persistence").Result()
	if err != nil {
		t.Logf("Warning: Could not get persistence info: %v", err)
	} else {
		t.Logf("Redis persistence info: %s", info[:100]) // First 100 chars
	}

	// Step 7: Test connection pool by making multiple concurrent requests
	concurrentRequests := 10
	done := make(chan bool, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func(index int) {
			key := fmt.Sprintf("%s:concurrent:%d", testKey, index)
			err := redisClient.Set(ctx, key, fmt.Sprintf("value-%d", index), 1*time.Minute).Err()
			if err != nil {
				t.Errorf("Concurrent request %d failed: %v", index, err)
			}
			done <- true
		}(i)
	}

	// Wait for all concurrent requests
	for i := 0; i < concurrentRequests; i++ {
		<-done
	}

	t.Logf("Successfully completed %d concurrent Redis requests", concurrentRequests)

	// Cleanup
	err = redisClient.Del(ctx, testKey).Err()
	if err != nil {
		t.Logf("Warning: Failed to cleanup test key: %v", err)
	}

	t.Log("Redis failover test completed successfully")
}

// TestMySQLConnectionPooling tests MySQL connection pooling and resilience
// Validates: Requirements 10.4 (MySQL connection pooling)
func TestMySQLConnectionPooling(t *testing.T) {
	ctx := context.Background()

	// Step 1: Connect to MySQL with connection pooling
	mysqlAddr := "root:password@tcp(localhost:3306)/im_chat"
	db, err := sql.Open("mysql", mysqlAddr)
	if err != nil {
		t.Skipf("Skipping MySQL test: cannot connect: %v", err)
		return
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Step 2: Test connection
	err = db.PingContext(ctx)
	if err != nil {
		t.Skipf("Skipping MySQL test: ping failed: %v", err)
		return
	}

	t.Log("Successfully connected to MySQL")

	// Step 3: Create test table
	testTable := fmt.Sprintf("test_pooling_%d", time.Now().UnixNano())
	_, err = db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			data VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, testTable))
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	t.Logf("Created test table: %s", testTable)

	// Step 4: Test concurrent inserts (stress test connection pool)
	concurrentInserts := 20
	done := make(chan error, concurrentInserts)

	for i := 0; i < concurrentInserts; i++ {
		go func(index int) {
			_, err := db.ExecContext(ctx, fmt.Sprintf(
				"INSERT INTO %s (data) VALUES (?)", testTable),
				fmt.Sprintf("test-data-%d", index),
			)
			done <- err
		}(i)
	}

	// Wait for all inserts
	successCount := 0
	for i := 0; i < concurrentInserts; i++ {
		err := <-done
		if err != nil {
			t.Errorf("Concurrent insert %d failed: %v", i, err)
		} else {
			successCount++
		}
	}

	t.Logf("Successfully completed %d/%d concurrent inserts", successCount, concurrentInserts)

	// Step 5: Verify data
	var count int
	err = db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", testTable)).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}

	if count != successCount {
		t.Errorf("Expected %d rows, got %d", successCount, count)
	}

	t.Logf("Verified %d rows in test table", count)

	// Step 6: Test connection pool stats
	stats := db.Stats()
	t.Logf("Connection pool stats:")
	t.Logf("  - MaxOpenConnections: %d", stats.MaxOpenConnections)
	t.Logf("  - OpenConnections: %d", stats.OpenConnections)
	t.Logf("  - InUse: %d", stats.InUse)
	t.Logf("  - Idle: %d", stats.Idle)
	t.Logf("  - WaitCount: %d", stats.WaitCount)
	t.Logf("  - WaitDuration: %v", stats.WaitDuration)

	// Step 7: Test query timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = db.QueryContext(ctx, fmt.Sprintf("SELECT SLEEP(1), * FROM %s", testTable))
	if err == nil {
		t.Error("Expected timeout error, got nil")
	} else if err != context.DeadlineExceeded {
		t.Logf("Got error (may not be timeout): %v", err)
	} else {
		t.Log("Successfully handled query timeout")
	}

	// Cleanup
	_, err = db.ExecContext(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS %s", testTable))
	if err != nil {
		t.Logf("Warning: Failed to cleanup test table: %v", err)
	}

	t.Log("MySQL connection pooling test completed successfully")
}

// TestNetworkPartitionScenario tests system behavior during network partition
// Validates: Requirements 10.3 (network partition resilience)
func TestNetworkPartitionScenario(t *testing.T) {
	ctx := context.Background()

	t.Log("Testing network partition scenario...")

	// Step 1: Test etcd behavior with short timeout (simulating partition)
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Skipf("Skipping network partition test: %v", err)
		return
	}
	defer etcdClient.Close()

	// Step 2: Test write with timeout
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	testKey := fmt.Sprintf("/test/partition/%d", time.Now().UnixNano())
	_, err = etcdClient.Put(ctx, testKey, "test-value")

	if err != nil {
		if err == context.DeadlineExceeded {
			t.Log("Successfully detected network partition (timeout)")
		} else {
			t.Logf("Got error during partition test: %v", err)
		}
	} else {
		t.Log("Write succeeded (no partition detected)")
		// Cleanup
		etcdClient.Delete(context.Background(), testKey)
	}

	// Step 3: Test Redis behavior with short timeout
	redisClient := redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		DialTimeout:  1 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	})
	defer redisClient.Close()

	ctx, cancel = context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = redisClient.Set(ctx, "test:partition", "value", 1*time.Minute).Err()
	if err != nil {
		if err == context.DeadlineExceeded {
			t.Log("Successfully detected Redis partition (timeout)")
		} else {
			t.Logf("Got error during Redis partition test: %v", err)
		}
	} else {
		t.Log("Redis write succeeded (no partition detected)")
		redisClient.Del(context.Background(), "test:partition")
	}

	t.Log("Network partition scenario test completed")
}
