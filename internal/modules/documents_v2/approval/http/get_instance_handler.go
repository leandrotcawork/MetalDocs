package approvalhttp

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
	"metaldocs/internal/modules/documents_v2/approval/repository"
)

func (h *Handler) GetInstanceHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	actorID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	instanceID := r.PathValue("instance_id")

	if h.readSvc == nil {
		WriteError(w, reqID, errors.New("read service not configured"))
		return
	}

	inst, err := h.readSvc.LoadInstance(r.Context(), h.db, tenantID, actorID, instanceID)
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

func mapInstanceResponse(inst *domain.Instance) contracts.InstanceResponse {
	var completedAt *string
	if inst.CompletedAt != nil {
		v := inst.CompletedAt.UTC().Format(time.RFC3339)
		completedAt = &v
	}

	return contracts.InstanceResponse{
		InstanceID:  inst.ID,
		DocumentID:  inst.DocumentID,
		TenantID:    inst.TenantID,
		Status:      string(inst.Status),
		SubmittedBy: inst.SubmittedBy,
		CreatedAt:   inst.SubmittedAt.UTC().Format(time.RFC3339),
		CompletedAt: completedAt,
		Stages:      nil, // deferred - no stage list in domain.Instance yet
		ETag:        "\"v1\"",
	}
}
