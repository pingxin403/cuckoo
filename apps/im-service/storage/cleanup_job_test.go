package storage

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockStatsLogger struct {
	stats []CleanupStats
}

func (m *MockStatsLogger) LogCleanupStats(stats CleanupStats) {
	m.stats = append(m.stats, stats)
}

func TestNewCleanupJob(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	store := &OfflineStore{db: db}

	t.Run("with default config", func(t *testing.T) {
		job := NewCleanupJob(store, CleanupJobConfig{})
		assert.NotNil(t, job)
		assert.Equal(t, 10000, job.batchSize)
		assert.Equal(t, 1*time.Hour, job.interval)
		assert.NotNil(t, job.statsLogger)
	})

	t.Run("with custom config", func(t *testing.T) {
		logger := &MockStatsLogger{}
		config := CleanupJobConfig{
			BatchSize:   5000,
			Interval:    30 * time.Minute,
			StatsLogger: logger,
		}
		job := NewCleanupJob(store, config)
		assert.NotNil(t, job)
		assert.Equal(t, 5000, job.batchSize)
		assert.Equal(t, 30*time.Minute, job.interval)
		assert.Equal(t, logger, job.statsLogger)
	})
}

func TestRunOnce(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	store := &OfflineStore{db: db}
	logger := &MockStatsLogger{}

	job := NewCleanupJob(store, CleanupJobConfig{
		BatchSize:   1000,
		StatsLogger: logger,
	})

	t.Run("successful cleanup", func(t *testing.T) {
		deletedCount := int64(500)
		remainingCount := int64(100)
		oldestTime := time.Now().Add(-8 * 24 * time.Hour)

		// Expect delete query
		mock.ExpectExec("DELETE FROM offline_messages").
			WithArgs(1000).
			WillReturnResult(sqlmock.NewResult(0, deletedCount))

		// Expect count query
		rows := sqlmock.NewRows([]string{"count"}).AddRow(remainingCount)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE expires_at").
			WillReturnRows(rows)

		// Expect oldest message query
		oldestRows := sqlmock.NewRows([]string{"expires_at"}).AddRow(oldestTime)
		mock.ExpectQuery("SELECT MIN\\(expires_at\\) FROM offline_messages WHERE expires_at").
			WillReturnRows(oldestRows)

		stats := job.RunOnce()

		assert.NoError(t, stats.Error)
		assert.Equal(t, deletedCount, stats.MessagesDeleted)
		assert.Equal(t, remainingCount, stats.RemainingExpired)
		assert.NotNil(t, stats.OldestExpiredTime)
		assert.WithinDuration(t, oldestTime, *stats.OldestExpiredTime, time.Second)
		assert.Greater(t, stats.Duration, time.Duration(0))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no expired messages", func(t *testing.T) {
		// Expect delete query
		mock.ExpectExec("DELETE FROM offline_messages").
			WithArgs(1000).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Expect count query
		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE expires_at").
			WillReturnRows(rows)

		// Expect oldest message query (returns NULL)
		oldestRows := sqlmock.NewRows([]string{"expires_at"}).AddRow(nil)
		mock.ExpectQuery("SELECT MIN\\(expires_at\\) FROM offline_messages WHERE expires_at").
			WillReturnRows(oldestRows)

		stats := job.RunOnce()

		assert.NoError(t, stats.Error)
		assert.Equal(t, int64(0), stats.MessagesDeleted)
		assert.Equal(t, int64(0), stats.RemainingExpired)
		assert.Nil(t, stats.OldestExpiredTime)
	})

	t.Run("delete failure", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM offline_messages").
			WithArgs(1000).
			WillReturnError(assert.AnError)

		stats := job.RunOnce()

		assert.Error(t, stats.Error)
		assert.Contains(t, stats.Error.Error(), "failed to delete expired messages")
	})

	t.Run("count failure", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM offline_messages").
			WithArgs(1000).
			WillReturnResult(sqlmock.NewResult(0, 100))

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE expires_at").
			WillReturnError(assert.AnError)

		stats := job.RunOnce()

		assert.Error(t, stats.Error)
		assert.Contains(t, stats.Error.Error(), "failed to count remaining expired messages")
	})

	t.Run("oldest message query failure", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM offline_messages").
			WithArgs(1000).
			WillReturnResult(sqlmock.NewResult(0, 100))

		rows := sqlmock.NewRows([]string{"count"}).AddRow(50)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE expires_at").
			WillReturnRows(rows)

		mock.ExpectQuery("SELECT MIN\\(expires_at\\) FROM offline_messages WHERE expires_at").
			WillReturnError(assert.AnError)

		stats := job.RunOnce()

		assert.Error(t, stats.Error)
		assert.Contains(t, stats.Error.Error(), "failed to get oldest expired message")
	})
}

func TestStartStop(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	store := &OfflineStore{db: db}
	logger := &MockStatsLogger{}

	job := NewCleanupJob(store, CleanupJobConfig{
		BatchSize:   1000,
		Interval:    100 * time.Millisecond, // Short interval for testing
		StatsLogger: logger,
	})

	// Expect at least one cleanup run
	mock.ExpectExec("DELETE FROM offline_messages").
		WillReturnResult(sqlmock.NewResult(0, 10))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE expires_at").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT MIN\\(expires_at\\) FROM offline_messages WHERE expires_at").
		WillReturnRows(sqlmock.NewRows([]string{"expires_at"}).AddRow(nil))

	// Start job in goroutine
	go job.Start()

	// Wait for at least one run
	time.Sleep(150 * time.Millisecond)

	// Stop job
	job.Stop()

	// Verify at least one cleanup was logged
	assert.GreaterOrEqual(t, len(logger.stats), 1)
}

func TestDefaultStatsLogger(t *testing.T) {
	logger := &DefaultStatsLogger{}

	t.Run("log successful cleanup", func(t *testing.T) {
		oldestTime := time.Now().Add(-8 * 24 * time.Hour)
		stats := CleanupStats{
			RunTime:           time.Now(),
			MessagesDeleted:   100,
			RemainingExpired:  50,
			OldestExpiredTime: &oldestTime,
			Duration:          500 * time.Millisecond,
		}

		// Should not panic
		logger.LogCleanupStats(stats)
	})

	t.Run("log cleanup with error", func(t *testing.T) {
		stats := CleanupStats{
			RunTime: time.Now(),
			Error:   assert.AnError,
		}

		// Should not panic
		logger.LogCleanupStats(stats)
	})

	t.Run("log cleanup without oldest message", func(t *testing.T) {
		stats := CleanupStats{
			RunTime:          time.Now(),
			MessagesDeleted:  0,
			RemainingExpired: 0,
			Duration:         100 * time.Millisecond,
		}

		// Should not panic
		logger.LogCleanupStats(stats)
	})
}
