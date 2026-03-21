package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/messaging"
	"metaldocs/internal/platform/render/carbone"
)

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now().UTC()
}

type Service struct {
	repo             domain.Repository
	attachmentStore  domain.AttachmentStore
	publisher        messaging.Publisher
	clock            Clock
	carboneClient    *carbone.Client
	carboneTemplates *carbone.TemplateRegistry
}

func NewService(repo domain.Repository, publisher messaging.Publisher, clock Clock) *Service {
	if clock == nil {
		clock = realClock{}
	}
	return &Service{repo: repo, publisher: publisher, clock: clock}
}

func (s *Service) WithAttachmentStore(store domain.AttachmentStore) *Service {
	s.attachmentStore = store
	return s
}

func (s *Service) WithCarbone(client *carbone.Client, registry *carbone.TemplateRegistry) *Service {
	s.carboneClient = client
	s.carboneTemplates = registry
	return s
}

func (s *Service) CreateDocument(ctx context.Context, cmd domain.CreateDocumentCommand) (domain.Document, error) {
	if strings.TrimSpace(cmd.DocumentID) == "" ||
		strings.TrimSpace(cmd.Title) == "" ||
		strings.TrimSpace(cmd.OwnerID) == "" ||
		strings.TrimSpace(cmd.BusinessUnit) == "" ||
		strings.TrimSpace(cmd.Department) == "" {
		return domain.Document{}, domain.ErrInvalidCommand
	}

	classification := strings.TrimSpace(cmd.Classification)
	if classification == "" {
		classification = domain.ClassificationInternal
	}

	profile, err := s.resolveDocumentProfile(ctx, cmd.DocumentProfile, cmd.DocumentType)
	if err != nil {
		return domain.Document{}, domain.ErrInvalidDocumentType
	}
	activeSchema, err := s.resolveActiveProfileSchema(ctx, profile.Code)
	if err != nil {
		return domain.Document{}, domain.ErrInvalidCommand
	}
	processArea, subject, err := s.resolveTaxonomy(ctx, cmd.ProcessArea, cmd.Subject)
	if err != nil {
		return domain.Document{}, domain.ErrInvalidCommand
	}

	metadata := normalizeMetadata(cmd.MetadataJSON)
	if _, err := json.Marshal(metadata); err != nil {
		return domain.Document{}, domain.ErrInvalidCommand
	}
	if err := validateMetadata(activeSchema.MetadataRules, metadata); err != nil {
		return domain.Document{}, err
	}

	policies, err := buildAudiencePolicies(cmd, classification)
	if err != nil {
		return domain.Document{}, err
	}

	now := s.clock.Now()
	doc := domain.Document{
		ID:                   strings.TrimSpace(cmd.DocumentID),
		Title:                strings.TrimSpace(cmd.Title),
		DocumentType:         profile.Code,
		DocumentProfile:      profile.Code,
		DocumentFamily:       profile.FamilyCode,
		ProfileSchemaVersion: activeSchema.Version,
		ProcessArea:          processArea,
		Subject:              subject,
		OwnerID:              strings.TrimSpace(cmd.OwnerID),
		BusinessUnit:         strings.TrimSpace(cmd.BusinessUnit),
		Department:           strings.TrimSpace(cmd.Department),
		Classification:       classification,
		Status:               domain.StatusDraft,
		Tags:                 normalizeTags(cmd.Tags),
		EffectiveAt:          cloneTimePtr(cmd.EffectiveAt),
		ExpiryAt:             cloneTimePtr(cmd.ExpiryAt),
		MetadataJSON:         metadata,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	v1 := domain.Version{
		DocumentID:    doc.ID,
		Number:        1,
		Content:       cmd.InitialContent,
		ContentHash:   contentHash(cmd.InitialContent),
		ChangeSummary: "Initial version",
		ContentSource: domain.ContentSourceNative,
		CreatedAt:     now,
	}

	if atomicRepo, ok := s.repo.(domain.AtomicCreateRepository); ok {
		if len(policies) > 0 {
			if atomicWithPolicies, ok := s.repo.(domain.AtomicCreateRepositoryWithPolicies); ok {
				if err := atomicWithPolicies.CreateDocumentWithInitialVersionAndPolicies(ctx, doc, v1, policies); err != nil {
					return domain.Document{}, err
				}
			} else {
				if err := atomicRepo.CreateDocumentWithInitialVersion(ctx, doc, v1); err != nil {
					return domain.Document{}, err
				}
				if err := s.repo.ReplaceAccessPolicies(ctx, domain.ResourceScopeDocument, doc.ID, policies); err != nil {
					return domain.Document{}, err
				}
			}
		} else if err := atomicRepo.CreateDocumentWithInitialVersion(ctx, doc, v1); err != nil {
			return domain.Document{}, err
		}
	} else {
		if err := s.repo.CreateDocument(ctx, doc); err != nil {
			return domain.Document{}, err
		}
		if err := s.repo.SaveVersion(ctx, v1); err != nil {
			return domain.Document{}, err
		}
		if len(policies) > 0 {
			if err := s.repo.ReplaceAccessPolicies(ctx, domain.ResourceScopeDocument, doc.ID, policies); err != nil {
				return domain.Document{}, err
			}
		}
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-create-%s", doc.ID),
			EventType:         "document.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: now.Format(time.RFC3339),
			Version:           1,
			IdempotencyKey:    fmt.Sprintf("document.created:%s", doc.ID),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id":      doc.ID,
				"title":            doc.Title,
				"document_type":    doc.DocumentType,
				"document_profile": doc.DocumentProfile,
				"document_family":  doc.DocumentFamily,
				"process_area":     doc.ProcessArea,
				"subject":          doc.Subject,
				"business_unit":    doc.BusinessUnit,
				"department":       doc.Department,
			},
		})

		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-version-create-%s-1", doc.ID),
			EventType:         "document.version.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: now.Format(time.RFC3339),
			Version:           1,
			IdempotencyKey:    fmt.Sprintf("document.version.created:%s:%d", doc.ID, 1),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id": doc.ID,
				"version":     1,
			},
		})
	}

	return doc, nil
}

func (s *Service) CreateDocumentAuthorized(ctx context.Context, cmd domain.CreateDocumentCommand) (domain.Document, error) {
	if !s.isAllowedForCreate(ctx, cmd) {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	return s.CreateDocument(ctx, cmd)
}

func (s *Service) AddVersion(ctx context.Context, cmd domain.AddVersionCommand) (domain.Version, error) {
	if strings.TrimSpace(cmd.DocumentID) == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}

	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
	if err != nil {
		return domain.Version{}, err
	}
	if doc.Status != domain.StatusDraft && doc.Status != domain.StatusInReview {
		return domain.Version{}, domain.ErrVersioningNotAllowed
	}

	next, err := s.repo.NextVersionNumber(ctx, doc.ID)
	if err != nil {
		return domain.Version{}, err
	}

	version := domain.Version{
		DocumentID:    doc.ID,
		Number:        next,
		Content:       cmd.Content,
		ContentHash:   contentHash(cmd.Content),
		ChangeSummary: strings.TrimSpace(cmd.ChangeSummary),
		ContentSource: domain.ContentSourceNative,
		CreatedAt:     s.clock.Now(),
	}
	if version.ChangeSummary == "" {
		version.ChangeSummary = fmt.Sprintf("Version %d update", next)
	}

	if err := s.repo.SaveVersion(ctx, version); err != nil {
		return domain.Version{}, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-version-create-%s-%d", doc.ID, next),
			EventType:         "document.version.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: version.CreatedAt.Format(time.RFC3339),
			Version:           next,
			IdempotencyKey:    fmt.Sprintf("document.version.created:%s:%d", doc.ID, next),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id": doc.ID,
				"version":     next,
			},
		})
	}

	return version, nil
}

func (s *Service) AddVersionAuthorized(ctx context.Context, cmd domain.AddVersionCommand) (domain.Version, error) {
	if strings.TrimSpace(cmd.DocumentID) == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
	if err != nil {
		return domain.Version{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return domain.Version{}, err
	}
	if !allowed {
		return domain.Version{}, domain.ErrDocumentNotFound
	}
	return s.AddVersion(ctx, cmd)
}

func (s *Service) DiffVersions(ctx context.Context, documentID string, fromVersion, toVersion int) (domain.VersionDiff, error) {
	if strings.TrimSpace(documentID) == "" || fromVersion < 1 || toVersion < 1 || fromVersion == toVersion {
		return domain.VersionDiff{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return domain.VersionDiff{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return domain.VersionDiff{}, err
	}
	if !allowed {
		return domain.VersionDiff{}, domain.ErrDocumentNotFound
	}

	fromItem, err := s.repo.GetVersion(ctx, documentID, fromVersion)
	if err != nil {
		return domain.VersionDiff{}, err
	}
	toItem, err := s.repo.GetVersion(ctx, documentID, toVersion)
	if err != nil {
		return domain.VersionDiff{}, err
	}

	return domain.VersionDiff{
		DocumentID:            documentID,
		FromVersion:           fromVersion,
		ToVersion:             toVersion,
		ContentChanged:        fromItem.ContentHash != toItem.ContentHash,
		MetadataChanged:       []string{},
		ClassificationChanged: false,
		EffectiveAtChanged:    false,
		ExpiryAtChanged:       false,
	}, nil
}

func (s *Service) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	return s.repo.ListDocuments(ctx)
}

func (s *Service) ListDocumentsAuthorized(ctx context.Context) ([]domain.Document, error) {
	docs, err := s.repo.ListDocuments(ctx)
	if err != nil {
		return nil, err
	}
	if shouldBypassPolicy(ctx) {
		return docs, nil
	}

	filtered := make([]domain.Document, 0, len(docs))
	for _, doc := range docs {
		allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
		if err != nil {
			return nil, err
		}
		if allowed {
			filtered = append(filtered, doc)
		}
	}
	return filtered, nil
}
