# Reconcile Package

The `reconcile` package provides Merkle Tree-based data reconciliation for multi-region active-active architectures. It enables efficient detection and repair of data inconsistencies between regions.

## Features

### 1. Merkle Tree Implementation (`merkle_tree.go`)
- **Efficient Diff Algorithm**: O(log n) comparison of message datasets
- **Fast Difference Detection**: Only traverses subtrees with different hashes
- **Incremental Updates**: Rebuild trees with new messages
- **Range Queries**: Retrieve messages within specific GlobalID ranges
- **Concurrent Access**: Thread-safe read operations

### 2. Reconciliation Engine (`reconciler.go`)
- **Periodic Reconciliation**: Automated scheduled reconciliation runs
- **On-Demand Reconciliation**: Manual reconciliation for specific time ranges
- **Auto-Repair**: Automatically fix detected differences
- **Dry-Run Mode**: Detect differences without making changes
- **Statistics Tracking**: Monitor reconciliation performance and results
- **Integrated Reporting**: Automatic report generation for each run
- **Integrated Alerting**: Rule-based alert evaluation and notification

### 3. Incremental Repair (`incremental_repair.go`)
- **Repair Queue**: Priority-based task queue for repairs
- **Batch Processing**: Process multiple repairs efficiently
- **Retry Logic**: Automatic retry with exponential backoff
- **Multiple Strategies**: Pull, push, or bidirectional repair

### 4. Report Generation (`report.go`)
- **Comprehensive Reports**: Detailed reconciliation reports with all metrics
- **Persistent Storage**: Save reports to disk in JSON format
- **Report Querying**: Retrieve reports by ID or get recent reports
- **Summary Generation**: Human-readable report summaries
- **JSON Export**: Export reports for external processing
- **Automatic Pruning**: Maintain configurable report history

### 5. Alert Management (`alerting.go`)
- **Rule-Based Alerts**: Configurable alert rules with thresholds
- **Multiple Alert Types**: High differences, failure rate, duration, failures, inconsistencies
- **Severity Levels**: Info, Warning, Critical
- **Pluggable Notifiers**: Easy integration with notification systems
- **Alert Lifecycle**: Create, query, resolve alerts
- **Batch Notifications**: Efficient batch alert processing

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Reconciler                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │   Periodic   │  │  On-Demand   │  │   Dry-Run    │     │
│  │ Reconciliation│  │Reconciliation│  │     Mode     │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         │                  │                  │             │
│         └──────────────────┴──────────────────┘             │
│                            │                                │
└────────────────────────────┼────────────────────────────────┘
                             │
                    ┌────────▼────────┐
                    │  Merkle Tree    │
                    │  Diff Algorithm │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
     ┌────────▼────────┐         ┌─────────▼────────┐
     │  Local Messages │         │ Remote Messages  │
     │   (Region A)    │         │   (Region B)     │
     └─────────────────┘         └──────────────────┘
                             │
                    ┌────────▼────────┐
                    │ Incremental     │
                    │ Repairer        │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
     ┌────────▼────────┐         ┌─────────▼────────┐
     │  Repair Queue   │         │  Batch Processor │
     │  (Priority)     │         │  (Concurrent)    │
     └─────────────────┘         └──────────────────┘
```

## Usage

### Basic Reconciliation

```go
package main

import (
    "context"
    "time"
    
    "github.com/pingxin403/cuckoo/libs/reconcile"
)

func main() {
    // Create reconciler configuration
    config := reconcile.DefaultReconcilerConfig("region-a")
    config.CheckInterval = 1 * time.Hour
    config.TimeWindow = 24 * time.Hour
    config.EnableAutoRepair = true
    
    // Create reconciler
    reconciler := reconcile.NewReconciler(config, store, provider)
    
    // Start periodic reconciliation
    ctx := context.Background()
    if err := reconciler.Start(ctx); err != nil {
        panic(err)
    }
    
    // Stop when done
    defer reconciler.Stop()
}
```

### Reconciliation with Reporting and Alerting

```go
// Create report generator
reportConfig := reconcile.ReportGeneratorConfig{
    OutputDir:     "./reconcile-reports",
    MaxReports:    100,
    EnablePersist: true,
}
reportGenerator, err := reconcile.NewReportGenerator(reportConfig)
if err != nil {
    panic(err)
}

// Create alert manager
alertConfig := reconcile.AlertManagerConfig{
    Rules:     reconcile.DefaultAlertRules(),
    Notifiers: []reconcile.AlertNotifier{
        reconcile.NewLogNotifier(&reconcile.SimpleLogger{}),
    },
    MaxAlerts: 1000,
}
alertManager := reconcile.NewAlertManager(alertConfig)

// Create reconciler with reporting and alerting
reconciler := reconcile.NewReconcilerWithReporting(
    config,
    store,
    provider,
    reportGenerator,
    alertManager,
)

// Start reconciliation
ctx := context.Background()
reconciler.Start(ctx)

// Query reports
reports := reportGenerator.GetRecentReports(10)
for _, report := range reports {
    summary, _ := reportGenerator.GetReportSummary(report.ReportID)
    fmt.Println(summary)
}

// Check alerts
unresolved := alertManager.GetUnresolvedAlerts()
fmt.Printf("Unresolved alerts: %d\n", len(unresolved))
```

### On-Demand Reconciliation

```go
// Perform reconciliation for a specific time range
startTime := time.Now().Add(-24 * time.Hour)
endTime := time.Now()

stats, diff, err := reconciler.RunOnDemandReconciliation(
    ctx, 
    startTime, 
    endTime, 
    "region-b",
)

if err != nil {
    log.Fatalf("Reconciliation failed: %v", err)
}

log.Printf("Checked: %d, Differences: %d, Repaired: %d",
    stats.MessagesChecked,
    stats.Differences,
    stats.Repaired,
)
```

### Merkle Tree Operations

```go
// Create messages
messages := []reconcile.MessageData{
    {
        GlobalID:       "region-a-1000-0-1",
        Content:        "Hello",
        Timestamp:      1000,
        RegionID:       "region-a",
        ConversationID: "conv1",
        SequenceNumber: 1,
    },
    // ... more messages
}

// Compute hashes
for i := range messages {
    messages[i].Hash = reconcile.ComputeMessageHash(messages[i])
}

// Build Merkle tree
tree := reconcile.NewMerkleTree("region-a", messages)

// Get root hash
rootHash := tree.GetRootHash()

// Find differences with remote tree
diff, err := tree.FindDifferences(ctx, remoteTree)
if err != nil {
    log.Fatalf("Diff failed: %v", err)
}

log.Printf("Missing in local: %d", len(diff.MissingInLocal))
log.Printf("Missing in remote: %d", len(diff.MissingInRemote))
log.Printf("Conflicts: %d", len(diff.Conflicts))
```

### Incremental Repair

```go
// Create repair configuration
repairConfig := reconcile.DefaultRepairConfig("region-a")
repairConfig.BatchSize = 50
repairConfig.RetryAttempts = 3

// Create repairer
repairer := reconcile.NewIncrementalRepairer(repairConfig, store, provider)

// Queue repairs from diff result
err := repairer.RepairDifferences(diff, "region-b", 10)
if err != nil {
    log.Fatalf("Failed to queue repairs: %v", err)
}

// Process batch
result, err := repairer.ProcessBatch(ctx)
if err != nil {
    log.Fatalf("Batch processing failed: %v", err)
}

log.Printf("Successful: %d, Failed: %d", result.Successful, result.Failed)
```

## Interfaces

### MessageStore

Implement this interface to provide message storage:

```go
type MessageStore interface {
    GetMessagesForReconciliation(ctx context.Context, startTime, endTime time.Time) ([]MessageData, error)
    GetMessageByGlobalID(ctx context.Context, globalID string) (*MessageData, error)
    StoreMessage(ctx context.Context, msg *MessageData) error
    DeleteMessage(ctx context.Context, globalID string) error
}
```

### RemoteTreeProvider

Implement this interface to fetch remote Merkle trees:

```go
type RemoteTreeProvider interface {
    GetRemoteTree(ctx context.Context, regionID string, startTime, endTime time.Time) (*MerkleTree, error)
    GetRemoteMessages(ctx context.Context, regionID string, globalIDs []string) ([]MessageData, error)
}
```

## Configuration

### ReconcilerConfig

```go
type ReconcilerConfig struct {
    RegionID           string        // Current region ID
    CheckInterval      time.Duration // How often to run reconciliation
    TimeWindow         time.Duration // Time window for each run
    MaxConcurrentFixes int           // Max concurrent repair operations
    EnableAutoRepair   bool          // Auto-repair differences
    DryRun             bool          // Only detect, don't repair
}
```

### RepairConfig

```go
type RepairConfig struct {
    RegionID       string
    Strategy       RepairStrategy // pull, push, or bidirectional
    BatchSize      int            // Messages per batch
    RetryAttempts  int            // Retry attempts for failures
    RetryDelay     time.Duration  // Delay between retries
    MaxQueueSize   int            // Maximum queue size
    EnablePriority bool           // Enable priority-based repair
}
```

## Performance Characteristics

### Merkle Tree
- **Build Time**: O(n log n) where n is the number of messages
- **Diff Time**: O(log n) for identical subtrees, O(n) worst case
- **Space**: O(n) for storing the tree
- **Concurrent Reads**: Lock-free for read operations

### Reconciliation
- **Periodic Overhead**: Minimal, only builds trees and compares hashes
- **Repair Throughput**: Configurable via `MaxConcurrentFixes`
- **Memory Usage**: Proportional to time window size

## Best Practices

1. **Time Window Selection**
   - Use smaller windows (1-6 hours) for frequent reconciliation
   - Use larger windows (24 hours) for comprehensive checks
   - Balance between completeness and performance

2. **Repair Strategy**
   - Start with dry-run mode to understand differences
   - Enable auto-repair after validating behavior
   - Monitor repair success rate

3. **Batch Size**
   - Larger batches (50-100) for better throughput
   - Smaller batches (10-20) for lower latency
   - Adjust based on network conditions

4. **Priority Management**
   - Assign higher priority to recent messages
   - Use priority for critical conversations
   - Balance queue to prevent starvation

## Monitoring

Track these metrics for production deployments:

- **Reconciliation Frequency**: How often reconciliation runs
- **Differences Detected**: Number of inconsistencies found
- **Repair Success Rate**: Percentage of successful repairs
- **Repair Latency**: Time to repair differences
- **Queue Size**: Number of pending repairs

## Testing

Run tests:

```bash
cd reconcile
go test -v ./...
```

Run with coverage:

```bash
go test -v -cover ./...
```

## Requirements

- Go 1.21 or later
- HLC library (`github.com/pingxin403/cuckoo/libs/hlc`)

## Related Documentation

- [Multi-Region Active-Active Design](../.kiro/specs/multi-region-active-active/design.md)
- [HLC Library](../libs/hlc/README.md)
- [Conflict Resolution](../sync/README.md)

## License

See project root LICENSE file.
