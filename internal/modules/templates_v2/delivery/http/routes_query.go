package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"metaldocs/internal/modules/templates_v2/application"
)

func (h *Handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromReq(r)
	q := r.URL.Query()

	limit, ok := readQueryInt(q.Get("limit"), 50)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_limit", "limit must be an integer")
		return
	}
	offset, ok := readQueryInt(q.Get("offset"), 0)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_offset", "offset must be an integer")
		return
	}

	docType := strings.TrimSpace(q.Get("doc_type"))
	var docTypeCode *string
	if docType != "" {
		docTypeCode = &docType
	}

	templates, err := h.svc.ListTemplates(r.Context(), application.ListFilter{
		TenantID:    tenantID,
		AreaAny:     readCSVQuery(q["area"]),
		DocTypeCode: docTypeCode,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		writeMappedErr(w, err)
		return
	}

	out := make([]map[string]any, 0, len(templates))
	for _, tpl := range templates {
		out = append(out, toTemplateResponse(tpl))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"templates": out,
		},
		"meta": map[string]int{
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (h *Handler) getTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromReq(r)
	templateID := r.PathValue("id")

	tpl, err := h.svc.GetTemplate(r.Context(), tenantID, templateID)
	if err != nil {
		writeMappedErr(w, err)
		return
	}

	latest, err := h.svc.GetVersion(r.Context(), tenantID, templateID, tpl.LatestVersion)
	if err != nil {
		writeMappedErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"template":       toTemplateResponse(tpl),
			"latest_version": toVersionResponse(latest),
		},
	})
}

func (h *Handler) getVersion(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromReq(r)
	templateID := r.PathValue("id")
	versionNum, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_version_number", "version must be an integer")
		return
	}

	v, err := h.svc.GetVersion(r.Context(), tenantID, templateID, versionNum)
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

func (h *Handler) listAudit(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromReq(r)
	templateID := r.PathValue("id")
	q := r.URL.Query()

	limit, ok := readQueryInt(q.Get("limit"), 50)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_limit", "limit must be an integer")
		return
	}
	offset, ok := readQueryInt(q.Get("offset"), 0)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_offset", "offset must be an integer")
		return
	}

	events, err := h.svc.ListAudit(r.Context(), tenantID, templateID, limit, offset)
	if err != nil {
		writeMappedErr(w, err)
		return
	}

	out := make([]map[string]any, 0, len(events))
	for _, event := range events {
		out = append(out, map[string]any{
			"tenant_id":   event.TenantID,
			"template_id": event.TemplateID,
			"version_id":  event.VersionID,
			"actor_id":    event.ActorID,
			"action":      event.Action,
			"details":     event.Details,
			"occurred_at": event.OccurredAt.UTC().Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"audit": out,
		},
		"meta": map[string]int{
			"limit":  limit,
			"offset": offset,
		},
	})
}

func readCSVQuery(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			v := strings.TrimSpace(part)
			if v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

func readQueryInt(raw string, fallback int) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return v, true
}
