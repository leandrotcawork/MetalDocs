package approvalhttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
	"metaldocs/internal/modules/iam/authz"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func obsoleteTestMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/documents/{id}/obsolete", h.ObsoleteHandler)
	return mux
}

func TestObsoleteHandler(t *testing.T) {
	origMarkObsolete := markObsolete
	t.Cleanup(func() {
		markObsolete = origMarkObsolete
	})

	tests := []struct {
		name         string
		body         string
		svcErr       error
		wantStatus   int
		wantSvcCalls bool
	}{
		{
			name:         "happy path",
			body:         `{"reason":"sunset old release"}`,
			wantStatus:   http.StatusOK,
			wantSvcCalls: true,
		},
		{
			name:         "reason missing",
			body:         `{"reason":""}`,
			wantStatus:   http.StatusBadRequest,
			wantSvcCalls: false,
		},
		{
			name:         "authz denied",
			body:         `{"reason":"sunset old release"}`,
			svcErr:       authz.ErrCapabilityDenied{Capability: "doc.obsolete", AreaCode: "tenant-1", ActorID: "actor-1"},
			wantStatus:   http.StatusForbidden,
			wantSvcCalls: true,
		},
		{
			name:         "illegal transition",
			body:         `{"reason":"sunset old release"}`,
			svcErr:       application.ErrInvalidObsoleteSource,
			wantStatus:   http.StatusBadRequest,
			wantSvcCalls: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			var gotReq application.MarkObsoleteRequest
			markObsolete = func(_ *Handler, _ context.Context, _ *sql.DB, req application.MarkObsoleteRequest) (application.MarkObsoleteResult, error) {
				called = true
				gotReq = req
				if tt.svcErr != nil {
					return application.MarkObsoleteResult{}, tt.svcErr
				}
				return application.MarkObsoleteResult{PriorStatus: "published"}, nil
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc-3/obsolete", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-1", []iamdomain.Role{}))
			req.Header.Set("Idempotency-Key", "idem-1")
			req.Header.Set("If-Match", "\"v7\"")

			rr := httptest.NewRecorder()
			obsoleteTestMux(&Handler{}).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if called != tt.wantSvcCalls {
				t.Fatalf("service called = %v, want %v", called, tt.wantSvcCalls)
			}

			if tt.wantSvcCalls {
				if gotReq.TenantID != "tenant-1" || gotReq.DocumentID != "doc-3" || gotReq.MarkedBy != "actor-1" || gotReq.RevisionVersion != 7 {
					t.Fatalf("unexpected service request: %+v", gotReq)
				}
			}

			if tt.wantStatus == http.StatusOK {
				var out contracts.ObsoleteResponse
				if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if out.DocumentID != "doc-3" {
					t.Fatalf("document_id = %q, want %q", out.DocumentID, "doc-3")
				}
			}
		})
	}
}
