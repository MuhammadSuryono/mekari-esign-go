package httpclient

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

type HMACSignature struct {
	ClientID     string
	ClientSecret string
}

func NewHMACSignature(clientID, clientSecret string) *HMACSignature {
	return &HMACSignature{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}

// GenerateSignature generates HMAC-SHA256 signature for Mekari API
// The signature is created from: date: {date}\n{request-line}
// Where request-line is: {method} {path} HTTP/1.1
func (h *HMACSignature) GenerateSignature(method, fullURL string, date time.Time) (authHeader string, dateHeader string, err error) {
	// Parse URL to get path and query
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Build request line: GET /v2/esign-hmac/v1/profile HTTP/1.1
	requestPath := parsedURL.Path
	if parsedURL.RawQuery != "" {
		requestPath = requestPath + "?" + parsedURL.RawQuery
	}
	requestLine := fmt.Sprintf("%s %s HTTP/1.1", method, requestPath)

	// Format date according to RFC1123 (HTTP Date format)
	dateHeader = date.UTC().Format(http.TimeFormat)

	// Create payload to sign: "date: {date}\n{request-line}"
	payload := fmt.Sprintf("date: %s\n%s", dateHeader, requestLine)

	// Generate HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(h.ClientSecret))
	mac.Write([]byte(payload))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Build Authorization header
	// Format: hmac username="client_id", algorithm="hmac-sha256", headers="date request-line", signature="signature"
	authHeader = fmt.Sprintf(`hmac username="%s", algorithm="hmac-sha256", headers="date request-line", signature="%s"`,
		h.ClientID, signature)

	// Debug logging for HMAC signature
	log.Printf("[HMAC-DEBUG] ==================== HMAC SIGNATURE DEBUG ====================")
	log.Printf("[HMAC-DEBUG] Local Time      : %s", date.Format("2006-01-02 15:04:05 MST"))
	log.Printf("[HMAC-DEBUG] UTC Time        : %s", date.UTC().Format("2006-01-02 15:04:05 MST"))
	log.Printf("[HMAC-DEBUG] Date Header     : %s", dateHeader)
	log.Printf("[HMAC-DEBUG] Method          : %s", method)
	log.Printf("[HMAC-DEBUG] Full URL        : %s", fullURL)
	log.Printf("[HMAC-DEBUG] Request Path    : %s", requestPath)
	log.Printf("[HMAC-DEBUG] Request Line    : %s", requestLine)
	log.Printf("[HMAC-DEBUG] Payload to Sign : %s", payload)
	log.Printf("[HMAC-DEBUG] Client ID       : %s", h.ClientID)
	log.Printf("[HMAC-DEBUG] Signature       : %s", signature)
	log.Printf("[HMAC-DEBUG] Auth Header     : %s", authHeader)
	log.Printf("[HMAC-DEBUG] ==============================================================")

	return authHeader, dateHeader, nil
}

// SignRequest signs an HTTP request with HMAC-SHA256 signature
func (h *HMACSignature) SignRequest(req *http.Request) error {
	now := time.Now()
	log.Printf("[HMAC-DEBUG] SignRequest called at: %s (Local) / %s (UTC)",
		now.Format("2006-01-02 15:04:05 MST"),
		now.UTC().Format("2006-01-02 15:04:05 UTC"))

	authHeader, dateHeader, err := h.GenerateSignature(req.Method, req.URL.String(), now)
	if err != nil {
		return err
	}

	req.Header.Set("Date", dateHeader)
	req.Header.Set("Authorization", authHeader)

	return nil
}
