package entity

import "time"

// WebhookPayload represents the callback payload from Mekari eSign
type WebhookPayload struct {
	Data WebhookData `json:"data"`
}

// WebhookData represents the data in webhook payload
type WebhookData struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Attributes WebhookAttributes `json:"attributes"`
}

// WebhookAttributes represents the document attributes in webhook
type WebhookAttributes struct {
	Filename         string          `json:"filename"`
	Category         string          `json:"category"`
	DocURL           string          `json:"doc_url"`
	SigningStatus    string          `json:"signing_status"`  // pending, in_progress, completed
	StampingStatus   string          `json:"stamping_status"` // pending, success
	TypeOfMeterai    string          `json:"type_of_meterai"`
	Signers          []WebhookSigner `json:"signers"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	TemplateID       *string         `json:"template_id"`
	IsAutoSign       bool            `json:"is_autosign"`
	QRCodeAuditTrail bool            `json:"qr_code_audit_trail"`
}

// WebhookSigner represents a signer in webhook payload
type WebhookSigner struct {
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	Order      int     `json:"order"`
	Status     string  `json:"status"` // pending, completed
	SignedAt   *string `json:"signed_at"`
	SigningURL *string `json:"signing_url"`
	IsAutoSign bool    `json:"is_autosign"`
	Phone      string  `json:"phone"`
	Passcode   *string `json:"passcode"`
}

// DocumentInfo represents the document info stored in Redis
type DocumentInfo struct {
	DocumentID     string    `json:"document_id"`
	Email          string    `json:"email"`
	InvoiceNumber  string    `json:"invoice_number"`
	Filename       string    `json:"filename"`
	SigningStatus  string    `json:"signing_status"`
	StampingStatus string    `json:"stamping_status"`
	DocURL         string    `json:"doc_url"`
	UpdatedAt      time.Time `json:"updated_at"`
}
