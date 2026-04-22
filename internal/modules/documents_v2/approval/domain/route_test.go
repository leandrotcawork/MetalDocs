package domain

import (
	"testing"
)

func intPtr(n int) *int { return &n }

func happyRoute() Route {
	return Route{
		ID: "r1", TenantID: "t1", ProfileCode: "SOP", Version: 1,
		Stages: []Stage{
			{Order: 1, Name: "QA", RequiredRole: "reviewer", RequiredCapability: "workflow.signoff", AreaCode: "qa", Quorum: QuorumAny1Of, OnEligibilityDrift: DriftReduceQuorum},
			{Order: 2, Name: "Manager", RequiredRole: "manager", RequiredCapability: "workflow.signoff", AreaCode: "mgmt", Quorum: QuorumAllOf, OnEligibilityDrift: DriftKeepSnapshot},
			{Order: 3, Name: "Director", RequiredRole: "director", RequiredCapability: "workflow.signoff", AreaCode: "exec", Quorum: QuorumMofN, QuorumM: intPtr(2), OnEligibilityDrift: DriftFailStage},
		},
	}
}

func TestRouteValidateHappy(t *testing.T) {
	if err := happyRoute().Validate(); err != nil {
		t.Fatalf("happy route invalid: %v", err)
	}
}

func TestRouteValidateEmptyStages(t *testing.T) {
	r := Route{Stages: []Stage{}}
	if err := r.Validate(); err == nil {
		t.Error("empty stages should fail validation")
	}
}

func TestRouteValidateNonDenseOrder(t *testing.T) {
	r := Route{
		Stages: []Stage{
			{Order: 1, Name: "A", Quorum: QuorumAny1Of, OnEligibilityDrift: DriftReduceQuorum},
			{Order: 3, Name: "B", Quorum: QuorumAny1Of, OnEligibilityDrift: DriftReduceQuorum},
		},
	}
	if err := r.Validate(); err == nil {
		t.Error("non-dense order [1,3] should fail validation")
	}
}

func TestRouteValidateMofNWithoutM(t *testing.T) {
	r := Route{
		Stages: []Stage{
			{Order: 1, Name: "A", Quorum: QuorumMofN, OnEligibilityDrift: DriftReduceQuorum},
		},
	}
	if err := r.Validate(); err == nil {
		t.Error("m_of_n without QuorumM should fail")
	}
}

func TestRouteValidateAny1OfWithM(t *testing.T) {
	r := Route{
		Stages: []Stage{
			{Order: 1, Name: "A", Quorum: QuorumAny1Of, QuorumM: intPtr(1), OnEligibilityDrift: DriftReduceQuorum},
		},
	}
	if err := r.Validate(); err == nil {
		t.Error("any_1_of with QuorumM set should fail")
	}
}

func TestRouteValidateMofNZeroM(t *testing.T) {
	r := Route{
		Stages: []Stage{
			{Order: 1, Name: "A", Quorum: QuorumMofN, QuorumM: intPtr(0), OnEligibilityDrift: DriftReduceQuorum},
		},
	}
	if err := r.Validate(); err == nil {
		t.Error("QuorumM=0 should fail")
	}
}

func TestRouteValidateDuplicateNames(t *testing.T) {
	r := Route{
		Stages: []Stage{
			{Order: 1, Name: "Dup", Quorum: QuorumAny1Of, OnEligibilityDrift: DriftReduceQuorum},
			{Order: 2, Name: "Dup", Quorum: QuorumAny1Of, OnEligibilityDrift: DriftReduceQuorum},
		},
	}
	if err := r.Validate(); err == nil {
		t.Error("duplicate stage names should fail")
	}
}
