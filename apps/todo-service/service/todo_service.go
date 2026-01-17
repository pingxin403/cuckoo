package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/pingxin/cuckoo/apps/todo-service/gen/todopb"
	"github.com/pingxin/cuckoo/apps/todo-service/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TodoServiceServer implements the TodoService gRPC service
type TodoServiceServer struct {
	todopb.UnimplementedTodoServiceServer
	store storage.TodoStore
}

// NewTodoServiceServer creates a new TodoServiceServer
func NewTodoServiceServer(store storage.TodoStore) *TodoServiceServer {
	return &TodoServiceServer{
		store: store,
	}
}

// CreateTodo creates a new TODO item
func (s *TodoServiceServer) CreateTodo(ctx context.Context, req *todopb.CreateTodoRequest) (*todopb.CreateTodoResponse, error) {
	// Validate input
	if req.Title == "" || strings.TrimSpace(req.Title) == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required and cannot be empty")
	}

	// Generate unique ID
	id := uuid.New().String()

	// Create TODO
	now := timestamppb.Now()
	todo := &todopb.Todo{
		Id:          id,
		Title:       req.Title,
		Description: req.Description,
		Completed:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Save to store
	if err := s.store.Create(todo); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create todo: %v", err)
	}

	return &todopb.CreateTodoResponse{
		Todo: todo,
	}, nil
}

// ListTodos returns all TODO items
func (s *TodoServiceServer) ListTodos(ctx context.Context, req *todopb.ListTodosRequest) (*todopb.ListTodosResponse, error) {
	todos, err := s.store.List()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list todos: %v", err)
	}

	return &todopb.ListTodosResponse{
		Todos: todos,
	}, nil
}

// UpdateTodo updates an existing TODO item
func (s *TodoServiceServer) UpdateTodo(ctx context.Context, req *todopb.UpdateTodoRequest) (*todopb.UpdateTodoResponse, error) {
	// Validate input
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if req.Title == "" || strings.TrimSpace(req.Title) == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required and cannot be empty")
	}

	// Get existing TODO
	existingTodo, err := s.store.Get(req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "todo not found")
	}

	// Update fields
	existingTodo.Title = req.Title
	existingTodo.Description = req.Description
	existingTodo.Completed = req.Completed
	existingTodo.UpdatedAt = timestamppb.Now()

	// Save updated TODO
	if err := s.store.Update(existingTodo); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update todo: %v", err)
	}

	return &todopb.UpdateTodoResponse{
		Todo: existingTodo,
	}, nil
}

// DeleteTodo deletes a TODO item
func (s *TodoServiceServer) DeleteTodo(ctx context.Context, req *todopb.DeleteTodoRequest) (*todopb.DeleteTodoResponse, error) {
	// Validate input
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Delete TODO
	if err := s.store.Delete(req.Id); err != nil {
		return nil, status.Error(codes.NotFound, "todo not found")
	}

	return &todopb.DeleteTodoResponse{
		Success: true,
	}, nil
}
