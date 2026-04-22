package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func makeSignoff(actor string, dec Decision) Signoff {
	h := strings.Repeat("a", 64)
	s, _ := NewSignoff(SignoffParams{
		ID: actor, ApprovalInstanceID: "i1", StageInstanceID: "st1",
		ActorUserID: actor, ActorTenantID: "t1",
		Decision: dec, SignedAt: time.Now(),
		SignatureMethod: "password", SignaturePayload: json.RawMessage(`{}`),
		ContentHash: h,
	})
	return *s
}

func stageWithEligible(eligible []string, q QuorumPolicy, m *int) StageInstance {
	return StageInstance{
		Status:          StageActive,
		QuorumSnapshot:  q,
		QuorumMSnapshot: m,
		EligibleActorIDs: eligible,
	}
}

func TestComputeEffectiveDenominator(t *testing.T) {
	stage := stageWithEligible([]string{"a", "b", "c"}, QuorumAny1Of, nil)

	// Intersection a,b from current [a,b] → 2.
	if n := ComputeEffectiveDenominator(stage, []string{"a", "b"}); n != 2 {
		t.Errorf("want 2; got %d", n)
	}
	// No intersection → 0.
	if n := ComputeEffectiveDenominator(stage, []string{"d"}); n != 0 {
		t.Errorf("want 0; got %d", n)
	}
	// Empty snapshot → 0.
	empty := stageWithEligible(nil, QuorumAny1Of, nil)
	if n := ComputeEffectiveDenominator(empty, []string{"a"}); n != 0 {
		t.Errorf("empty snapshot: want 0; got %d", n)
	}
	// Nil current → 0.
	if n := ComputeEffectiveDenominator(stage, nil); n != 0 {
		t.Errorf("nil current: want 0; got %d", n)
	}
}

func TestQuorumAny1Of(t *testing.T) {
	st := stageWithEligible([]string{"a", "b"}, QuorumAny1Of, nil)

	// 0 signoffs → pending.
	if o := EvaluateQuorum(st, nil, nil, 2); o != QuorumPending {
		t.Errorf("want pending; got %s", o)
	}
	// First approval → approved.
	if o := EvaluateQuorum(st, []Signoff{makeSignoff("a", DecisionApprove)}, nil, 2); o != QuorumApprovedStage {
		t.Errorf("want approved; got %s", o)
	}
	// First rejection → rejected.
	if o := EvaluateQuorum(st, nil, []Signoff{makeSignoff("a", DecisionReject)}, 2); o != QuorumRejectedStage {
		t.Errorf("want rejected; got %s", o)
	}
}

func TestQuorumAllOf(t *testing.T) {
	st := stageWithEligible([]string{"a", "b"}, QuorumAllOf, nil)

	// 1 of 2 approved → pending.
	if o := EvaluateQuorum(st, []Signoff{makeSignoff("a", DecisionApprove)}, nil, 2); o != QuorumPending {
		t.Errorf("want pending; got %s", o)
	}
	// Both approved → approved.
	if o := EvaluateQuorum(st, []Signoff{makeSignoff("a", DecisionApprove), makeSignoff("b", DecisionApprove)}, nil, 2); o != QuorumApprovedStage {
		t.Errorf("want approved; got %s", o)
	}
	// Any rejection → rejected.
	if o := EvaluateQuorum(st, nil, []Signoff{makeSignoff("a", DecisionReject)}, 2); o != QuorumRejectedStage {
		t.Errorf("want rejected; got %s", o)
	}
}

func TestQuorumMofN(t *testing.T) {
	m := 2
	st := stageWithEligible([]string{"a", "b", "c"}, QuorumMofN, &m)

	// 1 approval → pending.
	if o := EvaluateQuorum(st, []Signoff{makeSignoff("a", DecisionApprove)}, nil, 3); o != QuorumPending {
		t.Errorf("want pending; got %s", o)
	}
	// 2 approvals → approved.
	if o := EvaluateQuorum(st, []Signoff{makeSignoff("a", DecisionApprove), makeSignoff("b", DecisionApprove)}, nil, 3); o != QuorumApprovedStage {
		t.Errorf("want approved; got %s", o)
	}
	// M=2, N=3: 1 approve + 2 reject → rejected (rejections > denom-m = 1).
	if o := EvaluateQuorum(st, []Signoff{makeSignoff("a", DecisionApprove)}, []Signoff{makeSignoff("b", DecisionReject), makeSignoff("c", DecisionReject)}, 3); o != QuorumRejectedStage {
		t.Errorf("want rejected_stage; got %s", o)
	}
}

func TestQuorumZeroDenominator(t *testing.T) {
	st := stageWithEligible([]string{"a"}, QuorumAny1Of, nil)
	if o := EvaluateQuorum(st, nil, nil, 0); o != QuorumRejectedStage {
		t.Errorf("zero denom: want rejected; got %s", o)
	}
}

func TestQuorumIneligibleActorIgnored(t *testing.T) {
	st := stageWithEligible([]string{"a"}, QuorumAny1Of, nil)
	// "outsider" not in eligible list.
	if o := EvaluateQuorum(st, []Signoff{makeSignoff("outsider", DecisionApprove)}, nil, 1); o != QuorumPending {
		t.Errorf("ineligible actor vote should be ignored; got %s", o)
	}
}
