package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	httphandler "metaldocs/internal/modules/documents_v2/delivery/http"
	"metaldocs/internal/modules/documents_v2/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type commentsStatefulSvc struct {
	*fakeSvc
	comments   map[int]domain.Comment
	nextOffset int
}

func newCommentsStatefulSvc() *commentsStatefulSvc {
	return &commentsStatefulSvc{
		fakeSvc:    &fakeSvc{},
		comments:   map[int]domain.Comment{},
		nextOffset: 0,
	}
}

func (s *commentsStatefulSvc) ListDocumentComments(_ context.Context, _, _, _ string) ([]domain.Comment, error) {
	out := make([]domain.Comment, 0, len(s.comments))
	for _, c := range s.comments {
		out = append(out, c)
	}
	for i := 0; i < len(out)-1; i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.Before(out[i].CreatedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (s *commentsStatefulSvc) AddDocumentComment(_ context.Context, _, userID, _, _ string, in domain.CommentCreateInput) (*domain.Comment, error) {
	now := time.Now().UTC().Add(time.Duration(s.nextOffset) * time.Second)
	s.nextOffset++
	comment := domain.Comment{
		ID:               uuid.New(),
		LibraryCommentID: in.LibraryCommentID,
		ParentLibraryID:  in.ParentLibraryID,
		AuthorID:         userID,
		AuthorDisplay:    in.AuthorDisplay,
		ContentJSON:      append([]byte(nil), in.ContentJSON...),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.comments[in.LibraryCommentID] = comment
	copyComment := comment
	return &copyComment, nil
}

func (s *commentsStatefulSvc) UpdateDocumentComment(_ context.Context, _, userID, _ string, libraryID int, in domain.CommentUpdateInput) (*domain.Comment, error) {
	comment, ok := s.comments[libraryID]
	if !ok {
		return nil, domain.ErrCommentNotFound
	}
	now := time.Now().UTC().Add(time.Duration(s.nextOffset) * time.Second)
	s.nextOffset++
	if in.ContentJSON != nil {
		comment.ContentJSON = append([]byte(nil), (*in.ContentJSON)...)
	}
	if in.Done != nil {
		if *in.Done {
			comment.ResolvedAt = &now
			comment.ResolvedBy = &userID
		} else {
			comment.ResolvedAt = nil
			comment.ResolvedBy = nil
		}
	}
	comment.UpdatedAt = now
	s.comments[libraryID] = comment
	copyComment := comment
	return &copyComment, nil
}

func (s *commentsStatefulSvc) DeleteDocumentComment(_ context.Context, _, _, _ string, libraryID int) error {
	if _, ok := s.comments[libraryID]; !ok {
		return domain.ErrCommentNotFound
	}
	delete(s.comments, libraryID)
	for id, c := range s.comments {
		if c.ParentLibraryID != nil && *c.ParentLibraryID == libraryID {
			delete(s.comments, id)
		}
	}
	return nil
}

func mustJSONEqual(t *testing.T, got, want json.RawMessage) {
	t.Helper()
	var gotAny any
	var wantAny any
	if err := json.Unmarshal(got, &gotAny); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if err := json.Unmarshal(want, &wantAny); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if !reflect.DeepEqual(gotAny, wantAny) {
		t.Fatalf("json mismatch want=%s got=%s", string(want), string(got))
	}
}

func newMuxWithCommentsSvc(t *testing.T, svc *commentsStatefulSvc) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	h := httphandler.NewHandler(svc)
	h.RegisterRoutes(mux)
	return mux
}

func TestCreateComment_RoundTrip(t *testing.T) {
	svc := newCommentsStatefulSvc()
	mux := newMuxWithCommentsSvc(t, svc)

	content := json.RawMessage(`[{"type":"paragraph","children":[{"text":"hello"}]}]`)
	payload := []byte(`{"library_comment_id":42,"author_display":"Alice","content":[{"type":"paragraph","children":[{"text":"hello"}]}]}`)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc_1/comments", bytes.NewReader(payload))
	withAuthHeaders(postReq, "document_filler")
	postRR := httptest.NewRecorder()
	mux.ServeHTTP(postRR, postReq)
	if postRR.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", postRR.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v2/documents/doc_1/comments", nil)
	withAuthHeaders(getReq, "document_filler")
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRR.Code)
	}

	var out []struct {
		LibraryCommentID int             `json:"library_comment_id"`
		Content          json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(getRR.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 1 || out[0].LibraryCommentID != 42 {
		t.Fatalf("unexpected comments list: %+v", out)
	}
	mustJSONEqual(t, out[0].Content, content)
}

func TestResolveComment_DerivedDoneField(t *testing.T) {
	svc := newCommentsStatefulSvc()
	mux := newMuxWithCommentsSvc(t, svc)

	postReq := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc_1/comments", bytes.NewReader([]byte(`{"library_comment_id":7,"author_display":"Alice","content":[{"type":"paragraph","children":[{"text":"todo"}]}]}`)))
	withAuthHeaders(postReq, "document_filler")
	postRR := httptest.NewRecorder()
	mux.ServeHTTP(postRR, postReq)
	if postRR.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", postRR.Code)
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v2/documents/doc_1/comments/7", bytes.NewReader([]byte(`{"done":true}`)))
	withAuthHeaders(patchReq, "document_filler")
	patchRR := httptest.NewRecorder()
	mux.ServeHTTP(patchRR, patchReq)
	if patchRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", patchRR.Code)
	}

	var patchOut struct {
		Done       bool       `json:"done"`
		ResolvedAt *time.Time `json:"resolved_at"`
	}
	if err := json.Unmarshal(patchRR.Body.Bytes(), &patchOut); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	if !patchOut.Done || patchOut.ResolvedAt == nil {
		t.Fatalf("expected done=true with resolved_at, got %+v", patchOut)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v2/documents/doc_1/comments", nil)
	withAuthHeaders(getReq, "document_filler")
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRR.Code)
	}
	var out []struct {
		Done       bool       `json:"done"`
		ResolvedAt *time.Time `json:"resolved_at"`
	}
	if err := json.Unmarshal(getRR.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(out) != 1 || !out[0].Done || out[0].ResolvedAt == nil {
		t.Fatalf("expected done=true with resolved_at set, got %+v", out)
	}
}

func TestReplyThread_ParentLibraryID(t *testing.T) {
	svc := newCommentsStatefulSvc()
	mux := newMuxWithCommentsSvc(t, svc)

	rootReq := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc_1/comments", bytes.NewReader([]byte(`{"library_comment_id":100,"author_display":"Alice","content":[{"type":"paragraph","children":[{"text":"root"}]}]}`)))
	withAuthHeaders(rootReq, "document_filler")
	rootReq = rootReq.WithContext(iamdomain.WithAuthContext(rootReq.Context(), "user_root", []iamdomain.Role{}))
	rootRR := httptest.NewRecorder()
	mux.ServeHTTP(rootRR, rootReq)
	if rootRR.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rootRR.Code)
	}

	replyReq := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc_1/comments", bytes.NewReader([]byte(`{"library_comment_id":101,"parent_library_id":100,"author_display":"Bob","content":[{"type":"paragraph","children":[{"text":"reply"}]}]}`)))
	withAuthHeaders(replyReq, "document_filler")
	replyReq = replyReq.WithContext(iamdomain.WithAuthContext(replyReq.Context(), "user_reply", []iamdomain.Role{}))
	replyRR := httptest.NewRecorder()
	mux.ServeHTTP(replyRR, replyReq)
	if replyRR.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", replyRR.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v2/documents/doc_1/comments", nil)
	withAuthHeaders(getReq, "document_filler")
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRR.Code)
	}

	var out []struct {
		LibraryCommentID int    `json:"library_comment_id"`
		ParentLibraryID  *int   `json:"parent_library_id"`
		Author           string `json:"author"`
		CreatedAt        string `json:"created_at"`
	}
	if err := json.Unmarshal(getRR.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(out))
	}
	if out[0].LibraryCommentID != 100 || out[0].ParentLibraryID != nil || out[0].Author != "Alice" || out[0].CreatedAt == "" {
		t.Fatalf("unexpected root comment: %+v", out[0])
	}
	if out[1].LibraryCommentID != 101 || out[1].ParentLibraryID == nil || *out[1].ParentLibraryID != 100 || out[1].Author != "Bob" || out[1].CreatedAt == "" {
		t.Fatalf("unexpected reply comment: %+v", out[1])
	}
}
