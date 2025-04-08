package middlewares

import (
	"time"

	appLogger "github.com/arabkood/backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

func Logger(logger *appLogger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()

		c.Next()

		logger.HTTPRequest(c.Request, c.Writer.Status(), time.Since(t), c.ClientIP())
	}
}
