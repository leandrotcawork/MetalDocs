package httpdelivery

import (
	"net/http"

	"github.com/google/uuid"
)

const docxContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

type ExportHandler struct{}

func NewExportHandler() *ExportHandler {
	return &ExportHandler{}
}

func (h *ExportHandler) ExportDocx(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)
	if userIDFromContext(r.Context()) == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "production"
	}
	if mode != "production" && mode != "debug" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid mode", traceID)
		return
	}

	versionID := r.URL.Query().Get("version_id")
	if versionID != "" {
		if _, err := uuid.Parse(versionID); err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid version_id", traceID)
			return
		}
	}

	// TODO: wire ExportService here once the export slice is connected.
	_ = mode

	if versionID == "" {
		writeAPIError(w, http.StatusNotFound, "VERSION_NOT_FOUND", "Version not found", traceID)
		return
	}

	w.Header().Set("Content-Type", docxContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("docx-stub"))
}
