package sync

import (
	"context"
	"testing"
)

func TestStandaloneConflictResolver_Creation(t *testing.T) {
	resolver := NewStandaloneConflictResolver("region-a", true, nil)

	if resolver == nil {
		t.Fatal("Expected non-nil conflict resolver")
	}

	if resolver.regionID != "region-a" {
		t.Errorf("Expected region ID 'region-a', got '%s'", resolver.regionID)
	}

	if !resolver.enableDetailedLogging {
		t.Error("Expected detailed logging to be enabled")
	}
}

func TestStandaloneConflictResolver_NoConflict(t *testing.T) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)
	ctx := context.Background()

	// Create identical message versions
	globalID := StandaloneGlobalID{
		RegionID: "region-a",
		HLC:      "1000-0",
		Sequence: 1,
	}

	localVersion := StandaloneMessageVersion{
		GlobalID:  globalID,
		MessageID: "msg-1",
		Content:   "Hello World",
		Timestamp: 1000,
		RegionID:  "region-a",
		Version:   1,
	}

	remoteVersion := localVersion // Identical

	resolution, err := resolver.ResolveConflict(ctx, localVersion, remoteVersion)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resolution.Resolution != "no_conflict" {
		t.Errorf("Expected 'no_conflict', got '%s'", resolution.Resolution)
	}

	if resolution.Winner.MessageID != localVersion.MessageID {
		t.Error("Expected winner to be local version for no conflict")
	}

	if resolution.ResolutionReason != "identical global IDs" {
		t.Errorf("Expected specific reason, got '%s'", resolution.ResolutionReason)
	}
}

func TestStandaloneConflictResolver_LocalWins(t *testing.T) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)
	ctx := context.Background()

	// Local version has later timestamp
	localGlobalID := StandaloneGlobalID{
		RegionID: "region-a",
		HLC:      "2000-0", // Later timestamp
		Sequence: 1,
	}

	remoteGlobalID := StandaloneGlobalID{
		RegionID: "region-b",
		HLC:      "1000-0", // Earlier timestamp
		Sequence: 1,
	}

	localVersion := StandaloneMessageVersion{
		GlobalID:  localGlobalID,
		MessageID: "msg-1",
		Content:   "Local Content",
		Timestamp: 2000,
		RegionID:  "region-a",
		Version:   2,
	}

	remoteVersion := StandaloneMessageVersion{
		GlobalID:  remoteGlobalID,
		MessageID: "msg-1",
		Content:   "Remote Content",
		Timestamp: 1000,
		RegionID:  "region-b",
		Version:   1,
	}

	resolution, err := resolver.ResolveConflict(ctx, localVersion, remoteVersion)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resolution.Resolution != "local_wins" {
		t.Errorf("Expected 'local_wins', got '%s'", resolution.Resolution)
	}

	if resolution.Winner.RegionID != "region-a" {
		t.Error("Expected winner to be local version")
	}

	if resolution.Winner.Content != "Local Content" {
		t.Error("Expected winner content to match local version")
	}

	if resolution.ResolutionReason != "local version has later HLC timestamp" {
		t.Errorf("Expected specific reason, got '%s'", resolution.ResolutionReason)
	}
}

func TestStandaloneConflictResolver_RemoteWins(t *testing.T) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)
	ctx := context.Background()

	// Remote version has later timestamp
	localGlobalID := StandaloneGlobalID{
		RegionID: "region-a",
		HLC:      "1000-0", // Earlier timestamp
		Sequence: 1,
	}

	remoteGlobalID := StandaloneGlobalID{
		RegionID: "region-b",
		HLC:      "2000-0", // Later timestamp
		Sequence: 1,
	}

	localVersion := StandaloneMessageVersion{
		GlobalID:  localGlobalID,
		MessageID: "msg-1",
		Content:   "Local Content",
		Timestamp: 1000,
		RegionID:  "region-a",
		Version:   1,
	}

	remoteVersion := StandaloneMessageVersion{
		GlobalID:  remoteGlobalID,
		MessageID: "msg-1",
		Content:   "Remote Content",
		Timestamp: 2000,
		RegionID:  "region-b",
		Version:   2,
	}

	resolution, err := resolver.ResolveConflict(ctx, localVersion, remoteVersion)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resolution.Resolution != "remote_wins" {
		t.Errorf("Expected 'remote_wins', got '%s'", resolution.Resolution)
	}

	if resolution.Winner.RegionID != "region-b" {
		t.Error("Expected winner to be remote version")
	}

	if resolution.Winner.Content != "Remote Content" {
		t.Error("Expected winner content to match remote version")
	}

	if resolution.ResolutionReason != "remote version has later HLC timestamp" {
		t.Errorf("Expected specific reason, got '%s'", resolution.ResolutionReason)
	}
}

func TestStandaloneConflictResolver_RegionTiebreaker(t *testing.T) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)
	ctx := context.Background()

	// Same HLC timestamp, different regions - should use region ID as tiebreaker
	localGlobalID := StandaloneGlobalID{
		RegionID: "region-b", // Lexicographically later
		HLC:      "1000-0",
		Sequence: 1,
	}

	remoteGlobalID := StandaloneGlobalID{
		RegionID: "region-a", // Lexicographically earlier
		HLC:      "1000-0",   // Same timestamp
		Sequence: 1,
	}

	localVersion := StandaloneMessageVersion{
		GlobalID:  localGlobalID,
		MessageID: "msg-1",
		Content:   "Local Content",
		RegionID:  "region-b",
	}

	remoteVersion := StandaloneMessageVersion{
		GlobalID:  remoteGlobalID,
		MessageID: "msg-1",
		Content:   "Remote Content",
		RegionID:  "region-a",
	}

	resolution, err := resolver.ResolveConflict(ctx, localVersion, remoteVersion)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Local should win because region-b > region-a lexicographically
	if resolution.Resolution != "local_wins" {
		t.Errorf("Expected 'local_wins' due to region tiebreaker, got '%s'", resolution.Resolution)
	}
}

func TestStandaloneConflictResolver_SequenceTiebreaker(t *testing.T) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)
	ctx := context.Background()

	// Same HLC timestamp and region, different sequences
	localGlobalID := StandaloneGlobalID{
		RegionID: "region-a",
		HLC:      "1000-0",
		Sequence: 2, // Higher sequence
	}

	remoteGlobalID := StandaloneGlobalID{
		RegionID: "region-a", // Same region
		HLC:      "1000-0",   // Same timestamp
		Sequence: 1,          // Lower sequence
	}

	localVersion := StandaloneMessageVersion{
		GlobalID:  localGlobalID,
		MessageID: "msg-1",
		Content:   "Local Content",
		RegionID:  "region-a",
	}

	remoteVersion := StandaloneMessageVersion{
		GlobalID:  remoteGlobalID,
		MessageID: "msg-1",
		Content:   "Remote Content",
		RegionID:  "region-a",
	}

	resolution, err := resolver.ResolveConflict(ctx, localVersion, remoteVersion)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Local should win because sequence 2 > sequence 1
	if resolution.Resolution != "local_wins" {
		t.Errorf("Expected 'local_wins' due to sequence tiebreaker, got '%s'", resolution.Resolution)
	}
}

func TestStandaloneConflictResolver_Metrics(t *testing.T) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)
	ctx := context.Background()

	// Initially, metrics should be zero
	metrics := resolver.GetMetrics()
	if metrics["total_conflicts"].(int64) != 0 {
		t.Errorf("Expected 0 total conflicts, got %d", metrics["total_conflicts"])
	}

	// Create some conflicts
	localGlobalID := StandaloneGlobalID{RegionID: "region-a", HLC: "1000-0", Sequence: 1}
	remoteGlobalID := StandaloneGlobalID{RegionID: "region-b", HLC: "2000-0", Sequence: 1}

	localVersion := StandaloneMessageVersion{
		GlobalID:  localGlobalID,
		MessageID: "msg-1",
		RegionID:  "region-a",
	}

	remoteVersion := StandaloneMessageVersion{
		GlobalID:  remoteGlobalID,
		MessageID: "msg-1",
		RegionID:  "region-b",
	}

	// Resolve a conflict (remote should win)
	_, err := resolver.ResolveConflict(ctx, localVersion, remoteVersion)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check metrics
	metrics = resolver.GetMetrics()
	if metrics["total_conflicts"].(int64) != 1 {
		t.Errorf("Expected 1 total conflict, got %d", metrics["total_conflicts"])
	}

	if metrics["remote_wins"].(int64) != 1 {
		t.Errorf("Expected 1 remote win, got %d", metrics["remote_wins"])
	}

	if metrics["local_wins"].(int64) != 0 {
		t.Errorf("Expected 0 local wins, got %d", metrics["local_wins"])
	}

	if metrics["lww_resolutions"].(int64) != 1 {
		t.Errorf("Expected 1 LWW resolution, got %d", metrics["lww_resolutions"])
	}
}

func TestStandaloneConflictResolver_ResetMetrics(t *testing.T) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)
	ctx := context.Background()

	// Create a conflict to increment metrics
	localGlobalID := StandaloneGlobalID{RegionID: "region-a", HLC: "1000-0", Sequence: 1}
	remoteGlobalID := StandaloneGlobalID{RegionID: "region-b", HLC: "2000-0", Sequence: 1}

	localVersion := StandaloneMessageVersion{GlobalID: localGlobalID, MessageID: "msg-1"}
	remoteVersion := StandaloneMessageVersion{GlobalID: remoteGlobalID, MessageID: "msg-1"}

	_, err := resolver.ResolveConflict(ctx, localVersion, remoteVersion)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify metrics are non-zero
	metrics := resolver.GetMetrics()
	if metrics["total_conflicts"].(int64) == 0 {
		t.Error("Expected non-zero conflicts before reset")
	}

	// Reset metrics
	resolver.ResetMetrics()

	// Verify metrics are zero
	metrics = resolver.GetMetrics()
	if metrics["total_conflicts"].(int64) != 0 {
		t.Errorf("Expected 0 conflicts after reset, got %d", metrics["total_conflicts"])
	}
}

func TestStandaloneConflictResolver_ValidateResolution(t *testing.T) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)

	// Test nil resolution
	err := resolver.ValidateConflictResolution(nil)
	if err == nil {
		t.Error("Expected error for nil resolution")
	}

	// Test empty message ID
	resolution := &StandaloneConflictResolution{
		MessageID: "",
	}
	err = resolver.ValidateConflictResolution(resolution)
	if err == nil {
		t.Error("Expected error for empty message ID")
	}

	// Test mismatched message IDs
	resolution = &StandaloneConflictResolution{
		MessageID: "msg-1",
		LocalVersion: StandaloneMessageVersion{
			MessageID: "msg-1",
		},
		RemoteVersion: StandaloneMessageVersion{
			MessageID: "msg-2", // Different ID
		},
	}
	err = resolver.ValidateConflictResolution(resolution)
	if err == nil {
		t.Error("Expected error for mismatched message IDs")
	}

	// Test invalid resolution type
	resolution = &StandaloneConflictResolution{
		MessageID: "msg-1",
		LocalVersion: StandaloneMessageVersion{
			MessageID: "msg-1",
		},
		RemoteVersion: StandaloneMessageVersion{
			MessageID: "msg-1",
		},
		Resolution: "invalid_resolution",
	}
	err = resolver.ValidateConflictResolution(resolution)
	if err == nil {
		t.Error("Expected error for invalid resolution type")
	}

	// Test valid resolution
	resolution = &StandaloneConflictResolution{
		MessageID: "msg-1",
		LocalVersion: StandaloneMessageVersion{
			MessageID: "msg-1",
			RegionID:  "region-a",
		},
		RemoteVersion: StandaloneMessageVersion{
			MessageID: "msg-1",
			RegionID:  "region-b",
		},
		Resolution: "local_wins",
	}
	err = resolver.ValidateConflictResolution(resolution)
	if err != nil {
		t.Errorf("Unexpected error for valid resolution: %v", err)
	}
}

func TestStandaloneConflictResolver_Strategy(t *testing.T) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)

	strategy := resolver.GetConflictResolutionStrategy()
	if strategy != "LWW" {
		t.Errorf("Expected 'LWW' strategy, got '%s'", strategy)
	}
}

func TestStandaloneConflictResolver_Deterministic(t *testing.T) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)

	// Test identical versions (non-deterministic in terms of conflict, but no conflict)
	localGlobalID := StandaloneGlobalID{RegionID: "region-a", HLC: "1000-0", Sequence: 1}
	remoteGlobalID := StandaloneGlobalID{RegionID: "region-a", HLC: "1000-0", Sequence: 1}

	localVersion := StandaloneMessageVersion{GlobalID: localGlobalID}
	remoteVersion := StandaloneMessageVersion{GlobalID: remoteGlobalID}

	isDeterministic := resolver.IsConflictResolutionDeterministic(localVersion, remoteVersion)
	if isDeterministic {
		t.Error("Expected non-deterministic for identical versions")
	}

	// Test different versions (deterministic)
	remoteGlobalID.HLC = "2000-0"
	remoteVersion.GlobalID = remoteGlobalID

	isDeterministic = resolver.IsConflictResolutionDeterministic(localVersion, remoteVersion)
	if !isDeterministic {
		t.Error("Expected deterministic for different versions")
	}
}

func TestStandaloneConflictResolver_String(t *testing.T) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)

	str := resolver.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	// Should contain region ID and strategy
	if !contains(str, "region-a") {
		t.Error("Expected string to contain region ID")
	}

	if !contains(str, "LWW") {
		t.Error("Expected string to contain LWW strategy")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkStandaloneResolveConflict(b *testing.B) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)
	ctx := context.Background()

	localVersion := StandaloneMessageVersion{
		GlobalID:  StandaloneGlobalID{RegionID: "region-a", HLC: "1000-0", Sequence: 1},
		MessageID: "msg-1",
	}

	remoteVersion := StandaloneMessageVersion{
		GlobalID:  StandaloneGlobalID{RegionID: "region-b", HLC: "2000-0", Sequence: 1},
		MessageID: "msg-1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.ResolveConflict(ctx, localVersion, remoteVersion)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkStandaloneGetMetrics(b *testing.B) {
	resolver := NewStandaloneConflictResolver("region-a", false, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = resolver.GetMetrics()
	}
}
