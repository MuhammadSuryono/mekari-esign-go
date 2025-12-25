package usecase

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"mekari-esign/internal/config"
	"mekari-esign/internal/domain/entity"
	"mekari-esign/internal/infrastructure/document"
	"mekari-esign/internal/infrastructure/oauth2"
	"mekari-esign/internal/infrastructure/redis"
)

const (
	// Redis key prefix for document info
	documentInfoKeyPrefix = "mekari:document:info:"
)

type WebhookUsecase interface {
	// ProcessWebhook processes the webhook callback from Mekari eSign
	ProcessWebhook(ctx context.Context, payload *entity.WebhookPayload) error
}

type webhookUsecase struct {
	config       *config.Config
	redisClient  *redis.RedisClient
	docService   document.DocumentService
	tokenService oauth2.TokenService
	logger       *zap.Logger
	httpClient   *http.Client
}

func NewWebhookUsecase(
	cfg *config.Config,
	redisClient *redis.RedisClient,
	docService document.DocumentService,
	tokenService oauth2.TokenService,
	logger *zap.Logger,
) WebhookUsecase {
	return &webhookUsecase{
		config:       cfg,
		redisClient:  redisClient,
		docService:   docService,
		tokenService: tokenService,
		logger:       logger,
		httpClient: &http.Client{
			Timeout: cfg.Mekari.Timeout,
		},
	}
}

func (u *webhookUsecase) ProcessWebhook(ctx context.Context, payload *entity.WebhookPayload) error {
	documentID := payload.Data.ID

	u.logger.Info("Processing webhook callback",
		zap.String("document_id", documentID),
		zap.String("signing_status", payload.Data.Attributes.SigningStatus),
		zap.String("stamping_status", payload.Data.Attributes.StampingStatus),
		zap.String("filename", payload.Data.Attributes.Filename),
	)

	// Get document mapping from Redis using document ID
	documentKey := documentKeyPrefix + documentID
	mappingData, err := u.redisClient.Get(ctx, documentKey)
	if err != nil {
		u.logger.Error("Failed to get document mapping from Redis",
			zap.String("document_id", documentID),
			zap.Error(err),
		)
		return fmt.Errorf("document not found in Redis: %w", err)
	}

	// Parse document mapping
	var mapping DocumentMapping
	if err := json.Unmarshal([]byte(mappingData), &mapping); err != nil {
		// Fallback: old format might be just email string
		mapping = DocumentMapping{Email: mappingData}
	}

	email := mapping.Email
	invoiceNumber := mapping.InvoiceNumber

	// If invoice number is empty, try to extract from filename
	if invoiceNumber == "" {
		invoiceNumber = extractInvoiceNumber(payload.Data.Attributes.Filename)
	}

	// Build document info
	docInfo := &entity.DocumentInfo{
		DocumentID:     documentID,
		Email:          email,
		InvoiceNumber:  invoiceNumber,
		Filename:       payload.Data.Attributes.Filename,
		SigningStatus:  payload.Data.Attributes.SigningStatus,
		StampingStatus: payload.Data.Attributes.StampingStatus,
		DocURL:         payload.Data.Attributes.DocURL,
		UpdatedAt:      time.Now(),
	}

	// Save document info to Redis
	docInfoKey := documentInfoKeyPrefix + documentID
	docInfoJSON, err := json.Marshal(docInfo)
	if err != nil {
		u.logger.Error("Failed to marshal document info", zap.Error(err))
		return fmt.Errorf("failed to marshal document info: %w", err)
	}

	if err := u.redisClient.Set(ctx, docInfoKey, string(docInfoJSON), 0); err != nil {
		u.logger.Error("Failed to save document info to Redis", zap.Error(err))
		return fmt.Errorf("failed to save document info: %w", err)
	}

	u.logger.Info("Document info saved to Redis",
		zap.String("key", docInfoKey),
		zap.String("email", email),
		zap.String("invoice_number", invoiceNumber),
	)

	// Handle signing completed
	if payload.Data.Attributes.SigningStatus == "completed" && payload.Data.Attributes.StampingStatus != "success" {
		u.logger.Info("Signing completed",
			zap.String("document_id", documentID),
			zap.String("stamping_status", payload.Data.Attributes.StampingStatus),
		)

		// Download a signed document
		signedContent, err := u.downloadDocument(ctx, email, payload.Data.Attributes.DocURL)
		if err != nil {
			u.logger.Error("Failed to download signed document",
				zap.String("document_id", documentID),
				zap.Error(err),
			)
			return fmt.Errorf("failed to download signed document: %w", err)
		}

		// If stamping_status is "none" and we have stamp positions, request stamping
		if payload.Data.Attributes.StampingStatus == "none" && mapping.StampPositions != nil {
			u.logger.Info("Stamping required, sending stamp request",
				zap.String("document_id", documentID),
			)

			if err := u.requestStamping(ctx, email, signedContent, mapping); err != nil {
				u.logger.Error("Failed to request stamping",
					zap.String("document_id", documentID),
					zap.Error(err),
				)
				// Don't return error, just log it - stamping can be retried
			}
		} else {
			// No stamping needed, replace the file in progress folder
			if err := u.replaceDocumentInProgress(invoiceNumber, signedContent); err != nil {
				u.logger.Error("Failed to replace document in progress",
					zap.String("document_id", documentID),
					zap.Error(err),
				)
			}
		}
	}

	// Handle stamping completed - download final document and save to finish
	if payload.Data.Attributes.StampingStatus == "success" {
		u.logger.Info("Stamping completed, downloading final document",
			zap.String("document_id", documentID),
		)

		// Use the filename from mapping (original filename)
		originalFilename := mapping.Filename
		if originalFilename == "" {
			originalFilename = payload.Data.Attributes.Filename
		}

		finalContent, err := u.downloadDocument(ctx, email, payload.Data.Attributes.DocURL)
		if err != nil {
			u.logger.Error("Failed to download final document",
				zap.String("document_id", documentID),
				zap.Error(err),
			)
			return fmt.Errorf("failed to download final document: %w", err)
		}

		// Save to finish folder and delete from progress
		if err := u.docService.SaveToFinishAndDeleteProgress(originalFilename, finalContent); err != nil {
			u.logger.Error("Failed to save final document to finish folder",
				zap.String("document_id", documentID),
				zap.String("filename", originalFilename),
				zap.Error(err),
			)
			return fmt.Errorf("failed to save final document: %w", err)
		}

		u.logger.Info("Stamped document saved to finish folder",
			zap.String("document_id", documentID),
			zap.String("filename", originalFilename),
			zap.Int("size_bytes", len(finalContent)),
		)
	}

	return nil
}

func (u *webhookUsecase) downloadDocument(ctx context.Context, email, docURL string) ([]byte, error) {
	// Get access token for the email
	accessToken, err := u.tokenService.GetAccessToken(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Build full download URL
	downloadURL := u.config.Mekari.BaseURL + docURL

	u.logger.Info("Downloading document",
		zap.String("url", downloadURL),
		zap.String("email", email),
	)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))

	// Execute request
	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// Read response body (PDF content)
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read download response: %w", err)
	}

	u.logger.Info("Document downloaded successfully",
		zap.Int("size_bytes", len(content)),
	)

	return content, nil
}

func (u *webhookUsecase) replaceDocumentInProgress(invoiceNumber string, content []byte) error {
	// Find the filename in progress folder
	filename, err := u.docService.FindFilenameInProgress(invoiceNumber)
	if err != nil {
		return fmt.Errorf("failed to find file in progress: %w", err)
	}

	// Replace the file in progress folder
	if err := u.docService.ReplaceFileInProgress(filename, content); err != nil {
		return fmt.Errorf("failed to replace file: %w", err)
	}

	u.logger.Info("Document replaced in progress folder",
		zap.String("filename", filename),
		zap.Int("size_bytes", len(content)),
	)

	return nil
}

func (u *webhookUsecase) requestStamping(ctx context.Context, email string, signedPDFContent []byte, mapping DocumentMapping) error {
	// Get access token
	accessToken, err := u.tokenService.GetAccessToken(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// Encode PDF to base64
	base64Doc := base64.StdEncoding.EncodeToString(signedPDFContent)

	// Build stamp annotation from saved stamp positions
	annotations := []entity.StampAnnotation{}
	if mapping.StampPositions != nil {
		annotations = append(annotations, entity.StampAnnotation{
			Page:          mapping.StampPositions.Page,
			PositionX:     mapping.StampPositions.X,
			PositionY:     mapping.StampPositions.Y,
			ElementWidth:  80, // Default e-meterai size
			ElementHeight: 80,
			CanvasWidth:   595, // A4 width
			CanvasHeight:  841, // A4 height
			TypeOf:        "meterai",
		})
	}

	// Build stamp request
	stampReq := &entity.StampRequest{
		Doc:         base64Doc,
		Filename:    mapping.Filename,
		Annotations: annotations,
		//CallbackURL:      u.config.App.BaseURL + "/webhook/mekari",
		CallbackURL:      "https://webhook.site/acf98cf8-c888-4720-a907-32614ae8fbca",
		DocumentDeadline: mapping.DocumentDeadline,
	}

	// Marshal request body
	reqBody, err := json.Marshal(stampReq)
	if err != nil {
		return fmt.Errorf("failed to marshal stamp request: %w", err)
	}

	// Build stamp URL
	stampURL := u.config.Mekari.BaseURL + "/documents/stamp"

	u.logger.Info("Sending stamp request",
		zap.String("url", stampURL),
		zap.String("email", email),
		zap.String("filename", mapping.Filename),
		zap.Int("annotations_count", len(annotations)),
	)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stampURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create stamp request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))

	// Execute request
	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send stamp request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read stamp response: %w", err)
	}

	u.logger.Info("Stamp request response",
		zap.Int("status_code", resp.StatusCode),
		zap.String("body", string(respBody)),
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("stamp request failed: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var stampResp entity.StampResponse
	if err := json.Unmarshal(respBody, &stampResp); err != nil {
		return fmt.Errorf("failed to parse stamp response: %w", err)
	}

	u.logger.Info("Stamp request successful",
		zap.String("stamp_doc_id", stampResp.Data.ID),
		zap.String("status", stampResp.Data.Attributes.Status),
	)

	// Save stamp document ID -> original mapping to Redis
	// This is needed to retrieve the original filename when stamping completes
	stampDocKey := documentKeyPrefix + stampResp.Data.ID
	mappingJSON, _ := json.Marshal(mapping)
	if err := u.redisClient.Set(ctx, stampDocKey, string(mappingJSON), 0); err != nil {
		u.logger.Warn("Failed to save stamp document mapping to Redis",
			zap.String("stamp_doc_id", stampResp.Data.ID),
			zap.Error(err),
		)
	} else {
		u.logger.Info("Stamp document mapping saved to Redis",
			zap.String("key", stampDocKey),
			zap.String("email", email),
			zap.String("invoice_number", mapping.InvoiceNumber),
			zap.String("filename", mapping.Filename),
		)
	}

	return nil
}

// extractInvoiceNumber extracts invoice number from filename
// Example: INV-2024-001_contract.pdf -> INV-2024-001
func extractInvoiceNumber(filename string) string {
	// Remove extension
	name := filename
	if idx := lastIndex(name, "."); idx != -1 {
		name = name[:idx]
	}

	// Try to extract invoice number (assumes format like INV-2024-001 or similar)
	// For now, return the full name without extension
	// You can customize this based on your invoice number format
	return name
}

func lastIndex(s string, substr string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == substr {
			return i
		}
	}
	return -1
}
