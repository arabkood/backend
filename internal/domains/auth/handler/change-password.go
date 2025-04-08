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
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ChangePasswordRequest struct {
	CurrentPassword string `form:"oldPassword" binding:"required" json:"oldPassword"`
	NewPassword     string `form:"newPassword" binding:"required,app_password" json:"newPassword"`
}

type ChangePasswordResponse struct {
	Success bool `json:"success"`
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	logError := func() *zerolog.Event {
		return h.logger.Error().Str("handler", "auth.change_password")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get user ID from context
	userID, ok := c.Get("userID")
	if !ok {
		appErrors.ErrorInvalidAuthToken().AbortWithErrorJson(c)
		return
	}
	userId, ok := userID.(uuid.UUID)
	if !ok {
		logError().
			Any("userID", userID).
			Msg("Failed to parse user ID")
		appErrors.ErrorInvalidAuthToken().AbortWithErrorJson(c)
		return
	}

	// Validate request
	var req ChangePasswordRequest
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
			Msg("Failed to begin transaction for password change")
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

	// Get current user
	currentUser, aerr := userRepo.GetByID(ctx, userId)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Failed to get user by ID")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Verify current password
	if ok, err := appPassword.VerifyPassword(req.CurrentPassword, currentUser.EncryptedPassword); err != nil || !ok {
		// Create failed password change audit log
		auditLog := &repo.AuditLog{
			UserID:    &userId,
			Type:      repo.AuditLogTypePasswordChange,
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Metadata: json.RawMessage(`{
				"success": false,
				"reason": "invalid_current_password"
			}`),
		}

		if aerr := auditLogsRepo.CreateLog(ctx, auditLog); aerr != nil {
			logError().
				Err(aerr).
				Msg("Failed to create failed password change audit log")
		}

		if err := tx.Commit(ctx); err != nil {
			logError().
				Err(err).
				Msg("Failed to commit failed password change audit log")
		}

		appErrors.ErrorBadCredentials().AbortWithErrorJson(c)
		return
	}

	// Hash new password
	hashedPassword, err := appPassword.HashPassword(req.NewPassword, appPassword.DefaultArgon2Params())
	if err != nil {
		logError().
			Err(err).
			Msg("Failed to hash new password")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Update password in DB
	currentUser.EncryptedPassword = hashedPassword
	currentUser.UpdatedAt = time.Now()

	aerr = userRepo.Update(ctx, currentUser)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Couldn't update user password in DB")
		aerr.AbortWithErrorJson(c)
		return
	}

	// Invalidate all session tokens for security
	aerr = sessionTokensRepo.DeleteUserTokens(ctx, userId)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Couldn't invalidate session tokens")
		aerr.AbortWithErrorJson(c)
		return
	}

	// Generate session token
	sessionTokenStruct, err := domainToken.NewSessionToken(currentUser.ID, time.Hour*time.Duration(h.config.Auth.SessionTokenExpiryHours))
	if err != nil {
		logError().
			Err(err).
			Msg("Couldn't generate new session token")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Store new session token
	aerr = sessionTokensRepo.StoreToken(ctx, sessionTokenStruct)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Couldn't store new session token in DB")
		aerr.AbortWithErrorJson(c)
		return
	}

	// Create successful password change audit log
	auditLog := &repo.AuditLog{
		UserID:    &currentUser.ID,
		Type:      repo.AuditLogTypePasswordChange,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Metadata: json.RawMessage(`{
			"success": true
		}`),
	}

	aerr = auditLogsRepo.CreateLog(ctx, auditLog)
	if aerr != nil {
		aerr.Log(logError(), true).
			Msg("Couldn't create password change audit log")
		aerr.AbortWithErrorJson(c)
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		logError().
			Err(err).
			Msg("Couldn't commit transaction")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Set new authentication cookies
	h.setAuthCookies(c, sessionTokenStruct)

	// Return success response
	c.JSON(200, ChangePasswordResponse{Success: true})
}
