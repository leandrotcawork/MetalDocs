package application

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/authn"
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
			IdempotencyKey:    fmt.Sprintf("doc-create-%s", doc.ID),
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
			IdempotencyKey:    fmt.Sprintf("doc-version-create-%s-1", doc.ID),
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
			IdempotencyKey:    fmt.Sprintf("doc-version-create-%s-%d", doc.ID, next),
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

func (s *Service) GetNativeContentAuthorized(ctx context.Context, documentID string) (domain.Version, error) {
	if strings.TrimSpace(documentID) == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return domain.Version{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return domain.Version{}, err
	}
	if !allowed {
		return domain.Version{}, domain.ErrDocumentNotFound
	}
	return s.latestVersion(ctx, doc.ID)
}

func (s *Service) SaveNativeContentAuthorized(ctx context.Context, cmd domain.SaveNativeContentCommand) (domain.Version, error) {
	if s.attachmentStore == nil {
		return domain.Version{}, domain.ErrAttachmentStoreUnavailable
	}
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
	if !isVersioningAllowed(doc) {
		return domain.Version{}, domain.ErrVersioningNotAllowed
	}

	contentPayload := cmd.Content
	if contentPayload == nil {
		contentPayload = map[string]any{}
	}
	if schema, ok, err := s.resolveDocumentProfileSchema(ctx, doc.DocumentProfile, doc.ProfileSchemaVersion); err != nil {
		return domain.Version{}, err
	} else if ok {
		if err := validateContentSchema(schema.ContentSchema, contentPayload); err != nil {
			return domain.Version{}, err
		}
	}
	rawContent, err := json.Marshal(contentPayload)
	if err != nil {
		return domain.Version{}, domain.ErrInvalidCommand
	}
	contentText := string(rawContent)

	next, err := s.repo.NextVersionNumber(ctx, doc.ID)
	if err != nil {
		return domain.Version{}, err
	}
	now := s.clock.Now()

	version := domain.Version{
		DocumentID:    doc.ID,
		Number:        next,
		Content:       contentText,
		ContentHash:   contentHash(contentText),
		ChangeSummary: fmt.Sprintf("Content version %d", next),
		ContentSource: domain.ContentSourceNative,
		NativeContent: contentPayload,
		TextContent:   contentText,
		CreatedAt:     now,
	}

	pdfBytes, err := s.renderDocumentPDF(ctx, doc, version, cmd.Content, cmd.TraceID)
	if err != nil {
		return domain.Version{}, err
	}
	pdfKey := documentContentStorageKey(doc.ID, next, "pdf")
	if err := s.attachmentStore.Save(ctx, pdfKey, pdfBytes); err != nil {
		return domain.Version{}, err
	}
	version.PdfStorageKey = pdfKey

	if err := s.repo.SaveVersion(ctx, version); err != nil {
		_ = s.attachmentStore.Delete(ctx, pdfKey)
		return domain.Version{}, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-version-create-%s-%d", doc.ID, next),
			EventType:         "document.version.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: now.Format(time.RFC3339),
			Version:           next,
			IdempotencyKey:    fmt.Sprintf("doc-version-create-%s-%d", doc.ID, next),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id": doc.ID,
				"version":     next,
				"source":      version.ContentSource,
			},
		})
	}

	return version, nil
}

func (s *Service) RenderContentPDFAuthorized(ctx context.Context, documentID, traceID string) (domain.Version, error) {
	if s.attachmentStore == nil {
		return domain.Version{}, domain.ErrAttachmentStoreUnavailable
	}
	if strings.TrimSpace(documentID) == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
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

	version, err := s.latestVersion(ctx, doc.ID)
	if err != nil {
		return domain.Version{}, err
	}

	contentSource := strings.TrimSpace(version.ContentSource)
	if contentSource == "" {
		contentSource = domain.ContentSourceNative
	}
	var pdfBytes []byte
	switch contentSource {
	case domain.ContentSourceDocxUpload:
		if strings.TrimSpace(version.DocxStorageKey) == "" {
			return domain.Version{}, domain.ErrVersionNotFound
		}
		docxPayload, err := s.OpenContentStorage(ctx, version.DocxStorageKey)
		if err != nil {
			return domain.Version{}, err
		}
		pdfBytes, err = s.convertDocxToPDF(ctx, docxPayload, traceID)
		if err != nil {
			return domain.Version{}, err
		}
	default:
		content := version.NativeContent
		if len(content) == 0 && strings.TrimSpace(version.Content) != "" {
			var parsed map[string]any
			if err := json.Unmarshal([]byte(version.Content), &parsed); err == nil {
				content = parsed
			}
		}
		pdfBytes, err = s.renderDocumentPDF(ctx, doc, version, content, traceID)
		if err != nil {
			return domain.Version{}, err
		}
	}

	pdfKey := strings.TrimSpace(version.PdfStorageKey)
	if pdfKey == "" {
		pdfKey = documentContentStorageKey(doc.ID, version.Number, "pdf")
	}
	if err := s.attachmentStore.Save(ctx, pdfKey, pdfBytes); err != nil {
		return domain.Version{}, err
	}
	if pdfKey != version.PdfStorageKey || version.PageCount == 0 {
		if err := s.repo.UpdateVersionPDF(ctx, doc.ID, version.Number, pdfKey, version.PageCount); err != nil {
			return domain.Version{}, err
		}
	}
	version.PdfStorageKey = pdfKey
	return version, nil
}

func (s *Service) UploadDocxContentAuthorized(ctx context.Context, cmd domain.UploadDocxContentCommand) (domain.Version, error) {
	if s.attachmentStore == nil {
		return domain.Version{}, domain.ErrAttachmentStoreUnavailable
	}
	if strings.TrimSpace(cmd.DocumentID) == "" || strings.TrimSpace(cmd.FileName) == "" || len(cmd.Content) == 0 {
		return domain.Version{}, domain.ErrInvalidAttachment
	}
	if len(cmd.Content) > 10*1024*1024 {
		return domain.Version{}, domain.ErrInvalidAttachment
	}
	if !isDocxPayload(cmd.Content) {
		return domain.Version{}, domain.ErrInvalidAttachment
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
	if !isVersioningAllowed(doc) {
		return domain.Version{}, domain.ErrVersioningNotAllowed
	}

	next, err := s.repo.NextVersionNumber(ctx, doc.ID)
	if err != nil {
		return domain.Version{}, err
	}
	now := s.clock.Now()

	docxKey := documentContentStorageKey(doc.ID, next, "docx")
	if err := s.attachmentStore.Save(ctx, docxKey, cmd.Content); err != nil {
		return domain.Version{}, err
	}

	pdfBytes, err := s.convertDocxToPDF(ctx, cmd.Content, cmd.TraceID)
	if err != nil {
		_ = s.attachmentStore.Delete(ctx, docxKey)
		return domain.Version{}, err
	}
	pdfKey := documentContentStorageKey(doc.ID, next, "pdf")
	if err := s.attachmentStore.Save(ctx, pdfKey, pdfBytes); err != nil {
		_ = s.attachmentStore.Delete(ctx, docxKey)
		return domain.Version{}, err
	}

	textContent := extractDocxText(cmd.Content)

	version := domain.Version{
		DocumentID:       doc.ID,
		Number:           next,
		Content:          textContent,
		ContentHash:      contentHash(textContent),
		ChangeSummary:    fmt.Sprintf("Content version %d", next),
		ContentSource:    domain.ContentSourceDocxUpload,
		DocxStorageKey:   docxKey,
		PdfStorageKey:    pdfKey,
		TextContent:      textContent,
		FileSizeBytes:    int64(len(cmd.Content)),
		OriginalFilename: strings.TrimSpace(cmd.FileName),
		CreatedAt:        now,
	}

	if err := s.repo.SaveVersion(ctx, version); err != nil {
		_ = s.attachmentStore.Delete(ctx, pdfKey)
		_ = s.attachmentStore.Delete(ctx, docxKey)
		return domain.Version{}, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-version-create-%s-%d", doc.ID, next),
			EventType:         "document.version.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: now.Format(time.RFC3339),
			Version:           next,
			IdempotencyKey:    fmt.Sprintf("doc-version-create-%s-%d", doc.ID, next),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id": doc.ID,
				"version":     next,
				"source":      version.ContentSource,
			},
		})
	}

	return version, nil
}

func (s *Service) RenderProfileTemplateDocx(ctx context.Context, profileCode string) ([]byte, error) {
	return s.renderProfileTemplate(ctx, profileCode, map[string]any{}, "docx", "profile-template")
}

func (s *Service) RenderDocumentTemplateDocxAuthorized(ctx context.Context, documentID string) ([]byte, error) {
	if strings.TrimSpace(documentID) == "" {
		return nil, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return nil, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, domain.ErrDocumentNotFound
	}
	data := buildDocumentTemplateData(doc, nil)
	return s.renderProfileTemplate(ctx, doc.DocumentProfile, data, "docx", "document-template")
}

func (s *Service) OpenContentStorage(ctx context.Context, storageKey string) ([]byte, error) {
	if s.attachmentStore == nil {
		return nil, domain.ErrAttachmentStoreUnavailable
	}
	if strings.TrimSpace(storageKey) == "" {
		return nil, domain.ErrInvalidCommand
	}
	reader, err := s.attachmentStore.Open(ctx, storageKey)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()
	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return payload, nil
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

func (s *Service) ListDocumentTypes(ctx context.Context) ([]domain.DocumentType, error) {
	items, err := s.repo.ListDocumentTypes(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return domain.DefaultDocumentTypes(), nil
	}
	return items, nil
}

func (s *Service) ListDocumentFamilies(ctx context.Context) ([]domain.DocumentFamily, error) {
	items, err := s.repo.ListDocumentFamilies(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return domain.DefaultDocumentFamilies(), nil
	}
	return items, nil
}

func (s *Service) ListDocumentProfiles(ctx context.Context) ([]domain.DocumentProfile, error) {
	items, err := s.repo.ListDocumentProfiles(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		items = domain.DefaultDocumentProfiles()
	}
	out := make([]domain.DocumentProfile, 0, len(items))
	for _, item := range items {
		normalized, err := domain.NormalizeDocumentProfile(item)
		if err != nil {
			return nil, err
		}
		out = append(out, normalized)
	}
	return out, nil
}

func (s *Service) UpsertDocumentProfile(ctx context.Context, item domain.DocumentProfile) error {
	normalized, err := domain.NormalizeDocumentProfile(item)
	if err != nil {
		return err
	}
	families, err := s.ListDocumentFamilies(ctx)
	if err != nil {
		return err
	}
	hasFamily := false
	for _, family := range families {
		if strings.EqualFold(family.Code, normalized.FamilyCode) {
			hasFamily = true
			break
		}
	}
	if !hasFamily {
		return domain.ErrInvalidCommand
	}
	return s.repo.UpsertDocumentProfile(ctx, normalized)
}

func (s *Service) DeactivateDocumentProfile(ctx context.Context, code string) error {
	normalizedCode := strings.ToLower(strings.TrimSpace(code))
	if normalizedCode == "" {
		return domain.ErrInvalidCommand
	}
	return s.repo.DeactivateDocumentProfile(ctx, normalizedCode)
}

func (s *Service) ListDocumentProfileSchemas(ctx context.Context, profileCode string) ([]domain.DocumentProfileSchemaVersion, error) {
	profileCode = strings.ToLower(strings.TrimSpace(profileCode))
	items, err := s.repo.ListDocumentProfileSchemas(ctx, profileCode)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return filterDefaultSchemas(profileCode), nil
	}
	return items, nil
}

func (s *Service) UpsertDocumentProfileSchemaVersion(ctx context.Context, item domain.DocumentProfileSchemaVersion) error {
	normalized, err := domain.NormalizeDocumentProfileSchemaVersion(item)
	if err != nil {
		return err
	}
	profiles, err := s.ListDocumentProfiles(ctx)
	if err != nil {
		return err
	}
	hasProfile := false
	for _, profile := range profiles {
		if strings.EqualFold(profile.Code, normalized.ProfileCode) {
			hasProfile = true
			break
		}
	}
	if !hasProfile {
		return domain.ErrInvalidCommand
	}
	return s.repo.UpsertDocumentProfileSchemaVersion(ctx, normalized)
}

func (s *Service) ActivateDocumentProfileSchemaVersion(ctx context.Context, profileCode string, version int) error {
	normalizedCode := strings.ToLower(strings.TrimSpace(profileCode))
	if normalizedCode == "" || version <= 0 {
		return domain.ErrInvalidCommand
	}
	items, err := s.ListDocumentProfileSchemas(ctx, normalizedCode)
	if err != nil {
		return err
	}
	found := false
	for _, item := range items {
		if item.Version == version {
			found = true
			break
		}
	}
	if !found {
		return domain.ErrInvalidCommand
	}
	return s.repo.ActivateDocumentProfileSchemaVersion(ctx, normalizedCode, version)
}

func (s *Service) GetDocumentProfileGovernance(ctx context.Context, profileCode string) (domain.DocumentProfileGovernance, error) {
	profileCode = strings.ToLower(strings.TrimSpace(profileCode))
	item, err := s.repo.GetDocumentProfileGovernance(ctx, profileCode)
	if err == nil {
		return item, nil
	}
	for _, fallback := range domain.DefaultDocumentProfileGovernance() {
		if strings.EqualFold(fallback.ProfileCode, profileCode) {
			return fallback, nil
		}
	}
	return domain.DocumentProfileGovernance{}, err
}

func (s *Service) UpsertDocumentProfileGovernance(ctx context.Context, item domain.DocumentProfileGovernance) error {
	normalized, err := domain.NormalizeDocumentProfileGovernance(item)
	if err != nil {
		return err
	}
	profiles, err := s.ListDocumentProfiles(ctx)
	if err != nil {
		return err
	}
	hasProfile := false
	for _, profile := range profiles {
		if strings.EqualFold(profile.Code, normalized.ProfileCode) {
			hasProfile = true
			break
		}
	}
	if !hasProfile {
		return domain.ErrInvalidCommand
	}
	return s.repo.UpsertDocumentProfileGovernance(ctx, normalized)
}

func (s *Service) ListProcessAreas(ctx context.Context) ([]domain.ProcessArea, error) {
	items, err := s.repo.ListProcessAreas(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return domain.DefaultProcessAreas(), nil
	}
	return items, nil
}

func (s *Service) ListDocumentDepartments(ctx context.Context) ([]domain.DocumentDepartment, error) {
	items, err := s.repo.ListDocumentDepartments(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return domain.DefaultDocumentDepartments(), nil
	}
	return items, nil
}

func (s *Service) UpsertProcessArea(ctx context.Context, item domain.ProcessArea) error {
	normalized, err := domain.NormalizeProcessArea(item)
	if err != nil {
		return err
	}
	return s.repo.UpsertProcessArea(ctx, normalized)
}

func (s *Service) UpsertDocumentDepartment(ctx context.Context, item domain.DocumentDepartment) error {
	normalized, err := domain.NormalizeDocumentDepartment(item)
	if err != nil {
		return err
	}
	return s.repo.UpsertDocumentDepartment(ctx, normalized)
}

func (s *Service) DeactivateProcessArea(ctx context.Context, code string) error {
	normalizedCode := strings.ToLower(strings.TrimSpace(code))
	if normalizedCode == "" {
		return domain.ErrInvalidCommand
	}
	return s.repo.DeactivateProcessArea(ctx, normalizedCode)
}

func (s *Service) DeactivateDocumentDepartment(ctx context.Context, code string) error {
	normalizedCode := strings.ToLower(strings.TrimSpace(code))
	if normalizedCode == "" {
		return domain.ErrInvalidCommand
	}
	return s.repo.DeactivateDocumentDepartment(ctx, normalizedCode)
}

func (s *Service) ListSubjects(ctx context.Context) ([]domain.Subject, error) {
	items, err := s.repo.ListSubjects(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return domain.DefaultSubjects(), nil
	}
	return items, nil
}

func (s *Service) UpsertSubject(ctx context.Context, item domain.Subject) error {
	normalized, err := domain.NormalizeSubject(item)
	if err != nil {
		return err
	}
	areas, err := s.ListProcessAreas(ctx)
	if err != nil {
		return err
	}
	hasArea := false
	for _, area := range areas {
		if strings.EqualFold(area.Code, normalized.ProcessAreaCode) {
			hasArea = true
			break
		}
	}
	if !hasArea {
		return domain.ErrInvalidCommand
	}
	return s.repo.UpsertSubject(ctx, normalized)
}

func (s *Service) DeactivateSubject(ctx context.Context, code string) error {
	normalizedCode := strings.ToLower(strings.TrimSpace(code))
	if normalizedCode == "" {
		return domain.ErrInvalidCommand
	}
	return s.repo.DeactivateSubject(ctx, normalizedCode)
}

func (s *Service) ListVersions(ctx context.Context, documentID string) ([]domain.Version, error) {
	if strings.TrimSpace(documentID) == "" {
		return nil, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return nil, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, domain.ErrDocumentNotFound
	}
	return s.repo.ListVersions(ctx, strings.TrimSpace(documentID))
}

func (s *Service) GetDocumentAuthorized(ctx context.Context, documentID string) (domain.Document, error) {
	if strings.TrimSpace(documentID) == "" {
		return domain.Document{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return domain.Document{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return domain.Document{}, err
	}
	if !allowed {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	return doc, nil
}

func (s *Service) HeartbeatCollaborationPresenceAuthorized(ctx context.Context, documentID, userID, displayName string) error {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(userID) == "" {
		userID = authn.UserIDFromContext(ctx)
	}
	normalized, err := domain.NormalizeCollaborationPresence(domain.CollaborationPresence{
		DocumentID:  doc.ID,
		UserID:      userID,
		DisplayName: strings.TrimSpace(displayName),
		LastSeenAt:  s.clock.Now(),
	})
	if err != nil {
		return err
	}
	return s.repo.UpsertCollaborationPresence(ctx, normalized)
}

func (s *Service) ListCollaborationPresenceAuthorized(ctx context.Context, documentID string) ([]domain.CollaborationPresence, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return nil, err
	}
	activeSince := s.clock.Now().Add(-time.Duration(domain.DefaultPresenceWindowSeconds) * time.Second)
	return s.repo.ListCollaborationPresence(ctx, doc.ID, activeSince)
}

func (s *Service) AcquireDocumentEditLockAuthorized(ctx context.Context, documentID, userID, displayName, reason string, ttlSeconds int) (domain.DocumentEditLock, error) {
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return domain.DocumentEditLock{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return domain.DocumentEditLock{}, err
	}
	if !allowed {
		return domain.DocumentEditLock{}, domain.ErrDocumentNotFound
	}

	if strings.TrimSpace(userID) == "" {
		userID = authn.UserIDFromContext(ctx)
	}
	if ttlSeconds <= 0 {
		ttlSeconds = domain.DefaultLockTTLSeconds
	}
	now := s.clock.Now()
	normalized, err := domain.NormalizeDocumentEditLock(domain.DocumentEditLock{
		DocumentID:  doc.ID,
		LockedBy:    userID,
		DisplayName: strings.TrimSpace(displayName),
		LockReason:  strings.TrimSpace(reason),
		AcquiredAt:  now,
		ExpiresAt:   now.Add(time.Duration(ttlSeconds) * time.Second),
	})
	if err != nil {
		return domain.DocumentEditLock{}, err
	}
	return s.repo.AcquireDocumentEditLock(ctx, normalized, now)
}

func (s *Service) GetDocumentEditLockAuthorized(ctx context.Context, documentID string) (domain.DocumentEditLock, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return domain.DocumentEditLock{}, err
	}
	return s.repo.GetDocumentEditLock(ctx, doc.ID, s.clock.Now())
}

func (s *Service) ReleaseDocumentEditLockAuthorized(ctx context.Context, documentID, userID string) error {
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return err
	}
	if !allowed {
		return domain.ErrDocumentNotFound
	}
	if strings.TrimSpace(userID) == "" {
		userID = authn.UserIDFromContext(ctx)
	}
	return s.repo.ReleaseDocumentEditLock(ctx, doc.ID, strings.TrimSpace(userID))
}

func (s *Service) UploadAttachmentAuthorized(ctx context.Context, cmd domain.UploadAttachmentCommand) (domain.Attachment, error) {
	if s.attachmentStore == nil {
		return domain.Attachment{}, domain.ErrAttachmentStoreUnavailable
	}
	if strings.TrimSpace(cmd.DocumentID) == "" || strings.TrimSpace(cmd.FileName) == "" || len(cmd.Content) == 0 {
		return domain.Attachment{}, domain.ErrInvalidAttachment
	}
	if len(cmd.Content) > 10*1024*1024 {
		return domain.Attachment{}, domain.ErrInvalidAttachment
	}

	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
	if err != nil {
		return domain.Attachment{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentUploadAttachment)
	if err != nil {
		return domain.Attachment{}, err
	}
	if !allowed {
		return domain.Attachment{}, domain.ErrDocumentNotFound
	}

	attachmentID := newAttachmentID()
	storageKey := attachmentStorageKey(doc.ID, attachmentID, cmd.FileName)
	attachment := domain.Attachment{
		ID:          attachmentID,
		DocumentID:  doc.ID,
		FileName:    strings.TrimSpace(cmd.FileName),
		ContentType: normalizeContentType(cmd.ContentType),
		SizeBytes:   int64(len(cmd.Content)),
		StorageKey:  storageKey,
		UploadedBy:  strings.TrimSpace(cmd.UploadedBy),
		CreatedAt:   s.clock.Now(),
	}
	if attachment.UploadedBy == "" {
		attachment.UploadedBy = authn.UserIDFromContext(ctx)
	}

	if err := s.attachmentStore.Save(ctx, storageKey, cmd.Content); err != nil {
		return domain.Attachment{}, err
	}
	if err := s.repo.CreateAttachment(ctx, attachment); err != nil {
		_ = s.attachmentStore.Delete(ctx, storageKey)
		return domain.Attachment{}, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-attachment-create-%s", attachment.ID),
			EventType:         "document.attachment.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: attachment.CreatedAt.Format(time.RFC3339),
			Version:           1,
			IdempotencyKey:    fmt.Sprintf("doc-attachment-create-%s", attachment.ID),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id":   doc.ID,
				"attachment_id": attachment.ID,
				"file_name":     attachment.FileName,
				"content_type":  attachment.ContentType,
				"size_bytes":    attachment.SizeBytes,
			},
		})
	}

	return attachment, nil
}

func (s *Service) ListAttachmentsAuthorized(ctx context.Context, documentID string) ([]domain.Attachment, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListAttachments(ctx, doc.ID)
}

func (s *Service) GetAttachmentAuthorized(ctx context.Context, documentID, attachmentID string) (domain.Attachment, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return domain.Attachment{}, err
	}
	attachment, err := s.repo.GetAttachment(ctx, strings.TrimSpace(attachmentID))
	if err != nil {
		return domain.Attachment{}, err
	}
	if attachment.DocumentID != doc.ID {
		return domain.Attachment{}, domain.ErrAttachmentNotFound
	}
	return attachment, nil
}

func (s *Service) OpenAttachmentContent(ctx context.Context, attachmentID string) (domain.Attachment, []byte, error) {
	if s.attachmentStore == nil {
		return domain.Attachment{}, nil, domain.ErrAttachmentStoreUnavailable
	}
	attachment, err := s.repo.GetAttachment(ctx, strings.TrimSpace(attachmentID))
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	reader, err := s.attachmentStore.Open(ctx, attachment.StorageKey)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	return attachment, content, nil
}

func (s *Service) ListAccessPolicies(ctx context.Context, resourceScope, resourceID string) ([]domain.AccessPolicy, error) {
	resourceScope = normalizeResourceScope(resourceScope)
	resourceID = strings.TrimSpace(resourceID)
	if resourceScope == "" || resourceID == "" {
		return nil, domain.ErrInvalidCommand
	}
	if !isKnownResourceScope(resourceScope) {
		return nil, domain.ErrInvalidAccessPolicy
	}
	return s.repo.ListAccessPolicies(ctx, resourceScope, resourceID)
}

func (s *Service) ReplaceAccessPolicies(ctx context.Context, resourceScope, resourceID string, policies []domain.AccessPolicy) error {
	resourceScope = normalizeResourceScope(resourceScope)
	resourceID = strings.TrimSpace(resourceID)
	if resourceScope == "" || resourceID == "" {
		return domain.ErrInvalidCommand
	}
	if !isKnownResourceScope(resourceScope) {
		return domain.ErrInvalidAccessPolicy
	}

	normalized := make([]domain.AccessPolicy, 0, len(policies))
	for _, policy := range policies {
		item, ok := normalizeAccessPolicy(resourceScope, resourceID, policy)
		if !ok {
			return domain.ErrInvalidAccessPolicy
		}
		normalized = append(normalized, item)
	}
	return s.repo.ReplaceAccessPolicies(ctx, resourceScope, resourceID, normalized)
}

func (s *Service) isKnownDocumentType(ctx context.Context, code string) bool {
	items, err := s.ListDocumentProfiles(ctx)
	if err != nil {
		return false
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Code), code) {
			return true
		}
	}
	return false
}

func (s *Service) resolveDocumentProfile(ctx context.Context, preferredProfile, legacyDocumentType string) (domain.DocumentProfile, error) {
	code := strings.ToLower(strings.TrimSpace(preferredProfile))
	if code == "" {
		code = strings.ToLower(strings.TrimSpace(legacyDocumentType))
	}
	if code == "" {
		return domain.DocumentProfile{}, domain.ErrInvalidDocumentType
	}

	items, err := s.ListDocumentProfiles(ctx)
	if err != nil {
		return domain.DocumentProfile{}, err
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Code), code) {
			return item, nil
		}
	}
	return domain.DocumentProfile{}, domain.ErrInvalidDocumentType
}

func (s *Service) resolveTaxonomy(ctx context.Context, processAreaCode, subjectCode string) (string, string, error) {
	processAreaCode = strings.ToLower(strings.TrimSpace(processAreaCode))
	subjectCode = strings.ToLower(strings.TrimSpace(subjectCode))

	if processAreaCode == "" && subjectCode == "" {
		return "", "", nil
	}

	var selectedArea domain.ProcessArea
	if processAreaCode != "" {
		areas, err := s.ListProcessAreas(ctx)
		if err != nil {
			return "", "", err
		}
		found := false
		for _, item := range areas {
			if strings.EqualFold(strings.TrimSpace(item.Code), processAreaCode) {
				selectedArea = item
				found = true
				break
			}
		}
		if !found {
			return "", "", domain.ErrInvalidCommand
		}
	}

	if subjectCode == "" {
		return processAreaCode, "", nil
	}

	subjects, err := s.ListSubjects(ctx)
	if err != nil {
		return "", "", err
	}
	for _, item := range subjects {
		if !strings.EqualFold(strings.TrimSpace(item.Code), subjectCode) {
			continue
		}
		if processAreaCode != "" && !strings.EqualFold(strings.TrimSpace(item.ProcessAreaCode), selectedArea.Code) {
			return "", "", domain.ErrInvalidCommand
		}
		if processAreaCode == "" {
			processAreaCode = strings.ToLower(strings.TrimSpace(item.ProcessAreaCode))
		}
		return processAreaCode, subjectCode, nil
	}
	return "", "", domain.ErrInvalidCommand
}

func (s *Service) resolveActiveProfileSchema(ctx context.Context, profileCode string) (domain.DocumentProfileSchemaVersion, error) {
	items, err := s.ListDocumentProfileSchemas(ctx, profileCode)
	if err != nil {
		return domain.DocumentProfileSchemaVersion{}, err
	}
	for _, item := range items {
		if item.IsActive {
			return item, nil
		}
	}
	if len(items) == 0 {
		return domain.DocumentProfileSchemaVersion{}, domain.ErrInvalidCommand
	}
	return items[len(items)-1], nil
}

func (s *Service) resolveDocumentProfileSchema(ctx context.Context, profileCode string, version int) (domain.DocumentProfileSchemaVersion, bool, error) {
	items, err := s.ListDocumentProfileSchemas(ctx, profileCode)
	if err != nil {
		return domain.DocumentProfileSchemaVersion{}, false, err
	}
	if version > 0 {
		for _, item := range items {
			if item.Version == version {
				return item, true, nil
			}
		}
	}
	for _, item := range items {
		if item.IsActive {
			return item, true, nil
		}
	}
	if len(items) == 0 {
		return domain.DocumentProfileSchemaVersion{}, false, nil
	}
	return items[len(items)-1], true, nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, tag := range tags {
		normalized := strings.TrimSpace(tag)
		if normalized == "" {
			continue
		}
		key := strings.ToLower(normalized)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizeMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		out[trimmed] = value
	}
	return out
}

func validateMetadata(rules []domain.MetadataFieldRule, metadata map[string]any) error {
	if len(rules) == 0 {
		return nil
	}
	for _, rule := range rules {
		value, exists := metadata[rule.Name]
		if rule.Required && (!exists || isEmptyMetadataValue(value)) {
			return domain.ErrInvalidMetadata
		}
		if exists && !matchesMetadataType(rule.Type, value) {
			return domain.ErrInvalidMetadata
		}
	}
	return nil
}

func validateContentSchema(schema map[string]any, content map[string]any) error {
	if len(schema) == 0 {
		return nil
	}
	rawSections, ok := schema["sections"]
	if !ok {
		return nil
	}
	sections, ok := rawSections.([]any)
	if !ok {
		return nil
	}
	for _, rawSection := range sections {
		section, ok := rawSection.(map[string]any)
		if !ok {
			continue
		}
		sectionKey, _ := asSchemaString(section["key"])
		if sectionKey == "" {
			continue
		}
		sectionValue, _ := content[sectionKey].(map[string]any)
		if sectionValue == nil {
			sectionValue = map[string]any{}
		}
		fields, _ := section["fields"].([]any)
		for _, rawField := range fields {
			field, ok := rawField.(map[string]any)
			if !ok {
				continue
			}
			if err := validateContentField(field, sectionValue); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateContentField(field map[string]any, container map[string]any) error {
	key, _ := asSchemaString(field["key"])
	if key == "" {
		return nil
	}
	fieldType, _ := asSchemaString(field["type"])
	required, _ := field["required"].(bool)
	value, exists := container[key]
	if !exists || isEmptyContentValue(value) {
		if required {
			return domain.ErrInvalidNativeContent
		}
		return nil
	}

	switch fieldType {
	case "text", "textarea":
		if _, ok := value.(string); !ok {
			return domain.ErrInvalidNativeContent
		}
	case "number":
		if !isNumericValue(value) {
			return domain.ErrInvalidNativeContent
		}
	case "select":
		selected, ok := value.(string)
		if !ok {
			return domain.ErrInvalidNativeContent
		}
		options := normalizeSchemaStringList(field["options"])
		if len(options) > 0 && !containsSchemaOption(options, selected) {
			return domain.ErrInvalidNativeContent
		}
	case "array":
		items, ok := value.([]any)
		if !ok {
			return domain.ErrInvalidNativeContent
		}
		if required && len(items) == 0 {
			return domain.ErrInvalidNativeContent
		}
		itemType, _ := asSchemaString(field["itemType"])
		if itemType != "" {
			for _, item := range items {
				if isEmptyContentValue(item) {
					continue
				}
				if !matchesContentType(itemType, item, field) {
					return domain.ErrInvalidNativeContent
				}
			}
		}
	case "table":
		rows, ok := value.([]any)
		if !ok {
			return domain.ErrInvalidNativeContent
		}
		if required && len(rows) == 0 {
			return domain.ErrInvalidNativeContent
		}
		columns, _ := field["columns"].([]any)
		for _, rawRow := range rows {
			row, ok := rawRow.(map[string]any)
			if !ok {
				return domain.ErrInvalidNativeContent
			}
			for _, rawColumn := range columns {
				column, ok := rawColumn.(map[string]any)
				if !ok {
					continue
				}
				if err := validateContentField(column, row); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func matchesContentType(fieldType string, value any, field map[string]any) bool {
	switch fieldType {
	case "text", "textarea":
		_, ok := value.(string)
		return ok
	case "number":
		return isNumericValue(value)
	case "select":
		selected, ok := value.(string)
		if !ok {
			return false
		}
		options := normalizeSchemaStringList(field["options"])
		if len(options) == 0 {
			return true
		}
		return containsSchemaOption(options, selected)
	default:
		return true
	}
}

func isNumericValue(value any) bool {
	switch value.(type) {
	case float64, float32, int, int32, int64, uint, uint32, uint64, json.Number:
		return true
	default:
		return false
	}
}

func isEmptyContentValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}

func asSchemaString(value any) (string, bool) {
	typed, ok := value.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(typed)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func normalizeSchemaStringList(value any) []string {
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			str, ok := item.(string)
			if !ok {
				continue
			}
			if trimmed := strings.TrimSpace(str); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	default:
		return nil
	}
}

func containsSchemaOption(options []string, value string) bool {
	for _, option := range options {
		if strings.EqualFold(option, value) {
			return true
		}
	}
	return false
}

func filterDefaultSchemas(profileCode string) []domain.DocumentProfileSchemaVersion {
	if profileCode == "" {
		return domain.DefaultDocumentProfileSchemas()
	}
	items := domain.DefaultDocumentProfileSchemas()
	filtered := make([]domain.DocumentProfileSchemaVersion, 0, len(items))
	for _, item := range items {
		if strings.EqualFold(item.ProfileCode, profileCode) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func contentHash(value string) string {
	sum := md5.Sum([]byte(value))
	return fmt.Sprintf("%x", sum[:])
}

func buildAudiencePolicies(cmd domain.CreateDocumentCommand, classification string) ([]domain.AccessPolicy, error) {
	audience, err := normalizeAudience(cmd, classification)
	if err != nil {
		return nil, err
	}
	if audience == nil {
		return nil, nil
	}

	policies := make([]domain.AccessPolicy, 0)
	seen := map[string]struct{}{}
	addPolicy := func(subjectType, subjectID, capability string) error {
		item, ok := normalizeAccessPolicy(domain.ResourceScopeDocument, cmd.DocumentID, domain.AccessPolicy{
			SubjectType: subjectType,
			SubjectID:   subjectID,
			Capability:  capability,
			Effect:      domain.PolicyEffectAllow,
		})
		if !ok {
			return domain.ErrInvalidAccessPolicy
		}
		key := item.SubjectType + "|" + item.SubjectID + "|" + item.Capability
		if _, exists := seen[key]; exists {
			return nil
		}
		seen[key] = struct{}{}
		policies = append(policies, item)
		return nil
	}

	ownerID := strings.TrimSpace(cmd.OwnerID)
	if ownerID == "" {
		return nil, domain.ErrInvalidAccessPolicy
	}
	for _, capability := range []string{
		domain.CapabilityDocumentView,
		domain.CapabilityDocumentEdit,
		domain.CapabilityDocumentUploadAttachment,
	} {
		if err := addPolicy(domain.SubjectTypeUser, ownerID, capability); err != nil {
			return nil, err
		}
	}

	const adminRoleCode = "admin"
	for _, capability := range []string{
		domain.CapabilityDocumentView,
		domain.CapabilityDocumentEdit,
		domain.CapabilityDocumentUploadAttachment,
	} {
		if err := addPolicy(domain.SubjectTypeRole, adminRoleCode, capability); err != nil {
			return nil, err
		}
	}

	switch audience.Mode {
	case domain.AudienceModeDepartment:
		for _, dept := range audience.DepartmentCodes {
			if err := addPolicy(domain.SubjectTypeRole, "dept:"+dept, domain.CapabilityDocumentView); err != nil {
				return nil, err
			}
		}
	case domain.AudienceModeAreas:
		for _, dept := range audience.DepartmentCodes {
			for _, area := range audience.ProcessAreaCodes {
				compoundRole := "dept:" + dept + ":area:" + area
				if err := addPolicy(domain.SubjectTypeRole, compoundRole, domain.CapabilityDocumentView); err != nil {
					return nil, err
				}
			}
		}
	case domain.AudienceModeExplicit:
		for _, role := range audience.RoleCodes {
			if err := addPolicy(domain.SubjectTypeRole, role, domain.CapabilityDocumentView); err != nil {
				return nil, err
			}
		}
		for _, userID := range audience.UserIDs {
			if err := addPolicy(domain.SubjectTypeUser, userID, domain.CapabilityDocumentView); err != nil {
				return nil, err
			}
		}
	}

	return policies, nil
}

func normalizeAudience(cmd domain.CreateDocumentCommand, classification string) (*domain.DocumentAudience, error) {
	mode := ""
	if cmd.Audience != nil {
		mode = strings.ToUpper(strings.TrimSpace(cmd.Audience.Mode))
	}

	if mode == "" {
		switch classification {
		case domain.ClassificationConfidential, domain.ClassificationRestricted:
			dept := strings.ToLower(strings.TrimSpace(cmd.Department))
			if dept == "" {
				return nil, domain.ErrInvalidAccessPolicy
			}
			if classification == domain.ClassificationRestricted {
				area := strings.ToLower(strings.TrimSpace(cmd.ProcessArea))
				if area == "" {
					return nil, domain.ErrInvalidAccessPolicy
				}
				return &domain.DocumentAudience{
					Mode:             domain.AudienceModeAreas,
					DepartmentCodes:  []string{dept},
					ProcessAreaCodes: []string{area},
				}, nil
			}
			return &domain.DocumentAudience{
				Mode:            domain.AudienceModeDepartment,
				DepartmentCodes: []string{dept},
			}, nil
		default:
			return nil, nil
		}
	}

	switch mode {
	case domain.AudienceModeInternal:
		return nil, nil
	case domain.AudienceModeDepartment:
		if classification == domain.ClassificationRestricted {
			return nil, domain.ErrInvalidAccessPolicy
		}
		departments := normalizeCodeList(cmd.Audience.DepartmentCodes)
		if len(departments) == 0 {
			if dept := strings.ToLower(strings.TrimSpace(cmd.Department)); dept != "" {
				departments = []string{dept}
			}
		}
		if len(departments) == 0 {
			return nil, domain.ErrInvalidAccessPolicy
		}
		return &domain.DocumentAudience{
			Mode:            domain.AudienceModeDepartment,
			DepartmentCodes: departments,
		}, nil
	case domain.AudienceModeAreas:
		departments := normalizeCodeList(cmd.Audience.DepartmentCodes)
		if len(departments) == 0 {
			if dept := strings.ToLower(strings.TrimSpace(cmd.Department)); dept != "" {
				departments = []string{dept}
			}
		}
		areas := normalizeCodeList(cmd.Audience.ProcessAreaCodes)
		if len(areas) == 0 {
			if area := strings.ToLower(strings.TrimSpace(cmd.ProcessArea)); area != "" {
				areas = []string{area}
			}
		}
		if len(departments) == 0 || len(areas) == 0 {
			return nil, domain.ErrInvalidAccessPolicy
		}
		return &domain.DocumentAudience{
			Mode:             domain.AudienceModeAreas,
			DepartmentCodes:  departments,
			ProcessAreaCodes: areas,
		}, nil
	case domain.AudienceModeExplicit:
		roleCodes := normalizeCodeList(cmd.Audience.RoleCodes)
		userIDs := normalizeUserIDList(cmd.Audience.UserIDs)
		if len(roleCodes) == 0 && len(userIDs) == 0 {
			return nil, domain.ErrInvalidAccessPolicy
		}
		return &domain.DocumentAudience{
			Mode:      domain.AudienceModeExplicit,
			RoleCodes: roleCodes,
			UserIDs:   userIDs,
		}, nil
	default:
		return nil, domain.ErrInvalidAccessPolicy
	}
}

func normalizeCodeList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeUserIDList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeAccessPolicy(resourceScope, resourceID string, policy domain.AccessPolicy) (domain.AccessPolicy, bool) {
	subjectType := strings.ToLower(strings.TrimSpace(policy.SubjectType))
	subjectID := strings.TrimSpace(policy.SubjectID)
	capability := strings.TrimSpace(policy.Capability)
	effect := strings.ToLower(strings.TrimSpace(policy.Effect))

	if !isKnownSubjectType(subjectType) || subjectID == "" || !isKnownCapability(capability) || !isKnownEffect(effect) {
		return domain.AccessPolicy{}, false
	}

	return domain.AccessPolicy{
		SubjectType:   subjectType,
		SubjectID:     subjectID,
		ResourceScope: resourceScope,
		ResourceID:    resourceID,
		Capability:    capability,
		Effect:        effect,
	}, true
}

func normalizeResourceScope(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func isKnownResourceScope(raw string) bool {
	switch raw {
	case domain.ResourceScopeDocument, domain.ResourceScopeDocumentType, domain.ResourceScopeArea:
		return true
	default:
		return false
	}
}

func isKnownSubjectType(raw string) bool {
	switch raw {
	case domain.SubjectTypeUser, domain.SubjectTypeRole, domain.SubjectTypeGroup:
		return true
	default:
		return false
	}
}

func isKnownCapability(raw string) bool {
	switch raw {
	case domain.CapabilityDocumentCreate,
		domain.CapabilityDocumentView,
		domain.CapabilityDocumentEdit,
		domain.CapabilityDocumentUploadAttachment,
		domain.CapabilityDocumentChangeWorkflow,
		domain.CapabilityDocumentManagePermissions:
		return true
	default:
		return false
	}
}

func (s *Service) isAllowedForCreate(ctx context.Context, cmd domain.CreateDocumentCommand) bool {
	if shouldBypassPolicy(ctx) {
		return true
	}
	items, err := s.policiesForCreate(ctx, cmd, domain.CapabilityDocumentCreate)
	if err != nil {
		return false
	}
	return decidePolicies(ctx, items)
}

func (s *Service) isAllowed(ctx context.Context, doc domain.Document, capability string) (bool, error) {
	if shouldBypassPolicy(ctx) {
		return true, nil
	}
	items, err := s.policiesForDocument(ctx, doc, capability)
	if err != nil {
		return false, err
	}
	return decidePolicies(ctx, items), nil
}

func (s *Service) policiesForDocument(ctx context.Context, doc domain.Document, capability string) ([]domain.AccessPolicy, error) {
	scopes := []struct {
		scope string
		id    string
	}{
		{scope: domain.ResourceScopeDocument, id: doc.ID},
		{scope: domain.ResourceScopeDocumentType, id: doc.DocumentProfile},
		{scope: domain.ResourceScopeArea, id: areaResourceID(doc.BusinessUnit, doc.Department)},
	}
	return s.loadPoliciesForScopes(ctx, scopes, capability)
}

func (s *Service) policiesForCreate(ctx context.Context, cmd domain.CreateDocumentCommand, capability string) ([]domain.AccessPolicy, error) {
	scopes := []struct {
		scope string
		id    string
	}{
		{scope: domain.ResourceScopeDocumentType, id: strings.ToLower(strings.TrimSpace(firstNonEmpty(cmd.DocumentProfile, cmd.DocumentType)))},
		{scope: domain.ResourceScopeArea, id: areaResourceID(cmd.BusinessUnit, cmd.Department)},
	}
	return s.loadPoliciesForScopes(ctx, scopes, capability)
}

func (s *Service) loadPoliciesForScopes(ctx context.Context, scopes []struct {
	scope string
	id    string
}, capability string) ([]domain.AccessPolicy, error) {
	var out []domain.AccessPolicy
	for _, scope := range scopes {
		if strings.TrimSpace(scope.id) == "" {
			continue
		}
		items, err := s.repo.ListAccessPolicies(ctx, scope.scope, scope.id)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.Capability == capability {
				out = append(out, item)
			}
		}
	}
	return out, nil
}

func decidePolicies(ctx context.Context, items []domain.AccessPolicy) bool {
	if len(items) == 0 {
		return true
	}
	userID := authn.UserIDFromContext(ctx)
	roles := authn.RolesFromContext(ctx)
	rolesSet := map[string]struct{}{}
	for _, role := range roles {
		rolesSet[strings.ToLower(strings.TrimSpace(role))] = struct{}{}
	}

	matchedAny := false
	for _, item := range items {
		if !matchesPolicySubject(item, userID, rolesSet) {
			continue
		}
		matchedAny = true
		if item.Effect == domain.PolicyEffectDeny {
			return false
		}
	}
	if matchedAny {
		return true
	}
	return false
}

func matchesPolicySubject(item domain.AccessPolicy, userID string, rolesSet map[string]struct{}) bool {
	switch item.SubjectType {
	case domain.SubjectTypeUser:
		return strings.EqualFold(item.SubjectID, userID)
	case domain.SubjectTypeRole:
		_, ok := rolesSet[strings.ToLower(strings.TrimSpace(item.SubjectID))]
		return ok
	default:
		return false
	}
}

func (s *Service) latestVersion(ctx context.Context, documentID string) (domain.Version, error) {
	versions, err := s.repo.ListVersions(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}
	if len(versions) == 0 {
		return domain.Version{}, domain.ErrVersionNotFound
	}
	return versions[len(versions)-1], nil
}

func isVersioningAllowed(doc domain.Document) bool {
	return doc.Status == domain.StatusDraft || doc.Status == domain.StatusInReview
}

func documentContentStorageKey(documentID string, version int, extension string) string {
	return fmt.Sprintf("documents/%s/versions/%d/content.%s", documentID, version, strings.TrimPrefix(extension, "."))
}

func isDocxPayload(content []byte) bool {
	return len(content) >= 4 && bytes.Equal(content[:4], []byte("PK\x03\x04"))
}

func extractDocxText(content []byte) string {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return ""
	}
	for _, file := range reader.File {
		if file.Name != "word/document.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return ""
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		decoder := xml.NewDecoder(bytes.NewReader(data))
		var builder strings.Builder
		for {
			token, err := decoder.Token()
			if err != nil {
				break
			}
			switch value := token.(type) {
			case xml.CharData:
				text := strings.TrimSpace(string(value))
				if text != "" {
					if builder.Len() > 0 {
						builder.WriteString(" ")
					}
					builder.WriteString(text)
				}
			}
		}
		return builder.String()
	}
	return ""
}

func (s *Service) renderProfileTemplate(ctx context.Context, profileCode string, data map[string]any, convertTo, traceID string) ([]byte, error) {
	if s.carboneClient == nil || s.carboneTemplates == nil {
		return nil, fmt.Errorf("carbone client not configured")
	}
	normalized := strings.ToLower(strings.TrimSpace(profileCode))
	templateID, ok := s.carboneTemplates.TemplateID(normalized)
	if !ok {
		return nil, fmt.Errorf("template not registered for profile %s", normalized)
	}
	renderID, err := s.carboneClient.RenderTemplate(ctx, traceID, templateID, data, convertTo)
	if err != nil {
		return nil, err
	}
	return s.carboneClient.DownloadRender(ctx, traceID, renderID)
}

func (s *Service) renderDocumentPDF(ctx context.Context, doc domain.Document, version domain.Version, content map[string]any, traceID string) ([]byte, error) {
	data := buildDocumentTemplateData(doc, content)
	data["version"] = version.Number
	data["createdAt"] = version.CreatedAt.Format(time.RFC3339)
	return s.renderProfileTemplate(ctx, doc.DocumentProfile, data, "pdf", traceID)
}

func (s *Service) convertDocxToPDF(ctx context.Context, content []byte, traceID string) ([]byte, error) {
	if s.carboneClient == nil {
		return nil, fmt.Errorf("carbone client not configured")
	}
	tmpFile, err := os.CreateTemp("", "metaldocs-docx-*.docx")
	if err != nil {
		return nil, fmt.Errorf("create temp docx: %w", err)
	}
	tempPath := tmpFile.Name()
	if _, err := tmpFile.Write(content); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("write temp docx: %w", err)
	}
	_ = tmpFile.Close()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	templateID, err := s.carboneClient.RegisterTemplate(ctx, traceID, tempPath)
	if err != nil {
		return nil, err
	}
	renderID, err := s.carboneClient.RenderTemplate(ctx, traceID, templateID, map[string]any{}, "pdf")
	if err != nil {
		return nil, err
	}
	return s.carboneClient.DownloadRender(ctx, traceID, renderID)
}

func buildDocumentTemplateData(doc domain.Document, content map[string]any) map[string]any {
	data := map[string]any{
		"documentId":     doc.ID,
		"title":          doc.Title,
		"profile":        doc.DocumentProfile,
		"family":         doc.DocumentFamily,
		"processArea":    doc.ProcessArea,
		"subject":        doc.Subject,
		"owner":          doc.OwnerID,
		"businessUnit":   doc.BusinessUnit,
		"department":     doc.Department,
		"classification": doc.Classification,
	}
	if content != nil {
		data["content"] = content
	}
	return data
}

func areaResourceID(businessUnit, department string) string {
	return strings.TrimSpace(businessUnit) + ":" + strings.TrimSpace(department)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func shouldBypassPolicy(ctx context.Context) bool {
	return authn.UserIDFromContext(ctx) == "" && len(authn.RolesFromContext(ctx)) == 0
}

func isKnownEffect(raw string) bool {
	switch raw {
	case domain.PolicyEffectAllow, domain.PolicyEffectDeny:
		return true
	default:
		return false
	}
}

func isEmptyMetadataValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	default:
		return false
	}
}

func matchesMetadataType(expected string, value any) bool {
	switch expected {
	case "string":
		_, ok := value.(string)
		return ok && !isEmptyMetadataValue(value)
	case "date":
		raw, ok := value.(string)
		if !ok {
			return false
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return false
		}
		if _, err := time.Parse("2006-01-02", raw); err == nil {
			return true
		}
		_, err := time.Parse(time.RFC3339, raw)
		return err == nil
	default:
		return true
	}
}

var attachmentSafeChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func newAttachmentID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("att_%d", time.Now().UTC().UnixNano())
	}
	return "att_" + hex.EncodeToString(buf)
}

func attachmentStorageKey(documentID, attachmentID, fileName string) string {
	safeName := attachmentSafeChars.ReplaceAllString(strings.TrimSpace(filepath.Base(fileName)), "_")
	if safeName == "" {
		safeName = "attachment.bin"
	}
	return strings.TrimSpace(documentID) + "/" + attachmentID + "/" + safeName
}

func normalizeContentType(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "application/octet-stream"
	}
	return normalized
}
