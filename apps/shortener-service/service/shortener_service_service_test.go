package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
)

// TestNewShortenerServiceImpl tests service creation
func TestNewShortenerServiceImpl(t *testing.T) {
	// Create a mock storage (we'll use mysql_store in real implementation)
	store, err := storage.NewMySQLStore()
	if err != nil {
		t.Skip("MySQL not available for testing")
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Logf("Error closing store: %v", err)
		}
	}()

	// Create service
	service := NewShortenerServiceImpl(store)

	// Assert
	assert.NotNil(t, service)
	assert.NotNil(t, service.storage)
}

// TODO: Implement service method tests in task 11
// - TestCreateShortLink
// - TestGetLinkInfo
// - TestDeleteShortLink
