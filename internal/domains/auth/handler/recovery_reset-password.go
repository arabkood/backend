package handler

import (
	"context"
	"errors"
	"time"

	domainToken "github.com/arabkood/backend/internal/domains/auth/interfaces/token"
	"github.com/arabkood/backend/internal/domains/auth/repo"
	userRepo "github.com/arabkood/backend/internal/domains/user/repo"
	appError "github.com/arabkood/backend/pkg/errors"
	appPassword "github.com/arabkood/backend/pkg/password"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

type PasswordForgotResetRequest struct {
	Token       string `form:"token" binding:"required,min=32,max=256" json:"token"`
	NewPassword string `form:"newPassword" binding:"required,app_password" json:"newPassword"`
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	logError := func() *zerolog.Event {
		return h.logger.Error().Str("handler", "auth.reset-password")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req PasswordForgotResetRequest
	if err := c.ShouldBind(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			appError.StandarizeValidationErrors(ve).AbortWithErrorJson(c)
			return
		}
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// Hash the new password, before validating the token to prevent holding the tx
	hashedPassword, err := appPassword.HashPassword(req.NewPassword, appPassword.DefaultArgon2Params())
	if err != nil {
		logError().
			Err(err).
			Msg("Failed to hash password")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		logError().
			Err(err).
			Msg("Failed to begin transaction for password reset")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repositories instances
	userRepo := userRepo.NewUserRepository(tx)
	oneTimeTokenRepo := repo.NewOneTimeTokensRepository(tx)
	sessionTokenRepo := repo.NewSessionTokensRepository(tx)

	// Get token
	token, aerr := oneTimeTokenRepo.GetTokenByToken(ctx, req.Token)

	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to retrieve reset token")
		appError.ErrorInvalidRecoveryToken().AbortWithErrorJson(c)
		return
	}

	if token == nil || token.ExpiresAt.Before(time.Now()) {
		appError.ErrorInvalidRecoveryToken().AbortWithErrorJson(c)
		return
	}

	// Get user
	user, aerr := userRepo.GetByID(ctx, token.UserID)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to retrieve user")
		appError.ErrorInvalidRecoveryToken().AbortWithErrorJson(c)
		return
	}
	if user == nil {
		appError.ErrorInvalidRecoveryToken().AbortWithErrorJson(c)
		return
	}

	// Update password
	user.EncryptedPassword = string(hashedPassword)
	user.UpdatedAt = time.Now()

	aerr = userRepo.Update(ctx, user)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to update user password")
		aerr.AbortWithErrorJson(c)
		return
	}

	// Revoke all session tokens
	aerr = sessionTokenRepo.DeleteUserTokens(ctx, user.ID)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to revoke session tokens")
		aerr.AbortWithErrorJson(c)
		return
	}

	// Delete the reset token
	aerr = oneTimeTokenRepo.DeleteUserToken(ctx, user.ID, domainToken.TokenTypePasswordRecovery)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to delete reset token")
		aerr.AbortWithErrorJson(c)
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		logError().
			Err(err).
			Msg("Failed to commit transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	c.Status(200)
}
