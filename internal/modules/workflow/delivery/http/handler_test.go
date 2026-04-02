package httpdelivery

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleDocumentWorkflowRoutes_MalformedApprovalsPathReturnsNotFound(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workflow/documents//approvals", nil)
	recorder := httptest.NewRecorder()

	handler.handleDocumentWorkflowRoutes(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status mismatch: got %d want %d", recorder.Code, http.StatusNotFound)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"WORKFLOW_ROUTE_NOT_FOUND"`) {
		t.Fatalf("expected WORKFLOW_ROUTE_NOT_FOUND error code, got body=%s", recorder.Body.String())
	}
}
