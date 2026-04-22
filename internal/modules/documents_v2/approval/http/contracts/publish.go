package contracts

import (
	"fmt"
	"time"
)

type PublishRequest struct {
	IdempotencyKey string
	IfMatchVersion int
}

type SchedulePublishRequest struct {
	EffectiveFrom  string `json:"effective_from"`
	IdempotencyKey string
	IfMatchVersion int
}

func (r SchedulePublishRequest) Validate() error {
	if err := validateRequired("effective_from", r.EffectiveFrom); err != nil {
		return err
	}
	t, err := time.Parse(time.RFC3339, r.EffectiveFrom)
	if err != nil {
		return fmt.Errorf("effective_from must be parseable RFC3339: %w", err)
	}
	_, offset := t.Zone()
	if offset != 0 {
		return fmt.Errorf("effective_from must be UTC")
	}
	return nil
}

type PublishResponse struct {
	DocumentID    string `json:"document_id"`
	NewStatus     string `json:"new_status"`
	EffectiveFrom string `json:"effective_from,omitempty"`
}
