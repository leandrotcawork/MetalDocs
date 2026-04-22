package approvalhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
	approvalsignature "metaldocs/internal/modules/documents_v2/approval/infra/signature"
	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
)

func TestMapErrorToResponse(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantMsg    string
	}{
		{
			name:       "repository stale revision",
			err:        repository.ErrStaleRevision,
			wantStatus: http.StatusConflict,
			wantCode:   "conflict.stale_revision",
			wantMsg:    repository.ErrStaleRevision.Error(),
		},
		{
			name:       "repository no active instance",
			err:        repository.ErrNoActiveInstance,
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found.instance",
			wantMsg:    repository.ErrNoActiveInstance.Error(),
		},
		{
			name:       "repository duplicate submission",
			err:        repository.ErrDuplicateSubmission,
			wantStatus: http.StatusConflict,
			wantCode:   "conflict.duplicate_submission",
			wantMsg:    repository.ErrDuplicateSubmission.Error(),
		},
		{
			name:       "repository actor already signed",
			err:        repository.ErrActorAlreadySigned,
			wantStatus: http.StatusConflict,
			wantCode:   "signoff.duplicate",
			wantMsg:    repository.ErrActorAlreadySigned.Error(),
		},
		{
			name:       "repository instance completed",
			err:        repository.ErrInstanceCompleted,
			wantStatus: http.StatusConflict,
			wantCode:   "state.instance_completed",
			wantMsg:    repository.ErrInstanceCompleted.Error(),
		},
		{
			name:       "repository route in use",
			err:        repository.ErrRouteInUse,
			wantStatus: http.StatusConflict,
			wantCode:   "route.in_use",
			wantMsg:    repository.ErrRouteInUse.Error(),
		},
		{
			name:       "repository duplicate route profile",
			err:        repository.ErrDuplicateRouteProfile,
			wantStatus: http.StatusConflict,
			wantCode:   "route.duplicate_profile",
			wantMsg:    repository.ErrDuplicateRouteProfile.Error(),
		},
		{
			name:       "domain sod submitter cannot sign",
			err:        domain.ErrAuthorCannotSign,
			wantStatus: http.StatusForbidden,
			wantCode:   "sod.submitter_cannot_sign",
			wantMsg:    domain.ErrAuthorCannotSign.Error(),
		},
		{
			name:       "domain sod cross-stage duplicate",
			err:        domain.ErrActorAlreadySigned,
			wantStatus: http.StatusForbidden,
			wantCode:   "sod.cross_stage_duplicate",
			wantMsg:    domain.ErrActorAlreadySigned.Error(),
		},
		{
			name:       "repository fk violation",
			err:        repository.ErrFKViolation,
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "db.fk_violation",
			wantMsg:    repository.ErrFKViolation.Error(),
		},
		{
			name:       "repository check violation",
			err:        repository.ErrCheckViolation,
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "db.check_violation",
			wantMsg:    repository.ErrCheckViolation.Error(),
		},
		{
			name:       "repository insufficient privilege",
			err:        repository.ErrInsufficientPrivilege,
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal.db_privilege_missing",
			wantMsg:    "internal error",
		},
		{
			name:       "repository unknown db",
			err:        repository.ErrUnknownDB,
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal.db_unknown",
			wantMsg:    "internal error",
		},
		{
			name:       "authz capability denied",
			err:        fmt.Errorf("wrap: %w", authz.ErrCapabilityDenied{Capability: "x", AreaCode: "tenant", ActorID: "u1"}),
			wantStatus: http.StatusForbidden,
			wantCode:   "authz.capability_denied",
			wantMsg:    "wrap: authz: capability \"x\" denied for actor \"u1\" in area \"tenant\"",
		},
		{
			name:       "application reason required",
			err:        application.ErrReasonRequired,
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation.reason_required",
			wantMsg:    application.ErrReasonRequired.Error(),
		},
		{
			name:       "application route not found",
			err:        application.ErrRouteNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found.route",
			wantMsg:    application.ErrRouteNotFound.Error(),
		},
		{
			name:       "context deadline exceeded",
			err:        context.DeadlineExceeded,
			wantStatus: http.StatusGatewayTimeout,
			wantCode:   "timeout",
			wantMsg:    "internal error",
		},
		{
			name:       "context canceled",
			err:        context.Canceled,
			wantStatus: http.StatusGatewayTimeout,
			wantCode:   "timeout",
			wantMsg:    "internal error",
		},
		{
			name: "json syntax error",
			err: func() error {
				var v map[string]any
				return json.Unmarshal([]byte("{"), &v)
			}(),
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation.json_decode",
			wantMsg:    "unexpected end of JSON input",
		},
		{
			name: "json type error",
			err: func() error {
				var v struct {
					N int `json:"n"`
				}
				return json.Unmarshal([]byte(`{"n":"x"}`), &v)
			}(),
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation.json_type_error",
			wantMsg:    "json: cannot unmarshal string into Go struct field .n of type int",
		},
		{
			name:       "io EOF",
			err:        io.EOF,
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation.empty_body",
			wantMsg:    io.EOF.Error(),
		},
		{
			name:       "contracts content type",
			err:        contracts.ErrContentType,
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "validation.content_type",
			wantMsg:    contracts.ErrContentType.Error(),
		},
		{
			name:       "contracts body too large",
			err:        contracts.ErrBodyTooLarge,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   "validation.body_too_large",
			wantMsg:    contracts.ErrBodyTooLarge.Error(),
		},
		{
			name:       "contracts empty body",
			err:        contracts.ErrEmptyBody,
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation.empty_body",
			wantMsg:    contracts.ErrEmptyBody.Error(),
		},
		{
			name:       "contracts duplicate key",
			err:        contracts.ErrDuplicateKey,
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation.duplicate_key",
			wantMsg:    contracts.ErrDuplicateKey.Error(),
		},
		{
			name:       "if-match required",
			err:        ErrIfMatchRequired,
			wantStatus: http.StatusPreconditionRequired,
			wantCode:   "precondition.if_match_required",
			wantMsg:    ErrIfMatchRequired.Error(),
		},
		{
			name:       "if-match malformed",
			err:        ErrIfMatchMalformed,
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation.if_match_malformed",
			wantMsg:    ErrIfMatchMalformed.Error(),
		},
		{
			name:       "idempotency key required",
			err:        ErrIdempotencyRequired,
			wantStatus: http.StatusBadRequest,
			wantCode:   "idempotency.key_required",
			wantMsg:    ErrIdempotencyRequired.Error(),
		},
		{
			name:       "content hash mismatch",
			err:        ErrContentHashMismatch,
			wantStatus: http.StatusPreconditionFailed,
			wantCode:   "precondition.content_hash_mismatch",
			wantMsg:    ErrContentHashMismatch.Error(),
		},
		{
			name:       "signature invalid",
			err:        approvalsignature.ErrInvalidCredentials,
			wantStatus: http.StatusUnauthorized,
			wantCode:   "authn.signature_invalid",
			wantMsg:    approvalsignature.ErrInvalidCredentials.Error(),
		},
		{
			name:       "signature rate-limited",
			err:        approvalsignature.ErrRateLimited,
			wantStatus: http.StatusTooManyRequests,
			wantCode:   "authn.signature_rate_limited",
			wantMsg:    approvalsignature.ErrRateLimited.Error(),
		},
		{
			name:       "generic validation error",
			err:        errors.New("route_id is required"),
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation.request_invalid",
			wantMsg:    "route_id is required",
		},
		{
			name:       "unknown error",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal.unknown",
			wantMsg:    "internal error",
		},
		{
			name:       "nil error",
			err:        nil,
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal.unknown",
			wantMsg:    "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := MapErrorToResponse(tt.err)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status, tt.wantStatus)
			}
			if body.Error.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q", body.Error.Code, tt.wantCode)
			}
			if body.Error.Message != tt.wantMsg {
				t.Fatalf("message = %q, want %q", body.Error.Message, tt.wantMsg)
			}
		})
	}
}

func TestMapErrorToResponse_WrappedSentinel(t *testing.T) {
	err := fmt.Errorf("service: %w", repository.ErrStaleRevision)
	status, body := MapErrorToResponse(err)

	if status != http.StatusConflict {
		t.Fatalf("status = %d, want %d", status, http.StatusConflict)
	}
	if body.Error.Code != "conflict.stale_revision" {
		t.Fatalf("code = %q, want %q", body.Error.Code, "conflict.stale_revision")
	}
	if body.Error.Message != err.Error() {
		t.Fatalf("message = %q, want %q", body.Error.Message, err.Error())
	}
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()

	WriteError(rr, "req-123", repository.ErrStaleRevision)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want %q", got, "application/json")
	}

	var body contracts.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.RequestID != "req-123" {
		t.Fatalf("request_id = %q, want %q", body.RequestID, "req-123")
	}
	if body.Error.Code != "conflict.stale_revision" {
		t.Fatalf("code = %q, want %q", body.Error.Code, "conflict.stale_revision")
	}
}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	payload := map[string]string{"ok": "yes"}

	WriteJSON(rr, http.StatusAccepted, payload)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want %q", got, "application/json")
	}

	want, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	if got := bytes.TrimSpace(rr.Body.Bytes()); !bytes.Equal(got, want) {
		t.Fatalf("body = %s, want %s", got, want)
	}
}
