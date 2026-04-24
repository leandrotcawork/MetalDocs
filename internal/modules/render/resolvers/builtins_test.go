package resolvers

import "testing"

func TestRegisterBuiltins_RegistersAllV1Resolvers(t *testing.T) {
	r := NewRegistry()
	RegisterBuiltins(r)

	known := r.Known()
	if len(known) != 7 {
		t.Fatalf("expected 7 resolvers, got %d", len(known))
	}

	expected := []string{
		"doc_code",
		"revision_number",
		"effective_date",
		"controlled_by_area",
		"author",
		"approvers",
		"approval_date",
	}
	for _, key := range expected {
		if known[key] != 1 {
			t.Fatalf("expected resolver %s to be version 1, got %d", key, known[key])
		}
	}
}
