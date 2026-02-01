package sync

import (
	"testing"
)

// Simple test that doesn't depend on external packages
func TestConflictResolverBasic(t *testing.T) {
	config := ConflictResolverConfig{
		RegionID:              "region-a",
		EnableDetailedLogging: true,
	}

	resolver := NewConflictResolver(config, nil)

	if resolver == nil {
		t.Fatal("Expected non-nil conflict resolver")
	}

	if resolver.regionID != "region-a" {
		t.Errorf("Expected region ID 'region-a', got '%s'", resolver.regionID)
	}

	strategy := resolver.GetConflictResolutionStrategy()
	if strategy != "LWW" {
		t.Errorf("Expected 'LWW' strategy, got '%s'", strategy)
	}
}

func TestConflictMetrics(t *testing.T) {
	config := ConflictResolverConfig{
		RegionID: "region-a",
	}

	resolver := NewConflictResolver(config, nil)

	// Initially, metrics should be zero
	metrics := resolver.GetMetrics()
	if metrics.TotalConflicts != 0 {
		t.Errorf("Expected 0 total conflicts, got %d", metrics.TotalConflicts)
	}

	if metrics.RegionID != "region-a" {
		t.Errorf("Expected region ID 'region-a', got '%s'", metrics.RegionID)
	}
}

func TestValidateConflictResolutionSimple(t *testing.T) {
	config := ConflictResolverConfig{
		RegionID: "region-a",
	}

	resolver := NewConflictResolver(config, nil)

	// Test nil resolution
	err := resolver.ValidateConflictResolution(nil)
	if err == nil {
		t.Error("Expected error for nil resolution")
	}

	// Test empty message ID
	resolution := &ConflictResolution{
		MessageID: "",
	}
	err = resolver.ValidateConflictResolution(resolution)
	if err == nil {
		t.Error("Expected error for empty message ID")
	}
}

func TestResetMetrics(t *testing.T) {
	config := ConflictResolverConfig{
		RegionID: "region-a",
	}

	resolver := NewConflictResolver(config, nil)

	// Reset metrics should not cause any errors
	resolver.ResetMetrics()

	// Verify metrics are zero after reset
	metrics := resolver.GetMetrics()
	if metrics.TotalConflicts != 0 {
		t.Errorf("Expected 0 conflicts after reset, got %d", metrics.TotalConflicts)
	}
}
