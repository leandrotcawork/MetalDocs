package http

import (
	"net/http"
	"time"

	"metaldocs/internal/modules/templates_v2/application"
	"metaldocs/internal/modules/templates_v2/domain"
)

func (h *Handler) createTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromReq(r)
	actorID := userIDFromReq(r)

	if err := h.authz(r, tenantID, "*", "template.create"); err != nil {
		writeMappedErr(w, err)
		return
	}

	var req struct {
		DocTypeCode   string            `json:"doc_type_code"`
		Key           string            `json:"key"`
		Name          string            `json:"name"`
		Description   string            `json:"description"`
		Areas         []string          `json:"areas"`
		Visibility    domain.Visibility `json:"visibility"`
		SpecificAreas []string          `json:"specific_areas"`
		ApproverRole  string            `json:"approver_role"`
		ReviewerRole  *string           `json:"reviewer_role"`
	}
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	res, err := h.svc.CreateTemplate(r.Context(), application.CreateTemplateCmd{
		TenantID:      tenantID,
		ActorUserID:   actorID,
		DocTypeCode:   req.DocTypeCode,
		Key:           req.Key,
		Name:          req.Name,
		Description:   req.Description,
		Areas:         req.Areas,
		Visibility:    req.Visibility,
		SpecificAreas: req.SpecificAreas,
		ApproverRole:  req.ApproverRole,
		ReviewerRole:  req.ReviewerRole,
	})
	if err != nil {
		writeMappedErr(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]any{
			"template": toTemplateResponse(res.Template),
			"version":  toVersionResponse(res.Version),
		},
	})
}

func (h *Handler) createNextVersion(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromReq(r)
	actorID := userIDFromReq(r)
	templateID := r.PathValue("id")

	if err := h.authz(r, tenantID, "*", "template.create"); err != nil {
		writeMappedErr(w, err)
		return
	}

	v, err := h.svc.CreateNextVersion(r.Context(), application.CreateVersionCmd{
		TenantID:    tenantID,
		ActorUserID: actorID,
		TemplateID:  templateID,
	})
	if err != nil {
		writeMappedErr(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]any{
			"version": toVersionResponse(v),
		},
	})
}

func toTemplateResponse(t *domain.Template) map[string]any {
	if t == nil {
		return nil
	}

	return map[string]any{
		"id":                   t.ID,
		"tenant_id":            t.TenantID,
		"doc_type_code":        t.DocTypeCode,
		"key":                  t.Key,
		"name":                 t.Name,
		"description":          t.Description,
		"areas":                t.Areas,
		"visibility":           t.Visibility,
		"specific_areas":       t.SpecificAreas,
		"latest_version":       t.LatestVersion,
		"published_version_id": t.PublishedVersionID,
		"created_by":           t.CreatedBy,
		"created_at":           t.CreatedAt.UTC().Format(time.RFC3339),
		"archived_at":          timePtrRFC3339(t.ArchivedAt),
	}
}

func toVersionResponse(v *domain.TemplateVersion) map[string]any {
	if v == nil {
		return nil
	}

	return map[string]any{
		"id":                    v.ID,
		"template_id":           v.TemplateID,
		"version_number":        v.VersionNumber,
		"status":                v.Status,
		"docx_storage_key":      v.DocxStorageKey,
		"content_hash":          v.ContentHash,
		"metadata_schema":       v.MetadataSchema,
		"placeholder_schema":    v.PlaceholderSchema,
		"author_id":             v.AuthorID,
		"pending_reviewer_role": v.PendingReviewerRole,
		"pending_approver_role": v.PendingApproverRole,
		"reviewer_id":           v.ReviewerID,
		"approver_id":           v.ApproverID,
		"submitted_at":          timePtrRFC3339(v.SubmittedAt),
		"reviewed_at":           timePtrRFC3339(v.ReviewedAt),
		"approved_at":           timePtrRFC3339(v.ApprovedAt),
		"published_at":          timePtrRFC3339(v.PublishedAt),
		"obsoleted_at":          timePtrRFC3339(v.ObsoletedAt),
		"created_at":            v.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func timePtrRFC3339(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
