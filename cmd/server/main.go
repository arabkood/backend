package main

import (
	"context"

	"github.com/arabkood/backend/config"
	"github.com/arabkood/backend/internal/email"
	"github.com/arabkood/backend/internal/postgres"
	"github.com/arabkood/backend/internal/server"
	iserver "github.com/arabkood/backend/internal/server/interfaces/server"
	"github.com/arabkood/backend/migrations"
	"github.com/arabkood/backend/pkg/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
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

	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(config.Aws.Region),
		awsConfig.WithSharedCredentialsFiles([]string{config.Aws.CredentialsPath}),
		awsConfig.WithSharedConfigProfile(config.Aws.Profile),
	)
	if err != nil {
		panic(err)
	}

	sqsClient := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(config.Aws.Endpoint)
	})

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(config.Aws.Endpoint)
	})

	logger.Info().Msg("Initializing a new Postgres Pool for the router")
	pgPool, err := postgres.NewPool(context.Background(), &config.Database)
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

	router := gin.Default()

	srv := &iserver.Server{
		Config:       config,
		Logger:       logger,
		Router:       router,
		PostgresPool: pgPool,
		EmailService: emailService,
		AwsConfig:    &awsCfg,
		Sqs:          sqsClient,
		S3:           s3Client,
	}

	err = server.SetupRoutes(router, srv)
	if err != nil {
		panic(err)
	}

	err = srv.Start()
	if err != nil {
		panic(err)
	}
}
