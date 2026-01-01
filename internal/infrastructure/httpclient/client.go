package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"mekari-esign/internal/config"
	"mekari-esign/internal/domain/entity"
	"mekari-esign/internal/infrastructure/oauth2"
)

const (
	maxBodyLogLength = 500 // Maximum characters to log for body
)

// ErrUnauthorized is returned when token is invalid and refresh failed
var ErrUnauthorized = errors.New("unauthorized: token refresh failed, re-authorization required")

// RequestContext contains context for authenticated requests
type RequestContext struct {
	Email string // Email for token lookup (only used for OAuth2)
}

type HTTPClient interface {
	// Get performs GET request with configured auth method
	Get(ctx context.Context, reqCtx *RequestContext, path string, result interface{}) error
	// Post performs POST request with configured auth method
	Post(ctx context.Context, reqCtx *RequestContext, path string, body interface{}, result interface{}) error
	// PostMultipart performs multipart POST request with configured auth method
	PostMultipart(ctx context.Context, reqCtx *RequestContext, path string, fields map[string]string, files map[string]FileUpload, result interface{}) error
	// Put performs PUT request with configured auth method
	Put(ctx context.Context, reqCtx *RequestContext, path string, body interface{}, result interface{}) error
	// Delete performs DELETE request with configured auth method
	Delete(ctx context.Context, reqCtx *RequestContext, path string, result interface{}) error
}

// FileUpload represents a file to be uploaded
type FileUpload struct {
	Filename string
	Content  []byte
}

// APILogSaver interface for saving API logs
type APILogSaver interface {
	Save(ctx context.Context, log *entity.APILog) error
}

// NAVAPILogSender interface for sending API logs to NAV
type NAVAPILogSender interface {
	SendAPILog(ctx context.Context, log *entity.NAVAPILog) error
}

type httpClient struct {
	client          *http.Client
	config          *config.Config
	baseURL         string
	tokenService    oauth2.TokenService
	hmacSignature   *HMACSignature
	apiLogSaver     APILogSaver
	navAPILogSender NAVAPILogSender
	logger          *zap.Logger
}

func NewHTTPClient(cfg *config.Config, tokenService oauth2.TokenService, apiLogSaver APILogSaver, navAPILogSender NAVAPILogSender, logger *zap.Logger) HTTPClient {
	c := &httpClient{
		client: &http.Client{
			Timeout: cfg.Mekari.Timeout,
		},
		config:          cfg,
		baseURL:         cfg.Mekari.BaseURL,
		tokenService:    tokenService,
		apiLogSaver:     apiLogSaver,
		navAPILogSender: navAPILogSender,
		logger:          logger,
	}

	// Initialize HMAC signature if using HMAC auth
	if cfg.Mekari.IsHMAC() {
		c.hmacSignature = NewHMACSignature(cfg.Mekari.HMAC.ClientID, cfg.Mekari.HMAC.ClientSecret)
		logger.Info("HTTP Client initialized with HMAC authentication",
			zap.String("client_id", cfg.Mekari.HMAC.ClientID),
		)
	} else {
		logger.Info("HTTP Client initialized with OAuth2 authentication")
	}

	return c
}

// truncateString truncates a string if it exceeds maxLength
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + fmt.Sprintf("... [truncated, total %d chars]", len(s))
}

// truncateBase64InJSON truncates base64-like values in JSON string
func truncateBase64InJSON(jsonStr string, maxLength int) string {
	// Pattern to match base64-like content (long strings of alphanumeric + /+=)
	base64Pattern := regexp.MustCompile(`"([A-Za-z0-9+/=]{100,})"`)

	return base64Pattern.ReplaceAllStringFunc(jsonStr, func(match string) string {
		// Remove quotes for processing
		content := match[1 : len(match)-1]
		if len(content) > maxLength {
			return fmt.Sprintf(`"%s... [base64 truncated, total %d chars]"`, content[:maxLength], len(content))
		}
		return match
	})
}

// formatHeadersForLog formats HTTP headers for logging in "Header Key=Value" format
func formatHeadersForLog(headers http.Header) string {
	var sb strings.Builder
	for key, values := range headers {
		for _, value := range values {
			// Truncate very long header values
			if len(value) > 100 {
				value = value[:100] + "..."
			}
			sb.WriteString(fmt.Sprintf("Header %s=%s\n", key, value))
		}
	}
	return sb.String()
}

// logRequest logs the HTTP request details
func (c *httpClient) logRequest(method, url string, headers http.Header, body []byte) {
	var logBuilder strings.Builder

	logBuilder.WriteString("\n>>> [WEBCLIENT-REQ]\n")
	logBuilder.WriteString(fmt.Sprintf("Method: %s\n", method))
	logBuilder.WriteString(fmt.Sprintf("URL: %s\n", url))
	logBuilder.WriteString(fmt.Sprintf("Auth-Type: %s\n", c.config.Mekari.AuthType))
	logBuilder.WriteString(formatHeadersForLog(headers))

	if len(body) > 0 {
		bodyStr := truncateBase64InJSON(string(body), 100)
		bodyStr = truncateString(bodyStr, maxBodyLogLength)
		logBuilder.WriteString(fmt.Sprintf("REQUEST BODY: %s\n", bodyStr))
	}

	c.logger.Info(logBuilder.String())
}

// logResponse logs the HTTP response details
func (c *httpClient) logResponse(statusCode int, statusText string, duration time.Duration, headers http.Header, body []byte) {
	var logBuilder strings.Builder

	logBuilder.WriteString("\n>>> [WEBCLIENT-RESPONSE]\n")
	logBuilder.WriteString(fmt.Sprintf("Status: %d %s\n", statusCode, statusText))
	logBuilder.WriteString(fmt.Sprintf("Duration: %s\n", duration))
	logBuilder.WriteString(formatHeadersForLog(headers))

	bodyStr := truncateString(string(body), maxBodyLogLength)
	logBuilder.WriteString(fmt.Sprintf("Body: %s\n", bodyStr))

	c.logger.Info(logBuilder.String())
}

// saveAPILog saves the API request/response log to database
func (c *httpClient) saveAPILog(ctx context.Context, method, endpoint string, requestBody []byte, responseBody []byte, statusCode int, duration time.Duration, email string) {
	if c.apiLogSaver == nil {
		return
	}

	// Truncate base64 in request body
	reqBodyStr := ""
	if len(requestBody) > 0 {
		reqBodyStr = truncateBase64InJSON(string(requestBody), 100)
		// Limit total size
		if len(reqBodyStr) > 10000 {
			reqBodyStr = reqBodyStr[:10000] + "... [truncated]"
		}
	}

	// Truncate response body if too long
	respBodyStr := string(responseBody)
	if len(respBodyStr) > 10000 {
		respBodyStr = respBodyStr[:10000] + "... [truncated]"
	}

	apiLog := &entity.APILog{
		Endpoint:     endpoint,
		Method:       method,
		RequestBody:  reqBodyStr,
		ResponseBody: respBodyStr,
		StatusCode:   statusCode,
		Duration:     duration.Milliseconds(),
		Email:        email,
		CreatedAt:    time.Now(),
	}

	// Save asynchronously to not block the request
	go func() {
		if err := c.apiLogSaver.Save(context.Background(), apiLog); err != nil {
			c.logger.Warn("Failed to save API log to database",
				zap.String("endpoint", endpoint),
				zap.Error(err),
			)
		}
	}()

	// Also send to NAV if enabled
	if c.navAPILogSender != nil {
		go func() {
			// Determine status description
			statusDesc := "SUCCESS"
			if statusCode < 200 || statusCode >= 300 {
				statusDesc = "ERROR"
			}

			// Build body summary (combine request and response info)
			bodySummary := fmt.Sprintf(`{"method":"%s","status_code":%d,"duration_ms":%d, "requester": %s}`,
				method, statusCode, duration.Milliseconds(), email)

			navLog := &entity.NAVAPILog{
				StatusDescription: statusDesc,
				DateTime:          time.Now().UTC().Format(time.RFC3339),
				InvoiceNo:         endpoint, // Using endpoint as an identifier
				Body:              bodySummary,
			}

			if err := c.navAPILogSender.SendAPILog(context.Background(), navLog); err != nil {
				c.logger.Warn("Failed to send API log to NAV",
					zap.String("endpoint", endpoint),
					zap.Error(err),
				)
			}
		}()
	}
}

// setAuthHeaders sets the appropriate authorization headers based on config
func (c *httpClient) setAuthHeaders(ctx context.Context, req *http.Request, reqCtx *RequestContext) error {
	if c.config.Mekari.IsHMAC() {
		// Use HMAC authentication
		return c.hmacSignature.SignRequest(req)
	}

	// Use OAuth2 authentication
	accessToken, err := c.tokenService.GetAccessToken(ctx, reqCtx.Email)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	return nil
}

func (c *httpClient) doRequest(ctx context.Context, reqCtx *RequestContext, method, path string, body interface{}, result interface{}, isRetry bool) error {
	fullURL := c.baseURL + path

	var bodyReader io.Reader
	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set auth headers based on config
	if err := c.setAuthHeaders(ctx, req, reqCtx); err != nil {
		return err
	}

	// Log request details
	c.logRequest(method, fullURL, req.Header, jsonBody)

	startTime := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response details
	c.logResponse(resp.StatusCode, resp.Status, duration, resp.Header, respBody)

	// Save API log to database
	email := ""
	if reqCtx != nil {
		email = reqCtx.Email
	}
	c.saveAPILog(ctx, method, fullURL, jsonBody, respBody, resp.StatusCode, duration, email)

	// Handle 401 Unauthorized - try to refresh token and retry (OAuth2 only)
	if resp.StatusCode == http.StatusUnauthorized && !isRetry && c.config.Mekari.IsOAuth2() {
		c.logger.Info("Received 401 Unauthorized, attempting to refresh token",
			zap.String("email", reqCtx.Email),
		)

		// Refresh token
		_, err := c.tokenService.RefreshToken(ctx, reqCtx.Email)
		if err != nil {
			c.logger.Error("Failed to refresh token", zap.Error(err))
			return ErrUnauthorized
		}

		// Retry request with new token
		c.logger.Info("Token refreshed, retrying request",
			zap.String("email", reqCtx.Email),
		)
		return c.doRequest(ctx, reqCtx, method, path, body, result, true)
	}

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	// Parse response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

func (c *httpClient) Get(ctx context.Context, reqCtx *RequestContext, path string, result interface{}) error {
	return c.doRequest(ctx, reqCtx, http.MethodGet, path, nil, result, false)
}

func (c *httpClient) Post(ctx context.Context, reqCtx *RequestContext, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, reqCtx, http.MethodPost, path, body, result, false)
}

func (c *httpClient) Put(ctx context.Context, reqCtx *RequestContext, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, reqCtx, http.MethodPut, path, body, result, false)
}

func (c *httpClient) Delete(ctx context.Context, reqCtx *RequestContext, path string, result interface{}) error {
	return c.doRequest(ctx, reqCtx, http.MethodDelete, path, nil, result, false)
}

// PostMultipart sends a multipart/form-data POST request
func (c *httpClient) PostMultipart(ctx context.Context, reqCtx *RequestContext, path string, fields map[string]string, files map[string]FileUpload, result interface{}) error {
	return c.doMultipartRequest(ctx, reqCtx, path, fields, files, result, false)
}

func (c *httpClient) doMultipartRequest(ctx context.Context, reqCtx *RequestContext, path string, fields map[string]string, files map[string]FileUpload, result interface{}, isRetry bool) error {
	fullURL := c.baseURL + path

	// Create multipart writer
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form fields
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return fmt.Errorf("failed to write field %s: %w", key, err)
		}
	}

	// Add files
	for fieldName, file := range files {
		part, err := writer.CreateFormFile(fieldName, file.Filename)
		if err != nil {
			return fmt.Errorf("failed to create form file %s: %w", fieldName, err)
		}
		if _, err := part.Write(file.Content); err != nil {
			return fmt.Errorf("failed to write file content %s: %w", fieldName, err)
		}
	}

	// Close writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set content type with boundary
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	// Set auth headers based on config
	if err := c.setAuthHeaders(ctx, req, reqCtx); err != nil {
		return err
	}

	// Build multipart body summary for logging
	var bodySummary strings.Builder
	bodySummary.WriteString("{fields: [")
	fieldKeys := make([]string, 0, len(fields))
	for k := range fields {
		fieldKeys = append(fieldKeys, k)
	}
	bodySummary.WriteString(strings.Join(fieldKeys, ", "))
	bodySummary.WriteString("], files: [")
	fileKeys := make([]string, 0, len(files))
	for k, f := range files {
		fileKeys = append(fileKeys, fmt.Sprintf("%s(%s, %d bytes)", k, f.Filename, len(f.Content)))
	}
	bodySummary.WriteString(strings.Join(fileKeys, ", "))
	bodySummary.WriteString("]}")

	// Log request details
	c.logRequest(http.MethodPost, fullURL, req.Header, []byte(bodySummary.String()))

	startTime := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response details
	c.logResponse(resp.StatusCode, resp.Status, duration, resp.Header, respBody)

	// Save API log to database (for multipart, log the body summary)
	multipartEmail := ""
	if reqCtx != nil {
		multipartEmail = reqCtx.Email
	}
	c.saveAPILog(ctx, http.MethodPost, fullURL, []byte(bodySummary.String()), respBody, resp.StatusCode, duration, multipartEmail)

	// Handle 401 Unauthorized - try to refresh token and retry (OAuth2 only)
	if resp.StatusCode == http.StatusUnauthorized && !isRetry && c.config.Mekari.IsOAuth2() {
		c.logger.Info("Received 401 Unauthorized, attempting to refresh token",
			zap.String("email", reqCtx.Email),
		)

		// Refresh token
		_, err := c.tokenService.RefreshToken(ctx, reqCtx.Email)
		if err != nil {
			c.logger.Error("Failed to refresh token", zap.Error(err))
			return ErrUnauthorized
		}

		// Retry request with new token
		c.logger.Info("Token refreshed, retrying request",
			zap.String("email", reqCtx.Email),
		)
		return c.doMultipartRequest(ctx, reqCtx, path, fields, files, result, true)
	}

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	// Parse response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}
