package postgres

import (
	"context"
	"fmt"
	"time"

	appConfig "github.com/arabkood/backend/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxUUID "github.com/vgarvardt/pgx-google-uuid/v5"
)

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// TestConnection tests if the pool can successfully connect to the database
func TestConnection(ctx context.Context, pool *pgxpool.Pool) error {
	// Create a timeout context for the connection test
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Test basic connection by acquiring from pool
	conn, err := pool.Acquire(testCtx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Test ping
	if err := conn.Ping(testCtx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Test simple query
	var result int
	err = conn.QueryRow(testCtx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("test query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected test query result: got %d, want 1", result)
	}

	return nil
}

func NewPool(ctx context.Context, config *appConfig.DatabaseConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(config.GetDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	poolConfig.MaxConns = config.MaxConns
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnLifetimeJitter = config.MaxConnLifetimeJitter
	poolConfig.MinConns = config.MinConns
	poolConfig.HealthCheckPeriod = config.HealthCheckPeriod

	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxUUID.Register(conn.TypeMap())
		return nil
	}

	dbpool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Test the connection before returning the pool
	if err := TestConnection(ctx, dbpool); err != nil {
		dbpool.Close() // Clean up the pool if connection test fails
		return nil, fmt.Errorf("connection test failed: %w", err)
	}

	return dbpool, nil
}
