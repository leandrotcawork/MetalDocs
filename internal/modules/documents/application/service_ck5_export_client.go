package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CK5ExportError is returned when ck5-export responds with a non-200 status.
type CK5ExportError struct {
	Status int
	Body   string
}

func (e *CK5ExportError) Error() string {
	return fmt.Sprintf("ck5-export returned %d: %s", e.Status, e.Body)
}

// CK5ExportClient is a thin HTTP client for the ck5-export Node.js service.
type CK5ExportClient struct {
	baseURL string
	http    *http.Client
}

// NewCK5ExportClient creates a client with a 30s timeout.
func NewCK5ExportClient(baseURL string) *CK5ExportClient {
	return &CK5ExportClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// newCK5ExportClientWithHTTP creates a client with a custom http.Client (for testing).
func newCK5ExportClientWithHTTP(baseURL string, client *http.Client) *CK5ExportClient {
	return &CK5ExportClient{baseURL: baseURL, http: client}
}

func (c *CK5ExportClient) post(ctx context.Context, path string, html string) ([]byte, error) {
	body, err := json.Marshal(map[string]string{"html": html})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, &CK5ExportError{Status: resp.StatusCode, Body: string(respBody)}
	}
	return respBody, nil
}

// RenderDocx calls POST /render/docx and returns the raw DOCX bytes.
func (c *CK5ExportClient) RenderDocx(ctx context.Context, html string) ([]byte, error) {
	return c.post(ctx, "/render/docx", html)
}

// RenderPDFHtml calls POST /render/pdf-html and returns the wrapped HTML string.
func (c *CK5ExportClient) RenderPDFHtml(ctx context.Context, html string) (string, error) {
	b, err := c.post(ctx, "/render/pdf-html", html)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
