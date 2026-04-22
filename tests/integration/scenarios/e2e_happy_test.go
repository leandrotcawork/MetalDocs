//go:build integration
// +build integration

package scenarios_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestE2E_HappyPath_HTTP(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("METALDOCS_E2E_URL"))
	if baseURL == "" {
		t.Skip("requires running server - set METALDOCS_E2E_URL")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	tenantID := envOrDefault("METALDOCS_E2E_TENANT_ID", "00000000-0000-0000-0000-000000000001")
	userID := envOrDefault("METALDOCS_E2E_USER_ID", "e2e-user")
	userRoles := envOrDefault("METALDOCS_E2E_USER_ROLES", "admin,document_filler,reviewer,approver")
	templateVersionID := envOrDefault("METALDOCS_E2E_TEMPLATE_VERSION_ID", "11111111-1111-1111-1111-111111111111")
	routeID := envOrDefault("METALDOCS_E2E_ROUTE_ID", "22222222-2222-2222-2222-222222222222")
	contentHash := strings.Repeat("a", 64)

	var documentID string
	var instanceID string
	var submitETag string
	var stageIDs []string

	// 1) POST /api/v2/documents -> create document
	t.Run("CreateDocument", func(t *testing.T) {
		body := map[string]any{
			"template_version_id": templateVersionID,
			"name":                fmt.Sprintf("E2E Happy %d", time.Now().UnixNano()),
			"form_data":           map[string]any{},
		}

		resp, raw := doJSONRequest(t, client, http.MethodPost, baseURL+"/api/v2/documents", body, map[string]string{
			"X-Tenant-ID":  tenantID,
			"X-User-ID":    userID,
			"X-User-Roles": userRoles,
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create document status=%d body=%s", resp.StatusCode, raw)
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("decode create response: %v", err)
		}
		documentID = asString(payload["document_id"])
		if documentID == "" {
			t.Fatalf("missing document_id in create response: %s", raw)
		}
	})

	// 2) POST /api/v2/documents/{id}/submit with Idempotency-Key + If-Match
	t.Run("SubmitForReview", func(t *testing.T) {
		submitBody := map[string]any{
			"route_id":     routeID,
			"content_hash": contentHash,
		}

		resp, raw := doJSONRequest(t, client, http.MethodPost, fmt.Sprintf("%s/api/v2/documents/%s/submit", baseURL, documentID), submitBody, map[string]string{
			"X-Tenant-ID":      tenantID,
			"X-User-ID":        userID,
			"Idempotency-Key":  "e2e-submit-idem-1",
			"If-Match":         "\"v1\"",
			"X-User-Roles":     userRoles,
			"Content-Type":     "application/json",
			"Accept":           "application/json",
			"X-Request-Source": "integration-e2e",
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("submit status=%d body=%s", resp.StatusCode, raw)
		}

		submitETag = strings.TrimSpace(resp.Header.Get("ETag"))
		if submitETag == "" {
			t.Fatalf("submit response missing ETag")
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("decode submit response: %v", err)
		}
		instanceID = asString(payload["instance_id"])
		if instanceID == "" {
			t.Fatalf("missing instance_id in submit response: %s", raw)
		}
	})

	// 2b) Replay submit with same idempotency key; expect replay marker header.
	t.Run("SubmitReplayHeader", func(t *testing.T) {
		submitBody := map[string]any{
			"route_id":     routeID,
			"content_hash": contentHash,
		}

		resp, raw := doJSONRequest(t, client, http.MethodPost, fmt.Sprintf("%s/api/v2/documents/%s/submit", baseURL, documentID), submitBody, map[string]string{
			"X-Tenant-ID":     tenantID,
			"X-User-ID":       userID,
			"Idempotency-Key": "e2e-submit-idem-1",
			"If-Match":        "\"v1\"",
			"X-User-Roles":    userRoles,
		})

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
			t.Fatalf("submit replay unexpected status=%d body=%s", resp.StatusCode, raw)
		}

		replayHeader := strings.TrimSpace(resp.Header.Get("Idempotent-Replay"))
		if replayHeader == "" {
			t.Fatalf("expected Idempotent-Replay header on replay submit; status=%d body=%s", resp.StatusCode, raw)
		}
	})

	// 3) GET /api/v2/documents/{id}/approval-instance (fallback to approval instance route)
	t.Run("GetApprovalInstanceAfterSubmit", func(t *testing.T) {
		status, raw := getApprovalInstance(t, client, baseURL, tenantID, userID, userRoles, documentID, instanceID)
		if status != http.StatusOK {
			t.Fatalf("get approval instance status=%d body=%s", status, raw)
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("decode approval instance response: %v", err)
		}

		gotStatus := asString(payload["status"])
		if gotStatus == "" {
			t.Fatalf("missing status in approval instance response: %s", raw)
		}
		if gotStatus != "in_progress" && gotStatus != "approved" {
			t.Fatalf("unexpected instance status after submit: %q body=%s", gotStatus, raw)
		}

		if stages, ok := payload["stages"].([]any); ok {
			for _, stage := range stages {
				stageMap, ok := stage.(map[string]any)
				if !ok {
					continue
				}
				stageID := asString(stageMap["stage_id"])
				if stageID != "" {
					stageIDs = append(stageIDs, stageID)
				}
			}
		}
	})

	// 4) POST signoff stage 1
	t.Run("SignoffStage1", func(t *testing.T) {
		stageID := stageIDAt(stageIDs, 0, os.Getenv("METALDOCS_E2E_STAGE1_ID"))
		if stageID == "" {
			t.Skip("no stage 1 id found in instance response; set METALDOCS_E2E_STAGE1_ID to force this step")
		}

		resp, raw := doJSONRequest(t, client, http.MethodPost,
			fmt.Sprintf("%s/api/v2/approval/instances/%s/stages/%s/signoffs", baseURL, instanceID, stageID),
			map[string]any{
				"decision":       "approve",
				"password_token": "e2e-token-1",
				"content_hash":   contentHash,
			},
			map[string]string{
				"X-Tenant-ID":     tenantID,
				"X-User-ID":       userID,
				"Idempotency-Key": "e2e-signoff-1",
				"If-Match":        submitETag,
				"X-User-Roles":    userRoles,
			},
		)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("stage1 signoff status=%d body=%s", resp.StatusCode, raw)
		}
	})

	// 5) POST signoff stage 2
	t.Run("SignoffStage2", func(t *testing.T) {
		stageID := stageIDAt(stageIDs, 1, os.Getenv("METALDOCS_E2E_STAGE2_ID"))
		if stageID == "" {
			t.Skip("no stage 2 id found in instance response; set METALDOCS_E2E_STAGE2_ID to force this step")
		}

		resp, raw := doJSONRequest(t, client, http.MethodPost,
			fmt.Sprintf("%s/api/v2/approval/instances/%s/stages/%s/signoffs", baseURL, instanceID, stageID),
			map[string]any{
				"decision":       "approve",
				"password_token": "e2e-token-2",
				"content_hash":   contentHash,
			},
			map[string]string{
				"X-Tenant-ID":     tenantID,
				"X-User-ID":       userID,
				"Idempotency-Key": "e2e-signoff-2",
				"If-Match":        submitETag,
				"X-User-Roles":    userRoles,
			},
		)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("stage2 signoff status=%d body=%s", resp.StatusCode, raw)
		}
	})

	// 6) GET approval instance and expect completion
	t.Run("GetApprovalInstanceCompleted", func(t *testing.T) {
		status, raw := getApprovalInstance(t, client, baseURL, tenantID, userID, userRoles, documentID, instanceID)
		if status != http.StatusOK {
			t.Fatalf("get approval instance status=%d body=%s", status, raw)
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("decode final instance response: %v", err)
		}

		gotStatus := asString(payload["status"])
		if gotStatus != "approved" && gotStatus != "completed" {
			t.Fatalf("expected completed/approved instance after signoffs, got %q body=%s", gotStatus, raw)
		}
	})

	// 7) POST /api/v2/documents/{id}/publish
	t.Run("Publish", func(t *testing.T) {
		resp, raw := doJSONRequest(t, client, http.MethodPost, fmt.Sprintf("%s/api/v2/documents/%s/publish", baseURL, documentID), nil, map[string]string{
			"X-Tenant-ID":     tenantID,
			"X-User-ID":       userID,
			"Idempotency-Key": "e2e-publish-1",
			"If-Match":        submitETag,
			"X-User-Roles":    userRoles,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("publish status=%d body=%s", resp.StatusCode, raw)
		}
	})

	// 8) Optional DB-level governance_events assertion when DATABASE_URL is set
	t.Run("GovernanceEventsCountOptionalDB", func(t *testing.T) {
		db := openOptionalDirectDB(t)
		if db == nil {
			t.Skip("DATABASE_URL/METALDOCS_DATABASE_URL not set; skipping DB verification")
		}
		defer db.Close()

		var count int
		if err := db.QueryRowContext(context.Background(), `
			SELECT count(*)
			  FROM metaldocs.governance_events
			 WHERE tenant_id = $1::uuid
			   AND resource_type = 'document'
			   AND resource_id = $2`,
			tenantID, documentID,
		).Scan(&count); err != nil {
			t.Fatalf("query governance_events count: %v", err)
		}
		if count < 1 {
			t.Fatalf("expected at least one governance event for document %s, got %d", documentID, count)
		}
	})
}

func doJSONRequest(t *testing.T, client *http.Client, method, url string, body any, headers map[string]string) (*http.Response, string) {
	t.Helper()

	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		payload = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, url, err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		if strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("http %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return resp, strings.TrimSpace(string(raw))
}

func getApprovalInstance(t *testing.T, client *http.Client, baseURL, tenantID, userID, userRoles, documentID, instanceID string) (int, string) {
	t.Helper()

	headers := map[string]string{
		"X-Tenant-ID":  tenantID,
		"X-User-ID":    userID,
		"X-User-Roles": userRoles,
	}

	resp, raw := doJSONRequest(t, client, http.MethodGet, fmt.Sprintf("%s/api/v2/documents/%s/approval-instance", baseURL, documentID), nil, headers)
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed {
		return resp.StatusCode, raw
	}

	if strings.TrimSpace(instanceID) == "" {
		return resp.StatusCode, raw
	}

	resp2, raw2 := doJSONRequest(t, client, http.MethodGet, fmt.Sprintf("%s/api/v2/approval/instances/%s", baseURL, instanceID), nil, headers)
	return resp2.StatusCode, raw2
}

func openOptionalDirectDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("METALDOCS_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" {
		return nil
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		t.Skipf("integration DB unreachable: %v", err)
	}
	return db
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func asString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func stageIDAt(stageIDs []string, idx int, fallback string) string {
	if idx >= 0 && idx < len(stageIDs) && strings.TrimSpace(stageIDs[idx]) != "" {
		return strings.TrimSpace(stageIDs[idx])
	}
	return strings.TrimSpace(fallback)
}
