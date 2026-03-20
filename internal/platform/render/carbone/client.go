package carbone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"metaldocs/internal/platform/config"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
	Error   string          `json:"error"`
}

type registerTemplateData struct {
	TemplateID string `json:"templateId"`
}

type renderTemplateData struct {
	RenderID string `json:"renderId"`
}

func NewClient(cfg config.CarboneConfig) *Client {
	if !cfg.Enabled {
		return nil
	}
	base := strings.TrimRight(cfg.APIURL, "/")
	if base == "" {
		return nil
	}
	timeout := time.Duration(cfg.RequestTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL: base,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Ping(ctx context.Context, traceID string) error {
	if c == nil {
		return fmt.Errorf("carbone client not configured")
	}
	if traceID == "" {
		traceID = "trace-local"
	}
	endpoints := []string{"/health", "/"}
	var lastErr error
	for _, endpoint := range endpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Trace-Id", traceID)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		if resp.StatusCode == http.StatusNotFound {
			lastErr = fmt.Errorf("carbone health endpoint not found")
			continue
		}
		lastErr = fmt.Errorf("carbone health returned status %d", resp.StatusCode)
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("carbone health check failed")
}

func (c *Client) RegisterTemplate(ctx context.Context, traceID, filePath string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("carbone client not configured")
	}
	if traceID == "" {
		traceID = "trace-local"
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open template: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("template", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("create multipart: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("copy template: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/template", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Trace-Id", traceID)

	log.Printf("carbone register template trace_id=%s file=%s", traceID, filepath.Base(filePath))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("carbone register request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("carbone register failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var envelope apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", fmt.Errorf("decode carbone register response: %w", err)
	}
	if !envelope.Success {
		return "", fmt.Errorf("carbone register failed: %s", errorMessage(envelope))
	}

	var data registerTemplateData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return "", fmt.Errorf("decode carbone register data: %w", err)
	}
	if strings.TrimSpace(data.TemplateID) == "" {
		return "", fmt.Errorf("carbone register missing templateId")
	}
	return data.TemplateID, nil
}

func (c *Client) RenderTemplate(ctx context.Context, traceID, templateID string, data any, convertTo string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("carbone client not configured")
	}
	if traceID == "" {
		traceID = "trace-local"
	}
	payload := map[string]any{
		"data": data,
	}
	if convertTo != "" {
		payload["convertTo"] = convertTo
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal render payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/render/"+templateID, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-Id", traceID)

	log.Printf("carbone render template trace_id=%s template_id=%s", traceID, templateID)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("carbone render request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("carbone render failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var envelope apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", fmt.Errorf("decode carbone render response: %w", err)
	}
	if !envelope.Success {
		return "", fmt.Errorf("carbone render failed: %s", errorMessage(envelope))
	}

	var dataResp renderTemplateData
	if err := json.Unmarshal(envelope.Data, &dataResp); err != nil {
		return "", fmt.Errorf("decode carbone render data: %w", err)
	}
	if strings.TrimSpace(dataResp.RenderID) == "" {
		return "", fmt.Errorf("carbone render missing renderId")
	}
	return dataResp.RenderID, nil
}

func (c *Client) DownloadRender(ctx context.Context, traceID, renderID string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("carbone client not configured")
	}
	if traceID == "" {
		traceID = "trace-local"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/render/"+renderID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Trace-Id", traceID)

	log.Printf("carbone download render trace_id=%s render_id=%s", traceID, renderID)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("carbone download request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("carbone download failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read carbone download: %w", err)
	}
	return payload, nil
}

func errorMessage(resp apiResponse) string {
	msg := strings.TrimSpace(resp.Error)
	if msg == "" {
		msg = strings.TrimSpace(resp.Message)
	}
	if msg == "" {
		msg = "unknown carbone error"
	}
	return msg
}
