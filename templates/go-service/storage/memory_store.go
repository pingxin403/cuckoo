package storage

import (
	"fmt"
	"sync"

	"{{MODULE_PATH}}/gen/{{PROTO_PACKAGE}}"
)

// YourStore defines the interface for storage operations
// TODO: Update this interface to match your data model
type YourStore interface {
	Create(item *{{PROTO_PACKAGE}}.YourItem) error
	Get(id string) (*{{PROTO_PACKAGE}}.YourItem, error)
	List() ([]*{{PROTO_PACKAGE}}.YourItem, error)
	Update(item *{{PROTO_PACKAGE}}.YourItem) error
	Delete(id string) error
}

// MemoryStore implements YourStore using an in-memory map
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]*{{PROTO_PACKAGE}}.YourItem
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items: make(map[string]*{{PROTO_PACKAGE}}.YourItem),
	}
}

// Create adds a new item to the store
func (s *MemoryStore) Create(item *{{PROTO_PACKAGE}}.YourItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	s.items[item.Id] = item
	return nil
}

// Get retrieves an item by ID
func (s *MemoryStore) Get(id string) (*{{PROTO_PACKAGE}}.YourItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[id]
	if !exists {
		return nil, fmt.Errorf("item with id %s not found", id)
	}

	return item, nil
}

// List returns all items
func (s *MemoryStore) List() ([]*{{PROTO_PACKAGE}}.YourItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*{{PROTO_PACKAGE}}.YourItem, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}

	return items, nil
}

// Update modifies an existing item
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
