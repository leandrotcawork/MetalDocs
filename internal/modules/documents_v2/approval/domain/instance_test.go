package domain

import (
	"errors"
	"testing"
)

func threeStageInstance() Instance {
	return Instance{
		ID: "inst-1", TenantID: "t1", DocumentID: "doc-1",
		Status:          InstanceInProgress,
		RevisionVersion: 0,
		Stages: []StageInstance{
			{ID: "s1", StageOrder: 1, Status: StageActive, NameSnapshot: "QA"},
			{ID: "s2", StageOrder: 2, Status: StagePending, NameSnapshot: "Manager"},
			{ID: "s3", StageOrder: 3, Status: StagePending, NameSnapshot: "Director"},
		},
	}
}

func TestInstanceActive(t *testing.T) {
	inst := threeStageInstance()
	a := inst.Active()
	if a == nil || a.ID != "s1" {
		t.Fatalf("Active() = %v; want s1", a)
	}
}

func TestInstanceAdvanceStage(t *testing.T) {
	inst := threeStageInstance()

	// Advance 1→2.
	if err := inst.AdvanceStage(); err != nil {
		t.Fatalf("AdvanceStage: %v", err)
	}
	if inst.Stages[0].Status != StageCompleted {
		t.Error("stage 1 should be completed")
	}
	if inst.Stages[1].Status != StageActive {
		t.Error("stage 2 should be active")
	}
	if inst.Status != InstanceInProgress {
		t.Error("instance should still be in_progress")
	}

	// Advance 2→3.
	if err := inst.AdvanceStage(); err != nil {
		t.Fatalf("AdvanceStage: %v", err)
	}
	if inst.Stages[2].Status != StageActive {
		t.Error("stage 3 should be active")
	}

	// Advance 3→approved.
	if err := inst.AdvanceStage(); err != nil {
		t.Fatalf("AdvanceStage: %v", err)
	}
	if inst.Status != InstanceApproved {
		t.Errorf("instance should be approved; got %s", inst.Status)
	}
	if inst.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestInstanceAdvanceNoActive(t *testing.T) {
	inst := Instance{Status: InstanceInProgress, Stages: []StageInstance{
		{Status: StagePending},
	}}
	if err := inst.AdvanceStage(); !errors.Is(err, ErrNoActiveStage) {
		t.Errorf("want ErrNoActiveStage; got %v", err)
	}
}

func TestInstanceRejectHere(t *testing.T) {
	inst := threeStageInstance()
	if err := inst.RejectHere("quality failure"); err != nil {
		t.Fatalf("RejectHere: %v", err)
	}
	if inst.Status != InstanceRejected {
		t.Errorf("want InstanceRejected; got %s", inst.Status)
	}
	if inst.Stages[0].Status != StageRejectedHere {
		t.Error("stage 1 should be rejected_here")
	}
	if inst.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestInstanceSkipStage(t *testing.T) {
	inst := threeStageInstance()
	if err := inst.SkipStage("exempted"); err != nil {
		t.Fatalf("SkipStage: %v", err)
	}
	if inst.Stages[0].Status != StageSkipped {
		t.Error("stage 1 should be skipped")
	}
	if inst.Stages[1].Status != StageActive {
		t.Error("stage 2 should be active after skip")
	}
}

func TestInstanceSkipLastStage(t *testing.T) {
	inst := Instance{Status: InstanceInProgress, Stages: []StageInstance{
		{Status: StageActive},
	}}
	if err := inst.SkipStage("skip"); !errors.Is(err, ErrCannotSkipLastStage) {
		t.Errorf("want ErrCannotSkipLastStage; got %v", err)
	}
}

func TestInstanceBumpRevisionVersion(t *testing.T) {
	inst := threeStageInstance()

	if err := inst.BumpRevisionVersion(1); err != nil {
		t.Fatalf("BumpRevisionVersion(1): %v", err)
	}
	if inst.RevisionVersion != 1 {
		t.Errorf("want 1; got %d", inst.RevisionVersion)
	}

	// Regression.
	if err := inst.BumpRevisionVersion(0); !errors.Is(err, ErrRevisionRegression) {
		t.Errorf("want ErrRevisionRegression; got %v", err)
	}

	// No-op equal.
	if err := inst.BumpRevisionVersion(1); err != nil {
		t.Errorf("same version should be no-op; got %v", err)
	}
	if inst.RevisionVersion != 1 {
		t.Error("version should remain 1")
	}
}

func TestInstanceCancel(t *testing.T) {
	inst := threeStageInstance()
	if err := inst.Cancel("admin revoke"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if inst.Status != InstanceCancelled {
		t.Errorf("want InstanceCancelled; got %s", inst.Status)
	}
	if inst.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}

	// Re-cancel → ErrInstanceTerminal.
	if err := inst.Cancel("again"); !errors.Is(err, ErrInstanceTerminal) {
		t.Errorf("want ErrInstanceTerminal; got %v", err)
	}
}

func TestInstanceCancelAfterApproved(t *testing.T) {
	inst := Instance{Status: InstanceApproved}
	if err := inst.Cancel("oops"); !errors.Is(err, ErrInstanceTerminal) {
		t.Errorf("want ErrInstanceTerminal; got %v", err)
	}
}
