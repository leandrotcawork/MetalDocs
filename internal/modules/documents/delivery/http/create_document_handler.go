package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strings"
)

const maxCreateDocumentPayloadBytes = 5 * 1024 * 1024

type CreateDocumentHandler struct{}

func NewCreateDocumentHandler() *CreateDocumentHandler {
	return &CreateDocumentHandler{}
}

type createDocumentRequest struct {
	TemplateID string `json:"template_id"`
	Title      string `json:"title"`
	Profile    string `json:"profile"`
}

type createDocumentResponse struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

func (h *CreateDocumentHandler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)
	r.Body = http.MaxBytesReader(w, r.Body, maxCreateDocumentPayloadBytes)

	var req createDocumentRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	if strings.TrimSpace(req.TemplateID) == "" || strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Profile) == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "template_id, title, profile are required", traceID)
		return
	}

	// TODO: wire CreationService here in the next task.
	writeJSON(w, http.StatusCreated, createDocumentResponse{
		ID:   "stub",
		Code: "PO-001",
	})
}

func newTestCreateHandler(t interface{ Helper() }) *CreateDocumentHandler {
	t.Helper()
	return NewCreateDocumentHandler()
}
