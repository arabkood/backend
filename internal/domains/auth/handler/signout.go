package handler

import (
	"context"
	"time"

	"github.com/arabkood/backend/internal/domains/auth/repo"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/gin-gonic/gin"
)

func (h *AuthHandler) Signout(c *gin.Context) {
	logError := func() *zerolog.Event {
		return h.logger.Error().Str("handler", "auth.signout")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		logError().
			Err(err).
			Msg("Failed to begin transaction for user signout")
		appErrors.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repositories instances
	sessionTokensRepo := repo.NewSessionTokensRepository(tx)

	// 1. Get session token from cookie
	sessionToken, err := c.Cookie(h.config.Auth.SessionCookieName)
	if err == nil && sessionToken != "" {
		// 2. Invalidate session token in database
		if aerr := sessionTokensRepo.DeleteToken(ctx, sessionToken); aerr != nil {
			aerr.Log(logError(), false).
				Msg("Failed to invalidate session token")
			// Continue execution as this is not critical
		}
		// 4. Commit transaction
		if err := tx.Commit(ctx); err != nil {
			logError().
				Err(err).
				Msg("Failed to commit transaction")
			appErrors.ErrorInternal().AbortWithErrorJson(c)
			return
		}
	}

	// 5. Clear auth cookies
	h.unsetAuthCookies(c)

	// 6. Return success response
	c.Status(200)
}
