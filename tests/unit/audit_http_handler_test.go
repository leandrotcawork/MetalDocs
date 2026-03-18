package unit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	auditapp "metaldocs/internal/modules/audit/application"
	auditdelivery "metaldocs/internal/modules/audit/delivery/http"
	auditdomain "metaldocs/internal/modules/audit/domain"
	auditmemory "metaldocs/internal/modules/audit/infrastructure/memory"
)

func TestListAuditEventsByDocument(t *testing.T) {
	store := auditmemory.NewWriter()
	svc := auditapp.NewService(store)
	h := auditdelivery.NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	_ = store.Record(context.Background(), auditdomain.Event{
		ID:           "evt-1",
		OccurredAt:   now.Add(-time.Minute),
		ActorID:      "editor-1",
		Action:       "workflow.transitioned",
		ResourceType: "document",
		ResourceID:   "doc-1",
		PayloadJSON:  `{"to_status":"IN_REVIEW"}`,
		TraceID:      "trace-1",
	})
	_ = store.Record(context.Background(), auditdomain.Event{
		ID:           "evt-2",
		OccurredAt:   now,
		ActorID:      "reviewer-1",
		Action:       "workflow.transitioned",
		ResourceType: "document",
		ResourceID:   "doc-2",
		PayloadJSON:  `{"to_status":"APPROVED"}`,
		TraceID:      "trace-2",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?resourceType=document&resourceId=doc-1&limit=10", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"resourceId":"doc-1"`) {
		t.Fatalf("expected doc-1 in body: %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"resourceId":"doc-2"`) {
		t.Fatalf("did not expect doc-2 in body: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"to_status":"IN_REVIEW"`) {
		t.Fatalf("expected payload to be expanded: %s", rr.Body.String())
	}
}
