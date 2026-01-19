package storage

import (
	"fmt"
	"sync"

	"github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb"
)

// YourStore defines the interface for storage operations
// TODO: Update this interface to match your data model
type YourStore interface {
	Create(item *shortener_servicepb.YourItem) error
	Get(id string) (*shortener_servicepb.YourItem, error)
	List() ([]*shortener_servicepb.YourItem, error)
	Update(item *shortener_servicepb.YourItem) error
	Delete(id string) error
}

// MemoryStore implements YourStore using an in-memory map
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]*shortener_servicepb.YourItem
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items: make(map[string]*shortener_servicepb.YourItem),
	}
}

// Create adds a new item to the store
func (s *MemoryStore) Create(item *shortener_servicepb.YourItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	s.items[item.Id] = item
	return nil
}

// Get retrieves an item by ID
func (s *MemoryStore) Get(id string) (*shortener_servicepb.YourItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[id]
	if !exists {
		return nil, fmt.Errorf("item with id %s not found", id)
	}

	return item, nil
}

// List returns all items
func (s *MemoryStore) List() ([]*shortener_servicepb.YourItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*shortener_servicepb.YourItem, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}

	return items, nil
}

// Update modifies an existing item
func (s *MemoryStore) Update(item *shortener_servicepb.YourItem) error {
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

// Delete removes an item by ID
func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[id]; !exists {
		return fmt.Errorf("item with id %s not found", id)
	}

	delete(s.items, id)
	return nil
}
