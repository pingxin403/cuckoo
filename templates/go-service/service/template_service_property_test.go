//go:build property
// +build property

package service

import (
	"context"
	"testing"

	"pgregory.net/rapid"
)

// TestProperty_ServiceMethodIdempotent verifies that calling the same method
// with the same input always produces the same result.
func TestProperty_ServiceMethodIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random input
		input := rapid.String().Draw(t, "input")

		// Create service
		service := NewTemplateServiceServer(nil)
		ctx := context.Background()

		// Call method twice with same input
		// result1, err1 := service.YourMethod(ctx, &templatepb.YourRequest{Field: input})
		// result2, err2 := service.YourMethod(ctx, &templatepb.YourRequest{Field: input})

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
func TestProperty_ServiceMethodNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random input
		input := rapid.String().Draw(t, "input")

		// Create service
		service := NewTemplateServiceServer(nil)
		ctx := context.Background()

		// Verify no panic
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("method panicked with input %q: %v", input, r)
			}
		}()

		// Call method
		// _, _ = service.YourMethod(ctx, &templatepb.YourRequest{Field: input})

		// TODO: Implement your property test
		_ = service
		_ = ctx
		_ = input
	})
}

// TestProperty_ServiceMethodValidatesInput verifies that invalid inputs
// are properly rejected with appropriate errors.
func TestProperty_ServiceMethodValidatesInput(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random input that should be invalid
		// For example, empty strings, very long strings, special characters, etc.
		invalidInput := rapid.StringMatching("[^a-zA-Z0-9]*").Draw(t, "invalidInput")

		// Create service
		service := NewTemplateServiceServer(nil)
		ctx := context.Background()

		// Call method with invalid input
		// _, err := service.YourMethod(ctx, &templatepb.YourRequest{Field: invalidInput})

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
		prefix := rapid.SampledFrom([]string{"id", "key", "ref"}).Draw(t, "prefix")
		number := rapid.IntRange(1, 999999).Draw(t, "number")
		return rapid.StringMatching("[a-z]{3}[0-9]{6}").Example(0)
	})
}

// TestProperty_WithCustomGenerator shows how to use custom generators
func TestProperty_WithCustomGenerator(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Use custom generator
		id := genValidID().Draw(t, "id")

		// Create service
		service := NewTemplateServiceServer(nil)
		ctx := context.Background()

		// Test with generated ID
		// result, err := service.GetByID(ctx, &templatepb.GetRequest{Id: id})
		// if err != nil {
		// 	t.Fatalf("unexpected error for valid ID %q: %v", id, err)
		// }

		// TODO: Implement your property test
		_ = service
		_ = ctx
		_ = id
	})
}
