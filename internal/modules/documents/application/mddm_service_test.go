package application

import (
	"testing"
)

func TestReleaseApprovalSequence_ArchivePreviousBeforePromote(t *testing.T) {
	// This is a logic-only test. The real integration test lives in postgres/.
	// Here we verify the service builds the correct sequence of operations.
	steps := planReleaseSteps("doc-1", "draft-id", "prev-released-id")
	if len(steps) < 4 {
		t.Fatalf("expected at least 4 steps, got %d", len(steps))
	}
	if steps[0] != "archive_previous_released" {
		t.Errorf("step 0 must archive previous, got %s", steps[0])
	}
	if steps[1] != "promote_draft_to_released" {
		t.Errorf("step 1 must promote draft, got %s", steps[1])
	}
}
