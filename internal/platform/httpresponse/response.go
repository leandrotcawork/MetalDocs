package httpresponse

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, map[string]string{"code": code, "message": message})
}

func ReadJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
