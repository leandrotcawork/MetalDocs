package http

import (
	"net/http"

	"metaldocs/internal/modules/registry/application"
)

type Handler struct {
	svc *application.RegistryService
}

func NewHandler(svc *application.RegistryService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(_ *http.ServeMux) {
	// Phase 3 HTTP routes intentionally deferred.
}
