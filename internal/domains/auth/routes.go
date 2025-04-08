package auth

import (
	"github.com/arabkood/backend/internal/domains/auth/handler"
	"github.com/arabkood/backend/internal/domains/auth/middlewares"
	"github.com/arabkood/backend/internal/server/interfaces/server"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, srv *server.Server) {
	authRoutes := router.Group("/auth")

	handler := handler.NewAuthHandler(srv.Config, srv.PostgresPool, srv.EmailService, srv.Logger)

	authRoutes.POST("/signup", handler.Signup)
	authRoutes.POST("/verify-email", handler.VerifyEmail)
	authRoutes.POST("/resend-email-verification", handler.ResendEmailVerification)

	authRoutes.POST("/signin", handler.Signin)
	authRoutes.POST("/signout", handler.Signout)

	authRoutes.POST("/forgot-password", handler.ForgotPassword)
	authRoutes.POST("/reset-password", handler.ResetPassword)

	// requires auth
	authRoutes.POST("/change-password", middlewares.AuthRequired(srv), handler.ChangePassword)
}
