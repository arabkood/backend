package repo

import (
	"context"
	"errors"

	"github.com/arabkood/backend/internal/postgres"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type TrackRepository struct {
	querier postgres.Querier
}

func NewTrackRepository(querier postgres.Querier) *TrackRepository {
	return &TrackRepository{querier: querier}
}

func (r *TrackRepository) StartTrack(ctx context.Context, userID, trackID uuid.UUID) *appErrors.Error {
	query := `
        INSERT INTO users.track (
            user_id, track_id, started_at
        ) VALUES (
            $1, $2, NOW()
        )`

	_, err := r.querier.Exec(ctx, query, userID, trackID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Check for unique violation error code
			if pgErr.ConstraintName == "user_tracks_pkey" {
				return appErrors.ErrorTrackAlreadyStarted()
			}
		}
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}
