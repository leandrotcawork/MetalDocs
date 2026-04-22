package application

import "metaldocs/internal/modules/documents_v2/approval/repository"

// SubmitService handles document submission for approval.
// Implementation is added in task 5.2.
type SubmitService struct {
	repo    repository.ApprovalRepository
	emitter EventEmitter
	clock   Clock
}
