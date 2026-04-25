package documentshttp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	v2dom "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/iam/authz"
	iamdomain "metaldocs/internal/modules/iam/domain"
	"metaldocs/internal/modules/render/fanout"
)

type ReconstructService interface {
	GetReconstruction(ctx context.Context, tenantID, actorID, docID string) (fanout.ReconstructionEntry, error)
}

type ReconstructHandler struct {
	svc ReconstructService
}

func NewReconstructHandler(svc ReconstructService) *ReconstructHandler {
	return &ReconstructHandler{svc: svc}
}

func (h *ReconstructHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v2/documents/{id}/reconstruct", h.HandleReconstruct)
}

func (h *ReconstructHandler) HandleReconstruct(w http.ResponseWriter, r *http.Request) {
	entry, err := h.svc.GetReconstruction(
		r.Context(),
		strings.TrimSpace(r.Header.Get("X-Tenant-ID")),
		iamdomain.UserIDFromContext(r.Context()),
		r.PathValue("id"),
	)
	if err != nil {
		writeReconstructError(w, err)
		return
	}

	writeFillInJSON(w, http.StatusOK, entry)
}

func writeReconstructError(w http.ResponseWriter, err error) {
	switch {
	case errors.As(err, &authz.ErrCapabilityDenied{}):
		writeFillInJSON(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
	case errors.Is(err, v2dom.ErrNotFound):
		writeFillInJSON(w, http.StatusNotFound, map[string]any{"error": "not_found"})
	default:
		writeFillInJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal"})
	}
}
