package approvalhttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
)

func supersedeTestMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/documents/{id}/supersede", h.SupersedeHandler)
	return mux
}

func TestSupersedeHandler(t *testing.T) {
	origPublishSuperseding := publishSuperseding
	t.Cleanup(func() {
		publishSuperseding = origPublishSuperseding
	})

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{
			name:       "happy path",
			wantStatus: http.StatusOK,
		},
		{
			name:       "authz denied",
			svcErr:     authz.ErrCapabilityDenied{Capability: "doc.supersede", AreaCode: "tenant-1", ActorID: "actor-1"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "illegal transition",
			svcErr:     errors.New("approval: illegal transition"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "stale occ",
			svcErr:     repository.ErrStaleRevision,
			wantStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq application.SupersedeRequest
			publishSuperseding = func(_ *Handler, _ context.Context, _ *sql.DB, req application.SupersedeRequest) (application.SupersedeResult, error) {
				gotReq = req
				if tt.svcErr != nil {
					return application.SupersedeResult{}, tt.svcErr
				}
				return application.SupersedeResult{
					NewDocumentStatus:   "published",
					PriorDocumentStatus: "superseded",
				}, nil
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc-2/supersede", strings.NewReader(`{"superseded_document_id":"11111111-1111-1111-1111-111111111111"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req.Header.Set("X-User-ID", "actor-1")
			req.Header.Set("Idempotency-Key", "idem-1")
			req.Header.Set("If-Match", "\"v5\"")

			rr := httptest.NewRecorder()
			supersedeTestMux(&Handler{}).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}

			if gotReq.TenantID != "tenant-1" || gotReq.NewDocumentID != "doc-2" || gotReq.PriorDocumentID != "11111111-1111-1111-1111-111111111111" || gotReq.SupersededBy != "actor-1" {
				t.Fatalf("unexpected service request: %+v", gotReq)
			}
			if gotReq.NewRevisionVersion != 5 || gotReq.PriorRevisionVersion != 5 {
				t.Fatalf("unexpected revision mapping: %+v", gotReq)
			}

			if tt.wantStatus == http.StatusOK {
				var out contracts.SupersedeResponse
				if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if out.DocumentID != "doc-2" || out.SupersededID != "11111111-1111-1111-1111-111111111111" {
					t.Fatalf("unexpected response: %+v", out)
				}
			}
		})
	}
}
