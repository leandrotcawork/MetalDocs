package approvalhttp

import (
	"errors"
	"net/http"
	"strings"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
)

func (h *Handler) CreateRouteHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	tenantID := tenantIDFromReq(r)
	actorID := actorIDFromRequest(r)
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		WriteError(w, reqID, ErrIdempotencyRequired)
		return
	}

	var req contracts.CreateRouteRequest
	if err := contracts.Decode(r, &req); err != nil {
		WriteError(w, reqID, err)
		return
	}
	if err := req.Validate(); err != nil {
		WriteError(w, reqID, err)
		return
	}

	routeAdminSvc := h.routeAdmin
	if routeAdminSvc == nil {
		WriteError(w, reqID, errors.New("route admin service not configured"))
		return
	}

	result, err := routeAdminSvc.Create(r.Context(), h.db, application.CreateRouteInput{
		TenantID:    tenantID,
		ProfileCode: req.ProfileCode,
		Name:        req.Name,
		ActorUserID: actorID,
		Stages:      mapStageRequests(req.Stages),
	})
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	WriteJSON(w, http.StatusCreated, contracts.RouteResponse{
		RouteID: result.RouteID,
	})
}

func (h *Handler) UpdateRouteHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	tenantID := tenantIDFromReq(r)
	actorID := actorIDFromRequest(r)
	routeID := r.PathValue("id")
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		WriteError(w, reqID, ErrIdempotencyRequired)
		return
	}
	if _, err := parseIfMatch(r.Header.Get("If-Match")); err != nil {
		WriteError(w, reqID, err)
		return
	}

	var req contracts.UpdateRouteRequest
	if err := contracts.Decode(r, &req); err != nil {
		WriteError(w, reqID, err)
		return
	}
	if err := req.Validate(); err != nil {
		WriteError(w, reqID, err)
		return
	}

	routeAdminSvc := h.routeAdmin
	if routeAdminSvc == nil {
		WriteError(w, reqID, errors.New("route admin service not configured"))
		return
	}

	result, err := routeAdminSvc.Update(r.Context(), h.db, application.UpdateRouteInput{
		TenantID:    tenantID,
		RouteID:     routeID,
		Name:        req.Name,
		ActorUserID: actorID,
		Stages:      mapStageRequests(req.Stages),
	})
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	WriteJSON(w, http.StatusOK, contracts.RouteResponse{
		RouteID:    result.RouteID,
		NewVersion: result.NewVersion,
	})
}

func (h *Handler) DeactivateRouteHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	tenantID := tenantIDFromReq(r)
	actorID := actorIDFromRequest(r)
	routeID := r.PathValue("id")
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		WriteError(w, reqID, ErrIdempotencyRequired)
		return
	}
	if _, err := parseIfMatch(r.Header.Get("If-Match")); err != nil {
		WriteError(w, reqID, err)
		return
	}

	routeAdminSvc := h.routeAdmin
	if routeAdminSvc == nil {
		WriteError(w, reqID, errors.New("route admin service not configured"))
		return
	}

	result, err := routeAdminSvc.Deactivate(r.Context(), h.db, application.DeactivateRouteInput{
		TenantID:    tenantID,
		RouteID:     routeID,
		ActorUserID: actorID,
	})
	if err != nil {
		WriteError(w, reqID, err)
		return
	}

	WriteJSON(w, http.StatusOK, contracts.RouteResponse{
		RouteID: result.RouteID,
	})
}

func (h *Handler) ListRoutesHandler(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"routes": []contracts.RouteResponse{},
		"total":  0,
	})
}

func mapStageRequests(stages []contracts.StageRequest) []domain.Stage {
	out := make([]domain.Stage, 0, len(stages))
	for _, s := range stages {
		out = append(out, domain.Stage{
			Order:              s.Order,
			Name:               s.Name,
			RequiredRole:       s.RequiredRole,
			RequiredCapability: s.RequiredCapability,
			AreaCode:           s.AreaCode,
			Quorum:             domain.QuorumPolicy(s.Quorum),
			QuorumM:            s.QuorumM,
			OnEligibilityDrift: domain.DriftPolicy(s.DriftPolicy),
		})
	}
	return out
}
