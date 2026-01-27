package storage

import (
	"sync"
	"testing"

	"github.com/pingxin403/cuckoo/api/gen/go/{{PROTO_PACKAGE}}"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryStore_Create tests the Create method
func TestMemoryStore_Create(t *testing.T) {
	store := NewMemoryStore()

	t.Run("should create a new item", func(t *testing.T) {
		item := &{{PROTO_PACKAGE}}.YourItem{
			Id: "test-id-1",
		}

		err := store.Create(item)
		require.NoError(t, err)

		// Verify item was created
		retrieved, err := store.Get("test-id-1")
		require.NoError(t, err)
		assert.Equal(t, item.Id, retrieved.Id)
	})

	t.Run("should return error for nil item", func(t *testing.T) {
		err := store.Create(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("should allow duplicate IDs (overwrite)", func(t *testing.T) {
		item1 := &{{PROTO_PACKAGE}}.YourItem{
			Id: "dup-id",
		}
		item2 := &{{PROTO_PACKAGE}}.YourItem{
			Id: "dup-id",
		}

		err := store.Create(item1)
		require.NoError(t, err)

		err = store.Create(item2)
		require.NoError(t, err)

		retrieved, err := store.Get("dup-id")
		require.NoError(t, err)
		assert.Equal(t, "dup-id", retrieved.Id)
	})
}

// TestMemoryStore_Get tests the Get method
func TestMemoryStore_Get(t *testing.T) {
	store := NewMemoryStore()

	t.Run("should retrieve existing item", func(t *testing.T) {
		item := &{{PROTO_PACKAGE}}.YourItem{
			Id: "get-test-1",
		}
		err := store.Create(item)
		require.NoError(t, err)

		retrieved, err := store.Get("get-test-1")
		require.NoError(t, err)
		assert.Equal(t, item.Id, retrieved.Id)
	})

	t.Run("should return error for non-existent item", func(t *testing.T) {
		_, err := store.Get("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestMemoryStore_List tests the List method
func TestMemoryStore_List(t *testing.T) {
	t.Run("should return empty list for new store", func(t *testing.T) {
		store := NewMemoryStore()
		items, err := store.List()
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("should return all items", func(t *testing.T) {
		store := NewMemoryStore()

		// Create multiple items
		for i := 1; i <= 3; i++ {
			item := &{{PROTO_PACKAGE}}.YourItem{
				Id: string(rune('0' + i)),
			}
			err := store.Create(item)
			require.NoError(t, err)
		}

		items, err := store.List()
		require.NoError(t, err)
		assert.Len(t, items, 3)
	})
}

// TestMemoryStore_Update tests the Update method
func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore()

	t.Run("should update existing item", func(t *testing.T) {
		// Create initial item
		item := &{{PROTO_PACKAGE}}.YourItem{
			Id: "update-test-1",
		}
		err := store.Create(item)
		require.NoError(t, err)

		// Update item
		updatedItem := &{{PROTO_PACKAGE}}.YourItem{
			Id: "update-test-1",
		}
		err = store.Update(updatedItem)
		require.NoError(t, err)

		// Verify update
		retrieved, err := store.Get("update-test-1")
		require.NoError(t, err)
		assert.Equal(t, "update-test-1", retrieved.Id)
	})

	t.Run("should return error for non-existent item", func(t *testing.T) {
		item := &{{PROTO_PACKAGE}}.YourItem{
			Id: "non-existent",
		}
		err := store.Update(item)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("should return error for nil item", func(t *testing.T) {
		err := store.Update(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})
}

// TestMemoryStore_Delete tests the Delete method
func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()

	t.Run("should delete existing item", func(t *testing.T) {
		// Create item
		item := &{{PROTO_PACKAGE}}.YourItem{
			Id: "delete-test-1",
		}
		err := store.Create(item)
		require.NoError(t, err)

		// Delete item
		err = store.Delete("delete-test-1")
		require.NoError(t, err)

		// Verify deletion
		_, err = store.Get("delete-test-1")
		assert.Error(t, err)
	})

	t.Run("should return error for non-existent item", func(t *testing.T) {
		err := store.Delete("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestMemoryStore_ConcurrentAccess tests thread safety
func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()
	const numGoroutines = 100

	t.Run("should handle concurrent creates", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				item := &{{PROTO_PACKAGE}}.YourItem{
					Id: string(rune(id)),
				}
				err := store.Create(item)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()

		// Verify all items were created
		items, err := store.List()
		require.NoError(t, err)
		assert.Len(t, items, numGoroutines)
	})

	t.Run("should handle concurrent reads", func(t *testing.T) {
		// Create an item first
		item := &{{PROTO_PACKAGE}}.YourItem{
			Id: "concurrent-read",
		}
		err := store.Create(item)
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				retrieved, err := store.Get("concurrent-read")
				assert.NoError(t, err)
				assert.Equal(t, "concurrent-read", retrieved.Id)
			}()
		}

		wg.Wait()
	})

	t.Run("should handle concurrent updates", func(t *testing.T) {
		// Create an item first
		item := &{{PROTO_PACKAGE}}.YourItem{
			Id: "concurrent-update",
		}
		err := store.Create(item)
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				updatedItem := &{{PROTO_PACKAGE}}.YourItem{
					Id: "concurrent-update",
				}
				err := store.Update(updatedItem)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()

		// Verify item still exists
		retrieved, err := store.Get("concurrent-update")
		require.NoError(t, err)
		assert.Equal(t, "concurrent-update", retrieved.Id)
	})
}

// TestMemoryStore_CRUDCycle tests a complete CRUD cycle
func TestMemoryStore_CRUDCycle(t *testing.T) {
	store := NewMemoryStore()

	// Create
	item := &{{PROTO_PACKAGE}}.YourItem{
		Id: "cycle-id",
	}
	err := store.Create(item)
	require.NoError(t, err)

	// Read
	retrieved, err := store.Get("cycle-id")
	require.NoError(t, err)
	assert.Equal(t, "cycle-id", retrieved.Id)

	// Update
	updated := &{{PROTO_PACKAGE}}.YourItem{
		Id: "cycle-id",
	}
	err = store.Update(updated)
	require.NoError(t, err)

	// Verify update
	retrieved, err = store.Get("cycle-id")
	require.NoError(t, err)
	assert.Equal(t, "cycle-id", retrieved.Id)

	// List
	items, err := store.List()
	require.NoError(t, err)
	assert.Len(t, items, 1)

	// Delete
	err = store.Delete("cycle-id")
	require.NoError(t, err)

	// Verify deletion
	_, err = store.Get("cycle-id")
	require.Error(t, err)

	// List should be empty
	items, err = store.List()
	require.NoError(t, err)
	assert.Empty(t, items)
}
