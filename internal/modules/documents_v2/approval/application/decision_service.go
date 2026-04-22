package application

import "metaldocs/internal/modules/documents_v2/approval/repository"

// DecisionService handles approver approve/reject decisions.
// Implementation is added in task 5.3.
type DecisionService struct {
	repo    repository.ApprovalRepository
	emitter EventEmitter
	clock   Clock
}
