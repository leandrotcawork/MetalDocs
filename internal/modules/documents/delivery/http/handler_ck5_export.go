package httpdelivery

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
)

var safeFilenameRe = regexp.MustCompile(`[^A-Za-z0-9\-_]+`)

func sanitizeFilename(name string) string {
	safe := safeFilenameRe.ReplaceAllString(name, "_")
	if len(safe) > 128 {
		safe = safe[:128]
	}
	if safe == "" {
		return "document"
	}
	return safe
}

func (h *Handler) handleDocumentExportCK5Docx(w http.ResponseWriter, r *http.Request, docID string) {
	traceID := requestTraceID(r)

	// Auth: GetCK5DocumentContent uses GetDocumentAuthorized internally -> 404 on auth failure.
	html, title, err := h.service.GetCK5DocumentContent(r.Context(), docID)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			writeAPIError(w, http.StatusNotFound, "DOC_NOT_FOUND", "Document not found", traceID)
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal error", traceID)
		return
	}

	if h.ck5Export == nil {
		writeAPIError(w, http.StatusBadGateway, "EXPORT_UNAVAILABLE", "CK5 export service not configured", traceID)
		return
	}

	docxBytes, err := h.ck5Export.RenderDocx(r.Context(), html)
	if err != nil {
		var ck5Err *application.CK5ExportError
		if errors.As(err, &ck5Err) {
			if ck5Err.Status >= 400 && ck5Err.Status < 500 {
				writeAPIError(w, http.StatusUnprocessableEntity, "EXPORT_ERROR", ck5Err.Body, traceID)
				return
			}
		}
		writeAPIError(w, http.StatusBadGateway, "EXPORT_UPSTREAM_ERROR", "Upstream CK5 export error", traceID)
		return
	}

	filename := fmt.Sprintf("%s.docx", sanitizeFilename(title))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(docxBytes)
}

func (h *Handler) handleDocumentExportCK5PDF(w http.ResponseWriter, r *http.Request, docID string) {
	traceID := requestTraceID(r)

	html, title, err := h.service.GetCK5DocumentContent(r.Context(), docID)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			writeAPIError(w, http.StatusNotFound, "DOC_NOT_FOUND", "Document not found", traceID)
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal error", traceID)
		return
	}

	if h.ck5Export == nil {
		writeAPIError(w, http.StatusBadGateway, "EXPORT_UNAVAILABLE", "CK5 export service not configured", traceID)
		return
	}

	wrappedHTML, err := h.ck5Export.RenderPDFHtml(r.Context(), html)
	if err != nil {
		var ck5Err *application.CK5ExportError
		if errors.As(err, &ck5Err) {
			if ck5Err.Status >= 400 && ck5Err.Status < 500 {
				writeAPIError(w, http.StatusUnprocessableEntity, "EXPORT_ERROR", ck5Err.Body, traceID)
				return
			}
		}
		writeAPIError(w, http.StatusBadGateway, "EXPORT_UPSTREAM_ERROR", "Upstream CK5 export error", traceID)
		return
	}

	if h.pdfConverter == nil {
		writeAPIError(w, http.StatusBadGateway, "RENDER_UNAVAILABLE", "PDF renderer not configured", traceID)
		return
	}

	pdfBytes, err := h.pdfConverter.ConvertHTMLToPDF(r.Context(), []byte(wrappedHTML), nil)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "RENDER_UPSTREAM_ERROR", "Upstream Gotenberg error", traceID)
		return
	}

	filename := fmt.Sprintf("%s.pdf", sanitizeFilename(title))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}
