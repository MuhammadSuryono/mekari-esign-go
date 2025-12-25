package oauth2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"mekari-esign/internal/config"
	"mekari-esign/internal/domain/repository"
	"mekari-esign/internal/infrastructure/redis"
)

const (
	accessTokenKeyPrefix  = "mekari:access_token:"
	refreshTokenKeyPrefix = "mekari:refresh_token:"
)

// TokenResponse represents the OAuth2 token response from Mekari
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // seconds
	RefreshToken string `json:"refresh_token"`
}

// TokenService handles OAuth2 token operations
type TokenService interface {
	// ExchangeCode exchanges authorization code for access token
	ExchangeCode(ctx context.Context, email, code string) (*TokenResponse, error)

	// GetAccessToken retrieves access token from Redis, refreshes if expired
	GetAccessToken(ctx context.Context, email string) (string, error)

	// RefreshToken refreshes the access token using refresh token
	RefreshToken(ctx context.Context, email string) (*TokenResponse, error)

	// InvalidateTokens removes tokens from Redis (for logout or re-auth)
	InvalidateTokens(ctx context.Context, email string) error
}

type tokenService struct {
	config    *config.Config
	redis     *redis.RedisClient
	oauthRepo repository.OAuthRepository
	logger    *zap.Logger
	client    *http.Client
}

func NewTokenService(cfg *config.Config, redisClient *redis.RedisClient, oauthRepo repository.OAuthRepository, logger *zap.Logger) TokenService {
	return &tokenService{
		config:    cfg,
		redis:     redisClient,
		oauthRepo: oauthRepo,
		logger:    logger,
		client: &http.Client{
			Timeout: cfg.Mekari.Timeout,
		},
	}
}

func (s *tokenService) ExchangeCode(ctx context.Context, email, code string) (*TokenResponse, error) {
	s.logger.Info("Exchanging authorization code for tokens",
		zap.String("email", email),
	)

	// Build request body using OAuth2 credentials
	reqBody := map[string]string{
		"client_id":     s.config.Mekari.OAuth2.ClientID,
		"client_secret": s.config.Mekari.OAuth2.ClientSecret,
		"grant_type":    "authorization_code",
		"code":          code,
	}

	tokenResp, err := s.requestToken(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Store tokens in Redis
	if err := s.storeTokens(ctx, email, tokenResp); err != nil {
		return nil, fmt.Errorf("failed to store tokens: %w", err)
	}

	s.logger.Info("Successfully exchanged code for tokens",
		zap.String("email", email),
		zap.Int("expires_in", tokenResp.ExpiresIn),
	)

	return tokenResp, nil
}

func (s *tokenService) GetAccessToken(ctx context.Context, email string) (string, error) {
	accessTokenKey := accessTokenKeyPrefix + email

	// Try to get access token from Redis
	accessToken, err := s.redis.Get(ctx, accessTokenKey)
	if err == nil && accessToken != "" {
		s.logger.Debug("Access token found in Redis", zap.String("email", email))
		return accessToken, nil
	}

	// Access token not found or expired, try to refresh
	s.logger.Info("Access token not found, attempting to refresh",
		zap.String("email", email),
	)

	tokenResp, err := s.RefreshToken(ctx, email)
	if err == nil {
		return tokenResp.AccessToken, nil
	}

	// Refresh token also failed, try to get code from database and exchange
	s.logger.Info("Refresh token failed, attempting to exchange code",
		zap.String("email", email),
		zap.Error(err),
	)

	// Get OAuth code from database
	oauthToken, err := s.oauthRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("no valid token or code found for email %s: %w", email, err)
	}

	if oauthToken.Code == "" {
		return "", fmt.Errorf("no authorization code found for email %s, re-authorization required", email)
	}

	// Exchange code for new tokens
	tokenResp, err = s.ExchangeCode(ctx, email, oauthToken.Code)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code for new token: %w", err)
	}

	s.logger.Info("Successfully exchanged code for new access token",
		zap.String("email", email),
	)

	return tokenResp.AccessToken, nil
}

func (s *tokenService) RefreshToken(ctx context.Context, email string) (*TokenResponse, error) {
	refreshTokenKey := refreshTokenKeyPrefix + email

	// Get refresh token from Redis
	refreshToken, err := s.redis.Get(ctx, refreshTokenKey)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found, re-authorization required: %w", err)
	}

	s.logger.Info("Refreshing access token",
		zap.String("email", email),
	)

	// Build request body using OAuth2 credentials
	reqBody := map[string]string{
		"client_id":     s.config.Mekari.OAuth2.ClientID,
		"client_secret": s.config.Mekari.OAuth2.ClientSecret,
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}

	tokenResp, err := s.requestToken(ctx, reqBody)
	if err != nil {
		// If refresh fails, invalidate tokens and require re-auth
		s.InvalidateTokens(ctx, email)
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Store new tokens in Redis
	if err := s.storeTokens(ctx, email, tokenResp); err != nil {
		return nil, fmt.Errorf("failed to store refreshed tokens: %w", err)
	}

	s.logger.Info("Successfully refreshed tokens",
		zap.String("email", email),
		zap.Int("expires_in", tokenResp.ExpiresIn),
	)

	return tokenResp, nil
}

func (s *tokenService) InvalidateTokens(ctx context.Context, email string) error {
	accessTokenKey := accessTokenKeyPrefix + email
	refreshTokenKey := refreshTokenKeyPrefix + email

	if err := s.redis.Del(ctx, accessTokenKey, refreshTokenKey); err != nil {
		return fmt.Errorf("failed to invalidate tokens: %w", err)
	}

	s.logger.Info("Tokens invalidated", zap.String("email", email))
	return nil
}

func (s *tokenService) requestToken(ctx context.Context, reqBody map[string]string) (*TokenResponse, error) {
	tokenURL := s.config.Mekari.SsoBaseURL + "/oauth2/token"

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Log request
	s.logger.Info(">>> [OAUTH2-TOKEN-REQ]",
		zap.String("url", tokenURL),
		zap.String("code", reqBody["code"]),
		zap.String("client_id", reqBody["client_id"]),
		zap.String("client_secret", reqBody["client_secret"]),
		zap.String("grant_type", reqBody["grant_type"]),
	)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response
	s.logger.Info(">>> [OAUTH2-TOKEN-RESPONSE]",
		zap.Int("status", resp.StatusCode),
		zap.String("body", string(respBody)),
	)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token response: %w", err)
	}

	return &tokenResp, nil
}

func (s *tokenService) storeTokens(ctx context.Context, email string, tokenResp *TokenResponse) error {
	accessTokenKey := accessTokenKeyPrefix + email
	refreshTokenKey := refreshTokenKeyPrefix + email

	// Store access token with expiry (subtract 60 seconds for safety margin)
	accessTokenExpiry := time.Duration(tokenResp.ExpiresIn-60) * time.Second
	if accessTokenExpiry < 0 {
		accessTokenExpiry = time.Duration(tokenResp.ExpiresIn) * time.Second
	}

	if err := s.redis.Set(ctx, accessTokenKey, tokenResp.AccessToken, accessTokenExpiry); err != nil {
		return fmt.Errorf("failed to store access token: %w", err)
	}

	// Store refresh token with 22 days expiry
	refreshTokenExpiry := time.Duration(s.config.OAuth.RefreshTokenAgeDays) * 24 * time.Hour
	if err := s.redis.Set(ctx, refreshTokenKey, tokenResp.RefreshToken, refreshTokenExpiry); err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	s.logger.Debug("Tokens stored in Redis",
		zap.String("email", email),
		zap.Duration("access_token_expiry", accessTokenExpiry),
		zap.Duration("refresh_token_expiry", refreshTokenExpiry),
	)

	return nil
}
