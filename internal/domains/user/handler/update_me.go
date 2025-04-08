package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/arabkood/backend/internal/domains/user/repo"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type UpdateMeRequest struct {
	Username *string `form:"username" binding:"required,app_username_strict" json:"username"`
	// Email    *string `json:"email"`
}

type UpdateMeResponse struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	logError := func() *zerolog.Event {
		return h.logger.Error().Str("handler", "auth.updateMe")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get user ID from auth context
	userID, exists := c.Get("userID")
	if !exists {
		appError.ErrorUnauthorized().AbortWithErrorJson(c)
		return
	}

	// Bind request body
	var req UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		logError().
			Err(err).
			Msg("Failed to begin transaction for update user")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repository instance
	userRepo := repo.NewUserRepository(tx)

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

	// Update username if provided
	if req.Username != nil {
		// Check if username is already taken
		exists, aerr := userRepo.ExistsByUsername(ctx, *req.Username)
		if aerr != nil {
			aerr.Log(logError(), true).
				Msg("Failed to check if username exists")
			appError.ErrorInternal().AbortWithErrorJson(c)
			return
		}
		if exists {
			appError.ErrorUsernameConflict().AbortWithErrorJson(c)
			return
		}

		user.Username = *req.Username
	}

	// TODO: Add logic for email and other stuff

	// Update user in database
	if aerr := userRepo.Update(ctx, user); aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to update user")
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
	res := UpdateMeResponse{
		ID:            user.ID.String(),
		Username:      user.Username,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
	}
	c.JSON(http.StatusOK, res)
}
