package httpdelivery

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func requireAPIError(t testing.TB, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d", rec.Code, wantStatus)
	}

	var envelope apiErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Error.Code != wantCode {
		t.Fatalf("code = %q, want %q", envelope.Error.Code, wantCode)
	}
}

type fakePdfRenderer struct {
	lastHTML []byte
	lastCSS  []byte
	result   []byte
	err      error
}

func (f *fakePdfRenderer) ConvertHTMLToPDF(_ context.Context, html []byte, css []byte) ([]byte, error) {
	f.lastHTML = html
	f.lastCSS = css
	return f.result, f.err
}
