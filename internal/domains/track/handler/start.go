package handler

import (
	"context"
	"time"

	"github.com/arabkood/backend/internal/domains/track/repo"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type StartTrackResponse struct{}

func (h *TrackHandler) StartTrack(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	trackID := c.Query("id")
	if trackID == "" {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}
	// convert id to uuid
	trackId, err := uuid.Parse(trackID)
	if err != nil {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}
	userID, exists := c.Get("userID")
	if !exists {
		appError.ErrorUnauthorized().AbortWithErrorJson(c)
		return
	}

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		h.logger.Error().Str("handler", "track.StartTrack").Err(err).Msg("Failed to begin transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repository instance
	trackRepo := repo.NewTrackRepository(tx)

	// Start track
	aerr := trackRepo.StartTrack(ctx, userID.(uuid.UUID), trackId)

	if aerr != nil {
		aerr.AbortWithErrorJson(c)
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		h.logger.Error().Str("handler", "track.StartTrack").
			Err(err).
			Msg("Couldn't commit transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	c.JSON(201, map[string]bool{
		"success": true,
	})
}
