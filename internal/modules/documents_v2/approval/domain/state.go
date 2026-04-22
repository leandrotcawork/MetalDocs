package domain

import "errors"

// ErrLegacyStateRejected returned when legacy state string (finalized, archived) parsed.
var ErrLegacyStateRejected = errors.New("legacy document state is not valid in the Spec 2 approval graph")

// DocState represents the 8-state Spec 2 document lifecycle.
type DocState string

const (
	StateDraft       DocState = "draft"
	StateUnderReview DocState = "under_review"
	StateApproved    DocState = "approved"
	StateRejected    DocState = "rejected"
	StateScheduled   DocState = "scheduled"
	StatePublished   DocState = "published"
	StateSuperseded  DocState = "superseded"
	StateObsolete    DocState = "obsolete"
)

// AllStates returns all 8 valid states for exhaustive test matrices.
func AllStates() []DocState {
	return []DocState{
		StateDraft, StateUnderReview, StateApproved, StateRejected,
		StateScheduled, StatePublished, StateSuperseded, StateObsolete,
	}
}

// String returns the canonical lowercase snake_case form.
func (s DocState) String() string { return string(s) }

// StateFromString parses a string into a DocState.
// Returns ErrLegacyStateRejected for "finalized" or "archived".
func StateFromString(s string) (DocState, error) {
	switch DocState(s) {
	case StateDraft, StateUnderReview, StateApproved, StateRejected,
		StateScheduled, StatePublished, StateSuperseded, StateObsolete:
		return DocState(s), nil
	case "finalized", "archived":
		return "", ErrLegacyStateRejected
	default:
		return "", errors.New("unknown document state: " + s)
	}
}

// legalTransitions encodes the full Spec 2 directed graph.
var legalTransitions = map[DocState]map[DocState]bool{
	StateDraft:       {StateUnderReview: true},
	StateUnderReview: {StateApproved: true, StateRejected: true},
	StateRejected:    {StateDraft: true},
	StateApproved:    {StatePublished: true, StateScheduled: true, StateDraft: true},
	StateScheduled:   {StatePublished: true, StateDraft: true},
	StatePublished:   {StateSuperseded: true, StateObsolete: true},
	StateSuperseded:  {StateObsolete: true},
	StateObsolete:    {},
}

// IsLegalTransition returns true only for edges in Spec 2 graph.
// Self-transitions and legacy states always false.
func IsLegalTransition(from, to DocState) bool {
	if from == to {
		return false
	}
	targets, ok := legalTransitions[from]
	if !ok {
		return false
	}
	return targets[to]
}
