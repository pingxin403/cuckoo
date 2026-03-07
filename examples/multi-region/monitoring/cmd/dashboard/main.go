package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cuckoo-org/cuckoo/libs/hlc"
	"github.com/cuckoo-org/cuckoo/examples/mvp/queue"
	"github.com/cuckoo-org/cuckoo/examples/mvp/storage"
	"github.com/cuckoo-org/cuckoo/examples/multi-region/sync"
)

func main() {
	// Create logger
	logger := log.New(os.Stdout, "[Dashboard] ", log.LstdFlags|log.Lshortfile)

	// Initialize HLC
	hlcClock := hlc.NewHLC("region-a", "node-1")

	// Initialize storage
	localStorage, err := storage.NewLocalStore("dashboard_test.db")
	if err != nil {
		log.Fatalf("Failed to create local store: %v", err)
	}
	defer localStorage.Close()

	// Initialize queue
	localQueue := queue.NewLocalQueue()

	// Initialize conflict resolver
	conflictConfig := sync.DefaultConflictResolverConfig("region-a")
	conflictResolver := sync.NewConflictResolver(conflictConfig, logger)

	// Initialize message syncer
	syncConfig := sync.DefaultConfig("region-a")
	messageSyncer, err := sync.NewMessageSyncer("region-a", hlcClock, localQueue, localStorage, syncConfig, logger)
	if err != nil {
		log.Fatalf("Failed to create message syncer: %v", err)
	}

	// Start message syncer
	if err := messageSyncer.Start(); err != nil {
		log.Fatalf("Failed to start message syncer: %v", err)
	}
	defer messageSyncer.Stop()

	// Create and start web dashboard
	dashboard := NewWebDashboard(8090, hlcClock, conflictResolver, messageSyncer)

	// Start dashboard in a goroutine
	go func() {
		logger.Println("Starting web dashboard on http://localhost:8090")
		if err := dashboard.Start(); err != nil {
			logger.Printf("Dashboard server error: %v", err)
		}
	}()

	// Simulate some activity to generate metrics
	go simulateActivity(hlcClock, conflictResolver, messageSyncer, logger)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Println("Dashboard is running. Visit http://localhost:8090 to view metrics")
	logger.Println("Press Ctrl+C to stop...")

	<-sigChan
	logger.Println("Shutting down...")

	// Stop dashboard
	if err := dashboard.Stop(); err != nil {
		logger.Printf("Error stopping dashboard: %v", err)
	}

	logger.Println("Dashboard stopped")
}

// simulateActivity generates some test data to populate the dashboard
func simulateActivity(hlcClock *hlc.HLC, conflictResolver *sync.ConflictResolver, messageSyncer *sync.MessageSyncer, logger *log.Logger) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	messageCount := 0

	for range ticker.C {
		messageCount++

		// Generate some HLC IDs
		globalID := hlcClock.GenerateID()
		logger.Printf("Generated GlobalID: %+v", globalID)

		// Simulate some conflicts occasionally
		if messageCount%5 == 0 {
			// Create two versions of the same message to simulate conflict
			localVersion := sync.MessageVersion{
				GlobalID:       globalID,
				MessageID:      "msg-123",
				Content:        "Local version",
				RegionID:       "region-a",
				Version:        1,
				SequenceNumber: int64(messageCount),
				CreatedAt:      time.Now(),
			}

			// Remote version with slightly different timestamp
			remoteGlobalID := hlc.GlobalID{
				RegionID: "region-b",
				HLC:      globalID.HLC,
				Sequence: globalID.Sequence + 1,
			}

			remoteVersion := sync.MessageVersion{
				GlobalID:       remoteGlobalID,
				MessageID:      "msg-123",
				Content:        "Remote version",
				RegionID:       "region-b",
				Version:        1,
				SequenceNumber: int64(messageCount),
				CreatedAt:      time.Now().Add(10 * time.Millisecond),
			}

			// Resolve conflict
			resolution := conflictResolver.ResolveConflict(localVersion, remoteVersion)
			logger.Printf("Conflict resolved: %s wins", resolution.Resolution)
		}

		// Simulate some sync operations
		if messageCount%3 == 0 {
			// This would normally trigger actual sync operations
			// For demo purposes, we'll just log
			logger.Printf("Simulating sync operation %d", messageCount)
		}

		// Stop after 50 iterations to avoid infinite simulation
		if messageCount >= 50 {
			logger.Println("Simulation complete")
			return
		}
	}
}
