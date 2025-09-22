package handler

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	domainToken "github.com/arabkood/backend/internal/domains/auth/interfaces/token"
	"github.com/arabkood/backend/internal/domains/auth/repo"
	userRepo "github.com/arabkood/backend/internal/domains/user/repo"
	appErrors "github.com/arabkood/backend/pkg/errors"
	appPassword "github.com/arabkood/backend/pkg/password"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

type SigninRequest struct {
	Identifier string `form:"identifier" binding:"required,min=1,max=254" json:"identifier"`
	Password   string `form:"password" binding:"required,min=1,max=1000" json:"password"`
}

type SigninResponse struct {
	EmailVerificationAlreadySent *bool  `json:"emailVerificationAlreadySent,omitempty"`
	ID                           string `json:"id"`
	Username                     string `json:"username"`
	Email                        string `json:"email"`
	EmailVerified                bool   `json:"emailVerified"`
}

func (h *AuthHandler) Signin(c *gin.Context) {
	logError := func() *zerolog.Event {
		return h.logger.Error().Str("handler", "auth.signin")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req SigninRequest
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
			Msg("Failed to begin transaction for user signin")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repositories instances
	userRepo := userRepo.NewUserRepository(tx)
	sessionTokensRepo := repo.NewSessionTokensRepository(tx)
	auditLogsRepo := repo.NewAuditLogsRepository(tx)
	oneTimeTokenRepo := repo.NewOneTimeTokensRepository(tx)

	// 1. Get user
	user, aerr := userRepo.GetByEmailOrUsername(ctx, req.Identifier)
	if aerr != nil {
		if aerr.Type == appErrors.ErrUserNotFound {
			appErrors.ErrorBadCredentials().AbortWithErrorJson(c)
			return
		}
		aerr.Log(logError(), true).
			Msg("Failed to get user by email")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// 2. Verify password
	if ok, err := appPassword.VerifyPassword(req.Password, user.EncryptedPassword); err != nil || !ok {
		// Create failed login audit log
		auditLog := &repo.AuditLog{
			UserID:    &user.ID,
			Type:      repo.AuditLogTypeSignin,
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Metadata: json.RawMessage(`{
				"success": false,
				"reason": "invalid_password"
			}`),
		}

		if aerr := auditLogsRepo.CreateLog(ctx, auditLog); aerr != nil {
			logError().
				Err(aerr).
				Msg("Failed to create failed login audit log")
		}

		if err := tx.Commit(ctx); err != nil {
			logError().
				Err(err).
				Msg("Failed to commit failed login audit log")
		}

		appErrors.ErrorBadCredentials().AbortWithErrorJson(c)
		return
	}

	// 3. If email not verified check for valid token, otherwise

	emailConfirmationCodeAlreadySent := false
	if !user.EmailVerified {
		verificationToken, aerr := oneTimeTokenRepo.GetValidUserToken(ctx, user.ID, domainToken.TokenTypeEmailConfirmation)
		if aerr == nil {
			// Check if token exists and is valid
			if verificationToken != nil &&
				verificationToken.Token != "" &&
				verificationToken.ExpiresAt.After(time.Now()) {
				emailConfirmationCodeAlreadySent = true
			}
		}
	}

	// 3. Generate session token
	sessionTokenStruct, err := domainToken.NewSessionToken(user.ID, time.Hour*time.Duration(h.config.Auth.SessionTokenExpiryHours))
	if err != nil {
		logError().
			Err(err).
			Msg("Couldn't generate session token")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// 4. Store session token
	aerr = sessionTokensRepo.StoreToken(ctx, sessionTokenStruct)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Couldn't store session token in DB")
		aerr.AbortWithErrorJson(c)
		return
	}

	// 5. Create successful login audit log
	auditLog := &repo.AuditLog{
		UserID:    &user.ID,
		Type:      repo.AuditLogTypeSignin,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Metadata: json.RawMessage(`{
			"success": true
		}`),
	}

	aerr = auditLogsRepo.CreateLog(ctx, auditLog)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Couldn't create login audit log")
		aerr.AbortWithErrorJson(c)
		return
	}

	// 6. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		logError().
			Err(err).
			Msg("Couldn't commit transaction")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// 8. Return success response
	res := SigninResponse{
		ID:            user.ID.String(),
		EmailVerified: user.EmailVerified,
		Email:         user.Email,
		Username:      user.Username,
	}
	// 7. Set auth cookies
	res.EmailVerificationAlreadySent = &emailConfirmationCodeAlreadySent
	h.setAuthCookies(c, sessionTokenStruct)
	c.JSON(200, res)
}
