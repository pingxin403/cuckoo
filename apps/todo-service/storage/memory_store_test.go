package storage

import (
	"sync"
	"testing"

	"github.com/pingxin403/cuckoo/api/gen/go/todopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMemoryStore_Create(t *testing.T) {
	store := NewMemoryStore()

	t.Run("should create a new TODO", func(t *testing.T) {
		todo := &todopb.Todo{
			Id:          "test-id-1",
			Title:       "Test TODO",
			Description: "Test Description",
			Completed:   false,
			CreatedAt:   timestamppb.Now(),
			UpdatedAt:   timestamppb.Now(),
		}

		err := store.Create(todo)
		require.NoError(t, err)

		// Verify TODO was created
		retrieved, err := store.Get("test-id-1")
		require.NoError(t, err)
		assert.Equal(t, todo.Id, retrieved.Id)
		assert.Equal(t, todo.Title, retrieved.Title)
	})

	t.Run("should return error for nil TODO", func(t *testing.T) {
		err := store.Create(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("should allow duplicate IDs (overwrite)", func(t *testing.T) {
		todo1 := &todopb.Todo{
			Id:    "dup-id",
			Title: "First",
		}
		todo2 := &todopb.Todo{
			Id:    "dup-id",
			Title: "Second",
		}

		err := store.Create(todo1)
		require.NoError(t, err)

		err = store.Create(todo2)
		require.NoError(t, err)

		retrieved, err := store.Get("dup-id")
		require.NoError(t, err)
		assert.Equal(t, "Second", retrieved.Title)
	})
}

func TestMemoryStore_Get(t *testing.T) {
	store := NewMemoryStore()

	t.Run("should retrieve existing TODO", func(t *testing.T) {
		todo := &todopb.Todo{
			Id:    "get-test-1",
			Title: "Get Test",
		}
		err := store.Create(todo)
		require.NoError(t, err)

		retrieved, err := store.Get("get-test-1")
		require.NoError(t, err)
		assert.Equal(t, todo.Id, retrieved.Id)
		assert.Equal(t, todo.Title, retrieved.Title)
	})

	t.Run("should return error for non-existent TODO", func(t *testing.T) {
		_, err := store.Get("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestMemoryStore_List(t *testing.T) {
	t.Run("should return empty list for new store", func(t *testing.T) {
		store := NewMemoryStore()
		todos, err := store.List()
		require.NoError(t, err)
		assert.Empty(t, todos)
	})

	t.Run("should return all TODOs", func(t *testing.T) {
		store := NewMemoryStore()

		// Create multiple TODOs
		for i := 1; i <= 3; i++ {
			todo := &todopb.Todo{
				Id:    string(rune(i)),
				Title: "TODO " + string(rune(i)),
			}
			err := store.Create(todo)
			require.NoError(t, err)
		}

		todos, err := store.List()
		require.NoError(t, err)
		assert.Len(t, todos, 3)
	})
}

func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore()

	t.Run("should update existing TODO", func(t *testing.T) {
		// Create initial TODO
		todo := &todopb.Todo{
			Id:        "update-test-1",
			Title:     "Original Title",
			Completed: false,
		}
		err := store.Create(todo)
		require.NoError(t, err)

		// Update TODO
		todo.Title = "Updated Title"
		todo.Completed = true
		err = store.Update(todo)
		require.NoError(t, err)

		// Verify update
		retrieved, err := store.Get("update-test-1")
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", retrieved.Title)
		assert.True(t, retrieved.Completed)
	})

	t.Run("should return error for non-existent TODO", func(t *testing.T) {
		todo := &todopb.Todo{
			Id:    "non-existent",
			Title: "Test",
		}
		err := store.Update(todo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("should return error for nil TODO", func(t *testing.T) {
		err := store.Update(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()

	t.Run("should delete existing TODO", func(t *testing.T) {
		// Create TODO
		todo := &todopb.Todo{
			Id:    "delete-test-1",
			Title: "To Be Deleted",
		}
		err := store.Create(todo)
		require.NoError(t, err)

		// Delete TODO
		err = store.Delete("delete-test-1")
		require.NoError(t, err)

		// Verify deletion
		_, err = store.Get("delete-test-1")
		assert.Error(t, err)
	})

	t.Run("should return error for non-existent TODO", func(t *testing.T) {
		err := store.Delete("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()
	const numGoroutines = 100

	t.Run("should handle concurrent creates", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				todo := &todopb.Todo{
					Id:    string(rune(id)),
					Title: "Concurrent TODO",
				}
				err := store.Create(todo)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()

		// Verify all TODOs were created
		todos, err := store.List()
		require.NoError(t, err)
		assert.Len(t, todos, numGoroutines)
	})

	t.Run("should handle concurrent reads", func(t *testing.T) {
		// Create a TODO first
		todo := &todopb.Todo{
			Id:    "concurrent-read",
			Title: "Read Test",
		}
		err := store.Create(todo)
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				retrieved, err := store.Get("concurrent-read")
				assert.NoError(t, err)
				assert.Equal(t, "Read Test", retrieved.Title)
			}()
		}

		wg.Wait()
	})

	t.Run("should handle concurrent updates", func(t *testing.T) {
		// Create a TODO first
		todo := &todopb.Todo{
			Id:    "concurrent-update",
			Title: "Original",
		}
		err := store.Create(todo)
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				updatedTodo := &todopb.Todo{
					Id:    "concurrent-update",
					Title: "Updated",
				}
				err := store.Update(updatedTodo)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()

		// Verify TODO still exists
		retrieved, err := store.Get("concurrent-update")
		require.NoError(t, err)
		assert.Equal(t, "Updated", retrieved.Title)
	})
}
