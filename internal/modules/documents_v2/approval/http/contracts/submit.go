package contracts

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	uuidPattern   = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	sha256Pattern = regexp.MustCompile(`(?i)^[0-9a-f]{64}$`)
)

type SubmitRequest struct {
	RouteID        string `json:"route_id"`
	IdempotencyKey string
	ContentHash    string `json:"content_hash"`
}

func (r SubmitRequest) Validate() error {
	if err := validateUUID("route_id", r.RouteID); err != nil {
		return err
	}
	if err := validateSHA256Hex("content_hash", r.ContentHash); err != nil {
		return err
	}
	return nil
}

type SubmitResponse struct {
	InstanceID string `json:"instance_id"`
	WasReplay  bool   `json:"was_replay"`
	ETag       string `json:"etag"`
}

func validateUUID(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	if !uuidPattern.MatchString(value) {
		return fmt.Errorf("%s must be a valid UUID", field)
	}
	return nil
}

func validateSHA256Hex(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	if !sha256Pattern.MatchString(value) {
		return fmt.Errorf("%s must be 64 hex characters", field)
	}
	return nil
}

func validateRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}
