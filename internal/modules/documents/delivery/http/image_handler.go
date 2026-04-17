package httpdelivery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/domain"
)

const maxImageBytes = 10 * 1024 * 1024

var allowedMIMEs = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
	"image/gif":  true,
}

type ImageHandler struct {
	storage domain.ImageStorage
}

func NewImageHandler(storage domain.ImageStorage) *ImageHandler {
	return &ImageHandler{storage: storage}
}

type uploadResponse struct {
	ImageID  uuid.UUID `json:"image_id"`
	MimeType string    `json:"mime_type"`
}

func (h *ImageHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxImageBytes + 1024); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	bytes, err := io.ReadAll(io.LimitReader(file, maxImageBytes+1))
	if err != nil {
		http.Error(w, "read failure", http.StatusInternalServerError)
		return
	}
	if len(bytes) > maxImageBytes {
		http.Error(w, "image too large (max 10 MB)", http.StatusRequestEntityTooLarge)
		return
	}

	mimeType := http.DetectContentType(bytes)
	if !allowedMIMEs[mimeType] {
		http.Error(w, "unsupported image type", http.StatusUnsupportedMediaType)
		return
	}

	sum := sha256.Sum256(bytes)
	hash := hex.EncodeToString(sum[:])

	id, err := h.storage.Put(r.Context(), hash, mimeType, bytes)
	if err != nil {
		http.Error(w, "storage failure: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uploadResponse{ImageID: id, MimeType: mimeType})
}

func (h *ImageHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/images/"):]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid image id", http.StatusBadRequest)
		return
	}
	bytes, mimeType, err := h.storage.Get(r.Context(), id)
	if err == domain.ErrImageNotFound {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "read failure", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("ETag", `"`+id.String()+`"`)
	w.Write(bytes)
}
