package httpdelivery

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req createDocumentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.TemplateID) == "" || strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Profile) == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
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
