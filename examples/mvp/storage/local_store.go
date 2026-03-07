package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// LocalMessage represents a message in the local storage
type LocalMessage struct {
	ID               int64             `json:"id"`
	MsgID            string            `json:"msg_id"`
	UserID           string            `json:"user_id"`
	SenderID         string            `json:"sender_id"`
	ConversationID   string            `json:"conversation_id"`
	ConversationType string            `json:"conversation_type"` // "private" or "group"
	Content          string            `json:"content"`
	SequenceNumber   int64             `json:"sequence_number"`
	Timestamp        int64             `json:"timestamp"`
	CreatedAt        time.Time         `json:"created_at"`
	ExpiresAt        time.Time         `json:"expires_at"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	RegionID         string            `json:"region_id"`
	GlobalID         string            `json:"global_id"` // HLC-based global ID
	Version          int64             `json:"version"`   // For conflict resolution
}

// ConflictInfo represents information about a detected conflict
type ConflictInfo struct {
	MessageID     string    `json:"message_id"`
	LocalVersion  int64     `json:"local_version"`
	RemoteVersion int64     `json:"remote_version"`
	LocalRegion   string    `json:"local_region"`
	RemoteRegion  string    `json:"remote_region"`
	ConflictTime  time.Time `json:"conflict_time"`
	Resolution    string    `json:"resolution"` // "local_wins", "remote_wins"
}

// LocalStore provides simplified storage for multi-region active-active MVP
type LocalStore struct {
	db       *sql.DB
	regionID string
	mu       sync.RWMutex

	// In-memory fallback for testing
	memoryMode bool
	messages   map[string]*LocalMessage
	conflicts  []ConflictInfo
}

// Config holds local store configuration
type Config struct {
	DatabasePath string        // SQLite database file path
	RegionID     string        // Current region identifier
	WALMode      bool          // Enable WAL mode for SQLite
	MemoryMode   bool          // Use in-memory storage instead of SQLite
	TTL          time.Duration // Message TTL (default 7 days)
}

// NewLocalStore creates a new local storage instance
func NewLocalStore(config Config) (*LocalStore, error) {
	store := &LocalStore{
		regionID:   config.RegionID,
		memoryMode: config.MemoryMode,
		messages:   make(map[string]*LocalMessage),
		conflicts:  make([]ConflictInfo, 0),
	}

	if config.MemoryMode {
		// Use in-memory storage for testing/development
		return store, nil
	}

	// Initialize SQLite database
	dbPath := config.DatabasePath
	if dbPath == "" {
		dbPath = fmt.Sprintf("./data/%s.db", config.RegionID)
	}

	var err error
	store.db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Configure SQLite for multi-region use
	if err := store.configureSQLite(config.WALMode); err != nil {
		return nil, fmt.Errorf("failed to configure SQLite: %w", err)
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// configureSQLite sets up SQLite for optimal multi-region performance
func (s *LocalStore) configureSQLite(walMode bool) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA synchronous = NORMAL", // Balance between safety and performance
		"PRAGMA cache_size = -64000",  // 64MB cache
		"PRAGMA temp_store = MEMORY",
		"PRAGMA mmap_size = 268435456", // 256MB memory map
	}

	if walMode {
		pragmas = append(pragmas, "PRAGMA journal_mode = WAL")
		pragmas = append(pragmas, "PRAGMA wal_autocheckpoint = 1000")
	}

	for _, pragma := range pragmas {
		if _, err := s.db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute pragma %s: %w", pragma, err)
		}
	}

	return nil
}

// initSchema creates the necessary tables
func (s *LocalStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		msg_id TEXT UNIQUE NOT NULL,
		user_id TEXT NOT NULL,
		sender_id TEXT NOT NULL,
		conversation_id TEXT NOT NULL,
		conversation_type TEXT NOT NULL,
		content TEXT NOT NULL,
		sequence_number INTEGER NOT NULL,
		timestamp INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		metadata TEXT, -- JSON string
		region_id TEXT NOT NULL,
		global_id TEXT NOT NULL,
		version INTEGER NOT NULL DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_user_timestamp ON messages(user_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_conversation_seq ON messages(conversation_id, sequence_number);
	CREATE INDEX IF NOT EXISTS idx_expires_at ON messages(expires_at);
	CREATE INDEX IF NOT EXISTS idx_global_id ON messages(global_id);
	CREATE INDEX IF NOT EXISTS idx_region_id ON messages(region_id);

	CREATE TABLE IF NOT EXISTS conflicts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id TEXT NOT NULL,
		local_version INTEGER NOT NULL,
		remote_version INTEGER NOT NULL,
		local_region TEXT NOT NULL,
		remote_region TEXT NOT NULL,
		conflict_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		resolution TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_conflict_time ON conflicts(conflict_time);
	CREATE INDEX IF NOT EXISTS idx_conflict_message ON conflicts(message_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection
func (s *LocalStore) Close() error {
	if s.memoryMode || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Insert inserts a single message
func (s *LocalStore) Insert(ctx context.Context, message LocalMessage) error {
	if s.memoryMode {
		return s.insertMemory(message)
	}
	return s.insertSQLite(ctx, message)
}

// insertMemory inserts a message into in-memory storage
func (s *LocalStore) insertMemory(message LocalMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicates
	if _, exists := s.messages[message.MsgID]; exists {
		return fmt.Errorf("message %s already exists", message.MsgID)
	}

	// Set defaults
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}
	if message.RegionID == "" {
		message.RegionID = s.regionID
	}
	if message.Version == 0 {
		message.Version = 1
	}

	// Create a copy to avoid pointer issues
	msgCopy := message
	// Set ID for memory mode (simulate auto-increment)
	msgCopy.ID = int64(len(s.messages) + 1)
	s.messages[message.MsgID] = &msgCopy
	return nil
}

// insertSQLite inserts a message into SQLite database
func (s *LocalStore) insertSQLite(ctx context.Context, message LocalMessage) error {
	metadataJSON, err := json.Marshal(message.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO messages (
			msg_id, user_id, sender_id, conversation_id, conversation_type,
			content, sequence_number, timestamp, expires_at, metadata,
			region_id, global_id, version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		message.MsgID,
		message.UserID,
		message.SenderID,
		message.ConversationID,
		message.ConversationType,
		message.Content,
		message.SequenceNumber,
		message.Timestamp,
		message.ExpiresAt,
		string(metadataJSON),
		message.RegionID,
		message.GlobalID,
		message.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	return nil
}

// BatchInsert inserts multiple messages in a transaction
func (s *LocalStore) BatchInsert(ctx context.Context, messages []LocalMessage) error {
	if len(messages) == 0 {
		return nil
	}

	if s.memoryMode {
		return s.batchInsertMemory(messages)
	}
	return s.batchInsertSQLite(ctx, messages)
}

// batchInsertMemory inserts multiple messages into memory
func (s *LocalStore) batchInsertMemory(messages []LocalMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicates first
	for _, msg := range messages {
		if _, exists := s.messages[msg.MsgID]; exists {
			return fmt.Errorf("message %s already exists", msg.MsgID)
		}
	}

	// Insert all messages
	for i, msg := range messages {
		if msg.CreatedAt.IsZero() {
			msg.CreatedAt = time.Now()
		}
		if msg.RegionID == "" {
			msg.RegionID = s.regionID
		}
		if msg.Version == 0 {
			msg.Version = 1
		}
		// Create a copy to avoid pointer issues
		msgCopy := msg
		s.messages[msg.MsgID] = &msgCopy

		// Set ID for memory mode (simulate auto-increment)
		s.messages[msg.MsgID].ID = int64(len(s.messages) + i)
	}

	return nil
}

// batchInsertSQLite inserts multiple messages into SQLite
func (s *LocalStore) batchInsertSQLite(ctx context.Context, messages []LocalMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO messages (
			msg_id, user_id, sender_id, conversation_id, conversation_type,
			content, sequence_number, timestamp, expires_at, metadata,
			region_id, global_id, version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, msg := range messages {
		metadataJSON, err := json.Marshal(msg.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata for message %s: %w", msg.MsgID, err)
		}

		_, err = stmt.ExecContext(ctx,
			msg.MsgID,
			msg.UserID,
			msg.SenderID,
			msg.ConversationID,
			msg.ConversationType,
			msg.Content,
			msg.SequenceNumber,
			msg.Timestamp,
			msg.ExpiresAt,
			string(metadataJSON),
			msg.RegionID,
			msg.GlobalID,
			msg.Version,
		)
		if err != nil {
			return fmt.Errorf("failed to insert message %s: %w", msg.MsgID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetMessages retrieves messages for a user with pagination
func (s *LocalStore) GetMessages(ctx context.Context, userID string, cursor int64, limit int) ([]LocalMessage, error) {
	if limit <= 0 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	if s.memoryMode {
		return s.getMessagesMemory(userID, cursor, limit)
	}
	return s.getMessagesSQLite(ctx, userID, cursor, limit)
}

// getMessagesMemory retrieves messages from memory
func (s *LocalStore) getMessagesMemory(userID string, cursor int64, limit int) ([]LocalMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var messages []LocalMessage
	for _, msg := range s.messages {
		if msg.UserID == userID && msg.ID > cursor {
			messages = append(messages, *msg)
		}
	}

	// Sort by sequence number
	for i := 0; i < len(messages)-1; i++ {
		for j := i + 1; j < len(messages); j++ {
			if messages[i].SequenceNumber > messages[j].SequenceNumber {
				messages[i], messages[j] = messages[j], messages[i]
			}
		}
	}

	// Apply limit
	if len(messages) > limit {
		messages = messages[:limit]
	}

	return messages, nil
}

// getMessagesSQLite retrieves messages from SQLite
func (s *LocalStore) getMessagesSQLite(ctx context.Context, userID string, cursor int64, limit int) ([]LocalMessage, error) {
	query := `
		SELECT 
			id, msg_id, user_id, sender_id, conversation_id, conversation_type,
			content, sequence_number, timestamp, created_at, expires_at,
			metadata, region_id, global_id, version
		FROM messages
		WHERE user_id = ? AND id > ?
		ORDER BY sequence_number ASC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, userID, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var messages []LocalMessage
	for rows.Next() {
		var msg LocalMessage
		var metadataJSON string

		err := rows.Scan(
			&msg.ID,
			&msg.MsgID,
			&msg.UserID,
			&msg.SenderID,
			&msg.ConversationID,
			&msg.ConversationType,
			&msg.Content,
			&msg.SequenceNumber,
			&msg.Timestamp,
			&msg.CreatedAt,
			&msg.ExpiresAt,
			&metadataJSON,
			&msg.RegionID,
			&msg.GlobalID,
			&msg.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Parse metadata JSON
		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &msg.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// GetMessageByID retrieves a message by its ID
func (s *LocalStore) GetMessageByID(ctx context.Context, msgID string) (*LocalMessage, error) {
	if s.memoryMode {
		return s.getMessageByIDMemory(msgID)
	}
	return s.getMessageByIDSQLite(ctx, msgID)
}

// getMessageByIDMemory retrieves a message from memory by ID
func (s *LocalStore) getMessageByIDMemory(msgID string) (*LocalMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msg, exists := s.messages[msgID]
	if !exists {
		return nil, fmt.Errorf("message %s not found", msgID)
	}

	// Return a copy to prevent external modification
	msgCopy := *msg
	return &msgCopy, nil
}

// getMessageByIDSQLite retrieves a message from SQLite by ID
func (s *LocalStore) getMessageByIDSQLite(ctx context.Context, msgID string) (*LocalMessage, error) {
	query := `
		SELECT 
			id, msg_id, user_id, sender_id, conversation_id, conversation_type,
			content, sequence_number, timestamp, created_at, expires_at,
			metadata, region_id, global_id, version
		FROM messages
		WHERE msg_id = ?
	`

	row := s.db.QueryRowContext(ctx, query, msgID)

	var msg LocalMessage
	var metadataJSON string

	err := row.Scan(
		&msg.ID,
		&msg.MsgID,
		&msg.UserID,
		&msg.SenderID,
		&msg.ConversationID,
		&msg.ConversationType,
		&msg.Content,
		&msg.SequenceNumber,
		&msg.Timestamp,
		&msg.CreatedAt,
		&msg.ExpiresAt,
		&metadataJSON,
		&msg.RegionID,
		&msg.GlobalID,
		&msg.Version,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message %s not found", msgID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query message: %w", err)
	}

	// Parse metadata JSON
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &msg.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &msg, nil
}

// DetectConflict checks if a message conflicts with existing data
func (s *LocalStore) DetectConflict(ctx context.Context, remoteMessage LocalMessage) (*ConflictInfo, error) {
	localMessage, err := s.GetMessageByID(ctx, remoteMessage.MsgID)
	if err != nil {
		// Message doesn't exist locally, no conflict
		return nil, nil
	}

	// Check if versions differ
	if localMessage.Version != remoteMessage.Version ||
		localMessage.RegionID != remoteMessage.RegionID {
		conflict := &ConflictInfo{
			MessageID:     remoteMessage.MsgID,
			LocalVersion:  localMessage.Version,
			RemoteVersion: remoteMessage.Version,
			LocalRegion:   localMessage.RegionID,
			RemoteRegion:  remoteMessage.RegionID,
			ConflictTime:  time.Now(),
		}

		// Simple LWW resolution based on global ID comparison
		if remoteMessage.GlobalID > localMessage.GlobalID {
			conflict.Resolution = "remote_wins"
		} else {
			conflict.Resolution = "local_wins"
		}

		return conflict, nil
	}

	return nil, nil
}

// RecordConflict records a conflict for monitoring
func (s *LocalStore) RecordConflict(ctx context.Context, conflict ConflictInfo) error {
	if s.memoryMode {
		s.mu.Lock()
		s.conflicts = append(s.conflicts, conflict)
		s.mu.Unlock()
		return nil
	}

	query := `
		INSERT INTO conflicts (
			message_id, local_version, remote_version, 
			local_region, remote_region, resolution
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		conflict.MessageID,
		conflict.LocalVersion,
		conflict.RemoteVersion,
		conflict.LocalRegion,
		conflict.RemoteRegion,
		conflict.Resolution,
	)

	if err != nil {
		return fmt.Errorf("failed to record conflict: %w", err)
	}

	return nil
}

// GetConflicts retrieves recent conflicts for monitoring
func (s *LocalStore) GetConflicts(ctx context.Context, limit int) ([]ConflictInfo, error) {
	if s.memoryMode {
		s.mu.RLock()
		defer s.mu.RUnlock()

		conflicts := make([]ConflictInfo, len(s.conflicts))
		copy(conflicts, s.conflicts)

		// Apply limit
		if len(conflicts) > limit {
			conflicts = conflicts[len(conflicts)-limit:]
		}

		return conflicts, nil
	}

	query := `
		SELECT 
			message_id, local_version, remote_version,
			local_region, remote_region, conflict_time, resolution
		FROM conflicts
		ORDER BY conflict_time DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query conflicts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var conflicts []ConflictInfo
	for rows.Next() {
		var conflict ConflictInfo
		err := rows.Scan(
			&conflict.MessageID,
			&conflict.LocalVersion,
			&conflict.RemoteVersion,
			&conflict.LocalRegion,
			&conflict.RemoteRegion,
			&conflict.ConflictTime,
			&conflict.Resolution,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conflict: %w", err)
		}
		conflicts = append(conflicts, conflict)
	}

	return conflicts, nil
}

// DeleteExpiredMessages removes expired messages
func (s *LocalStore) DeleteExpiredMessages(ctx context.Context, batchSize int) (int64, error) {
	if s.memoryMode {
		return s.deleteExpiredMemory(), nil
	}
	return s.deleteExpiredSQLite(ctx, batchSize)
}

// deleteExpiredMemory removes expired messages from memory
func (s *LocalStore) deleteExpiredMemory() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	deleted := int64(0)

	for msgID, msg := range s.messages {
		if now.After(msg.ExpiresAt) {
			delete(s.messages, msgID)
			deleted++
		}
	}

	return deleted
}

// deleteExpiredSQLite removes expired messages from SQLite
func (s *LocalStore) deleteExpiredSQLite(ctx context.Context, batchSize int) (int64, error) {
	// Use proper datetime comparison for SQLite
	query := `DELETE FROM messages WHERE datetime(expires_at) < datetime('now')`
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired messages: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// GetStats returns storage statistics
func (s *LocalStore) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	stats["region_id"] = s.regionID
	stats["memory_mode"] = s.memoryMode

	if s.memoryMode {
		s.mu.RLock()
		stats["total_messages"] = len(s.messages)
		stats["total_conflicts"] = len(s.conflicts)
		s.mu.RUnlock()
	} else {
		// Get message count
		var messageCount int64
		err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM messages").Scan(&messageCount)
		if err != nil {
			return nil, fmt.Errorf("failed to count messages: %w", err)
		}
		stats["total_messages"] = messageCount

		// Get conflict count
		var conflictCount int64
		err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM conflicts").Scan(&conflictCount)
		if err != nil {
			return nil, fmt.Errorf("failed to count conflicts: %w", err)
		}
		stats["total_conflicts"] = conflictCount

		// Get expired message count
		var expiredCount int64
		err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM messages WHERE expires_at < datetime('now')").Scan(&expiredCount)
		if err != nil {
			return nil, fmt.Errorf("failed to count expired messages: %w", err)
		}
		stats["expired_messages"] = expiredCount
	}

	return stats, nil
}
