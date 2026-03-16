package memory

import (
	"context"
	"sync"

	"metaldocs/internal/modules/audit/domain"
)

type Writer struct {
	mu     sync.Mutex
	events []domain.Event
}

func NewWriter() *Writer {
	return &Writer{events: make([]domain.Event, 0, 16)}
}

func (w *Writer) Record(_ context.Context, event domain.Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.events = append(w.events, event)
	return nil
}
