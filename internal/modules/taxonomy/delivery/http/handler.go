package http

import (
	"context"
	"net/http"

	"metaldocs/internal/modules/taxonomy/application"
	"metaldocs/internal/modules/taxonomy/domain"
)

type profileService interface {
	List(ctx context.Context, tenantID string, includeArchived bool) ([]domain.DocumentProfile, error)
	Get(ctx context.Context, tenantID, code string) (*domain.DocumentProfile, error)
	Create(ctx context.Context, p *domain.DocumentProfile) error
	Update(ctx context.Context, p *domain.DocumentProfile) error
	SetDefaultTemplate(ctx context.Context, tenantID, profileCode, templateVersionID, actorID string) error
	Archive(ctx context.Context, tenantID, profileCode, actorID string) error
}

type areaService interface {
	List(ctx context.Context, tenantID string, includeArchived bool) ([]domain.ProcessArea, error)
	Get(ctx context.Context, tenantID, code string) (*domain.ProcessArea, error)
	Create(ctx context.Context, a *domain.ProcessArea) error
	Update(ctx context.Context, a *domain.ProcessArea) error
	Archive(ctx context.Context, tenantID, areaCode, actorID string) error
}

type Handler struct {
	profiles profileService
	areas    areaService
}

func NewHandler(profiles *application.ProfileService, areas *application.AreaService) *Handler {
	return &Handler{
		profiles: profiles,
		areas:    areas,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/taxonomy/profiles", h.listProfiles)
	mux.HandleFunc("POST /api/v2/taxonomy/profiles", h.createProfile)
	mux.HandleFunc("GET /api/v2/taxonomy/profiles/{code}", h.getProfile)
	mux.HandleFunc("PATCH /api/v2/taxonomy/profiles/{code}", h.updateProfile)
	mux.HandleFunc("DELETE /api/v2/taxonomy/profiles/{code}", h.archiveProfile)
	mux.HandleFunc("PUT /api/v2/taxonomy/profiles/{code}/default-template", h.setDefaultTemplate)

	mux.HandleFunc("GET /api/v2/taxonomy/areas", h.listAreas)
	mux.HandleFunc("POST /api/v2/taxonomy/areas", h.createArea)
	mux.HandleFunc("GET /api/v2/taxonomy/areas/{code}", h.getArea)
	mux.HandleFunc("PUT /api/v2/taxonomy/areas/{code}", h.updateArea)
	mux.HandleFunc("DELETE /api/v2/taxonomy/areas/{code}", h.archiveArea)
}
