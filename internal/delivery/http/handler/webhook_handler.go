package handler

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"mekari-esign/internal/domain/entity"
	"mekari-esign/internal/usecase"
)

type WebhookHandler struct {
	usecase usecase.WebhookUsecase
	logger  *zap.Logger
}

func NewWebhookHandler(usecase usecase.WebhookUsecase, logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{
		usecase: usecase,
		logger:  logger,
	}
}

// MekariCallback godoc
// @Summary Mekari eSign webhook callback
// @Description Receives webhook callbacks from Mekari eSign when document status changes
// @Tags webhook
// @Accept json
// @Produce json
// @Param payload body entity.WebhookPayload true "Webhook payload"
// @Success 200 {object} entity.APIResponse
// @Failure 400 {object} entity.APIResponse
// @Failure 500 {object} entity.APIResponse
// @Router /webhook/mekari [post]
func (h *WebhookHandler) MekariCallback(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// Log raw body for debugging
	h.logger.Info("Received Mekari webhook callback",
		zap.String("body", string(c.Body())),
	)

	// Parse webhook payload
	var payload entity.WebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		h.logger.Error("Failed to parse webhook payload", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Invalid webhook payload"),
		)
	}

	// Validate payload
	if payload.Data.ID == "" {
		h.logger.Error("Missing document ID in webhook payload")
		return c.Status(fiber.StatusBadRequest).JSON(
			entity.NewErrorResponse("BAD_REQUEST", "Missing document ID"),
		)
	}

	// Process webhook
	if err := h.usecase.ProcessWebhook(ctx, &payload); err != nil {
		h.logger.Error("Failed to process webhook",
			zap.String("document_id", payload.Data.ID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(
			entity.NewErrorResponse("INTERNAL_ERROR", err.Error()),
		)
	}

	return c.JSON(entity.NewSuccessResponse(map[string]interface{}{
		"document_id":    payload.Data.ID,
		"signing_status": payload.Data.Attributes.SigningStatus,
		"processed":      true,
	}, "Webhook processed successfully"))
}
