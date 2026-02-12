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

// APILogSaver saves API logs to local database
type APILogSaver interface {
	Save(ctx context.Context, log *entity.APILog) error
}

// Client is the NAV API client for sending log entries
type Client struct {
	config     *config.Config
	httpClient *http.Client
	logSaver   APILogSaver
	logger     *zap.Logger
}

// NewClient creates a new NAV client
func NewClient(cfg *config.Config, logSaver APILogSaver, logger *zap.Logger) *Client {
	timeout := time.Duration(cfg.NAV.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logSaver: logSaver,
		logger:   logger,
	}
}

// UpdateLogEntry updates a log entry in NAV using PATCH
func (c *Client) UpdateLogEntry(ctx context.Context, entry *entity.NAVLogEntry) error {
	if !c.config.NAV.Enabled {
		c.logger.Debug("NAV integration disabled, skipping log entry update")
		return nil
	}
	startTime := time.Now()

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
		c.saveAPILog(ctx, http.MethodPatch, apiURL, reqBody, []byte("request creation failed: "+err.Error()), 0, time.Since(startTime), entry.InvoiceNo, entry.EntryNo)
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
		c.saveAPILog(ctx, http.MethodPatch, apiURL, reqBody, []byte("request failed: "+err.Error()), 0, time.Since(startTime), entry.InvoiceNo, entry.EntryNo)
		return fmt.Errorf("failed to update NAV log entry: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.saveAPILog(ctx, http.MethodPatch, apiURL, reqBody, []byte("read response failed: "+err.Error()), resp.StatusCode, time.Since(startTime), entry.InvoiceNo, entry.EntryNo)
		return fmt.Errorf("failed to read NAV response: %w", err)
	}

	c.saveAPILog(ctx, http.MethodPatch, apiURL, reqBody, respBody, resp.StatusCode, time.Since(startTime), entry.InvoiceNo, entry.EntryNo)

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

// SendAPILog sends an API log entry to NAV (MekariApiLogEntries)
func (c *Client) SendAPILog(ctx context.Context, log *entity.NAVAPILog) error {
	if !c.config.NAV.Enabled {
		return nil
	}
	startTime := time.Now()

	// Build URL
	apiURL := fmt.Sprintf("%s/ODataV4/Company('%s')/MekariApiLogEntries",
		c.config.NAV.BaseURL,
		url.PathEscape(c.config.NAV.Company),
	)

	// Marshal request body
	reqBody, err := json.Marshal(log)
	if err != nil {
		c.saveAPILog(ctx, http.MethodPost, apiURL, []byte{}, []byte("marshal failed: "+err.Error()), 0, time.Since(startTime), log.InvoiceNo, 0)
		return fmt.Errorf("failed to marshal NAV API log: %w", err)
	}

	// Create POST request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		c.saveAPILog(ctx, http.MethodPost, apiURL, reqBody, []byte("request creation failed: "+err.Error()), 0, time.Since(startTime), log.InvoiceNo, 0)
		return fmt.Errorf("failed to create NAV API log request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	auth := base64.StdEncoding.EncodeToString([]byte(c.config.NAV.Username + ":" + c.config.NAV.Password))
	req.Header.Set("Authorization", "Basic "+auth)

	startTime = time.Now()
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.saveAPILog(ctx, http.MethodPost, apiURL, reqBody, []byte("request failed: "+err.Error()), 0, time.Since(startTime), log.InvoiceNo, 0)
		return fmt.Errorf("failed to send NAV API log: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	c.saveAPILog(ctx, http.MethodPost, apiURL, reqBody, respBody, resp.StatusCode, time.Since(startTime), log.InvoiceNo, 0)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("NAV API log failed: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	c.logger.Debug("Successfully sent API log to NAV",
		zap.String("invoice_no", log.InvoiceNo),
		zap.String("status", log.StatusDescription),
	)

	return nil
}

// GetSetup fetches the Mekari setup configuration from NAV
func (c *Client) GetSetup(ctx context.Context) (*entity.NAVSetup, error) {
	if !c.config.NAV.Enabled {
		return nil, nil
	}
	startTime := time.Now()

	apiURL := fmt.Sprintf("%s/ODataV4/Company('%s')/Api_MekariSetup",
		c.config.NAV.BaseURL,
		url.PathEscape(c.config.NAV.Company),
	)

	c.logger.Info("Fetching Mekari setup from NAV", zap.String("url", apiURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		c.saveAPILog(ctx, http.MethodGet, apiURL, nil, []byte("request creation failed: "+err.Error()), 0, time.Since(startTime), "", 0)
		return nil, fmt.Errorf("failed to create NAV setup request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.config.NAV.Username + ":" + c.config.NAV.Password))
	req.Header.Set("Authorization", "Basic "+auth)

	startTime = time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.saveAPILog(ctx, http.MethodGet, apiURL, nil, []byte("request failed: "+err.Error()), 0, time.Since(startTime), "", 0)
		return nil, fmt.Errorf("failed to fetch NAV setup: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.saveAPILog(ctx, http.MethodGet, apiURL, nil, body, resp.StatusCode, time.Since(startTime), "", 0)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NAV setup failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var setupResp entity.NAVSetupResponse
	if err := json.Unmarshal(body, &setupResp); err != nil {
		c.saveAPILog(ctx, http.MethodGet, apiURL, nil, []byte("parse failed: "+err.Error()), resp.StatusCode, time.Since(startTime), "", 0)
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

func (c *Client) saveAPILog(ctx context.Context, method, endpoint string, requestBody []byte, responseBody []byte, statusCode int, duration time.Duration, invoiceNo string, entryNo int) {
	if c.logSaver == nil {
		return
	}

	reqBodyStr := ""
	if len(requestBody) > 0 {
		reqBodyStr = string(requestBody)
		if len(reqBodyStr) > 10000 {
			reqBodyStr = reqBodyStr[:10000] + "... [truncated]"
		}
	}

	respBodyStr := string(responseBody)
	if len(respBodyStr) > 10000 {
		respBodyStr = respBodyStr[:10000] + "... [truncated]"
	}

	apiLog := &entity.APILog{
		InvoiceNo:    invoiceNo,
		EntryNo:      entryNo,
		Endpoint:     endpoint,
		Method:       method,
		RequestBody:  reqBodyStr,
		ResponseBody: respBodyStr,
		StatusCode:   statusCode,
		Duration:     duration.Milliseconds(),
		CreatedAt:    time.Now(),
		From:         "SYSTEM",
		To:           "NAV",
	}

	go func() {
		_ = c.logSaver.Save(context.Background(), apiLog)
	}()
}
