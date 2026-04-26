package http

import (
	"database/sql"
	"net/http"

	"metaldocs/internal/modules/registry/application"
)

type Handler struct {
	svc *application.RegistryService
	db  *sql.DB
}

func NewHandler(svc *application.RegistryService, db *sql.DB) *Handler {
	return &Handler{svc: svc, db: db}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/controlled-documents", h.listDocs)
	mux.HandleFunc("POST /api/v2/controlled-documents", h.createDoc)
	mux.HandleFunc("GET /api/v2/controlled-documents/{id}", h.getDoc)
	mux.HandleFunc("GET /api/v2/controlled-documents/{id}/active-document", h.getActiveDocument)
	mux.HandleFunc("PUT /api/v2/controlled-documents/{id}/obsolete", h.obsoleteDoc)
	mux.HandleFunc("PUT /api/v2/controlled-documents/{id}/supersede", h.supersedeDoc)
}
