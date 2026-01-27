package sequence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// SequenceBackup handles periodic snapshots of sequence numbers to MySQL
type SequenceBackup struct {
	db                *sql.DB
	generator         *SequenceGenerator
	snapshotInterval  int64 // Snapshot every N messages
	snapshotThreshold int64 // Current threshold for next snapshot
}

// NewSequenceBackup creates a new sequence backup manager
func NewSequenceBackup(db *sql.DB, generator *SequenceGenerator, snapshotInterval int64) *SequenceBackup {
	if snapshotInterval <= 0 {
		snapshotInterval = 10000 // Default: snapshot every 10,000 messages
	}

	return &SequenceBackup{
		db:                db,
		generator:         generator,
		snapshotInterval:  snapshotInterval,
		snapshotThreshold: snapshotInterval,
	}
}

// SaveSnapshot saves the current sequence number to MySQL
func (sb *SequenceBackup) SaveSnapshot(ctx context.Context, conversationType ConversationType, conversationID string, sequenceNumber int64) error {
	query := `
		INSERT INTO sequence_snapshots (conversation_type, conversation_id, sequence_number, snapshot_time)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			sequence_number = VALUES(sequence_number),
			snapshot_time = VALUES(snapshot_time),
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := sb.db.ExecContext(ctx, query, string(conversationType), conversationID, sequenceNumber, time.Now())
	if err != nil {
		return fmt.Errorf("failed to save sequence snapshot: %w", err)
	}

	return nil
}

// LoadSnapshot retrieves the last known sequence number from MySQL
func (sb *SequenceBackup) LoadSnapshot(ctx context.Context, conversationType ConversationType, conversationID string) (int64, error) {
	query := `
		SELECT sequence_number
		FROM sequence_snapshots
		WHERE conversation_type = ? AND conversation_id = ?
	`

	var sequenceNumber int64
	err := sb.db.QueryRowContext(ctx, query, string(conversationType), conversationID).Scan(&sequenceNumber)
	if err == sql.ErrNoRows {
		// No snapshot exists, return 0
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to load sequence snapshot: %w", err)
	}

	return sequenceNumber, nil
}

// RecoverSequence recovers sequence numbers from MySQL when Redis fails
// This sets the Redis key to the last known snapshot value
func (sb *SequenceBackup) RecoverSequence(ctx context.Context, conversationType ConversationType, conversationID string) (int64, error) {
	// Load the last snapshot from MySQL
	snapshotSeq, err := sb.LoadSnapshot(ctx, conversationType, conversationID)
	if err != nil {
		return 0, fmt.Errorf("failed to load snapshot for recovery: %w", err)
	}

	// If no snapshot exists, start from 0
	if snapshotSeq == 0 {
		return 0, nil
	}

	// Note: In a real implementation, we would need to set the Redis key to this value
	// This would require a SET operation on Redis, which is not part of the current interface
	// For now, we return the snapshot value and let the caller handle Redis initialization

	return snapshotSeq, nil
}

// ShouldSnapshot checks if a snapshot should be taken based on the current sequence number
func (sb *SequenceBackup) ShouldSnapshot(sequenceNumber int64) bool {
	return sequenceNumber%sb.snapshotInterval == 0
}

// GetSnapshotInterval returns the configured snapshot interval
func (sb *SequenceBackup) GetSnapshotInterval() int64 {
	return sb.snapshotInterval
}

// ListSnapshots retrieves all snapshots for a given conversation type
func (sb *SequenceBackup) ListSnapshots(ctx context.Context, conversationType ConversationType, limit int) ([]SequenceSnapshot, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT conversation_type, conversation_id, sequence_number, snapshot_time
		FROM sequence_snapshots
		WHERE conversation_type = ?
		ORDER BY snapshot_time DESC
		LIMIT ?
	`

	rows, err := sb.db.QueryContext(ctx, query, string(conversationType), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	var snapshots []SequenceSnapshot
	for rows.Next() {
		var snapshot SequenceSnapshot
		var snapshotTime time.Time

		err := rows.Scan(
			&snapshot.ConversationType,
			&snapshot.ConversationID,
			&snapshot.SequenceNumber,
			&snapshotTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}

		snapshot.SnapshotTime = snapshotTime
		snapshots = append(snapshots, snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating snapshots: %w", err)
	}

	return snapshots, nil
}

// SequenceSnapshot represents a snapshot record
type SequenceSnapshot struct {
	ConversationType string
	ConversationID   string
	SequenceNumber   int64
	SnapshotTime     time.Time
}
