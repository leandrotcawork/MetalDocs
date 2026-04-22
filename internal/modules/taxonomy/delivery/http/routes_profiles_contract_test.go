package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiv2 "metaldocs/internal/api/v2"
	"metaldocs/internal/modules/taxonomy/domain"
)

type fakeProfileService struct{}

func (f fakeProfileService) List(ctx context.Context, tenantID string, includeArchived bool) ([]domain.DocumentProfile, error) {
	return nil, nil
}

func (f fakeProfileService) Get(ctx context.Context, tenantID, code string) (*domain.DocumentProfile, error) {
	return nil, domain.ErrProfileNotFound
}

func (f fakeProfileService) Create(ctx context.Context, p *domain.DocumentProfile) error {
	return nil
}

func (f fakeProfileService) Update(ctx context.Context, p *domain.DocumentProfile) error {
	return nil
}

func (f fakeProfileService) SetDefaultTemplate(ctx context.Context, tenantID, profileCode, templateVersionID, actorID string) error {
	return nil
}

func (f fakeProfileService) Archive(ctx context.Context, tenantID, profileCode, actorID string) error {
	return nil
}

func TestProfilesHandler_ErrorEnvelopeContract(t *testing.T) {
	handler := &Handler{profiles: fakeProfileService{}}
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/taxonomy/profiles/missing", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var apiErr apiv2.APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("unmarshal api error: %v body=%s", err, rec.Body.String())
	}
	if apiErr.Code == "" {
		t.Fatalf("expected non-empty code in API error: %s", rec.Body.String())
	}
}
