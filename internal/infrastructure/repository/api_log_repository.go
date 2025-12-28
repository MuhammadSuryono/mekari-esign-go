package repository

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"mekari-esign/internal/domain/entity"
	"mekari-esign/internal/infrastructure/database"
)

// APILogRepository interface for API log operations
type APILogRepository interface {
	Save(ctx context.Context, log *entity.APILog) error
}

type apiLogRepository struct {
	db     *database.Database
	logger *zap.Logger
}

// NewAPILogRepository creates a new API log repository
func NewAPILogRepository(db *database.Database, logger *zap.Logger) APILogRepository {
	return &apiLogRepository{
		db:     db,
		logger: logger,
	}
}

// Save saves an API log entry to the database
func (r *apiLogRepository) Save(ctx context.Context, log *entity.APILog) error {
	query := `
		INSERT INTO api_logs (endpoint, method, request_body, response_body, status_code, duration_ms, email, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.DB.ExecContext(ctx, query,
		log.Endpoint,
		log.Method,
		log.RequestBody,
		log.ResponseBody,
		log.StatusCode,
		log.Duration,
		log.Email,
		log.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to save API log",
			zap.String("endpoint", log.Endpoint),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save API log: %w", err)
	}

	return nil
}
