package servicebus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ConvertPDFRequest struct {
	DocxKey    string         `json:"docx_key"`
	OutputKey  string         `json:"output_key"`
	RenderOpts *PDFRenderOpts `json:"render_opts,omitempty"`
}

type PDFRenderOpts struct {
	PaperSize string `json:"paper_size,omitempty"`
	Landscape bool   `json:"landscape,omitempty"`
}

type ConvertPDFResult struct {
	OutputKey       string `json:"output_key"`
	ContentHash     string `json:"content_hash"`
	SizeBytes       int64  `json:"size_bytes"`
	DocgenV2Version string `json:"docgen_v2_version"`
}

func (c *DocgenV2Client) ConvertPDF(ctx context.Context, req ConvertPDFRequest) (ConvertPDFResult, error) {
	var zero ConvertPDFResult

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	body, err := json.Marshal(req)
	if err != nil {
		return zero, fmt.Errorf("docgen-v2 convert pdf: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/convert/pdf", bytes.NewReader(body))
	if err != nil {
		return zero, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Service-Token", c.token)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return zero, fmt.Errorf("docgen-v2 convert pdf: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("docgen-v2 convert pdf: unexpected status %d body=%s", resp.StatusCode, string(respBody))
	}

	var out ConvertPDFResult
	if err := json.Unmarshal(respBody, &out); err != nil {
		return zero, fmt.Errorf("docgen-v2 convert pdf: decode: %w", err)
	}
	return out, nil
}
