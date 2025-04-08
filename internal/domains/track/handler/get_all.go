package handler

import (
	"context"
	"time"

	domainTrack "github.com/arabkood/backend/internal/domains/track/interfaces/track"
	"github.com/arabkood/backend/internal/domains/track/repo"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
)

type ListTracksResponse struct {
	Data []domainTrack.Track `json:"data"`
}

func (h *TrackHandler) ListTracks(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		h.logger.Error().Str("handler", "track.ListTracks").Err(err).Msg("Failed to begin transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repository instance
	trackRepo := repo.NewTrackRepository(tx)

	// Get all tracks
	tracks, aerr := trackRepo.GetAllTracks(ctx)
	if aerr != nil {
		aerr.Log(h.logger.Error().Str("handler", "track.ListTracks"), true).
			Msg("Failed to get all tracks")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		h.logger.Error().Str("handler", "track.ListTracks").
			Err(err).
			Msg("Couldn't commit transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Return response
	res := ListTracksResponse{
		Data: tracks,
	}
	c.JSON(200, res)
}
