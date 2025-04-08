package repo

import (
	"context"
	"time"

	domainToken "github.com/arabkood/backend/internal/domains/auth/interfaces/token"
	"github.com/arabkood/backend/internal/postgres"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SessionTokensRepository struct {
	querier postgres.Querier
}

func NewSessionTokensRepository(querier postgres.Querier) *SessionTokensRepository {
	return &SessionTokensRepository{querier: querier}
}

// StoreToken stores a session token
func (r *SessionTokensRepository) StoreToken(ctx context.Context, token *domainToken.SessionToken) *appErrors.Error {
	query := `
        INSERT INTO auth.session_tokens (
            token, user_id, expires_at, last_used_at
        ) VALUES (
            $1, $2, $3, $4
        )
        RETURNING created_at`

	err := r.querier.QueryRow(ctx, query,
		token.Token,
		token.UserID,
		token.ExpiresAt,
		token.LastUsedAt,
	).Scan(&token.CreatedAt)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}
	return nil
}

// GetValidToken retrieves a valid non expired token by its value
func (r *SessionTokensRepository) GetValidToken(ctx context.Context, tokenString string) (*domainToken.SessionToken, *appErrors.Error) {
	query := `
        SELECT token, user_id, created_at, expires_at, last_used_at
        FROM auth.session_tokens
        WHERE token = $1
        AND expires_at > NOW()`

	token := &domainToken.SessionToken{}
	err := r.querier.QueryRow(ctx, query, tokenString).Scan(
		&token.Token,
		&token.UserID,
		&token.CreatedAt,
		&token.ExpiresAt,
		&token.LastUsedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, appErrors.ErrorInvalidToken()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return token, nil
}

// UpdateLastUsed updates the last_used_at timestamp for a token
func (r *SessionTokensRepository) UpdateLastUsed(ctx context.Context, tokenString string) *appErrors.Error {
	query := `
        UPDATE auth.session_tokens
        SET last_used_at = NOW()
        WHERE token = $1
        AND expires_at > NOW()`

	result, err := r.querier.Exec(ctx, query, tokenString)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	if result.RowsAffected() == 0 {
		return appErrors.ErrorInvalidToken()
	}

	return nil
}

// DeleteToken deletes a specific session token
func (r *SessionTokensRepository) DeleteToken(ctx context.Context, tokenString string) *appErrors.Error {
	query := `
        DELETE FROM auth.session_tokens
        WHERE token = $1`

	_, err := r.querier.Exec(ctx, query, tokenString)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}

// DeleteUserTokens deletes all session tokens for a specific user
func (r *SessionTokensRepository) DeleteUserTokens(ctx context.Context, userID uuid.UUID) *appErrors.Error {
	query := `
        DELETE FROM auth.session_tokens
        WHERE user_id = $1`

	_, err := r.querier.Exec(ctx, query, userID)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}

// CleanupExpiredTokens removes expired tokens older than the specified duration
func (r *SessionTokensRepository) CleanupExpiredTokens(ctx context.Context, olderThan time.Duration) *appErrors.Error {
	query := `
        DELETE FROM auth.session_tokens
        WHERE expires_at < $1`

	cutoff := time.Now().Add(-olderThan)
	_, err := r.querier.Exec(ctx, query, cutoff)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}
