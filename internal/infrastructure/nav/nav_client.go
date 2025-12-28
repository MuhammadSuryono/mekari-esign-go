package nav

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"mekari-esign/internal/config"
	"mekari-esign/internal/domain/entity"
)

// Client is the NAV API client for sending log entries
type Client struct {
	config     *config.Config
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new NAV client
func NewClient(cfg *config.Config, logger *zap.Logger) *Client {
	timeout := time.Duration(cfg.NAV.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// SendLogEntry sends a log entry to NAV
func (c *Client) SendLogEntry(ctx context.Context, entry *entity.NAVLogEntry) error {
	if !c.config.NAV.Enabled {
		c.logger.Debug("NAV integration disabled, skipping log entry")
		return nil
	}

	// Build URL with company parameter
	apiURL := fmt.Sprintf("%s/ODataV4/Company('%s')/Api_MekariInvoiceLogEntries",
		c.config.NAV.BaseURL,
		url.PathEscape(c.config.NAV.Company),
	)

	// Marshal request body
	reqBody, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal NAV log entry: %w", err)
	}

	c.logger.Info("Sending log entry to NAV",
		zap.String("url", apiURL),
		zap.String("document_id", entry.ID),
		zap.String("invoice_number", entry.InvoiceNumber),
		zap.String("signing_status", entry.SigningStatus),
		zap.String("stamping_status", entry.StampingStatus),
	)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create NAV request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	auth := base64.StdEncoding.EncodeToString([]byte(c.config.NAV.Username + ":" + c.config.NAV.Password))
	req.Header.Set("Authorization", "Basic "+auth)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send NAV log entry: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read NAV response: %w", err)
	}

	c.logger.Info("NAV response",
		zap.Int("status_code", resp.StatusCode),
		zap.String("body", string(respBody)),
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("NAV request failed: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	c.logger.Info("Successfully sent log entry to NAV",
		zap.String("document_id", entry.ID),
		zap.String("invoice_number", entry.InvoiceNumber),
	)

	return nil
}
