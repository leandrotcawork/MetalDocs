package application

import "metaldocs/internal/modules/documents_v2/approval/repository"

// SchedulerService processes scheduled publish jobs (F6 — ListScheduledDue).
// Implementation is added in task 5.5.
type SchedulerService struct {
	repo    repository.ApprovalRepository
	emitter EventEmitter
	clock   Clock
}
