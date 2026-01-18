package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"{{MODULE_PATH}}/gen/templatepb"
	"{{MODULE_PATH}}/storage"
)

// TestTemplateService_Create tests the Create method
// This is a template test. Replace with actual service tests.
//
// Test Coverage Requirements:
// - Overall: 80% minimum
// - Service/storage packages: 90% minimum
//
// Run tests with coverage:
//
//	go test -v -race -coverprofile=coverage.out ./...
//	go tool cover -html=coverage.out
//
// Verify coverage thresholds:
//
//	./scripts/test-coverage.sh
func TestTemplateService_Create(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewTemplateService(store)

	req := &templatepb.CreateRequest{
		Field: "test-value",
	}

	// Act
	resp, err := service.Create(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Id)
	assert.Equal(t, "test-value", resp.Field)
}

func TestTemplateService_Create_EmptyField(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewTemplateService(store)

	req := &templatepb.CreateRequest{
		Field: "",
	}

	// Act
	resp, err := service.Create(context.Background(), req)

	// Assert
	// Depending on your validation logic:
	// Option 1: Allow empty fields
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Id)

	// Option 2: Reject empty fields
	// require.Error(t, err)
	// assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestTemplateService_Get(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewTemplateService(store)

	// Create an item first
	createReq := &templatepb.CreateRequest{Field: "test"}
	createResp, err := service.Create(context.Background(), createReq)
	require.NoError(t, err)

	// Act
	getReq := &templatepb.GetRequest{Id: createResp.Id}
	resp, err := service.Get(context.Background(), getReq)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, createResp.Id, resp.Id)
	assert.Equal(t, "test", resp.Field)
}

func TestTemplateService_Get_NotFound(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewTemplateService(store)

	req := &templatepb.GetRequest{Id: "non-existent-id"}

	// Act
	_, err := service.Get(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestTemplateService_List(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewTemplateService(store)

	// Create multiple items
	for i := 0; i < 3; i++ {
		req := &templatepb.CreateRequest{Field: "test"}
		_, err := service.Create(context.Background(), req)
		require.NoError(t, err)
	}

	// Act
	resp, err := service.List(context.Background(), &templatepb.ListRequest{})

	// Assert
	require.NoError(t, err)
	assert.Len(t, resp.Items, 3)
}

func TestTemplateService_Update(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewTemplateService(store)

	// Create an item
	createReq := &templatepb.CreateRequest{Field: "original"}
	createResp, err := service.Create(context.Background(), createReq)
	require.NoError(t, err)

	// Act
	updateReq := &templatepb.UpdateRequest{
		Id:    createResp.Id,
		Field: "updated",
	}
	resp, err := service.Update(context.Background(), updateReq)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, createResp.Id, resp.Id)
	assert.Equal(t, "updated", resp.Field)

	// Verify the update persisted
	getResp, err := service.Get(context.Background(), &templatepb.GetRequest{Id: createResp.Id})
	require.NoError(t, err)
	assert.Equal(t, "updated", getResp.Field)
}

func TestTemplateService_Update_NotFound(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewTemplateService(store)

	req := &templatepb.UpdateRequest{
		Id:    "non-existent-id",
		Field: "updated",
	}

	// Act
	_, err := service.Update(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestTemplateService_Delete(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewTemplateService(store)

	// Create an item
	createReq := &templatepb.CreateRequest{Field: "test"}
	createResp, err := service.Create(context.Background(), createReq)
	require.NoError(t, err)

	// Act
	deleteReq := &templatepb.DeleteRequest{Id: createResp.Id}
	_, err = service.Delete(context.Background(), deleteReq)

	// Assert
	require.NoError(t, err)

	// Verify deletion
	_, err = service.Get(context.Background(), &templatepb.GetRequest{Id: createResp.Id})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestTemplateService_Delete_NotFound(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewTemplateService(store)

	req := &templatepb.DeleteRequest{Id: "non-existent-id"}

	// Act
	_, err := service.Delete(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// TestTemplateService_CRUDCycle tests a complete CRUD cycle
func TestTemplateService_CRUDCycle(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewTemplateService(store)
	ctx := context.Background()

	// Create
	createResp, err := service.Create(ctx, &templatepb.CreateRequest{Field: "initial"})
	require.NoError(t, err)
	id := createResp.Id

	// Read
	getResp, err := service.Get(ctx, &templatepb.GetRequest{Id: id})
	require.NoError(t, err)
	assert.Equal(t, "initial", getResp.Field)

	// Update
	updateResp, err := service.Update(ctx, &templatepb.UpdateRequest{Id: id, Field: "modified"})
	require.NoError(t, err)
	assert.Equal(t, "modified", updateResp.Field)

	// List
	listResp, err := service.List(ctx, &templatepb.ListRequest{})
	require.NoError(t, err)
	assert.Len(t, listResp.Items, 1)

	// Delete
	_, err = service.Delete(ctx, &templatepb.DeleteRequest{Id: id})
	require.NoError(t, err)

	// Verify deletion
	_, err = service.Get(ctx, &templatepb.GetRequest{Id: id})
	require.Error(t, err)
}

// Add more test cases specific to your service:
// - Test concurrent operations
// - Test validation logic
// - Test error handling
// - Test business rules
// - Test integration with external services
