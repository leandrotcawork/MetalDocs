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

type submitServiceI interface {
	SubmitRevisionForReview(ctx context.Context, db *sql.DB, req application.SubmitRequest) (application.SubmitResult, error)
}

type fakeSubmitService struct {
	result application.SubmitResult
	err    error
	gotReq application.SubmitRequest
}

func (f *fakeSubmitService) SubmitRevisionForReview(_ context.Context, _ *sql.DB, req application.SubmitRequest) (application.SubmitResult, error) {
	f.gotReq = req
	if f.err != nil {
		return application.SubmitResult{}, f.err
	}
	return f.result, nil
}

func submitTestMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/documents/{id}/submit", h.SubmitHandler)
	return mux
}

func TestSubmitHandler(t *testing.T) {
	tests := []struct {
		name           string
		ifMatch        string
		body           string
		svcErr         error
		wantStatus     int
		wantInstanceID string
		wantETag       string
	}{
		{
			name:           "happy path",
			ifMatch:        "\"v3\"",
			body:           `{"route_id":"11111111-1111-1111-1111-111111111111","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
			svcErr:         nil,
			wantStatus:     http.StatusCreated,
			wantInstanceID: "inst-123",
			wantETag:       "\"v4\"",
		},
		{
			name:       "missing if-match",
			ifMatch:    "",
			body:       `{"route_id":"11111111-1111-1111-1111-111111111111","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
			wantStatus: http.StatusPreconditionRequired,
		},
		{
			name:       "malformed if-match",
			ifMatch:    "oops",
			body:       `{"route_id":"11111111-1111-1111-1111-111111111111","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "validate fails",
			ifMatch:    "\"v1\"",
			body:       `{"route_id":"","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service stale revision",
			ifMatch:    "\"v2\"",
			body:       `{"route_id":"11111111-1111-1111-1111-111111111111","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
			svcErr:     repository.ErrStaleRevision,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "service capability denied",
			ifMatch:    "\"v2\"",
			body:       `{"route_id":"11111111-1111-1111-1111-111111111111","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
			svcErr:     authz.ErrCapabilityDenied{Capability: "doc.submit", AreaCode: "tenant", ActorID: "actor-1"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "service generic error",
			ifMatch:    "\"v2\"",
			body:       `{"route_id":"11111111-1111-1111-1111-111111111111","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
			svcErr:     errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeSubmitService{result: application.SubmitResult{InstanceID: "inst-123"}, err: tt.svcErr}
			h := &Handler{submitSvc: svc}
			mux := submitTestMux(h)

			req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc-1/submit", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req.Header.Set("X-User-ID", "actor-1")
			req.Header.Set("Idempotency-Key", "idem-1")
			req.Header.Set("X-Request-ID", "req-123")
			if tt.ifMatch != "" {
				req.Header.Set("If-Match", tt.ifMatch)
			}

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusCreated {
				var out contracts.SubmitResponse
				if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if out.InstanceID != tt.wantInstanceID {
					t.Fatalf("instance_id = %q, want %q", out.InstanceID, tt.wantInstanceID)
				}
				if got := rr.Header().Get("ETag"); got != tt.wantETag {
					t.Fatalf("etag = %q, want %q", got, tt.wantETag)
				}
			}
		})
	}
}
