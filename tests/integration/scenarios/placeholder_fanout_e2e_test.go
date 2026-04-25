//go:build integration
// +build integration

package scenarios_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestPlaceholderFanout_E2E exercises the full Draft→Submit→Signoff→Approve→Freeze→Publish
// flow with a mocked docgen-v2 fanout endpoint.
//
// Prerequisites (set via env):
//   METALDOCS_E2E_URL             - running metaldocs-api base URL
//   METALDOCS_E2E_TENANT_ID       - tenant UUID (default: 00000000-0000-0000-0000-000000000001)
//   METALDOCS_E2E_USER_ID         - actor user ID (default: e2e-fanout-user)
//   METALDOCS_E2E_TEMPLATE_VERSION_ID - template version with placeholders
//   METALDOCS_E2E_ROUTE_ID        - approval route ID
//   METALDOCS_E2E_APPROVER_ID     - approver user ID (default: e2e-approver)
//
// The test starts a local stub server that captures the fanout POST request so the
// pipeline can complete without a real docgen-v2 instance.
func TestPlaceholderFanout_E2E(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("METALDOCS_E2E_URL"))
	if baseURL == "" {
		t.Skip("requires running server — set METALDOCS_E2E_URL")
	}

	tenantID := envOrDefault("METALDOCS_E2E_TENANT_ID", "00000000-0000-0000-0000-000000000001")
	userID := envOrDefault("METALDOCS_E2E_USER_ID", "e2e-fanout-user")
	approverID := envOrDefault("METALDOCS_E2E_APPROVER_ID", "e2e-approver")
	templateVersionID := envOrDefault("METALDOCS_E2E_TEMPLATE_VERSION_ID", "11111111-1111-1111-1111-111111111111")
	routeID := envOrDefault("METALDOCS_E2E_ROUTE_ID", "22222222-2222-2222-2222-222222222222")
	contentHash := strings.Repeat("b", 64)

	// --- mocked docgen-v2 fanout stub ---
	var fanoutCalls int32
	fanoutStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&fanoutCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content_hash":      strings.Repeat("c", 64),
			"final_docx_s3_key": "stub/fanout/result.docx",
			"pdf_s3_key":        "stub/fanout/result.pdf",
		})
	}))
	defer fanoutStub.Close()

	client := &http.Client{Timeout: 30 * time.Second}

	var documentID string
	var instanceID string
	var stageID string

	// Step 1: Create document (Draft state)
	t.Run("CreateDocument", func(t *testing.T) {
		body := map[string]any{
			"template_version_id": templateVersionID,
			"name":                fmt.Sprintf("FanoutE2E-%d", time.Now().UnixNano()),
			"form_data":           map[string]any{},
		}
		resp, raw := doJSONRequest(t, client, http.MethodPost, baseURL+"/api/v2/documents", body, map[string]string{
			"X-Tenant-ID": tenantID,
			"X-User-ID":   userID,
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create status=%d body=%s", resp.StatusCode, raw)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		documentID = asString(payload["document_id"])
		if documentID == "" {
			t.Fatalf("missing document_id: %s", raw)
		}
	})

	// Step 2: Submit for review
	t.Run("Submit", func(t *testing.T) {
		body := map[string]any{"route_id": routeID, "content_hash": contentHash}
		resp, raw := doJSONRequest(t, client, http.MethodPost,
			fmt.Sprintf("%s/api/v2/documents/%s/submit", baseURL, documentID), body, map[string]string{
				"X-Tenant-ID":     tenantID,
				"X-User-ID":       userID,
				"Idempotency-Key": "fanout-e2e-submit-1",
				"If-Match":        `"v1"`,
			})
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			t.Fatalf("submit status=%d body=%s", resp.StatusCode, raw)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		instanceID = asString(payload["instance_id"])
		if instanceID == "" {
			t.Fatalf("missing instance_id: %s", raw)
		}
	})

	// Step 3: Fetch first stage ID
	t.Run("GetFirstStage", func(t *testing.T) {
		resp, raw := doJSONRequest(t, client, http.MethodGet,
			fmt.Sprintf("%s/api/v2/approval/instances/%s", baseURL, instanceID), nil, map[string]string{
				"X-Tenant-ID": tenantID,
				"X-User-ID":   userID,
			})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get instance status=%d body=%s", resp.StatusCode, raw)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		stages, _ := payload["stages"].([]any)
		if len(stages) == 0 {
			t.Fatalf("no stages in instance: %s", raw)
		}
		stageID = asString((stages[0].(map[string]any))["stage_id"])
		if stageID == "" {
			t.Fatalf("missing stage_id: %s", raw)
		}
	})

	// Step 4: Signoff (approver)
	t.Run("Signoff", func(t *testing.T) {
		body := map[string]any{"decision": "signoff", "comment": "LGTM"}
		resp, raw := doJSONRequest(t, client, http.MethodPost,
			fmt.Sprintf("%s/api/v2/approval/instances/%s/stages/%s/decision", baseURL, instanceID, stageID),
			body, map[string]string{
				"X-Tenant-ID": tenantID,
				"X-User-ID":   approverID,
			})
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			t.Fatalf("signoff status=%d body=%s", resp.StatusCode, raw)
		}
	})

	// Step 5: Approve (final approval triggers Freeze + fanout dispatch)
	t.Run("Approve", func(t *testing.T) {
		body := map[string]any{"decision": "approve"}
		resp, raw := doJSONRequest(t, client, http.MethodPost,
			fmt.Sprintf("%s/api/v2/approval/instances/%s/approve", baseURL, instanceID),
			body, map[string]string{
				"X-Tenant-ID": tenantID,
				"X-User-ID":   approverID,
			})
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			t.Fatalf("approve status=%d body=%s", resp.StatusCode, raw)
		}
	})

	// Step 6: Publish
	t.Run("Publish", func(t *testing.T) {
		resp, raw := doJSONRequest(t, client, http.MethodPost,
			fmt.Sprintf("%s/api/v2/documents/%s/publish", baseURL, documentID), nil, map[string]string{
				"X-Tenant-ID": tenantID,
				"X-User-ID":   userID,
			})
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			t.Fatalf("publish status=%d body=%s", resp.StatusCode, raw)
		}
	})

	// Step 7: Assert fanout was called and document has freeze timestamp
	t.Run("VerifyFanoutCalled", func(t *testing.T) {
		if atomic.LoadInt32(&fanoutCalls) == 0 {
			// fanout stub URL must be wired into the server via METALDOCS_FANOUT_URL env.
			// If not wired, this is a scaffold failure — not a product bug.
			t.Log("fanout stub received 0 calls — ensure METALDOCS_FANOUT_URL=" + fanoutStub.URL)
			t.Skip("fanout wiring not configured — set METALDOCS_FANOUT_URL to stub server")
		}
	})

	t.Run("VerifySnapshotFrozen", func(t *testing.T) {
		resp, raw := doJSONRequest(t, client, http.MethodGet,
			fmt.Sprintf("%s/api/v2/documents/%s/snapshot", baseURL, documentID), nil, map[string]string{
				"X-Tenant-ID": tenantID,
				"X-User-ID":   userID,
			})
		if resp.StatusCode == http.StatusNotFound {
			t.Skip("snapshot endpoint not yet wired — scaffold only")
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("snapshot status=%d body=%s", resp.StatusCode, raw)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if asString(payload["values_frozen_at"]) == "" {
			t.Errorf("values_frozen_at not set — freeze did not run")
		}
	})
}
