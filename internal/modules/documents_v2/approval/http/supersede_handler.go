package approvalhttp

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
)

var publishSuperseding = func(h *Handler, ctx context.Context, db *sql.DB, req application.SupersedeRequest) (application.SupersedeResult, error) {
	if h.services == nil || h.services.Supersede == nil {
		return application.SupersedeResult{}, errors.New("supersede service not configured")
	}
	return h.services.Supersede.PublishSuperseding(ctx, db, req)
}

func (h *Handler) SupersedeHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	tenantID := tenantIDFromReq(r)
	actorID := actorIDFromRequest(r)
	documentID := r.PathValue("id")

	idempKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempKey == "" {
		WriteError(w, reqID, ErrIdempotencyRequired)
		return
	}

	expectedRevisionVersion, err := parseIfMatch(r.Header.Get("If-Match"))
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	var body contracts.SupersedeRequest
	if err := contracts.Decode(r, &body); err != nil {
		WriteError(w, reqID, err)
		return
	}
	if err := body.Validate(); err != nil {
		WriteError(w, reqID, err)
		return
	}

	var priorRevisionVersion int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT revision_version FROM documents WHERE id = $1 AND tenant_id = $2`,
		body.SupersededDocumentID, tenantID,
	).Scan(&priorRevisionVersion); err != nil {
		WriteError(w, reqID, err)
		return
	}

	_, err = publishSuperseding(h, r.Context(), h.db, application.SupersedeRequest{
		TenantID:             tenantID,
		NewDocumentID:        documentID,
		PriorDocumentID:      body.SupersededDocumentID,
		SupersededBy:         actorID,
		NewRevisionVersion:   expectedRevisionVersion,
		PriorRevisionVersion: priorRevisionVersion,
	})
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	WriteJSON(w, http.StatusOK, contracts.SupersedeResponse{
		DocumentID:   documentID,
		SupersededID: body.SupersededDocumentID,
	})
}
