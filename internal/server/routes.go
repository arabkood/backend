package server

import (
	"errors"

	"github.com/arabkood/backend/internal/domains/auth"
	"github.com/arabkood/backend/internal/domains/module"
	"github.com/arabkood/backend/internal/domains/track"
	"github.com/arabkood/backend/internal/domains/user"
	"github.com/arabkood/backend/internal/middlewares"
	"github.com/arabkood/backend/internal/server/interfaces/server"
	"github.com/arabkood/backend/pkg/validators"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// SetupRoutes initializes API routes
func SetupRoutes(router *gin.Engine, srv *server.Server) error {
	router.Use(middlewares.CORS(srv.Config))

	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "UP"})
	})

	api := router.Group("/api/v1")

	api.Use(gin.Recovery())
	api.Use(middlewares.Logger(srv.Logger))

	if validatorEngine, ok := binding.Validator.Engine().(*validator.Validate); ok {
		srv.Logger.Info().Msg("Registering Gin Custom Validator...")
		validators.RegisterValidators(validatorEngine)
	} else {
		return errors.New("api shouldn't be allowed to launch without proper validators")
	}

	// Auth routes
	auth.RegisterRoutes(api, srv)
	module.RegisterRoutes(api, srv)
	track.RegisterRoutes(api, srv)
	user.RegisterRoutes(api, srv)

	return nil
}
