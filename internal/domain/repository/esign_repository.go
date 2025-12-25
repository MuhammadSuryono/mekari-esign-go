package repository

import (
	"context"

	"mekari-esign/internal/domain/entity"
)

type EsignRepository interface {
	GetProfile(ctx context.Context, email string) (*entity.Profile, error)
	GetDocuments(ctx context.Context, email string, page, perPage int) (*entity.DocumentListResponse, error)
	// GlobalRequestSign sends sign request to Mekari API
	// The doc (base64 PDF) will be fetched from invoice service based on invoice_number
	GlobalRequestSign(ctx context.Context, email string, req *entity.GlobalSignRequest) (*entity.GlobalSignResponse, error)
}
