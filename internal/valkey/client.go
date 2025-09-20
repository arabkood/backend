package valkey

import (
	"context"
	"fmt"
	"time"

	"github.com/arabkood/backend/config"
	"github.com/arabkood/backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// Client wraps redis.Client with additional functionality
type Client struct {
	*redis.Client
	config *config.ValkeyConfig
	logger *logger.Logger
}

// NewClient creates a new Valkey/Redis client
func NewClient(cfg *config.ValkeyConfig, logger *logger.Logger) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("valkey config is required")
	}

	// Create Redis client options
	opts := &redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.GeneralDB,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		PoolTimeout:  30 * time.Second,
	}

	// Create the client
	rdb := redis.NewClient(opts)

	client := &Client{
		Client: rdb,
		config: cfg,
		logger: logger,
	}

	return client, nil
}

// TestConnection tests the connection to Valkey/Redis
func (c *Client) TestConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pong, err := c.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to ping Valkey: %w", err)
	}

	if pong != "PONG" {
		return fmt.Errorf("unexpected ping response: %s", pong)
	}

	c.logger.Info().Str("addr", c.config.Addr).Msg("Valkey connection successful")
	return nil
}

// Close closes the Valkey client connection
func (c *Client) Close() error {
	if err := c.Client.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Error closing Valkey client")
		return err
	}
	c.logger.Info().Msg("Valkey client closed successfully")
	return nil
}

// Health checks if the client is healthy
func (c *Client) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := c.Ping(ctx).Result()
	return err
}
