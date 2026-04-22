package application

import "metaldocs/internal/modules/documents_v2/approval/repository"

// ObsoleteService marks a document as obsolete (end-of-life).
// Implementation is added in task 5.7.
type ObsoleteService struct {
	repo    repository.ApprovalRepository
	emitter EventEmitter
	clock   Clock
}
