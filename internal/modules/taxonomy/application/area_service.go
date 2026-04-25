package application

import (
	"context"
	"time"

	"metaldocs/internal/modules/taxonomy/domain"
)

type AreaService struct {
	areas     domain.AreaRepository
	govLogger domain.GovernanceLogger
	now       func() time.Time
}

func NewAreaService(areas domain.AreaRepository, govLogger domain.GovernanceLogger) *AreaService {
	return &AreaService{
		areas:     areas,
		govLogger: govLogger,
		now:       time.Now,
	}
}

func (s *AreaService) List(ctx context.Context, tenantID string, includeArchived bool) ([]domain.ProcessArea, error) {
	return s.areas.List(ctx, tenantID, includeArchived)
}

func (s *AreaService) Get(ctx context.Context, tenantID, code string) (*domain.ProcessArea, error) {
	return s.areas.GetByCode(ctx, tenantID, code)
}

func (s *AreaService) Create(ctx context.Context, a *domain.ProcessArea) error {
	return s.areas.Create(ctx, a)
}

func (s *AreaService) Update(ctx context.Context, a *domain.ProcessArea) error {
	return s.areas.Update(ctx, a)
}

func (s *AreaService) SetParent(ctx context.Context, tenantID, areaCode string, parentCode *string, actorID string) error {
	area, err := s.areas.GetByCode(ctx, tenantID, areaCode)
	if err != nil {
		return err
	}
	if !area.IsActive() {
		return domain.ErrAreaArchived
	}

	if parentCode != nil {
		if *parentCode == areaCode {
			return domain.ErrAreaParentCycle
		}
		if _, err := s.areas.GetByCode(ctx, tenantID, *parentCode); err != nil {
			return err
		}
		ancestors, err := s.areas.ListAncestors(ctx, tenantID, *parentCode)
		if err != nil {
			return err
		}
		for _, ancestorCode := range ancestors {
			if ancestorCode == areaCode {
				return domain.ErrAreaParentCycle
			}
		}
	}

	area.ParentCode = parentCode
	if err := s.areas.Update(ctx, area); err != nil {
		return err
	}
	return s.govLogger.Log(ctx, domain.GovernanceEvent{
		TenantID:     tenantID,
		EventType:    "area.parent_changed",
		ActorUserID:  actorID,
		ResourceType: "process_area",
		ResourceID:   areaCode,
		PayloadJSON:  []byte(`{}`),
	})
}

func (s *AreaService) Archive(ctx context.Context, tenantID, areaCode, actorID string) error {
	area, err := s.areas.GetByCode(ctx, tenantID, areaCode)
	if err != nil {
		return err
	}
	if err := area.Archive(s.now()); err != nil {
		return err
	}
	if err := s.areas.Update(ctx, area); err != nil {
		return err
	}
	return s.govLogger.Log(ctx, domain.GovernanceEvent{
		TenantID:     tenantID,
		EventType:    "area.archived",
		ActorUserID:  actorID,
		ResourceType: "process_area",
		ResourceID:   areaCode,
		PayloadJSON:  []byte(`{}`),
	})
}
