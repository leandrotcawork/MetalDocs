package httpdelivery

import (
	"io"
	"net/http"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func (h *Handler) handleUploadAttachment(w http.ResponseWriter, r *http.Request, documentID string) {
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

	attachment, err := h.service.UploadAttachmentAuthorized(r.Context(), domain.UploadAttachmentCommand{
		DocumentID:  documentID,
		FileName:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Content:     content,
		UploadedBy:  iamdomain.UserIDFromContext(r.Context()),
		TraceID:     traceID,
	})
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusCreated, mapAttachmentResponse(attachment))
}

func (h *Handler) handleListAttachments(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	items, err := h.service.ListAttachmentsAuthorized(r.Context(), documentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]AttachmentResponse, 0, len(items))
	for _, item := range items {
		out = append(out, mapAttachmentResponse(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleCreateAttachmentDownloadURL(w http.ResponseWriter, r *http.Request, documentID, attachmentID string) {
	traceID := requestTraceID(r)
	if h.signer == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment signer is not configured", traceID)
		return
	}
	attachment, err := h.service.GetAttachmentAuthorized(r.Context(), documentID, attachmentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	expiresAt := time.Now().UTC().Add(h.downloadTTL)
	downloadURL := h.signer.BuildDownloadURL("/api/v1/attachments/"+attachment.ID+"/content", attachment.ID, expiresAt)
	writeJSON(w, http.StatusOK, AttachmentDownloadURLResponse{
		AttachmentID: attachment.ID,
		DownloadURL:  downloadURL,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleAttachmentDownloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.signer == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment signer is not configured", requestTraceID(r))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/attachments/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || parts[1] != "content" {
		writeAPIError(w, http.StatusNotFound, "ATTACHMENT_NOT_FOUND", "Attachment route not found", requestTraceID(r))
		return
	}

	attachmentID := parts[0]
	expiresAt := strings.TrimSpace(r.URL.Query().Get("expiresAt"))
	signature := strings.TrimSpace(r.URL.Query().Get("signature"))
	if !h.signer.Verify(attachmentID, expiresAt, signature) {
		writeAPIError(w, http.StatusForbidden, "ATTACHMENT_URL_INVALID", "Attachment URL is invalid or expired", requestTraceID(r))
		return
	}

	attachment, content, err := h.service.OpenAttachmentContent(r.Context(), attachmentID)
	if err != nil {
		h.writeDomainError(w, err, requestTraceID(r))
		return
	}

	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+attachment.FileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
