package http

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/platform/ratelimit"
)

type ExportHandler struct {
	svc *application.ExportService
}

type exportPDFReq struct {
	PaperSize string `json:"paper_size,omitempty"`
	Landscape bool   `json:"landscape,omitempty"`
}

type exportPDFResp struct {
	StorageKey    string `json:"storage_key"`
	SignedURL     string `json:"signed_url"`
	CompositeHash string `json:"composite_hash"`
	SizeBytes     int64  `json:"size_bytes"`
	Cached        bool   `json:"cached"`
	RevisionID    string `json:"revision_id"`
}

type exportDocxURLResp struct {
	SignedURL  string `json:"signed_url"`
	RevisionID string `json:"revision_id"`
}

func NewExportHandler(svc *application.ExportService) *ExportHandler { return &ExportHandler{svc: svc} }

func (h *ExportHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v2/documents/{id}/export/pdf", h.exportPDF)
	mux.HandleFunc("GET /api/v2/documents/{id}/export/docx-url", h.exportDocxURL)
}

func (h *ExportHandler) RegisterRoutesWithRateLimit(mux *http.ServeMux, rl *ratelimit.Middleware, userFn func(*http.Request) string) {
	mux.Handle(
		"POST /api/v2/documents/{id}/export/pdf",
		rl.Limit(ratelimit.RouteExportPDF, userFn, http.HandlerFunc(h.exportPDF)),
	)
	mux.HandleFunc("GET /api/v2/documents/{id}/export/docx-url", h.exportDocxURL)
}

func (h *ExportHandler) exportPDF(w http.ResponseWriter, r *http.Request) {
	docID := r.PathValue("id")
	tenantID := tenantIDFromReq(r)
	userID := userIDFromReq(r)

	req := exportPDFReq{PaperSize: "A4"}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpErr(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if req.PaperSize == "" {
			req.PaperSize = "A4"
		}
	}

	res, err := h.svc.ExportPDF(r.Context(), tenantID, userID, docID, domain.RenderOptions{
		PaperSize:  req.PaperSize,
		LandscapeP: req.Landscape,
	})
	if err != nil {
		status, code := mapExportErr(err)
		httpErr(w, status, code)
		return
	}

	signedURL, err := h.svc.SignExportURL(r.Context(), res.Export.StorageKey)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, "internal")
		return
	}

	writeJSON(w, http.StatusOK, exportPDFResp{
		StorageKey:    res.Export.StorageKey,
		SignedURL:     signedURL,
		CompositeHash: hex.EncodeToString(res.Export.CompositeHash),
		SizeBytes:     res.Export.SizeBytes,
		Cached:        res.Cached,
		RevisionID:    res.Export.RevisionID,
	})
}

func (h *ExportHandler) exportDocxURL(w http.ResponseWriter, r *http.Request) {
	docID := r.PathValue("id")
	tenantID := tenantIDFromReq(r)
	userID := userIDFromReq(r)

	signedURL, err := h.svc.SignedDocxURL(r.Context(), tenantID, userID, docID)
	if err != nil {
		status, code := mapExportErr(err)
		httpErr(w, status, code)
		return
	}

	summary, err := h.svc.GetDocumentSummary(r.Context(), tenantID, docID)
	if err != nil {
		status, code := mapExportErr(err)
		httpErr(w, status, code)
		return
	}

	writeJSON(w, http.StatusOK, exportDocxURLResp{
		SignedURL:  signedURL,
		RevisionID: summary.CurrentRevisionID,
	})
}

func mapExportErr(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrExportDocxMissing):
		return http.StatusConflict, "docx_missing"
	case errors.Is(err, domain.ErrExportGotenbergFailed):
		return http.StatusBadGateway, "gotenberg_failed"
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "document_not_found"
	default:
		return http.StatusInternalServerError, "internal"
	}
}
