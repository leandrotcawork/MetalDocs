package documentshttp

import (
	"context"
	"net/http"
	"strings"

	templatesdomain "metaldocs/internal/modules/templates_v2/domain"
)

type placeholderOptionsSchemaReader interface {
	LoadPlaceholderSchema(ctx context.Context, tenantID, revisionID string) ([]templatesdomain.Placeholder, error)
}

// UserOptionView is a local view model for user placeholder options.
type UserOptionView struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
}

type placeholderOptionsIAMReader interface {
	ListUserOptions(ctx context.Context, tenantID string) ([]UserOptionView, error)
}

type PlaceholderOptionsHandler struct {
	schema placeholderOptionsSchemaReader
	iam    placeholderOptionsIAMReader
}

func NewPlaceholderOptionsHandler(schema placeholderOptionsSchemaReader, iam placeholderOptionsIAMReader) *PlaceholderOptionsHandler {
	return &PlaceholderOptionsHandler{schema: schema, iam: iam}
}

func (h *PlaceholderOptionsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/documents/{id}/placeholder-options/{pid}", h.HandleGetOptions)
}

func (h *PlaceholderOptionsHandler) HandleGetOptions(w http.ResponseWriter, r *http.Request) {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	revisionID := r.PathValue("id")
	placeholderID := r.PathValue("pid")

	schema, err := h.schema.LoadPlaceholderSchema(r.Context(), tenantID, revisionID)
	if err != nil {
		writeFillInError(w, requestID(r), err)
		return
	}

	var ph *templatesdomain.Placeholder
	for i := range schema {
		if schema[i].ID == placeholderID {
			ph = &schema[i]
			break
		}
	}
	if ph == nil {
		writeFillInError(w, requestID(r), errNotChoicePlaceholder(placeholderID))
		return
	}

	switch ph.Type {
	case templatesdomain.PHSelect:
		writeFillInJSON(w, http.StatusOK, map[string]any{"options": selectOptions(ph.Options)})
	case templatesdomain.PHUser:
		opts, err := h.iam.ListUserOptions(r.Context(), tenantID)
		if err != nil {
			writeFillInError(w, requestID(r), err)
			return
		}
		writeFillInJSON(w, http.StatusOK, map[string]any{"options": opts})
	default:
		writeFillInError(w, requestID(r), errNotChoicePlaceholder(placeholderID))
	}
}

func selectOptions(values []string) []map[string]string {
	out := make([]map[string]string, len(values))
	for i, v := range values {
		out[i] = map[string]string{"value": v, "display_name": v}
	}
	return out
}

type notChoicePlaceholderError struct{ id string }

func (e notChoicePlaceholderError) Error() string {
	return "not_a_choice_placeholder: " + e.id
}

func errNotChoicePlaceholder(id string) error { return notChoicePlaceholderError{id: id} }
