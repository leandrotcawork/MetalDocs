package httpdelivery

import "net/http"

const docxContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

type ExportHandler struct{}

func NewExportHandler() *ExportHandler {
	return &ExportHandler{}
}

func (h *ExportHandler) ExportDocx(w http.ResponseWriter, r *http.Request) {
	versionID := r.URL.Query().Get("version_id")
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "production"
	}

	// TODO: wire ExportService here once the export slice is connected.
	_ = mode

	if versionID == "" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", docxContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("docx-stub"))
}
