package repo

import (
	"context"
	"encoding/json"
	"time"

	domainToken "github.com/arabkood/backend/internal/domains/auth/interfaces/token"
	"github.com/arabkood/backend/internal/postgres"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OneTimeTokensRepository struct {
	querier postgres.Querier
}

func NewOneTimeTokensRepository(querier postgres.Querier) *OneTimeTokensRepository {
	return &OneTimeTokensRepository{querier: querier}
}

// StoreToken stores a one-time token
func (r *OneTimeTokensRepository) StoreToken(ctx context.Context, token *domainToken.OneTimeToken) *appErrors.Error {
	query := `
        INSERT INTO auth.one_time_tokens (
            user_id, token, type, expires_at, metadata
        ) VALUES (
            $1, $2, $3, $4, $5
        )
        ON CONFLICT (user_id, type) DO UPDATE SET
            token = EXCLUDED.token,
            expires_at = EXCLUDED.expires_at,
            metadata = EXCLUDED.metadata,
            updated_at = CURRENT_TIMESTAMP
        RETURNING updated_at`

	metadataJSON, err := json.Marshal(token.Metadata)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	err = r.querier.QueryRow(ctx, query,
		token.UserID,
		token.Token,
		token.Type,
		token.ExpiresAt,
		metadataJSON,
	).Scan(&token.UpdatedAt)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}

// retrieves a valid token of a specific type
func (r *OneTimeTokensRepository) GetValidUserToken(ctx context.Context, userId uuid.UUID, tokenType domainToken.OneTimeTokenType) (*domainToken.OneTimeToken, *appErrors.Error) {
	query := `
        SELECT user_id, token, type, updated_at, expires_at, metadata
        FROM auth.one_time_tokens
        WHERE user_id = $1 
        AND type = $2
        AND expires_at > NOW()`

	token := &domainToken.OneTimeToken{}
	var metadataJSON []byte

	err := r.querier.QueryRow(ctx, query, userId, tokenType).Scan(
		&token.UserID,
		&token.Token,
		&token.Type,
		&token.UpdatedAt,
		&token.ExpiresAt,
		&metadataJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, appErrors.ErrorNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	if metadataJSON != nil {
		token.Metadata = metadataJSON
	}

	return token, nil
}

func (r *OneTimeTokensRepository) GetTokenByToken(ctx context.Context, token string) (*domainToken.OneTimeToken, *appErrors.Error) {
	query := `
        SELECT user_id, token, type, updated_at, expires_at, metadata
        FROM auth.one_time_tokens
        WHERE token = $1`
	tokenEntity := &domainToken.OneTimeToken{}
	var metadataJSON []byte
	err := r.querier.QueryRow(ctx, query, token).Scan(
		&tokenEntity.UserID,
		&tokenEntity.Token,
		&tokenEntity.Type,
		&tokenEntity.UpdatedAt,
		&tokenEntity.ExpiresAt,
		&metadataJSON,
	)
	if err == pgx.ErrNoRows {
		return nil, appErrors.ErrorNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}
	if metadataJSON != nil {
		tokenEntity.Metadata = metadataJSON
	}
	return tokenEntity, nil
}

// DeleteUserTokens deletes all tokens for a user of a specific type
func (r *OneTimeTokensRepository) DeleteUserToken(ctx context.Context, userID uuid.UUID, tokenType domainToken.OneTimeTokenType) *appErrors.Error {
	query := `
        DELETE FROM auth.one_time_tokens
        WHERE user_id = $1 
        AND type = $2`

	_, err := r.querier.Exec(ctx, query, userID, tokenType)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}

// CleanupExpiredTokens removes expired tokens older than the specified duration
func (r *OneTimeTokensRepository) CleanupExpiredTokens(ctx context.Context, olderThan time.Duration) *appErrors.Error {
	query := `
        DELETE FROM auth.one_time_tokens
        WHERE expires_at < $1`

	cutoff := time.Now().Add(-olderThan)
	_, err := r.querier.Exec(ctx, query, cutoff)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}
