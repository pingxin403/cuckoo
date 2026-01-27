//go:build property
// +build property

package service

import (
	"context"
	"testing"

	"github.com/pingxin403/cuckoo/api/gen/go/{{PROTO_PACKAGE}}"
	"github.com/pingxin403/cuckoo/libs/observability"
	"{{MODULE_PATH}}/storage"
	"pgregory.net/rapid"
)

// Helper function to create a test observability instance for property tests
func createPropertyTestObservability() observability.Observability {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "{{SERVICE_NAME}}-property-test",
		EnableMetrics: false,
		LogLevel:      "error",
	})
	return obs
}

// TestProperty_ServiceMethodIdempotent verifies that calling the same method
// with the same input always produces the same result.
// **Validates: Requirements 1.3**
func TestProperty_ServiceMethodIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random input
		input := rapid.String().Draw(t, "input")

		// Create service with store and observability
		store := storage.NewMemoryStore()
		obs := createPropertyTestObservability()
		service := New{{ServiceName}}ServiceServer(store, obs)
		ctx := context.Background()

		// Call method twice with same input
		// result1, err1 := service.YourMethod(ctx, &{{PROTO_PACKAGE}}.YourRequest{Field: input})
		// result2, err2 := service.YourMethod(ctx, &{{PROTO_PACKAGE}}.YourRequest{Field: input})

		// Verify idempotence
		// if err1 != nil || err2 != nil {
		// 	if err1 == nil || err2 == nil || err1.Error() != err2.Error() {
		// 		t.Fatalf("inconsistent errors: %v vs %v", err1, err2)
		// 	}
		// 	return
		// }
		//
		// if result1.Result != result2.Result {
		// 	t.Fatalf("not idempotent: %v != %v", result1.Result, result2.Result)
		// }

		// TODO: Implement your property test
		_ = service
		_ = ctx
		_ = input
	})
}

// TestProperty_ServiceMethodNeverPanics verifies that the service method
// never panics regardless of input.
// **Validates: Requirements 1.3**
func TestProperty_ServiceMethodNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random input
		input := rapid.String().Draw(t, "input")

		// Create service with store and observability
		store := storage.NewMemoryStore()
		obs := createPropertyTestObservability()
		service := New{{ServiceName}}ServiceServer(store, obs)
		ctx := context.Background()

		// Verify no panic
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("method panicked with input %q: %v", input, r)
			}
		}()

		// Call method
		// _, _ = service.YourMethod(ctx, &{{PROTO_PACKAGE}}.YourRequest{Field: input})

		// TODO: Implement your property test
		_ = service
		_ = ctx
		_ = input
	})
}

// TestProperty_ServiceMethodValidatesInput verifies that invalid inputs
// are properly rejected with appropriate errors.
// **Validates: Requirements 1.3**
func TestProperty_ServiceMethodValidatesInput(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random input that should be invalid
		// For example, empty strings, very long strings, special characters, etc.
		invalidInput := rapid.StringMatching("[^a-zA-Z0-9]*").Draw(t, "invalidInput")

		// Create service with store and observability
		store := storage.NewMemoryStore()
		obs := createPropertyTestObservability()
		service := New{{ServiceName}}ServiceServer(store, obs)
		ctx := context.Background()

		// Call method with invalid input
		// _, err := service.YourMethod(ctx, &{{PROTO_PACKAGE}}.YourRequest{Field: invalidInput})

		// Verify error is returned for invalid input
		// if err == nil && invalidInput != "" {
		// 	t.Fatalf("expected error for invalid input %q, got nil", invalidInput)
		// }

		// TODO: Implement your property test
		_ = service
		_ = ctx
		_ = invalidInput
	})
}

// Custom generator example: Generate valid IDs
func genValidID() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		return rapid.StringMatching("[a-z]{3}[0-9]{6}").Draw(t, "id")
	})
}

// TestProperty_WithCustomGenerator shows how to use custom generators
// **Validates: Requirements 1.3**
func TestProperty_WithCustomGenerator(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Use custom generator
		id := genValidID().Draw(t, "id")

		// Create service with store and observability
		store := storage.NewMemoryStore()
		obs := createPropertyTestObservability()
		service := New{{ServiceName}}ServiceServer(store, obs)
		ctx := context.Background()

		// Test with generated ID
		// result, err := service.GetByID(ctx, &{{PROTO_PACKAGE}}.GetRequest{Id: id})
		// if err != nil {
		// 	t.Fatalf("unexpected error for valid ID %q: %v", id, err)
		// }

		// TODO: Implement your property test
		_ = service
		_ = ctx
		_ = id
	})
}

// Ensure unused imports are referenced (remove these lines when implementing actual methods)
var (
	_ {{PROTO_PACKAGE}}.{{ServiceName}}ServiceServer
)
