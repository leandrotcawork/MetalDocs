package documentshttp

import (
	"context"
	"errors"
	"net/http"

	v2domain "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/iam/authz"
)

// ErrPDFPending distinguishes "approved but PDF not yet generated" from plain
// not-found so the handler can surface a specific `pdf_pending` error code.
var ErrPDFPending = errors.New("pdf_pending")

type ViewResult struct {
	SignedURL string
}

type ViewService interface {
	GetViewURL(ctx context.Context, tenantID, actorID, docID string) (ViewResult, error)
}

type ViewHandler struct {
	svc ViewService
}

func NewViewHandler(svc ViewService) *ViewHandler {
	return &ViewHandler{svc: svc}
}

func (h *ViewHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/documents/{id}/view", h.HandleView)
}

func (h *ViewHandler) HandleView(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.GetViewURL(r.Context(),
		tenantID(r),
		actorID(r),
		r.PathValue("id"),
	)
	if err != nil {
		writeViewError(w, err)
		return
	}
	writeFillInJSON(w, http.StatusOK, map[string]any{
		"signed_url": result.SignedURL,
	})
}

func writeViewError(w http.ResponseWriter, err error) {
	switch {
	case errors.As(err, &authz.ErrCapabilityDenied{}):
		writeFillInJSON(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
	case errors.Is(err, ErrPDFPending):
		writeFillInJSON(w, http.StatusNotFound, map[string]any{"error": "pdf_pending"})
	case errors.Is(err, v2domain.ErrNotFound):
		writeFillInJSON(w, http.StatusNotFound, map[string]any{"error": "not_found"})
	default:
		writeFillInJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal"})
	}
}
