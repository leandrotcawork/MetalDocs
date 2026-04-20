package http

import (
	"errors"
	"net/http"

	"metaldocs/internal/modules/templates_v2/domain"
)

func MapErr(err error) (httpStatus int, code string) {
	switch {
	case err == nil:
		return http.StatusOK, ""
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrKeyConflict):
		return http.StatusConflict, "key_conflict"
	case errors.Is(err, domain.ErrInvalidVisibility):
		return http.StatusBadRequest, "invalid_visibility"
	case errors.Is(err, domain.ErrInvalidStateTransition):
		return http.StatusConflict, "invalid_state_transition"
	case errors.Is(err, domain.ErrStaleBase):
		return http.StatusConflict, "stale_base"
	case errors.Is(err, domain.ErrContentHashMismatch):
		return http.StatusConflict, "content_hash_mismatch"
	case errors.Is(err, domain.ErrUploadMissing):
		return http.StatusConflict, "upload_missing"
	case errors.Is(err, domain.ErrISOSegregationViolation):
		return http.StatusForbidden, "iso_segregation_violation"
	case errors.Is(err, domain.ErrForbiddenRole):
		return http.StatusForbidden, "forbidden_role"
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, domain.ErrArchived):
		return http.StatusConflict, "archived"
	case errors.Is(err, domain.ErrInvalidApprovalConfig):
		return http.StatusBadRequest, "invalid_approval_config"
	default:
		return http.StatusInternalServerError, "internal_error"
	}
}
