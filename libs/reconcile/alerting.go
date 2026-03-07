package reconcile

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AlertLevel represents the severity level of an alert
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeHighDifferences   AlertType = "high_differences"
	AlertTypeHighFailureRate   AlertType = "high_failure_rate"
	AlertTypeLongDuration      AlertType = "long_duration"
	AlertTypeReconcileFailed   AlertType = "reconcile_failed"
	AlertTypeRepairQueueFull   AlertType = "repair_queue_full"
	AlertTypeDataInconsistency AlertType = "data_inconsistency"
)

// Alert represents a reconciliation alert
type Alert struct {
	ID         string                 `json:"id"`
	Level      AlertLevel             `json:"level"`
	Type       AlertType              `json:"type"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	RegionID   string                 `json:"region_id"`
	ReportID   string                 `json:"report_id,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	Type                 AlertType
	Level                AlertLevel
	DifferenceThreshold  int           // Trigger if differences exceed this
	FailureRateThreshold float64       // Trigger if failure rate exceeds this (0-100)
	DurationThreshold    time.Duration // Trigger if reconciliation takes longer than this
	Enabled              bool
}

// AlertNotifier defines the interface for sending alerts
type AlertNotifier interface {
	// SendAlert sends an alert notification
	SendAlert(ctx context.Context, alert *Alert) error

	// SendBatchAlerts sends multiple alerts
	SendBatchAlerts(ctx context.Context, alerts []*Alert) error
}

// AlertManager manages alert generation and notification
type AlertManager struct {
	mu        sync.RWMutex
	rules     map[AlertType]*AlertRule
	notifiers []AlertNotifier
	alerts    map[string]*Alert // alertID -> alert
	maxAlerts int               // Maximum alerts to keep in memory
}

// AlertManagerConfig holds configuration for alert manager
type AlertManagerConfig struct {
	Rules     []*AlertRule
	Notifiers []AlertNotifier
	MaxAlerts int
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config AlertManagerConfig) *AlertManager {
	am := &AlertManager{
		rules:     make(map[AlertType]*AlertRule),
		notifiers: config.Notifiers,
		alerts:    make(map[string]*Alert),
		maxAlerts: config.MaxAlerts,
	}

	// Register rules
	for _, rule := range config.Rules {
		am.rules[rule.Type] = rule
	}

	return am
}

// EvaluateReport evaluates a reconciliation report and generates alerts if needed
func (am *AlertManager) EvaluateReport(ctx context.Context, report *ReconciliationReport) ([]*Alert, error) {
	alerts := make([]*Alert, 0)

	// Check high differences
	if alert := am.checkHighDifferences(report); alert != nil {
		alerts = append(alerts, alert)
	}

	// Check high failure rate
	if alert := am.checkHighFailureRate(report); alert != nil {
		alerts = append(alerts, alert)
	}

	// Check long duration
	if alert := am.checkLongDuration(report); alert != nil {
		alerts = append(alerts, alert)
	}

	// Check reconciliation failure
	if alert := am.checkReconcileFailed(report); alert != nil {
		alerts = append(alerts, alert)
	}

	// Check data inconsistency
	if alert := am.checkDataInconsistency(report); alert != nil {
		alerts = append(alerts, alert)
	}

	// Store alerts
	am.mu.Lock()
	for _, alert := range alerts {
		am.alerts[alert.ID] = alert
	}
	am.pruneOldAlerts()
	am.mu.Unlock()

	// Send notifications
	if len(alerts) > 0 {
		if err := am.sendAlerts(ctx, alerts); err != nil {
			return alerts, fmt.Errorf("failed to send alerts: %w", err)
		}
	}

	return alerts, nil
}

// checkHighDifferences checks if differences exceed threshold
func (am *AlertManager) checkHighDifferences(report *ReconciliationReport) *Alert {
	rule, exists := am.rules[AlertTypeHighDifferences]
	if !exists || !rule.Enabled {
		return nil
	}

	if report.Summary.TotalDifferences > rule.DifferenceThreshold {
		return &Alert{
			ID:        generateAlertID(report.RegionID, AlertTypeHighDifferences, time.Now()),
			Level:     rule.Level,
			Type:      AlertTypeHighDifferences,
			Title:     "High Number of Differences Detected",
			Message:   fmt.Sprintf("Reconciliation found %d differences (threshold: %d)", report.Summary.TotalDifferences, rule.DifferenceThreshold),
			RegionID:  report.RegionID,
			ReportID:  report.ReportID,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"differences":       report.Summary.TotalDifferences,
				"threshold":         rule.DifferenceThreshold,
				"missing_in_local":  report.Summary.MissingInLocal,
				"missing_in_remote": report.Summary.MissingInRemote,
				"conflicts":         report.Summary.Conflicts,
			},
			Resolved: false,
		}
	}

	return nil
}

// checkHighFailureRate checks if repair failure rate exceeds threshold
func (am *AlertManager) checkHighFailureRate(report *ReconciliationReport) *Alert {
	rule, exists := am.rules[AlertTypeHighFailureRate]
	if !exists || !rule.Enabled {
		return nil
	}

	failureRate := 100.0 - report.Summary.SuccessRate
	if failureRate > rule.FailureRateThreshold {
		return &Alert{
			ID:        generateAlertID(report.RegionID, AlertTypeHighFailureRate, time.Now()),
			Level:     rule.Level,
			Type:      AlertTypeHighFailureRate,
			Title:     "High Repair Failure Rate",
			Message:   fmt.Sprintf("Repair failure rate is %.2f%% (threshold: %.2f%%)", failureRate, rule.FailureRateThreshold),
			RegionID:  report.RegionID,
			ReportID:  report.ReportID,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"failure_rate":   failureRate,
				"threshold":      rule.FailureRateThreshold,
				"total_failed":   report.Summary.TotalFailed,
				"total_repaired": report.Summary.TotalRepaired,
			},
			Resolved: false,
		}
	}

	return nil
}

// checkLongDuration checks if reconciliation took too long
func (am *AlertManager) checkLongDuration(report *ReconciliationReport) *Alert {
	rule, exists := am.rules[AlertTypeLongDuration]
	if !exists || !rule.Enabled {
		return nil
	}

	if report.Duration > rule.DurationThreshold {
		return &Alert{
			ID:        generateAlertID(report.RegionID, AlertTypeLongDuration, time.Now()),
			Level:     rule.Level,
			Type:      AlertTypeLongDuration,
			Title:     "Reconciliation Duration Exceeded Threshold",
			Message:   fmt.Sprintf("Reconciliation took %s (threshold: %s)", report.Duration, rule.DurationThreshold),
			RegionID:  report.RegionID,
			ReportID:  report.ReportID,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"duration":  report.Duration.String(),
				"threshold": rule.DurationThreshold.String(),
			},
			Resolved: false,
		}
	}

	return nil
}

// checkReconcileFailed checks if reconciliation failed
func (am *AlertManager) checkReconcileFailed(report *ReconciliationReport) *Alert {
	rule, exists := am.rules[AlertTypeReconcileFailed]
	if !exists || !rule.Enabled {
		return nil
	}

	if report.Status == ReportStatusFailed {
		return &Alert{
			ID:        generateAlertID(report.RegionID, AlertTypeReconcileFailed, time.Now()),
			Level:     rule.Level,
			Type:      AlertTypeReconcileFailed,
			Title:     "Reconciliation Failed",
			Message:   fmt.Sprintf("Reconciliation failed for region %s", report.RegionID),
			RegionID:  report.RegionID,
			ReportID:  report.ReportID,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"status":      string(report.Status),
				"error_count": len(report.Errors),
			},
			Resolved: false,
		}
	}

	return nil
}

// checkDataInconsistency checks for critical data inconsistencies
func (am *AlertManager) checkDataInconsistency(report *ReconciliationReport) *Alert {
	rule, exists := am.rules[AlertTypeDataInconsistency]
	if !exists || !rule.Enabled {
		return nil
	}

	// Trigger if there are conflicts (same message ID, different content)
	if report.Summary.Conflicts > 0 {
		return &Alert{
			ID:        generateAlertID(report.RegionID, AlertTypeDataInconsistency, time.Now()),
			Level:     rule.Level,
			Type:      AlertTypeDataInconsistency,
			Title:     "Data Inconsistency Detected",
			Message:   fmt.Sprintf("Found %d conflicting messages between regions", report.Summary.Conflicts),
			RegionID:  report.RegionID,
			ReportID:  report.ReportID,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"conflicts":    report.Summary.Conflicts,
				"conflict_ids": report.Differences.ConflictIDs,
			},
			Resolved: false,
		}
	}

	return nil
}

// sendAlerts sends alerts to all registered notifiers
func (am *AlertManager) sendAlerts(ctx context.Context, alerts []*Alert) error {
	if len(am.notifiers) == 0 {
		return nil // No notifiers configured
	}

	var errs []error
	for _, notifier := range am.notifiers {
		if err := notifier.SendBatchAlerts(ctx, alerts); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to send alerts to %d notifiers", len(errs))
	}

	return nil
}

// GetAlert retrieves an alert by ID
func (am *AlertManager) GetAlert(alertID string) (*Alert, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return nil, fmt.Errorf("alert not found: %s", alertID)
	}

	return alert, nil
}

// ListAlerts returns all alerts
func (am *AlertManager) ListAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]*Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		alerts = append(alerts, alert)
	}

	return alerts
}

// GetUnresolvedAlerts returns all unresolved alerts
func (am *AlertManager) GetUnresolvedAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]*Alert, 0)
	for _, alert := range am.alerts {
		if !alert.Resolved {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// ResolveAlert marks an alert as resolved
func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	now := time.Now()
	alert.Resolved = true
	alert.ResolvedAt = &now

	return nil
}

// pruneOldAlerts removes old alerts to maintain maxAlerts limit
func (am *AlertManager) pruneOldAlerts() {
	if len(am.alerts) <= am.maxAlerts {
		return
	}

	// Find oldest resolved alerts
	type alertAge struct {
		id  string
		age time.Time
	}

	ages := make([]alertAge, 0)
	for id, alert := range am.alerts {
		if alert.Resolved {
			ages = append(ages, alertAge{id: id, age: alert.Timestamp})
		}
	}

	// Sort by age (oldest first)
	for i := 0; i < len(ages)-1; i++ {
		for j := i + 1; j < len(ages); j++ {
			if ages[i].age.After(ages[j].age) {
				ages[i], ages[j] = ages[j], ages[i]
			}
		}
	}

	// Remove oldest resolved alerts
	toRemove := len(am.alerts) - am.maxAlerts
	for i := 0; i < toRemove && i < len(ages); i++ {
		delete(am.alerts, ages[i].id)
	}
}

// generateAlertID generates a unique alert ID
func generateAlertID(regionID string, alertType AlertType, timestamp time.Time) string {
	return fmt.Sprintf("%s-%s-%d", regionID, alertType, timestamp.UnixNano())
}

// DefaultAlertRules returns default alert rules
func DefaultAlertRules() []*AlertRule {
	return []*AlertRule{
		{
			Type:                AlertTypeHighDifferences,
			Level:               AlertLevelWarning,
			DifferenceThreshold: 100,
			Enabled:             true,
		},
		{
			Type:                 AlertTypeHighFailureRate,
			Level:                AlertLevelCritical,
			FailureRateThreshold: 10.0, // 10% failure rate
			Enabled:              true,
		},
		{
			Type:              AlertTypeLongDuration,
			Level:             AlertLevelWarning,
			DurationThreshold: 5 * time.Minute,
			Enabled:           true,
		},
		{
			Type:    AlertTypeReconcileFailed,
			Level:   AlertLevelCritical,
			Enabled: true,
		},
		{
			Type:    AlertTypeDataInconsistency,
			Level:   AlertLevelCritical,
			Enabled: true,
		},
	}
}

// LogNotifier is a simple notifier that logs alerts
type LogNotifier struct {
	logger Logger
}

// Logger interface for logging
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// NewLogNotifier creates a new log notifier
func NewLogNotifier(logger Logger) *LogNotifier {
	return &LogNotifier{logger: logger}
}

// SendAlert sends a single alert
func (ln *LogNotifier) SendAlert(ctx context.Context, alert *Alert) error {
	message := fmt.Sprintf("[%s] %s: %s (Region: %s, Report: %s)",
		alert.Level, alert.Type, alert.Message, alert.RegionID, alert.ReportID)

	switch alert.Level {
	case AlertLevelCritical:
		ln.logger.Error(message)
	case AlertLevelWarning:
		ln.logger.Warn(message)
	default:
		ln.logger.Info(message)
	}

	return nil
}

// SendBatchAlerts sends multiple alerts
func (ln *LogNotifier) SendBatchAlerts(ctx context.Context, alerts []*Alert) error {
	for _, alert := range alerts {
		if err := ln.SendAlert(ctx, alert); err != nil {
			return err
		}
	}
	return nil
}

// SimpleLogger is a basic logger implementation
type SimpleLogger struct{}

// Info logs info messages
func (sl *SimpleLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("[INFO] "+msg+"\n", args...)
}

// Warn logs warning messages
func (sl *SimpleLogger) Warn(msg string, args ...interface{}) {
	fmt.Printf("[WARN] "+msg+"\n", args...)
}

// Error logs error messages
func (sl *SimpleLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("[ERROR] "+msg+"\n", args...)
}
