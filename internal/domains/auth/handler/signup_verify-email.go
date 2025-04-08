package handler

import (
	"context"
	"errors"
	"time"

	domainToken "github.com/arabkood/backend/internal/domains/auth/interfaces/token"
	"github.com/arabkood/backend/internal/domains/auth/repo"
	userRepo "github.com/arabkood/backend/internal/domains/user/repo"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/go-playground/validator/v10"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type VerifyEmailRequest struct {
	Code  string `form:"code" binding:"required" json:"code"`
	Email string `form:"email" binding:"required,app_email" json:"email"`
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	logError := func() *zerolog.Event {
		return h.logger.Error().Str("handler", "auth.verify-email")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req VerifyEmailRequest
	if err := c.ShouldBind(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			appErrors.StandarizeValidationErrors(ve).AbortWithErrorJson(c)
			return
		}
		appErrors.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	if !domainToken.ValidateCode(req.Code, h.config.Auth.EmailVerificationCodeLength, h.config.Auth.EmailVerificationCodeChars) {
		appErrors.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// 0. Get and verify email verification token
	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		logError().
			Err(err).
			Msg("Failed to begin transaction for email verification")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repositories instances
	userRepo := userRepo.NewUserRepository(tx)
	oneTimeTokenRepo := repo.NewOneTimeTokensRepository(tx)
	sessionTokensRepo := repo.NewSessionTokensRepository(tx)

	// 0. Check if email already verified
	user, aerr := userRepo.GetByEmailOrUsername(c, req.Email)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to retrieve user")
			// aerr.AbortWithErrorJson(c)
		// We don't want to expose this info, so just say code in invalid
		appErrors.ErrorInvalidEmailVerificationToken().AbortWithErrorJson(c)
		return
	}
	if user == nil {
		// We don't want to expose this info, so just say code in invalid
		appErrors.ErrorInvalidEmailVerificationToken().AbortWithErrorJson(c)
		return
	}
	if user.EmailVerified {
		// We don't want to expose this info, so just say code in invalid
		appErrors.ErrorInvalidEmailVerificationToken().AbortWithErrorJson(c)
		return
	}

	// 1. Find and validate token
	verificationToken, aerr := oneTimeTokenRepo.GetValidUserToken(ctx, user.ID, domainToken.TokenTypeEmailConfirmation)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to retrieve verification token")
		// We don't want to expose this info, so just say code in invalid
		appErrors.ErrorInvalidEmailVerificationToken().AbortWithErrorJson(c)
		return
	}

	// Check if token exists and is valid
	if verificationToken == nil ||
		verificationToken.ExpiresAt.Before(time.Now()) {
		appErrors.ErrorInvalidEmailVerificationToken().AbortWithErrorJson(c)
		return
	}

	if verificationToken.Token == "" || verificationToken.Token != req.Code {
		appErrors.ErrorInvalidEmailVerificationToken().AbortWithErrorJson(c)
		return
	}

	// 3. Update user's email verification status
	user.EmailVerified = true
	now := time.Now()
	user.EmailVerifiedAt = &now
	aerr = userRepo.Update(ctx, user)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to update user's email verification status")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// 4. Revoke the verification token
	aerr = oneTimeTokenRepo.DeleteUserToken(ctx, user.ID, domainToken.TokenTypeEmailConfirmation)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to revoke verification token")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// 5. Generate session token
	sessionTokenStruct, err := domainToken.NewSessionToken(user.ID, time.Hour*time.Duration(h.config.Auth.SessionTokenExpiryHours))
	if err != nil {
		logError().
			Err(err).
			Msg("Couldn't generate session token")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// 6. Store session token
	aerr = sessionTokensRepo.StoreToken(ctx, sessionTokenStruct)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Couldn't store session token in DB")
		aerr.AbortWithErrorJson(c)
		return
	}

	// 5. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		logError().
			Err(err).
			Msg("Failed to commit transaction")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// 6. Set auth cookies
	h.setAuthCookies(c, sessionTokenStruct)

	// 7. Return success response
	c.JSON(200, gin.H{
		"message": "Email verified successfully",
	})
}
