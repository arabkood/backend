package handler

import (
	appConfig "github.com/arabkood/backend/config"
	appLogger "github.com/arabkood/backend/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TrackHandler struct {
	config *appConfig.Config
	db     *pgxpool.Pool
	logger *appLogger.Logger
}

func NewTrackHandler(config *appConfig.Config, db *pgxpool.Pool, logger *appLogger.Logger) *TrackHandler {
	return &TrackHandler{
		config,
		db,
		logger,
	}
}
