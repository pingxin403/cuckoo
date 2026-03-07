package capacity

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// RetentionPolicy 数据保留策略
type RetentionPolicy struct {
	MessageType  string        `yaml:"message_type"`
	HotTTL       time.Duration `yaml:"hot_ttl"`       // 热存储保留时间
	ArchiveAfter time.Duration `yaml:"archive_after"` // 归档时间
}

// ArchiveResult 归档结果
type ArchiveResult struct {
	ArchivedCount int           `json:"archived_count"`
	FailedCount   int           `json:"failed_count"`
	Duration      time.Duration `json:"duration"`
	Errors        []string      `json:"errors,omitempty"`
}

// LifecycleManager 数据生命周期管理器
type LifecycleManager struct {
	regionID  string
	policies  []RetentionPolicy
	db        *sql.DB // 热存储数据库
	archiveDB *sql.DB // 冷存储/归档数据库
}

// NewLifecycleManager 创建生命周期管理器
func NewLifecycleManager(regionID string, policies []RetentionPolicy, db, archiveDB *sql.DB) *LifecycleManager {
	return &LifecycleManager{
		regionID:  regionID,
		policies:  policies,
		db:        db,
		archiveDB: archiveDB,
	}
}

// ArchiveExpiredMessages 归档过期消息（按批次处理）
func (lm *LifecycleManager) ArchiveExpiredMessages(ctx context.Context, batchSize int) (*ArchiveResult, error) {
	startTime := time.Now()
	result := &ArchiveResult{}

	for _, policy := range lm.policies {
		archived, failed, errs := lm.archiveByPolicy(ctx, policy, batchSize)
		result.ArchivedCount += archived
		result.FailedCount += failed
		result.Errors = append(result.Errors, errs...)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// archiveByPolicy 按策略归档消息
func (lm *LifecycleManager) archiveByPolicy(ctx context.Context, policy RetentionPolicy, batchSize int) (int, int, []string) {
	var archivedCount, failedCount int
	var errors []string

	// 计算过期时间
	expiredBefore := time.Now().Add(-policy.ArchiveAfter)

	// 查询需要归档的消息
	query := `
		SELECT id, user_id, content, timestamp, expires_at, region_id, global_id, sync_status
		FROM offline_messages
		WHERE timestamp < ? AND region_id = ?
		LIMIT ?
	`

	rows, err := lm.db.QueryContext(ctx, query, expiredBefore.Unix(), lm.regionID, batchSize)
	if err != nil {
		errors = append(errors, fmt.Sprintf("query failed: %v", err))
		return 0, 0, errors
	}
	defer rows.Close()

	// 开始事务
	tx, err := lm.db.BeginTx(ctx, nil)
	if err != nil {
		errors = append(errors, fmt.Sprintf("begin transaction failed: %v", err))
		return 0, 0, errors
	}
	defer tx.Rollback()

	archiveTx, err := lm.archiveDB.BeginTx(ctx, nil)
	if err != nil {
		errors = append(errors, fmt.Sprintf("begin archive transaction failed: %v", err))
		return 0, 0, errors
	}
	defer archiveTx.Rollback()

	// 归档消息
	archiveStmt, err := archiveTx.PrepareContext(ctx, `
		INSERT INTO archived_messages 
		(id, user_id, content, timestamp, expires_at, region_id, global_id, sync_status, archived_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		errors = append(errors, fmt.Sprintf("prepare archive statement failed: %v", err))
		return 0, 0, errors
	}
	defer archiveStmt.Close()

	deleteStmt, err := tx.PrepareContext(ctx, `
		DELETE FROM offline_messages WHERE id = ?
	`)
	if err != nil {
		errors = append(errors, fmt.Sprintf("prepare delete statement failed: %v", err))
		return 0, 0, errors
	}
	defer deleteStmt.Close()

	archivedAt := time.Now()
	var messageIDs []string

	for rows.Next() {
		var id, userID, content, regionID, globalID, syncStatus string
		var timestamp, expiresAt int64

		if err := rows.Scan(&id, &userID, &content, &timestamp, &expiresAt, &regionID, &globalID, &syncStatus); err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("scan row failed: %v", err))
			continue
		}

		// 插入到归档表
		_, err := archiveStmt.ExecContext(ctx, id, userID, content, timestamp, expiresAt, regionID, globalID, syncStatus, archivedAt.Unix())
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("archive message %s failed: %v", id, err))
			continue
		}

		messageIDs = append(messageIDs, id)
	}

	if err := rows.Err(); err != nil {
		errors = append(errors, fmt.Sprintf("rows iteration failed: %v", err))
		return 0, failedCount, errors
	}

	// 从热存储删除已归档的消息
	for _, id := range messageIDs {
		_, err := deleteStmt.ExecContext(ctx, id)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("delete message %s failed: %v", id, err))
			continue
		}
		archivedCount++
	}

	// 提交事务
	if err := archiveTx.Commit(); err != nil {
		errors = append(errors, fmt.Sprintf("commit archive transaction failed: %v", err))
		return 0, failedCount, errors
	}

	if err := tx.Commit(); err != nil {
		errors = append(errors, fmt.Sprintf("commit delete transaction failed: %v", err))
		return 0, failedCount, errors
	}

	return archivedCount, failedCount, errors
}

// GetRetentionPolicy 获取指定消息类型的保留策略
func (lm *LifecycleManager) GetRetentionPolicy(messageType string) *RetentionPolicy {
	for _, policy := range lm.policies {
		if policy.MessageType == messageType {
			return &policy
		}
	}
	return nil
}

// ValidateArchiveConsistency 验证归档一致性（用于测试）
func (lm *LifecycleManager) ValidateArchiveConsistency(ctx context.Context, messageID string) (bool, error) {
	// 检查热存储中是否还存在
	var hotCount int
	err := lm.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM offline_messages WHERE id = ?", messageID).Scan(&hotCount)
	if err != nil {
		return false, fmt.Errorf("query hot storage failed: %w", err)
	}

	// 检查冷存储中是否存在
	var coldCount int
	err = lm.archiveDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM archived_messages WHERE id = ?", messageID).Scan(&coldCount)
	if err != nil {
		return false, fmt.Errorf("query cold storage failed: %w", err)
	}

	// 归档一致性：消息应该只存在于一个存储中
	return (hotCount == 0 && coldCount == 1) || (hotCount == 1 && coldCount == 0), nil
}
