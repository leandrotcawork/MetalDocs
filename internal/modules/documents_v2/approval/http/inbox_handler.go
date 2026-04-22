package approvalhttp

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
)

func (h *Handler) InboxHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	actorID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	areaCode := strings.TrimSpace(r.URL.Query().Get("area_code"))

	limit, err := parseInboxLimit(r.URL.Query().Get("limit"))
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	offset, err := parseInboxOffset(r.URL.Query().Get("offset"))
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	if h.readSvc == nil {
		WriteError(w, reqID, errors.New("read service not configured"))
		return
	}

	items, err := h.readSvc.ListPendingForActor(r.Context(), h.db, tenantID, actorID, areaCode, limit, offset)
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	respItems := make([]contracts.InboxItem, 0, len(items))
	for i := range items {
		respItems = append(respItems, mapInboxItem(items[i]))
	}

	WriteJSON(w, http.StatusOK, contracts.InboxResponse{
		Items:   respItems,
		HasMore: false,
	})
}

func parseInboxLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 25, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("limit must be an integer")
	}
	if v <= 0 || v > 100 {
		return 0, fmt.Errorf("limit must be between 1 and 100")
	}
	return v, nil
}

func parseInboxOffset(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("offset must be an integer")
	}
	if v < 0 {
		return 0, fmt.Errorf("offset must be >= 0")
	}
	return v, nil
}

func mapInboxItem(inst domain.Instance) contracts.InboxItem {
	item := contracts.InboxItem{
		InstanceID:  inst.ID,
		DocumentID:  inst.DocumentID,
		SubmittedBy: inst.SubmittedBy,
		CreatedAt:   inst.SubmittedAt.UTC().Format(time.RFC3339),
	}

	active := inst.Active()
	if active != nil {
		item.StageID = active.ID
		item.StageName = active.NameSnapshot
		item.AreaCode = active.AreaCodeSnapshot
	}

	return item
}
