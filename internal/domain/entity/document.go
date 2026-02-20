package entity

import "time"

type Document struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	SignerCount int       `json:"signer_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DocumentDetailResponse struct {
	Data    *DocumentResponse `json:"data"`
	Message string            `json:"message"`
	Status  int               `json:"status"`
	Meta    *Meta             `json:"meta,omitempty"`
}

type DocumentListResponse struct {
	Data    []DocumentResponse `json:"data"`
	Message string             `json:"message"`
	Status  int                `json:"status"`
	Meta    *Meta              `json:"meta,omitempty"`
}

type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	TotalCount int `json:"total_count"`
}

type DocumentResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Attributes DocumentAttributes `json:"attributes"`
}

type DocumentAttributes struct {
	Filename       string    `json:"filename"`
	Category       string    `json:"category"`
	DocURL         string    `json:"doc_url"`
	SigningStatus  string    `json:"signing_status"`
	StampingStatus string    `json:"stamping_status"`
	TypeOfMeterai  string    `json:"type_of_meterai"`
	Signers        []any     `json:"signers"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	TemplateID     *string   `json:"template_id"`
	IsAutosign     bool      `json:"is_autosign"`
}
