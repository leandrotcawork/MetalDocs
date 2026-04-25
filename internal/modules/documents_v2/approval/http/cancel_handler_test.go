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
	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func cancelTestMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/approval/instances/{instance_id}/cancel", h.CancelHandler)
	return mux
}

func TestCancelHandler(t *testing.T) {
	origCancelInstance := cancelInstance
	t.Cleanup(func() {
		cancelInstance = origCancelInstance
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
			body:         `{"reason":"request withdrawn"}`,
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
			name:         "instance completed",
			body:         `{"reason":"request withdrawn"}`,
			svcErr:       repository.ErrInstanceCompleted,
			wantStatus:   http.StatusConflict,
			wantSvcCalls: true,
		},
		{
			name:         "authz denied",
			body:         `{"reason":"request withdrawn"}`,
			svcErr:       authz.ErrCapabilityDenied{Capability: "workflow.instance.cancel", AreaCode: "tenant-1", ActorID: "actor-1"},
			wantStatus:   http.StatusForbidden,
			wantSvcCalls: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			var gotReq application.CancelInput
			cancelInstance = func(_ *Handler, _ context.Context, _ *sql.DB, req application.CancelInput) (application.CancelResult, error) {
				called = true
				gotReq = req
				if tt.svcErr != nil {
					return application.CancelResult{}, tt.svcErr
				}
				return application.CancelResult{DocumentID: "doc-4"}, nil
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v2/approval/instances/inst-4/cancel", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-1", []iamdomain.Role{}))
			req.Header.Set("Idempotency-Key", "idem-1")
			req.Header.Set("If-Match", "\"v9\"")

			rr := httptest.NewRecorder()
			cancelTestMux(&Handler{}).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if called != tt.wantSvcCalls {
				t.Fatalf("service called = %v, want %v", called, tt.wantSvcCalls)
			}

			if tt.wantSvcCalls {
				if gotReq.TenantID != "tenant-1" || gotReq.InstanceID != "inst-4" || gotReq.ActorUserID != "actor-1" || gotReq.ExpectedRevisionVersion != 9 {
					t.Fatalf("unexpected service request: %+v", gotReq)
				}
			}

			if tt.wantStatus == http.StatusOK {
				var out contracts.CancelResponse
				if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if out.DocumentID != "doc-4" {
					t.Fatalf("document_id = %q, want %q", out.DocumentID, "doc-4")
				}
			}
		})
	}
}
