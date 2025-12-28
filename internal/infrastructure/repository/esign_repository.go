package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"go.uber.org/zap"

	"mekari-esign/internal/config"
	"mekari-esign/internal/domain/entity"
	"mekari-esign/internal/domain/repository"
	"mekari-esign/internal/infrastructure/document"
	"mekari-esign/internal/infrastructure/httpclient"
	"mekari-esign/internal/infrastructure/redis"
)

const (
	navSetupPrefix = "mekari:nav_setup:"
)

type esignRepository struct {
	config      *config.Config
	client      httpclient.HTTPClient
	docService  document.DocumentService
	redisClient *redis.RedisClient
	logger      *zap.Logger
}

func NewEsignRepository(cfg *config.Config, client httpclient.HTTPClient, docService document.DocumentService, redisClient *redis.RedisClient, logger *zap.Logger) repository.EsignRepository {
	return &esignRepository{
		config:      cfg,
		client:      client,
		docService:  docService,
		redisClient: redisClient,
		logger:      logger,
	}
}

// getNAVSetup gets NAV setup from Redis cache
func (r *esignRepository) getNAVSetup(ctx context.Context, entryNo int) *entity.NAVSetup {
	cacheKey := navSetupPrefix + strconv.Itoa(entryNo)

	cached, err := r.redisClient.Get(ctx, cacheKey)
	if err != nil || cached == "" {
		return nil
	}

	var setup entity.NAVSetup
	if err := json.Unmarshal([]byte(cached), &setup); err != nil {
		return nil
	}

	return &setup
}

func (r *esignRepository) GetProfile(ctx context.Context, email string) (*entity.Profile, error) {
	var response entity.ProfileResponse

	reqCtx := &httpclient.RequestContext{Email: email}
	err := r.client.Get(ctx, reqCtx, "/profile", &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return response.Data, nil
}

func (r *esignRepository) GetDocuments(ctx context.Context, email string, page, perPage int) (*entity.DocumentListResponse, error) {
	var response entity.DocumentListResponse

	reqCtx := &httpclient.RequestContext{Email: email}
	path := fmt.Sprintf("/documents?page=%d&limit=%d", page, perPage)
	err := r.client.Get(ctx, reqCtx, path, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	return &response, nil
}

func (r *esignRepository) GlobalRequestSign(ctx context.Context, email string, req *entity.GlobalSignRequest) (*entity.GlobalSignResponse, error) {
	var response entity.GlobalSignResponse

	// Get NAV setup for folder paths
	navSetup := r.getNAVSetup(ctx, req.EntryNo)

	var base64Doc, filename string
	var err error

	// Find and load document from ready folder by invoice number
	if navSetup != nil && navSetup.FileLocationIn != "" {
		r.logger.Info("Using NAV Setup paths",
			zap.String("ready_path", navSetup.FileLocationIn),
			zap.String("progress_path", navSetup.FileLocationProcess),
		)
		base64Doc, filename, err = r.docService.FindDocumentByInvoiceNumberWithPath(req.InvoiceNumber, navSetup.FileLocationIn)
	} else {
		r.logger.Info("Using config paths (NAV Setup not available)")
		base64Doc, filename, err = r.docService.FindDocumentByInvoiceNumber(req.InvoiceNumber)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	// Convert SignerRequest to MekariSigner format with annotations
	mekariSigners := make([]entity.MekariSigner, len(req.Signers))
	for i, signer := range req.Signers {
		// Build annotation from signature position
		annotations := []entity.SignerAnnotation{}
		if signer.SignaturePositions != nil {
			// Determine page - use signature position page or sign_page
			page := signer.SignaturePositions.Page
			if page == 0 {
				page = signer.SignPage
			}

			annotation := entity.SignerAnnotation{
				TypeOf:        "signature",
				SignatureType: entity.DefaultSignatureTypes,
				Page:          page,
				PositionX:     signer.SignaturePositions.X,
				PositionY:     signer.SignaturePositions.Y,
				ElementWidth:  entity.DefaultElementWidth,
				ElementHeight: entity.DefaultElementHeight,
				CanvasWidth:   entity.DefaultCanvasWidth,
				CanvasHeight:  entity.DefaultCanvasHeight,
				AutoFields:    entity.DefaultAutoFields,
			}
			annotations = append(annotations, annotation)
		}

		// Build phone number if provided
		var phoneNumber *entity.PhoneNumber
		if signer.Phone != "" {
			phoneNumber = &entity.PhoneNumber{
				CountryCode: "62",
				Number:      signer.Phone,
			}
		}

		mekariSigners[i] = entity.MekariSigner{
			Name:        signer.Name,
			Email:       signer.Email,
			PhoneNumber: phoneNumber,
			RequiresOTP: signer.RequiresOTP,
			Annotations: annotations,
			Order:       signer.Order,
		}
	}

	// Build callback URL
	callbackURL := r.config.App.BaseURL + "/webhook/mekari"

	// Build Mekari API request with document from local folder
	// Note: StampPositions are NOT sent here - they are saved and used later for stamping
	mekariReq := &entity.MekariSignRequest{
		Doc:              base64Doc,
		Filename:         filename,
		Signers:          mekariSigners,
		CallbackURL:      callbackURL,
		DocumentDeadline: req.DocumentDeadline,
		EntryNo:          req.EntryNo,
	}

	reqCtx := &httpclient.RequestContext{Email: email}
	// Send JSON POST request to Mekari API
	err = r.client.Post(ctx, reqCtx, "/documents/request_global_sign", mekariReq, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to request global sign: %w", err)
	}

	// Move document from ready to progress folder after successful upload
	if navSetup != nil && navSetup.FileLocationIn != "" && navSetup.FileLocationProcess != "" {
		if err := r.docService.MoveToProgressWithPath(filename, navSetup.FileLocationIn, navSetup.FileLocationProcess); err != nil {
			r.logger.Warn("Failed to move document to progress",
				zap.String("filename", filename),
				zap.Error(err),
			)
		}
	} else {
		if err := r.docService.MoveToProgress(filename); err != nil {
			r.logger.Warn("Failed to move document to progress",
				zap.String("filename", filename),
				zap.Error(err),
			)
		}
	}

	return &response, nil
}
