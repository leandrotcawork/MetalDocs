package gotenberg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ConvertHTMLToPDF sends an HTML document plus an auxiliary stylesheet to
// Gotenberg's Chromium route and returns the rendered PDF bytes.
func (c *Client) ConvertHTMLToPDF(ctx context.Context, htmlBytes []byte, cssBytes []byte) ([]byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	htmlPart, err := writer.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, fmt.Errorf("gotenberg: create html form file: %w", err)
	}
	if _, err := htmlPart.Write(htmlBytes); err != nil {
		return nil, fmt.Errorf("gotenberg: write html content: %w", err)
	}

	if len(cssBytes) > 0 {
		cssPart, err := writer.CreateFormFile("files", "style.css")
		if err != nil {
			return nil, fmt.Errorf("gotenberg: create css form file: %w", err)
		}
		if _, err := cssPart.Write(cssBytes); err != nil {
			return nil, fmt.Errorf("gotenberg: write css content: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("gotenberg: close multipart: %w", err)
	}

	url := c.baseURL + "/forms/chromium/convert/html"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return nil, fmt.Errorf("gotenberg: create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gotenberg: html request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gotenberg: html conversion returned status %d: %s", resp.StatusCode, string(payload))
	}

	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gotenberg: read pdf response: %w", err)
	}
	return pdfBytes, nil
}

func (c *Client) ConvertDocxToPDF(ctx context.Context, docxContent []byte) ([]byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files", "document.docx")
	if err != nil {
		return nil, fmt.Errorf("gotenberg: create form file: %w", err)
	}
	if _, err := part.Write(docxContent); err != nil {
		return nil, fmt.Errorf("gotenberg: write docx content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("gotenberg: close multipart: %w", err)
	}

	url := c.baseURL + "/forms/libreoffice/convert"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return nil, fmt.Errorf("gotenberg: create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gotenberg: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gotenberg: status %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}
