package httpdelivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	iamdomain "metaldocs/internal/modules/iam/domain"

	"metaldocs/internal/modules/documents/application"
)

type routeLoadRepo struct{}

func (r *routeLoadRepo) GetActiveDraft(ctx context.Context, documentID, userID string) (*application.LoadVersion, error) {
	return &application.LoadVersion{
		DocumentID:      documentID,
		Version:         1,
		Status:          "draft",
		Content:         json.RawMessage(`{"mddm_version":1,"blocks":[]}`),
		TemplateKey:     "po-default-canvas",
		TemplateVersion: 1,
		ContentHash:     "hash-1",
	}, nil
}

func (r *routeLoadRepo) GetCurrentReleased(ctx context.Context, documentID string) (*application.LoadVersion, error) {
	return nil, nil
}

type routeSubmitRepo struct{}

func (r *routeSubmitRepo) TransitionDraftToPendingApproval(ctx context.Context, draftID uuid.UUID) error {
	return nil
}

func TestHandleDocumentSubRoutes_LoadRouteUsesMDDMHandler(t *testing.T) {
	handler := NewHandler(nil).WithMDDMHandlers(application.NewLoadService(&routeLoadRepo{}), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/PO-118/load", nil)
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-123", nil))
	rec := httptest.NewRecorder()

	handler.handleDocumentSubRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleDocumentSubRoutes_SubmitForApprovalRouteUsesMDDMHandler(t *testing.T) {
	handler := NewHandler(nil).WithMDDMHandlers(nil, application.NewSubmitForApprovalService(&routeSubmitRepo{}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/PO-118/submit-for-approval?draft_id="+uuid.NewString(), nil)
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-123", nil))
	rec := httptest.NewRecorder()

	handler.handleDocumentSubRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
