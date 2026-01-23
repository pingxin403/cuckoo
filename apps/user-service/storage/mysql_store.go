package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pingxin403/cuckoo/apps/user-service/gen/userpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserStore defines the interface for user storage operations
type UserStore interface {
	// User profile operations
	GetUser(ctx context.Context, userID string) (*userpb.UserProfile, error)
	BatchGetUsers(ctx context.Context, userIDs []string) (map[string]*userpb.UserProfile, error)

	// Group membership operations
	GetGroupMembers(ctx context.Context, groupID string, cursor string, limit int32) ([]*userpb.GroupMember, string, int32, error)
	ValidateGroupMembership(ctx context.Context, userID, groupID string) (*userpb.GroupMember, error)
}

// MySQLStore implements UserStore using MySQL database
type MySQLStore struct {
	db *sql.DB
}

// NewMySQLStore creates a new MySQL-backed user store
func NewMySQLStore(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &MySQLStore{db: db}, nil
}

// Close closes the database connection
func (s *MySQLStore) Close() error {
	return s.db.Close()
}

// GetUser retrieves a single user's profile
func (s *MySQLStore) GetUser(ctx context.Context, userID string) (*userpb.UserProfile, error) {
	query := `
		SELECT user_id, username, display_name, avatar_url, status, created_at, updated_at
		FROM users
		WHERE user_id = ?
	`

	var user userpb.UserProfile
	var status int32
	var createdAt, updatedAt time.Time

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.UserId,
		&user.Username,
		&user.DisplayName,
		&user.AvatarUrl,
		&status,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	user.Status = userpb.UserStatus(status)
	user.CreatedAt = timestamppb.New(createdAt)
	user.UpdatedAt = timestamppb.New(updatedAt)

	return &user, nil
}

// BatchGetUsers retrieves multiple users' profiles
func (s *MySQLStore) BatchGetUsers(ctx context.Context, userIDs []string) (map[string]*userpb.UserProfile, error) {
	if len(userIDs) == 0 {
		return make(map[string]*userpb.UserProfile), nil
	}

	// Build query with placeholders using strings.Builder for safety
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT user_id, username, display_name, avatar_url, status, created_at, updated_at
		FROM users
		WHERE user_id IN (?`)

	for i := 1; i < len(userIDs); i++ {
		queryBuilder.WriteString(",?")
	}
	queryBuilder.WriteString(")")

	query := queryBuilder.String()

	// Convert userIDs to any slice
	args := make([]any, len(userIDs))
	for i, id := range userIDs {
		args[i] = id
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	users := make(map[string]*userpb.UserProfile)
	for rows.Next() {
		var user userpb.UserProfile
		var status int32
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&user.UserId,
			&user.Username,
			&user.DisplayName,
			&user.AvatarUrl,
			&status,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		user.Status = userpb.UserStatus(status)
		user.CreatedAt = timestamppb.New(createdAt)
		user.UpdatedAt = timestamppb.New(updatedAt)

		users[user.UserId] = &user
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return users, nil
}

// GetGroupMembers retrieves group members with pagination
func (s *MySQLStore) GetGroupMembers(ctx context.Context, groupID string, cursor string, limit int32) ([]*userpb.GroupMember, string, int32, error) {
	// Default limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	// Get total count
	var totalCount int32
	countQuery := `SELECT COUNT(*) FROM group_members WHERE group_id = ?`
	err := s.db.QueryRowContext(ctx, countQuery, groupID).Scan(&totalCount)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to count group members: %w", err)
	}

	// Build query with cursor-based pagination
	query := `
		SELECT user_id, group_id, role, group_display_name, joined_at, is_muted
		FROM group_members
		WHERE group_id = ?
	`
	args := []any{groupID}

	if cursor != "" {
		query += ` AND user_id > ?`
		args = append(args, cursor)
	}

	query += ` ORDER BY user_id ASC LIMIT ?`
	args = append(args, limit+1) // Fetch one extra to determine if there are more pages

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to query group members: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	members := make([]*userpb.GroupMember, 0, limit)
	for rows.Next() {
		var member userpb.GroupMember
		var role int32
		var joinedAt time.Time
		var groupDisplayName sql.NullString

		err := rows.Scan(
			&member.UserId,
			&member.GroupId,
			&role,
			&groupDisplayName,
			&joinedAt,
			&member.IsMuted,
		)
		if err != nil {
			return nil, "", 0, fmt.Errorf("failed to scan group member: %w", err)
		}

		member.Role = userpb.GroupRole(role)
		member.JoinedAt = timestamppb.New(joinedAt)
		if groupDisplayName.Valid {
			member.GroupDisplayName = groupDisplayName.String
		}

		members = append(members, &member)
	}

	if err := rows.Err(); err != nil {
		return nil, "", 0, fmt.Errorf("error iterating rows: %w", err)
	}

	// Determine next cursor and has_more
	var nextCursor string
	if len(members) > int(limit) {
		// We have more pages
		nextCursor = members[limit-1].UserId
		members = members[:limit] // Trim the extra record
	}

	return members, nextCursor, totalCount, nil
}

// ValidateGroupMembership checks if a user is a member of a group
func (s *MySQLStore) ValidateGroupMembership(ctx context.Context, userID, groupID string) (*userpb.GroupMember, error) {
	query := `
		SELECT user_id, group_id, role, group_display_name, joined_at, is_muted
		FROM group_members
		WHERE user_id = ? AND group_id = ?
	`

	var member userpb.GroupMember
	var role int32
	var joinedAt time.Time
	var groupDisplayName sql.NullString

	err := s.db.QueryRowContext(ctx, query, userID, groupID).Scan(
		&member.UserId,
		&member.GroupId,
		&role,
		&groupDisplayName,
		&joinedAt,
		&member.IsMuted,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not a member
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query group membership: %w", err)
	}

	member.Role = userpb.GroupRole(role)
	member.JoinedAt = timestamppb.New(joinedAt)
	if groupDisplayName.Valid {
		member.GroupDisplayName = groupDisplayName.String
	}

	return &member, nil
}
