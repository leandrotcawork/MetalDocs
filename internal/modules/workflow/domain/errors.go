package domain

import "errors"

var (
	ErrInvalidCommand         = errors.New("invalid workflow command")
	ErrInvalidTransition      = errors.New("invalid workflow transition")
	ErrApprovalNotFound       = errors.New("workflow approval not found")
	ErrApprovalReviewerDenied = errors.New("workflow approval reviewer denied")
)
