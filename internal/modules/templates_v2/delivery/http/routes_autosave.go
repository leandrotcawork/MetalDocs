package http

import (
	"net/http"
	"strconv"
	"time"

	"metaldocs/internal/modules/templates_v2/application"
)

func (h *Handler) presignAutosave(w http.ResponseWriter, r *http.Request) {
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

	res, err := h.svc.PresignAutosave(r.Context(), application.PresignAutosaveCmd{
		TenantID:      tenantID,
		ActorUserID:   actorID,
		TemplateID:    templateID,
		VersionNumber: versionNum,
	})
	if err != nil {
		writeMappedErr(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]any{
			"upload_url":  res.UploadURL,
			"storage_key": res.StorageKey,
			"expires_at":  res.ExpiresAt.UTC().Format(time.RFC3339),
		},
	})
}

func (h *Handler) commitAutosave(w http.ResponseWriter, r *http.Request) {
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
		ExpectedContentHash string `json:"expected_content_hash"`
	}
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	v, err := h.svc.CommitAutosave(r.Context(), application.CommitAutosaveCmd{
		TenantID:            tenantID,
		ActorUserID:         actorID,
		TemplateID:          templateID,
		VersionNumber:       versionNum,
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
