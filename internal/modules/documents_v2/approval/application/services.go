package application

import (
	"errors"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/repository"
)

// Clock abstracts time so services can be tested deterministically.
type Clock interface {
	Now() time.Time
}

// RealClock is the production Clock implementation.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }

// ErrFloatInPayload is returned by ValidateEventPayload when a float64 value
// is found in the payload map. JSON-unmarshalled numbers default to float64
// and break canonical hashing.
var ErrFloatInPayload = errors.New("event payload must not contain float64 values; use string or int")

// Services is the top-level application service container for the approval
// subsystem. Each field is a focused service; all share the same repo,
// emitter, and clock.
type Services struct {
	Submit    *SubmitService
	Decision  *DecisionService
	Publish   *PublishService
	Scheduler *SchedulerService
	Supersede *SupersedeService
	Obsolete  *ObsoleteService
	clock     Clock
}

// NewServices constructs a fully wired Services value.
func NewServices(repo repository.ApprovalRepository, emitter EventEmitter, clock Clock) *Services {
	return &Services{
		Submit:    &SubmitService{repo: repo, emitter: emitter, clock: clock},
		Decision:  &DecisionService{repo: repo, emitter: emitter, clock: clock},
		Publish:   &PublishService{repo: repo, emitter: emitter, clock: clock},
		Scheduler: &SchedulerService{repo: repo, emitter: emitter, clock: clock},
		Supersede: &SupersedeService{repo: repo, emitter: emitter, clock: clock},
		Obsolete:  &ObsoleteService{repo: repo, emitter: emitter, clock: clock},
		clock:     clock,
	}
}

// ValidateEventPayload returns ErrFloatInPayload if any value in payload is a
// float64. JSON unmarshal defaults numeric values to float64, which breaks
// canonical hashing; callers must use strings or ints instead.
func ValidateEventPayload(payload map[string]any) error {
	for _, v := range payload {
		if _, ok := v.(float64); ok {
			return ErrFloatInPayload
		}
	}
	return nil
}
