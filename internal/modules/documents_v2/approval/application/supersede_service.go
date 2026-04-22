package application

import "metaldocs/internal/modules/documents_v2/approval/repository"

// SupersedeService marks a published document as superseded by a newer revision.
// Implementation is added in task 5.6.
type SupersedeService struct {
	repo    repository.ApprovalRepository
	emitter EventEmitter
	clock   Clock
}
