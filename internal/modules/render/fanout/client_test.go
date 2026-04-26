package fanout

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_Fanout_Success(t *testing.T) {
	var gotPath, gotContentType, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("content-type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content_hash":      "deadbeef",
			"final_docx_s3_key": "out/a.docx",
			"unreplaced_vars":   []string{"missing_var"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())
	resp, err := c.Fanout(context.Background(), FanoutRequest{
		TenantID:          "t1",
		RevisionID:        "r1",
		BodyDocxS3Key:     "templates/x.docx",
		PlaceholderValues: map[string]string{"doc_code": "ABC"},
		Composition:       json.RawMessage(`{}`),
		ResolvedValues:    map[string]any{"doc_code": "ABC"},
	})
	if err != nil {
		t.Fatalf("fanout err: %v", err)
	}
	if gotPath != "/render/fanout" {
		t.Errorf("path = %q, want /render/fanout", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("content-type = %q", gotContentType)
	}
	if !strings.Contains(gotBody, `"tenant_id":"t1"`) {
		t.Errorf("body missing tenant_id: %s", gotBody)
	}
	if resp.ContentHash != "deadbeef" || resp.FinalDocxS3Key != "out/a.docx" {
		t.Errorf("resp = %+v", resp)
	}
	if len(resp.UnreplacedVars) != 1 || resp.UnreplacedVars[0] != "missing_var" {
		t.Errorf("unreplaced_vars = %v", resp.UnreplacedVars)
	}
}

func TestClient_Fanout_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())
	_, err := c.Fanout(context.Background(), FanoutRequest{})
	if err == nil {
		t.Fatal("expected error on non-200")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("err should mention status: %v", err)
	}
}
