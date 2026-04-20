package http

import (
	"net/http"
	"strconv"

	"metaldocs/internal/modules/templates_v2/application"
	"metaldocs/internal/modules/templates_v2/domain"
)

func (h *Handler) updateSchemas(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		MetadataSchema      domain.MetadataSchema `json:"metadata_schema"`
		PlaceholderSchema   []domain.Placeholder  `json:"placeholder_schema"`
		EditableZones       []domain.EditableZone `json:"editable_zones"`
		ExpectedContentHash string                `json:"expected_content_hash"`
	}
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	v, err := h.svc.UpdateSchemas(r.Context(), application.UpdateSchemasCmd{
		TenantID:            tenantID,
		ActorUserID:         actorID,
		TemplateID:          templateID,
		VersionNumber:       versionNum,
		MetadataSchema:      req.MetadataSchema,
		PlaceholderSchema:   req.PlaceholderSchema,
		EditableZones:       req.EditableZones,
		ExpectedContentHash: req.ExpectedContentHash,
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
