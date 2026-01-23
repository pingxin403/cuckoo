package storage

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryStore_Create tests the Create method
func TestMemoryStore_Create(t *testing.T) {
	// Arrange
	store := NewMemoryStore()
	item := &Item{
		ID:    "test-id",
		Field: "test-value",
	}

	// Act
	err := store.Create(item)

	// Assert
	require.NoError(t, err)

	// Verify item was stored
	retrieved, err := store.Get("test-id")
	require.NoError(t, err)
	assert.Equal(t, item.ID, retrieved.ID)
	assert.Equal(t, item.Field, retrieved.Field)
}

func TestMemoryStore_Create_Duplicate(t *testing.T) {
	// Arrange
	store := NewMemoryStore()
	item := &Item{ID: "test-id", Field: "value"}

	// Act
	err1 := store.Create(item)
	err2 := store.Create(item)

	// Assert
	require.NoError(t, err1)
	require.Error(t, err2)
	assert.Contains(t, err2.Error(), "already exists")
}

func TestMemoryStore_Get(t *testing.T) {
	// Arrange
	store := NewMemoryStore()
	item := &Item{ID: "test-id", Field: "test-value"}
	err := store.Create(item)
	require.NoError(t, err)

	// Act
	retrieved, err := store.Get("test-id")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, item.ID, retrieved.ID)
	assert.Equal(t, item.Field, retrieved.Field)
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	// Arrange
	store := NewMemoryStore()

	// Act
	_, err := store.Get("non-existent-id")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryStore_List(t *testing.T) {
	// Arrange
	store := NewMemoryStore()
	items := []*Item{
		{ID: "id1", Field: "value1"},
		{ID: "id2", Field: "value2"},
		{ID: "id3", Field: "value3"},
	}

	for _, item := range items {
		err := store.Create(item)
		require.NoError(t, err)
	}

	// Act
	result := store.List()

	// Assert
	assert.Len(t, result, 3)

	// Verify all items are present
	ids := make(map[string]bool)
	for _, item := range result {
		ids[item.ID] = true
	}
	assert.True(t, ids["id1"])
	assert.True(t, ids["id2"])
	assert.True(t, ids["id3"])
}

func TestMemoryStore_List_Empty(t *testing.T) {
	// Arrange
	store := NewMemoryStore()

	// Act
	result := store.List()

	// Assert
	assert.Empty(t, result)
}

func TestMemoryStore_Update(t *testing.T) {
	// Arrange
	store := NewMemoryStore()
	original := &Item{ID: "test-id", Field: "original"}
	err := store.Create(original)
	require.NoError(t, err)

	// Act
	updated := &Item{ID: "test-id", Field: "updated"}
	err = store.Update(updated)

	// Assert
	require.NoError(t, err)

	// Verify update
	retrieved, err := store.Get("test-id")
	require.NoError(t, err)
	assert.Equal(t, "updated", retrieved.Field)
}

func TestMemoryStore_Update_NotFound(t *testing.T) {
	// Arrange
	store := NewMemoryStore()
	item := &Item{ID: "non-existent-id", Field: "value"}

	// Act
	err := store.Update(item)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryStore_Delete(t *testing.T) {
	// Arrange
	store := NewMemoryStore()
	item := &Item{ID: "test-id", Field: "value"}
	err := store.Create(item)
	require.NoError(t, err)

	// Act
	err = store.Delete("test-id")

	// Assert
	require.NoError(t, err)

	// Verify deletion
	_, err = store.Get("test-id")
	require.Error(t, err)
}

func TestMemoryStore_Delete_NotFound(t *testing.T) {
	// Arrange
	store := NewMemoryStore()

	// Act
	err := store.Delete("non-existent-id")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestMemoryStore_ConcurrentAccess tests thread safety
func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	// Arrange
	store := NewMemoryStore()
	const numGoroutines = 100
	var wg sync.WaitGroup

	// Act - Concurrent creates
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			item := &Item{
				ID:    string(rune(id)),
				Field: "value",
			}
			_ = store.Create(item)
		}(i)
	}
	wg.Wait()

	// Assert - All items should be stored
	items := store.List()
	assert.Len(t, items, numGoroutines)
}

// TestMemoryStore_ConcurrentReadWrite tests concurrent reads and writes
func TestMemoryStore_ConcurrentReadWrite(t *testing.T) {
	// Arrange
	store := NewMemoryStore()
	item := &Item{ID: "test-id", Field: "initial"}
	err := store.Create(item)
	require.NoError(t, err)

	const numReaders = 50
	const numWriters = 50
	var wg sync.WaitGroup

	// Act - Concurrent reads and writes
	wg.Add(numReaders + numWriters)

	// Readers
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			_, _ = store.Get("test-id")
		}()
	}

	// Writers
	for i := 0; i < numWriters; i++ {
		go func(val int) {
			defer wg.Done()
			updated := &Item{ID: "test-id", Field: string(rune(val))}
			_ = store.Update(updated)
		}(i)
	}

	wg.Wait()

	// Assert - No panics, item still exists
	retrieved, err := store.Get("test-id")
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
}

// TestMemoryStore_CRUDCycle tests a complete CRUD cycle
func TestMemoryStore_CRUDCycle(t *testing.T) {
	// Arrange
	store := NewMemoryStore()

	// Create
	item := &Item{ID: "cycle-id", Field: "initial"}
	err := store.Create(item)
	require.NoError(t, err)

	// Read
	retrieved, err := store.Get("cycle-id")
	require.NoError(t, err)
	assert.Equal(t, "initial", retrieved.Field)

	// Update
	updated := &Item{ID: "cycle-id", Field: "modified"}
	err = store.Update(updated)
	require.NoError(t, err)

	// Verify update
	retrieved, err = store.Get("cycle-id")
	require.NoError(t, err)
	assert.Equal(t, "modified", retrieved.Field)

	// List
	items := store.List()
	assert.Len(t, items, 1)

	// Delete
	err = store.Delete("cycle-id")
	require.NoError(t, err)

	// Verify deletion
	_, err = store.Get("cycle-id")
	require.Error(t, err)

	// List should be empty
	items = store.List()
	assert.Empty(t, items)
}
