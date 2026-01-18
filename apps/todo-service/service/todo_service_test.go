package service

import (
	"context"
	"testing"

	"github.com/pingxin403/cuckoo/apps/todo-service/gen/todopb"
	"github.com/pingxin403/cuckoo/apps/todo-service/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTodoServiceServer_CreateTodo(t *testing.T) {
	store := storage.NewMemoryStore()
	service := NewTodoServiceServer(store)
	ctx := context.Background()

	t.Run("should create TODO with valid input", func(t *testing.T) {
		req := &todopb.CreateTodoRequest{
			Title:       "Test TODO",
			Description: "Test Description",
		}

		resp, err := service.CreateTodo(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Todo)

		assert.NotEmpty(t, resp.Todo.Id)
		assert.Equal(t, "Test TODO", resp.Todo.Title)
		assert.Equal(t, "Test Description", resp.Todo.Description)
		assert.False(t, resp.Todo.Completed)
		assert.NotNil(t, resp.Todo.CreatedAt)
		assert.NotNil(t, resp.Todo.UpdatedAt)
	})

	t.Run("should return error for empty title", func(t *testing.T) {
		req := &todopb.CreateTodoRequest{
			Title:       "",
			Description: "Description",
		}

		resp, err := service.CreateTodo(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "title is required")
	})

	t.Run("should return error for whitespace-only title", func(t *testing.T) {
		req := &todopb.CreateTodoRequest{
			Title:       "   ",
			Description: "Description",
		}

		resp, err := service.CreateTodo(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("should create TODO without description", func(t *testing.T) {
		req := &todopb.CreateTodoRequest{
			Title: "Title Only",
		}

		resp, err := service.CreateTodo(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, "Title Only", resp.Todo.Title)
		assert.Empty(t, resp.Todo.Description)
	})

	t.Run("should generate unique IDs for multiple TODOs", func(t *testing.T) {
		ids := make(map[string]bool)

		for i := 0; i < 10; i++ {
			req := &todopb.CreateTodoRequest{
				Title: "TODO",
			}

			resp, err := service.CreateTodo(ctx, req)
			require.NoError(t, err)

			// Check ID is unique
			assert.False(t, ids[resp.Todo.Id], "Duplicate ID generated")
			ids[resp.Todo.Id] = true
		}

		assert.Len(t, ids, 10)
	})
}

func TestTodoServiceServer_ListTodos(t *testing.T) {
	store := storage.NewMemoryStore()
	service := NewTodoServiceServer(store)
	ctx := context.Background()

	t.Run("should return empty list for new store", func(t *testing.T) {
		req := &todopb.ListTodosRequest{}

		resp, err := service.ListTodos(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Todos)
	})

	t.Run("should return all created TODOs", func(t *testing.T) {
		// Create multiple TODOs
		for i := 1; i <= 3; i++ {
			createReq := &todopb.CreateTodoRequest{
				Title: "TODO",
			}
			_, err := service.CreateTodo(ctx, createReq)
			require.NoError(t, err)
		}

		// List TODOs
		listReq := &todopb.ListTodosRequest{}
		resp, err := service.ListTodos(ctx, listReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.GreaterOrEqual(t, len(resp.Todos), 3)
	})
}

func TestTodoServiceServer_UpdateTodo(t *testing.T) {
	store := storage.NewMemoryStore()
	service := NewTodoServiceServer(store)
	ctx := context.Background()

	t.Run("should update existing TODO", func(t *testing.T) {
		// Create TODO first
		createReq := &todopb.CreateTodoRequest{
			Title:       "Original Title",
			Description: "Original Description",
		}
		createResp, err := service.CreateTodo(ctx, createReq)
		require.NoError(t, err)

		// Update TODO
		updateReq := &todopb.UpdateTodoRequest{
			Id:          createResp.Todo.Id,
			Title:       "Updated Title",
			Description: "Updated Description",
			Completed:   true,
		}

		updateResp, err := service.UpdateTodo(ctx, updateReq)
		require.NoError(t, err)
		require.NotNil(t, updateResp)
		require.NotNil(t, updateResp.Todo)

		assert.Equal(t, createResp.Todo.Id, updateResp.Todo.Id)
		assert.Equal(t, "Updated Title", updateResp.Todo.Title)
		assert.Equal(t, "Updated Description", updateResp.Todo.Description)
		assert.True(t, updateResp.Todo.Completed)
		// UpdatedAt should be set (not nil)
		assert.NotNil(t, updateResp.Todo.UpdatedAt)
	})

	t.Run("should return error for empty ID", func(t *testing.T) {
		req := &todopb.UpdateTodoRequest{
			Id:    "",
			Title: "Title",
		}

		resp, err := service.UpdateTodo(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "id is required")
	})

	t.Run("should return error for empty title", func(t *testing.T) {
		req := &todopb.UpdateTodoRequest{
			Id:    "some-id",
			Title: "",
		}

		resp, err := service.UpdateTodo(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "title is required")
	})

	t.Run("should return error for non-existent TODO", func(t *testing.T) {
		req := &todopb.UpdateTodoRequest{
			Id:    "non-existent-id",
			Title: "Title",
		}

		resp, err := service.UpdateTodo(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	t.Run("should preserve created timestamp", func(t *testing.T) {
		// Create TODO
		createReq := &todopb.CreateTodoRequest{
			Title: "Test",
		}
		createResp, err := service.CreateTodo(ctx, createReq)
		require.NoError(t, err)

		// Update TODO
		updateReq := &todopb.UpdateTodoRequest{
			Id:    createResp.Todo.Id,
			Title: "Updated",
		}
		updateResp, err := service.UpdateTodo(ctx, updateReq)
		require.NoError(t, err)

		// CreatedAt should remain the same
		assert.Equal(t, createResp.Todo.CreatedAt.AsTime(), updateResp.Todo.CreatedAt.AsTime())
	})
}

func TestTodoServiceServer_DeleteTodo(t *testing.T) {
	store := storage.NewMemoryStore()
	service := NewTodoServiceServer(store)
	ctx := context.Background()

	t.Run("should delete existing TODO", func(t *testing.T) {
		// Create TODO first
		createReq := &todopb.CreateTodoRequest{
			Title: "To Be Deleted",
		}
		createResp, err := service.CreateTodo(ctx, createReq)
		require.NoError(t, err)

		// Delete TODO
		deleteReq := &todopb.DeleteTodoRequest{
			Id: createResp.Todo.Id,
		}
		deleteResp, err := service.DeleteTodo(ctx, deleteReq)
		require.NoError(t, err)
		require.NotNil(t, deleteResp)
		assert.True(t, deleteResp.Success)

		// Verify TODO is deleted
		listReq := &todopb.ListTodosRequest{}
		listResp, err := service.ListTodos(ctx, listReq)
		require.NoError(t, err)

		for _, todo := range listResp.Todos {
			assert.NotEqual(t, createResp.Todo.Id, todo.Id)
		}
	})

	t.Run("should return error for empty ID", func(t *testing.T) {
		req := &todopb.DeleteTodoRequest{
			Id: "",
		}

		resp, err := service.DeleteTodo(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("should return error for non-existent TODO", func(t *testing.T) {
		req := &todopb.DeleteTodoRequest{
			Id: "non-existent-id",
		}

		resp, err := service.DeleteTodo(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}

func TestTodoServiceServer_CRUDRoundTrip(t *testing.T) {
	store := storage.NewMemoryStore()
	service := NewTodoServiceServer(store)
	ctx := context.Background()

	t.Run("should complete full CRUD cycle", func(t *testing.T) {
		// Create
		createReq := &todopb.CreateTodoRequest{
			Title:       "Round Trip Test",
			Description: "Testing CRUD operations",
		}
		createResp, err := service.CreateTodo(ctx, createReq)
		require.NoError(t, err)
		todoID := createResp.Todo.Id

		// List (verify creation)
		listResp, err := service.ListTodos(ctx, &todopb.ListTodosRequest{})
		require.NoError(t, err)
		found := false
		for _, todo := range listResp.Todos {
			if todo.Id == todoID {
				found = true
				assert.Equal(t, "Round Trip Test", todo.Title)
				break
			}
		}
		assert.True(t, found, "Created TODO should appear in list")

		// Update
		updateReq := &todopb.UpdateTodoRequest{
			Id:          todoID,
			Title:       "Updated Round Trip",
			Description: "Updated description",
			Completed:   true,
		}
		updateResp, err := service.UpdateTodo(ctx, updateReq)
		require.NoError(t, err)
		assert.Equal(t, "Updated Round Trip", updateResp.Todo.Title)
		assert.True(t, updateResp.Todo.Completed)

		// Delete
		deleteResp, err := service.DeleteTodo(ctx, &todopb.DeleteTodoRequest{Id: todoID})
		require.NoError(t, err)
		assert.True(t, deleteResp.Success)

		// List (verify deletion)
		listResp2, err := service.ListTodos(ctx, &todopb.ListTodosRequest{})
		require.NoError(t, err)
		for _, todo := range listResp2.Todos {
			assert.NotEqual(t, todoID, todo.Id, "Deleted TODO should not appear in list")
		}
	})
}
