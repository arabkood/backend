package repo

import (
	"context"
	"errors"
	"time"

	domainUser "github.com/arabkood/backend/internal/domains/user/interfaces/user"
	"github.com/arabkood/backend/internal/postgres"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserRepository struct {
	querier postgres.Querier
}

func NewUserRepository(querier postgres.Querier) *UserRepository {
	return &UserRepository{querier: querier}
}

func (r *UserRepository) Create(ctx context.Context, user *domainUser.User) *appErrors.Error {
	now := time.Now()
	id, err := uuid.NewRandom()
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}
	user.ID = id
	user.CreatedAt = now
	user.UpdatedAt = now

	query := `
        WITH inserted_user AS (
            INSERT INTO auth.users (
                id, email, username, role,
                encrypted_password, email_verified, email_verified_at,
                created_at, updated_at
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7, $8, $9
            )
            RETURNING id
        )
        INSERT INTO users.stats (
            user_id, total_xp, completed_items, longest_streak, last_active_at
        )
        SELECT id, 0, 0, 0, NULL
        FROM inserted_user;`

	_, err = r.querier.Exec(
		ctx,
		query,
		user.ID,
		user.Email,
		user.Username,
		user.Role,
		user.EncryptedPassword,
		user.EmailVerified,
		user.EmailVerifiedAt,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.ConstraintName {
			case "users_username_key":
				return appErrors.ErrorUsernameConflict()
			case "users_email_key":
				return appErrors.ErrorEmailConflict()
			}
		}
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainUser.User, *appErrors.Error) {
	user := &domainUser.User{}

	query := `
        SELECT 
            id, email, username, role,
            encrypted_password, email_verified, email_verified_at,
            created_at, updated_at
        FROM auth.users
        WHERE id = $1`

	err := r.querier.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.Role,
		&user.EncryptedPassword,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrorUserNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return user, nil
}

func (r *UserRepository) GetStatsByID(ctx context.Context, id uuid.UUID) (*domainUser.Stats, *appErrors.Error) {
	stats := &domainUser.Stats{}

	query := `
        SELECT 
            user_id, total_xp, completed_items, longest_streak, last_active_at
        FROM users.stats
        WHERE user_id = $1`

	err := r.querier.QueryRow(ctx, query, id).Scan(
		&stats.UserID,
		&stats.TotalXP,
		&stats.CompletedItems,
		&stats.LongestStreak,
		&stats.LastActiveAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrorUserNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return stats, nil
}

func (r *UserRepository) GetByEmailOrUsername(ctx context.Context, identifier string) (*domainUser.User, *appErrors.Error) {
	user := &domainUser.User{}

	query := `
        SELECT 
            id, email, username, role,
            encrypted_password, email_verified, email_verified_at,
            created_at, updated_at
        FROM auth.users
        WHERE email = $1 OR username = $1`

	err := r.querier.QueryRow(ctx, query, identifier).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.Role,
		&user.EncryptedPassword,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrorUserNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domainUser.User) *appErrors.Error {
	user.UpdatedAt = time.Now()
	query := `
        UPDATE auth.users
        SET 
            username = $2,
            role = $3,
            email_verified = $4,
            email_verified_at = $5,
            updated_at = $6,
            encrypted_password = $7,
            premium_active = $8,
            polar_last_synced_at = $9,
            polar_customer_id = $10,
            polar_subscription_ids = $11
        WHERE id = $1`

	commandTag, err := r.querier.Exec(
		ctx,
		query,
		user.ID,
		user.Username,
		user.Role,
		user.EmailVerified,
		user.EmailVerifiedAt,
		user.UpdatedAt,
		user.EncryptedPassword,
		user.PremiumActive,
		user.PolarLastSyncedAt,
		user.PolarCustomerID,
		user.PolarSubscriptionIDs,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.ConstraintName == "users_username_key" {
				return appErrors.ErrorUsernameConflict()
			}
		}
		return appErrors.ErrorInternal().WithError(err)
	}

	if commandTag.RowsAffected() == 0 {
		return appErrors.ErrorUserNotFound()
	}

	return nil
}

func (r *UserRepository) VerifyEmail(ctx context.Context, id uuid.UUID) *appErrors.Error {
	now := time.Now()
	query := `
        UPDATE auth.users
        SET 
            email_verified = true,
            email_verified_at = $2,
            updated_at = $3
        WHERE id = $1`

	commandTag, err := r.querier.Exec(ctx, query, id, now, now)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	if commandTag.RowsAffected() == 0 {
		return appErrors.ErrorUserNotFound()
	}

	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, encryptedPassword string) *appErrors.Error {
	query := `
        UPDATE auth.users
        SET 
            encrypted_password = $2,
            updated_at = $3
        WHERE id = $1`

	commandTag, err := r.querier.Exec(ctx, query, id, encryptedPassword, time.Now())
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	if commandTag.RowsAffected() == 0 {
		return appErrors.ErrorUserNotFound()
	}

	return nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, *appErrors.Error) {
	var exists bool
	query := `
        SELECT EXISTS(
            SELECT 1 FROM auth.users WHERE email = $1
        )`

	err := r.querier.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, appErrors.ErrorInternal().WithError(err)
	}

	return exists, nil
}

func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, *appErrors.Error) {
	var exists bool
	query := `
        SELECT EXISTS(
            SELECT 1 FROM auth.users WHERE username = $1
        )`

	err := r.querier.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, appErrors.ErrorInternal().WithError(err)
	}

	return exists, nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) *appErrors.Error {
	query := `DELETE FROM auth.users WHERE id = $1`

	commandTag, err := r.querier.Exec(ctx, query, id)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	if commandTag.RowsAffected() == 0 {
		return appErrors.ErrorUserNotFound()
	}

	return nil
}
