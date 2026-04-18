package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
	"metaldocs/internal/platform/ratelimit"
)

const (
	roleAdmin          = "admin"
	roleTemplateAuthor = "template_author"
	roleDocumentFiller = "document_filler"
)

type Service interface {
	CreateDocument(ctx context.Context, cmd application.CreateDocumentCmd) (*application.CreateDocumentResult, error)
	GetDocument(ctx context.Context, tenantID, id string) (*domain.Document, error)
	ListDocuments(ctx context.Context, tenantID string) ([]domain.Document, error)
	ListDocumentsForUser(ctx context.Context, tenantID, userID string) ([]domain.Document, error)
	IsDocumentOwner(ctx context.Context, tenantID, docID, userID string) (bool, error)
	AcquireSession(ctx context.Context, tenantID, docID, userID string) (*domain.Session, bool, error)
	HeartbeatSession(ctx context.Context, sessionID, userID string) error
	ReleaseSession(ctx context.Context, tenantID, sessionID, userID, docID string) error
	ForceReleaseSession(ctx context.Context, tenantID, adminID, sessionID, docID string) error
	PresignAutosave(ctx context.Context, cmd application.PresignAutosaveCmd) (*application.PresignAutosaveResult, error)
	CommitAutosave(ctx context.Context, cmd application.CommitAutosaveCmd) (*application.CommitResult, error)
	CreateCheckpoint(ctx context.Context, tenantID, docID, actorID, label string) (*domain.Checkpoint, error)
	ListCheckpoints(ctx context.Context, tenantID, docID string) ([]domain.Checkpoint, error)
	RestoreCheckpoint(ctx context.Context, tenantID, docID, actorID string, versionNum int) (*application.RestoreResult, error)
	Finalize(ctx context.Context, tenantID, docID, actorID string) error
	Archive(ctx context.Context, tenantID, docID, actorID string, fromFinalized bool) error
	SignedRevisionURL(ctx context.Context, tenantID, docID, revID string) (string, error)
}

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/documents", h.listDocuments)
	mux.HandleFunc("POST /api/v2/documents", h.createDocument)

	mux.HandleFunc("GET /api/v2/documents/{id}", h.getDocument)
	mux.HandleFunc("POST /api/v2/documents/{id}/finalize", h.finalizeDocument)
	mux.HandleFunc("POST /api/v2/documents/{id}/archive", h.archiveDocument)

	mux.HandleFunc("POST /api/v2/documents/{id}/sessions/acquire", h.acquireSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/sessions/heartbeat", h.heartbeatSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/sessions/release", h.releaseSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/sessions/force-release", h.forceReleaseSession)

	mux.HandleFunc("POST /api/v2/documents/{id}/autosave/presign", h.presignAutosave)
	mux.HandleFunc("POST /api/v2/documents/{id}/autosave/commit", h.commitAutosave)

	mux.HandleFunc("GET /api/v2/documents/{id}/checkpoints", h.listCheckpoints)
	mux.HandleFunc("POST /api/v2/documents/{id}/checkpoints", h.createCheckpoint)
	mux.HandleFunc("POST /api/v2/documents/{id}/checkpoints/{version}/restore", h.restoreCheckpoint)

	mux.HandleFunc("GET /api/v2/documents/{id}/revisions/{rid}/url", h.signedRevisionURL)
}

func (h *Handler) RegisterRoutesWithRateLimit(mux *http.ServeMux, rl *ratelimit.Middleware, userFn func(*http.Request) string) {
	mux.HandleFunc("GET /api/v2/documents", h.listDocuments)
	mux.HandleFunc("POST /api/v2/documents", h.createDocument)

	mux.HandleFunc("GET /api/v2/documents/{id}", h.getDocument)
	mux.HandleFunc("POST /api/v2/documents/{id}/finalize", h.finalizeDocument)
	mux.HandleFunc("POST /api/v2/documents/{id}/archive", h.archiveDocument)

	mux.HandleFunc("POST /api/v2/documents/{id}/sessions/acquire", h.acquireSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/sessions/heartbeat", h.heartbeatSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/sessions/release", h.releaseSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/sessions/force-release", h.forceReleaseSession)

	mux.Handle(
		"POST /api/v2/documents/{id}/autosave/presign",
		rl.Limit(ratelimit.RouteAutosavePresign, userFn, http.HandlerFunc(h.presignAutosave)),
	)
	mux.Handle(
		"POST /api/v2/documents/{id}/autosave/commit",
		rl.Limit(ratelimit.RouteAutosaveCommit, userFn, http.HandlerFunc(h.commitAutosave)),
	)

	mux.HandleFunc("GET /api/v2/documents/{id}/checkpoints", h.listCheckpoints)
	mux.HandleFunc("POST /api/v2/documents/{id}/checkpoints", h.createCheckpoint)
	mux.HandleFunc("POST /api/v2/documents/{id}/checkpoints/{version}/restore", h.restoreCheckpoint)

	mux.HandleFunc("GET /api/v2/documents/{id}/revisions/{rid}/url", h.signedRevisionURL)
}

func (h *Handler) createDocument(w http.ResponseWriter, r *http.Request) {
	if !hasAnyRole(r, roleAdmin, roleDocumentFiller) {
		httpErr(w, http.StatusForbidden, "forbidden")
		return
	}

	var req struct {
		TemplateVersionID string          `json:"template_version_id"`
		Name              string          `json:"name"`
		FormData          json.RawMessage `json:"form_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_body")
		return
	}

	res, err := h.svc.CreateDocument(r.Context(), application.CreateDocumentCmd{
		TenantID:          tenantIDFromReq(r),
		ActorUserID:       userIDFromReq(r),
		TemplateVersionID: req.TemplateVersionID,
		Name:              req.Name,
		FormData:          req.FormData,
	})
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"document_id":         res.DocumentID,
		"initial_revision_id": res.InitialRevisionID,
		"session_id":          res.SessionID,
	})
}

func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request) {
	if !hasAnyRole(r, roleAdmin, roleDocumentFiller) {
		httpErr(w, http.StatusForbidden, "forbidden")
		return
	}

	tenantID := tenantIDFromReq(r)
	userID := userIDFromReq(r)

	var (
		docs []domain.Document
		err  error
	)
	if hasRole(r, roleAdmin) {
		docs, err = h.svc.ListDocuments(r.Context(), tenantID)
	} else {
		docs, err = h.svc.ListDocumentsForUser(r.Context(), tenantID, userID)
	}
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, docs)
}

func (h *Handler) getDocument(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, _, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	doc, err := h.svc.GetDocument(r.Context(), tenantID, docID)
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) finalizeDocument(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	if err := h.svc.Finalize(r.Context(), tenantID, docID, userID); err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) archiveDocument(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	errFirst := h.svc.Archive(r.Context(), tenantID, docID, userID, true)
	if errFirst == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Only fall back to draft→archived when the doc is not in finalized state.
	if !errors.Is(errFirst, domain.ErrInvalidStateTransition) {
		status, msg := mapErr(errFirst)
		httpErr(w, status, msg)
		return
	}
	if err := h.svc.Archive(r.Context(), tenantID, docID, userID, false); err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) acquireSession(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	sess, readonly, err := h.svc.AcquireSession(r.Context(), tenantID, docID, userID)
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}

	status := http.StatusCreated
	if readonly {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{
		"session":  sess,
		"readonly": readonly,
	})
}

func (h *Handler) heartbeatSession(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	_, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_body")
		return
	}

	if err := h.svc.HeartbeatSession(r.Context(), req.SessionID, userID); err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) releaseSession(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_body")
		return
	}

	if err := h.svc.ReleaseSession(r.Context(), tenantID, req.SessionID, userID, docID); err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) forceReleaseSession(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	if !hasRole(r, roleAdmin) {
		httpErr(w, http.StatusForbidden, "forbidden")
		return
	}
	tenantID := tenantIDFromReq(r)
	adminID := userIDFromReq(r)

	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_body")
		return
	}

	if err := h.svc.ForceReleaseSession(r.Context(), tenantID, adminID, req.SessionID, docID); err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) presignAutosave(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	var req struct {
		SessionID      string `json:"session_id"`
		BaseRevisionID string `json:"base_revision_id"`
		ContentHash    string `json:"content_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_body")
		return
	}

	res, err := h.svc.PresignAutosave(r.Context(), application.PresignAutosaveCmd{
		TenantID:       tenantID,
		ActorUserID:    userID,
		DocumentID:     docID,
		SessionID:      req.SessionID,
		BaseRevisionID: req.BaseRevisionID,
		ContentHash:    req.ContentHash,
	})
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"upload_url":        res.UploadURL,
		"pending_upload_id": res.PendingUploadID,
		"expires_at":        res.ExpiresAt,
	})
}

func (h *Handler) commitAutosave(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	var req struct {
		SessionID        string          `json:"session_id"`
		PendingUploadID  string          `json:"pending_upload_id"`
		FormDataSnapshot json.RawMessage `json:"form_data_snapshot"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_body")
		return
	}

	res, err := h.svc.CommitAutosave(r.Context(), application.CommitAutosaveCmd{
		TenantID:          tenantID,
		ActorUserID:       userID,
		DocumentID:        docID,
		SessionID:         req.SessionID,
		PendingUploadID:   req.PendingUploadID,
		FormDataSnapshot:  req.FormDataSnapshot,
	})
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"revision_id":       res.RevisionID,
		"revision_num":      res.RevisionNum,
		"idempotent_replay": res.AlreadyConsumed,
	})
}

func (h *Handler) listCheckpoints(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, _, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	items, err := h.svc.ListCheckpoints(r.Context(), tenantID, docID)
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) createCheckpoint(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	var req struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_body")
		return
	}

	cp, err := h.svc.CreateCheckpoint(r.Context(), tenantID, docID, userID, req.Label)
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	writeJSON(w, http.StatusCreated, cp)
}

func (h *Handler) restoreCheckpoint(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	versionNum, err := strconv.Atoi(r.PathValue("version"))
	if err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_version")
		return
	}

	res, err := h.svc.RestoreCheckpoint(r.Context(), tenantID, docID, userID, versionNum)
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"new_revision_id":               res.NewRevisionID,
		"new_revision_num":              res.NewRevisionNum,
		"source_checkpoint_version_num": versionNum,
		"idempotent":        res.Idempotent,
	})
}

func (h *Handler) signedRevisionURL(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, _, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	url, err := h.svc.SignedRevisionURL(r.Context(), tenantID, docID, r.PathValue("rid"))
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

func (h *Handler) authorizeDocumentScope(w http.ResponseWriter, r *http.Request, docID string) (tenantID string, userID string, ok bool) {
	if !hasAnyRole(r, roleAdmin, roleDocumentFiller) {
		httpErr(w, http.StatusForbidden, "forbidden")
		return "", "", false
	}
	tenantID = tenantIDFromReq(r)
	userID = userIDFromReq(r)
	if hasRole(r, roleAdmin) {
		return tenantID, userID, true
	}

	owner, err := h.svc.IsDocumentOwner(r.Context(), tenantID, docID, userID)
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return "", "", false
	}
	if !owner {
		httpErr(w, http.StatusForbidden, "forbidden")
		return "", "", false
	}
	return tenantID, userID, true
}

func withAdminCtx(r *http.Request) *http.Request {
	userID := userIDFromReq(r)
	if userID == "" {
		return r
	}

	roles := rolesFromHeader(r.Header.Get("X-User-Roles"))
	ctxRoles := make([]iamdomain.Role, 0, len(roles))
	for _, role := range roles {
		ctxRoles = append(ctxRoles, iamdomain.Role(role))
	}
	ctx := iamdomain.WithAuthContext(r.Context(), userID, ctxRoles)
	return r.WithContext(ctx)
}

func hasAnyRole(r *http.Request, want ...string) bool {
	for _, w := range want {
		if hasRole(r, w) {
			return true
		}
	}
	return false
}

func hasRole(r *http.Request, want string) bool {
	for _, role := range rolesFromHeader(r.Header.Get("X-User-Roles")) {
		if role == want {
			return true
		}
	}
	return false
}

func rolesFromHeader(header string) []string {
	if strings.TrimSpace(header) == "" {
		return nil
	}
	parts := strings.Split(header, ",")
	roles := make([]string, 0, len(parts))
	for _, part := range parts {
		role := strings.ToLower(strings.TrimSpace(part))
		if role != "" {
			roles = append(roles, role)
		}
	}
	return roles
}

func tenantIDFromReq(r *http.Request) string { return strings.TrimSpace(r.Header.Get("X-Tenant-ID")) }

func userIDFromReq(r *http.Request) string { return strings.TrimSpace(r.Header.Get("X-User-ID")) }

func mapErr(err error) (int, string) {
	switch {
	case err == nil:
		return http.StatusOK, ""
	case errors.Is(err, domain.ErrForbidden), errors.Is(err, domain.ErrDocumentNotOwner):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, domain.ErrPendingNotFound):
		return http.StatusNotFound, "pending_not_found"
	case errors.Is(err, domain.ErrCheckpointNotFound):
		return http.StatusNotFound, "checkpoint_not_found"
	case errors.Is(err, domain.ErrExpiredUpload):
		return http.StatusGone, "expired_upload"
	case errors.Is(err, domain.ErrUploadMissing):
		return http.StatusGone, "upload_missing"
	case errors.Is(err, domain.ErrContentHashMismatch):
		return http.StatusUnprocessableEntity, "content_hash_mismatch"
	case errors.Is(err, domain.ErrSessionTaken):
		return http.StatusConflict, "session_taken"
	case errors.Is(err, domain.ErrSessionInactive):
		return http.StatusConflict, "session_inactive"
	case errors.Is(err, domain.ErrSessionNotHolder):
		return http.StatusConflict, "session_not_holder"
	case errors.Is(err, domain.ErrStaleBase):
		return http.StatusConflict, "stale_base"
	case errors.Is(err, domain.ErrMisbound):
		return http.StatusConflict, "misbound"
	case errors.Is(err, domain.ErrInvalidStateTransition):
		return http.StatusConflict, "invalid_state_transition"
	case strings.HasPrefix(err.Error(), "form_data_invalid"):
		return http.StatusUnprocessableEntity, "form_data_invalid"
	default:
		return http.StatusInternalServerError, "internal_error"
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func httpErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
