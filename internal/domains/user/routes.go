package user

import (
	"github.com/arabkood/backend/internal/domains/auth/middlewares"
	"github.com/arabkood/backend/internal/domains/user/handler"
	"github.com/arabkood/backend/internal/server/interfaces/server"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, srv *server.Server) {
	userRoutes := router.Group("/user")

	handler := handler.NewUserHandler(srv.Config, srv.PostgresPool, srv.Logger)

	// FIX: THIS IS NOT SECURE AT ALL, USER CAN SUBMIT ANYTHING THEY WANT IF THEY HAVE SCRET WORD
	router.POST("/internal/user/stripesync", middlewares.InternalOnly(srv), handler.StripeSync)

	// require auth
	{
		meRoutes := userRoutes.Group("/me")
		meRoutes.Use(middlewares.AuthRequired(srv))
		meRoutes.GET("/profile", handler.GetMe)
		meRoutes.PUT("/profile", handler.UpdateMe)
		meRoutes.GET("/stats", handler.GetMyStats)
	}
}
