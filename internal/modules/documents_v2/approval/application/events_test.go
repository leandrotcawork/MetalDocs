package application

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMemoryEmitter(t *testing.T) {
	em := &MemoryEmitter{}
	ev := GovernanceEvent{
		TenantID: "t1", EventType: "doc_submitted",
		ActorUserID: "u1", ResourceType: "document_v2",
		ResourceID: "doc-1", Reason: "approval submit",
		PayloadJSON: json.RawMessage(`{"stage":1}`),
	}
	if err := em.Emit(context.Background(), nil, ev); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if len(em.Events) != 1 {
		t.Fatalf("want 1 event; got %d", len(em.Events))
	}
	if em.Events[0].EventType != "doc_submitted" {
		t.Error("EventType mismatch")
	}
}
