package application

import (
	"context"
	"fmt"

	"metaldocs/internal/modules/templates/domain"
)

type PublishCmd struct {
	VersionID   string
	ActorUserID string
	DocxKey     string
	SchemaKey   string
}

type PublishResult struct {
	NewDraftID      string
	NewDraftVersion int
}

type ValidationError struct {
	Raw []byte
}

func (v ValidationError) Error() string { return fmt.Sprintf("template invalid: %s", string(v.Raw)) }

func (s *Service) PublishVersion(ctx context.Context, cmd PublishCmd) (PublishResult, error) {
	valid, errs, err := s.docgen.ValidateTemplate(ctx, cmd.DocxKey, cmd.SchemaKey)
	if err != nil {
		return PublishResult{}, fmt.Errorf("docgen-v2 validate: %w", err)
	}
	if !valid {
		return PublishResult{}, ValidationError{Raw: errs}
	}

	newDraftID, newNum, err := s.repo.PublishVersion(ctx, cmd.VersionID, cmd.ActorUserID)
	if err != nil {
		return PublishResult{}, err
	}
	_ = domain.StatusPublished
	return PublishResult{NewDraftID: newDraftID, NewDraftVersion: newNum}, nil
}
