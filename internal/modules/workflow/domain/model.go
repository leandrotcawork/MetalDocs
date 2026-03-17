package domain

import "time"

type TransitionCommand struct {
	DocumentID       string
	ToStatus         string
	ActorID          string
	Reason           string
	AssignedReviewer string
	TraceID          string
}

type TransitionResult struct {
	DocumentID       string
	FromStatus       string
	ToStatus         string
	ApprovalID       string
	ApprovalStatus   string
	AssignedReviewer string
}

type Approval struct {
	ID               string
	DocumentID       string
	RequestedBy      string
	AssignedReviewer string
	DecisionBy       string
	Status           string
	RequestReason    string
	DecisionReason   string
	RequestedAt      time.Time
	DecidedAt        *time.Time
}

const (
	ApprovalStatusPending  = "PENDING"
	ApprovalStatusApproved = "APPROVED"
	ApprovalStatusRejected = "REJECTED"
)
