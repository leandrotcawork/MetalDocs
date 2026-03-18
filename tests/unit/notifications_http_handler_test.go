package unit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	iamdomain "metaldocs/internal/modules/iam/domain"
	notificationapp "metaldocs/internal/modules/notifications/application"
	notificationdelivery "metaldocs/internal/modules/notifications/delivery/http"
	notificationdomain "metaldocs/internal/modules/notifications/domain"
	notificationmemory "metaldocs/internal/modules/notifications/infrastructure/memory"
)

func TestListNotificationsForCurrentUser(t *testing.T) {
	repo := notificationmemory.NewRepository()
	svc := notificationapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)})
	h := notificationdelivery.NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	_ = repo.Create(context.Background(), notificationdomain.Notification{
		ID:              "notif-1",
		RecipientUserID: "user-1",
		EventType:       "workflow.approval.requested",
		ResourceType:    "document",
		ResourceID:      "doc-1",
		Title:           "Approval requested",
		Message:         "Document doc-1 needs review.",
		Status:          notificationdomain.StatusPending,
		IdempotencyKey:  "notif-1",
		CreatedAt:       time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
	})
	_ = repo.Create(context.Background(), notificationdomain.Notification{
		ID:              "notif-2",
		RecipientUserID: "user-2",
		EventType:       "workflow.approval.decisioned",
		ResourceType:    "document",
		ResourceID:      "doc-2",
		Title:           "Approval decided",
		Message:         "Document doc-2 was approved.",
		Status:          notificationdomain.StatusPending,
		IdempotencyKey:  "notif-2",
		CreatedAt:       time.Date(2026, 3, 21, 11, 0, 0, 0, time.UTC),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=10", nil)
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-1", []iamdomain.Role{iamdomain.RoleViewer}))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"id":"notif-1"`) {
		t.Fatalf("expected notif-1 in body: %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"id":"notif-2"`) {
		t.Fatalf("did not expect notif-2 in body: %s", rr.Body.String())
	}
}

func TestMarkNotificationRead(t *testing.T) {
	repo := notificationmemory.NewRepository()
	svc := notificationapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)})
	h := notificationdelivery.NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	_ = repo.Create(context.Background(), notificationdomain.Notification{
		ID:              "notif-read",
		RecipientUserID: "user-1",
		EventType:       "workflow.approval.requested",
		ResourceType:    "document",
		ResourceID:      "doc-1",
		Title:           "Approval requested",
		Message:         "Document doc-1 needs review.",
		Status:          notificationdomain.StatusPending,
		IdempotencyKey:  "notif-read",
		CreatedAt:       time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/notif-read/read", nil)
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-1", []iamdomain.Role{iamdomain.RoleViewer}))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	items := repo.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(items))
	}
	if items[0].Status != notificationdomain.StatusRead {
		t.Fatalf("expected READ status, got %s", items[0].Status)
	}
	if items[0].ReadAt == nil {
		t.Fatalf("expected read_at to be set")
	}
}
