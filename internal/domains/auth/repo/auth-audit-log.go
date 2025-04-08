package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/arabkood/backend/internal/postgres"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/google/uuid"
)

// AuditLogType represents the type of audit log entry
type AuditLogType string

const (
	AuditLogTypeSignup         AuditLogType = "signup"
	AuditLogTypeSignin         AuditLogType = "signin"
	AuditLogTypeSignout        AuditLogType = "signout"
	AuditLogTypePasswordChange AuditLogType = "password_change"
	AuditLogTypeEmailChange    AuditLogType = "email_change"
	AuditLogTypeUsernameChange AuditLogType = "username_change"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        int64
	UserID    *uuid.UUID
	Type      AuditLogType
	IPAddress string
	UserAgent string
	CreatedAt time.Time
	Metadata  json.RawMessage
}

type AuditLogsRepository struct {
	querier postgres.Querier
}

func NewAuditLogsRepository(querier postgres.Querier) *AuditLogsRepository {
	return &AuditLogsRepository{querier: querier}
}

// CreateLog creates a new audit log entry
func (r *AuditLogsRepository) CreateLog(ctx context.Context, log *AuditLog) *appErrors.Error {
	query := `
        INSERT INTO auth.audit_logs (
            user_id, type, ip_address, user_agent, metadata
        ) VALUES (
            $1, $2, $3, $4, $5
        )
        RETURNING id, created_at`

	err := r.querier.QueryRow(ctx, query,
		log.UserID,
		log.Type,
		log.IPAddress,
		log.UserAgent,
		log.Metadata,
	).Scan(&log.ID, &log.CreatedAt)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}

// GetUserLogs retrieves audit logs for a specific user
func (r *AuditLogsRepository) GetUserLogs(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*AuditLog, *appErrors.Error) {
	query := `
        SELECT id, user_id, type, ip_address, user_agent, created_at, metadata
        FROM auth.audit_logs
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3`

	rows, err := r.querier.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Type,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
			&log.Metadata,
		)
		if err != nil {
			return nil, appErrors.ErrorInternal().WithError(err)
		}
		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return logs, nil
}

// GetLogsByType retrieves audit logs of a specific type within a date range
func (r *AuditLogsRepository) GetLogsByType(ctx context.Context, logType AuditLogType, startDate, endDate time.Time, limit, offset int) ([]*AuditLog, *appErrors.Error) {
	query := `
        SELECT id, user_id, type, ip_address, user_agent, created_at, metadata
        FROM auth.audit_logs
        WHERE type = $1
        AND created_at BETWEEN $2 AND $3
        ORDER BY created_at DESC
        LIMIT $4 OFFSET $5`

	rows, err := r.querier.Query(ctx, query, logType, startDate, endDate, limit, offset)
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Type,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
			&log.Metadata,
		)
		if err != nil {
			return nil, appErrors.ErrorInternal().WithError(err)
		}
		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return logs, nil
}

// DeleteOldLogs deletes audit logs older than the specified duration
func (r *AuditLogsRepository) DeleteOldLogs(ctx context.Context, olderThan time.Duration) *appErrors.Error {
	query := `
        DELETE FROM auth.audit_logs
        WHERE created_at < $1`

	cutoff := time.Now().Add(-olderThan)
	_, err := r.querier.Exec(ctx, query, cutoff)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}
