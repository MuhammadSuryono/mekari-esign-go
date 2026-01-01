package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"go.uber.org/zap"

	"mekari-esign/internal/config"
	"mekari-esign/internal/domain/entity"
	"mekari-esign/internal/domain/repository"
	"mekari-esign/internal/infrastructure/nav"
	"mekari-esign/internal/infrastructure/redis"
)

const (
	// Redis key prefix for document tracking
	documentKeyPrefix = "mekari:document:"
	// Redis key prefix for NAV setup cache (by entry_no)
	navSetupPrefix = "mekari:nav_setup:"
	// Redis key prefix for entry_no cache (by document_id)
	entryNoKeyPrefix = "mekari:entry_no:"
)

// DocumentMapping stores document info for webhook processing
type DocumentMapping struct {
	DocumentID       string                   `json:"document_id"`
	Email            string                   `json:"email"`
	InvoiceNumber    string                   `json:"invoice_number"`
	Filename         string                   `json:"filename"`
	StampPositions   *entity.StampPosition    `json:"stamp_positions,omitempty"`
	DocumentDeadline *entity.DocumentDeadline `json:"document_deadline,omitempty"`
	EntryNo          int                      `json:"entry_no"`
	Signing          bool                     `json:"signing"`
	Stamping         bool                     `json:"stamping"`
}

type EsignUsecase interface {
	GetProfile(ctx context.Context, email string) (*entity.Profile, error)
	GetDocuments(ctx context.Context, email string, page, perPage int) (*entity.DocumentListResponse, error)
	GlobalRequestSign(ctx context.Context, req *entity.GlobalSignRequest) (*entity.GlobalSignResult, error)
	// GetDocumentMapping retrieves email and invoice number by document ID from Redis
	GetDocumentMapping(ctx context.Context, documentID string) (*DocumentMapping, error)
}

type esignUsecase struct {
	config       *config.Config
	repo         repository.EsignRepository
	oauthUsecase OAuthUsecase
	navClient    *nav.Client
	redisClient  *redis.RedisClient
	logger       *zap.Logger
	wbUsecase    WebhookUsecase
}

func NewEsignUsecase(cfg *config.Config, repo repository.EsignRepository, oauthUsecase OAuthUsecase, navClient *nav.Client, redisClient *redis.RedisClient, logger *zap.Logger, webhook WebhookUsecase) EsignUsecase {
	return &esignUsecase{
		config:       cfg,
		repo:         repo,
		oauthUsecase: oauthUsecase,
		navClient:    navClient,
		redisClient:  redisClient,
		logger:       logger,
		wbUsecase:    webhook,
	}
}

func (u *esignUsecase) GetProfile(ctx context.Context, email string) (*entity.Profile, error) {
	u.logger.Info("Getting user profile", zap.String("email", email))

	profile, err := u.repo.GetProfile(ctx, email)
	if err != nil {
		u.logger.Error("Failed to get profile", zap.Error(err))
		return nil, err
	}

	u.logger.Info("Successfully retrieved profile",
		zap.String("email", profile.Attributes.Email),
		zap.String("name", profile.Attributes.Name),
	)

	return profile, nil
}

func (u *esignUsecase) GetDocuments(ctx context.Context, email string, page, perPage int) (*entity.DocumentListResponse, error) {
	u.logger.Info("Getting documents",
		zap.String("email", email),
		zap.Int("page", page),
		zap.Int("per_page", perPage),
	)

	// Set default values
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 10
	}

	docs, err := u.repo.GetDocuments(ctx, email, page, perPage)
	if err != nil {
		u.logger.Error("Failed to get documents", zap.Error(err))
		return nil, err
	}

	u.logger.Info("Successfully retrieved documents",
		zap.Int("count", len(docs.Data)),
	)

	return docs, nil
}

func (u *esignUsecase) GlobalRequestSign(ctx context.Context, req *entity.GlobalSignRequest) (*entity.GlobalSignResult, error) {
	u.logger.Info("Requesting global document sign",
		zap.String("email", req.Email),
		zap.String("invoice_number", req.InvoiceNumber),
		zap.Int("signers_count", len(req.Signers)),
	)

	// Fetch and cache NAV setup at the beginning (entry_no = 1 for new requests)
	entryNo := req.EntryNo
	if err := u.fetchAndCacheNAVSetup(ctx, entryNo); err != nil {
		u.logger.Warn("Failed to fetch NAV setup, will use config fallback",
			zap.Error(err),
		)
	}

	// Validate email (only required for OAuth2)
	if u.config.Mekari.IsOAuth2() && req.Email == "" {
		return nil, fmt.Errorf("email is required for OAuth2 authentication")
	}

	// Check if OAuth code exists for this email (only for OAuth2 auth)
	if u.config.Mekari.IsOAuth2() {
		codeCheck, err := u.oauthUsecase.CheckCode(ctx, req.Email)
		if err != nil {
			u.logger.Error("Failed to check OAuth code", zap.Error(err))
			return nil, fmt.Errorf("failed to check OAuth code: %w", err)
		}

		// If no code exists, return redirect URL
		if !codeCheck.HasCode {
			u.logger.Info("No OAuth code found, returning redirect URL",
				zap.String("email", req.Email),
				zap.String("redirect_url", codeCheck.RedirectURL),
			)
			return &entity.GlobalSignResult{
				Success:     false,
				NeedAuth:    true,
				RedirectURL: codeCheck.RedirectURL,
				Message:     "Authorization required. Please authorize first.",
			}, nil
		}
	}

	if req.Signing == false && req.Stamping == true {
		return u.stampingProcess(ctx, req, entryNo)
	}

	// Validate request
	if len(req.Signers) == 0 {
		return nil, fmt.Errorf("at least one signer is required")
	}

	// Validate signers
	for i, signer := range req.Signers {
		if signer.Name == "" {
			return nil, fmt.Errorf("signer %d: name is required", i+1)
		}
		if signer.Email == "" {
			return nil, fmt.Errorf("signer %d: email is required", i+1)
		}
		if signer.SignPage <= 0 {
			return nil, fmt.Errorf("signer %d: sign_page must be greater than 0", i+1)
		}
		if signer.SignaturePositions == nil {
			return nil, fmt.Errorf("signer %d: signature_positions is required", i+1)
		}
	}

	// Validate document deadline if provided
	if req.DocumentDeadline != nil {
		if req.DocumentDeadline.SigningDeadline != 0 {
			if req.DocumentDeadline.SigningDeadline < 3 || req.DocumentDeadline.SigningDeadline > 31 {
				return nil, fmt.Errorf("signing_deadline must be between 3 and 31")
			}
		}
		if req.DocumentDeadline.DaysReminderAfterReceive != 0 {
			if req.DocumentDeadline.DaysReminderAfterReceive < 1 || req.DocumentDeadline.DaysReminderAfterReceive > 31 {
				return nil, fmt.Errorf("days_reminder_after_received must be between 1 and 31")
			}
		}
		validReminders := map[string]bool{"": true, "none": true, "daily": true, "three_days": true, "weekly": true, "monthly": true}
		if !validReminders[req.DocumentDeadline.RecurringReminder] {
			return nil, fmt.Errorf("recurring_reminder must be one of: none, daily, three_days, weekly, monthly")
		}
	}

	// Call repository to make the API request
	response, err := u.repo.GlobalRequestSign(ctx, req.Email, req)
	if err != nil {
		u.logger.Error("Failed to request global sign",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		return nil, err
	}

	u.logger.Info("Successfully requested global sign",
		zap.String("doc_id", response.Data.Attributes.DocID),
		zap.String("status", response.Data.Attributes.Status),
	)

	// Save document mapping to Redis for webhook processing
	u.saveDocumentAndEntryNoToCache(ctx, req, response, entryNo)

	return &entity.GlobalSignResult{
		Success: true,
		Data:    response.Data,
		Message: "Document sign request created successfully",
	}, nil
}

func (u *esignUsecase) saveDocumentAndEntryNoToCache(ctx context.Context, req *entity.GlobalSignRequest, response *entity.GlobalSignResponse, entryNo int) {
	// Key: mekari:document:{document_id}, Value: JSON with all necessary info
	documentKey := documentKeyPrefix + response.Data.ID
	mapping := DocumentMapping{
		DocumentID:       response.Data.ID,
		Email:            req.Email,
		InvoiceNumber:    req.InvoiceNumber,
		Filename:         response.Data.Attributes.Filename,
		StampPositions:   req.StampPositions,
		DocumentDeadline: req.DocumentDeadline,
		EntryNo:          req.EntryNo,
		Signing:          req.Signing,
		Stamping:         req.Stamping,
	}
	mappingJSON, _ := json.Marshal(mapping)
	if err := u.redisClient.Set(ctx, documentKey, string(mappingJSON), 0); err != nil {
		u.logger.Warn("Failed to save document mapping to Redis",
			zap.String("document_id", response.Data.ID),
			zap.String("email", req.Email),
			zap.String("invoice_number", req.InvoiceNumber),
			zap.Error(err),
		)
		// Don't fail the request, just log warning
	} else {
		u.logger.Info("Document mapping saved to Redis",
			zap.String("key", documentKey),
			zap.String("email", req.Email),
			zap.String("invoice_number", req.InvoiceNumber),
			zap.Bool("has_stamp_positions", req.StampPositions != nil),
		)
	}

	byEntryNoKey := entryNoKeyPrefix + strconv.Itoa(entryNo)
	if err := u.redisClient.Set(ctx, byEntryNoKey, string(mappingJSON), 0); err != nil {
		u.logger.Warn("Failed to save entry no mapping to Redis",
			zap.String("document_id", response.Data.ID),
			zap.String("email", req.Email),
			zap.String("invoice_number", req.InvoiceNumber),
			zap.Error(err),
		)
		// Don't fail the request, just log warning
	}
}

func (u *esignUsecase) stampingProcess(ctx context.Context, req *entity.GlobalSignRequest, entryNo int) (*entity.GlobalSignResult, error) {
	// Get document mapping from Redis using document ID
	byEntryNoKey := entryNoKeyPrefix + strconv.Itoa(entryNo)
	mappingData, err := u.redisClient.Get(ctx, byEntryNoKey)
	if err != nil || mappingData == "" {
		u.logger.Error("Failed to get initial entry no mapping from Redis",
			zap.Int("entry_no", entryNo),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to stamping, Please sign first your document: %s", req.InvoiceNumber)
	}

	// Parse document mapping
	var mapping DocumentMapping
	if err := json.Unmarshal([]byte(mappingData), &mapping); err != nil {
		// Fallback: old format might be just email string
		return nil, fmt.Errorf("failed to parse initial entry no mapping: %w", err)
	}

	signedContent, err := u.wbUsecase.DownloadDocument(ctx, req.Email, fmt.Sprintf("/documents/%s/download", mapping.DocumentID))
	if err != nil {
		u.logger.Error("Failed to download signed document",
			zap.String("document_id", mapping.DocumentID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to download signed document: %w", err)
	}

	if err := u.wbUsecase.RequestStamping(ctx, req.Email, signedContent, mapping); err != nil {
		u.logger.Error("Failed to request stamping",
			zap.String("document_id", mapping.DocumentID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to request stamping: %w", err)
	}

	return &entity.GlobalSignResult{
		Success: true,
		Message: "Document stamping request created successfully",
	}, nil
}

// GetDocumentMapping retrieves email and invoice number by document ID from Redis
func (u *esignUsecase) GetDocumentMapping(ctx context.Context, documentID string) (*DocumentMapping, error) {
	documentKey := documentKeyPrefix + documentID

	data, err := u.redisClient.Get(ctx, documentKey)
	if err != nil {
		return nil, fmt.Errorf("document not found: %w", err)
	}

	var mapping DocumentMapping
	if err := json.Unmarshal([]byte(data), &mapping); err != nil {
		// Fallback: old format might be just email string
		return &DocumentMapping{Email: data}, nil
	}

	return &mapping, nil
}

// fetchAndCacheNAVSetup fetches NAV setup and caches it to Redis by entry_no
func (u *esignUsecase) fetchAndCacheNAVSetup(ctx context.Context, entryNo int) error {
	cacheKey := navSetupPrefix + strconv.Itoa(entryNo)

	// Check if already cached
	cached, err := u.redisClient.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		u.logger.Debug("NAV setup already cached", zap.Int("entry_no", entryNo))
		return nil
	}

	// Fetch from NAV
	setup, err := u.navClient.GetSetup(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch NAV setup: %w", err)
	}
	if setup == nil {
		return fmt.Errorf("NAV setup not available")
	}

	// Cache the setup (no expiration - permanent for this entry_no)
	setupJSON, _ := json.Marshal(setup)
	if err := u.redisClient.Set(ctx, cacheKey, string(setupJSON), 0); err != nil {
		return fmt.Errorf("failed to cache NAV setup: %w", err)
	}

	u.logger.Info("NAV setup fetched and cached",
		zap.Int("entry_no", entryNo),
		zap.String("key", cacheKey),
		zap.String("file_location_in", setup.FileLocationIn),
		zap.String("file_location_process", setup.FileLocationProcess),
		zap.String("file_location_out", setup.FileLocationOut),
	)

	return nil
}
