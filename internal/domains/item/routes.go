package item

import (
	"github.com/arabkood/backend/internal/domains/auth/middlewares"
	"github.com/arabkood/backend/internal/domains/item/handler"
	"github.com/arabkood/backend/internal/server/interfaces/server"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, srv *server.Server) {
	itemRoutes := router.Group("/item")

	handler := &handler.ItemHandler{
		Config:      srv.Config,
		DB:          srv.PostgresPool,
		Logger:      srv.Logger,
		AsynqClient: srv.AsynqClient,
		S3Client:    srv.S3,
	}

	// Require Internal auth
	// FIX: THIS IS NOT SECURE AT ALL, USER CAN SUBMIT ANYTHING THEY WANT IF THEY HAVE SCRET WORD
	itemRoutes.POST("/code/result", middlewares.InternalOnly(srv), handler.PostResult)

	withAuth := itemRoutes.Use(middlewares.AuthRequired(srv))
	withAuth.POST("/submit/:itemId", handler.Submit)
}
