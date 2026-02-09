package reconcile

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReportGenerator_GenerateReport(t *testing.T) {
	// Create temporary directory for reports
	tmpDir := t.TempDir()

	config := ReportGeneratorConfig{
		OutputDir:     tmpDir,
		MaxReports:    10,
		EnablePersist: true,
	}

	rg, err := NewReportGenerator(config)
	if err != nil {
		t.Fatalf("Failed to create report generator: %v", err)
	}

	// Create test data
	stats := &ReconcileStats{
		StartTime:       time.Now().Add(-1 * time.Hour),
		EndTime:         time.Now(),
		Duration:        1 * time.Hour,
		MessagesChecked: 1000,
		Differences:     50,
		Repaired:        45,
		Failed:          5,
	}

	diff := &DiffResult{
		MissingInLocal:  []string{"msg1", "msg2", "msg3"},
		MissingInRemote: []string{"msg4", "msg5"},
		Conflicts:       []string{"msg6"},
		TotalChecked:    1000,
		DiffCount:       6,
	}

	repairResults := []RepairResult{
		{GlobalID: "msg1", Success: true, Duration: 100 * time.Millisecond, Attempts: 1},
		{GlobalID: "msg2", Success: false, Duration: 200 * time.Millisecond, Attempts: 2},
	}

	errors := []ErrorRecord{
		{
			Timestamp:   time.Now(),
			Phase:       "repair",
			Message:     "Failed to fetch message",
			GlobalID:    "msg2",
			ErrorType:   "network_error",
			Recoverable: true,
		},
	}

	timeWindow := TimeWindow{
		StartTime: time.Now().Add(-24 * time.Hour),
		EndTime:   time.Now(),
	}

	// Generate report
	report, err := rg.GenerateReport("region-a", "region-b", stats, diff, repairResults, errors, timeWindow)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Verify report
	if report.RegionID != "region-a" {
		t.Errorf("Expected region_id 'region-a', got '%s'", report.RegionID)
	}

	if report.TargetRegion != "region-b" {
		t.Errorf("Expected target_region 'region-b', got '%s'", report.TargetRegion)
	}

	if report.Summary.TotalMessagesChecked != 1000 {
		t.Errorf("Expected 1000 messages checked, got %d", report.Summary.TotalMessagesChecked)
	}

	if report.Summary.TotalDifferences != 50 {
		t.Errorf("Expected 50 differences, got %d", report.Summary.TotalDifferences)
	}

	if report.Summary.TotalRepaired != 45 {
		t.Errorf("Expected 45 repaired, got %d", report.Summary.TotalRepaired)
	}

	if report.Summary.TotalFailed != 5 {
		t.Errorf("Expected 5 failed, got %d", report.Summary.TotalFailed)
	}

	// Verify report was persisted
	reportPath := filepath.Join(tmpDir, report.ReportID+".json")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Errorf("Report file was not created: %s", reportPath)
	}
}

func TestReportGenerator_GetReport(t *testing.T) {
	tmpDir := t.TempDir()

	config := ReportGeneratorConfig{
		OutputDir:     tmpDir,
		MaxReports:    10,
		EnablePersist: true,
	}

	rg, err := NewReportGenerator(config)
	if err != nil {
		t.Fatalf("Failed to create report generator: %v", err)
	}

	// Generate a report
	stats := &ReconcileStats{
		StartTime:       time.Now(),
		EndTime:         time.Now(),
		Duration:        1 * time.Minute,
		MessagesChecked: 100,
		Differences:     5,
		Repaired:        5,
		Failed:          0,
	}

	diff := &DiffResult{
		MissingInLocal: []string{"msg1"},
		TotalChecked:   100,
		DiffCount:      1,
	}

	timeWindow := TimeWindow{
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
	}

	report, err := rg.GenerateReport("region-a", "region-b", stats, diff, []RepairResult{}, []ErrorRecord{}, timeWindow)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Retrieve report
	retrieved, err := rg.GetReport(report.ReportID)
	if err != nil {
		t.Fatalf("Failed to get report: %v", err)
	}

	if retrieved.ReportID != report.ReportID {
		t.Errorf("Expected report ID '%s', got '%s'", report.ReportID, retrieved.ReportID)
	}
}

func TestReportGenerator_ListReports(t *testing.T) {
	tmpDir := t.TempDir()

	config := ReportGeneratorConfig{
		OutputDir:     tmpDir,
		MaxReports:    10,
		EnablePersist: false,
	}

	rg, err := NewReportGenerator(config)
	if err != nil {
		t.Fatalf("Failed to create report generator: %v", err)
	}

	// Generate multiple reports
	for i := 0; i < 3; i++ {
		stats := &ReconcileStats{
			StartTime:       time.Now(),
			EndTime:         time.Now(),
			Duration:        1 * time.Minute,
			MessagesChecked: 100,
			Differences:     i,
			Repaired:        i,
			Failed:          0,
		}

		diff := &DiffResult{
			TotalChecked: 100,
			DiffCount:    i,
		}

		timeWindow := TimeWindow{
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
		}

		_, err := rg.GenerateReport("region-a", "region-b", stats, diff, []RepairResult{}, []ErrorRecord{}, timeWindow)
		if err != nil {
			t.Fatalf("Failed to generate report: %v", err)
		}

		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// List reports
	reports := rg.ListReports()
	if len(reports) != 3 {
		t.Errorf("Expected 3 reports, got %d", len(reports))
	}
}

func TestReportGenerator_GetRecentReports(t *testing.T) {
	tmpDir := t.TempDir()

	config := ReportGeneratorConfig{
		OutputDir:     tmpDir,
		MaxReports:    10,
		EnablePersist: false,
	}

	rg, err := NewReportGenerator(config)
	if err != nil {
		t.Fatalf("Failed to create report generator: %v", err)
	}

	// Generate multiple reports
	for i := 0; i < 5; i++ {
		stats := &ReconcileStats{
			StartTime:       time.Now(),
			EndTime:         time.Now(),
			Duration:        1 * time.Minute,
			MessagesChecked: 100,
			Differences:     i,
			Repaired:        i,
			Failed:          0,
		}

		diff := &DiffResult{
			TotalChecked: 100,
			DiffCount:    i,
		}

		timeWindow := TimeWindow{
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
		}

		_, err := rg.GenerateReport("region-a", "region-b", stats, diff, []RepairResult{}, []ErrorRecord{}, timeWindow)
		if err != nil {
			t.Fatalf("Failed to generate report: %v", err)
		}

		time.Sleep(10 * time.Millisecond)
	}

	// Get recent reports
	recent := rg.GetRecentReports(3)
	if len(recent) != 3 {
		t.Errorf("Expected 3 recent reports, got %d", len(recent))
	}

	// Verify they are sorted by time (most recent first)
	for i := 0; i < len(recent)-1; i++ {
		if recent[i].GeneratedAt.Before(recent[i+1].GeneratedAt) {
			t.Errorf("Reports not sorted correctly")
		}
	}
}

func TestReportGenerator_ExportReportToJSON(t *testing.T) {
	tmpDir := t.TempDir()

	config := ReportGeneratorConfig{
		OutputDir:     tmpDir,
		MaxReports:    10,
		EnablePersist: false,
	}

	rg, err := NewReportGenerator(config)
	if err != nil {
		t.Fatalf("Failed to create report generator: %v", err)
	}

	// Generate a report
	stats := &ReconcileStats{
		StartTime:       time.Now(),
		EndTime:         time.Now(),
		Duration:        1 * time.Minute,
		MessagesChecked: 100,
		Differences:     5,
		Repaired:        5,
		Failed:          0,
	}

	diff := &DiffResult{
		MissingInLocal: []string{"msg1"},
		TotalChecked:   100,
		DiffCount:      1,
	}

	timeWindow := TimeWindow{
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
	}

	report, err := rg.GenerateReport("region-a", "region-b", stats, diff, []RepairResult{}, []ErrorRecord{}, timeWindow)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Export to JSON
	jsonData, err := rg.ExportReportToJSON(report.ReportID)
	if err != nil {
		t.Fatalf("Failed to export report: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Expected non-empty JSON data")
	}
}

func TestReportGenerator_GetReportSummary(t *testing.T) {
	tmpDir := t.TempDir()

	config := ReportGeneratorConfig{
		OutputDir:     tmpDir,
		MaxReports:    10,
		EnablePersist: false,
	}

	rg, err := NewReportGenerator(config)
	if err != nil {
		t.Fatalf("Failed to create report generator: %v", err)
	}

	// Generate a report
	stats := &ReconcileStats{
		StartTime:       time.Now(),
		EndTime:         time.Now(),
		Duration:        1 * time.Minute,
		MessagesChecked: 100,
		Differences:     5,
		Repaired:        4,
		Failed:          1,
	}

	diff := &DiffResult{
		MissingInLocal:  []string{"msg1", "msg2"},
		MissingInRemote: []string{"msg3"},
		Conflicts:       []string{"msg4"},
		TotalChecked:    100,
		DiffCount:       4,
	}

	timeWindow := TimeWindow{
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
	}

	report, err := rg.GenerateReport("region-a", "region-b", stats, diff, []RepairResult{}, []ErrorRecord{}, timeWindow)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Get summary
	summary, err := rg.GetReportSummary(report.ReportID)
	if err != nil {
		t.Fatalf("Failed to get report summary: %v", err)
	}

	if len(summary) == 0 {
		t.Error("Expected non-empty summary")
	}

	// Verify summary contains key information
	if !contains(summary, "region-a") {
		t.Error("Summary should contain region ID")
	}

	if !contains(summary, "100") {
		t.Error("Summary should contain messages checked count")
	}
}

func TestReportGenerator_PruneOldReports(t *testing.T) {
	tmpDir := t.TempDir()

	config := ReportGeneratorConfig{
		OutputDir:     tmpDir,
		MaxReports:    3,
		EnablePersist: false,
	}

	rg, err := NewReportGenerator(config)
	if err != nil {
		t.Fatalf("Failed to create report generator: %v", err)
	}

	// Generate more reports than maxReports
	for i := 0; i < 5; i++ {
		stats := &ReconcileStats{
			StartTime:       time.Now(),
			EndTime:         time.Now(),
			Duration:        1 * time.Minute,
			MessagesChecked: 100,
			Differences:     i,
			Repaired:        i,
			Failed:          0,
		}

		diff := &DiffResult{
			TotalChecked: 100,
			DiffCount:    i,
		}

		timeWindow := TimeWindow{
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
		}

		_, err := rg.GenerateReport("region-a", "region-b", stats, diff, []RepairResult{}, []ErrorRecord{}, timeWindow)
		if err != nil {
			t.Fatalf("Failed to generate report: %v", err)
		}

		time.Sleep(10 * time.Millisecond)
	}

	// Verify only maxReports are kept
	reports := rg.ListReports()
	if len(reports) != 3 {
		t.Errorf("Expected 3 reports after pruning, got %d", len(reports))
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
