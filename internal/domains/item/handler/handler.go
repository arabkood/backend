package handler

import (
	appConfig "github.com/arabkood/backend/config"
	"github.com/arabkood/backend/internal/valkey"
	appLogger "github.com/arabkood/backend/pkg/logger"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hibiken/asynq"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemHandler struct {
	Config       *appConfig.Config
	DB           *pgxpool.Pool
	Logger       *appLogger.Logger
	AsynqClient  *asynq.Client
	S3Client     *s3.Client
	ValkeyClient *valkey.Client
}
