package documentshttp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// PDFWriter persists PDF-completion columns on documents.
type PDFWriter interface {
	WritePDF(ctx context.Context, tenant, docID, s3Key string, pdfHash []byte, generatedAt time.Time) error
}

// PDFWebhookHandler receives completion callbacks from docgen_v2_pdf workers.
// Authentication is HMAC-SHA256 over the raw request body, shared secret in env.
type PDFWebhookHandler struct {
	writer PDFWriter
	secret string
}

func NewPDFWebhookHandler(w PDFWriter, secret string) *PDFWebhookHandler {
	return &PDFWebhookHandler{writer: w, secret: secret}
}

func (h *PDFWebhookHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v2/documents/{id}/pdf-complete", h.HandlePDFComplete)
}

type pdfCompleteBody struct {
	TenantID        string `json:"tenant_id"`
	FinalPDFS3Key   string `json:"final_pdf_s3_key"`
	PDFHash         string `json:"pdf_hash"`
	PDFGeneratedAt  string `json:"pdf_generated_at"`
}

func (h *PDFWebhookHandler) HandlePDFComplete(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeFillInJSON(w, http.StatusBadRequest, map[string]any{"error": "read_body"})
		return
	}
	defer r.Body.Close()

	sig := r.Header.Get("X-Docgen-Signature")
	if !validSignature(raw, sig, h.secret) {
		writeFillInJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid_signature"})
		return
	}

	var body pdfCompleteBody
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&body); err != nil {
		writeFillInJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if strings.TrimSpace(body.TenantID) == "" || strings.TrimSpace(body.FinalPDFS3Key) == "" || body.PDFHash == "" {
		writeFillInJSON(w, http.StatusBadRequest, map[string]any{"error": "missing_fields"})
		return
	}

	hashBytes, err := hex.DecodeString(body.PDFHash)
	if err != nil {
		writeFillInJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_pdf_hash"})
		return
	}

	generatedAt := time.Now().UTC()
	if body.PDFGeneratedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, body.PDFGeneratedAt); err == nil {
			generatedAt = parsed.UTC()
		}
	}

	if err := h.writer.WritePDF(r.Context(), body.TenantID, r.PathValue("id"), body.FinalPDFS3Key, hashBytes, generatedAt); err != nil {
		writeFillInJSON(w, http.StatusInternalServerError, map[string]any{"error": "persist_failed"})
		return
	}

	writeFillInJSON(w, http.StatusOK, map[string]any{
		"document_id":      r.PathValue("id"),
		"final_pdf_s3_key": body.FinalPDFS3Key,
	})
}

func validSignature(body []byte, sigHex, secret string) bool {
	if sigHex == "" || secret == "" {
		return false
	}
	expected, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(expected, mac.Sum(nil))
}
