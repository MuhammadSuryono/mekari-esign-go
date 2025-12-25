package entity

import "time"

// OAuthToken represents stored OAuth authorization code/token for a user
type OAuthToken struct {
	ID           int64     `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	Code         string    `json:"code" db:"code"`
	AccessToken  string    `json:"access_token,omitempty" db:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty" db:"refresh_token"`
	TokenType    string    `json:"token_type,omitempty" db:"token_type"`
	ExpiresAt    time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// CheckCodeRequest represents the request to check if code exists
type CheckCodeRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// CheckCodeResponse represents the response for check code endpoint
type CheckCodeResponse struct {
	HasCode     bool   `json:"has_code"`
	RedirectURL string `json:"redirect_url,omitempty"`
}

// SaveCodeRequest represents the request to save OAuth code
type SaveCodeRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required"`
}
