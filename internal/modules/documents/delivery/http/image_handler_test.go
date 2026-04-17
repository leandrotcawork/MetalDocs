package httpdelivery

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/domain"
)

func makePNG(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestImageUploadHandler_AcceptsValidPNG(t *testing.T) {
	handler := newTestImageHandler(t)
	pngBytes := makePNG(t)

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, _ := mw.CreateFormFile("file", "test.png")
	fw.Write(pngBytes)
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/uploads/images", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()

	handler.UploadImage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestImageUploadHandler_RejectsTextFile(t *testing.T) {
	handler := newTestImageHandler(t)

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, _ := mw.CreateFormFile("file", "test.png")
	fw.Write([]byte("this is not an image"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/uploads/images", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()

	handler.UploadImage(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d", rec.Code)
	}
}

// --- test helpers ---

type fakeImageStorage struct {
	mu     sync.Mutex
	byID   map[uuid.UUID][]byte
	byHash map[string]uuid.UUID
	mimes  map[uuid.UUID]string
}

func newFakeImageStorage() *fakeImageStorage {
	return &fakeImageStorage{
		byID:   make(map[uuid.UUID][]byte),
		byHash: make(map[string]uuid.UUID),
		mimes:  make(map[uuid.UUID]string),
	}
}

func (f *fakeImageStorage) Put(ctx context.Context, sha string, mime string, b []byte) (uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id, ok := f.byHash[sha]; ok {
		return id, nil
	}
	id := uuid.New()
	f.byID[id] = b
	f.byHash[sha] = id
	f.mimes[id] = mime
	return id, nil
}

func (f *fakeImageStorage) Get(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.byID[id]
	if !ok {
		return nil, "", domain.ErrImageNotFound
	}
	return b, f.mimes[id], nil
}

func (f *fakeImageStorage) Delete(ctx context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byID, id)
	delete(f.mimes, id)
	return nil
}

func (f *fakeImageStorage) Exists(ctx context.Context, sha string) (uuid.UUID, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.byHash[sha]
	return id, ok, nil
}

func newTestImageHandler(t *testing.T) *ImageHandler {
	t.Helper()
	return NewImageHandler(newFakeImageStorage())
}
