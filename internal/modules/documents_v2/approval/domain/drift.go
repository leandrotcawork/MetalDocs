package domain

// DriftResult holds the output of ApplyEligibilityDrift.
type DriftResult struct {
	EffectiveDenominator int
	ForcedOutcome        QuorumOutcome // QuorumPending means "no force"
	Reason               string
}

// ApplyEligibilityDrift applies the stage's drift policy given the current eligible set.
// Pure function — no DB, no time source.
func ApplyEligibilityDrift(stage StageInstance, currentEligible []string) DriftResult {
	snapshot := make(map[string]bool, len(stage.EligibleActorIDs))
	for _, id := range stage.EligibleActorIDs {
		snapshot[id] = true
	}

	current := make(map[string]bool, len(currentEligible))
	for _, id := range currentEligible {
		current[id] = true
	}

	switch stage.OnEligibilityDriftSnapshot {
	case DriftReduceQuorum:
		// Denominator = |snapshot ∩ current|.
		denom := 0
		for _, id := range stage.EligibleActorIDs {
			if current[id] {
				denom++
			}
		}
		delta := len(stage.EligibleActorIDs) - denom
		reason := ""
		if delta > 0 {
			reason = "eligibility drift: reduce_quorum policy, denominator reduced by " + itoa(delta)
		}
		return DriftResult{EffectiveDenominator: denom, ForcedOutcome: QuorumPending, Reason: reason}

	case DriftFailStage:
		// If any snapshot actor departed → forced rejection.
		for _, id := range stage.EligibleActorIDs {
			if !current[id] {
				return DriftResult{
					EffectiveDenominator: 0,
					ForcedOutcome:        QuorumRejectedStage,
					Reason:               "eligibility drift: fail_stage policy",
				}
			}
		}
		// All snapshot actors still eligible.
		return DriftResult{
			EffectiveDenominator: len(stage.EligibleActorIDs),
			ForcedOutcome:        QuorumPending,
			Reason:               "",
		}

	case DriftKeepSnapshot:
		// Ignore current; denominator stays at snapshot count.
		return DriftResult{
			EffectiveDenominator: len(stage.EligibleActorIDs),
			ForcedOutcome:        QuorumPending,
			Reason:               "",
		}

	default:
		return DriftResult{EffectiveDenominator: len(stage.EligibleActorIDs), ForcedOutcome: QuorumPending}
	}
}

// itoa converts int to string without importing strconv in domain.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := [20]byte{}
	pos := len(buf)
	for n >= 10 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	pos--
	buf[pos] = byte('0' + n)
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
