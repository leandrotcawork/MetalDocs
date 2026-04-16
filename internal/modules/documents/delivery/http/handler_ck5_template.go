package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/authn"
)

type ck5DraftResponse struct {
	HTML string `json:"html"`
}

// handleGetCK5Draft handles GET /api/v1/templates/{key}/ck5-draft
//
// Query params:
//
//	?mode=author  (default) - returns live draft contentHtml
//	?mode=fill    - returns published_html if the draft has DraftStatus=published,
//	  otherwise falls back to live contentHtml
//
// Auth: 401 if not authenticated, 404 if template not found.
func (h *Handler) handleGetCK5Draft(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	draft, err := h.service.GetTemplateDraft(r.Context(), key)
	if err != nil {
		if errors.Is(err, domain.ErrTemplateDraftNotFound) {
			writeAPIError(w, http.StatusNotFound, "TEMPLATE_NOT_FOUND", "Template draft not found", traceID)
		} else {
			h.writeDomainError(w, err, traceID)
		}
		return
	}

	mode := strings.TrimSpace(r.URL.Query().Get("mode"))

	// Fill mode: return frozen published_html when the draft is published.
	if mode == "fill" && draft.DraftStatus == domain.TemplateStatusPublished && draft.PublishedHTML != nil {
		writeJSON(w, http.StatusOK, ck5DraftResponse{HTML: *draft.PublishedHTML})
		return
	}

	// Author mode (default): return live contentHtml from BlocksJSON.
	html := extractCK5ContentHtmlFromJSON(draft.BlocksJSON)
	writeJSON(w, http.StatusOK, ck5DraftResponse{HTML: html})
}

// extractCK5ContentHtmlFromJSON reads _ck5.contentHtml from blocks_json bytes.
// Returns empty string on any parse error or if the key is absent.
func extractCK5ContentHtmlFromJSON(blocksJSON json.RawMessage) string {
	if len(blocksJSON) == 0 {
		return ""
	}
	var wrapper struct {
		CK5 *struct {
			ContentHTML string `json:"contentHtml"`
		} `json:"_ck5"`
	}
	if err := json.Unmarshal(blocksJSON, &wrapper); err != nil || wrapper.CK5 == nil {
		return ""
	}
	return wrapper.CK5.ContentHTML
}
