package approvalhttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
)

type fakeReadServicePublish struct {
	inst *domain.Instance
}

func (f *fakeReadServicePublish) LoadInstance(_ context.Context, _ *sql.DB, _, _, _ string) (*domain.Instance, error) {
	return nil, nil
}

func (f *fakeReadServicePublish) LoadActiveInstanceByDocument(_ context.Context, _ *sql.DB, _, _ string) (*domain.Instance, error) {
	return f.inst, nil
}

func (f *fakeReadServicePublish) ListPendingForActor(_ context.Context, _ *sql.DB, _, _, _ string, _, _ int) ([]domain.Instance, error) {
	return nil, nil
}

func publishTestMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/documents/{id}/publish", h.PublishHandler)
	mux.HandleFunc("POST /api/v2/documents/{id}/schedule-publish", h.SchedulePublishHandler)
	return mux
}

func TestPublishHandler(t *testing.T) {
	origPublishApproved := publishApproved
	t.Cleanup(func() {
		publishApproved = origPublishApproved
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
			svcErr:     authz.ErrCapabilityDenied{Capability: "doc.publish", AreaCode: "tenant-1", ActorID: "actor-1"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "illegal transition",
			svcErr:     application.ErrInstanceNotApproved,
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
			var gotReq application.PublishRequest
			publishApproved = func(_ *Handler, _ context.Context, _ *sql.DB, req application.PublishRequest) (application.PublishResult, error) {
				gotReq = req
				if tt.svcErr != nil {
					return application.PublishResult{}, tt.svcErr
				}
				return application.PublishResult{DocumentID: "doc-1", NewStatus: "published"}, nil
			}

			fakeRead := &fakeReadServicePublish{
				inst: &domain.Instance{ID: "inst-1", DocumentID: "doc-1"},
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc-1/publish", nil)
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req.Header.Set("X-User-ID", "actor-1")
			req.Header.Set("Idempotency-Key", "idem-1")
			req.Header.Set("If-Match", "\"v3\"")

			rr := httptest.NewRecorder()
			publishTestMux(&Handler{readSvc: fakeRead}).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}

			if gotReq.TenantID != "tenant-1" || gotReq.InstanceID != "inst-1" || gotReq.PublishedBy != "actor-1" {
				t.Fatalf("unexpected service request: %+v", gotReq)
			}

			if tt.wantStatus == http.StatusOK {
				var out contracts.PublishResponse
				if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if out.DocumentID != "doc-1" || out.NewStatus != "published" {
					t.Fatalf("unexpected response: %+v", out)
				}
			}
		})
	}
}

func TestSchedulePublishHandler(t *testing.T) {
	origSchedulePublish := schedulePublish
	t.Cleanup(func() {
		schedulePublish = origSchedulePublish
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
			svcErr:     authz.ErrCapabilityDenied{Capability: "doc.publish", AreaCode: "tenant-1", ActorID: "actor-1"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "illegal transition",
			svcErr:     application.ErrInstanceNotApproved,
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
			var gotReq application.SchedulePublishRequest
			schedulePublish = func(_ *Handler, _ context.Context, _ *sql.DB, req application.SchedulePublishRequest) (application.SchedulePublishResult, error) {
				gotReq = req
				if tt.svcErr != nil {
					return application.SchedulePublishResult{}, tt.svcErr
				}
				return application.SchedulePublishResult{
					DocumentID:    "doc-1",
					EffectiveDate: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
				}, nil
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc-1/schedule-publish", strings.NewReader(`{"effective_from":"2026-05-01T12:00:00Z"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req.Header.Set("X-User-ID", "actor-1")
			req.Header.Set("Idempotency-Key", "idem-1")
			req.Header.Set("If-Match", "\"v4\"")

			rr := httptest.NewRecorder()
			publishTestMux(&Handler{}).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}

			if gotReq.TenantID != "tenant-1" || gotReq.InstanceID != "doc-1" || gotReq.ScheduledBy != "actor-1" {
				t.Fatalf("unexpected service request: %+v", gotReq)
			}
			if gotReq.EffectiveDate.UTC().Format(time.RFC3339) != "2026-05-01T12:00:00Z" {
				t.Fatalf("effective date = %s", gotReq.EffectiveDate.UTC().Format(time.RFC3339))
			}

			if tt.wantStatus == http.StatusOK {
				var out contracts.PublishResponse
				if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if out.DocumentID != "doc-1" || out.NewStatus != "scheduled" || out.EffectiveFrom != "2026-05-01T12:00:00Z" {
					t.Fatalf("unexpected response: %+v", out)
				}
			}
		})
	}
}
