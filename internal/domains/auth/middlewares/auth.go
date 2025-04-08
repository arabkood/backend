package middlewares

import (
	"time"

	"github.com/arabkood/backend/internal/domains/auth/repo"
	"github.com/arabkood/backend/internal/server/interfaces/server"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware is a middleware function to verify User's session
func AuthRequired(srv *server.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie(srv.Config.Auth.SessionCookieName)
		if err != nil || tokenString == "" {
			appError.ErrorMissingAuthToken().AbortWithErrorJson(c)
			return
		}

		sessionRepo := repo.NewSessionTokensRepository(srv.PostgresPool)
		session, aerr := sessionRepo.GetValidToken(c.Request.Context(), tokenString)
		if aerr != nil {
			aerr.Log(srv.Logger.Error().Str("middleware", "auth"), true).
				Msg("Couldn't Verify Session Token")
			aerr.AbortWithErrorJson(c)
			return
		}
		if session == nil || time.Now().After(session.ExpiresAt) {
			appError.ErrorInvalidAuthToken().AbortWithErrorJson(c)
			return
		}
		c.Set("userID", session.UserID)

		c.Next()
	}
}
