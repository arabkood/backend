package server

import (
	"strconv"

	"github.com/arabkood/backend/config"
	"github.com/arabkood/backend/internal/email"
	"github.com/arabkood/backend/pkg/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	Config       *config.Config
	Logger       *logger.Logger
	Router       *gin.Engine
	PostgresPool *pgxpool.Pool
	EmailService *email.ProductionEmailService
	Sqs          *sqs.Client
	AwsConfig    *aws.Config
	S3           *s3.Client
}

func (s *Server) Start() error {
	s.Logger.Info().Int("Port", s.Config.Server.Port).Msg("Server running on port")
	return s.Router.Run(":" + strconv.Itoa(s.Config.Server.Port))
}
