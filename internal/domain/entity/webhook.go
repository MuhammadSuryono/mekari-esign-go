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

// NAVLogEntry represents the log entry to send to NAV
type NAVLogEntry struct {
	ID                      string `json:"id"`
	InvoiceNumber           string `json:"invoice_number"`
	Filename                string `json:"filename"`
	EntryNo                 int    `json:"entry_no"`
	LocationDocumentOut     string `json:"location_document_out"`
	LocationDocumentProcess string `json:"location_document_process"`
	LocationDocumentIn      string `json:"location_document_in"`
	SigningStatus           string `json:"signing_status"`
	StampingStatus          string `json:"stamping_status"`
	SignersName1            string `json:"signersName1"`
	SignersEmail1           string `json:"signersEmail1"`
	SignersOrder1           string `json:"signersOrder1"`
	SignersSigningStatus1   string `json:"signersSigningStatus1"`
	SignersSigningDate1     string `json:"signersSigningDate1"`
}

// MapSigningStatus maps Mekari signing status to NAV status
func MapSigningStatus(status string) string {
	if status == "completed" {
		return "Completed"
	}
	return "Pending"
}

// MapStampingStatus maps Mekari stamping status to NAV status
func MapStampingStatus(status string) string {
	switch status {
	case "completed", "success":
		return "Completed"
	case "none":
		return ""
	default:
		return "Pending"
	}
}
