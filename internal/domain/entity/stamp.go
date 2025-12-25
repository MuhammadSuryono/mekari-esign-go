package entity

// StampRequest represents the request for e-meterai stamping
type StampRequest struct {
	Doc              string            `json:"doc"`                         // Base64 encoded PDF document
	Filename         string            `json:"filename"`                    // Document filename
	Annotations      []StampAnnotation `json:"annotations"`                 // Stamp annotations
	CallbackURL      string            `json:"callback_url,omitempty"`      // Webhook callback URL
	DocumentDeadline *DocumentDeadline `json:"document_deadline,omitempty"` // Deadline settings
}

// StampAnnotation represents the annotation for e-meterai stamp placement
type StampAnnotation struct {
	Page          int     `json:"page"`           // Page number (1-based)
	PositionX     float64 `json:"position_x"`     // X coordinate
	PositionY     float64 `json:"position_y"`     // Y coordinate
	ElementWidth  float64 `json:"element_width"`  // Width of stamp element
	ElementHeight float64 `json:"element_height"` // Height of stamp element
	CanvasWidth   float64 `json:"canvas_width"`   // Canvas width (default: 595 for A4)
	CanvasHeight  float64 `json:"canvas_height"`  // Canvas height (default: 841 for A4)
	TypeOf        string  `json:"type_of"`        // Type: "meterai"
}

// StampResponse represents the API response for stamp request
type StampResponse struct {
	Data    *StampData  `json:"data"`
	Message string      `json:"message,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// StampData represents the document data after stamp request
type StampData struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes StampAttributes `json:"attributes"`
}

// StampAttributes represents the attributes of stamped document
type StampAttributes struct {
	DocID          string `json:"doc_id"`
	Filename       string `json:"filename"`
	Status         string `json:"status"`
	StampingStatus string `json:"stamping_status"`
	DocURL         string `json:"doc_url"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}
