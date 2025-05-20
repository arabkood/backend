package track

import (
	"github.com/arabkood/backend/internal/domains/auth/middlewares"
	"github.com/arabkood/backend/internal/domains/track/handler"
	"github.com/arabkood/backend/internal/server/interfaces/server"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, srv *server.Server) {
	trackRoutes := router.Group("/track")

	handler := handler.NewTrackHandler(srv.Config, srv.PostgresPool, srv.Logger)

	// require auth
	trackRoutes.POST("/start", middlewares.AuthRequired(srv), handler.StartTrack)
}
