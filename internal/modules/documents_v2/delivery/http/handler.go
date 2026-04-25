package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
	registrydomain "metaldocs/internal/modules/registry/domain"
	"metaldocs/internal/platform/httpresponse"
	"metaldocs/internal/platform/ratelimit"
)

const (
	roleAdmin          = "admin"
	roleTemplateAuthor = "template_author"
	roleDocumentFiller = "document_filler"
)

type Service interface {
	CreateDocument(ctx context.Context, cmd application.CreateDocumentInput) (*application.CreateDocumentResult, error)
	GetDocument(ctx context.Context, tenantID, id string) (*domain.Document, error)
	RenameDocument(ctx context.Context, tenantID, userID, docID, newName string) error
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
	ListDocumentComments(ctx context.Context, tenantID, userID, documentID string) ([]domain.Comment, error)
	AddDocumentComment(ctx context.Context, tenantID, userID, authorDisplay, documentID string, in domain.CommentCreateInput) (*domain.Comment, error)
	UpdateDocumentComment(ctx context.Context, tenantID, userID, documentID string, libraryID int, in domain.CommentUpdateInput) (*domain.Comment, error)
	DeleteDocumentComment(ctx context.Context, tenantID, userID, documentID string, libraryID int) error
}

type Handler struct{ svc Service }

var writeJSON = httpresponse.WriteJSON

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/documents", h.listDocuments)
	mux.HandleFunc("POST /api/v2/documents", h.createDocument)

	mux.HandleFunc("GET /api/v2/documents/{id}", h.getDocument)
	mux.HandleFunc("PATCH /api/v2/documents/{id}", h.renameDocument)
	mux.HandleFunc("POST /api/v2/documents/{id}/finalize", h.finalizeDocument)
	mux.HandleFunc("POST /api/v2/documents/{id}/archive", h.archiveDocument)

	mux.HandleFunc("POST /api/v2/documents/{id}/session/acquire", h.acquireSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/session/heartbeat", h.heartbeatSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/session/release", h.releaseSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/session/force-release", h.forceReleaseSession)

	mux.HandleFunc("POST /api/v2/documents/{id}/autosave/presign", h.presignAutosave)
	mux.HandleFunc("POST /api/v2/documents/{id}/autosave/commit", h.commitAutosave)

	mux.HandleFunc("GET /api/v2/documents/{id}/checkpoints", h.listCheckpoints)
	mux.HandleFunc("POST /api/v2/documents/{id}/checkpoints", h.createCheckpoint)
	mux.HandleFunc("POST /api/v2/documents/{id}/checkpoints/{version}/restore", h.restoreCheckpoint)

	mux.HandleFunc("GET /api/v2/documents/{id}/revisions/{rid}/url", h.signedRevisionURL)
	mux.HandleFunc("GET /api/v2/documents/{id}/comments", h.listComments)
	mux.HandleFunc("POST /api/v2/documents/{id}/comments", h.createComment)
	mux.HandleFunc("PATCH /api/v2/documents/{id}/comments/{libraryID}", h.updateComment)
	mux.HandleFunc("DELETE /api/v2/documents/{id}/comments/{libraryID}", h.deleteComment)
}

func (h *Handler) RegisterRoutesWithRateLimit(mux *http.ServeMux, rl *ratelimit.Middleware, userFn func(*http.Request) string) {
	mux.HandleFunc("GET /api/v2/documents", h.listDocuments)
	mux.HandleFunc("POST /api/v2/documents", h.createDocument)

	mux.HandleFunc("GET /api/v2/documents/{id}", h.getDocument)
	mux.HandleFunc("PATCH /api/v2/documents/{id}", h.renameDocument)
	mux.HandleFunc("POST /api/v2/documents/{id}/finalize", h.finalizeDocument)
	mux.HandleFunc("POST /api/v2/documents/{id}/archive", h.archiveDocument)

	mux.HandleFunc("POST /api/v2/documents/{id}/session/acquire", h.acquireSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/session/heartbeat", h.heartbeatSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/session/release", h.releaseSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/session/force-release", h.forceReleaseSession)

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
	mux.HandleFunc("GET /api/v2/documents/{id}/comments", h.listComments)
	mux.HandleFunc("POST /api/v2/documents/{id}/comments", h.createComment)
	mux.HandleFunc("PATCH /api/v2/documents/{id}/comments/{libraryID}", h.updateComment)
	mux.HandleFunc("DELETE /api/v2/documents/{id}/comments/{libraryID}", h.deleteComment)
}

func (h *Handler) createDocument(w http.ResponseWriter, r *http.Request) {
	if !hasAnyRole(r, roleAdmin, roleDocumentFiller) {
		httpErr(w, http.StatusForbidden, "forbidden")
		return
	}

	var req struct {
		ControlledDocumentID string          `json:"controlled_document_id"`
		TemplateVersionID    string          `json:"template_version_id"`
		Name                 string          `json:"name"`
		FormData             json.RawMessage `json:"form_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_body")
		return
	}

	res, err := h.svc.CreateDocument(r.Context(), application.CreateDocumentInput{
		TenantID:             tenantIDFromReq(r),
		ActorUserID:          userIDFromReq(r),
		ControlledDocumentID: req.ControlledDocumentID,
		TemplateVersionID:    req.TemplateVersionID,
		Name:                 req.Name,
		FormData:             req.FormData,
	})
	if err != nil {
		status, msg := mapErr(err)
		if status == http.StatusInternalServerError {
			_ = err // logged below
			http.Error(w, `{"error":"`+msg+`","detail":"`+err.Error()+`"}`, status)
			return
		}
		httpErr(w, status, msg)
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, map[string]string{
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

	httpresponse.WriteJSON(w, http.StatusOK, docs)
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
	httpresponse.WriteJSON(w, http.StatusOK, doc)
}

func (h *Handler) renameDocument(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_body")
		return
	}

	if err := h.svc.RenameDocument(r.Context(), tenantID, userID, docID, req.Name); err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}

	doc, err := h.svc.GetDocument(r.Context(), tenantID, docID)
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	httpresponse.WriteJSON(w, http.StatusOK, doc)
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

	if readonly {
		httpresponse.WriteJSON(w, http.StatusOK, map[string]any{
			"mode":       "readonly",
			"held_by":    sess.UserID,
			"held_until": sess.ExpiresAt,
		})
		return
	}
	httpresponse.WriteJSON(w, http.StatusCreated, map[string]any{
		"mode":                 "writer",
		"session_id":           sess.ID,
		"expires_at":           sess.ExpiresAt,
		"last_ack_revision_id": sess.LastAcknowledgedRevisionID,
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
	httpresponse.WriteJSON(w, http.StatusOK, map[string]any{
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
		TenantID:         tenantID,
		ActorUserID:      userID,
		DocumentID:       docID,
		SessionID:        req.SessionID,
		PendingUploadID:  req.PendingUploadID,
		FormDataSnapshot: req.FormDataSnapshot,
	})
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	httpresponse.WriteJSON(w, http.StatusOK, map[string]any{
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
	httpresponse.WriteJSON(w, http.StatusOK, items)
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
	httpresponse.WriteJSON(w, http.StatusCreated, cp)
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
	httpresponse.WriteJSON(w, http.StatusOK, map[string]any{
		"new_revision_id":               res.NewRevisionID,
		"new_revision_num":              res.NewRevisionNum,
		"source_checkpoint_version_num": versionNum,
		"idempotent":                    res.Idempotent,
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
	httpresponse.WriteJSON(w, http.StatusOK, map[string]string{"url": url})
}

func (h *Handler) listComments(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	comments, err := h.svc.ListDocumentComments(r.Context(), tenantID, userID, docID)
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	resp := make([]commentResponse, 0, len(comments))
	for i := range comments {
		resp = append(resp, toCommentResponse(comments[i]))
	}
	httpresponse.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	var req struct {
		LibraryCommentID int             `json:"library_comment_id"`
		ParentLibraryID  *int            `json:"parent_library_id"`
		AuthorDisplay    string          `json:"author_display"`
		Content          json.RawMessage `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_body")
		return
	}

	comment, err := h.svc.AddDocumentComment(r.Context(), tenantID, userID, req.AuthorDisplay, docID, domain.CommentCreateInput{
		LibraryCommentID: req.LibraryCommentID,
		ParentLibraryID:  req.ParentLibraryID,
		AuthorDisplay:    req.AuthorDisplay,
		ContentJSON:      req.Content,
	})
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	httpresponse.WriteJSON(w, http.StatusCreated, toCommentResponse(*comment))
}

func (h *Handler) updateComment(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	libraryID, err := strconv.Atoi(r.PathValue("libraryID"))
	if err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_library_comment_id")
		return
	}
	var req struct {
		Content *json.RawMessage `json:"content"`
		Done    *bool            `json:"done"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_body")
		return
	}

	comment, err := h.svc.UpdateDocumentComment(r.Context(), tenantID, userID, docID, libraryID, domain.CommentUpdateInput{
		ContentJSON: req.Content,
		Done:        req.Done,
	})
	if err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	httpresponse.WriteJSON(w, http.StatusOK, toCommentResponse(*comment))
}

func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request) {
	r = withAdminCtx(r)
	docID := r.PathValue("id")
	tenantID, userID, ok := h.authorizeDocumentScope(w, r, docID)
	if !ok {
		return
	}

	libraryID, err := strconv.Atoi(r.PathValue("libraryID"))
	if err != nil {
		httpErr(w, http.StatusBadRequest, "invalid_library_comment_id")
		return
	}
	if err := h.svc.DeleteDocumentComment(r.Context(), tenantID, userID, docID, libraryID); err != nil {
		status, msg := mapErr(err)
		httpErr(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type commentResponse struct {
	ID               string          `json:"id"`
	LibraryCommentID int             `json:"library_comment_id"`
	ParentLibraryID  *int            `json:"parent_library_id"`
	Author           string          `json:"author"`
	AuthorID         string          `json:"author_id"`
	Content          json.RawMessage `json:"content"`
	Done             bool            `json:"done"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
	ResolvedAt       *time.Time      `json:"resolved_at"`
}

func toCommentResponse(c domain.Comment) commentResponse {
	return commentResponse{
		ID:               c.ID.String(),
		LibraryCommentID: c.LibraryCommentID,
		ParentLibraryID:  c.ParentLibraryID,
		Author:           c.AuthorDisplay,
		AuthorID:         c.AuthorID,
		Content:          c.ContentJSON,
		Done:             c.ResolvedAt != nil,
		CreatedAt:        c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        c.UpdatedAt.UTC().Format(time.RFC3339),
		ResolvedAt:       c.ResolvedAt,
	}
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
	if len(roles) == 0 {
		return r
	}
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
	for _, role := range iamdomain.RolesFromContext(r.Context()) {
		if string(role) == want {
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

const devTenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"

func tenantIDFromReq(r *http.Request) string {
	if t := strings.TrimSpace(r.Header.Get("X-Tenant-ID")); t != "" {
		return t
	}
	return devTenantID
}

func userIDFromReq(r *http.Request) string {
	if u := strings.TrimSpace(r.Header.Get("X-User-ID")); u != "" {
		return u
	}
	return iamdomain.UserIDFromContext(r.Context())
}

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
	case errors.Is(err, domain.ErrCommentNotFound):
		return http.StatusNotFound, "comment_not_found"
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrInvalidName):
		return http.StatusBadRequest, "invalid_name"
	case errors.Is(err, application.ErrControlledDocumentRequired):
		return http.StatusBadRequest, "controlled_document_required"
	case errors.Is(err, domain.ErrCommentInvalid):
		return http.StatusBadRequest, "comment_invalid"
	case errors.Is(err, iamapp.ErrAccessDenied):
		return http.StatusForbidden, "forbidden"
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
	case errors.Is(err, registrydomain.ErrCDNotFound):
		return http.StatusNotFound, "controlled_document_not_found"
	case errors.Is(err, registrydomain.ErrCDNotActive):
		return http.StatusConflict, "controlled_document_not_active"
	case errors.Is(err, registrydomain.ErrProfileHasNoDefaultTemplate):
		return http.StatusConflict, "profile_has_no_default_template"
	case strings.HasPrefix(err.Error(), "form_data_invalid"):
		return http.StatusUnprocessableEntity, "form_data_invalid"
	default:
		return http.StatusInternalServerError, "internal_error"
	}
}

func httpErr(w http.ResponseWriter, code int, msg string) {
	httpresponse.WriteJSON(w, code, map[string]string{"error": msg})
}
