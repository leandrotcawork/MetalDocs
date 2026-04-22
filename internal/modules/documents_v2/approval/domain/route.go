package domain

import (
	"errors"
	"fmt"
)

// QuorumPolicy defines how many signoffs satisfy a stage.
type QuorumPolicy string

const (
	QuorumAny1Of QuorumPolicy = "any_1_of"
	QuorumAllOf  QuorumPolicy = "all_of"
	QuorumMofN   QuorumPolicy = "m_of_n"
)

// DriftPolicy defines behavior when eligible actors change after stage opens.
type DriftPolicy string

const (
	DriftReduceQuorum DriftPolicy = "reduce_quorum"
	DriftFailStage    DriftPolicy = "fail_stage"
	DriftKeepSnapshot DriftPolicy = "keep_snapshot"
)

// Stage is a single step in an approval route.
type Stage struct {
	Order              int
	Name               string
	RequiredRole       string
	RequiredCapability string
	AreaCode           string
	Quorum             QuorumPolicy
	QuorumM            *int
	OnEligibilityDrift DriftPolicy
}

// Route is the per-profile approval route configuration.
type Route struct {
	ID          string
	TenantID    string
	ProfileCode string
	Version     int
	Stages      []Stage
}

// Validate enforces route structural invariants.
func (r Route) Validate() error {
	if len(r.Stages) == 0 {
		return errors.New("route must have at least one stage")
	}

	names := make(map[string]bool, len(r.Stages))
	for i, s := range r.Stages {
		// Dense order starting at 1.
		if s.Order != i+1 {
			return fmt.Errorf("stage order must be dense starting at 1: stage at index %d has order %d, expected %d", i, s.Order, i+1)
		}

		// Quorum + QuorumM consistency.
		if s.Quorum == QuorumMofN {
			if s.QuorumM == nil {
				return fmt.Errorf("stage %q: quorum m_of_n requires QuorumM", s.Name)
			}
			if *s.QuorumM < 1 {
				return fmt.Errorf("stage %q: QuorumM must be >= 1", s.Name)
			}
		} else {
			if s.QuorumM != nil {
				return fmt.Errorf("stage %q: QuorumM must be nil for quorum %s", s.Name, s.Quorum)
			}
		}

		// Unique names.
		if names[s.Name] {
			return fmt.Errorf("duplicate stage name %q in route", s.Name)
		}
		names[s.Name] = true
	}
	return nil
}
