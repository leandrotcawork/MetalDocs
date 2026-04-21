package http

import (
	"net/http"

	"metaldocs/internal/modules/taxonomy/application"
)

type Handler struct {
	profiles *application.ProfileService
	areas    *application.AreaService
}

func NewHandler(profiles *application.ProfileService, areas *application.AreaService) *Handler {
	return &Handler{
		profiles: profiles,
		areas:    areas,
	}
}

func (h *Handler) RegisterRoutes(_ *http.ServeMux) {}
