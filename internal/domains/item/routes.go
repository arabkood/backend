package item

import (
	"github.com/arabkood/backend/internal/domains/auth/middlewares"
	"github.com/arabkood/backend/internal/domains/item/handler"
	"github.com/arabkood/backend/internal/server/interfaces/server"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, srv *server.Server) {
	itemRoutes := router.Group("/item")

	handler := handler.NewItemHandler(srv.Config, srv.PostgresPool, srv.Logger, srv.Sqs, srv.S3)

	withAuth := itemRoutes.Use(middlewares.AuthRequired(srv))
	withAuth.POST("/submit/:itemId", handler.Submit)

	// Code
	codeRoutes := itemRoutes.Group("/code", middlewares.AuthRequired(srv))
	codeRoutes.POST("/attempt/:id", handler.Attempt)

	// Require Internal auth
	itemRoutes.POST("/code/result", middlewares.InternalOnly(srv), handler.PostResult)
}
