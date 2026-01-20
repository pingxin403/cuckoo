package analytics

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/pingxin403/cuckoo/apps/shortener-service/logger"
	"github.com/pingxin403/cuckoo/apps/shortener-service/metrics"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// ClickEvent represents a click event for analytics
// Requirements: 7.3
type ClickEvent struct {
	ShortCode string    `json:"short_code"`
	Timestamp time.Time `json:"timestamp"`
	SourceIP  string    `json:"source_ip"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer,omitempty"`
}

// AnalyticsWriter handles async click event logging to Kafka
// Requirements: 7.1, 7.2, 7.5
type AnalyticsWriter struct {
	writer       *kafka.Writer
	eventChannel chan ClickEvent
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	numWorkers   int
	closed       bool
	closeMu      sync.Mutex
}

// Config holds configuration for AnalyticsWriter
type Config struct {
	KafkaBrokers []string
	Topic        string
	NumWorkers   int
	BufferSize   int
}

// NewAnalyticsWriter creates a new AnalyticsWriter
// Requirements: 7.1, 7.2, 7.5
func NewAnalyticsWriter(config Config) *AnalyticsWriter {
	// Set defaults
	if config.NumWorkers == 0 {
		config.NumWorkers = 4
	}
	if config.BufferSize == 0 {
		config.BufferSize = 10000
	}
	if config.Topic == "" {
		config.Topic = "url-clicks"
	}

	// Create Kafka writer
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.KafkaBrokers...),
		Topic:        config.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Async:        true, // Non-blocking writes
		RequiredAcks: kafka.RequireOne,
		Compression:  kafka.Snappy,
	}

	ctx, cancel := context.WithCancel(context.Background())

	aw := &AnalyticsWriter{
		writer:       writer,
		eventChannel: make(chan ClickEvent, config.BufferSize),
		ctx:          ctx,
		cancel:       cancel,
		numWorkers:   config.NumWorkers,
	}

	// Start worker goroutines
	// Requirements: 7.2 - Background workers for async processing
	for i := 0; i < config.NumWorkers; i++ {
		aw.wg.Add(1)
		go aw.worker(i)
	}

	// Safe logging (logger may not be initialized in tests)
	if logger.Log != nil {
		logger.Log.Info("Analytics writer initialized",
			zap.Strings("brokers", config.KafkaBrokers),
			zap.String("topic", config.Topic),
			zap.Int("workers", config.NumWorkers),
			zap.Int("buffer_size", config.BufferSize),
		)
	}

	return aw
}

// LogClick logs a click event asynchronously
// Requirements: 7.1, 7.2
func (aw *AnalyticsWriter) LogClick(event ClickEvent) {
	select {
	case aw.eventChannel <- event:
		// Event queued successfully
	default:
		// Buffer full, drop event and increment metric
		// Requirements: 7.5 - Handle Kafka failures gracefully
		metrics.ErrorsTotal.WithLabelValues("analytics_buffer_full").Inc()
		if logger.Log != nil {
			logger.Log.Warn("Analytics buffer full, dropping event",
				zap.String("short_code", event.ShortCode),
			)
		}
	}
}

// worker processes events from the channel and writes to Kafka
// Requirements: 7.2, 7.5
func (aw *AnalyticsWriter) worker(id int) {
	defer aw.wg.Done()

	if logger.Log != nil {
		logger.Log.Info("Analytics worker started", zap.Int("worker_id", id))
	}

	for {
		select {
		case event := <-aw.eventChannel:
			// Marshal event to JSON
			data, err := json.Marshal(event)
			if err != nil {
				metrics.ErrorsTotal.WithLabelValues("analytics_marshal_error").Inc()
				if logger.Log != nil {
					logger.Log.Error("Failed to marshal click event",
						zap.Error(err),
						zap.String("short_code", event.ShortCode),
					)
				}
				continue
			}

			// Write to Kafka
			// Requirements: 7.5 - Handle Kafka failures gracefully
			msg := kafka.Message{
				Key:   []byte(event.ShortCode),
				Value: data,
				Time:  event.Timestamp,
			}

			// Use context with timeout for write
			writeCtx, cancel := context.WithTimeout(aw.ctx, 5*time.Second)
			err = aw.writer.WriteMessages(writeCtx, msg)
			cancel()

			if err != nil {
				// Requirements: 7.5 - Drop events on Kafka failure, increment metric
				metrics.ErrorsTotal.WithLabelValues("analytics_kafka_write_error").Inc()
				if logger.Log != nil {
					logger.Log.Warn("Failed to write click event to Kafka",
						zap.Error(err),
						zap.String("short_code", event.ShortCode),
					)
				}
			} else {
				metrics.ClickEventsLogged.Inc()
			}

		case <-aw.ctx.Done():
			if logger.Log != nil {
				logger.Log.Info("Analytics worker stopping", zap.Int("worker_id", id))
			}
			return
		}
	}
}

// Close gracefully shuts down the analytics writer
func (aw *AnalyticsWriter) Close() error {
	aw.closeMu.Lock()
	defer aw.closeMu.Unlock()

	// Prevent double close
	if aw.closed {
		return nil
	}
	aw.closed = true

	if logger.Log != nil {
		logger.Log.Info("Shutting down analytics writer")
	}

	// Cancel context to stop workers
	aw.cancel()

	// Wait for workers to finish processing
	aw.wg.Wait()

	// Close event channel
	close(aw.eventChannel)

	// Close Kafka writer
	if err := aw.writer.Close(); err != nil {
		if logger.Log != nil {
			logger.Log.Error("Error closing Kafka writer", zap.Error(err))
		}
		return err
	}

	if logger.Log != nil {
		logger.Log.Info("Analytics writer shut down complete")
	}
	return nil
}

// Stats returns current statistics about the analytics writer
func (aw *AnalyticsWriter) Stats() map[string]interface{} {
	return map[string]interface{}{
		"buffer_size":        cap(aw.eventChannel),
		"buffer_used":        len(aw.eventChannel),
		"num_workers":        aw.numWorkers,
		"buffer_utilization": float64(len(aw.eventChannel)) / float64(cap(aw.eventChannel)),
	}
}
