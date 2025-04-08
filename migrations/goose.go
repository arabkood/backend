package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"time"

	appConfig "github.com/arabkood/backend/config"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func MigrateUp(cfg *appConfig.Config) error {
	var db *sql.DB
	// setup database
	// First connect to 'postgres' database to create new database
	db, err := sql.Open("postgres", cfg.Database.GetPostgresURL())
	if err != nil {
		return err
	}
	defer db.Close()

	// Check if database exists
	var exists bool
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = '%s')`, cfg.Database.Name)
	err = db.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}
	// Create database only if it does not exist
	if !exists {
		_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE %s`, cfg.Database.Name))
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
	}

	time.Sleep(time.Second * 1)

	// Now connect to the actual database for migrations
	db, err = sql.Open("postgres", cfg.Database.GetDatabaseURL())
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return err
	}

	return nil
}
