package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pingxin403/cuckoo/apps/todo-service/gen/todopb"
	"github.com/pingxin403/cuckoo/apps/todo-service/storage"
	"github.com/pingxin403/cuckoo/libs/observability"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TodoServiceServer implements the TodoService gRPC service
type TodoServiceServer struct {
	todopb.UnimplementedTodoServiceServer
	store storage.TodoStore
	obs   observability.Observability
}

// NewTodoServiceServer creates a new TodoServiceServer
func NewTodoServiceServer(store storage.TodoStore, obs observability.Observability) *TodoServiceServer {
	return &TodoServiceServer{
		store: store,
		obs:   obs,
	}
}

// CreateTodo creates a new TODO item
func (s *TodoServiceServer) CreateTodo(ctx context.Context, req *todopb.CreateTodoRequest) (*todopb.CreateTodoResponse, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		s.obs.Metrics().RecordHistogram("todo_grpc_request_duration_seconds", duration, map[string]string{"method": "CreateTodo"})
	}()

	// Validate input
	if req.Title == "" || strings.TrimSpace(req.Title) == "" {
		s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "create", "status": "failure"})
		s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "CreateTodo", "status": "invalid_argument"})
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
		s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "create", "status": "failure"})
		s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "CreateTodo", "status": "internal"})
		return nil, status.Errorf(codes.Internal, "failed to create todo: %v", err)
	}

	// Update metrics
	s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "create", "status": "success"})
	s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "CreateTodo", "status": "ok"})

	// Update total items gauge
	todos, _ := s.store.List()
	s.obs.Metrics().SetGauge("todo_items_total", float64(len(todos)), nil)

	return &todopb.CreateTodoResponse{
		Todo: todo,
	}, nil
}

// ListTodos returns all TODO items
func (s *TodoServiceServer) ListTodos(ctx context.Context, req *todopb.ListTodosRequest) (*todopb.ListTodosResponse, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		s.obs.Metrics().RecordHistogram("todo_grpc_request_duration_seconds", duration, map[string]string{"method": "ListTodos"})
	}()

	todos, err := s.store.List()
	if err != nil {
		s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "list", "status": "failure"})
		s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "ListTodos", "status": "internal"})
		return nil, status.Errorf(codes.Internal, "failed to list todos: %v", err)
	}

	s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "list", "status": "success"})
	s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "ListTodos", "status": "ok"})

	// Update total items gauge
	s.obs.Metrics().SetGauge("todo_items_total", float64(len(todos)), nil)

	return &todopb.ListTodosResponse{
		Todos: todos,
	}, nil
}

// UpdateTodo updates an existing TODO item
func (s *TodoServiceServer) UpdateTodo(ctx context.Context, req *todopb.UpdateTodoRequest) (*todopb.UpdateTodoResponse, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		s.obs.Metrics().RecordHistogram("todo_grpc_request_duration_seconds", duration, map[string]string{"method": "UpdateTodo"})
	}()

	// Validate input
	if req.Id == "" {
		s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "update", "status": "failure"})
		s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "UpdateTodo", "status": "invalid_argument"})
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if req.Title == "" || strings.TrimSpace(req.Title) == "" {
		s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "update", "status": "failure"})
		s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "UpdateTodo", "status": "invalid_argument"})
		return nil, status.Error(codes.InvalidArgument, "title is required and cannot be empty")
	}

	// Get existing TODO
	existingTodo, err := s.store.Get(req.Id)
	if err != nil {
		s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "update", "status": "failure"})
		s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "UpdateTodo", "status": "not_found"})
		return nil, status.Error(codes.NotFound, "todo not found")
	}

	// Update fields
	existingTodo.Title = req.Title
	existingTodo.Description = req.Description
	existingTodo.Completed = req.Completed
	existingTodo.UpdatedAt = timestamppb.Now()

	// Save updated TODO
	if err := s.store.Update(existingTodo); err != nil {
		s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "update", "status": "failure"})
		s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "UpdateTodo", "status": "internal"})
		return nil, status.Errorf(codes.Internal, "failed to update todo: %v", err)
	}

	s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "update", "status": "success"})
	s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "UpdateTodo", "status": "ok"})

	return &todopb.UpdateTodoResponse{
		Todo: existingTodo,
	}, nil
}

// DeleteTodo deletes a TODO item
func (s *TodoServiceServer) DeleteTodo(ctx context.Context, req *todopb.DeleteTodoRequest) (*todopb.DeleteTodoResponse, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		s.obs.Metrics().RecordHistogram("todo_grpc_request_duration_seconds", duration, map[string]string{"method": "DeleteTodo"})
	}()

	// Validate input
	if req.Id == "" {
		s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "delete", "status": "failure"})
		s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "DeleteTodo", "status": "invalid_argument"})
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Delete TODO
	if err := s.store.Delete(req.Id); err != nil {
		s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "delete", "status": "failure"})
		s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "DeleteTodo", "status": "not_found"})
		return nil, status.Error(codes.NotFound, "todo not found")
	}

	s.obs.Metrics().IncrementCounter("todo_operations_total", map[string]string{"operation": "delete", "status": "success"})
	s.obs.Metrics().IncrementCounter("todo_grpc_requests_total", map[string]string{"method": "DeleteTodo", "status": "ok"})

	// Update total items gauge
	todos, _ := s.store.List()
	s.obs.Metrics().SetGauge("todo_items_total", float64(len(todos)), nil)

	return &todopb.DeleteTodoResponse{
		Success: true,
	}, nil
}
