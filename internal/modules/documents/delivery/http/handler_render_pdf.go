package httpdelivery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"metaldocs/internal/modules/documents/domain"
)

// PDFRenderer is the minimal contract the render handler needs from Gotenberg.
type PDFRenderer interface {
	ConvertHTMLToPDF(ctx context.Context, html []byte, css []byte) ([]byte, error)
}

// DocumentAuthorizer verifies the caller can read the target document.
type DocumentAuthorizer interface {
	GetDocumentAuthorized(ctx context.Context, documentID string) (domain.Document, error)
}

// RenderPDFHandler converts editor HTML/CSS to PDF via Gotenberg.
type RenderPDFHandler struct {
	renderer        PDFRenderer
	authz           DocumentAuthorizer
	MaxPayloadBytes int64
}

const defaultRenderPDFMaxPayload = 10 * 1024 * 1024 // 10 MB

func NewRenderPDFHandler(renderer PDFRenderer, authz DocumentAuthorizer) *RenderPDFHandler {
	return &RenderPDFHandler{
		renderer:        renderer,
		authz:           authz,
		MaxPayloadBytes: defaultRenderPDFMaxPayload,
	}
}

func (h *RenderPDFHandler) HandleRenderPDF(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)

	if userIDFromContext(r.Context()) == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	if documentID == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Document ID required", traceID)
		return
	}

	if h.authz != nil {
		if _, err := h.authz.GetDocumentAuthorized(r.Context(), documentID); err != nil {
			switch {
			case errors.Is(err, domain.ErrDocumentNotFound):
				writeAPIError(w, http.StatusNotFound, "DOCUMENT_NOT_FOUND", "Document not found", traceID)
			default:
				writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", fmt.Sprintf("authz: %v", err), traceID)
			}
			return
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.MaxPayloadBytes)
	if err := r.ParseMultipartForm(h.MaxPayloadBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeAPIError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Payload exceeds limit", traceID)
			return
		}
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", fmt.Sprintf("multipart parse: %v", err), traceID)
		return
	}

	htmlBytes, err := readFormFile(r, "index.html")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), traceID)
		return
	}
	cssBytes, _ := readFormFile(r, "style.css") // optional

	if h.renderer == nil {
		writeAPIError(w, http.StatusBadGateway, "RENDER_UNAVAILABLE", "PDF renderer not configured", traceID)
		return
	}

	pdf, err := h.renderer.ConvertHTMLToPDF(r.Context(), htmlBytes, cssBytes)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "RENDER_UPSTREAM_ERROR", fmt.Sprintf("pdf render failed: %v", err), traceID)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdf)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

func readFormFile(r *http.Request, name string) ([]byte, error) {
	file, _, err := r.FormFile(name)
	if err != nil {
		return nil, fmt.Errorf("missing form file %q: %w", name, err)
	}
	defer file.Close()
	return io.ReadAll(file)
}
