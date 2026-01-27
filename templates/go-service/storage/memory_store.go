package storage

import (
	"fmt"
	"sync"

	"github.com/pingxin403/cuckoo/api/gen/go/{{PROTO_PACKAGE}}"
)

// Store 定义存储接口
// TODO: 根据服务需求更新此接口的方法
type Store interface {
	Create(item *{{PROTO_PACKAGE}}.YourItem) error
	Get(id string) (*{{PROTO_PACKAGE}}.YourItem, error)
	List() ([]*{{PROTO_PACKAGE}}.YourItem, error)
	Update(item *{{PROTO_PACKAGE}}.YourItem) error
	Delete(id string) error
}

// MemoryStore 内存存储实现
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]*{{PROTO_PACKAGE}}.YourItem
}

// NewMemoryStore 创建内存存储实例
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items: make(map[string]*{{PROTO_PACKAGE}}.YourItem),
	}
}

// 确保 MemoryStore 实现 Store 接口
var _ Store = (*MemoryStore)(nil)

// Create 添加新项目到存储
func (s *MemoryStore) Create(item *{{PROTO_PACKAGE}}.YourItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	s.items[item.Id] = item
	return nil
}

// Get 根据 ID 获取项目
func (s *MemoryStore) Get(id string) (*{{PROTO_PACKAGE}}.YourItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[id]
	if !exists {
		return nil, fmt.Errorf("item with id %s not found", id)
	}

	return item, nil
}

// List 返回所有项目
func (s *MemoryStore) List() ([]*{{PROTO_PACKAGE}}.YourItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*{{PROTO_PACKAGE}}.YourItem, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}

	return items, nil
}

// Update 更新现有项目
func (s *MemoryStore) Update(item *{{PROTO_PACKAGE}}.YourItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	if _, exists := s.items[item.Id]; !exists {
		return fmt.Errorf("item with id %s not found", item.Id)
	}

	s.items[item.Id] = item
	return nil
}

// Delete 根据 ID 删除项目
func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[id]; !exists {
		return fmt.Errorf("item with id %s not found", id)
	}

	delete(s.items, id)
	return nil
}
