package fanout

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type FanoutRequest struct {
	TenantID          string            `json:"tenant_id"`
	RevisionID        string            `json:"revision_id"`
	BodyDocxS3Key     string            `json:"body_docx_s3_key"`
	PlaceholderValues map[string]string `json:"placeholder_values"`
	Composition       json.RawMessage   `json:"composition_config"`
	ResolvedValues    map[string]any    `json:"resolved_values"`
}

type FanoutResponse struct {
	ContentHash    string   `json:"content_hash"`
	FinalDocxS3Key string   `json:"final_docx_s3_key"`
	UnreplacedVars []string `json:"unreplaced_vars"`
}

type Client struct {
	baseURL      string
	serviceToken string
	http         *http.Client
}

func NewClient(baseURL, serviceToken string, h *http.Client) *Client {
	if h == nil {
		h = http.DefaultClient
	}
	return &Client{baseURL: baseURL, serviceToken: serviceToken, http: h}
}

func (c *Client) Fanout(ctx context.Context, req FanoutRequest) (FanoutResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return FanoutResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/render/fanout", bytes.NewReader(body))
	if err != nil {
		return FanoutResponse{}, err
	}
	httpReq.Header.Set("content-type", "application/json")
	if c.serviceToken != "" {
		httpReq.Header.Set("X-Service-Token", c.serviceToken)
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return FanoutResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return FanoutResponse{}, fmt.Errorf("fanout status %d: %s", resp.StatusCode, string(errBody))
	}
	var out FanoutResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return FanoutResponse{}, err
	}
	return out, nil
}
