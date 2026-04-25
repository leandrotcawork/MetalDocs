package approvalhttp

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func (h *Handler) SubmitHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	documentID := r.PathValue("id")
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	actorID := iamdomain.UserIDFromContext(r.Context())
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		WriteError(w, reqID, ErrIdempotencyRequired)
		return
	}

	expectedRevisionVersion, err := parseIfMatch(r.Header.Get("If-Match"))
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	var req contracts.SubmitRequest
	if err := contracts.Decode(r, &req); err != nil {
		WriteError(w, reqID, err)
		return
	}
	if err := req.Validate(); err != nil {
		WriteError(w, reqID, err)
		return
	}

	submitSvc := h.submitSvc
	if submitSvc == nil && h.services != nil {
		submitSvc = h.services.Submit
	}
	if submitSvc == nil {
		WriteError(w, reqID, errors.New("submit service not configured"))
		return
	}

	result, err := submitSvc.SubmitRevisionForReview(r.Context(), h.db, application.SubmitRequest{
		TenantID:        tenantID,
		DocumentID:      documentID,
		RouteID:         req.RouteID,
		SubmittedBy:     actorID,
		ContentFormData: map[string]any{"_content_hash": req.ContentHash},
		RevisionVersion: expectedRevisionVersion,
	})
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	newETag := "\"v" + strconv.Itoa(expectedRevisionVersion+1) + "\""
	w.Header().Set("ETag", newETag)
	WriteJSON(w, http.StatusCreated, contracts.SubmitResponse{
		InstanceID: result.InstanceID,
		WasReplay:  false,
		ETag:       newETag,
	})
}
