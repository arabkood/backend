package handler

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/arabkood/backend/config"
	domainToken "github.com/arabkood/backend/internal/domains/auth/interfaces/token"
	"github.com/arabkood/backend/internal/domains/auth/repo"
	domainUser "github.com/arabkood/backend/internal/domains/user/interfaces/user"
	userRepo "github.com/arabkood/backend/internal/domains/user/repo"
	infraEmail "github.com/arabkood/backend/internal/email"
	appErrors "github.com/arabkood/backend/pkg/errors"
	appPassword "github.com/arabkood/backend/pkg/password"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

type RegisterRequest struct {
	Email    string `form:"email" binding:"required,app_email" json:"email"`
	Username string `form:"username" binding:"required,app_username_strict" json:"username"`
	Password string `form:"password" binding:"required,app_password" json:"password"`
}

type RegisterResponse struct {
	EmailVerificationAlreadySent *bool  `json:"emailVerificationAlreadySent,omitempty"`
	ID                           string `json:"id"`
	Username                     string `json:"username"`
	Email                        string `json:"email"`
	EmailVerified                bool   `json:"emailVerified"`
	canResendCodeAt              int64  `json:"canResendCodeAt"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	logError := func() *zerolog.Event {
		return h.logger.Error().Str("handler", "auth.signup")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req RegisterRequest
	if err := c.ShouldBind(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			appErrors.StandarizeValidationErrors(ve).AbortWithErrorJson(c)
			return
		}
		appErrors.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// 1. Hash password
	hashedPassword, err := appPassword.HashPassword(req.Password, appPassword.DefaultArgon2Params())
	if err != nil {
		logError().
			Err(err).
			Msg("Failed to hash password")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Create user instance
	newUser := &domainUser.User{
		Email:             req.Email,
		Username:          req.Username,
		EncryptedPassword: string(hashedPassword),
		EmailVerified:     false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// only production need to verify email
	// if h.config.App.Environment != config.AppEnvProd {
	// 	newUser.EmailVerified = true
	// }

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		logError().
			Err(err).
			Msg("Failed to begin transaction for user signup")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repositories instances
	userRepo := userRepo.NewUserRepository(tx)
	oneTimeTokensRepo := repo.NewOneTimeTokensRepository(tx)
	auditLogsRepo := repo.NewAuditLogsRepository(tx)

	// 2. Create user in DB
	aerr := userRepo.Create(ctx, newUser)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Couldn't create user in DB")
		aerr.AbortWithErrorJson(c)
		return
	}

	// 3. Generate & store verification token
	emailVerificationCode, err := domainToken.GenerateRandomCode(h.config.Auth.EmailVerificationCodeLength, h.config.Auth.EmailVerificationCodeChars)
	if err != nil {
		logError().
			Err(err).
			Msg("Couldn't generate email verification code")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	verificationTokenStruct := &domainToken.OneTimeToken{
		UserID:    newUser.ID,
		Token:     emailVerificationCode,
		Type:      domainToken.TokenTypeEmailConfirmation,
		ExpiresAt: time.Now().Add(time.Minute * time.Duration(h.config.Auth.EmailVerificationExpiryMinutes)),
	}
	aerr = oneTimeTokensRepo.StoreToken(ctx, verificationTokenStruct)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Couldn't store verification token")
		aerr.AbortWithErrorJson(c)
		return
	}

	// Create audit log for signup
	auditLog := &repo.AuditLog{
		UserID:    &newUser.ID,
		Type:      repo.AuditLogTypeSignup,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Metadata: json.RawMessage(`{
        "email": "` + newUser.Email + `",
        "username": "` + newUser.Username + `"
    }`),
	}
	aerr = auditLogsRepo.CreateLog(ctx, auditLog)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Couldn't create audit log")
		aerr.AbortWithErrorJson(c)
		return
	}

	// 5. Generate and set session token
	sessionTokenStruct, err := domainToken.NewSessionToken(newUser.ID, time.Hour*time.Duration(h.config.Auth.SessionTokenExpiryHours))
	if err != nil {
		logError().
			Err(err).
			Msg("Couldn't generate session token after signup")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	sessionTokensRepo := repo.NewSessionTokensRepository(tx)
	aerr = sessionTokensRepo.StoreToken(ctx, sessionTokenStruct)
	if aerr != nil {
		aerr.Log(logError(), true).Msg("Couldn't store session token after signup")
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

	// 7. Set auth cookies
	h.setAuthCookies(c, sessionTokenStruct)

	// 8. If everything went good, send verification email asynchronously
	// only send code in production
	if h.config.App.Environment == config.AppEnvProd {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := h.emailSvc.SendVerificationEmail(ctx, &infraEmail.VerificationEmail{
				To:       newUser.Email,
				Username: newUser.Username,
				Token:    emailVerificationCode,
			})
			if err != nil {
				logError().
					Err(err).
					Msg("Failed to send verification email")
			}
		}()
	}

	// print code for non-production mode
	if h.config.App.Environment != config.AppEnvProd {
		h.logger.Warn().Str("token", emailVerificationCode).Msg("[DEV ONLY] token for " + newUser.Email)
	}

	// 9. Return success response
	emailSent := true
	canResendCodeAt := time.Now().Add(time.Minute)
	c.JSON(201, RegisterResponse{
		ID:                           newUser.ID.String(),
		EmailVerified:                newUser.EmailVerified,
		Email:                        newUser.Email,
		Username:                     newUser.Username,
		EmailVerificationAlreadySent: &emailSent,
		canResendCodeAt:              canResendCodeAt.UnixMilli(),
	})
}
