package docgen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"metaldocs/internal/platform/config"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

var ErrUnavailable = errors.New("docgen unavailable")

func NewClient(cfg config.DocgenConfig) *Client {
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

func (c *Client) Generate(ctx context.Context, payload RenderPayload, traceID string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("docgen client not configured")
	}
	if traceID == "" {
		traceID = "trace-local"
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal docgen payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-Id", traceID)

	log.Printf("docgen generate trace_id=%s document_type=%s document_code=%s", traceID, payload.DocumentType, payload.DocumentCode)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: docgen request: %v", ErrUnavailable, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= http.StatusInternalServerError {
			return nil, fmt.Errorf("%w: docgen generate failed status=%d body=%s", ErrUnavailable, resp.StatusCode, strings.TrimSpace(string(raw)))
		}
		return nil, fmt.Errorf("docgen generate failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	rendered, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read docgen response: %w", err)
	}
	return rendered, nil
}

func (c *Client) GenerateBrowser(ctx context.Context, payload BrowserRenderPayload, traceID string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("docgen client not configured")
	}
	if traceID == "" {
		traceID = "trace-local"
	}

	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		return nil, fmt.Errorf("marshal browser docgen payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/generate-browser", bytes.NewReader(body.Bytes()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-Id", traceID)

	log.Printf("docgen generate-browser trace_id=%s document_code=%s", traceID, payload.DocumentCode)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: docgen request: %v", ErrUnavailable, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= http.StatusInternalServerError {
			return nil, fmt.Errorf("%w: docgen generate-browser failed status=%d body=%s", ErrUnavailable, resp.StatusCode, strings.TrimSpace(string(raw)))
		}
		return nil, fmt.Errorf("docgen generate-browser failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	rendered, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read docgen response: %w", err)
	}
	return rendered, nil
}
