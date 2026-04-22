package domain

import "testing"

func driftStage(eligible []string, policy DriftPolicy) StageInstance {
	return StageInstance{
		Status:                     StageActive,
		EligibleActorIDs:           eligible,
		OnEligibilityDriftSnapshot: policy,
		QuorumSnapshot:             QuorumAny1Of,
	}
}

func TestDriftReduceQuorumNoDrift(t *testing.T) {
	st := driftStage([]string{"a", "b", "c"}, DriftReduceQuorum)
	r := ApplyEligibilityDrift(st, []string{"a", "b", "c"})
	if r.EffectiveDenominator != 3 {
		t.Errorf("want 3; got %d", r.EffectiveDenominator)
	}
	if r.ForcedOutcome != QuorumPending {
		t.Errorf("want pending; got %s", r.ForcedOutcome)
	}
}

func TestDriftReduceQuorumMinorDrift(t *testing.T) {
	st := driftStage([]string{"a", "b", "c"}, DriftReduceQuorum)
	r := ApplyEligibilityDrift(st, []string{"a", "b"}) // c departed
	if r.EffectiveDenominator != 2 {
		t.Errorf("want 2; got %d", r.EffectiveDenominator)
	}
	if r.ForcedOutcome != QuorumPending {
		t.Errorf("want pending force; got %s", r.ForcedOutcome)
	}
}

func TestDriftReduceQuorumTotalDrift(t *testing.T) {
	st := driftStage([]string{"a", "b"}, DriftReduceQuorum)
	r := ApplyEligibilityDrift(st, []string{"x"}) // all departed
	if r.EffectiveDenominator != 0 {
		t.Errorf("want 0; got %d", r.EffectiveDenominator)
	}
	// reduce_quorum leaves ForcedOutcome=Pending; caller uses zero denom for rejection.
	if r.ForcedOutcome != QuorumPending {
		t.Errorf("want pending; got %s", r.ForcedOutcome)
	}
}

func TestDriftFailStageNoDrift(t *testing.T) {
	st := driftStage([]string{"a", "b"}, DriftFailStage)
	r := ApplyEligibilityDrift(st, []string{"a", "b", "c"}) // new person added — ok
	if r.ForcedOutcome != QuorumPending {
		t.Errorf("no departure: want pending; got %s", r.ForcedOutcome)
	}
	if r.EffectiveDenominator != 2 {
		t.Errorf("want snapshot count 2; got %d", r.EffectiveDenominator)
	}
}

func TestDriftFailStageTotalDrift(t *testing.T) {
	st := driftStage([]string{"a", "b"}, DriftFailStage)
	r := ApplyEligibilityDrift(st, nil) // all departed
	if r.ForcedOutcome != QuorumRejectedStage {
		t.Errorf("want rejected_stage; got %s", r.ForcedOutcome)
	}
}

func TestDriftFailStageMinorDrift(t *testing.T) {
	st := driftStage([]string{"a", "b", "c"}, DriftFailStage)
	r := ApplyEligibilityDrift(st, []string{"a", "b"}) // c departed
	if r.ForcedOutcome != QuorumRejectedStage {
		t.Errorf("want rejected_stage on any departure; got %s", r.ForcedOutcome)
	}
}

func TestDriftKeepSnapshotIgnoresCurrent(t *testing.T) {
	st := driftStage([]string{"a", "b", "c"}, DriftKeepSnapshot)
	r := ApplyEligibilityDrift(st, []string{"x", "y"}) // completely different
	if r.EffectiveDenominator != 3 {
		t.Errorf("keep_snapshot: want snapshot count 3; got %d", r.EffectiveDenominator)
	}
	if r.ForcedOutcome != QuorumPending {
		t.Errorf("want pending; got %s", r.ForcedOutcome)
	}
}
