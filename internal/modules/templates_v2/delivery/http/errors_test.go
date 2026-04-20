package http

import (
	"errors"
	"net/http"
	"testing"

	"metaldocs/internal/modules/templates_v2/domain"
)

func TestMapErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "not found", err: domain.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "not_found"},
		{name: "key conflict", err: domain.ErrKeyConflict, wantStatus: http.StatusConflict, wantCode: "key_conflict"},
		{name: "invalid visibility", err: domain.ErrInvalidVisibility, wantStatus: http.StatusBadRequest, wantCode: "invalid_visibility"},
		{name: "invalid state transition", err: domain.ErrInvalidStateTransition, wantStatus: http.StatusConflict, wantCode: "invalid_state_transition"},
		{name: "stale base", err: domain.ErrStaleBase, wantStatus: http.StatusConflict, wantCode: "stale_base"},
		{name: "content hash mismatch", err: domain.ErrContentHashMismatch, wantStatus: http.StatusConflict, wantCode: "content_hash_mismatch"},
		{name: "upload missing", err: domain.ErrUploadMissing, wantStatus: http.StatusConflict, wantCode: "upload_missing"},
		{name: "iso segregation violation", err: domain.ErrISOSegregationViolation, wantStatus: http.StatusForbidden, wantCode: "iso_segregation_violation"},
		{name: "forbidden role", err: domain.ErrForbiddenRole, wantStatus: http.StatusForbidden, wantCode: "forbidden_role"},
		{name: "forbidden", err: domain.ErrForbidden, wantStatus: http.StatusForbidden, wantCode: "forbidden"},
		{name: "archived", err: domain.ErrArchived, wantStatus: http.StatusConflict, wantCode: "archived"},
		{name: "invalid approval config", err: domain.ErrInvalidApprovalConfig, wantStatus: http.StatusBadRequest, wantCode: "invalid_approval_config"},
		{name: "default", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, code := MapErr(tc.err)
			if status != tc.wantStatus {
				t.Fatalf("status: got %d want %d", status, tc.wantStatus)
			}
			if code != tc.wantCode {
				t.Fatalf("code: got %q want %q", code, tc.wantCode)
			}
		})
	}
}
