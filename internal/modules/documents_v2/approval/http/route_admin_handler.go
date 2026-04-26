package approvalhttp

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

func (h *Handler) ListRoutesHandler(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	tenantID := tenantIDFromReq(r)

	if h.db == nil {
		WriteError(w, reqID, fmt.Errorf("database not configured"))
		return
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT id, name, tenant_id::text, profile_code, active, created_at
		  FROM approval_routes
		 WHERE tenant_id = $1::uuid
		 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		WriteError(w, reqID, err)
		return
	}
	defer rows.Close()

	routeMap := make(map[string]*contracts.ListRouteItem)
	var routeOrder []string
	var routeIDs []string

	for rows.Next() {
		var (
			id, name, tid, profileCode string
			active                     bool
			createdAt                  time.Time
		)
		if err := rows.Scan(&id, &name, &tid, &profileCode, &active, &createdAt); err != nil {
			WriteError(w, reqID, err)
			return
		}

		ts := createdAt.UTC().Format(time.RFC3339)
		routeMap[id] = &contracts.ListRouteItem{
			ID:          id,
			Name:        name,
			TenantID:    tid,
			ProfileCode: profileCode,
			Active:      active,
			Stages:      []contracts.ListStageItem{},
			CreatedAt:   ts,
			UpdatedAt:   ts,
		}
		routeOrder = append(routeOrder, id)
		routeIDs = append(routeIDs, id)
	}

	if err := rows.Err(); err != nil {
		WriteError(w, reqID, err)
		return
	}

	if len(routeIDs) > 0 {
		placeholders := make([]string, 0, len(routeIDs))
		args := make([]any, 0, len(routeIDs))
		for i, id := range routeIDs {
			placeholders = append(placeholders, "$"+strconv.Itoa(i+1)+"::uuid")
			args = append(args, id)
		}

		stageRows, err := h.db.QueryContext(r.Context(), `
			SELECT route_id, name, required_role, quorum, on_eligibility_drift
			  FROM approval_route_stages
			 WHERE route_id IN (`+strings.Join(placeholders, ", ")+`)
			 ORDER BY route_id, stage_order`,
			args...,
		)
		if err != nil {
			WriteError(w, reqID, err)
			return
		}
		defer stageRows.Close()

		for stageRows.Next() {
			var (
				id                                        string
				stageName, stageRole, quorum, driftPolicy *string
			)
			if err := stageRows.Scan(&id, &stageName, &stageRole, &quorum, &driftPolicy); err != nil {
				WriteError(w, reqID, err)
				return
			}

			if stageName != nil {
				sn := *stageName
				sr := ""
				sq := ""
				sd := ""
				if stageRole != nil {
					sr = *stageRole
				}
				if quorum != nil {
					sq = *quorum
				}
				if driftPolicy != nil {
					sd = *driftPolicy
				}
				routeMap[id].Stages = append(routeMap[id].Stages, contracts.ListStageItem{
					Label:       sn,
					Members:     []string{sr},
					QuorumKind:  sq,
					DriftPolicy: sd,
				})
			}
		}

		if err := stageRows.Err(); err != nil {
			WriteError(w, reqID, err)
			return
		}
	}

	routes := make([]contracts.ListRouteItem, 0, len(routeOrder))
	for _, id := range routeOrder {
		routes = append(routes, *routeMap[id])
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"routes": routes,
		"total":  len(routes),
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
