package domain

import (
	"errors"
	"testing"
)

func TestStateLegalTransitions(t *testing.T) {
	states := AllStates()

	// Build truth table from spec.
	legal := map[[2]DocState]bool{
		{StateDraft, StateUnderReview}:     true,
		{StateUnderReview, StateApproved}:  true,
		{StateUnderReview, StateRejected}:  true,
		{StateRejected, StateDraft}:        true,
		{StateApproved, StatePublished}:    true,
		{StateApproved, StateScheduled}:    true,
		{StateApproved, StateDraft}:        true,
		{StateScheduled, StatePublished}:   true,
		{StateScheduled, StateDraft}:       true,
		{StatePublished, StateSuperseded}:  true,
		{StatePublished, StateObsolete}:    true,
		{StateSuperseded, StateObsolete}:   true,
	}

	for _, from := range states {
		for _, to := range states {
			pair := [2]DocState{from, to}
			want := legal[pair]
			got := IsLegalTransition(from, to)
			if got != want {
				t.Errorf("IsLegalTransition(%s, %s) = %v; want %v", from, to, got, want)
			}
		}
	}
}

func TestStateSelfTransitionsIllegal(t *testing.T) {
	for _, s := range AllStates() {
		if IsLegalTransition(s, s) {
			t.Errorf("self-transition %s->%s should be illegal", s, s)
		}
	}
}

func TestStateLegacyTransitionsIllegal(t *testing.T) {
	// Spec 2 domain rejects finalized/published cross-graph.
	if IsLegalTransition("finalized", StatePublished) {
		t.Error("finalized->published must be illegal at domain level")
	}
}

func TestStateFromString(t *testing.T) {
	for _, s := range AllStates() {
		got, err := StateFromString(string(s))
		if err != nil {
			t.Errorf("StateFromString(%q) unexpected error: %v", s, err)
		}
		if got != s {
			t.Errorf("StateFromString(%q) = %q; want %q", s, got, s)
		}
	}

	// Legacy states must return ErrLegacyStateRejected.
	for _, legacy := range []string{"finalized", "archived"} {
		_, err := StateFromString(legacy)
		if !errors.Is(err, ErrLegacyStateRejected) {
			t.Errorf("StateFromString(%q) should return ErrLegacyStateRejected; got %v", legacy, err)
		}
	}

	// Unknown state.
	_, err := StateFromString("banana")
	if err == nil {
		t.Error("StateFromString(unknown) should error")
	}

	// Empty string.
	_, err = StateFromString("")
	if err == nil {
		t.Error("StateFromString('') should error")
	}
}

func TestStateString(t *testing.T) {
	if StateDraft.String() != "draft" {
		t.Errorf("StateDraft.String() = %q; want 'draft'", StateDraft.String())
	}
	if StateUnderReview.String() != "under_review" {
		t.Errorf("StateUnderReview.String() = %q; want 'under_review'", StateUnderReview.String())
	}
}

func TestStateNamedEdges(t *testing.T) {
	edges := [][2]DocState{
		{StateApproved, StateScheduled},
		{StateScheduled, StatePublished},
		{StateScheduled, StateDraft},
		{StatePublished, StateSuperseded},
		{StatePublished, StateObsolete},
		{StateSuperseded, StateObsolete},
	}
	for _, e := range edges {
		if !IsLegalTransition(e[0], e[1]) {
			t.Errorf("expected legal: %s -> %s", e[0], e[1])
		}
	}
}
