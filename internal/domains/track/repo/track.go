package repo

import (
	"context"

	"github.com/arabkood/backend/internal/postgres"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/google/uuid"
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
    )
    ON CONFLICT ON CONSTRAINT track_pkey DO NOTHING
`

	_, err := r.querier.Exec(ctx, query, userID, trackID)
	if err != nil {
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}
