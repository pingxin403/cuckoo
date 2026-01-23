package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pingxin403/cuckoo/apps/im-service/gen/im_servicepb"
	"github.com/pingxin403/cuckoo/apps/im-service/storage"
)

// TestUimUserviceService_Create tests the Create method
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
func TestUimUserviceService_Create(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewUimUserviceService(store)

	req := &im_servicepb.CreateRequest{
		Field: "test-value",
	}

	// Act
	resp, err := service.Create(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Id)
	assert.Equal(t, "test-value", resp.Field)
}

func TestUimUserviceService_Create_EmptyField(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewUimUserviceService(store)

	req := &im_servicepb.CreateRequest{
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

func TestUimUserviceService_Get(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewUimUserviceService(store)

	// Create an item first
	createReq := &im_servicepb.CreateRequest{Field: "test"}
	createResp, err := service.Create(context.Background(), createReq)
	require.NoError(t, err)

	// Act
	getReq := &im_servicepb.GetRequest{Id: createResp.Id}
	resp, err := service.Get(context.Background(), getReq)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, createResp.Id, resp.Id)
	assert.Equal(t, "test", resp.Field)
}

func TestUimUserviceService_Get_NotFound(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewUimUserviceService(store)

	req := &im_servicepb.GetRequest{Id: "non-existent-id"}

	// Act
	_, err := service.Get(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUimUserviceService_List(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewUimUserviceService(store)

	// Create multiple items
	for i := 0; i < 3; i++ {
		req := &im_servicepb.CreateRequest{Field: "test"}
		_, err := service.Create(context.Background(), req)
		require.NoError(t, err)
	}

	// Act
	resp, err := service.List(context.Background(), &im_servicepb.ListRequest{})

	// Assert
	require.NoError(t, err)
	assert.Len(t, resp.Items, 3)
}

func TestUimUserviceService_Update(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewUimUserviceService(store)

	// Create an item
	createReq := &im_servicepb.CreateRequest{Field: "original"}
	createResp, err := service.Create(context.Background(), createReq)
	require.NoError(t, err)

	// Act
	updateReq := &im_servicepb.UpdateRequest{
		Id:    createResp.Id,
		Field: "updated",
	}
	resp, err := service.Update(context.Background(), updateReq)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, createResp.Id, resp.Id)
	assert.Equal(t, "updated", resp.Field)

	// Verify the update persisted
	getResp, err := service.Get(context.Background(), &im_servicepb.GetRequest{Id: createResp.Id})
	require.NoError(t, err)
	assert.Equal(t, "updated", getResp.Field)
}

func TestUimUserviceService_Update_NotFound(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewUimUserviceService(store)

	req := &im_servicepb.UpdateRequest{
		Id:    "non-existent-id",
		Field: "updated",
	}

	// Act
	_, err := service.Update(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUimUserviceService_Delete(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewUimUserviceService(store)

	// Create an item
	createReq := &im_servicepb.CreateRequest{Field: "test"}
	createResp, err := service.Create(context.Background(), createReq)
	require.NoError(t, err)

	// Act
	deleteReq := &im_servicepb.DeleteRequest{Id: createResp.Id}
	_, err = service.Delete(context.Background(), deleteReq)

	// Assert
	require.NoError(t, err)

	// Verify deletion
	_, err = service.Get(context.Background(), &im_servicepb.GetRequest{Id: createResp.Id})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUimUserviceService_Delete_NotFound(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewUimUserviceService(store)

	req := &im_servicepb.DeleteRequest{Id: "non-existent-id"}

	// Act
	_, err := service.Delete(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// TestUimUserviceService_CRUDCycle tests a complete CRUD cycle
func TestUimUserviceService_CRUDCycle(t *testing.T) {
	// Arrange
	store := storage.NewMemoryStore()
	service := NewUimUserviceService(store)
	ctx := context.Background()

	// Create
	createResp, err := service.Create(ctx, &im_servicepb.CreateRequest{Field: "initial"})
	require.NoError(t, err)
	id := createResp.Id

	// Read
	getResp, err := service.Get(ctx, &im_servicepb.GetRequest{Id: id})
	require.NoError(t, err)
	assert.Equal(t, "initial", getResp.Field)

	// Update
	updateResp, err := service.Update(ctx, &im_servicepb.UpdateRequest{Id: id, Field: "modified"})
	require.NoError(t, err)
	assert.Equal(t, "modified", updateResp.Field)

	// List
	listResp, err := service.List(ctx, &im_servicepb.ListRequest{})
	require.NoError(t, err)
	assert.Len(t, listResp.Items, 1)

	// Delete
	_, err = service.Delete(ctx, &im_servicepb.DeleteRequest{Id: id})
	require.NoError(t, err)

	// Verify deletion
	_, err = service.Get(ctx, &im_servicepb.GetRequest{Id: id})
	require.Error(t, err)
}

// Add more test cases specific to your service:
// - Test concurrent operations
// - Test validation logic
// - Test error handling
// - Test business rules
// - Test integration with external services
