package httpdelivery

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
)

func (h *Handler) handleDocumentContentNativeGet(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	version, err := h.service.GetNativeContentAuthorized(r.Context(), documentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	if strings.TrimSpace(version.ContentSource) == "" {
		version.ContentSource = domain.ContentSourceNative
	}
	content := version.NativeContent
	if content == nil {
		content = map[string]any{}
	}

	writeJSON(w, http.StatusOK, DocumentContentNativeResponse{
		DocumentID:    version.DocumentID,
		Version:       version.Number,
		ContentSource: version.ContentSource,
		Content:       content,
	})
}

func (h *Handler) handleDocumentContentNativePost(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)

	var req DocumentContentNativeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	version, err := h.service.SaveNativeContentAuthorized(r.Context(), domain.SaveNativeContentCommand{
		DocumentID: documentID,
		Content:    req.Content,
		TraceID:    traceID,
	})
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	if h.signer == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment signer is not configured", traceID)
		return
	}

	expiresAt := time.Now().UTC().Add(h.downloadTTL)
	pdfURL := h.signer.BuildDownloadURL("/api/v1/documents/"+documentID+"/content/pdf", documentID+":pdf", expiresAt)
	writeJSON(w, http.StatusCreated, DocumentContentSaveResponse{
		DocumentID:    documentID,
		Version:       version.Number,
		ContentSource: normalizeContentSource(version.ContentSource),
		PdfURL:        pdfURL,
		ExpiresAt:     expiresAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleDocumentContentRenderPDF(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)

	version, err := h.service.RenderContentPDFAuthorized(r.Context(), documentID, traceID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	if h.signer == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment signer is not configured", traceID)
		return
	}

	expiresAt := time.Now().UTC().Add(h.downloadTTL)
	pdfURL := h.signer.BuildDownloadURL("/api/v1/documents/"+documentID+"/content/pdf", documentID+":pdf", expiresAt)
	writeJSON(w, http.StatusOK, DocumentContentPdfResponse{
		DocumentID:    documentID,
		Version:       version.Number,
		ContentSource: normalizeContentSource(version.ContentSource),
		PdfURL:        pdfURL,
		ExpiresAt:     expiresAt.Format(time.RFC3339),
		PageCount:     version.PageCount,
	})
}

func (h *Handler) handleDocumentContentPDF(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	if h.signer == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment signer is not configured", traceID)
		return
	}
	expiresAt := strings.TrimSpace(r.URL.Query().Get("expiresAt"))
	signature := strings.TrimSpace(r.URL.Query().Get("signature"))
	if signature != "" || expiresAt != "" {
		if !h.signer.Verify(documentID+":pdf", expiresAt, signature) {
			writeAPIError(w, http.StatusForbidden, "CONTENT_URL_INVALID", "Content URL is invalid or expired", traceID)
			return
		}
		version, err := h.service.GetNativeContentAuthorized(r.Context(), documentID)
		if err != nil {
			h.writeDomainError(w, err, traceID)
			return
		}
		if strings.TrimSpace(version.PdfStorageKey) == "" {
			writeAPIError(w, http.StatusNotFound, "VERSION_NOT_FOUND", "PDF content not found", traceID)
			return
		}
		content, err := h.service.OpenContentStorage(r.Context(), version.PdfStorageKey)
		if err != nil {
			h.writeDomainError(w, err, traceID)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="document-`+documentID+`-v`+strconv.Itoa(version.Number)+`.pdf"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
		return
	}

	version, err := h.service.GetNativeContentAuthorized(r.Context(), documentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	if strings.TrimSpace(version.PdfStorageKey) == "" {
		writeAPIError(w, http.StatusNotFound, "VERSION_NOT_FOUND", "PDF content not found", traceID)
		return
	}

	exp := time.Now().UTC().Add(h.downloadTTL)
	pdfURL := h.signer.BuildDownloadURL("/api/v1/documents/"+documentID+"/content/pdf", documentID+":pdf", exp)
	writeJSON(w, http.StatusOK, DocumentContentPdfResponse{
		DocumentID:    documentID,
		Version:       version.Number,
		ContentSource: normalizeContentSource(version.ContentSource),
		PdfURL:        pdfURL,
		ExpiresAt:     exp.Format(time.RFC3339),
		PageCount:     version.PageCount,
	})
}

func (h *Handler) handleDocumentContentDocx(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	if h.signer == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment signer is not configured", traceID)
		return
	}
	expiresAt := strings.TrimSpace(r.URL.Query().Get("expiresAt"))
	signature := strings.TrimSpace(r.URL.Query().Get("signature"))
	if signature != "" || expiresAt != "" {
		if !h.signer.Verify(documentID+":docx", expiresAt, signature) {
			writeAPIError(w, http.StatusForbidden, "CONTENT_URL_INVALID", "Content URL is invalid or expired", traceID)
			return
		}
		version, err := h.service.GetNativeContentAuthorized(r.Context(), documentID)
		if err != nil {
			h.writeDomainError(w, err, traceID)
			return
		}
		if strings.TrimSpace(version.DocxStorageKey) == "" {
			writeAPIError(w, http.StatusNotFound, "VERSION_NOT_FOUND", "DOCX content not found", traceID)
			return
		}
		content, err := h.service.OpenContentStorage(r.Context(), version.DocxStorageKey)
		if err != nil {
			h.writeDomainError(w, err, traceID)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		w.Header().Set("Content-Disposition", `attachment; filename="document-`+documentID+`-v`+strconv.Itoa(version.Number)+`.docx"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
		return
	}

	version, err := h.service.GetNativeContentAuthorized(r.Context(), documentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	if strings.TrimSpace(version.DocxStorageKey) == "" {
		writeAPIError(w, http.StatusNotFound, "VERSION_NOT_FOUND", "DOCX content not found", traceID)
		return
	}

	exp := time.Now().UTC().Add(h.downloadTTL)
	docxURL := h.signer.BuildDownloadURL("/api/v1/documents/"+documentID+"/content/docx", documentID+":docx", exp)
	writeJSON(w, http.StatusOK, DocumentContentDocxResponse{
		DocxURL:   docxURL,
		ExpiresAt: exp.Format(time.RFC3339),
	})
}

func (h *Handler) handleDocumentTemplateDocx(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	payload, err := h.service.RenderDocumentTemplateDocxAuthorized(r.Context(), documentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", `attachment; filename="document-`+documentID+`-template.docx"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func (h *Handler) handleDocumentProfileTemplateDocx(w http.ResponseWriter, r *http.Request, profileCode string) {
	traceID := requestTraceID(r)
	payload, err := h.service.RenderProfileTemplateDocx(r.Context(), profileCode)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", `attachment; filename="profile-`+strings.ToLower(strings.TrimSpace(profileCode))+`-template.docx"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func (h *Handler) handleDocumentContentUpload(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid multipart payload", traceID)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Missing file field", traceID)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, 10<<20+1))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to read attachment payload", traceID)
		return
	}
	if len(content) == 0 || len(content) > 10*1024*1024 {
		writeAPIError(w, http.StatusBadRequest, "INVALID_ATTACHMENT", "Attachment size must be between 1 byte and 10 MB", traceID)
		return
	}

	version, err := h.service.UploadDocxContentAuthorized(r.Context(), domain.UploadDocxContentCommand{
		DocumentID: documentID,
		FileName:   header.Filename,
		Content:    content,
		TraceID:    traceID,
	})
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	if h.signer == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment signer is not configured", traceID)
		return
	}

	expiresAt := time.Now().UTC().Add(h.downloadTTL)
	docxURL := h.signer.BuildDownloadURL("/api/v1/documents/"+documentID+"/content/docx", documentID+":docx", expiresAt)
	pdfURL := h.signer.BuildDownloadURL("/api/v1/documents/"+documentID+"/content/pdf", documentID+":pdf", expiresAt)
	writeJSON(w, http.StatusCreated, DocumentContentUploadResponse{
		ContentSource: normalizeContentSource(version.ContentSource),
		DocxURL:       docxURL,
		PdfURL:        pdfURL,
		ExpiresAt:     expiresAt.Format(time.RFC3339),
		PageCount:     version.PageCount,
	})
}
