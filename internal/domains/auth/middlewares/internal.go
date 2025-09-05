package middlewares

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"

	"github.com/arabkood/backend/internal/server/interfaces/server"
	"github.com/gin-gonic/gin"
)

func InternalOnly(srv *server.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// Validate client IP
		ip := net.ParseIP(clientIP)

		providedSecret := c.GetHeader("X-Internal-Secret")
		if providedSecret == "" {
			fmt.Println("no secret", ip)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Use constant time comparison to prevent timing attacks
		expectedSecret := []byte(srv.Config.Server.InternalAuthSecret)
		providedBytes := []byte(providedSecret)
		if subtle.ConstantTimeCompare(expectedSecret, providedBytes) != 1 {
			fmt.Println("wrong secret", ip)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}
