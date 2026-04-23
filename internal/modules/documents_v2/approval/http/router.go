package approvalhttp

import "net/http"

// RegisterRoutes wires all approval routes onto mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Mutators — documents
	mux.HandleFunc("POST /api/v2/documents/{id}/submit", h.SubmitHandler)
	mux.HandleFunc("POST /api/v2/approval/instances/{instance_id}/stages/{stage_id}/signoffs", h.SignoffHandler)
	mux.HandleFunc("POST /api/v2/documents/{id}/publish", h.PublishHandler)
	mux.HandleFunc("POST /api/v2/documents/{id}/schedule-publish", h.SchedulePublishHandler)
	mux.HandleFunc("POST /api/v2/documents/{id}/supersede", h.SupersedeHandler)
	mux.HandleFunc("POST /api/v2/documents/{id}/obsolete", h.ObsoleteHandler)
	mux.HandleFunc("POST /api/v2/approval/instances/{instance_id}/cancel", h.CancelHandler)

	// Read
	mux.HandleFunc("GET /api/v2/approval/instances/{instance_id}", h.GetInstanceHandler)
	mux.HandleFunc("GET /api/v2/documents/{id}/approval-instance", h.GetInstanceByDocumentHandler)
	mux.HandleFunc("GET /api/v2/approval/inbox", h.InboxHandler)

	// Doc-centric mutation shims
	mux.HandleFunc("POST /api/v2/documents/{id}/signoff", h.SignoffByDocumentHandler)
	mux.HandleFunc("POST /api/v2/documents/{id}/cancel", h.CancelByDocumentHandler)

	// Route admin
	mux.HandleFunc("POST /api/v2/approval/routes", h.CreateRouteHandler)
	mux.HandleFunc("PUT /api/v2/approval/routes/{id}", h.UpdateRouteHandler)
	mux.HandleFunc("DELETE /api/v2/approval/routes/{id}", h.DeactivateRouteHandler)
	mux.HandleFunc("GET /api/v2/approval/routes", h.ListRoutesHandler)
}
