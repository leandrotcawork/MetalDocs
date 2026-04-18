// Package servicebus holds internal service-to-service clients.
// docgen_v2_client is the minimal W1 client (health only).
package servicebus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DocgenV2Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewDocgenV2Client(baseURL, token string, timeout time.Duration) *DocgenV2Client {
	return &DocgenV2Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: timeout},
	}
}

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Health pings /health (no auth). Returns remote version string on 200.
func (c *DocgenV2Client) Health(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("docgen-v2 health: unexpected status %d", resp.StatusCode)
	}
	var out healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("docgen-v2 health: decode: %w", err)
	}
	if out.Status != "ok" {
		return "", fmt.Errorf("docgen-v2 health: status=%q", out.Status)
	}
	return out.Version, nil
}
