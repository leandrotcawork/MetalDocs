package httpdelivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apiv2 "metaldocs/internal/api/v2"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type fakeUserAreaWriteRepository struct{}

func (f fakeUserAreaWriteRepository) ListActive(ctx context.Context, userID, tenantID string, now time.Time) ([]iamdomain.UserProcessArea, error) {
	return nil, nil
}

func (f fakeUserAreaWriteRepository) Insert(ctx context.Context, membership iamdomain.UserProcessArea) error {
	return nil
}

func (f fakeUserAreaWriteRepository) CloseActive(ctx context.Context, userID, tenantID, areaCode string, effectiveTo time.Time) error {
	return nil
}

func (f fakeUserAreaWriteRepository) GrantAtomic(ctx context.Context, oldMembership, newMembership iamdomain.UserProcessArea) error {
	return nil
}

func (f fakeUserAreaWriteRepository) GetActiveByUserAndArea(ctx context.Context, userID, tenantID, areaCode string, now time.Time) (*iamdomain.UserProcessArea, error) {
	return nil, nil
}

func TestMembershipsHandler_ErrorEnvelopeContract(t *testing.T) {
	svc := iamapp.NewAreaMembershipService(fakeUserAreaWriteRepository{}, nil)
	handler := NewMembershipHandler(svc)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodDelete, "/api/v2/iam/area-memberships?userId=user-1&areaCode=ops&revokedBy=attacker", nil)
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "session-user", nil))
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
