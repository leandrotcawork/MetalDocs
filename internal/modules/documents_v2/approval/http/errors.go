package approvalhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
)

const internalErrorMessage = "internal error"

func MapErrorToResponse(err error) (statusCode int, body contracts.ErrorResponse) {
	statusCode = http.StatusInternalServerError
	code := "internal.unknown"

	switch {
	case errors.Is(err, repository.ErrStaleRevision):
		statusCode = http.StatusConflict
		code = "conflict.stale_revision"
	case errors.Is(err, repository.ErrNoActiveInstance):
		statusCode = http.StatusNotFound
		code = "not_found.instance"
	case errors.Is(err, repository.ErrDuplicateSubmission):
		statusCode = http.StatusConflict
		code = "conflict.duplicate_submission"
	case errors.Is(err, repository.ErrActorAlreadySigned):
		statusCode = http.StatusConflict
		code = "signoff.duplicate"
	case errors.Is(err, repository.ErrInstanceCompleted):
		statusCode = http.StatusConflict
		code = "state.instance_completed"
	case errors.Is(err, repository.ErrRouteInUse):
		statusCode = http.StatusConflict
		code = "route.in_use"
	case errors.Is(err, repository.ErrDuplicateRouteProfile):
		statusCode = http.StatusConflict
		code = "route.duplicate_profile"
	case errors.Is(err, repository.ErrFKViolation):
		statusCode = http.StatusUnprocessableEntity
		code = "db.fk_violation"
	case errors.Is(err, repository.ErrCheckViolation):
		statusCode = http.StatusUnprocessableEntity
		code = "db.check_violation"
	case errors.Is(err, repository.ErrInsufficientPrivilege):
		statusCode = http.StatusInternalServerError
		code = "internal.db_privilege_missing"
	case errors.Is(err, repository.ErrUnknownDB):
		statusCode = http.StatusInternalServerError
		code = "internal.db_unknown"
	default:
		var capabilityDenied authz.ErrCapabilityDenied
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &capabilityDenied):
			statusCode = http.StatusForbidden
			code = "authz.capability_denied"
		case errors.Is(err, application.ErrReasonRequired):
			statusCode = http.StatusBadRequest
			code = "validation.reason_required"
		case errors.Is(err, application.ErrRouteNotFound):
			statusCode = http.StatusNotFound
			code = "not_found.route"
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			statusCode = http.StatusGatewayTimeout
			code = "timeout"
		case errors.As(err, &syntaxErr):
			statusCode = http.StatusBadRequest
			code = "validation.json_decode"
		case errors.As(err, &typeErr):
			statusCode = http.StatusBadRequest
			code = "validation.json_type_error"
		case errors.Is(err, io.EOF):
			statusCode = http.StatusBadRequest
			code = "validation.empty_body"
		case errors.Is(err, contracts.ErrContentType):
			statusCode = http.StatusUnsupportedMediaType
			code = "validation.content_type"
		case errors.Is(err, contracts.ErrBodyTooLarge):
			statusCode = http.StatusRequestEntityTooLarge
			code = "validation.body_too_large"
		case errors.Is(err, contracts.ErrEmptyBody):
			statusCode = http.StatusBadRequest
			code = "validation.empty_body"
		case errors.Is(err, contracts.ErrDuplicateKey):
			statusCode = http.StatusBadRequest
			code = "validation.duplicate_key"
		}
	}

	body = contracts.ErrorResponse{
		Error: contracts.ErrorBody{
			Code:    code,
			Message: responseMessage(err, statusCode),
		},
	}
	return statusCode, body
}

func WriteError(w http.ResponseWriter, requestID string, err error) {
	statusCode, body := MapErrorToResponse(err)
	body.RequestID = requestID
	WriteJSON(w, statusCode, body)
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	payload, err := json.Marshal(body)
	if err != nil {
		fallback := contracts.ErrorResponse{
			Error: contracts.ErrorBody{
				Code:    "internal.unknown",
				Message: internalErrorMessage,
			},
		}
		payload, _ = json.Marshal(fallback)
		status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

func responseMessage(err error, statusCode int) string {
	if statusCode >= http.StatusInternalServerError {
		return internalErrorMessage
	}
	if err == nil {
		return internalErrorMessage
	}
	return err.Error()
}
