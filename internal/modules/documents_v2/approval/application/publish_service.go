package application

import "metaldocs/internal/modules/documents_v2/approval/repository"

// PublishService handles transitioning an approved document to published state.
// Implementation is added in task 5.4.
type PublishService struct {
	repo    repository.ApprovalRepository
	emitter EventEmitter
	clock   Clock
}
