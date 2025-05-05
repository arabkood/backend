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

	logger.Info().Msg("Configuring AWS SDK")

	// Base options - always include the region
	sdkOptions := []func(*awsConfig.LoadOptions) error{
		awsConfig.WithRegion(config.Aws.Region),
	}

	// Add a flag in your config, e.g., `Aws.UseLocalCredentials` or check `App.Environment`
	isLocalDev := config.App.Environment == "local"

	if isLocalDev {
		logger.Info().Msg("Using local AWS configuration (credentials file/profile/endpoint)")
		// Add options for local dev (credentials file, profile)
		if config.Aws.CredentialsPath != "" {
			sdkOptions = append(sdkOptions, awsConfig.WithSharedCredentialsFiles([]string{config.Aws.CredentialsPath}))
		}
		if config.Aws.Profile != "" {
			sdkOptions = append(sdkOptions, awsConfig.WithSharedConfigProfile(config.Aws.Profile))
		}

		if config.Aws.Endpoint != "" {
			sdkOptions = append(sdkOptions, awsConfig.WithBaseEndpoint(config.Aws.Endpoint))
		}
	} else {
		logger.Info().Msg("Using AWS SDK default credential chain (Expecting EC2 Instance Role)")
	}

	// Load the configuration using the determined options
	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background(), sdkOptions...)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load AWS SDK config")
	}
	if err != nil {
		panic(err)
	}

	sqsClient := sqs.NewFromConfig(awsCfg)

	s3Client := s3.NewFromConfig(awsCfg)

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

	// TODO: Consider using gin.New() and adding middleware selectively
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
