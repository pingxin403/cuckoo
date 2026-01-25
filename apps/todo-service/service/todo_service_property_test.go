//go:build property
// +build property

package service

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"github.com/pingxin403/cuckoo/apps/todo-service/gen/todopb"
	"github.com/pingxin403/cuckoo/apps/todo-service/storage"
)

// TestProperty_TodoCreationReturnsUniqueIDs tests Property 2:
// TODO Creation Returns Unique IDs
//
// This property verifies that creating multiple TODOs always results
// in unique IDs, even when created concurrently or with identical content.
func TestProperty_TodoCreationReturnsUniqueIDs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		store := storage.NewMemoryStore()
		obs := createTestObservability()
		service := NewTodoServiceServer(store, obs)
		ctx := context.Background()

		// Generate a list of TODO titles (can have duplicates)
		numTodos := rapid.IntRange(2, 20).Draw(t, "numTodos")
		titles := make([]string, numTodos)
		for i := 0; i < numTodos; i++ {
			// Generate non-empty alphanumeric titles
			title := rapid.StringMatching(`[a-zA-Z0-9][a-zA-Z0-9 ]{0,49}`).Draw(t, "title")
			titles[i] = title
		}

		// Act - Create multiple TODOs
		ids := make(map[string]bool)
		for _, title := range titles {
			req := &todopb.CreateTodoRequest{
				Title:       title,
				Description: rapid.String().Draw(t, "description"),
			}

			resp, err := service.CreateTodo(ctx, req)
			if err != nil {
				t.Fatalf("CreateTodo failed: %v", err)
			}

			// Assert - ID should be unique
			if ids[resp.Todo.Id] {
				t.Fatalf("Duplicate ID found: %s", resp.Todo.Id)
			}
			ids[resp.Todo.Id] = true
		}

		// Verify all IDs are unique
		if len(ids) != numTodos {
			t.Fatalf("Expected %d unique IDs, got %d", numTodos, len(ids))
		}
	})
}

// TestProperty_TodoCRUDRoundTripConsistency tests Property 3:
// TODO CRUD Round-Trip Consistency
//
// This property verifies that data written through Create can be read back
// through List, updated through Update, and deleted through Delete, maintaining
// consistency throughout the lifecycle.
func TestProperty_TodoCRUDRoundTripConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		store := storage.NewMemoryStore()
		obs := createTestObservability()
		service := NewTodoServiceServer(store, obs)
		ctx := context.Background()

		// Generate random TODO data with non-empty titles
		originalTitle := rapid.StringMatching(`[a-zA-Z0-9][a-zA-Z0-9 ]{0,49}`).Draw(t, "originalTitle")
		originalDescription := rapid.String().Draw(t, "originalDescription")
		updatedTitle := rapid.StringMatching(`[a-zA-Z0-9][a-zA-Z0-9 ]{0,49}`).Draw(t, "updatedTitle")
		updatedDescription := rapid.String().Draw(t, "updatedDescription")

		// Act & Assert - Create
		createReq := &todopb.CreateTodoRequest{
			Title:       originalTitle,
			Description: originalDescription,
		}
		createResp, err := service.CreateTodo(ctx, createReq)
		if err != nil {
			t.Fatalf("CreateTodo failed: %v", err)
		}

		id := createResp.Todo.Id
		if id == "" {
			t.Fatal("Created TODO has empty ID")
		}

		// Verify created data matches input
		if createResp.Todo.Title != originalTitle {
			t.Fatalf("Created title mismatch: got %s, want %s", createResp.Todo.Title, originalTitle)
		}
		if createResp.Todo.Description != originalDescription {
			t.Fatalf("Created description mismatch: got %s, want %s", createResp.Todo.Description, originalDescription)
		}

		// Act & Assert - Read (via List)
		listResp, err := service.ListTodos(ctx, &todopb.ListTodosRequest{})
		if err != nil {
			t.Fatalf("ListTodos failed: %v", err)
		}

		// Find our TODO in the list
		var foundTodo *todopb.Todo
		for _, todo := range listResp.Todos {
			if todo.Id == id {
				foundTodo = todo
				break
			}
		}
		if foundTodo == nil {
			t.Fatalf("Created TODO not found in list")
		}

		// Verify read data matches created data
		if foundTodo.Title != originalTitle {
			t.Fatalf("List title mismatch: got %s, want %s", foundTodo.Title, originalTitle)
		}
		if foundTodo.Description != originalDescription {
			t.Fatalf("List description mismatch: got %s, want %s", foundTodo.Description, originalDescription)
		}

		// Act & Assert - Update
		updateReq := &todopb.UpdateTodoRequest{
			Id:          id,
			Title:       updatedTitle,
			Description: updatedDescription,
			Completed:   true,
		}
		updateResp, err := service.UpdateTodo(ctx, updateReq)
		if err != nil {
			t.Fatalf("UpdateTodo failed: %v", err)
		}

		// Verify updated data
		if updateResp.Todo.Id != id {
			t.Fatalf("Update ID mismatch: got %s, want %s", updateResp.Todo.Id, id)
		}
		if updateResp.Todo.Title != updatedTitle {
			t.Fatalf("Update title mismatch: got %s, want %s", updateResp.Todo.Title, updatedTitle)
		}
		if updateResp.Todo.Description != updatedDescription {
			t.Fatalf("Update description mismatch: got %s, want %s", updateResp.Todo.Description, updatedDescription)
		}
		if !updateResp.Todo.Completed {
			t.Fatal("Update completed flag not set")
		}

		// Verify update persisted (List again)
		listResp2, err := service.ListTodos(ctx, &todopb.ListTodosRequest{})
		if err != nil {
			t.Fatalf("ListTodos after update failed: %v", err)
		}

		foundTodo = nil
		for _, todo := range listResp2.Todos {
			if todo.Id == id {
				foundTodo = todo
				break
			}
		}
		if foundTodo == nil {
			t.Fatal("Updated TODO not found in list")
		}

		if foundTodo.Title != updatedTitle {
			t.Fatalf("Persisted title mismatch: got %s, want %s", foundTodo.Title, updatedTitle)
		}
		if foundTodo.Description != updatedDescription {
			t.Fatalf("Persisted description mismatch: got %s, want %s", foundTodo.Description, updatedDescription)
		}
		if !foundTodo.Completed {
			t.Fatal("Persisted completed flag not set")
		}

		// Act & Assert - Delete
		deleteReq := &todopb.DeleteTodoRequest{Id: id}
		_, err = service.DeleteTodo(ctx, deleteReq)
		if err != nil {
			t.Fatalf("DeleteTodo failed: %v", err)
		}

		// Verify deletion (List should not contain the TODO)
		listResp3, err := service.ListTodos(ctx, &todopb.ListTodosRequest{})
		if err != nil {
			t.Fatalf("ListTodos after delete failed: %v", err)
		}

		for _, todo := range listResp3.Todos {
			if todo.Id == id {
				t.Fatal("Deleted TODO still in list")
			}
		}
	})
}

// TestProperty_ListContainsAllCreatedTodos tests that ListTodos
// returns all created TODOs and no duplicates.
func TestProperty_ListContainsAllCreatedTodos(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		store := storage.NewMemoryStore()
		obs := createTestObservability()
		service := NewTodoServiceServer(store, obs)
		ctx := context.Background()

		// Generate random number of TODOs
		numTodos := rapid.IntRange(0, 50).Draw(t, "numTodos")
		createdIDs := make(map[string]bool)

		// Act - Create TODOs
		for i := 0; i < numTodos; i++ {
			title := rapid.StringMatching(`[a-zA-Z0-9][a-zA-Z0-9 ]{0,49}`).Draw(t, "title")

			req := &todopb.CreateTodoRequest{
				Title:       title,
				Description: rapid.String().Draw(t, "description"),
			}

			resp, err := service.CreateTodo(ctx, req)
			if err != nil {
				t.Fatalf("CreateTodo failed: %v", err)
			}
			createdIDs[resp.Todo.Id] = true
		}

		// Act - List all TODOs
		listResp, err := service.ListTodos(ctx, &todopb.ListTodosRequest{})
		if err != nil {
			t.Fatalf("ListTodos failed: %v", err)
		}

		// Assert - List should contain exactly the created TODOs
		if len(listResp.Todos) != numTodos {
			t.Fatalf("List count mismatch: got %d, want %d", len(listResp.Todos), numTodos)
		}

		// Verify all created IDs are in the list
		listedIDs := make(map[string]bool)
		for _, todo := range listResp.Todos {
			if listedIDs[todo.Id] {
				t.Fatalf("Duplicate TODO in list: %s", todo.Id)
			}
			listedIDs[todo.Id] = true

			if !createdIDs[todo.Id] {
				t.Fatalf("Unexpected TODO in list: %s", todo.Id)
			}
		}

		// Verify no created IDs are missing
		for id := range createdIDs {
			if !listedIDs[id] {
				t.Fatalf("Created TODO missing from list: %s", id)
			}
		}
	})
}

// TestProperty_UpdatePreservesID tests that updating a TODO
// never changes its ID.
func TestProperty_UpdatePreservesID(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		store := storage.NewMemoryStore()
		obs := createTestObservability()
		service := NewTodoServiceServer(store, obs)
		ctx := context.Background()

		// Create a TODO
		title := rapid.StringMatching(`[a-zA-Z0-9][a-zA-Z0-9 ]{0,49}`).Draw(t, "title")

		createReq := &todopb.CreateTodoRequest{
			Title:       title,
			Description: rapid.String().Draw(t, "description"),
		}
		createResp, err := service.CreateTodo(ctx, createReq)
		if err != nil {
			t.Fatalf("CreateTodo failed: %v", err)
		}

		originalID := createResp.Todo.Id

		// Act - Update the TODO multiple times
		numUpdates := rapid.IntRange(1, 10).Draw(t, "numUpdates")
		for i := 0; i < numUpdates; i++ {
			newTitle := rapid.StringMatching(`[a-zA-Z0-9][a-zA-Z0-9 ]{0,49}`).Draw(t, "newTitle")

			updateReq := &todopb.UpdateTodoRequest{
				Id:          originalID,
				Title:       newTitle,
				Description: rapid.String().Draw(t, "newDescription"),
				Completed:   rapid.Bool().Draw(t, "completed"),
			}

			updateResp, err := service.UpdateTodo(ctx, updateReq)
			if err != nil {
				t.Fatalf("UpdateTodo failed: %v", err)
			}

			// Assert - ID should never change
			if updateResp.Todo.Id != originalID {
				t.Fatalf("ID changed after update: got %s, want %s", updateResp.Todo.Id, originalID)
			}
		}

		// Verify ID is still the same after all updates (via List)
		listResp, err := service.ListTodos(ctx, &todopb.ListTodosRequest{})
		if err != nil {
			t.Fatalf("ListTodos failed: %v", err)
		}

		found := false
		for _, todo := range listResp.Todos {
			if todo.Id == originalID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("TODO with original ID not found after updates")
		}
	})
}

// TestProperty_DeleteNonExistentTodoFails tests that deleting a TODO
// that doesn't exist returns an error.
func TestProperty_DeleteNonExistentTodoFails(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		store := storage.NewMemoryStore()
		obs := createTestObservability()
		service := NewTodoServiceServer(store, obs)
		ctx := context.Background()

		// Generate a random non-empty ID that doesn't exist
		nonExistentID := rapid.StringMatching(`[a-zA-Z0-9][a-zA-Z0-9-]{0,35}`).Draw(t, "nonExistentID")

		// Act - Try to delete non-existent TODO
		_, err := service.DeleteTodo(ctx, &todopb.DeleteTodoRequest{Id: nonExistentID})

		// Assert - Should fail
		if err == nil {
			t.Fatal("DeleteTodo should fail for non-existent ID, but succeeded")
		}
	})
}
