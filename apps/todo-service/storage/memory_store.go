package storage

import (
	"fmt"
	"sync"

	"github.com/pingxin/cuckoo/apps/todo-service/gen/todopb"
)

// TodoStore defines the interface for TODO storage operations
type TodoStore interface {
	Create(todo *todopb.Todo) error
	Get(id string) (*todopb.Todo, error)
	List() ([]*todopb.Todo, error)
	Update(todo *todopb.Todo) error
	Delete(id string) error
}

// MemoryStore implements TodoStore using an in-memory map
type MemoryStore struct {
	mu    sync.RWMutex
	todos map[string]*todopb.Todo
}

// NewMemoryStore creates a new in-memory TODO store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		todos: make(map[string]*todopb.Todo),
	}
}

// Create adds a new TODO to the store
func (s *MemoryStore) Create(todo *todopb.Todo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if todo == nil {
		return fmt.Errorf("todo cannot be nil")
	}

	s.todos[todo.Id] = todo
	return nil
}

// Get retrieves a TODO by ID
func (s *MemoryStore) Get(id string) (*todopb.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todo, exists := s.todos[id]
	if !exists {
		return nil, fmt.Errorf("todo with id %s not found", id)
	}

	return todo, nil
}

// List returns all TODOs
func (s *MemoryStore) List() ([]*todopb.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todos := make([]*todopb.Todo, 0, len(s.todos))
	for _, todo := range s.todos {
		todos = append(todos, todo)
	}

	return todos, nil
}

// Update modifies an existing TODO
func (s *MemoryStore) Update(todo *todopb.Todo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if todo == nil {
		return fmt.Errorf("todo cannot be nil")
	}

	if _, exists := s.todos[todo.Id]; !exists {
		return fmt.Errorf("todo with id %s not found", todo.Id)
	}

	s.todos[todo.Id] = todo
	return nil
}

// Delete removes a TODO by ID
func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.todos[id]; !exists {
		return fmt.Errorf("todo with id %s not found", id)
	}

	delete(s.todos, id)
	return nil
}
