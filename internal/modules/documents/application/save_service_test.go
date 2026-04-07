package application

import (
	"context"
	"encoding/json"
	"testing"

	"metaldocs/internal/modules/documents/domain/mddm"
)

func TestSaveDraftService_RejectsInvalidEnvelope(t *testing.T) {
	svc := NewSaveDraftService(nil, nil, nil, mddm.RulesContext{})
	envelope := json.RawMessage(`{"mddm_version":1}`) // missing blocks/template_ref

	_, err := svc.SaveDraft(context.Background(), SaveDraftInput{
		DocumentID:   "PO-118",
		BaseVersion:  1,
		EnvelopeJSON: envelope,
		UserID:       "user-1",
	})
	if err == nil {
		t.Error("expected validation error, got nil")
	}
}
