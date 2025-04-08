package handler

import (
	appConfig "github.com/arabkood/backend/config"
	appLogger "github.com/arabkood/backend/pkg/logger"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ModuleHandler struct {
	config    *appConfig.Config
	db        *pgxpool.Pool
	logger    *appLogger.Logger
	sqsClient *sqs.Client
	s3Client  *s3.Client
}

func NewModuleHandler(config *appConfig.Config, db *pgxpool.Pool, logger *appLogger.Logger, sqsClient *sqs.Client, s3Client *s3.Client) *ModuleHandler {
	return &ModuleHandler{
		config,
		db,
		logger,
		sqsClient,
		s3Client,
	}
}
