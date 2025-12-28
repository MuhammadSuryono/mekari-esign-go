package entity

import "time"

// APILog represents a log entry for API requests to Mekari
type APILog struct {
	ID           int64     `json:"id"`
	Endpoint     string    `json:"endpoint"`
	Method       string    `json:"method"`
	RequestBody  string    `json:"request_body"`
	ResponseBody string    `json:"response_body"`
	StatusCode   int       `json:"status_code"`
	Duration     int64     `json:"duration_ms"`
	Email        string    `json:"email,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
