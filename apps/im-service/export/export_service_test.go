package export

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportUserData_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewExportService(db)

	// Mock messages query
	messageRows := sqlmock.NewRows([]string{
		"msg_id", "conversation_type", "conversation_id", "sender_id",
		"recipient_id", "content", "timestamp", "sequence_number",
		"delivered", "read_at",
	}).
		AddRow("msg1", "private", "conv1", "user123", "user456", "Hello", time.Now(), 1, true, time.Now()).
		AddRow("msg2", "private", "conv1", "user456", "user123", "Hi", time.Now(), 2, true, nil)

	mock.ExpectQuery("SELECT msg_id, conversation_type").
		WithArgs("user123", "user123").
		WillReturnRows(messageRows)

	// Mock statistics queries
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE sender_id").
		WithArgs("user123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE user_id").
		WithArgs("user123", "user123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(20))

	mock.ExpectQuery("SELECT COUNT\\(DISTINCT conversation_id\\)").
		WithArgs("user123", "user123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	ctx := context.Background()
	export, err := service.ExportUserData(ctx, "user123")

	require.NoError(t, err)
	assert.NotNil(t, export)
	assert.Equal(t, "user123", export.UserID)
	assert.WithinDuration(t, time.Now(), export.ExportDate, time.Second)

	// Verify messages
	messages, ok := export.Data["messages"].([]Message)
	require.True(t, ok)
	assert.Len(t, messages, 2)
	assert.Equal(t, "msg1", messages[0].MsgID)
	assert.Equal(t, "Hello", messages[0].Content)
	assert.NotNil(t, messages[0].ReadAt)
	assert.Nil(t, messages[1].ReadAt)

	// Verify statistics
	stats, ok := export.Data["statistics"].(*Statistics)
	require.True(t, ok)
	assert.Equal(t, 10, stats.TotalMessagesSent)
	assert.Equal(t, 20, stats.TotalMessagesReceived)
	assert.Equal(t, 5, stats.TotalConversations)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExportUserData_EmptyUserID(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewExportService(db)

	ctx := context.Background()
	export, err := service.ExportUserData(ctx, "")

	assert.Error(t, err)
	assert.Nil(t, export)
	assert.Equal(t, ErrUserIDRequired, err)
}

func TestExportToJSON_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewExportService(db)

	// Mock messages query
	messageRows := sqlmock.NewRows([]string{
		"msg_id", "conversation_type", "conversation_id", "sender_id",
		"recipient_id", "content", "timestamp", "sequence_number",
		"delivered", "read_at",
	}).
		AddRow("msg1", "private", "conv1", "user123", "user456", "Hello", time.Now(), 1, true, nil)

	mock.ExpectQuery("SELECT msg_id, conversation_type").
		WithArgs("user123", "user123").
		WillReturnRows(messageRows)

	// Mock statistics queries
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE sender_id").
		WithArgs("user123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE user_id").
		WithArgs("user123", "user123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery("SELECT COUNT\\(DISTINCT conversation_id\\)").
		WithArgs("user123", "user123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	ctx := context.Background()
	jsonStr, err := service.ExportToJSON(ctx, "user123")

	require.NoError(t, err)
	assert.NotEmpty(t, jsonStr)

	// Verify JSON is valid
	var export UserExport
	err = json.Unmarshal([]byte(jsonStr), &export)
	require.NoError(t, err)
	assert.Equal(t, "user123", export.UserID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExportMessages_NoMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewExportService(db)

	// Mock empty result
	messageRows := sqlmock.NewRows([]string{
		"msg_id", "conversation_type", "conversation_id", "sender_id",
		"recipient_id", "content", "timestamp", "sequence_number",
		"delivered", "read_at",
	})

	mock.ExpectQuery("SELECT msg_id, conversation_type").
		WithArgs("user123", "user123").
		WillReturnRows(messageRows)

	ctx := context.Background()
	messages, err := service.exportMessages(ctx, "user123")

	require.NoError(t, err)
	assert.Empty(t, messages)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExportMessages_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewExportService(db)

	mock.ExpectQuery("SELECT msg_id, conversation_type").
		WithArgs("user123", "user123").
		WillReturnError(assert.AnError)

	ctx := context.Background()
	messages, err := service.exportMessages(ctx, "user123")

	assert.Error(t, err)
	assert.Nil(t, messages)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExportStatistics_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewExportService(db)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE sender_id").
		WithArgs("user123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE user_id").
		WithArgs("user123", "user123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

	mock.ExpectQuery("SELECT COUNT\\(DISTINCT conversation_id\\)").
		WithArgs("user123", "user123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(8))

	ctx := context.Background()
	stats, err := service.exportStatistics(ctx, "user123")

	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 15, stats.TotalMessagesSent)
	assert.Equal(t, 25, stats.TotalMessagesReceived)
	assert.Equal(t, 8, stats.TotalConversations)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExportStatistics_NoData(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewExportService(db)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE sender_id").
		WithArgs("user123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE user_id").
		WithArgs("user123", "user123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery("SELECT COUNT\\(DISTINCT conversation_id\\)").
		WithArgs("user123", "user123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	ctx := context.Background()
	stats, err := service.exportStatistics(ctx, "user123")

	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalMessagesSent)
	assert.Equal(t, 0, stats.TotalMessagesReceived)
	assert.Equal(t, 0, stats.TotalConversations)
	assert.NoError(t, mock.ExpectationsWereMet())
}
