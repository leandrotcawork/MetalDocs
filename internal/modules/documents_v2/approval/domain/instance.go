package domain

import (
	"errors"
	"time"
)

var (
	ErrNoActiveStage       = errors.New("no active stage in instance")
	ErrCannotSkipLastStage = errors.New("cannot skip last stage: no successor exists")
	ErrRevisionRegression  = errors.New("revision_version cannot decrease")
	ErrInstanceTerminal    = errors.New("instance is already in a terminal state")
)

// InstanceStatus represents the top-level lifecycle of an approval instance.
type InstanceStatus string

const (
	InstanceInProgress InstanceStatus = "in_progress"
	InstanceApproved   InstanceStatus = "approved"
	InstanceRejected   InstanceStatus = "rejected"
	InstanceCancelled  InstanceStatus = "cancelled"
)

// StageStatus represents per-stage lifecycle.
type StageStatus string

const (
	StagePending      StageStatus = "pending"
	StageActive       StageStatus = "active"
	StageCompleted    StageStatus = "completed"
	StageSkipped      StageStatus = "skipped"
	StageRejectedHere StageStatus = "rejected_here"
)

// StageInstance holds runtime state for one approval stage.
type StageInstance struct {
	ID                          string
	ApprovalInstanceID          string
	StageOrder                  int
	NameSnapshot                string
	RequiredRoleSnapshot        string
	RequiredCapabilitySnapshot  string
	AreaCodeSnapshot            string
	QuorumSnapshot              QuorumPolicy
	QuorumMSnapshot             *int
	OnEligibilityDriftSnapshot  DriftPolicy
	EligibleActorIDs            []string
	EffectiveDenominator        *int
	Status                      StageStatus
	OpenedAt                    *time.Time
	CompletedAt                 *time.Time
	SkipReason                  string
}

// Instance is the approval instance aggregate.
type Instance struct {
	ID                   string
	TenantID             string
	DocumentID           string
	RouteID              string
	RouteVersionSnapshot int
	Status               InstanceStatus
	SubmittedBy          string
	SubmittedAt          time.Time
	CompletedAt          *time.Time
	ContentHashAtSubmit  string
	IdempotencyKey       string
	RevisionVersion      int
	Stages               []StageInstance
}

// Active returns the current active StageInstance or nil.
func (inst *Instance) Active() *StageInstance {
	for i := range inst.Stages {
		if inst.Stages[i].Status == StageActive {
			return &inst.Stages[i]
		}
	}
	return nil
}

// AdvanceStage moves the active stage to completed and activates the next pending stage.
// When the last stage completes, Status=InstanceApproved.
func (inst *Instance) AdvanceStage() error {
	activeIdx := -1
	for i, s := range inst.Stages {
		if s.Status == StageActive {
			activeIdx = i
			break
		}
	}
	if activeIdx == -1 {
		return ErrNoActiveStage
	}

	now := time.Now().UTC()
	inst.Stages[activeIdx].Status = StageCompleted
	inst.Stages[activeIdx].CompletedAt = &now

	// Activate next pending stage.
	for i := activeIdx + 1; i < len(inst.Stages); i++ {
		if inst.Stages[i].Status == StagePending {
			inst.Stages[i].Status = StageActive
			inst.Stages[i].OpenedAt = &now
			return nil
		}
	}

	// No more pending — instance approved.
	inst.Status = InstanceApproved
	inst.CompletedAt = &now
	return nil
}

// RejectHere marks the active stage as rejected_here and sets instance Status=InstanceRejected.
func (inst *Instance) RejectHere(reason string) error {
	activeIdx := -1
	for i, s := range inst.Stages {
		if s.Status == StageActive {
			activeIdx = i
			break
		}
	}
	if activeIdx == -1 {
		return ErrNoActiveStage
	}

	now := time.Now().UTC()
	inst.Stages[activeIdx].Status = StageRejectedHere
	inst.Stages[activeIdx].SkipReason = reason
	inst.Stages[activeIdx].CompletedAt = &now
	inst.Status = InstanceRejected
	inst.CompletedAt = &now
	return nil
}

// SkipStage marks the active stage as skipped and activates the next pending stage.
// Returns ErrCannotSkipLastStage if no successor exists.
func (inst *Instance) SkipStage(reason string) error {
	activeIdx := -1
	for i, s := range inst.Stages {
		if s.Status == StageActive {
			activeIdx = i
			break
		}
	}
	if activeIdx == -1 {
		return ErrNoActiveStage
	}

	// Check successor exists.
	hasSuccessor := false
	for i := activeIdx + 1; i < len(inst.Stages); i++ {
		if inst.Stages[i].Status == StagePending {
			hasSuccessor = true
			break
		}
	}
	if !hasSuccessor {
		return ErrCannotSkipLastStage
	}

	now := time.Now().UTC()
	inst.Stages[activeIdx].Status = StageSkipped
	inst.Stages[activeIdx].SkipReason = reason
	inst.Stages[activeIdx].CompletedAt = &now

	// Activate next pending.
	for i := activeIdx + 1; i < len(inst.Stages); i++ {
		if inst.Stages[i].Status == StagePending {
			inst.Stages[i].Status = StageActive
			inst.Stages[i].OpenedAt = &now
			return nil
		}
	}
	return nil
}

// BumpRevisionVersion enforces monotonic revision_version — mirrors DB trigger.
func (inst *Instance) BumpRevisionVersion(next int) error {
	if next < inst.RevisionVersion {
		return ErrRevisionRegression
	}
	inst.RevisionVersion = next
	return nil
}

// Cancel sets Status=InstanceCancelled. Errors if already terminal.
func (inst *Instance) Cancel(reason string) error {
	switch inst.Status {
	case InstanceApproved, InstanceRejected, InstanceCancelled:
		return ErrInstanceTerminal
	}
	now := time.Now().UTC()
	inst.Status = InstanceCancelled
	inst.CompletedAt = &now
	_ = reason // persisted by caller in governance_events
	return nil
}

func (inst *Instance) isTerminal() bool {
	return inst.Status == InstanceApproved ||
		inst.Status == InstanceRejected ||
		inst.Status == InstanceCancelled
}
