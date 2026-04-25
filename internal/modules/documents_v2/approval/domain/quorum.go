package domain

// QuorumOutcome is the result of evaluating quorum for a stage.
type QuorumOutcome string

const (
	QuorumPending       QuorumOutcome = "pending"
	QuorumApprovedStage QuorumOutcome = "approved_stage"
	QuorumRejectedStage QuorumOutcome = "rejected_stage"
)

// ComputeEffectiveDenominator intersects the stage's snapshot eligible set with the
// current eligible set, returning the count. Callers never hand-compute this.
func ComputeEffectiveDenominator(stage StageInstance, currentEligible []string) int {
	if len(stage.EligibleActorIDs) == 0 || len(currentEligible) == 0 {
		return 0
	}
	snapshot := make(map[string]bool, len(stage.EligibleActorIDs))
	for _, id := range stage.EligibleActorIDs {
		snapshot[id] = true
	}
	count := 0
	for _, id := range currentEligible {
		if snapshot[id] {
			count++
		}
	}
	return count
}

// EvaluateQuorum evaluates whether signoffs satisfy the stage's quorum policy.
// Signoffs from actors NOT in EligibleActorIDs are ignored.
func EvaluateQuorum(stage StageInstance, approvals []Signoff, rejections []Signoff, effectiveDenominator int) QuorumOutcome {
	if effectiveDenominator == 0 {
		return QuorumRejectedStage
	}

	approveCount := 0
	rejectCount := 0

	if len(stage.EligibleActorIDs) == 0 {
		// No eligible set configured — all signoffs count (matches effectiveDenominator=1 fallback).
		approveCount = len(approvals)
		rejectCount = len(rejections)
	} else {
		eligible := make(map[string]bool, len(stage.EligibleActorIDs))
		for _, id := range stage.EligibleActorIDs {
			eligible[id] = true
		}
		for _, s := range approvals {
			if eligible[s.ActorUserID()] {
				approveCount++
			}
		}
		for _, s := range rejections {
			if eligible[s.ActorUserID()] {
				rejectCount++
			}
		}
	}

	switch stage.QuorumSnapshot {
	case QuorumAny1Of:
		if approveCount >= 1 {
			return QuorumApprovedStage
		}
		if rejectCount >= 1 {
			return QuorumRejectedStage
		}
		return QuorumPending

	case QuorumAllOf:
		if rejectCount >= 1 {
			return QuorumRejectedStage
		}
		if approveCount >= effectiveDenominator {
			return QuorumApprovedStage
		}
		return QuorumPending

	case QuorumMofN:
		m := 1
		if stage.QuorumMSnapshot != nil {
			m = *stage.QuorumMSnapshot
		}
		if approveCount >= m {
			return QuorumApprovedStage
		}
		// Rejected when not enough approvals possible: rejections > denom - m
		if rejectCount > effectiveDenominator-m {
			return QuorumRejectedStage
		}
		return QuorumPending
	}

	return QuorumPending
}
