package documentshttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	v2domain "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/iam/authz"
)

type FillInService interface {
	SetPlaceholderValue(ctx context.Context, tenantID, actorID, revisionID, placeholderID, value string) error
	SetZoneContent(ctx context.Context, tenantID, actorID, revisionID, zoneID, ooxml string) error
}

type FillInHandler struct {
	service FillInService
}

func NewFillInHandler(service FillInService) *FillInHandler {
	return &FillInHandler{service: service}
}

func (h *FillInHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/v2/documents/{id}/placeholders/{pid}", h.PutPlaceholderValue)
	mux.HandleFunc("PUT /api/v2/documents/{id}/zones/{zid}", h.PutZoneContent)
}

func (h *FillInHandler) PutPlaceholderValue(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Value string `json:"value"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeFillInError(w, requestID(r), err)
		return
	}

	err := h.service.SetPlaceholderValue(r.Context(),
		strings.TrimSpace(r.Header.Get("X-Tenant-ID")),
		strings.TrimSpace(r.Header.Get("X-User-ID")),
		r.PathValue("id"),
		r.PathValue("pid"),
		body.Value,
	)
	if err != nil {
		writeFillInError(w, requestID(r), err)
		return
	}

	writeFillInJSON(w, http.StatusOK, map[string]any{
		"placeholder_id": r.PathValue("pid"),
		"updated_at":     time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *FillInHandler) PutZoneContent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ContentOOXML string `json:"content_ooxml"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeFillInError(w, requestID(r), err)
		return
	}

	err := h.service.SetZoneContent(r.Context(),
		strings.TrimSpace(r.Header.Get("X-Tenant-ID")),
		strings.TrimSpace(r.Header.Get("X-User-ID")),
		r.PathValue("id"),
		r.PathValue("zid"),
		body.ContentOOXML,
	)
	if err != nil {
		writeFillInError(w, requestID(r), err)
		return
	}

	writeFillInJSON(w, http.StatusOK, map[string]any{
		"zone_id":    r.PathValue("zid"),
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

type fillInErrorResponse struct {
	Error     fillInErrorBody `json:"error"`
	RequestID string          `json:"request_id"`
}

type fillInErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func mapFillInError(err error) (int, fillInErrorResponse) {
	status := http.StatusInternalServerError
	code := "internal.unknown"

	switch {
	case errors.As(err, &authz.ErrCapabilityDenied{}):
		status = http.StatusForbidden
		code = "authz.capability_denied"
	case errors.As(err, &notChoicePlaceholderError{}):
		status = http.StatusBadRequest
		code = "not_a_choice_placeholder"
	case errors.Is(err, v2domain.ErrNotFound):
		status = http.StatusNotFound
		code = "not_found.revision"
	case errors.Is(err, v2domain.ErrInvalidStateTransition):
		status = http.StatusConflict
		code = "state.revision_not_draft"
	case errors.Is(err, v2domain.ErrValidationFailed):
		status = http.StatusUnprocessableEntity
		code = "validation.failed"
	case errors.Is(err, io.EOF):
		status = http.StatusBadRequest
		code = "validation.empty_body"
	case looksLikeDecodeError(err):
		status = http.StatusBadRequest
		code = "validation.json_decode"
	}

	return status, fillInErrorResponse{
		Error: fillInErrorBody{
			Code:    code,
			Message: errorMessage(err, status),
		},
	}
}

func writeFillInError(w http.ResponseWriter, reqID string, err error) {
	status, body := mapFillInError(err)
	body.RequestID = reqID
	writeFillInJSON(w, status, body)
}

func writeFillInJSON(w http.ResponseWriter, status int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		status = http.StatusInternalServerError
		data = []byte(`{"error":{"code":"internal.unknown","message":"internal error"}}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func decodeJSON(r *http.Request, out any) error {
	if strings.TrimSpace(r.Header.Get("Content-Type")) != "" &&
		!strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		return fmt.Errorf("content-type must be application/json")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func errorMessage(err error, status int) string {
	if status >= http.StatusInternalServerError {
		return "internal error"
	}
	return err.Error()
}

func looksLikeDecodeError(err error) bool {
	if err == nil {
		return false
	}
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	return errors.As(err, &syntaxErr) || errors.As(err, &typeErr) || errors.Is(err, io.ErrUnexpectedEOF)
}

func requestID(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get("X-Request-ID")); id != "" {
		return id
	}
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
