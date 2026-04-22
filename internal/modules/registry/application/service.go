package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	registrydomain "metaldocs/internal/modules/registry/domain"
	taxonomydomain "metaldocs/internal/modules/taxonomy/domain"
)

type TemplateVersionChecker interface {
	GetTemplateVersionState(ctx context.Context, templateVersionID string) (*string, string, error)
}

type ProfileReader interface {
	GetByCode(ctx context.Context, tenantID, code string) (*taxonomydomain.DocumentProfile, error)
}

type AreaReader interface {
	GetByCode(ctx context.Context, tenantID, code string) (*taxonomydomain.ProcessArea, error)
}

type ControlledDocument = registrydomain.ControlledDocument
type CDFilter = registrydomain.CDFilter

type RegistryService struct {
	db        *sql.DB
	docs      registrydomain.ControlledDocumentRepository
	seq       registrydomain.SequenceAllocator
	tplCheck  TemplateVersionChecker
	profiles  ProfileReader
	areas     AreaReader
	govLogger taxonomydomain.GovernanceLogger
	now       func() time.Time
}

type CreateControlledDocumentCmd struct {
	TenantID                  string
	ProfileCode               string
	ProcessAreaCode           string
	DepartmentCode            *string
	Title                     string
	OwnerUserID               string
	ActorUserID               string
	ManualCode                *string
	ManualCodeReason          *string
	OverrideTemplateVersionID *string
	OverrideTemplateReason    *string
}

func NewRegistryService(
	db *sql.DB,
	docs registrydomain.ControlledDocumentRepository,
	seq registrydomain.SequenceAllocator,
	tplCheck TemplateVersionChecker,
	profiles ProfileReader,
	areas AreaReader,
	govLogger taxonomydomain.GovernanceLogger,
) *RegistryService {
	if govLogger == nil {
		panic("registry: governance logger must not be nil")
	}
	return &RegistryService{
		db:        db,
		docs:      docs,
		seq:       seq,
		tplCheck:  tplCheck,
		profiles:  profiles,
		areas:     areas,
		govLogger: govLogger,
		now:       time.Now,
	}
}

func (s *RegistryService) Create(ctx context.Context, cmd CreateControlledDocumentCmd) (*registrydomain.ControlledDocument, error) {
	profile, err := s.profiles.GetByCode(ctx, cmd.TenantID, cmd.ProfileCode)
	if err != nil {
		return nil, err
	}
	if !profile.IsActive() {
		return nil, taxonomydomain.ErrProfileArchived
	}

	area, err := s.areas.GetByCode(ctx, cmd.TenantID, cmd.ProcessAreaCode)
	if err != nil {
		return nil, err
	}
	if !area.IsActive() {
		return nil, taxonomydomain.ErrAreaArchived
	}

	var (
		code       string
		sequence   *int
		events     []taxonomydomain.GovernanceEvent
		overrideID *string
		createTx   *sql.Tx
	)

	if cmd.ManualCode != nil {
		if !isReasonValid(cmd.ManualCodeReason) {
			return nil, registrydomain.ErrManualCodeReasonRequired
		}
		code = strings.TrimSpace(*cmd.ManualCode)
		taken, err := s.docs.CodeExists(ctx, cmd.TenantID, cmd.ProfileCode, code)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, registrydomain.ErrCDCodeTaken
		}
		payload, _ := json.Marshal(map[string]string{"code": code})
		events = append(events, taxonomydomain.GovernanceEvent{
			TenantID:     cmd.TenantID,
			EventType:    "numbering.override",
			ActorUserID:  cmd.ActorUserID,
			ResourceType: "controlled_document",
			ResourceID:   code,
			Reason:       strings.TrimSpace(*cmd.ManualCodeReason),
			PayloadJSON:  payload,
		})
	} else {
		if s.db != nil {
			tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
			if err != nil {
				return nil, err
			}
			defer func() {
				if createTx != nil {
					_ = tx.Rollback()
				}
			}()
			createTx = tx

			next, err := s.seq.NextAndIncrement(ctx, tx, cmd.TenantID, cmd.ProfileCode)
			if err != nil {
				return nil, err
			}
			code = registrydomain.AutoCode(cmd.ProfileCode, next)
			sequence = &next
			taken, err := s.docs.CodeExists(ctx, cmd.TenantID, cmd.ProfileCode, code)
			if err != nil {
				return nil, err
			}
			if taken {
				return nil, registrydomain.ErrCDCodeTaken
			}
		} else {
			next, err := s.seq.NextAndIncrement(ctx, nil, cmd.TenantID, cmd.ProfileCode)
			if err != nil {
				return nil, err
			}
			code = registrydomain.AutoCode(cmd.ProfileCode, next)
			sequence = &next
			taken, err := s.docs.CodeExists(ctx, cmd.TenantID, cmd.ProfileCode, code)
			if err != nil {
				return nil, err
			}
			if taken {
				return nil, registrydomain.ErrCDCodeTaken
			}
		}
	}

	if cmd.OverrideTemplateVersionID != nil {
		if !isReasonValid(cmd.OverrideTemplateReason) {
			return nil, registrydomain.ErrOverrideReasonRequired
		}
		status, profileCode, err := s.tplCheck.GetTemplateVersionState(ctx, *cmd.OverrideTemplateVersionID)
		if err != nil {
			return nil, err
		}
		_, err = registrydomain.Resolve(registrydomain.TemplateResolutionInput{
			ProfileCode: cmd.ProfileCode,
			OverrideTemplate: &registrydomain.TemplateVersionCandidate{
				ID:          *cmd.OverrideTemplateVersionID,
				ProfileCode: profileCode,
				Status:      status,
			},
		})
		if err != nil {
			return nil, err
		}
		overrideID = cmd.OverrideTemplateVersionID
		payload, _ := json.Marshal(map[string]string{"override_template_version_id": *cmd.OverrideTemplateVersionID})
		events = append(events, taxonomydomain.GovernanceEvent{
			TenantID:     cmd.TenantID,
			EventType:    "template.override",
			ActorUserID:  cmd.ActorUserID,
			ResourceType: "controlled_document",
			ResourceID:   code,
			Reason:       strings.TrimSpace(*cmd.OverrideTemplateReason),
			PayloadJSON:  payload,
		})
	}

	now := s.now().UTC()
	doc := &registrydomain.ControlledDocument{
		TenantID:                  cmd.TenantID,
		ProfileCode:               cmd.ProfileCode,
		ProcessAreaCode:           cmd.ProcessAreaCode,
		DepartmentCode:            cmd.DepartmentCode,
		Code:                      code,
		SequenceNum:               sequence,
		Title:                     cmd.Title,
		OwnerUserID:               cmd.OwnerUserID,
		OverrideTemplateVersionID: overrideID,
		Status:                    registrydomain.CDStatusActive,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
	if createTx != nil {
		if err := s.docs.CreateTx(ctx, createTx, doc); err != nil {
			return nil, err
		}
		if err := createTx.Commit(); err != nil {
			return nil, err
		}
		createTx = nil
	} else {
		if err := s.docs.Create(ctx, doc); err != nil {
			return nil, err
		}
	}

	// Governance events are best-effort; document creation is already committed.
	for _, event := range events {
		if err := s.govLogger.Log(ctx, event); err != nil {
			slog.Warn("registry governance event logging failed", "event_type", event.EventType, "resource_id", event.ResourceID, "error", err)
		}
	}

	return doc, nil
}

func (s *RegistryService) Obsolete(ctx context.Context, tenantID, controlledDocumentID string) error {
	return s.changeStatus(ctx, tenantID, controlledDocumentID, registrydomain.CDStatusObsolete)
}

func (s *RegistryService) Supersede(ctx context.Context, tenantID, controlledDocumentID string) error {
	return s.changeStatus(ctx, tenantID, controlledDocumentID, registrydomain.CDStatusSuperseded)
}

func (s *RegistryService) List(ctx context.Context, tenantID string, filter CDFilter) ([]ControlledDocument, error) {
	return s.docs.List(ctx, tenantID, filter)
}

func (s *RegistryService) Get(ctx context.Context, tenantID, id string) (*ControlledDocument, error) {
	return s.docs.GetByID(ctx, tenantID, id)
}

func (s *RegistryService) changeStatus(ctx context.Context, tenantID, controlledDocumentID string, next registrydomain.CDStatus) error {
	doc, err := s.docs.GetByID(ctx, tenantID, controlledDocumentID)
	if err != nil {
		return err
	}
	if !doc.IsActive() {
		return registrydomain.ErrCDNotActive
	}
	return s.docs.UpdateStatus(ctx, tenantID, controlledDocumentID, next, s.now().UTC())
}

func isReasonValid(reason *string) bool {
	if reason == nil {
		return false
	}
	return len(strings.TrimSpace(*reason)) >= 10
}
