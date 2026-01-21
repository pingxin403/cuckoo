package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	// ErrNotFound is returned when a mapping is not found
	ErrNotFound = errors.New("mapping not found")
)

// URLMapping represents a URL mapping entity
type URLMapping struct {
	ShortCode  string
	LongURL    string
	CreatedAt  time.Time
	ExpiresAt  *time.Time
	CreatorIP  string
	ClickCount int64
	IsDeleted  bool
}

// Storage defines the interface for storage operations
type Storage interface {
	// Create stores a new URL mapping
	Create(ctx context.Context, mapping *URLMapping) error

	// Get retrieves a URL mapping by short code
	Get(ctx context.Context, shortCode string) (*URLMapping, error)

	// Exists checks if a short code already exists
	Exists(ctx context.Context, shortCode string) (bool, error)

	// Delete removes a URL mapping (soft delete)
	Delete(ctx context.Context, shortCode string) error

	// GetExpired returns expired mappings for cleanup
	GetExpired(ctx context.Context, limit int) ([]*URLMapping, error)

	// Close closes the database connection
	Close() error
}

// MySQLStore implements Storage using MySQL
type MySQLStore struct {
	db *sql.DB
}

// NewMySQLStore creates a new MySQL store with connection pooling
func NewMySQLStore() (*MySQLStore, error) {
	// Read configuration from environment variables
	host := getEnv("MYSQL_HOST", "localhost")
	port := getEnv("MYSQL_PORT", "3306")
	database := getEnv("MYSQL_DATABASE", "shortener")
	user := getEnv("MYSQL_USER", "root")
	password := getEnv("MYSQL_PASSWORD", "")

	// Build DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		user, password, host, port, database)

	// Open database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &MySQLStore{db: db}, nil
}

// Create stores a new URL mapping using parameterized queries
func (s *MySQLStore) Create(ctx context.Context, mapping *URLMapping) error {
	if mapping == nil {
		return fmt.Errorf("mapping cannot be nil")
	}

	query := `
		INSERT INTO url_mappings (short_code, long_url, created_at, expires_at, creator_ip, click_count, is_deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	var expiresAt interface{}
	if mapping.ExpiresAt != nil {
		expiresAt = mapping.ExpiresAt.Unix()
	}

	_, err := s.db.ExecContext(ctx, query,
		mapping.ShortCode,
		mapping.LongURL,
		mapping.CreatedAt.Unix(),
		expiresAt,
		mapping.CreatorIP,
		mapping.ClickCount,
		mapping.IsDeleted,
	)

	if err != nil {
		return fmt.Errorf("failed to create mapping: %w", err)
	}

	return nil
}

// Get retrieves a URL mapping by short code
func (s *MySQLStore) Get(ctx context.Context, shortCode string) (*URLMapping, error) {
	query := `
		SELECT short_code, long_url, created_at, expires_at, creator_ip, click_count, is_deleted
		FROM url_mappings
		WHERE short_code = ? AND is_deleted = FALSE
	`

	mapping := &URLMapping{}
	var createdAtUnix int64
	var expiresAtUnix sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, shortCode).Scan(
		&mapping.ShortCode,
		&mapping.LongURL,
		&createdAtUnix,
		&expiresAtUnix,
		&mapping.CreatorIP,
		&mapping.ClickCount,
		&mapping.IsDeleted,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get mapping: %w", err)
	}

	// Convert Unix timestamps to time.Time
	mapping.CreatedAt = time.Unix(createdAtUnix, 0)
	if expiresAtUnix.Valid {
		expiresAt := time.Unix(expiresAtUnix.Int64, 0)
		mapping.ExpiresAt = &expiresAt
	}

	return mapping, nil
}

// Exists checks if a short code already exists (for collision detection)
func (s *MySQLStore) Exists(ctx context.Context, shortCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM url_mappings WHERE short_code = ?)`

	var exists bool
	err := s.db.QueryRowContext(ctx, query, shortCode).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return exists, nil
}

// Delete performs a soft delete on a URL mapping
func (s *MySQLStore) Delete(ctx context.Context, shortCode string) error {
	query := `UPDATE url_mappings SET is_deleted = TRUE WHERE short_code = ?`

	result, err := s.db.ExecContext(ctx, query, shortCode)
	if err != nil {
		return fmt.Errorf("failed to delete mapping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetExpired returns expired mappings for cleanup
func (s *MySQLStore) GetExpired(ctx context.Context, limit int) ([]*URLMapping, error) {
	query := `
		SELECT short_code, long_url, created_at, expires_at, creator_ip, click_count, is_deleted
		FROM url_mappings
		WHERE expires_at IS NOT NULL AND expires_at < ? AND is_deleted = FALSE
		LIMIT ?
	`

	now := time.Now().Unix()
	rows, err := s.db.QueryContext(ctx, query, now, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired mappings: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var mappings []*URLMapping
	for rows.Next() {
		mapping := &URLMapping{}
		var createdAtUnix int64
		var expiresAtUnix sql.NullInt64

		err := rows.Scan(
			&mapping.ShortCode,
			&mapping.LongURL,
			&createdAtUnix,
			&expiresAtUnix,
			&mapping.CreatorIP,
			&mapping.ClickCount,
			&mapping.IsDeleted,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mapping: %w", err)
		}

		mapping.CreatedAt = time.Unix(createdAtUnix, 0)
		if expiresAtUnix.Valid {
			expiresAt := time.Unix(expiresAtUnix.Int64, 0)
			mapping.ExpiresAt = &expiresAt
		}

		mappings = append(mappings, mapping)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return mappings, nil
}

// Close closes the database connection
func (s *MySQLStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
