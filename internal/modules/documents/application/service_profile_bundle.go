package application

import (
	"context"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

type DocumentProfileBundle struct {
	Profile             domain.DocumentProfile
	Schema              domain.DocumentProfileSchemaVersion
	Governance          domain.DocumentProfileGovernance
	ProcessAreas        []domain.ProcessArea
	DocumentDepartments []domain.DocumentDepartment
	Subjects            []domain.Subject
}

func (s *Service) GetDocumentProfileBundle(ctx context.Context, profileCode string) (DocumentProfileBundle, error) {
	normalizedCode := strings.ToLower(strings.TrimSpace(profileCode))
	if normalizedCode == "" {
		return DocumentProfileBundle{}, domain.ErrInvalidCommand
	}

	profiles, err := s.ListDocumentProfiles(ctx)
	if err != nil {
		return DocumentProfileBundle{}, err
	}
	var profile domain.DocumentProfile
	found := false
	for _, item := range profiles {
		if strings.EqualFold(strings.TrimSpace(item.Code), normalizedCode) {
			profile = item
			found = true
			break
		}
	}
	if !found {
		return DocumentProfileBundle{}, domain.ErrInvalidDocumentType
	}

	schema, err := s.resolveActiveProfileSchema(ctx, normalizedCode)
	if err != nil {
		return DocumentProfileBundle{}, err
	}

	governance, err := s.GetDocumentProfileGovernance(ctx, normalizedCode)
	if err != nil {
		return DocumentProfileBundle{}, err
	}

	processAreas, err := s.ListProcessAreas(ctx)
	if err != nil {
		return DocumentProfileBundle{}, err
	}

	departments, err := s.ListDocumentDepartments(ctx)
	if err != nil {
		return DocumentProfileBundle{}, err
	}

	subjects, err := s.ListSubjects(ctx)
	if err != nil {
		return DocumentProfileBundle{}, err
	}

	return DocumentProfileBundle{
		Profile:             profile,
		Schema:              schema,
		Governance:          governance,
		ProcessAreas:        processAreas,
		DocumentDepartments: departments,
		Subjects:            subjects,
	}, nil
}
