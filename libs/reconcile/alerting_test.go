package reconcile

import (
	"context"
	"testing"
	"time"
)

func TestAlertManager_EvaluateReport_HighDifferences(t *testing.T) {
	rules := []*AlertRule{
		{
			Type:                AlertTypeHighDifferences,
			Level:               AlertLevelWarning,
			DifferenceThreshold: 50,
			Enabled:             true,
		},
	}

	config := AlertManagerConfig{
		Rules:     rules,
		Notifiers: []AlertNotifier{},
		MaxAlerts: 100,
	}

	am := NewAlertManager(config)

	// Create report with high differences
	report := &ReconciliationReport{
		ReportID:     "test-report-1",
		RegionID:     "region-a",
		TargetRegion: "region-b",
		Summary: ReconciliationSummary{
			TotalDifferences: 100, // Exceeds threshold
		},
		Status: ReportStatusSuccess,
	}

	ctx := context.Background()
	alerts, err := am.EvaluateReport(ctx, report)
	if err != nil {
		t.Fatalf("Failed to evaluate report: %v", err)
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	if len(alerts) > 0 {
		alert := alerts[0]
		if alert.Type != AlertTypeHighDifferences {
			t.Errorf("Expected alert type %s, got %s", AlertTypeHighDifferences, alert.Type)
		}

		if alert.Level != AlertLevelWarning {
			t.Errorf("Expected alert level %s, got %s", AlertLevelWarning, alert.Level)
		}
	}
}

func TestAlertManager_EvaluateReport_HighFailureRate(t *testing.T) {
	rules := []*AlertRule{
		{
			Type:                 AlertTypeHighFailureRate,
			Level:                AlertLevelCritical,
			FailureRateThreshold: 10.0,
			Enabled:              true,
		},
	}

	config := AlertManagerConfig{
		Rules:     rules,
		Notifiers: []AlertNotifier{},
		MaxAlerts: 100,
	}

	am := NewAlertManager(config)

	// Create report with high failure rate
	report := &ReconciliationReport{
		ReportID:     "test-report-2",
		RegionID:     "region-a",
		TargetRegion: "region-b",
		Summary: ReconciliationSummary{
			TotalRepaired:    80,
			TotalFailed:      20,
			SuccessRate:      80.0, // 20% failure rate
			TotalDifferences: 100,
		},
		Status: ReportStatusPartialSuccess,
	}

	ctx := context.Background()
	alerts, err := am.EvaluateReport(ctx, report)
	if err != nil {
		t.Fatalf("Failed to evaluate report: %v", err)
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	if len(alerts) > 0 {
		alert := alerts[0]
		if alert.Type != AlertTypeHighFailureRate {
			t.Errorf("Expected alert type %s, got %s", AlertTypeHighFailureRate, alert.Type)
		}

		if alert.Level != AlertLevelCritical {
			t.Errorf("Expected alert level %s, got %s", AlertLevelCritical, alert.Level)
		}
	}
}

func TestAlertManager_EvaluateReport_LongDuration(t *testing.T) {
	rules := []*AlertRule{
		{
			Type:              AlertTypeLongDuration,
			Level:             AlertLevelWarning,
			DurationThreshold: 5 * time.Minute,
			Enabled:           true,
		},
	}

	config := AlertManagerConfig{
		Rules:     rules,
		Notifiers: []AlertNotifier{},
		MaxAlerts: 100,
	}

	am := NewAlertManager(config)

	// Create report with long duration
	report := &ReconciliationReport{
		ReportID:     "test-report-3",
		RegionID:     "region-a",
		TargetRegion: "region-b",
		Duration:     10 * time.Minute, // Exceeds threshold
		Summary: ReconciliationSummary{
			TotalDifferences: 10,
		},
		Status: ReportStatusSuccess,
	}

	ctx := context.Background()
	alerts, err := am.EvaluateReport(ctx, report)
	if err != nil {
		t.Fatalf("Failed to evaluate report: %v", err)
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	if len(alerts) > 0 {
		alert := alerts[0]
		if alert.Type != AlertTypeLongDuration {
			t.Errorf("Expected alert type %s, got %s", AlertTypeLongDuration, alert.Type)
		}
	}
}

func TestAlertManager_EvaluateReport_ReconcileFailed(t *testing.T) {
	rules := []*AlertRule{
		{
			Type:    AlertTypeReconcileFailed,
			Level:   AlertLevelCritical,
			Enabled: true,
		},
	}

	config := AlertManagerConfig{
		Rules:     rules,
		Notifiers: []AlertNotifier{},
		MaxAlerts: 100,
	}

	am := NewAlertManager(config)

	// Create failed report
	report := &ReconciliationReport{
		ReportID:     "test-report-4",
		RegionID:     "region-a",
		TargetRegion: "region-b",
		Status:       ReportStatusFailed,
		Summary: ReconciliationSummary{
			TotalDifferences: 100,
			TotalFailed:      100,
		},
		Errors: []ErrorRecord{
			{Message: "Network error"},
		},
	}

	ctx := context.Background()
	alerts, err := am.EvaluateReport(ctx, report)
	if err != nil {
		t.Fatalf("Failed to evaluate report: %v", err)
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	if len(alerts) > 0 {
		alert := alerts[0]
		if alert.Type != AlertTypeReconcileFailed {
			t.Errorf("Expected alert type %s, got %s", AlertTypeReconcileFailed, alert.Type)
		}

		if alert.Level != AlertLevelCritical {
			t.Errorf("Expected alert level %s, got %s", AlertLevelCritical, alert.Level)
		}
	}
}

func TestAlertManager_EvaluateReport_DataInconsistency(t *testing.T) {
	rules := []*AlertRule{
		{
			Type:    AlertTypeDataInconsistency,
			Level:   AlertLevelCritical,
			Enabled: true,
		},
	}

	config := AlertManagerConfig{
		Rules:     rules,
		Notifiers: []AlertNotifier{},
		MaxAlerts: 100,
	}

	am := NewAlertManager(config)

	// Create report with conflicts
	report := &ReconciliationReport{
		ReportID:     "test-report-5",
		RegionID:     "region-a",
		TargetRegion: "region-b",
		Summary: ReconciliationSummary{
			Conflicts: 5,
		},
		Differences: DifferenceDetails{
			ConflictIDs: []string{"msg1", "msg2", "msg3", "msg4", "msg5"},
		},
		Status: ReportStatusPartialSuccess,
	}

	ctx := context.Background()
	alerts, err := am.EvaluateReport(ctx, report)
	if err != nil {
		t.Fatalf("Failed to evaluate report: %v", err)
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	if len(alerts) > 0 {
		alert := alerts[0]
		if alert.Type != AlertTypeDataInconsistency {
			t.Errorf("Expected alert type %s, got %s", AlertTypeDataInconsistency, alert.Type)
		}

		if alert.Level != AlertLevelCritical {
			t.Errorf("Expected alert level %s, got %s", AlertLevelCritical, alert.Level)
		}
	}
}

func TestAlertManager_EvaluateReport_MultipleAlerts(t *testing.T) {
	rules := DefaultAlertRules()

	config := AlertManagerConfig{
		Rules:     rules,
		Notifiers: []AlertNotifier{},
		MaxAlerts: 100,
	}

	am := NewAlertManager(config)

	// Create report that triggers multiple alerts
	report := &ReconciliationReport{
		ReportID:     "test-report-6",
		RegionID:     "region-a",
		TargetRegion: "region-b",
		Duration:     10 * time.Minute, // Long duration
		Summary: ReconciliationSummary{
			TotalDifferences: 200, // High differences
			TotalRepaired:    150,
			TotalFailed:      50,
			SuccessRate:      75.0, // 25% failure rate
			Conflicts:        10,   // Data inconsistency
		},
		Differences: DifferenceDetails{
			ConflictIDs: []string{"msg1", "msg2"},
		},
		Status: ReportStatusPartialSuccess,
	}

	ctx := context.Background()
	alerts, err := am.EvaluateReport(ctx, report)
	if err != nil {
		t.Fatalf("Failed to evaluate report: %v", err)
	}

	// Should trigger multiple alerts
	if len(alerts) < 2 {
		t.Errorf("Expected at least 2 alerts, got %d", len(alerts))
	}
}

func TestAlertManager_GetAlert(t *testing.T) {
	config := AlertManagerConfig{
		Rules:     DefaultAlertRules(),
		Notifiers: []AlertNotifier{},
		MaxAlerts: 100,
	}

	am := NewAlertManager(config)

	// Create and evaluate report
	report := &ReconciliationReport{
		ReportID:     "test-report-7",
		RegionID:     "region-a",
		TargetRegion: "region-b",
		Summary: ReconciliationSummary{
			TotalDifferences: 200,
		},
		Status: ReportStatusSuccess,
	}

	ctx := context.Background()
	alerts, err := am.EvaluateReport(ctx, report)
	if err != nil {
		t.Fatalf("Failed to evaluate report: %v", err)
	}

	if len(alerts) == 0 {
		t.Fatal("Expected at least one alert")
	}

	// Get alert by ID
	alert, err := am.GetAlert(alerts[0].ID)
	if err != nil {
		t.Fatalf("Failed to get alert: %v", err)
	}

	if alert.ID != alerts[0].ID {
		t.Errorf("Expected alert ID %s, got %s", alerts[0].ID, alert.ID)
	}
}

func TestAlertManager_ListAlerts(t *testing.T) {
	config := AlertManagerConfig{
		Rules:     DefaultAlertRules(),
		Notifiers: []AlertNotifier{},
		MaxAlerts: 100,
	}

	am := NewAlertManager(config)

	// Create multiple reports
	for i := 0; i < 3; i++ {
		report := &ReconciliationReport{
			ReportID:     "test-report-" + string(rune('0'+i)),
			RegionID:     "region-a",
			TargetRegion: "region-b",
			Summary: ReconciliationSummary{
				TotalDifferences: 200,
			},
			Status: ReportStatusSuccess,
		}

		ctx := context.Background()
		_, err := am.EvaluateReport(ctx, report)
		if err != nil {
			t.Fatalf("Failed to evaluate report: %v", err)
		}
	}

	// List all alerts
	alerts := am.ListAlerts()
	if len(alerts) < 3 {
		t.Errorf("Expected at least 3 alerts, got %d", len(alerts))
	}
}

func TestAlertManager_ResolveAlert(t *testing.T) {
	config := AlertManagerConfig{
		Rules:     DefaultAlertRules(),
		Notifiers: []AlertNotifier{},
		MaxAlerts: 100,
	}

	am := NewAlertManager(config)

	// Create and evaluate report
	report := &ReconciliationReport{
		ReportID:     "test-report-8",
		RegionID:     "region-a",
		TargetRegion: "region-b",
		Summary: ReconciliationSummary{
			TotalDifferences: 200,
		},
		Status: ReportStatusSuccess,
	}

	ctx := context.Background()
	alerts, err := am.EvaluateReport(ctx, report)
	if err != nil {
		t.Fatalf("Failed to evaluate report: %v", err)
	}

	if len(alerts) == 0 {
		t.Fatal("Expected at least one alert")
	}

	// Resolve alert
	err = am.ResolveAlert(alerts[0].ID)
	if err != nil {
		t.Fatalf("Failed to resolve alert: %v", err)
	}

	// Verify alert is resolved
	alert, err := am.GetAlert(alerts[0].ID)
	if err != nil {
		t.Fatalf("Failed to get alert: %v", err)
	}

	if !alert.Resolved {
		t.Error("Expected alert to be resolved")
	}

	if alert.ResolvedAt == nil {
		t.Error("Expected ResolvedAt to be set")
	}
}

func TestAlertManager_GetUnresolvedAlerts(t *testing.T) {
	config := AlertManagerConfig{
		Rules:     DefaultAlertRules(),
		Notifiers: []AlertNotifier{},
		MaxAlerts: 100,
	}

	am := NewAlertManager(config)

	// Create multiple reports
	for i := 0; i < 3; i++ {
		report := &ReconciliationReport{
			ReportID:     "test-report-" + string(rune('0'+i)),
			RegionID:     "region-a",
			TargetRegion: "region-b",
			Summary: ReconciliationSummary{
				TotalDifferences: 200,
			},
			Status: ReportStatusSuccess,
		}

		ctx := context.Background()
		_, err := am.EvaluateReport(ctx, report)
		if err != nil {
			t.Fatalf("Failed to evaluate report: %v", err)
		}
	}

	// Get all alerts
	allAlerts := am.ListAlerts()
	if len(allAlerts) == 0 {
		t.Fatal("Expected at least one alert")
	}

	// Resolve first alert
	err := am.ResolveAlert(allAlerts[0].ID)
	if err != nil {
		t.Fatalf("Failed to resolve alert: %v", err)
	}

	// Get unresolved alerts
	unresolved := am.GetUnresolvedAlerts()
	if len(unresolved) != len(allAlerts)-1 {
		t.Errorf("Expected %d unresolved alerts, got %d", len(allAlerts)-1, len(unresolved))
	}

	// Verify resolved alert is not in unresolved list
	for _, alert := range unresolved {
		if alert.ID == allAlerts[0].ID {
			t.Error("Resolved alert should not be in unresolved list")
		}
	}
}

func TestLogNotifier_SendAlert(t *testing.T) {
	logger := &SimpleLogger{}
	notifier := NewLogNotifier(logger)

	alert := &Alert{
		ID:       "test-alert-1",
		Level:    AlertLevelWarning,
		Type:     AlertTypeHighDifferences,
		Title:    "Test Alert",
		Message:  "This is a test alert",
		RegionID: "region-a",
		ReportID: "report-1",
	}

	ctx := context.Background()
	err := notifier.SendAlert(ctx, alert)
	if err != nil {
		t.Fatalf("Failed to send alert: %v", err)
	}
}

func TestLogNotifier_SendBatchAlerts(t *testing.T) {
	logger := &SimpleLogger{}
	notifier := NewLogNotifier(logger)

	alerts := []*Alert{
		{
			ID:       "test-alert-1",
			Level:    AlertLevelWarning,
			Type:     AlertTypeHighDifferences,
			Message:  "Alert 1",
			RegionID: "region-a",
		},
		{
			ID:       "test-alert-2",
			Level:    AlertLevelCritical,
			Type:     AlertTypeReconcileFailed,
			Message:  "Alert 2",
			RegionID: "region-a",
		},
	}

	ctx := context.Background()
	err := notifier.SendBatchAlerts(ctx, alerts)
	if err != nil {
		t.Fatalf("Failed to send batch alerts: %v", err)
	}
}
