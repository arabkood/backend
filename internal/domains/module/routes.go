package module

import (
	"github.com/arabkood/backend/internal/domains/auth/middlewares"
	"github.com/arabkood/backend/internal/domains/module/handler"
	"github.com/arabkood/backend/internal/server/interfaces/server"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, srv *server.Server) {
	moduleRoutes := router.Group("/module")

	handler := handler.NewModuleHandler(srv.Config, srv.PostgresPool, srv.Logger, srv.Sqs, srv.S3)

	moduleRoutes.GET("/:slug", handler.GetModule)

	// require auth
	moduleRoutesAuthorized := moduleRoutes.Group("/", middlewares.AuthRequired(srv))

	moduleRoutesAuthorized.POST("/attempt/:id", handler.Attempt)
	moduleRoutesAuthorized.GET("/attempt/:id", handler.GetAttempt)

	moduleRoutesAuthorized.GET("/submission/:id", handler.GetSubmission)
	// moduleRoutesAuthorized.GET("/result/:id")
	// moduleRoutesAuthorized.POST("/run/:id")

	// Require Internal auth
	moduleRoutes.POST("/result", middlewares.InternalOnly(srv), handler.PostResult)
}
