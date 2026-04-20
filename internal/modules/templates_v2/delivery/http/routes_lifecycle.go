package http

import (
	"net/http"
	"strconv"
	"strings"

	iamdomain "metaldocs/internal/modules/iam/domain"
	"metaldocs/internal/modules/templates_v2/application"
)

func (h *Handler) submitForReview(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromReq(r)
	actorID := userIDFromReq(r)
	templateID := r.PathValue("id")
	versionNum, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_version_number", "version must be an integer")
		return
	}

	if err := h.authz(r, tenantID, "*", "template.edit"); err != nil {
		writeMappedErr(w, err)
		return
	}

	v, err := h.svc.SubmitForReview(r.Context(), application.SubmitForReviewCmd{
		TenantID:      tenantID,
		ActorUserID:   actorID,
		TemplateID:    templateID,
		VersionNumber: versionNum,
	})
	if err != nil {
		writeMappedErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"version": toVersionResponse(v),
		},
	})
}

func (h *Handler) review(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromReq(r)
	actorID := userIDFromReq(r)
	templateID := r.PathValue("id")
	versionNum, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_version_number", "version must be an integer")
		return
	}

	if err := h.authz(r, tenantID, "*", "template.review"); err != nil {
		writeMappedErr(w, err)
		return
	}

	var req struct {
		Accept bool   `json:"accept"`
		Reason string `json:"reason"`
	}
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	v, err := h.svc.Review(r.Context(), application.ReviewCmd{
		TenantID:      tenantID,
		ActorUserID:   actorID,
		ActorRoles:    actorRolesFromReq(r),
		TemplateID:    templateID,
		VersionNumber: versionNum,
		Accept:        req.Accept,
		Reason:        req.Reason,
	})
	if err != nil {
		writeMappedErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"version": toVersionResponse(v),
		},
	})
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromReq(r)
	actorID := userIDFromReq(r)
	templateID := r.PathValue("id")
	versionNum, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_version_number", "version must be an integer")
		return
	}

	if err := h.authz(r, tenantID, "*", "template.approve"); err != nil {
		writeMappedErr(w, err)
		return
	}

	var req struct {
		Accept bool   `json:"accept"`
		Reason string `json:"reason"`
	}
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	v, err := h.svc.Approve(r.Context(), application.ApproveCmd{
		TenantID:      tenantID,
		ActorUserID:   actorID,
		ActorRoles:    actorRolesFromReq(r),
		TemplateID:    templateID,
		VersionNumber: versionNum,
		Accept:        req.Accept,
		Reason:        req.Reason,
	})
	if err != nil {
		writeMappedErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"version": toVersionResponse(v),
		},
	})
}

func (h *Handler) archiveTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromReq(r)
	actorID := userIDFromReq(r)
	templateID := r.PathValue("id")

	if err := h.authz(r, tenantID, "*", "template.archive"); err != nil {
		writeMappedErr(w, err)
		return
	}

	tpl, err := h.svc.ArchiveTemplate(r.Context(), application.ArchiveCmd{
		TenantID:    tenantID,
		ActorUserID: actorID,
		TemplateID:  templateID,
	})
	if err != nil {
		writeMappedErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"template": toTemplateResponse(tpl),
		},
	})
}

func (h *Handler) upsertApprovalConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromReq(r)
	actorID := userIDFromReq(r)
	templateID := r.PathValue("id")

	if err := h.authz(r, tenantID, "*", "template.admin"); err != nil {
		writeMappedErr(w, err)
		return
	}

	var req struct {
		ReviewerRole *string `json:"reviewer_role"`
		ApproverRole string  `json:"approver_role"`
	}
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	cfg, err := h.svc.UpsertApprovalConfig(r.Context(), application.UpsertApprovalConfigCmd{
		TenantID:     tenantID,
		ActorUserID:  actorID,
		TemplateID:   templateID,
		ActorRoles:   actorRolesFromReq(r),
		ReviewerRole: req.ReviewerRole,
		ApproverRole: req.ApproverRole,
	})
	if err != nil {
		writeMappedErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"approval_config": map[string]any{
				"template_id":   cfg.TemplateID,
				"reviewer_role": cfg.ReviewerRole,
				"approver_role": cfg.ApproverRole,
			},
		},
	})
}

func actorRolesFromReq(r *http.Request) []string {
	// Prefer IAM-middleware roles from context (set by iamMiddleware from DB).
	if ctxRoles := iamdomain.RolesFromContext(r.Context()); len(ctxRoles) > 0 {
		out := make([]string, len(ctxRoles))
		for i, role := range ctxRoles {
			out[i] = string(role)
		}
		return out
	}
	// Fallback: explicit header (used in tests / service-to-service calls).
	vals := r.Header.Values("X-Actor-Roles")
	if len(vals) == 0 {
		return nil
	}
	roles := make([]string, 0, len(vals))
	for _, v := range vals {
		for _, role := range strings.Split(v, ",") {
			role = strings.TrimSpace(role)
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	return roles
}
