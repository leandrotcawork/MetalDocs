package approvalhttp

import (
	"errors"
	"net/http"
	"strings"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
)

var (
	ErrIdempotencyRequired = errors.New("idempotency: Idempotency-Key header required on mutating requests")

	ErrContentHashMismatch = errors.New("approval: content hash mismatch")
)

func (h *Handler) SignoffHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	tenantID := tenantIDFromReq(r)
	actorID := actorIDFromRequest(r)
	instanceID := r.PathValue("instance_id")
	stageID := r.PathValue("stage_id")
	idempKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))

	if idempKey == "" {
		WriteError(w, reqID, ErrIdempotencyRequired)
		return
	}
	if _, err := parseIfMatch(r.Header.Get("If-Match")); err != nil {
		WriteError(w, reqID, err)
		return
	}
	if h.decisionSvc == nil {
		WriteError(w, reqID, errors.New("decision service not configured"))
		return
	}

	var body contracts.SignoffRequest
	if err := contracts.Decode(r, &body); err != nil {
		WriteError(w, reqID, err)
		return
	}
	if err := body.Validate(); err != nil {
		WriteError(w, reqID, err)
		return
	}

	result, err := h.decisionSvc.RecordSignoff(r.Context(), h.db, application.SignoffRequest{
		TenantID:         tenantID,
		InstanceID:       instanceID,
		StageInstanceID:  stageID,
		ActorUserID:      actorID,
		Decision:         body.Decision,
		Comment:          body.Reason,
		SignatureMethod:  "password_reauth",
		SignaturePayload: map[string]any{"password_token": body.PasswordToken},
		ContentFormData:  map[string]any{"_content_hash": body.ContentHash},
	})
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	WriteJSON(w, http.StatusOK, contracts.SignoffResponse{
		SignoffID: "",
		WasReplay: false,
		Outcome:   signoffOutcome(result),
	})
}

func signoffOutcome(result application.SignoffResult) string {
	switch {
	case result.InstanceApproved:
		return "approved"
	case result.InstanceRejected:
		return "rejected"
	case result.StageCompleted:
		return "stage_completed"
	default:
		return "pending"
	}
}
