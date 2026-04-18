package servicebus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ValidateTemplate calls docgen-v2 /validate/template.
// Returns (true, raw, nil) on HTTP 200, (false, raw, nil) on HTTP 422,
// or (false, raw, err) for any other status or transport error.
func (c *DocgenV2Client) ValidateTemplate(ctx context.Context, docxKey, schemaKey string) (bool, []byte, error) {
	body, _ := json.Marshal(map[string]string{"docx_key": docxKey, "schema_key": schemaKey})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/validate/template", bytes.NewReader(body))
	if err != nil {
		return false, nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-service-token", c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return false, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		return true, raw, nil
	}
	if resp.StatusCode == http.StatusUnprocessableEntity {
		return false, raw, nil
	}
	return false, raw, fmt.Errorf("docgen-v2: unexpected status %d", resp.StatusCode)
}
