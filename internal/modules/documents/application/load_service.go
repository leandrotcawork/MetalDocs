package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

// LoadRepo is the minimal read model required by the load endpoint.
// It prefers a user-specific active draft and falls back to the current released version.
type LoadRepo interface {
	GetActiveDraft(ctx context.Context, documentID, userID string) (*LoadVersion, error)
	GetCurrentReleased(ctx context.Context, documentID string) (*LoadVersion, error)
}

type LoadVersion struct {
	DocumentID      string
	Version         int
	Status          string
	Content         json.RawMessage
	TemplateKey     string
	TemplateVersion int
	ContentHash     string
}

type LoadOutput struct {
	DocumentID      string
	Version         int
	Status          string
	Content         json.RawMessage
	TemplateKey     string
	TemplateVersion int
	ContentHash     string
}

type LoadService struct {
	repo LoadRepo
}

var errLoadServiceNotConfigured = errors.New("load service not configured")

func NewLoadService(repo LoadRepo) *LoadService {
	return &LoadService{repo: repo}
}

// LoadForEdit prefers the user's active draft if one exists, otherwise returns
// the current released version.
func (s *LoadService) LoadForEdit(ctx context.Context, documentID, userID string) (LoadOutput, error) {
	normalizedDocumentID := strings.TrimSpace(documentID)
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedDocumentID == "" || normalizedUserID == "" {
		return LoadOutput{}, domain.ErrInvalidCommand
	}
	if s == nil || s.repo == nil {
		return LoadOutput{}, errLoadServiceNotConfigured
	}

	draft, err := s.repo.GetActiveDraft(ctx, normalizedDocumentID, normalizedUserID)
	if err != nil {
		return LoadOutput{}, err
	}
	if draft != nil {
		return LoadOutput{
			DocumentID:      draft.DocumentID,
			Version:         draft.Version,
			Status:          draft.Status,
			Content:         draft.Content,
			TemplateKey:     draft.TemplateKey,
			TemplateVersion: draft.TemplateVersion,
			ContentHash:     draft.ContentHash,
		}, nil
	}

	released, err := s.repo.GetCurrentReleased(ctx, normalizedDocumentID)
	if err != nil {
		return LoadOutput{}, err
	}
	if released == nil {
		return LoadOutput{}, domain.ErrDocumentNotFound
	}
	return LoadOutput{
		DocumentID:      released.DocumentID,
		Version:         released.Version,
		Status:          released.Status,
		Content:         released.Content,
		TemplateKey:     released.TemplateKey,
		TemplateVersion: released.TemplateVersion,
		ContentHash:     released.ContentHash,
	}, nil
}
