package loadtest

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// ConnectionPool WebSocket 连接池，模拟大量并发连接
type ConnectionPool struct {
	config      *LoadTestConfig
	connections []*WSConnection
	mu          sync.RWMutex

	// 统计
	activeCount    int32
	failedCount    int32
	reconnectCount int32
}

// WSConnection WebSocket 连接封装
type WSConnection struct {
	ID       string
	Region   string
	Endpoint string
	Conn     *websocket.Conn

	// 消息统计
	sentCount     int64
	receivedCount int64
	failedCount   int64

	// 延迟测量
	latencies   []time.Duration
	latenciesMu sync.Mutex

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	closed atomic.Bool
}

// NewConnectionPool 创建连接池
func NewConnectionPool(config *LoadTestConfig) *ConnectionPool {
	return &ConnectionPool{
		config:      config,
		connections: make([]*WSConnection, 0, config.TotalConnections),
	}
}

// Connect 建立所有连接 (支持预热)
func (cp *ConnectionPool) Connect(ctx context.Context) error {
	regionACount := cp.config.TotalConnections * cp.config.RegionAPercent / 100
	regionBCount := cp.config.TotalConnections - regionACount

	// 计算预热间隔
	rampUpInterval := time.Duration(0)
	if cp.config.RampUpTime > 0 {
		rampUpInterval = cp.config.RampUpTime / time.Duration(cp.config.TotalConnections)
	}

	// 建立 Region A 连接
	for i := 0; i < regionACount; i++ {
		if err := cp.connectOne(ctx, "region-a", cp.config.RegionAEndpoint); err != nil {
			return fmt.Errorf("failed to connect to region-a: %w", err)
		}
		if rampUpInterval > 0 {
			time.Sleep(rampUpInterval)
		}
	}

	// 建立 Region B 连接
	for i := 0; i < regionBCount; i++ {
		if err := cp.connectOne(ctx, "region-b", cp.config.RegionBEndpoint); err != nil {
			return fmt.Errorf("failed to connect to region-b: %w", err)
		}
		if rampUpInterval > 0 {
			time.Sleep(rampUpInterval)
		}
	}

	return nil
}

// connectOne 建立单个连接
func (cp *ConnectionPool) connectOne(ctx context.Context, region, endpoint string) error {
	// 构造 WebSocket URL
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}

	// 添加认证令牌
	header := http.Header{}
	if cp.config.AuthToken != "" {
		header.Add("Authorization", "Bearer "+cp.config.AuthToken)
	}

	// 建立 WebSocket 连接
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, u.String(), header)
	if err != nil {
		atomic.AddInt32(&cp.failedCount, 1)
		return fmt.Errorf("failed to dial: %w", err)
	}

	// 创建连接对象
	connCtx, cancel := context.WithCancel(ctx)
	wsConn := &WSConnection{
		ID:        fmt.Sprintf("%s-%d", region, len(cp.connections)),
		Region:    region,
		Endpoint:  endpoint,
		Conn:      conn,
		ctx:       connCtx,
		cancel:    cancel,
		latencies: make([]time.Duration, 0, 1000),
	}

	cp.mu.Lock()
	cp.connections = append(cp.connections, wsConn)
	cp.mu.Unlock()

	atomic.AddInt32(&cp.activeCount, 1)

	// 启动接收协程
	go wsConn.receiveLoop()

	return nil
}

// SendMessage 发送消息并测量延迟
func (wsc *WSConnection) SendMessage(msg []byte) (time.Duration, error) {
	if wsc.closed.Load() {
		return 0, fmt.Errorf("connection closed")
	}

	start := time.Now()

	err := wsc.Conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		atomic.AddInt64(&wsc.failedCount, 1)
		return 0, err
	}

	latency := time.Since(start)
	atomic.AddInt64(&wsc.sentCount, 1)

	// 记录延迟
	wsc.latenciesMu.Lock()
	wsc.latencies = append(wsc.latencies, latency)
	wsc.latenciesMu.Unlock()

	return latency, nil
}

// receiveLoop 接收消息循环
func (wsc *WSConnection) receiveLoop() {
	defer wsc.Close()

	for {
		select {
		case <-wsc.ctx.Done():
			return
		default:
			_, _, err := wsc.Conn.ReadMessage()
			if err != nil {
				if !wsc.closed.Load() {
					atomic.AddInt64(&wsc.failedCount, 1)
				}
				return
			}
			atomic.AddInt64(&wsc.receivedCount, 1)
		}
	}
}

// Close 关闭连接
func (wsc *WSConnection) Close() error {
	if wsc.closed.Swap(true) {
		return nil // 已关闭
	}

	wsc.cancel()
	return wsc.Conn.Close()
}

// GetStats 获取连接统计
func (wsc *WSConnection) GetStats() MessageStats {
	wsc.latenciesMu.Lock()
	latencies := make([]time.Duration, len(wsc.latencies))
	copy(latencies, wsc.latencies)
	wsc.latenciesMu.Unlock()

	return MessageStats{
		Sent:      atomic.LoadInt64(&wsc.sentCount),
		Received:  atomic.LoadInt64(&wsc.receivedCount),
		Failed:    atomic.LoadInt64(&wsc.failedCount),
		Latencies: latencies,
	}
}

// CloseAll 关闭所有连接
func (cp *ConnectionPool) CloseAll() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	var firstErr error
	for _, conn := range cp.connections {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// GetConnectionStats 获取连接池统计
func (cp *ConnectionPool) GetConnectionStats() ConnectionStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	stats := ConnectionStats{
		Active:     int(atomic.LoadInt32(&cp.activeCount)),
		Failed:     int(atomic.LoadInt32(&cp.failedCount)),
		Reconnects: int(atomic.LoadInt32(&cp.reconnectCount)),
	}

	for _, conn := range cp.connections {
		if conn.Region == "region-a" {
			stats.RegionA++
		} else {
			stats.RegionB++
		}
	}

	return stats
}

// GetConnections 获取所有连接
func (cp *ConnectionPool) GetConnections() []*WSConnection {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	conns := make([]*WSConnection, len(cp.connections))
	copy(conns, cp.connections)
	return conns
}
