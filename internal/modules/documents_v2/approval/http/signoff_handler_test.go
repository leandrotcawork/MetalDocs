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
	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
	approvalsignature "metaldocs/internal/modules/documents_v2/approval/infra/signature"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type fakeDecisionService struct {
	gotReq application.SignoffRequest
	result application.SignoffResult
	err    error
}

func (f *fakeDecisionService) RecordSignoff(_ context.Context, _ *sql.DB, req application.SignoffRequest) (application.SignoffResult, error) {
	f.gotReq = req
	if f.err != nil {
		return application.SignoffResult{}, f.err
	}
	return f.result, nil
}

func signoffTestMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/approval/instances/{instance_id}/stages/{stage_id}/signoff", h.SignoffHandler)
	return mux
}

func TestSignoffHandler_HappyApprove(t *testing.T) {
	fakeSvc := &fakeDecisionService{result: application.SignoffResult{InstanceApproved: true}}
	h := &Handler{decisionSvc: fakeSvc}
	mux := signoffTestMux(h)

	body := `{"decision":"approve","reason":"","password_token":"secret","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/approval/instances/inst-1/stages/stg-1/signoff", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-1", []iamdomain.Role{}))
	req.Header.Set("Idempotency-Key", "idem-1")
	req.Header.Set("If-Match", "v3")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var out contracts.SignoffResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Outcome != "approved" {
		t.Fatalf("outcome = %q, want %q", out.Outcome, "approved")
	}
	if fakeSvc.gotReq.TenantID != "tenant-1" || fakeSvc.gotReq.InstanceID != "inst-1" || fakeSvc.gotReq.StageInstanceID != "stg-1" || fakeSvc.gotReq.ActorUserID != "actor-1" {
		t.Fatalf("unexpected request mapped to service: %+v", fakeSvc.gotReq)
	}
	if fakeSvc.gotReq.Decision != "approve" {
		t.Fatalf("decision = %q, want %q", fakeSvc.gotReq.Decision, "approve")
	}
	if fakeSvc.gotReq.SignatureMethod != "password_reauth" {
		t.Fatalf("signature_method = %q, want %q", fakeSvc.gotReq.SignatureMethod, "password_reauth")
	}
	if got := fakeSvc.gotReq.SignaturePayload["password_token"]; got != "secret" {
		t.Fatalf("signature payload password_token = %#v", got)
	}
	if got := fakeSvc.gotReq.ContentFormData["_content_hash"]; got != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("content form data _content_hash = %#v", got)
	}
}

func TestSignoffHandler_HappyReject(t *testing.T) {
	fakeSvc := &fakeDecisionService{result: application.SignoffResult{InstanceRejected: true}}
	h := &Handler{decisionSvc: fakeSvc}
	mux := signoffTestMux(h)

	body := `{"decision":"reject","reason":"wrong value","password_token":"secret","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/approval/instances/inst-1/stages/stg-1/signoff", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")
	req.Header.Set("If-Match", "v2")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var out contracts.SignoffResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Outcome != "rejected" {
		t.Fatalf("outcome = %q, want %q", out.Outcome, "rejected")
	}
}

func TestSignoffHandler_SoDViolation(t *testing.T) {
	h := &Handler{decisionSvc: &fakeDecisionService{err: domain.ErrAuthorCannotSign}}
	mux := signoffTestMux(h)

	body := `{"decision":"approve","reason":"","password_token":"secret","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/approval/instances/inst-1/stages/stg-1/signoff", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")
	req.Header.Set("If-Match", "v1")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestSignoffHandler_SignatureInvalid(t *testing.T) {
	h := &Handler{decisionSvc: &fakeDecisionService{err: approvalsignature.ErrInvalidCredentials}}
	mux := signoffTestMux(h)

	body := `{"decision":"approve","reason":"","password_token":"bad","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/approval/instances/inst-1/stages/stg-1/signoff", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")
	req.Header.Set("If-Match", "v1")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestSignoffHandler_ContentHashMismatch(t *testing.T) {
	h := &Handler{decisionSvc: &fakeDecisionService{err: ErrContentHashMismatch}}
	mux := signoffTestMux(h)

	body := `{"decision":"approve","reason":"","password_token":"secret","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/approval/instances/inst-1/stages/stg-1/signoff", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")
	req.Header.Set("If-Match", "v1")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusPreconditionFailed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusPreconditionFailed)
	}
}

func TestSignoffHandler_MissingIdempotencyKey(t *testing.T) {
	h := &Handler{decisionSvc: &fakeDecisionService{}}
	mux := signoffTestMux(h)

	body := `{"decision":"approve","reason":"","password_token":"secret","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/approval/instances/inst-1/stages/stg-1/signoff", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "v1")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	var out contracts.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Error.Code != "idempotency.key_required" {
		t.Fatalf("error.code = %q, want %q", out.Error.Code, "idempotency.key_required")
	}
}

func TestSignoffHandler_MissingIfMatch(t *testing.T) {
	h := &Handler{decisionSvc: &fakeDecisionService{}}
	mux := signoffTestMux(h)

	body := `{"decision":"approve","reason":"","password_token":"secret","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/approval/instances/inst-1/stages/stg-1/signoff", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusPreconditionRequired {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusPreconditionRequired)
	}
}

func TestSignoffHandler_MalformedIfMatch(t *testing.T) {
	h := &Handler{decisionSvc: &fakeDecisionService{}}
	mux := signoffTestMux(h)

	body := `{"decision":"approve","reason":"","password_token":"secret","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/approval/instances/inst-1/stages/stg-1/signoff", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")
	req.Header.Set("If-Match", "invalid")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
