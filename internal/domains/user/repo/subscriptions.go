package repo

import (
	"context"
	"errors"

	domainUser "github.com/arabkood/backend/internal/domains/user/interfaces/user"
	"github.com/arabkood/backend/internal/postgres"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SubscriptionRepository struct {
	querier postgres.Querier
}

func NewSubscriptionRepository(querier postgres.Querier) *SubscriptionRepository {
	return &SubscriptionRepository{querier: querier}
}

func (r *SubscriptionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domainUser.Subscription, *appErrors.Error) {
	sub := &domainUser.Subscription{}

	query := `
        SELECT
            user_id, stripe_customer_id, stripe_subscription_id,
            plan, plan_interval, pro_until, cancel_at_period_end,
            created_at, updated_at
        FROM auth.user_subscriptions
        WHERE user_id = $1`

	err := r.querier.QueryRow(ctx, query, userID).Scan(
		&sub.UserID,
		&sub.StripeCustomerID,
		&sub.StripeSubscriptionID,
		&sub.Plan,
		&sub.PlanInterval,
		&sub.ProUntil,
		&sub.CancelAtPeriodEnd,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrorUserNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return sub, nil
}

// Upsert inserts or updates the subscription row keyed by user_id. The DB
// manages created_at/updated_at, which are returned into the passed struct.
func (r *SubscriptionRepository) Upsert(ctx context.Context, sub *domainUser.Subscription) *appErrors.Error {
	query := `
        INSERT INTO auth.user_subscriptions (
            user_id, stripe_customer_id, stripe_subscription_id,
            plan, plan_interval, pro_until, cancel_at_period_end
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7
        )
        ON CONFLICT (user_id) DO UPDATE SET
            stripe_customer_id     = EXCLUDED.stripe_customer_id,
            stripe_subscription_id = EXCLUDED.stripe_subscription_id,
            plan                   = EXCLUDED.plan,
            plan_interval          = EXCLUDED.plan_interval,
            pro_until              = EXCLUDED.pro_until,
            cancel_at_period_end   = EXCLUDED.cancel_at_period_end
        RETURNING created_at, updated_at`

	err := r.querier.QueryRow(
		ctx,
		query,
		sub.UserID,
		sub.StripeCustomerID,
		sub.StripeSubscriptionID,
		sub.Plan,
		sub.PlanInterval,
		sub.ProUntil,
		sub.CancelAtPeriodEnd,
	).Scan(&sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}
