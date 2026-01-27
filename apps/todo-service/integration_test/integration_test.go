//go:build integration
// +build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	todopb "github.com/pingxin403/cuckoo/api/gen/go/todopb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	grpcAddr = getEnv("GRPC_ADDR", "localhost:9091")
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupClient(t *testing.T) (todopb.TodoServiceClient, *grpc.ClientConn) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("Failed to connect to gRPC server: %v", err)
	}

	client := todopb.NewTodoServiceClient(conn)
	return client, conn
}

func TestEndToEndFlow(t *testing.T) {
	client, conn := setupClient(t)
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	ctx := context.Background()

	// 1. Create a TODO
	createReq := &todopb.CreateTodoRequest{
		Title:       "Test TODO",
		Description: "This is a test TODO item",
	}

	createResp, err := client.CreateTodo(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	if createResp.Todo == nil {
		t.Fatal("Created TODO is nil")
	}

	todoID := createResp.Todo.Id
	if todoID == "" {
		t.Fatal("Created TODO has empty ID")
	}

	t.Logf("Created TODO with ID: %s", todoID)

	// Verify fields
	if createResp.Todo.Title != "Test TODO" {
		t.Errorf("Expected title 'Test TODO', got '%s'", createResp.Todo.Title)
	}
	if createResp.Todo.Description != "This is a test TODO item" {
		t.Errorf("Expected description 'This is a test TODO item', got '%s'", createResp.Todo.Description)
	}
	if createResp.Todo.Completed {
		t.Error("New TODO should not be completed")
	}

	// 2. List TODOs
	listResp, err := client.ListTodos(ctx, &todopb.ListTodosRequest{})
	if err != nil {
		t.Fatalf("ListTodos failed: %v", err)
	}

	if len(listResp.Todos) == 0 {
		t.Fatal("ListTodos returned empty list")
	}

	found := false
	for _, todo := range listResp.Todos {
		if todo.Id == todoID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Created TODO with ID %s not found in list", todoID)
	}

	// 3. Update TODO
	updateReq := &todopb.UpdateTodoRequest{
		Id:          todoID,
		Title:       "Updated TODO",
		Description: "This TODO has been updated",
		Completed:   true,
	}

	updateResp, err := client.UpdateTodo(ctx, updateReq)
	if err != nil {
		t.Fatalf("UpdateTodo failed: %v", err)
	}

	if updateResp.Todo.Title != "Updated TODO" {
		t.Errorf("Expected updated title 'Updated TODO', got '%s'", updateResp.Todo.Title)
	}
	if !updateResp.Todo.Completed {
		t.Error("TODO should be marked as completed")
	}

	// 4. Delete TODO
	deleteReq := &todopb.DeleteTodoRequest{
		Id: todoID,
	}

	_, err = client.DeleteTodo(ctx, deleteReq)
	if err != nil {
		t.Fatalf("DeleteTodo failed: %v", err)
	}

	// 5. Verify deletion
	listResp2, err := client.ListTodos(ctx, &todopb.ListTodosRequest{})
	if err != nil {
		t.Fatalf("ListTodos after delete failed: %v", err)
	}

	for _, todo := range listResp2.Todos {
		if todo.Id == todoID {
			t.Errorf("Deleted TODO with ID %s still exists", todoID)
		}
	}

	t.Log("✓ End-to-end flow completed successfully")
}

func TestCreateMultipleTodos(t *testing.T) {
	client, conn := setupClient(t)
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	ctx := context.Background()

	// Create multiple TODOs
	todos := []struct {
		title       string
		description string
	}{
		{"Buy groceries", "Milk, eggs, bread"},
		{"Write report", "Q4 financial report"},
		{"Call dentist", "Schedule appointment"},
	}

	createdIDs := make([]string, 0, len(todos))

	for _, todo := range todos {
		req := &todopb.CreateTodoRequest{
			Title:       todo.title,
			Description: todo.description,
		}

		resp, err := client.CreateTodo(ctx, req)
		if err != nil {
			t.Fatalf("CreateTodo failed for '%s': %v", todo.title, err)
		}

		createdIDs = append(createdIDs, resp.Todo.Id)
	}

	// Verify all TODOs exist
	listResp, err := client.ListTodos(ctx, &todopb.ListTodosRequest{})
	if err != nil {
		t.Fatalf("ListTodos failed: %v", err)
	}

	for _, id := range createdIDs {
		found := false
		for _, todo := range listResp.Todos {
			if todo.Id == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("TODO with ID %s not found", id)
		}
	}

	// Cleanup
	for _, id := range createdIDs {
		_, err := client.DeleteTodo(ctx, &todopb.DeleteTodoRequest{Id: id})
		if err != nil {
			t.Logf("Warning: Failed to cleanup TODO %s: %v", id, err)
		}
	}

	t.Log("✓ Multiple TODOs created and verified successfully")
}

func TestUpdateNonexistentTodo(t *testing.T) {
	client, conn := setupClient(t)
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	ctx := context.Background()

	// Try to update a TODO that doesn't exist
	req := &todopb.UpdateTodoRequest{
		Id:          "nonexistent-id",
		Title:       "Should Fail",
		Description: "This should fail",
		Completed:   false,
	}

	_, err := client.UpdateTodo(ctx, req)
	if err == nil {
		t.Error("Expected error when updating nonexistent TODO, got nil")
	}

	t.Logf("✓ Correctly rejected update of nonexistent TODO: %v", err)
}

func TestDeleteNonexistentTodo(t *testing.T) {
	client, conn := setupClient(t)
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	ctx := context.Background()

	// Try to delete a TODO that doesn't exist
	req := &todopb.DeleteTodoRequest{
		Id: "nonexistent-id",
	}

	_, err := client.DeleteTodo(ctx, req)
	if err == nil {
		t.Error("Expected error when deleting nonexistent TODO, got nil")
	}

	t.Logf("✓ Correctly rejected deletion of nonexistent TODO: %v", err)
}

func TestConcurrentOperations(t *testing.T) {
	client, conn := setupClient(t)
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	ctx := context.Background()

	// Create multiple TODOs concurrently
	numTodos := 10
	results := make(chan string, numTodos)
	errors := make(chan error, numTodos)

	for i := 0; i < numTodos; i++ {
		go func(index int) {
			req := &todopb.CreateTodoRequest{
				Title:       fmt.Sprintf("Concurrent TODO %d", index),
				Description: fmt.Sprintf("Created concurrently %d", index),
			}

			resp, err := client.CreateTodo(ctx, req)
			if err != nil {
				errors <- err
				return
			}

			results <- resp.Todo.Id
		}(i)
	}

	// Collect results
	createdIDs := make([]string, 0, numTodos)
	for i := 0; i < numTodos; i++ {
		select {
		case id := <-results:
			createdIDs = append(createdIDs, id)
		case err := <-errors:
			t.Fatalf("Concurrent creation failed: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	// Verify all IDs are unique
	idMap := make(map[string]bool)
	for _, id := range createdIDs {
		if idMap[id] {
			t.Errorf("Duplicate ID found: %s", id)
		}
		idMap[id] = true
	}

	// Cleanup
	for _, id := range createdIDs {
		_, err := client.DeleteTodo(ctx, &todopb.DeleteTodoRequest{Id: id})
		if err != nil {
			t.Logf("Warning: Failed to cleanup TODO %s: %v", id, err)
		}
	}

	t.Logf("✓ Created %d TODOs concurrently, all with unique IDs", numTodos)
}

func TestEmptyList(t *testing.T) {
	client, conn := setupClient(t)
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	ctx := context.Background()

	// Get current list
	listResp, err := client.ListTodos(ctx, &todopb.ListTodosRequest{})
	if err != nil {
		t.Fatalf("ListTodos failed: %v", err)
	}

	// Delete all TODOs
	for _, todo := range listResp.Todos {
		_, err := client.DeleteTodo(ctx, &todopb.DeleteTodoRequest{Id: todo.Id})
		if err != nil {
			t.Logf("Warning: Failed to delete TODO %s: %v", todo.Id, err)
		}
	}

	// Verify list is empty
	listResp2, err := client.ListTodos(ctx, &todopb.ListTodosRequest{})
	if err != nil {
		t.Fatalf("ListTodos after cleanup failed: %v", err)
	}

	if len(listResp2.Todos) != 0 {
		t.Errorf("Expected empty list, got %d TODOs", len(listResp2.Todos))
	}

	t.Log("✓ Empty list handled correctly")
}

func TestServiceAvailability(t *testing.T) {
	client, conn := setupClient(t)
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	ctx := context.Background()

	// Simple availability check
	_, err := client.ListTodos(ctx, &todopb.ListTodosRequest{})
	if err != nil {
		t.Fatalf("Service not available: %v", err)
	}

	t.Log("✓ Service is available and responding")
}
