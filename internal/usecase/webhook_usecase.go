package usecase

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"mekari-esign/internal/config"
	"mekari-esign/internal/domain/entity"
	"mekari-esign/internal/infrastructure/document"
	"mekari-esign/internal/infrastructure/httpclient"
	"mekari-esign/internal/infrastructure/nav"
	"mekari-esign/internal/infrastructure/oauth2"
	"mekari-esign/internal/infrastructure/redis"
)

const (
	// Redis key prefix for document info
	documentInfoKeyPrefix = "mekari:document:info:"
	// Redis key prefix for NAV setup cache (by entry_no)
	navSetupKeyPrefix = "mekari:nav_setup:"
)

type WebhookUsecase interface {
	// ProcessWebhook processes the webhook callback from Mekari eSign
	ProcessWebhook(ctx context.Context, payload *entity.WebhookPayload) error
	RequestStamping(ctx context.Context, email string, signedPDFContent []byte, mapping DocumentMapping) error
	DownloadDocument(ctx context.Context, email, docURL string) ([]byte, error)
}

type webhookUsecase struct {
	config        *config.Config
	redisClient   *redis.RedisClient
	docService    document.DocumentService
	tokenService  oauth2.TokenService
	hmacSignature *httpclient.HMACSignature
	navClient     *nav.Client
	logger        *zap.Logger
	httpClient    *http.Client
}

func NewWebhookUsecase(
	cfg *config.Config,
	redisClient *redis.RedisClient,
	docService document.DocumentService,
	tokenService oauth2.TokenService,
	navClient *nav.Client,
	logger *zap.Logger,
) WebhookUsecase {
	uc := &webhookUsecase{
		config:       cfg,
		redisClient:  redisClient,
		docService:   docService,
		tokenService: tokenService,
		navClient:    navClient,
		logger:       logger,
		httpClient: &http.Client{
			Timeout: cfg.Mekari.Timeout,
		},
	}

	// Initialize HMAC signature if using HMAC auth
	if cfg.Mekari.IsHMAC() {
		uc.hmacSignature = httpclient.NewHMACSignature(cfg.Mekari.HMAC.ClientID, cfg.Mekari.HMAC.ClientSecret)
		logger.Info("WebhookUsecase initialized with HMAC authentication")
	} else {
		logger.Info("WebhookUsecase initialized with OAuth2 authentication")
	}

	return uc
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

	// Send log entry to NAV
	if err := u.sendNAVLogEntry(ctx, payload, &mapping); err != nil {
		u.logger.Warn("Failed to send log entry to NAV",
			zap.String("document_id", documentID),
			zap.Error(err),
		)
		// Don't fail the webhook processing, just log warning
	}

	// Get NAV setup for file paths
	var progressPath, finishPath string
	navSetup, err := u.getNAVSetupCached(ctx, mapping.EntryNo)
	if err != nil {
		u.logger.Warn("Failed to get NAV setup, using config values", zap.Error(err))
	}
	if navSetup != nil {
		progressPath = navSetup.FileLocationProcess
		finishPath = navSetup.FileLocationIn
		u.logger.Info("Using NAV setup paths",
			zap.String("progress_path", progressPath),
			zap.String("finish_path", finishPath),
		)
	}

	// Handle signing completed
	if payload.Data.Attributes.SigningStatus == "completed" && payload.Data.Attributes.StampingStatus != "success" {
		u.logger.Info("Signing completed",
			zap.String("document_id", documentID),
			zap.String("stamping_status", payload.Data.Attributes.StampingStatus),
		)

		// Download a signed document
		signedContent, err := u.DownloadDocument(ctx, email, payload.Data.Attributes.DocURL)
		if err != nil {
			u.logger.Error("Failed to download signed document",
				zap.String("document_id", documentID),
				zap.Error(err),
			)
			return fmt.Errorf("failed to download signed document: %w", err)
		}

		// If stamping_status is "none" and we have stamp positions, request stamping
		if payload.Data.Attributes.StampingStatus == "none" && mapping.StampPositions != nil && mapping.Stamping {
			u.logger.Info("Stamping required, sending stamp request",
				zap.String("document_id", documentID),
			)

			if err := u.replaceDocumentInProgress(invoiceNumber, signedContent, progressPath); err != nil {
				u.logger.Error("Failed to replace document in progress",
					zap.String("document_id", documentID),
					zap.Error(err),
				)
			}

			if err := u.RequestStamping(ctx, email, signedContent, mapping); err != nil {
				u.logger.Error("Failed to request stamping",
					zap.String("document_id", documentID),
					zap.Error(err),
				)
				// Don't return error, just log it - stamping can be retried
			}
		} else {
			// No stamping needed, replace the file in progress folder
			if err := u.replaceDocumentInProgress(invoiceNumber, signedContent, progressPath); err != nil {
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

		finalContent, err := u.DownloadDocument(ctx, email, payload.Data.Attributes.DocURL)
		if err != nil {
			u.logger.Error("Failed to download final document",
				zap.String("document_id", documentID),
				zap.Error(err),
			)
			return fmt.Errorf("failed to download final document: %w", err)
		}

		// Save to finish folder and delete from progress (use NAV setup paths if available)
		if finishPath != "" && progressPath != "" {
			err = u.docService.SaveToFinishAndDeleteProgressWithPath(originalFilename, finalContent, finishPath, progressPath)
		} else {
			err = u.docService.SaveToFinishAndDeleteProgress(originalFilename, finalContent)
		}
		if err != nil {
			u.logger.Error("Failed to save final document to finish folder",
				zap.String("document_id", documentID),
				zap.String("filename", originalFilename),
				zap.String("finish_path", finishPath),
				zap.Error(err),
			)
			return fmt.Errorf("failed to save final document: %w", err)
		}

		u.logger.Info("Stamped document saved to finish folder",
			zap.String("document_id", documentID),
			zap.String("filename", originalFilename),
			zap.String("finish_path", finishPath),
			zap.Int("size_bytes", len(finalContent)),
		)
	}

	return nil
}

func (u *webhookUsecase) DownloadDocument(ctx context.Context, email, docURL string) ([]byte, error) {
	// Build full download URL
	downloadURL := u.config.Mekari.BaseURL + docURL

	u.logger.Info("Downloading document",
		zap.String("url", downloadURL),
		zap.String("email", email),
		zap.String("auth_type", u.config.Mekari.AuthType),
	)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	// Set auth headers based on config
	if u.config.Mekari.IsHMAC() {
		// Use HMAC authentication
		if err := u.hmacSignature.SignRequest(req); err != nil {
			return nil, fmt.Errorf("failed to sign request with HMAC: %w", err)
		}
		u.logger.Debug("Using HMAC authentication for download request")
	} else {
		// Use OAuth2 authentication
		accessToken, err := u.tokenService.GetAccessToken(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("failed to get access token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
		u.logger.Debug("Using OAuth2 authentication for download request")
	}

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

func (u *webhookUsecase) replaceDocumentInProgress(invoiceNumber string, content []byte, progressPath string) error {
	var filename string
	var err error

	// Find the filename in progress folder (use NAV setup path if provided)
	if progressPath != "" {
		filename, err = u.docService.FindFilenameInProgressWithPath(invoiceNumber, progressPath)
	} else {
		filename, err = u.docService.FindFilenameInProgress(invoiceNumber)
	}
	if err != nil {
		return fmt.Errorf("failed to find file in progress: %w", err)
	}

	// Replace the file in progress folder
	if progressPath != "" {
		err = u.docService.ReplaceFileInProgressWithPath(filename, content, progressPath)
	} else {
		err = u.docService.ReplaceFileInProgress(filename, content)
	}
	if err != nil {
		return fmt.Errorf("failed to replace file: %w", err)
	}

	u.logger.Info("Document replaced in progress folder",
		zap.String("filename", filename),
		zap.String("progress_path", progressPath),
		zap.Int("size_bytes", len(content)),
	)

	return nil
}

func (u *webhookUsecase) RequestStamping(ctx context.Context, email string, signedPDFContent []byte, mapping DocumentMapping) error {
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
		CallbackURL: u.config.App.BaseURL + "/webhook/mekari",
		//CallbackURL:      "https://webhook.site/acf98cf8-c888-4720-a907-32614ae8fbca",
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
		zap.String("auth_type", u.config.Mekari.AuthType),
	)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stampURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create stamp request: %w", err)
	}

	// Set Content-Type header
	req.Header.Set("Content-Type", "application/json")

	// Set auth headers based on config
	if u.config.Mekari.IsHMAC() {
		// Use HMAC authentication
		if err := u.hmacSignature.SignRequest(req); err != nil {
			return fmt.Errorf("failed to sign request with HMAC: %w", err)
		}
		u.logger.Debug("Using HMAC authentication for stamp request")
	} else {
		// Use OAuth2 authentication
		accessToken, err := u.tokenService.GetAccessToken(ctx, email)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
		u.logger.Debug("Using OAuth2 authentication for stamp request")
	}

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

// getNAVSetupCached gets NAV setup from cache or fetches from NAV
func (u *webhookUsecase) getNAVSetupCached(ctx context.Context, entryNo int) (*entity.NAVSetup, error) {
	cacheKey := navSetupKeyPrefix + strconv.Itoa(entryNo)

	// Try to get from cache
	cached, err := u.redisClient.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var setup entity.NAVSetup
		if err := json.Unmarshal([]byte(cached), &setup); err == nil {
			u.logger.Debug("Using cached NAV setup", zap.Int("entry_no", entryNo))
			return &setup, nil
		}
	}

	// Fetch from NAV
	setup, err := u.navClient.GetSetup(ctx)
	if err != nil {
		return nil, err
	}
	if setup == nil {
		return nil, nil
	}

	// Cache the setup (no expiration - permanent for this entry_no)
	setupJSON, _ := json.Marshal(setup)
	if err := u.redisClient.Set(ctx, cacheKey, string(setupJSON), 0); err != nil {
		u.logger.Warn("Failed to cache NAV setup", zap.Error(err))
	} else {
		u.logger.Info("NAV setup cached",
			zap.Int("entry_no", entryNo),
			zap.String("key", cacheKey),
		)
	}

	return setup, nil
}

// sendNAVLogEntry sends a log entry to NAV using PATCH
func (u *webhookUsecase) sendNAVLogEntry(ctx context.Context, payload *entity.WebhookPayload, mapping *DocumentMapping) error {
	// Default locations from config
	locationIn := u.config.Document.BasePath + "/" + u.config.Document.ReadyFolder
	locationProcess := u.config.Document.BasePath + "/" + u.config.Document.ProgressFolder
	locationOut := u.config.Document.BasePath + "/" + u.config.Document.FinishFolder

	// Get NAV setup (cached by entry_no)
	navSetup, err := u.getNAVSetupCached(ctx, mapping.EntryNo)
	if err != nil {
		u.logger.Warn("Failed to get NAV setup, using config values", zap.Error(err))
	} else if navSetup != nil {
		locationIn = navSetup.FileLocationIn
		locationProcess = navSetup.FileLocationProcess
		locationOut = navSetup.FileLocationOut
	}

	// Build NAV log entry with OData field names
	navEntry := &entity.NAVLogEntry{
		EntryNo:         mapping.EntryNo,
		InvoiceNo:       mapping.InvoiceNumber,
		Filename:        payload.Data.Attributes.Filename,
		FilePathIn:      locationIn,
		FilePathProcess: locationProcess,
		FilePathOut:     locationOut,
		SigningStatus:   entity.MapSigningStatus(payload.Data.Attributes.SigningStatus),
		StampingStatus:  entity.MapStampingStatus(payload.Data.Attributes.StampingStatus),
	}

	// Populate signer info (up to 3 signers based on NAV API)
	signers := payload.Data.Attributes.Signers

	// Signer 1
	if len(signers) > 0 && navEntry.StampingStatus != "Completed" {
		//navEntry.Signer1Name = signers[0].Name
		//navEntry.Signer1Email = signers[0].Email
		//navEntry.Signer1Order = strconv.Itoa(signers[0].Order)
		navEntry.Signer1SigningStatus = entity.MapSigningStatus(signers[0].Status)
		if signers[0].SignedAt != nil {
			navEntry.Signer1SigningDate = *signers[0].SignedAt
		} else {
			navEntry.Signer1SigningDate = "0001-01-01T00:00:00Z"
		}
	}

	// Signer 2
	if len(signers) > 1 && navEntry.StampingStatus != "Completed" {
		//navEntry.Signer2Name = signers[1].Name
		//navEntry.Signer2Email = signers[1].Email
		//navEntry.Signer2Order = strconv.Itoa(signers[1].Order)
		navEntry.Signer2SigningStatus = entity.MapSigningStatus(signers[1].Status)
		if signers[1].SignedAt != nil {
			navEntry.Signer2SigningDate = *signers[1].SignedAt
		} else {
			navEntry.Signer2SigningDate = "0001-01-01T00:00:00Z"
		}
	}

	// Signer 3
	if len(signers) > 2 && navEntry.StampingStatus != "Completed" {
		//navEntry.Signer3Name = signers[2].Name
		//navEntry.Signer3Email = signers[2].Email
		//navEntry.Signer3Order = strconv.Itoa(signers[2].Order)
		navEntry.Signer3SigningStatus = entity.MapSigningStatus(signers[2].Status)
		if signers[2].SignedAt != nil {
			navEntry.Signer3SigningDate = *signers[2].SignedAt
		} else {
			navEntry.Signer3SigningDate = "0001-01-01T00:00:00Z"
		}
	}

	return u.navClient.UpdateLogEntry(ctx, navEntry)
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
