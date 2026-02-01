package sync

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/cuckoo-org/cuckoo/libs/hlc"
)

func TestNewConflictResolver(t *testing.T) {
	config := DefaultConflictResolverConfig("region-a")
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	resolver := NewConflictResolver(config, logger)

	if resolver == nil {
		t.Fatal("Expected non-nil conflict resolver")
	}

	if resolver.regionID != "region-a" {
		t.Errorf("Expected region ID 'region-a', got '%s'", resolver.regionID)
	}

	if resolver.logger == nil {
		t.Error("Expected non-nil logger")
	}
}

func TestResolveConflict_NoConflict(t *testing.T) {
	resolver := NewConflictResolver(DefaultConflictResolverConfig("region-a"), nil)
	ctx := context.Background()

	// Create identical message versions
	globalID := hlc.GlobalID{
		RegionID: "region-a",
		HLC:      "1000-0",
		Sequence: 1,
	}

	localVersion := MessageVersion{
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
}

func TestResolveConflict_LocalWins(t *testing.T) {
	resolver := NewConflictResolver(DefaultConflictResolverConfig("region-a"), nil)
	ctx := context.Background()

	// Local version has later timestamp
	localGlobalID := hlc.GlobalID{
		RegionID: "region-a",
		HLC:      "2000-0", // Later timestamp
		Sequence: 1,
	}

	remoteGlobalID := hlc.GlobalID{
		RegionID: "region-b",
		HLC:      "1000-0", // Earlier timestamp
		Sequence: 1,
	}

	localVersion := MessageVersion{
		GlobalID:  localGlobalID,
		MessageID: "msg-1",
		Content:   "Local Content",
		Timestamp: 2000,
		RegionID:  "region-a",
		Version:   2,
	}

	remoteVersion := MessageVersion{
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
}

func TestResolveConflict_RemoteWins(t *testing.T) {
	resolver := NewConflictResolver(DefaultConflictResolverConfig("region-a"), nil)
	ctx := context.Background()

	// Remote version has later timestamp
	localGlobalID := hlc.GlobalID{
		RegionID: "region-a",
		HLC:      "1000-0", // Earlier timestamp
		Sequence: 1,
	}

	remoteGlobalID := hlc.GlobalID{
		RegionID: "region-b",
		HLC:      "2000-0", // Later timestamp
		Sequence: 1,
	}

	localVersion := MessageVersion{
		GlobalID:  localGlobalID,
		MessageID: "msg-1",
		Content:   "Local Content",
		Timestamp: 1000,
		RegionID:  "region-a",
		Version:   1,
	}

	remoteVersion := MessageVersion{
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
}

func TestGetMetrics(t *testing.T) {
	resolver := NewConflictResolver(DefaultConflictResolverConfig("region-a"), nil)
	ctx := context.Background()

	// Initially, metrics should be zero
	metrics := resolver.GetMetrics()
	if metrics.TotalConflicts != 0 {
		t.Errorf("Expected 0 total conflicts, got %d", metrics.TotalConflicts)
	}

	// Create some conflicts
	localGlobalID := hlc.GlobalID{RegionID: "region-a", HLC: "1000-0", Sequence: 1}
	remoteGlobalID := hlc.GlobalID{RegionID: "region-b", HLC: "2000-0", Sequence: 1}

	localVersion := MessageVersion{
		GlobalID:  localGlobalID,
		MessageID: "msg-1",
		RegionID:  "region-a",
	}

	remoteVersion := MessageVersion{
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
	if metrics.TotalConflicts != 1 {
		t.Errorf("Expected 1 total conflict, got %d", metrics.TotalConflicts)
	}

	if metrics.RemoteWins != 1 {
		t.Errorf("Expected 1 remote win, got %d", metrics.RemoteWins)
	}

	if metrics.LocalWins != 0 {
		t.Errorf("Expected 0 local wins, got %d", metrics.LocalWins)
	}
}

func TestValidateConflictResolution(t *testing.T) {
	resolver := NewConflictResolver(DefaultConflictResolverConfig("region-a"), nil)

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

	// Test valid resolution
	resolution = &ConflictResolution{
		MessageID: "msg-1",
		LocalVersion: MessageVersion{
			MessageID: "msg-1",
			RegionID:  "region-a",
		},
		RemoteVersion: MessageVersion{
			MessageID: "msg-1",
			RegionID:  "region-b",
		},
		Winner: MessageVersion{
			MessageID: "msg-1",
			RegionID:  "region-a",
		},
		Resolution: "local_wins",
	}
	err = resolver.ValidateConflictResolution(resolution)
	if err != nil {
		t.Errorf("Unexpected error for valid resolution: %v", err)
	}
}

func TestGetConflictResolutionStrategy(t *testing.T) {
	resolver := NewConflictResolver(DefaultConflictResolverConfig("region-a"), nil)

	strategy := resolver.GetConflictResolutionStrategy()
	if strategy != "LWW" {
		t.Errorf("Expected 'LWW' strategy, got '%s'", strategy)
	}
}
