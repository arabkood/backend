package handler

import (
	appConfig "github.com/arabkood/backend/config"
	appLogger "github.com/arabkood/backend/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserHandler struct {
	config *appConfig.Config
	db     *pgxpool.Pool
	logger *appLogger.Logger
}

func NewUserHandler(config *appConfig.Config, db *pgxpool.Pool, logger *appLogger.Logger) *UserHandler {
	return &UserHandler{
		config,
		db,
		logger,
	}
}
