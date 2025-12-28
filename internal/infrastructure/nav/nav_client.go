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

// UpdateLogEntry updates a log entry in NAV using PATCH
func (c *Client) UpdateLogEntry(ctx context.Context, entry *entity.NAVLogEntry) error {
	if !c.config.NAV.Enabled {
		c.logger.Debug("NAV integration disabled, skipping log entry update")
		return nil
	}

	// Build URL with company and Entry_No parameter
	apiURL := fmt.Sprintf("%s/ODataV4/Company('%s')/Api_MekariInvoiceLogEntries(Entry_No=%d)",
		c.config.NAV.BaseURL,
		url.PathEscape(c.config.NAV.Company),
		entry.EntryNo,
	)

	// Marshal request body
	reqBody, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal NAV log entry: %w", err)
	}

	c.logger.Info("Updating log entry in NAV (PATCH)",
		zap.String("url", apiURL),
		zap.Int("entry_no", entry.EntryNo),
		zap.String("invoice_no", entry.InvoiceNo),
		zap.String("signing_status", entry.SigningStatus),
		zap.String("stamping_status", entry.StampingStatus),
		zap.String("request_body", string(reqBody)),
	)

	// Create PATCH request
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create NAV request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json;EEE754Compatible=true")
	req.Header.Set("If-Match", "*")
	auth := base64.StdEncoding.EncodeToString([]byte(c.config.NAV.Username + ":" + c.config.NAV.Password))
	req.Header.Set("Authorization", "Basic "+auth)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update NAV log entry: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read NAV response: %w", err)
	}

	c.logger.Info("NAV UpdateLogEntry response",
		zap.Int("status_code", resp.StatusCode),
		zap.String("body", string(respBody)),
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("NAV update failed: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	c.logger.Info("Successfully updated log entry in NAV",
		zap.Int("entry_no", entry.EntryNo),
		zap.String("invoice_no", entry.InvoiceNo),
	)

	return nil
}

// GetSetup fetches the Mekari setup configuration from NAV
func (c *Client) GetSetup(ctx context.Context) (*entity.NAVSetup, error) {
	if !c.config.NAV.Enabled {
		return nil, nil
	}

	apiURL := fmt.Sprintf("%s/ODataV4/Company('%s')/Api_MekariSetup",
		c.config.NAV.BaseURL,
		url.PathEscape(c.config.NAV.Company),
	)

	c.logger.Info("Fetching Mekari setup from NAV", zap.String("url", apiURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NAV setup request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.config.NAV.Username + ":" + c.config.NAV.Password))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NAV setup: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NAV setup failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var setupResp entity.NAVSetupResponse
	if err := json.NewDecoder(resp.Body).Decode(&setupResp); err != nil {
		return nil, fmt.Errorf("failed to parse NAV setup: %w", err)
	}

	if len(setupResp.Value) == 0 {
		return nil, fmt.Errorf("no setup found in NAV")
	}

	c.logger.Info("Successfully fetched NAV setup",
		zap.String("file_location_in", setupResp.Value[0].FileLocationIn),
		zap.String("file_location_process", setupResp.Value[0].FileLocationProcess),
		zap.String("file_location_out", setupResp.Value[0].FileLocationOut),
	)

	return &setupResp.Value[0], nil
}
