package connpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

// KafkaProducerPool manages a Kafka producer connection pool
type KafkaProducerPool struct {
	Name     string
	Producer sarama.SyncProducer
	config   KafkaProducerConfig

	// Metrics
	mu             sync.RWMutex
	messagesSent   int64
	messagesFailed int64
	bytesWritten   int64
	totalLatency   time.Duration
	lastError      error
	lastErrorTime  time.Time
}

// NewKafkaProducerPool creates a new Kafka producer pool
func NewKafkaProducerPool(name string, brokers []string, config KafkaProducerConfig) (*KafkaProducerPool, error) {
	// Configure Sarama
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V3_0_0_0

	// Producer configuration
	saramaConfig.Producer.MaxMessageBytes = config.MaxMessageBytes
	saramaConfig.Producer.RequiredAcks = config.RequiredAcks
	saramaConfig.Producer.Timeout = config.Timeout
	saramaConfig.Producer.Compression = config.Compression
	saramaConfig.Producer.Idempotent = config.Idempotent
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true

	// Retry configuration
	saramaConfig.Producer.Retry.Max = config.RetryMax
	saramaConfig.Producer.Retry.Backoff = config.RetryBackoff

	// Network configuration
	saramaConfig.Net.MaxOpenRequests = config.MaxOpenRequests
	saramaConfig.Net.DialTimeout = 10 * time.Second
	saramaConfig.Net.ReadTimeout = 10 * time.Second
	saramaConfig.Net.WriteTimeout = 10 * time.Second

	// Metadata configuration for cross-region
	saramaConfig.Metadata.Retry.Max = 3
	saramaConfig.Metadata.Retry.Backoff = 250 * time.Millisecond
	saramaConfig.Metadata.RefreshFrequency = 10 * time.Minute

	// Create producer
	producer, err := sarama.NewSyncProducer(brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	pool := &KafkaProducerPool{
		Name:     name,
		Producer: producer,
		config:   config,
	}

	return pool, nil
}

// Close closes the Kafka producer pool
func (p *KafkaProducerPool) Close() error {
	return p.Producer.Close()
}

// SendMessage sends a message and records metrics
func (p *KafkaProducerPool) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	start := time.Now()

	partition, offset, err = p.Producer.SendMessage(msg)

	latency := time.Since(start)

	p.mu.Lock()
	if err != nil {
		p.messagesFailed++
		p.lastError = err
		p.lastErrorTime = time.Now()
	} else {
		p.messagesSent++
		if msg.Value != nil {
			p.bytesWritten += int64(msg.Value.Length())
		}
	}
	p.totalLatency += latency
	p.mu.Unlock()

	return partition, offset, err
}

// GetMetrics returns producer metrics
func (p *KafkaProducerPool) GetMetrics() KafkaProducerMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	avgLatency := time.Duration(0)
	totalMessages := p.messagesSent + p.messagesFailed
	if totalMessages > 0 {
		avgLatency = p.totalLatency / time.Duration(totalMessages)
	}

	successRate := 0.0
	if totalMessages > 0 {
		successRate = float64(p.messagesSent) / float64(totalMessages)
	}

	return KafkaProducerMetrics{
		Name:           p.Name,
		MessagesSent:   p.messagesSent,
		MessagesFailed: p.messagesFailed,
		BytesWritten:   p.bytesWritten,
		AvgLatency:     avgLatency,
		SuccessRate:    successRate,
		LastError:      p.lastError,
		LastErrorTime:  p.lastErrorTime,
	}
}

// KafkaProducerMetrics holds Kafka producer metrics
type KafkaProducerMetrics struct {
	Name           string
	MessagesSent   int64
	MessagesFailed int64
	BytesWritten   int64
	AvgLatency     time.Duration
	SuccessRate    float64
	LastError      error
	LastErrorTime  time.Time
}

// IsHealthy checks if the producer is healthy
func (p *KafkaProducerPool) IsHealthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check if we have recent errors
	if p.lastError != nil && time.Since(p.lastErrorTime) < 1*time.Minute {
		return false
	}

	// Check success rate
	totalMessages := p.messagesSent + p.messagesFailed
	if totalMessages > 100 {
		successRate := float64(p.messagesSent) / float64(totalMessages)
		if successRate < 0.95 {
			return false
		}
	}

	return true
}

// KafkaConsumerPool manages a Kafka consumer connection pool
type KafkaConsumerPool struct {
	Name     string
	Consumer sarama.ConsumerGroup
	config   KafkaConsumerConfig

	// Metrics
	mu                sync.RWMutex
	messagesConsumed  int64
	messagesProcessed int64
	messagesFailed    int64
	bytesRead         int64
	totalLatency      time.Duration
	lastError         error
	lastErrorTime     time.Time
}

// NewKafkaConsumerPool creates a new Kafka consumer pool
func NewKafkaConsumerPool(name string, brokers []string, groupID string, config KafkaConsumerConfig) (*KafkaConsumerPool, error) {
	// Configure Sarama
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V3_0_0_0

	// Consumer configuration
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = false // Manual commit
	saramaConfig.Consumer.Return.Errors = true

	// Session configuration
	saramaConfig.Consumer.Group.Session.Timeout = config.SessionTimeout
	saramaConfig.Consumer.Group.Heartbeat.Interval = config.HeartbeatInterval
	saramaConfig.Consumer.Group.Rebalance.Timeout = config.RebalanceTimeout
	saramaConfig.Consumer.MaxProcessingTime = config.MaxProcessingTime

	// Fetch configuration
	saramaConfig.Consumer.Fetch.Min = config.FetchMin
	saramaConfig.Consumer.Fetch.Default = config.FetchDefault
	saramaConfig.Consumer.MaxWaitTime = config.MaxWaitTime

	// Network configuration for cross-region
	saramaConfig.Net.DialTimeout = 10 * time.Second
	saramaConfig.Net.ReadTimeout = 10 * time.Second
	saramaConfig.Net.WriteTimeout = 10 * time.Second

	// Create consumer group
	consumer, err := sarama.NewConsumerGroup(brokers, groupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	pool := &KafkaConsumerPool{
		Name:     name,
		Consumer: consumer,
		config:   config,
	}

	return pool, nil
}

// Close closes the Kafka consumer pool
func (p *KafkaConsumerPool) Close() error {
	return p.Consumer.Close()
}

// RecordMessageConsumed records a consumed message
func (p *KafkaConsumerPool) RecordMessageConsumed(size int64) {
	p.mu.Lock()
	p.messagesConsumed++
	p.bytesRead += size
	p.mu.Unlock()
}

// RecordMessageProcessed records a successfully processed message
func (p *KafkaConsumerPool) RecordMessageProcessed(latency time.Duration) {
	p.mu.Lock()
	p.messagesProcessed++
	p.totalLatency += latency
	p.mu.Unlock()
}

// RecordMessageFailed records a failed message
func (p *KafkaConsumerPool) RecordMessageFailed(err error) {
	p.mu.Lock()
	p.messagesFailed++
	p.lastError = err
	p.lastErrorTime = time.Now()
	p.mu.Unlock()
}

// GetMetrics returns consumer metrics
func (p *KafkaConsumerPool) GetMetrics() KafkaConsumerMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	avgLatency := time.Duration(0)
	if p.messagesProcessed > 0 {
		avgLatency = p.totalLatency / time.Duration(p.messagesProcessed)
	}

	successRate := 0.0
	totalProcessed := p.messagesProcessed + p.messagesFailed
	if totalProcessed > 0 {
		successRate = float64(p.messagesProcessed) / float64(totalProcessed)
	}

	return KafkaConsumerMetrics{
		Name:              p.Name,
		MessagesConsumed:  p.messagesConsumed,
		MessagesProcessed: p.messagesProcessed,
		MessagesFailed:    p.messagesFailed,
		BytesRead:         p.bytesRead,
		AvgLatency:        avgLatency,
		SuccessRate:       successRate,
		LastError:         p.lastError,
		LastErrorTime:     p.lastErrorTime,
	}
}

// KafkaConsumerMetrics holds Kafka consumer metrics
type KafkaConsumerMetrics struct {
	Name              string
	MessagesConsumed  int64
	MessagesProcessed int64
	MessagesFailed    int64
	BytesRead         int64
	AvgLatency        time.Duration
	SuccessRate       float64
	LastError         error
	LastErrorTime     time.Time
}

// IsHealthy checks if the consumer is healthy
func (p *KafkaConsumerPool) IsHealthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check if we have recent errors
	if p.lastError != nil && time.Since(p.lastErrorTime) < 1*time.Minute {
		return false
	}

	// Check success rate
	totalProcessed := p.messagesProcessed + p.messagesFailed
	if totalProcessed > 100 {
		successRate := float64(p.messagesProcessed) / float64(totalProcessed)
		if successRate < 0.95 {
			return false
		}
	}

	return true
}
