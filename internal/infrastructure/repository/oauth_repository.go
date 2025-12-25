package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"mekari-esign/internal/domain/entity"
	"mekari-esign/internal/domain/repository"
	"mekari-esign/internal/infrastructure/database"
)

type oauthRepository struct {
	db *database.Database
}

func NewOAuthRepository(db *database.Database) repository.OAuthRepository {
	return &oauthRepository{
		db: db,
	}
}

func (r *oauthRepository) FindByEmail(ctx context.Context, email string) (*entity.OAuthToken, error) {
	query := `
		SELECT id, email, code, access_token, refresh_token, token_type, expires_at, created_at, updated_at
		FROM oauth_tokens
		WHERE email = $1
	`

	var token entity.OAuthToken
	var expiresAt sql.NullTime

	err := r.db.DB.QueryRowContext(ctx, query, email).Scan(
		&token.ID,
		&token.Email,
		&token.Code,
		&token.AccessToken,
		&token.RefreshToken,
		&token.TokenType,
		&expiresAt,
		&token.CreatedAt,
		&token.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found, return nil without error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find oauth token by email: %w", err)
	}

	if expiresAt.Valid {
		token.ExpiresAt = expiresAt.Time
	}

	return &token, nil
}

func (r *oauthRepository) SaveCode(ctx context.Context, email, code string) error {
	// Upsert: Insert or update if exists (PostgreSQL syntax)
	query := `
		INSERT INTO oauth_tokens (email, code, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT(email) DO UPDATE SET
			code = EXCLUDED.code,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.DB.ExecContext(ctx, query, email, code, time.Now())
	if err != nil {
		return fmt.Errorf("failed to save oauth code: %w", err)
	}

	return nil
}

func (r *oauthRepository) UpdateTokens(ctx context.Context, email, accessToken, refreshToken, tokenType string, expiresAt int64) error {
	query := `
		UPDATE oauth_tokens
		SET access_token = $1, refresh_token = $2, token_type = $3, expires_at = $4, updated_at = $5
		WHERE email = $6
	`

	expiresTime := time.Now().Add(time.Duration(expiresAt) * time.Second)
	_, err := r.db.DB.ExecContext(ctx, query, accessToken, refreshToken, tokenType, expiresTime, time.Now(), email)
	if err != nil {
		return fmt.Errorf("failed to update oauth tokens: %w", err)
	}

	return nil
}
