package capacity

import (
	"context"
	"sort"
	"sync"
	"time"
)

// InMemoryHistoryStore 内存历史数据存储（用于测试和开发）
type InMemoryHistoryStore struct {
	mu      sync.RWMutex
	data    map[string][]ResourceUsage // key: resourceType:resourceName
	maxSize int
}

// NewInMemoryHistoryStore 创建内存历史存储
func NewInMemoryHistoryStore(maxSize int) *InMemoryHistoryStore {
	return &InMemoryHistoryStore{
		data:    make(map[string][]ResourceUsage),
		maxSize: maxSize,
	}
}

// Store 存储资源使用记录
func (s *InMemoryHistoryStore) Store(ctx context.Context, usage ResourceUsage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := string(usage.ResourceType) + ":" + usage.ResourceName
	history := s.data[key]

	// 添加新记录
	history = append(history, usage)

	// 按时间排序
	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp.Before(history[j].Timestamp)
	})

	// 限制大小
	if len(history) > s.maxSize {
		history = history[len(history)-s.maxSize:]
	}

	s.data[key] = history
	return nil
}

// Query 查询历史数据
func (s *InMemoryHistoryStore) Query(ctx context.Context, resourceType ResourceType, name string, since time.Time) ([]ResourceUsage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := string(resourceType) + ":" + name
	history := s.data[key]

	var result []ResourceUsage
	for _, usage := range history {
		if usage.Timestamp.After(since) || usage.Timestamp.Equal(since) {
			result = append(result, usage)
		}
	}

	return result, nil
}

// Clear 清空所有历史数据
func (s *InMemoryHistoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string][]ResourceUsage)
}

// GetAll 获取所有历史数据（用于测试）
func (s *InMemoryHistoryStore) GetAll() map[string][]ResourceUsage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string][]ResourceUsage)
	for k, v := range s.data {
		result[k] = append([]ResourceUsage{}, v...)
	}
	return result
}
