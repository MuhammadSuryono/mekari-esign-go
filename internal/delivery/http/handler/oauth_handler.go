package handler

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"mekari-esign/internal/domain/entity"
	"mekari-esign/internal/infrastructure/oauth2"
	"mekari-esign/internal/usecase"
)

type OAuthHandler struct {
	usecase      usecase.OAuthUsecase
	tokenService oauth2.TokenService
	logger       *zap.Logger
}

func NewOAuthHandler(usecase usecase.OAuthUsecase, tokenService oauth2.TokenService, logger *zap.Logger) *OAuthHandler {
	return &OAuthHandler{
		usecase:      usecase,
		tokenService: tokenService,
		logger:       logger,
	}
}

// CheckCode godoc
// @Summary Check if OAuth code exists for email
// @Description Check if OAuth authorization code exists in database for the given email.
//
//	If not exists, returns redirect URL to Mekari OAuth login page.
//
// @Tags oauth
// @Accept json
// @Produce json
// @Param email query string true "Email address"
// @Success 200 {object} entity.APIResponse
// @Failure 400 {object} entity.APIResponse
// @Failure 500 {object} entity.APIResponse
// @Router /api/v1/oauth/check [get]
func (h *OAuthHandler) CheckCode(c *fiber.Ctx) error {
	ctx := c.UserContext()

	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Email is required"),
		)
	}

	response, err := h.usecase.CheckCode(ctx, email)
	if err != nil {
		h.logger.Error("Failed to check OAuth code", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(
			entity.NewErrorResponse("INTERNAL_ERROR", err.Error()),
		)
	}

	// If no code exists, redirect to Mekari OAuth
	if !response.HasCode {
		return c.JSON(entity.NewSuccessResponse(response, "No OAuth code found. Please authorize."))
	}

	return c.JSON(entity.NewSuccessResponse(response, "OAuth code exists"))
}

// CheckCodeAndRedirect godoc
// @Summary Check code and redirect to Mekari OAuth if not exists
// @Description Check if OAuth code exists. If not, redirect browser to Mekari OAuth login page.
// @Tags oauth
// @Param email query string true "Email address"
// @Success 302 "Redirect to Mekari OAuth"
// @Success 200 {object} entity.APIResponse
// @Failure 400 {object} entity.APIResponse
// @Router /api/v1/oauth/authorize [get]
func (h *OAuthHandler) CheckCodeAndRedirect(c *fiber.Ctx) error {
	ctx := c.UserContext()

	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Email is required"),
		)
	}

	response, err := h.usecase.CheckCode(ctx, email)
	if err != nil {
		h.logger.Error("Failed to check OAuth code", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(
			entity.NewErrorResponse("INTERNAL_ERROR", err.Error()),
		)
	}

	// If no code exists, redirect to Mekari OAuth
	if !response.HasCode {
		h.logger.Info("Redirecting to Mekari OAuth",
			zap.String("email", email),
			zap.String("redirect_url", response.RedirectURL),
		)
		return c.Redirect(response.RedirectURL, fiber.StatusFound)
	}

	return c.JSON(entity.NewSuccessResponse(response, "OAuth code already exists"))
}

// OAuthCallback godoc
// @Summary OAuth callback to receive authorization code
// @Description Callback endpoint that Mekari redirects to after user authorizes.
//
//	Saves the authorization code to database.
//
// @Tags oauth
// @Param code query string true "Authorization code from Mekari"
// @Param state query string false "State parameter (contains email)"
// @Param locale query string false "Locale"
// @Success 200 {object} entity.APIResponse
// @Failure 400 {object} entity.APIResponse
// @Failure 500 {object} entity.APIResponse
// @Router /redirect/oauth [get]
func (h *OAuthHandler) OAuthCallback(c *fiber.Ctx) error {
	ctx := c.UserContext()

	code := c.Query("code")
	state := c.Query("state") // Contains email
	locale := c.Query("locale")

	h.logger.Info("OAuth callback received",
		zap.String("code", code),
		zap.String("state", state),
		zap.String("locale", locale),
	)

	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Authorization code is required"),
		)
	}

	if state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "State (email) is required"),
		)
	}

	// Save code to database
	email := state // State contains the email
	if err := h.usecase.SaveCode(ctx, email, code); err != nil {
		h.logger.Error("Failed to save OAuth code", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(
			entity.NewErrorResponse("INTERNAL_ERROR", err.Error()),
		)
	}

	return c.JSON(entity.NewSuccessResponse(map[string]interface{}{
		"email":  email,
		"code":   code,
		"locale": locale,
	}, "OAuth code saved successfully"))
}

// SaveCode godoc
// @Summary Manually save OAuth code
// @Description Save OAuth authorization code for an email (manual endpoint)
// @Tags oauth
// @Accept json
// @Produce json
// @Param request body entity.SaveCodeRequest true "Save code request"
// @Success 200 {object} entity.APIResponse
// @Failure 400 {object} entity.APIResponse
// @Failure 500 {object} entity.APIResponse
// @Router /api/v1/oauth/save-code [post]
func (h *OAuthHandler) SaveCode(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req entity.SaveCodeRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Invalid request body"),
		)
	}

	if req.Email == "" || req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Email and code are required"),
		)
	}

	if err := h.usecase.SaveCode(ctx, req.Email, req.Code); err != nil {
		h.logger.Error("Failed to save OAuth code", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(
			entity.NewErrorResponse("INTERNAL_ERROR", err.Error()),
		)
	}

	return c.JSON(entity.NewSuccessResponse(map[string]string{
		"email": req.Email,
	}, "OAuth code saved successfully"))
}

// GetToken godoc
// @Summary Get OAuth token by email
// @Description Retrieve stored OAuth token information for an email
// @Tags oauth
// @Accept json
// @Produce json
// @Param email query string true "Email address"
// @Success 200 {object} entity.APIResponse
// @Failure 400 {object} entity.APIResponse
// @Failure 404 {object} entity.APIResponse
// @Failure 500 {object} entity.APIResponse
// @Router /api/v1/oauth/token [get]
func (h *OAuthHandler) GetToken(c *fiber.Ctx) error {
	ctx := c.UserContext()

	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Email is required"),
		)
	}

	token, err := h.usecase.GetOAuthToken(ctx, email)
	if err != nil {
		h.logger.Error("Failed to get OAuth token", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(
			entity.NewErrorResponse("INTERNAL_ERROR", err.Error()),
		)
	}

	if token == nil {
		return c.Status(fiber.StatusNotFound).JSON(
			entity.NewErrorResponse("NOT_FOUND", "OAuth token not found for this email"),
		)
	}

	return c.JSON(entity.NewSuccessResponse(token, "OAuth token retrieved successfully"))
}

// ExchangeCodeRequest represents the request to exchange code for tokens
type ExchangeCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// ExchangeCode godoc
// @Summary Exchange authorization code for access token
// @Description Exchange OAuth authorization code for access and refresh tokens, stores them in Redis
// @Tags oauth
// @Accept json
// @Produce json
// @Param request body ExchangeCodeRequest true "Exchange code request"
// @Success 200 {object} entity.APIResponse
// @Failure 400 {object} entity.APIResponse
// @Failure 500 {object} entity.APIResponse
// @Router /api/v1/oauth/exchange [post]
func (h *OAuthHandler) ExchangeCode(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req ExchangeCodeRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Invalid request body"),
		)
	}

	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Email is required"),
		)
	}

	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Code is required"),
		)
	}

	// Exchange code for tokens
	tokenResp, err := h.tokenService.ExchangeCode(ctx, req.Email, req.Code)
	if err != nil {
		h.logger.Error("Failed to exchange code for tokens", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(
			entity.NewErrorResponse("INTERNAL_ERROR", err.Error()),
		)
	}

	// Also save code to database for reference
	if err := h.usecase.SaveCode(ctx, req.Email, req.Code); err != nil {
		h.logger.Warn("Failed to save code to database", zap.Error(err))
	}

	return c.JSON(entity.NewSuccessResponse(map[string]interface{}{
		"email":        req.Email,
		"access_token": tokenResp.AccessToken,
		"token_type":   tokenResp.TokenType,
		"expires_in":   tokenResp.ExpiresIn,
	}, "Code exchanged for tokens successfully"))
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Refresh the access token using the stored refresh token
// @Tags oauth
// @Accept json
// @Produce json
// @Param email query string true "Email address"
// @Success 200 {object} entity.APIResponse
// @Failure 400 {object} entity.APIResponse
// @Failure 500 {object} entity.APIResponse
// @Router /api/v1/oauth/refresh [post]
func (h *OAuthHandler) RefreshAccessToken(c *fiber.Ctx) error {
	ctx := c.UserContext()

	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Email is required"),
		)
	}

	tokenResp, err := h.tokenService.RefreshToken(ctx, email)
	if err != nil {
		h.logger.Error("Failed to refresh token", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(
			entity.NewErrorResponse("INTERNAL_ERROR", err.Error()),
		)
	}

	return c.JSON(entity.NewSuccessResponse(map[string]interface{}{
		"email":        email,
		"access_token": tokenResp.AccessToken,
		"token_type":   tokenResp.TokenType,
		"expires_in":   tokenResp.ExpiresIn,
	}, "Token refreshed successfully"))
}
