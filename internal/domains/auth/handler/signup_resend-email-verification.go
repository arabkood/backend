package handler

import (
	"context"
	"errors"
	"time"

	domainToken "github.com/arabkood/backend/internal/domains/auth/interfaces/token"
	"github.com/arabkood/backend/internal/domains/auth/repo"
	userRepo "github.com/arabkood/backend/internal/domains/user/repo"
	infraEmail "github.com/arabkood/backend/internal/email"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

type ResendEmailVerificationRequest struct {
	Email string `form:"email" binding:"required,app_email" json:"email"`
}
type ResendEmailVerificationResponse struct {
	CanResendCodeAt int64 `json:"canResendCodeAt"`
}

func (h *AuthHandler) ResendEmailVerification(c *gin.Context) {
	logError := func() *zerolog.Event {
		return h.logger.Error().Str("handler", "auth.resend-email-verification")
	}

	var req ResendEmailVerificationRequest
	if err := c.ShouldBind(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			appErrors.StandarizeValidationErrors(ve).AbortWithErrorJson(c)
			return
		}
		appErrors.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 0. Get and verify email verification token
	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		logError().
			Err(err).
			Msg("Failed to begin transaction for email verification resend")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repositories instances
	userRepo := userRepo.NewUserRepository(tx)
	oneTimeTokenRepo := repo.NewOneTimeTokensRepository(tx)

	// Check if email already verified
	user, aerr := userRepo.GetByEmailOrUsername(c, req.Email)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to retrieve user")
		aerr.AbortWithErrorJson(c)
		// We don't want to expose this info, so just return 404
		appErrors.ErrorNotFound().AbortWithErrorJson(c)
		return
	}
	if user == nil {
		appErrors.ErrorNotFound().AbortWithErrorJson(c)
		return
	}
	if user.EmailVerified {
		// We don't want to expose this info, so just return 404
		appErrors.ErrorNotFound().AbortWithErrorJson(c)
		return
	}

	// Get existing verification token
	existingToken, aerr := oneTimeTokenRepo.GetValidUserToken(ctx, user.ID, domainToken.TokenTypeEmailConfirmation)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to retrieve existing verification token")
		// do nothing, we will generate a new token
	}

	var emailVerificationCode string

	// If token doesn't exist or is expired, create a new one
	if existingToken == nil || existingToken.ExpiresAt.Before(time.Now()) {
		// Generate new verification token
		emailVerificationCode, err = domainToken.GenerateRandomCode(h.config.Auth.EmailVerificationCodeLength, h.config.Auth.EmailVerificationCodeChars)
		if err != nil {
			logError().
				Err(err).
				Msg("Failed to generate verification code")
			appErrors.ErrorInternal().AbortWithErrorJson(c)
			return
		}

		// Store new verification token
		verificationToken := &domainToken.OneTimeToken{
			UserID:    user.ID,
			Token:     emailVerificationCode,
			Type:      domainToken.TokenTypeEmailConfirmation,
			ExpiresAt: time.Now().Add(time.Minute * time.Duration(h.config.Auth.EmailVerificationExpiryMinutes)),
		}

		aerr = oneTimeTokenRepo.StoreToken(ctx, verificationToken)
		if aerr != nil {
			aerr.Log(logError(), true).
				Msg("Failed to store verification token")
			aerr.AbortWithErrorJson(c)
			return
		}
	} else {
		// Use existing token
		emailVerificationCode = existingToken.Token
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		logError().
			Err(err).
			Msg("Failed to commit transaction")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Send verification email asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := h.emailSvc.SendVerificationEmail(ctx, &infraEmail.VerificationEmail{
			To:       user.Email,
			Username: user.Username,
			Token:    emailVerificationCode,
		})
		if err != nil {
			logError().
				Err(err).
				Msg("Failed to send verification email")
		}
	}()

	// Return success response
	canResendCodeAt := time.Now().Add(time.Minute)
	c.JSON(200, ResendEmailVerificationResponse{
		CanResendCodeAt: canResendCodeAt.UnixMilli(),
	})
}
