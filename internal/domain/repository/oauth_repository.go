package repository

import (
	"context"

	"mekari-esign/internal/domain/entity"
)

type OAuthRepository interface {
	// FindByEmail finds OAuth token by email
	FindByEmail(ctx context.Context, email string) (*entity.OAuthToken, error)

	// SaveCode saves or updates OAuth code for an email
	SaveCode(ctx context.Context, email, code string) error

	// UpdateTokens updates access and refresh tokens
	UpdateTokens(ctx context.Context, email, accessToken, refreshToken, tokenType string, expiresAt int64) error
}
