package handler

import (
	appConfig "github.com/arabkood/backend/config"
	emailService "github.com/arabkood/backend/internal/email"
	appLogger "github.com/arabkood/backend/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	config   *appConfig.Config
	db       *pgxpool.Pool
	emailSvc *emailService.ProductionEmailService
	logger   *appLogger.Logger
}

func NewAuthHandler(config *appConfig.Config, db *pgxpool.Pool, emailSvc *emailService.ProductionEmailService, logger *appLogger.Logger) *AuthHandler {
	return &AuthHandler{
		config,
		db,
		emailSvc,
		logger,
	}
}
