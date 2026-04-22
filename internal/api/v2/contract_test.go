package apiv2_test

import (
	"encoding/json"
	"reflect"
	"testing"

	apiv2 "metaldocs/internal/api/v2"
)

func TestAPIError_MarshalJSON(t *testing.T) {
	original := apiv2.APIError{
		Code:    "VALIDATION_ERROR",
		Message: "invalid payload",
		Details: map[string]any{"field": "code"},
		TraceID: "trace-123",
	}

	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal api error: %v", err)
	}

	var decoded apiv2.APIError
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal api error: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round-trip mismatch: original=%+v decoded=%+v", original, decoded)
	}
}

func TestProfileResponse_MarshalJSON(t *testing.T) {
	defaultTemplateVersionID := "tpl-v1"
	ownerUserID := "user-1"
	archivedAt := "2026-04-21T12:00:00Z"
	original := apiv2.ProfileResponse{
		Code:                     "po",
		TenantID:                 "tenant-a",
		FamilyCode:               "ops",
		Name:                     "Purchase Order",
		Description:              "Controlled purchase order profile",
		ReviewIntervalDays:       365,
		DefaultTemplateVersionID: &defaultTemplateVersionID,
		OwnerUserID:              &ownerUserID,
		EditableByRole:           "editor",
		ArchivedAt:               &archivedAt,
		CreatedAt:                "2026-04-21T10:00:00Z",
	}

	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal profile response: %v", err)
	}

	var decoded apiv2.ProfileResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal profile response: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round-trip mismatch: original=%+v decoded=%+v", original, decoded)
	}
}
