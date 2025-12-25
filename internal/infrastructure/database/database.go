package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"mekari-esign/internal/config"
)

type Database struct {
	DB     *sql.DB
	logger *zap.Logger
}

func NewDatabase(cfg *config.Config, logger *zap.Logger) (*Database, error) {
	// Build PostgreSQL connection string
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	db, err := sql.Open(cfg.Database.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connected successfully",
		zap.String("driver", cfg.Database.Driver),
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("dbname", cfg.Database.DBName),
	)

	database := &Database{
		DB:     db,
		logger: logger,
	}

	// Run migrations
	if err := database.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return database, nil
}

func (d *Database) migrate() error {
	// Create oauth_tokens table (PostgreSQL syntax)
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS oauth_tokens (
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) NOT NULL UNIQUE,
		code TEXT NOT NULL,
		access_token TEXT DEFAULT '',
		refresh_token TEXT DEFAULT '',
		token_type VARCHAR(50) DEFAULT '',
		expires_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := d.DB.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create oauth_tokens table: %w", err)
	}

	// Create index separately (PostgreSQL doesn't support IF NOT EXISTS in same statement)
	createIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_oauth_tokens_email ON oauth_tokens(email);
	`
	_, err = d.DB.Exec(createIndexSQL)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	d.logger.Info("Database migrations completed successfully")
	return nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}
