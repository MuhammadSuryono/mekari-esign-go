package entity

// GlobalSignRequest represents the incoming request from client
type GlobalSignRequest struct {
	EntryNo          int               `json:"entry_no"`                    // Entry number for tracking
	Email            string            `json:"email"`                       // User email for OAuth token
	InvoiceNumber    string            `json:"invoice_number,omitempty"`    // Invoice number reference
	Signing          bool              `json:"signing"`                     // Signing only
	Stamping         bool              `json:"stamping"`                    // Stamping only
	Signers          []SignerRequest   `json:"signers"`                     // List of signers
	StampPositions   *StampPosition    `json:"stamp_positions,omitempty"`   // Stamp position (saved for later stamping)
	DocumentDeadline *DocumentDeadline `json:"document_deadline,omitempty"` // Optional deadline settings
}

// SignerRequest represents a signer in the client request
type SignerRequest struct {
	Name               string             `json:"name"`
	Email              string             `json:"email"`
	Phone              string             `json:"phone,omitempty"`
	Order              int                `json:"order,omitempty"`        // Signer order
	SignPage           int                `json:"sign_page"`              // Page number
	SignaturePositions *SignaturePosition `json:"signature_positions"`    // Signature placement position
	RequiresOTP        bool               `json:"requires_otp,omitempty"` // Require OTP verification
}

// SignaturePosition represents the position of signature on a document (client request)
type SignaturePosition struct {
	X      float64 `json:"x"` // X coordinate
	Y      float64 `json:"y"` // Y coordinate
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
	Page   int     `json:"page,omitempty"` // Page number (1-based)
}

// DocumentDeadline represents optional deadline settings
type DocumentDeadline struct {
	SigningDeadline          int    `json:"signing_deadline,omitempty"`             // value min 3 - max 31
	RecurringReminder        string `json:"recurring_reminder,omitempty"`           // none, daily, three_days, weekly, monthly
	DaysReminderAfterReceive int    `json:"days_reminder_after_received,omitempty"` // value min 1 - max 31
}

// StampPosition represents the position of e-meterai stamp on document
// This is stored temporarily and used later during stamping
type StampPosition struct {
	X      float64 `json:"x"` // X coordinate
	Y      float64 `json:"y"` // Y coordinate
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
	Page   int     `json:"page,omitempty"` // Page number (1-based)
}

// ========== Mekari API Request Structures ==========

// MekariSignRequest represents the actual request to Mekari API
type MekariSignRequest struct {
	Doc              string            `json:"doc"`                           // Base64 encoded PDF document
	Filename         string            `json:"filename"`                      // Document filename
	Signers          []MekariSigner    `json:"signers"`                       // List of signers for Mekari
	CallbackURL      string            `json:"callback_url,omitempty"`        // Webhook callback URL
	QRCodeAuditTrail *QRCodeAuditTrail `json:"qr_code_audit_trail,omitempty"` // QR code audit trail position
	DocumentDeadline *DocumentDeadline `json:"document_deadline,omitempty"`   // Deadline settings
	EntryNo          int               `json:"entry_no"`                      // Entry number for tracking
}

// MekariSigner represents a signer in Mekari API format
type MekariSigner struct {
	Name        string             `json:"name"`
	Email       string             `json:"email"`
	PhoneNumber *PhoneNumber       `json:"phone_number,omitempty"` // Phone number with country code
	RequiresOTP bool               `json:"requires_otp,omitempty"` // Require OTP verification
	Annotations []SignerAnnotation `json:"annotations"`            // Signature annotations
	Order       int                `json:"order,omitempty"`        // Signer order
}

// PhoneNumber represents phone number with country code
type PhoneNumber struct {
	CountryCode string `json:"country_code,omitempty"` // e.g., "62"
	Number      string `json:"number,omitempty"`       // e.g., "+62895355698652"
}

// SignerAnnotation represents annotation for signature placement
type SignerAnnotation struct {
	TypeOf        string   `json:"type_of,omitempty"`        // signature, meterai, initial, stamp (default: signature)
	SignatureType []string `json:"signature_type,omitempty"` // font, draw, image, qr_code
	Page          int      `json:"page"`                     // Page number
	PositionX     float64  `json:"position_x"`               // X coordinate
	PositionY     float64  `json:"position_y"`               // Y coordinate
	ElementWidth  float64  `json:"element_width"`            // Element width (default: 120)
	ElementHeight float64  `json:"element_height"`           // Element height (default: 100)
	CanvasWidth   float64  `json:"canvas_width"`             // Canvas width (default: 595 for A4)
	CanvasHeight  float64  `json:"canvas_height"`            // Canvas height (default: 841 for A4)
	AutoFields    []string `json:"auto_fields,omitempty"`    // date_signed, name, email, company
}

// QRCodeAuditTrail represents QR code audit trail position
type QRCodeAuditTrail struct {
	Page          int     `json:"page"`
	PositionX     float64 `json:"position_x"`
	PositionY     float64 `json:"position_y"`
	ElementWidth  float64 `json:"element_width"`
	ElementHeight float64 `json:"element_height"`
	CanvasWidth   float64 `json:"canvas_width"`
	CanvasHeight  float64 `json:"canvas_height"`
}

// ========== Response Structures ==========

// GlobalSignResult represents the result of global sign request
type GlobalSignResult struct {
	Success     bool            `json:"success"`
	NeedAuth    bool            `json:"need_auth,omitempty"`    // If true, need to authorize first
	RedirectURL string          `json:"redirect_url,omitempty"` // OAuth redirect URL if need_auth is true
	Data        *GlobalSignData `json:"data,omitempty"`         // Response data if success
	Message     string          `json:"message,omitempty"`
}

// GlobalSignResponse represents the API response for global sign request
type GlobalSignResponse struct {
	Data    *GlobalSignData `json:"data"`
	Message string          `json:"message,omitempty"`
	Meta    interface{}     `json:"meta,omitempty"`
}

// GlobalSignData represents the document data after sign request
type GlobalSignData struct {
	ID         string               `json:"id"`
	Type       string               `json:"type"`
	Attributes GlobalSignAttributes `json:"attributes"`
}

// GlobalSignAttributes represents the attributes of signed document
type GlobalSignAttributes struct {
	DocID      string         `json:"doc_id"`
	DocToken   string         `json:"doc_token"`
	DocURL     string         `json:"doc_url"`
	Filename   string         `json:"filename"`
	Status     string         `json:"status"`
	Message    string         `json:"message,omitempty"`
	Signers    []SignerStatus `json:"signers,omitempty"`
	CreatedAt  string         `json:"created_at,omitempty"`
	UpdatedAt  string         `json:"updated_at,omitempty"`
	ExpiryDate string         `json:"expiry_date,omitempty"`
}

// SignerStatus represents the signing status of each signer
type SignerStatus struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone,omitempty"`
	Status   string `json:"status"`
	SignedAt string `json:"signed_at,omitempty"`
	Sequence int    `json:"sequence,omitempty"`
}

// Default values for annotations
const (
	DefaultElementWidth  = 180.0 // Signature width (increased from 120 for better visibility)
	DefaultElementHeight = 140.0 // Signature height (increased from 100 for better visibility)
	DefaultCanvasWidth   = 595.0 // A4 width in points
	DefaultCanvasHeight  = 841.0 // A4 height in points
)

// Default signature types
var DefaultSignatureTypes = []string{"image", "qr_code", "draw"}

// Default auto fields
var DefaultAutoFields = []string{"date_signed", "name", "email", "company"}
