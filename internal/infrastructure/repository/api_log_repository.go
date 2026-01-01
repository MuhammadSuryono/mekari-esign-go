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
	FindByInvoice(ctx context.Context, invoiceNumber string) ([]entity.APILog, error)
	FindAll(ctx context.Context, limit int) ([]entity.APILog, error)
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
		INSERT INTO api_logs (endpoint, invoice_no, entry_no, method, request_body, response_body, status_code, duration_ms, email, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.DB.ExecContext(ctx, query,
		log.Endpoint,
		log.InvoiceNo,
		log.EntryNo,
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

// FindByInvoice finds API logs by invoice number (searches in endpoint or request_body)
func (r *apiLogRepository) FindByInvoice(ctx context.Context, invoiceNumber string) ([]entity.APILog, error) {
	query := `
		SELECT id, endpoint, invoice_no, entry_no, method, request_body, response_body, status_code, duration_ms, email, created_at
		FROM api_logs
		WHERE endpoint LIKE $1 OR request_body LIKE $1
		ORDER BY created_at DESC
		LIMIT 100
	`

	searchPattern := "%" + invoiceNumber + "%"
	rows, err := r.db.DB.QueryContext(ctx, query, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query API logs: %w", err)
	}
	defer rows.Close()

	var logs []entity.APILog
	for rows.Next() {
		var log entity.APILog
		if err := rows.Scan(&log.ID, &log.Endpoint, &log.InvoiceNo, &log.EntryNo, &log.Method, &log.RequestBody, &log.ResponseBody, &log.StatusCode, &log.Duration, &log.Email, &log.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan API log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// FindAll finds all API logs with limit
func (r *apiLogRepository) FindAll(ctx context.Context, limit int) ([]entity.APILog, error) {
	query := `
		SELECT id, endpoint, invoice_no, entry_no, method, request_body, response_body, status_code, duration_ms, email, created_at
		FROM api_logs
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.DB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query API logs: %w", err)
	}
	defer rows.Close()

	var logs []entity.APILog
	for rows.Next() {
		var log entity.APILog
		if err := rows.Scan(&log.ID, &log.Endpoint, &log.InvoiceNo, &log.EntryNo, &log.Method, &log.RequestBody, &log.ResponseBody, &log.StatusCode, &log.Duration, &log.Email, &log.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan API log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}
