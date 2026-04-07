package httpdelivery

import (
	"encoding/json"
	"io"
	"net/http"

	"metaldocs/internal/modules/documents/domain/mddm"
)

type MDDMHandler struct{}

func NewMDDMHandler() *MDDMHandler {
	return &MDDMHandler{}
}

func (h *MDDMHandler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var envelope map[string]any
	if err := json.Unmarshal(body, &envelope); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if err := mddm.ValidateMDDMBytes(body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error":  "validation_failed",
			"detail": err.Error(),
		})
		return
	}

	// Subsequent tasks add canonicalization, lock check, and persistence here.
	// For this task we only verify the schema validates and return 200.
	_ = envelope
	w.WriteHeader(http.StatusOK)
}
