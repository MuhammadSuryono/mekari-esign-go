package usecase

import (
	"context"
	"fmt"
	"net/url"

	"go.uber.org/zap"

	"mekari-esign/internal/config"
	"mekari-esign/internal/domain/entity"
	"mekari-esign/internal/domain/repository"
)

type OAuthUsecase interface {
	// CheckCode checks if OAuth code exists for the given email
	// Returns redirect URL if code doesn't exist
	CheckCode(ctx context.Context, email string) (*entity.CheckCodeResponse, error)

	// SaveCode saves the OAuth code for the given email
	SaveCode(ctx context.Context, email, code string) error

	// GetOAuthToken retrieves OAuth token by email
	GetOAuthToken(ctx context.Context, email string) (*entity.OAuthToken, error)

	// BuildAuthURL builds the Mekari OAuth authorization URL
	BuildAuthURL(email string) string
}

type oauthUsecase struct {
	repo   repository.OAuthRepository
	config *config.Config
	logger *zap.Logger
}

func NewOAuthUsecase(repo repository.OAuthRepository, cfg *config.Config, logger *zap.Logger) OAuthUsecase {
	return &oauthUsecase{
		repo:   repo,
		config: cfg,
		logger: logger,
	}
}

func (u *oauthUsecase) CheckCode(ctx context.Context, email string) (*entity.CheckCodeResponse, error) {
	u.logger.Info("Checking OAuth code for email", zap.String("email", email))

	// Validate email
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	// Check if code exists in database
	token, err := u.repo.FindByEmail(ctx, email)
	if err != nil {
		u.logger.Error("Failed to find OAuth token", zap.Error(err))
		return nil, err
	}

	response := &entity.CheckCodeResponse{}

	if token == nil || token.Code == "" {
		// Code doesn't exist, return redirect URL
		response.HasCode = false
		response.RedirectURL = u.BuildAuthURL(email)
		u.logger.Info("No OAuth code found, returning redirect URL",
			zap.String("email", email),
			zap.String("redirect_url", response.RedirectURL),
		)
	} else {
		// Code exists
		response.HasCode = true
		u.logger.Info("OAuth code found for email", zap.String("email", email))
	}

	return response, nil
}

func (u *oauthUsecase) SaveCode(ctx context.Context, email, code string) error {
	u.logger.Info("Saving OAuth code",
		zap.String("email", email),
		zap.String("code", code),
	)

	// Validate inputs
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if code == "" {
		return fmt.Errorf("code is required")
	}

	// Save code to database
	if err := u.repo.SaveCode(ctx, email, code); err != nil {
		u.logger.Error("Failed to save OAuth code", zap.Error(err))
		return err
	}

	u.logger.Info("OAuth code saved successfully", zap.String("email", email))
	return nil
}

func (u *oauthUsecase) GetOAuthToken(ctx context.Context, email string) (*entity.OAuthToken, error) {
	u.logger.Info("Getting OAuth token", zap.String("email", email))

	token, err := u.repo.FindByEmail(ctx, email)
	if err != nil {
		u.logger.Error("Failed to get OAuth token", zap.Error(err))
		return nil, err
	}

	return token, nil
}

func (u *oauthUsecase) BuildAuthURL(email string) string {
	// Build OAuth authorization URL
	// Format: https://sandbox-account.mekari.com/auth?client_id=xxx&response_type=code&scope=esign&lang=id&state=email
	baseURL := u.config.Mekari.AuthURL + "/auth"

	params := url.Values{}
	params.Set("client_id", u.config.Mekari.OAuth2.ClientID)
	params.Set("response_type", "code")
	params.Set("scope", "esign")
	params.Set("lang", "id")
	params.Set("state", email) // Use state to pass email back in callback

	return baseURL + "?" + params.Encode()
}
