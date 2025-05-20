package handler

import (
	appConfig "github.com/arabkood/backend/config"
	appLogger "github.com/arabkood/backend/pkg/logger"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemHandler struct {
	config    *appConfig.Config
	db        *pgxpool.Pool
	logger    *appLogger.Logger
	sqsClient *sqs.Client
	s3Client  *s3.Client
}

func NewItemHandler(config *appConfig.Config, db *pgxpool.Pool, logger *appLogger.Logger, sqsClient *sqs.Client, s3Client *s3.Client) *ItemHandler {
	return &ItemHandler{
		config,
		db,
		logger,
		sqsClient,
		s3Client,
	}
}
