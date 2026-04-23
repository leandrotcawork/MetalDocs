package documentshttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	v2domain "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/iam/authz"
)

type fakeFillInService struct {
	setPlaceholderErr error
	setZoneErr        error

	gotPlaceholderReq SetPlaceholderValueRequest
	gotZoneReq        SetZoneContentRequest
}

func (f *fakeFillInService) SetPlaceholderValue(_ context.Context, req SetPlaceholderValueRequest) error {
	f.gotPlaceholderReq = req
	return f.setPlaceholderErr
}

func (f *fakeFillInService) SetZoneContent(_ context.Context, req SetZoneContentRequest) error {
	f.gotZoneReq = req
	return f.setZoneErr
}

func fillInTestMux(h *FillInHandler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func TestFillInHandler_PutPlaceholderValue(t *testing.T) {
	tests := []struct {
		name       string
		svcErr     error
		body       string
		wantStatus int
	}{
		{name: "ok", body: `{"value":"ABC"}`, wantStatus: http.StatusOK},
		{name: "capability denied", body: `{"value":"ABC"}`, svcErr: authz.ErrCapabilityDenied{Capability: "doc.edit_draft", AreaCode: "qa", ActorID: "u1"}, wantStatus: http.StatusForbidden},
		{name: "not found", body: `{"value":"ABC"}`, svcErr: v2domain.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "not draft", body: `{"value":"ABC"}`, svcErr: v2domain.ErrInvalidStateTransition, wantStatus: http.StatusConflict},
		{name: "validation", body: `{"value":"abc"}`, svcErr: v2domain.ErrValidationFailed, wantStatus: http.StatusUnprocessableEntity},
		{name: "bad json", body: `{"value":`, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeFillInService{setPlaceholderErr: tt.svcErr}
			h := NewFillInHandler(svc)
			mux := fillInTestMux(h)

			req := httptest.NewRequest(http.MethodPut, "/api/v2/documents/rev-1/placeholders/p1", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req.Header.Set("X-User-ID", "user-1")
			req.Header.Set("X-Request-ID", "req-1")
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var out map[string]any
				if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if out["placeholder_id"] != "p1" {
					t.Fatalf("placeholder_id=%v", out["placeholder_id"])
				}
				if _, ok := out["updated_at"].(string); !ok {
					t.Fatalf("updated_at missing")
				}
			}
		})
	}
}

func TestFillInHandler_MapErrorInternal(t *testing.T) {
	status, body := mapFillInError(errors.New("boom"))
	if status != http.StatusInternalServerError {
		t.Fatalf("status=%d", status)
	}
	if body.Error.Code != "internal.unknown" {
		t.Fatalf("code=%q", body.Error.Code)
	}
}

func TestFillInHandler_PutZoneContent(t *testing.T) {
	tests := []struct {
		name       string
		svcErr     error
		body       string
		wantStatus int
	}{
		{name: "ok", body: `{"content_ooxml":"<w:p>ok</w:p>"}`, wantStatus: http.StatusOK},
		{name: "capability denied", body: `{"content_ooxml":"<w:p>ok</w:p>"}`, svcErr: authz.ErrCapabilityDenied{Capability: "doc.edit_draft", AreaCode: "qa", ActorID: "u1"}, wantStatus: http.StatusForbidden},
		{name: "not found", body: `{"content_ooxml":"<w:p>ok</w:p>"}`, svcErr: v2domain.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "not draft", body: `{"content_ooxml":"<w:p>ok</w:p>"}`, svcErr: v2domain.ErrInvalidStateTransition, wantStatus: http.StatusConflict},
		{name: "validation", body: `{"content_ooxml":"<w:tbl/>"}`, svcErr: v2domain.ErrValidationFailed, wantStatus: http.StatusUnprocessableEntity},
		{name: "bad json", body: `{"content_ooxml":`, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeFillInService{setZoneErr: tt.svcErr}
			h := NewFillInHandler(svc)
			mux := fillInTestMux(h)

			req := httptest.NewRequest(http.MethodPut, "/api/v2/documents/rev-1/zones/z1", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req.Header.Set("X-User-ID", "user-1")
			req.Header.Set("X-Request-ID", "req-1")
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", rr.Code, tt.wantStatus, rr.Body.String())
			}
		})
	}
}
