package xpRepo

import (
	"context"

	"github.com/arabkood/backend/internal/postgres"
)

func AddXP(ctx context.Context, querier postgres.Querier, event XPEvent) error {
	// Insert XP event
	_, err := querier.Exec(ctx, `
		INSERT INTO users.xp_events (user_id, xp_amount, source_type, source_id)
		VALUES ($1, $2, $3, $4)
	`, event.UserID, event.XPAmount, event.SourceType, event.SourceID)
	if err != nil {
		return err
	}

	// Update user stats
	_, err = querier.Exec(ctx, `
		INSERT INTO users.stats (user_id, total_xp, last_active_at)
		VALUES ($1, $2, now())
		ON CONFLICT (user_id) DO UPDATE
		SET total_xp = users.stats.total_xp + $2,
		    last_active_at = now()
	`, event.UserID, event.XPAmount)
	if err != nil {
		return err
	}

	// Update daily stats
	_, err = querier.Exec(ctx, `
		INSERT INTO users.daily_stats (user_id, date, xp_earned)
		VALUES ($1, CURRENT_DATE, $2)
		ON CONFLICT (user_id, date) DO UPDATE
		SET xp_earned = users.daily_stats.xp_earned + $2
	`, event.UserID, event.XPAmount)
	if err != nil {
		return err
	}

	return nil
}
