package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/arabkood/backend/config"
	"github.com/arabkood/backend/internal/email"
	"github.com/arabkood/backend/internal/postgres"
	s3internal "github.com/arabkood/backend/internal/s3"
	"github.com/arabkood/backend/internal/server"
	iserver "github.com/arabkood/backend/internal/server/interfaces/server"
	"github.com/arabkood/backend/internal/valkey"
	"github.com/arabkood/backend/migrations"
	"github.com/arabkood/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
)

func main() {
	config, err := config.NewConfigDefault()
	if err != nil {
		panic(err)
	}

	logger, err := logger.NewLogger(config)
	if err != nil {
		panic(err)
	}

	logger.Info().Msg("Running DB migrations")
	err = migrations.MigrateUp(config)
	if err != nil {
		panic(err)
	}

	logger.Info().Msg("Configuring AWS SDK")

	logger.Info().Msg("Initializing a new Postgres Pool for the router")
	pgPool, err := postgres.NewPool(context.Background(), &config.Database)
	defer pgPool.Close()
	if err != nil {
		panic(err)
	}

	logger.Info().Msg("Testing the new Postgres Pool")
	err = postgres.TestConnection(context.Background(), pgPool)
	if err != nil {
		panic(err)
	}

	logger.Info().Msg("Initializing a new Email Service for the router")
	emailService, err := email.NewProductionEmailService(config)
	if err != nil {
		pgPool.Close()
		panic(err)
	}

	logger.Info().Msg("Initializing S3 Client")
	s3Client := s3internal.NewS3Client(config.S3)

	logger.Info().Msg("Initializing Valkey Client")
	valkeyClient, err := valkey.NewClient(&config.ValKey, logger)
	if err != nil {
		pgPool.Close()
		panic(err)
	}
	defer valkeyClient.Close()

	logger.Info().Msg("Testing Valkey Connection")
	err = valkeyClient.TestConnection(context.Background())
	if err != nil {
		pgPool.Close()
		valkeyClient.Close()
		panic(err)
	}

	logger.Info().Msg("Initializing Async Client")
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     config.ValKey.Addr,
		Password: config.ValKey.Password,
		DB:       config.ValKey.AsynqDB,
	})
	defer asynqClient.Close()

	// TODO: Consider using gin.New() and adding middleware selectively
	router := gin.Default()

	srv := &iserver.Server{
		Config:       config,
		Logger:       logger,
		Router:       router,
		PostgresPool: pgPool,
		EmailService: emailService,
		AsynqClient:  asynqClient,
		S3:           s3Client,
		ValkeyClient: valkeyClient,
	}

	err = server.SetupRoutes(router, srv)
	if err != nil {
		panic(err)
	}

	// Handle graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Start(); err != nil {
			logger.Error().Err(err).Msg("Server failed to start")
		}
	}()

	<-ctx.Done()
	logger.Info().Msg("Shutting down server...")
}
