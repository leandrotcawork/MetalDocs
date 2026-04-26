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

var cancelInstance = func(h *Handler, ctx context.Context, db *sql.DB, req application.CancelInput) (application.CancelResult, error) {
	if h.services == nil || h.services.Cancel == nil {
		return application.CancelResult{}, errors.New("cancel service not configured")
	}
	return h.services.Cancel.CancelInstance(ctx, db, req)
}

func (h *Handler) CancelHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	tenantID := tenantIDFromReq(r)
	actorID := actorIDFromRequest(r)
	instanceID := r.PathValue("instance_id")

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

	var body contracts.CancelRequest
	if err := contracts.Decode(r, &body); err != nil {
		WriteError(w, reqID, err)
		return
	}
	if err := body.Validate(); err != nil {
		WriteError(w, reqID, err)
		return
	}

	result, err := cancelInstance(h, r.Context(), h.db, application.CancelInput{
		TenantID:                tenantID,
		InstanceID:              instanceID,
		ExpectedRevisionVersion: expectedRevisionVersion,
		ActorUserID:             actorID,
		Reason:                  body.Reason,
	})
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	WriteJSON(w, http.StatusOK, contracts.CancelResponse{
		DocumentID: result.DocumentID,
	})
}
