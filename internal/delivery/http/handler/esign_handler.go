package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"mekari-esign/internal/domain/entity"
	"mekari-esign/internal/usecase"
)

type EsignHandler struct {
	usecase usecase.EsignUsecase
	logger  *zap.Logger
}

func NewEsignHandler(usecase usecase.EsignUsecase, logger *zap.Logger) *EsignHandler {
	return &EsignHandler{
		usecase: usecase,
		logger:  logger,
	}
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get the authenticated user's profile from Mekari eSign
// @Tags esign
// @Accept json
// @Produce json
// @Param email query string true "User email for OAuth token"
// @Success 200 {object} entity.APIResponse
// @Failure 400 {object} entity.APIResponse
// @Failure 500 {object} entity.APIResponse
// @Router /api/v1/esign/profile [get]
func (h *EsignHandler) GetProfile(c *fiber.Ctx) error {
	ctx := c.UserContext()

	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Email is required"),
		)
	}

	profile, err := h.usecase.GetProfile(ctx, email)
	if err != nil {
		h.logger.Error("Failed to get profile", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(
			entity.NewErrorResponse("INTERNAL_ERROR", err.Error()),
		)
	}

	return c.JSON(entity.NewSuccessResponse(profile, "Profile retrieved successfully"))
}

// GetDocuments godoc
// @Summary Get documents
// @Description Get list of documents from Mekari eSign
// @Tags esign
// @Accept json
// @Produce json
// @Param email query string true "User email for OAuth token"
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(10)
// @Success 200 {object} entity.APIResponse
// @Failure 400 {object} entity.APIResponse
// @Failure 500 {object} entity.APIResponse
// @Router /api/v1/esign/documents [get]
func (h *EsignHandler) GetDocuments(c *fiber.Ctx) error {
	ctx := c.UserContext()

	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Email is required"),
		)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "10"))

	docs, err := h.usecase.GetDocuments(ctx, email, page, perPage)
	if err != nil {
		h.logger.Error("Failed to get documents", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(
			entity.NewErrorResponse("INTERNAL_ERROR", err.Error()),
		)
	}

	return c.JSON(entity.NewSuccessResponse(docs, "Documents retrieved successfully"))
}

// GlobalRequestSign godoc
// @Summary Request global document signing
// @Description Request signatures from multiple signers. Validates OAuth code first.
// @Tags esign
// @Accept json
// @Produce json
// @Param request body entity.GlobalSignRequest true "Global sign request"
// @Success 201 {object} entity.APIResponse
// @Success 200 {object} entity.APIResponse "Need authorization - returns redirect URL"
// @Failure 400 {object} entity.APIResponse
// @Failure 500 {object} entity.APIResponse
// @Router /api/v1/esign/documents/request-sign [post]
func (h *EsignHandler) GlobalRequestSign(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// Parse JSON request body
	var req entity.GlobalSignRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Invalid request body"),
		)
	}

	// Call usecase (which handles OAuth validation)
	result, err := h.usecase.GlobalRequestSign(ctx, &req)
	if err != nil {
		h.logger.Error("Failed to request global sign", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(
			entity.NewErrorResponse("INTERNAL_ERROR", err.Error()),
		)
	}

	// If authorization is needed, return 200 with redirect URL
	if result.NeedAuth {
		return c.Status(fiber.StatusOK).JSON(
			entity.NewSuccessResponse(result, result.Message),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(
		entity.NewSuccessResponse(result, result.Message),
	)
}
