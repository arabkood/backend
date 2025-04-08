package handler

import (
	"context"
	"time"

	"github.com/arabkood/backend/internal/domains/user/repo"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetStatsResponse struct {
	TotalXp        int `json:"totalXp"`
	CurrentStreak  int `json:"currentStreak"`
	GlobalRank     int `json:"globalRank"`
	LongestStreak  int `json:"longestStreak"`
	CompletedItems int `json:"completedItems"`
}

func (h *UserHandler) GetMyStats(c *gin.Context) {
	// h.logger.Error().Str("handler", "me.GetStats")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		h.logger.Error().Str("handler", "me.GetStats").Err(err).Msg("Failed to begin transaction for get dashboard")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repository instance
	userRepo := repo.NewUserRepository(tx)

	// Get user ID from auth context
	userID, exists := c.Get("userID")
	if !exists {
		appError.ErrorUnauthorized().AbortWithErrorJson(c)
		return
	}

	// Get userStats
	userStats, aerr := userRepo.GetStatsByID(ctx, userID.(uuid.UUID))
	if aerr != nil {
		if aerr.Type == appError.ErrUserNotFound {
			appError.ErrorUnauthorized().AbortWithErrorJson(c)
			return
		}
		aerr.Log(h.logger.Error().Str("handler", "me.GetStats"), true).
			Msg("Failed to get user by ID")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		h.logger.Error().Str("handler", "me.GetStats").
			Err(err).
			Msg("Couldn't commit transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Return response
	res := GetStatsResponse{
		TotalXp:        int(userStats.TotalXP),
		LongestStreak:  int(userStats.LongestStreak),
		CompletedItems: int(userStats.CompletedItems),

		// FIX:  implement
		CurrentStreak: 0,
		GlobalRank:    0,
	}
	c.JSON(200, res)
}
