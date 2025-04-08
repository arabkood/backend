package handler

import (
	"context"
	"time"

	domainTrack "github.com/arabkood/backend/internal/domains/track/interfaces/track"
	"github.com/arabkood/backend/internal/domains/track/repo"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetTrackResponse struct {
	Track   *domainTrack.Track               `json:"track"`
	Content []domainTrack.SectionWithModules `json:"content"`
}

func (h *TrackHandler) GetTrack(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	slug := c.Query("slug")
	id := c.Query("id")

	if slug == "" && id == "" {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		h.logger.Error().Str("handler", "track.GetTrack").Err(err).Msg("Failed to begin transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repository instance
	trackRepo := repo.NewTrackRepository(tx)

	// Get track
	var track *domainTrack.Track
	var aerr *appError.Error
	if slug != "" {
		track, aerr = trackRepo.GetTrackBySlug(ctx, slug)
	} else {
		// convert id to uuid
		trackId, err := uuid.Parse(id)
		if err != nil {
			appError.ErrorInvalidInput().AbortWithErrorJson(c)
			return
		}
		track, aerr = trackRepo.GetTrackByID(ctx, trackId)
	}
	if aerr != nil {
		if aerr.Type == appError.ErrNotFound {
			appError.ErrorTrackNotFound().AbortWithErrorJson(c)
			return
		}
		aerr.Log(h.logger.Error().Str("handler", "track.GetTrack"), true).
			Msg("Failed to get track")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Get track content
	trackContent, aerr := trackRepo.GetTrackContentByTrackID(ctx, track.ID)
	if aerr != nil {
		aerr.Log(h.logger.Error().Str("handler", "track.GetTrack"), true).
			Msg("Failed to get track content")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		h.logger.Error().Str("handler", "track.GetTrack").
			Err(err).
			Msg("Couldn't commit transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Return response
	res := GetTrackResponse{
		Track:   track,
		Content: trackContent,
	}
	c.JSON(200, res)
}
