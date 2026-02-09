package reconcile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ReconciliationReport represents a comprehensive reconciliation report
type ReconciliationReport struct {
	ReportID      string                 `json:"report_id"`
	RegionID      string                 `json:"region_id"`
	TargetRegion  string                 `json:"target_region"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	Duration      time.Duration          `json:"duration"`
	TimeWindow    TimeWindow             `json:"time_window"`
	Summary       ReconciliationSummary  `json:"summary"`
	Differences   DifferenceDetails      `json:"differences"`
	RepairResults []RepairRecord         `json:"repair_results"`
	Errors        []ErrorRecord          `json:"errors"`
	Status        ReportStatus           `json:"status"`
	GeneratedAt   time.Time              `json:"generated_at"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// TimeWindow represents the time range for reconciliation
type TimeWindow struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// ReconciliationSummary provides high-level statistics
type ReconciliationSummary struct {
	TotalMessagesChecked int     `json:"total_messages_checked"`
	TotalDifferences     int     `json:"total_differences"`
	MissingInLocal       int     `json:"missing_in_local"`
	MissingInRemote      int     `json:"missing_in_remote"`
	Conflicts            int     `json:"conflicts"`
	TotalRepaired        int     `json:"total_repaired"`
	TotalFailed          int     `json:"total_failed"`
	SuccessRate          float64 `json:"success_rate"`
}

// DifferenceDetails provides detailed information about differences
type DifferenceDetails struct {
	MissingInLocalIDs  []string `json:"missing_in_local_ids"`
	MissingInRemoteIDs []string `json:"missing_in_remote_ids"`
	ConflictIDs        []string `json:"conflict_ids"`
}

// RepairRecord represents a single repair operation record
type RepairRecord struct {
	GlobalID     string        `json:"global_id"`
	Operation    string        `json:"operation"`
	Status       string        `json:"status"` // "success", "failed", "skipped"
	Error        string        `json:"error,omitempty"`
	Duration     time.Duration `json:"duration"`
	Attempts     int           `json:"attempts"`
	Timestamp    time.Time     `json:"timestamp"`
	TargetRegion string        `json:"target_region"`
}

// ErrorRecord represents an error that occurred during reconciliation
type ErrorRecord struct {
	Timestamp   time.Time `json:"timestamp"`
	Phase       string    `json:"phase"` // "fetch", "compare", "repair"
	Message     string    `json:"message"`
	GlobalID    string    `json:"global_id,omitempty"`
	ErrorType   string    `json:"error_type"`
	Recoverable bool      `json:"recoverable"`
}

// ReportStatus represents the overall status of the reconciliation
type ReportStatus string

const (
	ReportStatusSuccess        ReportStatus = "success"
	ReportStatusPartialSuccess ReportStatus = "partial_success"
	ReportStatusFailed         ReportStatus = "failed"
	ReportStatusInProgress     ReportStatus = "in_progress"
)

// ReportGenerator generates reconciliation reports
type ReportGenerator struct {
	mu            sync.RWMutex
	outputDir     string
	reports       map[string]*ReconciliationReport // reportID -> report
	maxReports    int                              // Maximum number of reports to keep in memory
	enablePersist bool                             // Whether to persist reports to disk
}

// ReportGeneratorConfig holds configuration for report generation
type ReportGeneratorConfig struct {
	OutputDir     string // Directory to store report files
	MaxReports    int    // Maximum reports to keep in memory
	EnablePersist bool   // Enable persistence to disk
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(config ReportGeneratorConfig) (*ReportGenerator, error) {
	rg := &ReportGenerator{
		outputDir:     config.OutputDir,
		reports:       make(map[string]*ReconciliationReport),
		maxReports:    config.MaxReports,
		enablePersist: config.EnablePersist,
	}

	// Create output directory if it doesn't exist
	if rg.enablePersist && rg.outputDir != "" {
		if err := os.MkdirAll(rg.outputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	return rg, nil
}

// GenerateReport creates a comprehensive reconciliation report
func (rg *ReportGenerator) GenerateReport(
	regionID string,
	targetRegion string,
	stats *ReconcileStats,
	diff *DiffResult,
	repairResults []RepairResult,
	errors []ErrorRecord,
	timeWindow TimeWindow,
) (*ReconciliationReport, error) {
	reportID := generateReportID(regionID, time.Now())

	// Calculate success rate
	successRate := 0.0
	if stats.Repaired+stats.Failed > 0 {
		successRate = float64(stats.Repaired) / float64(stats.Repaired+stats.Failed) * 100.0
	}

	// Determine report status
	status := rg.determineStatus(stats, diff)

	// Convert repair results to repair records
	repairRecords := make([]RepairRecord, 0, len(repairResults))
	for _, result := range repairResults {
		record := RepairRecord{
			GlobalID:     result.GlobalID,
			Operation:    "repair",
			Status:       rg.getRepairStatus(result),
			Duration:     result.Duration,
			Attempts:     result.Attempts,
			Timestamp:    time.Now(),
			TargetRegion: targetRegion,
		}
		if result.Error != nil {
			record.Error = result.Error.Error()
		}
		repairRecords = append(repairRecords, record)
	}

	report := &ReconciliationReport{
		ReportID:     reportID,
		RegionID:     regionID,
		TargetRegion: targetRegion,
		StartTime:    stats.StartTime,
		EndTime:      stats.EndTime,
		Duration:     stats.Duration,
		TimeWindow:   timeWindow,
		Summary: ReconciliationSummary{
			TotalMessagesChecked: stats.MessagesChecked,
			TotalDifferences:     stats.Differences,
			MissingInLocal:       len(diff.MissingInLocal),
			MissingInRemote:      len(diff.MissingInRemote),
			Conflicts:            len(diff.Conflicts),
			TotalRepaired:        stats.Repaired,
			TotalFailed:          stats.Failed,
			SuccessRate:          successRate,
		},
		Differences: DifferenceDetails{
			MissingInLocalIDs:  diff.MissingInLocal,
			MissingInRemoteIDs: diff.MissingInRemote,
			ConflictIDs:        diff.Conflicts,
		},
		RepairResults: repairRecords,
		Errors:        errors,
		Status:        status,
		GeneratedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"version":   "1.0",
			"generator": "reconcile-report-generator",
		},
	}

	// Store report in memory
	rg.mu.Lock()
	rg.reports[reportID] = report
	rg.pruneOldReports()
	rg.mu.Unlock()

	// Persist to disk if enabled
	if rg.enablePersist {
		if err := rg.persistReport(report); err != nil {
			return report, fmt.Errorf("failed to persist report: %w", err)
		}
	}

	return report, nil
}

// GetReport retrieves a report by ID
func (rg *ReportGenerator) GetReport(reportID string) (*ReconciliationReport, error) {
	rg.mu.RLock()
	report, exists := rg.reports[reportID]
	rg.mu.RUnlock()

	if exists {
		return report, nil
	}

	// Try to load from disk if persistence is enabled
	if rg.enablePersist {
		return rg.loadReport(reportID)
	}

	return nil, fmt.Errorf("report not found: %s", reportID)
}

// ListReports returns all reports in memory
func (rg *ReportGenerator) ListReports() []*ReconciliationReport {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	reports := make([]*ReconciliationReport, 0, len(rg.reports))
	for _, report := range rg.reports {
		reports = append(reports, report)
	}

	return reports
}

// GetRecentReports returns the N most recent reports
func (rg *ReportGenerator) GetRecentReports(n int) []*ReconciliationReport {
	reports := rg.ListReports()

	// Sort by generated time (descending)
	for i := 0; i < len(reports)-1; i++ {
		for j := i + 1; j < len(reports); j++ {
			if reports[i].GeneratedAt.Before(reports[j].GeneratedAt) {
				reports[i], reports[j] = reports[j], reports[i]
			}
		}
	}

	if n > len(reports) {
		n = len(reports)
	}

	return reports[:n]
}

// persistReport saves a report to disk
func (rg *ReportGenerator) persistReport(report *ReconciliationReport) error {
	filename := fmt.Sprintf("%s.json", report.ReportID)
	filepath := filepath.Join(rg.outputDir, filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	return nil
}

// loadReport loads a report from disk
func (rg *ReportGenerator) loadReport(reportID string) (*ReconciliationReport, error) {
	filename := fmt.Sprintf("%s.json", reportID)
	filepath := filepath.Join(rg.outputDir, filename)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read report file: %w", err)
	}

	var report ReconciliationReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report: %w", err)
	}

	return &report, nil
}

// pruneOldReports removes old reports from memory to maintain maxReports limit
func (rg *ReportGenerator) pruneOldReports() {
	if len(rg.reports) <= rg.maxReports {
		return
	}

	// Find oldest reports
	type reportAge struct {
		id  string
		age time.Time
	}

	ages := make([]reportAge, 0, len(rg.reports))
	for id, report := range rg.reports {
		ages = append(ages, reportAge{id: id, age: report.GeneratedAt})
	}

	// Sort by age (oldest first)
	for i := 0; i < len(ages)-1; i++ {
		for j := i + 1; j < len(ages); j++ {
			if ages[i].age.After(ages[j].age) {
				ages[i], ages[j] = ages[j], ages[i]
			}
		}
	}

	// Remove oldest reports
	toRemove := len(rg.reports) - rg.maxReports
	for i := 0; i < toRemove; i++ {
		delete(rg.reports, ages[i].id)
	}
}

// determineStatus determines the overall status of the reconciliation
func (rg *ReportGenerator) determineStatus(stats *ReconcileStats, diff *DiffResult) ReportStatus {
	if stats.Differences == 0 {
		return ReportStatusSuccess
	}

	if stats.Failed == 0 {
		return ReportStatusSuccess
	}

	if stats.Repaired > 0 && stats.Failed > 0 {
		return ReportStatusPartialSuccess
	}

	if stats.Failed > 0 && stats.Repaired == 0 {
		return ReportStatusFailed
	}

	return ReportStatusSuccess
}

// getRepairStatus converts RepairResult to status string
func (rg *ReportGenerator) getRepairStatus(result RepairResult) string {
	if result.Success {
		return "success"
	}
	return "failed"
}

// generateReportID generates a unique report ID
func generateReportID(regionID string, timestamp time.Time) string {
	return fmt.Sprintf("%s-%d", regionID, timestamp.UnixNano())
}

// ExportReportToJSON exports a report to JSON format
func (rg *ReportGenerator) ExportReportToJSON(reportID string) ([]byte, error) {
	report, err := rg.GetReport(reportID)
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal report: %w", err)
	}

	return data, nil
}

// GetReportSummary returns a summary of a report
func (rg *ReportGenerator) GetReportSummary(reportID string) (string, error) {
	report, err := rg.GetReport(reportID)
	if err != nil {
		return "", err
	}

	summary := fmt.Sprintf(`Reconciliation Report Summary
================================
Report ID: %s
Region: %s -> %s
Status: %s
Time Window: %s to %s
Duration: %s

Messages Checked: %d
Differences Found: %d
  - Missing in Local: %d
  - Missing in Remote: %d
  - Conflicts: %d

Repairs:
  - Successful: %d
  - Failed: %d
  - Success Rate: %.2f%%

Generated At: %s
`,
		report.ReportID,
		report.RegionID,
		report.TargetRegion,
		report.Status,
		report.TimeWindow.StartTime.Format(time.RFC3339),
		report.TimeWindow.EndTime.Format(time.RFC3339),
		report.Duration,
		report.Summary.TotalMessagesChecked,
		report.Summary.TotalDifferences,
		report.Summary.MissingInLocal,
		report.Summary.MissingInRemote,
		report.Summary.Conflicts,
		report.Summary.TotalRepaired,
		report.Summary.TotalFailed,
		report.Summary.SuccessRate,
		report.GeneratedAt.Format(time.RFC3339),
	)

	return summary, nil
}

// DefaultReportGeneratorConfig returns default configuration
func DefaultReportGeneratorConfig() ReportGeneratorConfig {
	return ReportGeneratorConfig{
		OutputDir:     "./reconcile-reports",
		MaxReports:    100,
		EnablePersist: true,
	}
}
