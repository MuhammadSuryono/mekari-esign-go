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

// NAVLogEntry represents the log entry to send to NAV (matches OData field names)
type NAVLogEntry struct {
	EntryNo         int    `json:"Entry_No"`
	InvoiceNo       string `json:"Invoice_No"`
	Filename        string `json:"File_Name_Invoice_No"`
	FilePathIn      string `json:"File_Path_In"`
	FilePathProcess string `json:"File_Path_Process"`
	FilePathOut     string `json:"File_Path_Out"`
	SigningStatus   string `json:"Signing_Status"`
	StampingStatus  string `json:"Stamping_Status"`
	// Signer 1
	Signer1Name          string `json:"Signer1_Name,omitempty"`
	Signer1Email         string `json:"Signer1_Email,omitempty"`
	Signer1Order         string `json:"Signer1_Order,omitempty"`
	Signer1SigningStatus string `json:"Signer1_Signing_Status,omitempty"`
	Signer1SigningDate   string `json:"Signer1_Signing_DateTime,omitempty"`
	// Signer 2
	Signer2Name          string `json:"Signer2_Name,omitempty"`
	Signer2Email         string `json:"Signer2_Email,omitempty"`
	Signer2Order         string `json:"Signer2_Order,omitempty"`
	Signer2SigningStatus string `json:"Signer2_Signing_Status,omitempty"`
	Signer2SigningDate   string `json:"Signer2_Signing_DateTime,omitempty"`
	// Signer 3
	Signer3Name          string `json:"Signer3_Name,omitempty"`
	Signer3Email         string `json:"Signer3_Email,omitempty"`
	Signer3Order         string `json:"Signer3_Order,omitempty"`
	Signer3SigningStatus string `json:"Signer3_Signing_Status,omitempty"`
	Signer3SigningDate   string `json:"Signer3_Signing_DateTime,omitempty"`
	// Signer 4 (optional, not in current API but for future use)
	Signer4Name          string `json:"Signer4_Name,omitempty"`
	Signer4Email         string `json:"Signer4_Email,omitempty"`
	Signer4Order         string `json:"Signer4_Order,omitempty"`
	Signer4SigningStatus string `json:"Signer4_Signing_Status,omitempty"`
	Signer4SigningDate   string `json:"Signer4_Signing_DateTime,omitempty"`
	// Signer 5 (optional)
	Signer5Name          string `json:"Signer5_Name,omitempty"`
	Signer5Email         string `json:"Signer5_Email,omitempty"`
	Signer5Order         string `json:"Signer5_Order,omitempty"`
	Signer5SigningStatus string `json:"Signer5_Signing_Status,omitempty"`
	Signer5SigningDate   string `json:"Signer5_Signing_DateTime,omitempty"`
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

// NAVSetupResponse represents the response from NAV Api_MekariSetup
type NAVSetupResponse struct {
	Value []NAVSetup `json:"value"`
}

// NAVSetup represents the Mekari setup configuration from NAV
type NAVSetup struct {
	PrimaryKey          string `json:"Primary_Key"`
	FileLocationIn      string `json:"File_Location_In"`
	FileLocationProcess string `json:"File_Location_Process"`
	FileLocationOut     string `json:"File_Location_Out"`
}
