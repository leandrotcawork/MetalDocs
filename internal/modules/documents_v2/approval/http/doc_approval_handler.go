package approvalhttp

import (
	"errors"
	"net/http"
	"strings"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
	"metaldocs/internal/modules/documents_v2/approval/repository"
)

// GetInstanceByDocumentHandler handles GET /api/v2/documents/{id}/approval-instance.
// It looks up the active approval instance for the document and returns it.
func (h *Handler) GetInstanceByDocumentHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	docID := r.PathValue("id")

	if h.readSvc == nil {
		WriteError(w, reqID, errors.New("read service not configured"))
		return
	}

	inst, err := h.readSvc.LoadActiveInstanceByDocument(r.Context(), h.db, tenantID, docID)
	if err != nil {
		if errors.Is(err, repository.ErrNoActiveInstance) {
			WriteError(w, reqID, repository.ErrNoActiveInstance)
			return
		}
		WriteError(w, reqID, err)
		return
	}

	resp := mapInstanceResponse(inst)
	w.Header().Set("ETag", "\"v1\"")
	WriteJSON(w, http.StatusOK, resp)
}

// docSignoffRequest accepts the frontend's field name ("password" instead of "password_token").
type docSignoffRequest struct {
	Decision    string `json:"decision"`
	Reason      string `json:"reason"`
	Password    string `json:"password"`
	ContentHash string `json:"content_hash"`
}

// SignoffByDocumentHandler handles POST /api/v2/documents/{id}/signoff.
// It finds the active instance+stage for the document and records the signoff.
func (h *Handler) SignoffByDocumentHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	actorID := actorIDFromRequest(r)
	docID := r.PathValue("id")
	idempKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))

	if idempKey == "" {
		WriteError(w, reqID, ErrIdempotencyRequired)
		return
	}
	if h.decisionSvc == nil || h.readSvc == nil {
		WriteError(w, reqID, errors.New("services not configured"))
		return
	}

	var body docSignoffRequest
	if err := contracts.Decode(r, &body); err != nil {
		WriteError(w, reqID, err)
		return
	}
	if body.Decision != "approve" && body.Decision != "reject" {
		WriteError(w, reqID, errors.New("decision must be one of: approve, reject"))
		return
	}
	if body.Decision == "reject" && strings.TrimSpace(body.Reason) == "" {
		WriteError(w, reqID, errors.New("reason is required for reject"))
		return
	}
	if strings.TrimSpace(body.Password) == "" {
		WriteError(w, reqID, errors.New("password is required"))
		return
	}

	inst, err := h.readSvc.LoadActiveInstanceByDocument(r.Context(), h.db, tenantID, docID)
	if err != nil {
		if errors.Is(err, repository.ErrNoActiveInstance) {
			WriteError(w, reqID, repository.ErrNoActiveInstance)
			return
		}
		WriteError(w, reqID, err)
		return
	}

	activeStage := inst.Active()
	if activeStage == nil {
		WriteError(w, reqID, errors.New("no active stage in this approval instance"))
		return
	}

	result, err := h.decisionSvc.RecordSignoff(r.Context(), h.db, application.SignoffRequest{
		TenantID:         tenantID,
		InstanceID:       inst.ID,
		StageInstanceID:  activeStage.ID,
		ActorUserID:      actorID,
		Decision:         body.Decision,
		Comment:          body.Reason,
		SignatureMethod:  "password_reauth",
		SignaturePayload: map[string]any{"password_token": body.Password},
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

// CancelByDocumentHandler handles POST /api/v2/documents/{id}/cancel.
// It finds the active instance for the document and cancels it.
func (h *Handler) CancelByDocumentHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	actorID := actorIDFromRequest(r)
	docID := r.PathValue("id")
	idempKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))

	if idempKey == "" {
		WriteError(w, reqID, ErrIdempotencyRequired)
		return
	}
	if h.readSvc == nil {
		WriteError(w, reqID, errors.New("read service not configured"))
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

	inst, err := h.readSvc.LoadActiveInstanceByDocument(r.Context(), h.db, tenantID, docID)
	if err != nil {
		if errors.Is(err, repository.ErrNoActiveInstance) {
			WriteError(w, reqID, repository.ErrNoActiveInstance)
			return
		}
		WriteError(w, reqID, err)
		return
	}

	result, err := cancelInstance(h, r.Context(), h.db, application.CancelInput{
		TenantID:                tenantID,
		InstanceID:              inst.ID,
		ExpectedRevisionVersion: 0,
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
