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

var markObsolete = func(h *Handler, ctx context.Context, db *sql.DB, req application.MarkObsoleteRequest) (application.MarkObsoleteResult, error) {
	if h.services == nil || h.services.Obsolete == nil {
		return application.MarkObsoleteResult{}, errors.New("obsolete service not configured")
	}
	return h.services.Obsolete.MarkObsolete(ctx, db, req)
}

func (h *Handler) ObsoleteHandler(w http.ResponseWriter, r *http.Request) {
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

	var body contracts.ObsoleteRequest
	if err := contracts.Decode(r, &body); err != nil {
		WriteError(w, reqID, err)
		return
	}
	if err := body.Validate(); err != nil {
		WriteError(w, reqID, err)
		return
	}

	_, err = markObsolete(h, r.Context(), h.db, application.MarkObsoleteRequest{
		TenantID:        tenantID,
		DocumentID:      documentID,
		MarkedBy:        actorID,
		RevisionVersion: expectedRevisionVersion,
		Reason:          body.Reason,
	})
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	WriteJSON(w, http.StatusOK, contracts.ObsoleteResponse{
		DocumentID: documentID,
	})
}
