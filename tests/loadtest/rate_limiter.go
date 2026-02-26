package loadtest

import (
	"context"
	"time"
)

// RateLimiter 消息发送速率控制器
type RateLimiter struct {
	rate     int           // messages per second
	interval time.Duration // interval between messages
	ticker   *time.Ticker
}

// NewRateLimiter 创建速率控制器
func NewRateLimiter(messagesPerSecond int) *RateLimiter {
	if messagesPerSecond <= 0 {
		messagesPerSecond = 1
	}

	interval := time.Second / time.Duration(messagesPerSecond)

	return &RateLimiter{
		rate:     messagesPerSecond,
		interval: interval,
		ticker:   time.NewTicker(interval),
	}
}

// Wait 等待直到可以发送下一条消息
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-rl.ticker.C:
		return nil
	}
}

// Stop 停止速率控制器
func (rl *RateLimiter) Stop() {
	rl.ticker.Stop()
}

// GetRate 获取当前速率
func (rl *RateLimiter) GetRate() int {
	return rl.rate
}
