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

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/controlled-documents", h.listDocs)
	mux.HandleFunc("POST /api/v2/controlled-documents", h.createDoc)
	mux.HandleFunc("GET /api/v2/controlled-documents/{id}", h.getDoc)
	mux.HandleFunc("PUT /api/v2/controlled-documents/{id}/obsolete", h.obsoleteDoc)
	mux.HandleFunc("PUT /api/v2/controlled-documents/{id}/supersede", h.supersedeDoc)
}
