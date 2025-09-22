package handler

import (
	"context"
	"errors"
	"time"

	"github.com/arabkood/backend/config"
	domainToken "github.com/arabkood/backend/internal/domains/auth/interfaces/token"
	"github.com/arabkood/backend/internal/domains/auth/repo"
	userRepo "github.com/arabkood/backend/internal/domains/user/repo"
	infraEmail "github.com/arabkood/backend/internal/email"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

type PasswordForgotRequest struct {
	Email string `form:"email" binding:"required,app_email" json:"email"`
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	logError := func() *zerolog.Event {
		return h.logger.Error().Str("handler", "auth.forgot-password")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req PasswordForgotRequest
	if err := c.ShouldBind(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			appErrors.StandarizeValidationErrors(ve).AbortWithErrorJson(c)
			return
		}
		appErrors.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		logError().
			Err(err).
			Msg("Failed to begin transaction for password reset request")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repositories instances
	userRepo := userRepo.NewUserRepository(tx)
	oneTimeTokenRepo := repo.NewOneTimeTokensRepository(tx)

	// Get user by email
	user, aerr := userRepo.GetByEmailOrUsername(ctx, req.Email)
	if aerr != nil {
		// Don't reveal if email exists or not
		c.Status(200)
		return
	}
	if user == nil {
		// Don't reveal if email exists or not
		c.Status(200)
		return
	}

	// Check for any existing password reset token
	ott, aerr := oneTimeTokenRepo.GetValidUserToken(ctx, user.ID, domainToken.TokenTypePasswordRecovery)
	if aerr != nil {
		if aerr.Type != appErrors.ErrNotFound {
			aerr.Log(logError(), true).
				Msg("Failed to get valid user token")
		}
		// do nothing, we will generate a new token
	}

	if ott == nil {
		// Generate new reset token
		resetToken, err := domainToken.GenerateSecureToken(user.Email)
		if err != nil {
			logError().
				Err(err).
				Msg("Failed to generate reset code")
			appErrors.ErrorInternal().AbortWithErrorJson(c)
			return
		}

		// Store reset token
		ott = &domainToken.OneTimeToken{
			UserID:    user.ID,
			Token:     resetToken,
			Type:      domainToken.TokenTypePasswordRecovery,
			ExpiresAt: time.Now().Add(time.Minute * time.Duration(h.config.Auth.PasswordResetExpiryMinutes)),
		}

		aerr = oneTimeTokenRepo.StoreToken(ctx, ott)
		if aerr != nil {
			aerr.Log(logError(), true).
				Msg("Failed to store reset token")
			aerr.AbortWithErrorJson(c)
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		logError().
			Err(err).
			Msg("Failed to commit transaction")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Send password reset email asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := h.emailSvc.SendPasswordResetEmail(ctx, &infraEmail.PasswordResetEmail{
			To:       user.Email,
			Username: user.Username,
			Token:    ott.Token,
		})
		if err != nil {
			logError().
				Err(err).
				Msg("Failed to send password reset email")
		}
	}()

	// print code for non-production mode
	if h.config.App.Environment != config.AppEnvProd {
		h.logger.Warn().Str("token", ott.Token).Msg("[DEV ONLY] token for " + user.Email)
	}

	c.Status(200)
}
