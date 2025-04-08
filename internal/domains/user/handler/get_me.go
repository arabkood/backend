package handler

import (
	"context"
	"time"

	"github.com/arabkood/backend/internal/domains/user/repo"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type GetMeResponse struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
}

func (h *UserHandler) GetMe(c *gin.Context) {
	logError := func() *zerolog.Event {
		return h.logger.Error().Str("handler", "auth.getMe")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		logError().
			Err(err).
			Msg("Failed to begin transaction for get user")
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

	// Get user
	user, aerr := userRepo.GetByID(ctx, userID.(uuid.UUID))
	if aerr != nil {
		if aerr.Type == appError.ErrUserNotFound {
			appError.ErrorUnauthorized().AbortWithErrorJson(c)
			return
		}
		aerr.Log(logError(), true).
			Msg("Failed to get user by ID")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		logError().
			Err(err).
			Msg("Couldn't commit transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Return response
	res := GetMeResponse{
		ID:            user.ID.String(),
		Username:      user.Username,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
	}
	c.JSON(200, res)
}
