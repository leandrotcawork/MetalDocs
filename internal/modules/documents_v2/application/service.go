package application

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/documents_v2/repository"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
	registrydomain "metaldocs/internal/modules/registry/domain"
)

// Type aliases so handlers depend only on application types.
type PendingCommitMeta = repository.PendingCommitMeta
type CommitResult = repository.CommitResult
type RestoreResult = repository.RestoreResult

type Repository interface {
	CreateDocument(ctx context.Context, d *domain.Document, initialContentHash string) (docID, revID, sessionID string, err error)
	SetRevisionStorageKey(ctx context.Context, revID, storageKey string) error
	GetDocument(ctx context.Context, tenantID, id string) (*domain.Document, error)
	UpdateDocumentName(ctx context.Context, tenantID, docID, name string) error
	ListDocuments(ctx context.Context, tenantID string) ([]domain.Document, error)
	ListDocumentsForUser(ctx context.Context, tenantID, userID string) ([]domain.Document, error)
	UpdateDocumentStatus(ctx context.Context, tenantID, id string, cur, next domain.DocumentStatus, stampTime bool) error
	IsDocumentOwner(ctx context.Context, tenantID, docID, userID string) (bool, error)
	AcquireSession(ctx context.Context, tenantID, docID, userID string) (*domain.Session, error)
	HeartbeatSession(ctx context.Context, sessionID, userID string) error
	ReleaseSession(ctx context.Context, sessionID, userID string) error
	ForceReleaseSession(ctx context.Context, sessionID string) error
	ExpireStaleSessions(ctx context.Context, now time.Time) (int, error)
	PresignReserve(ctx context.Context, sessionID, userID, docID, baseRev, contentHash, storageKey string, expiresAt time.Time) (string, error)
	GetPendingForCommit(ctx context.Context, pendingID string) (*PendingCommitMeta, error)
	CommitUpload(ctx context.Context, sessionID, userID, docID, pendingID, serverComputedHash string, formDataSnapshot []byte) (*CommitResult, error)
	CreateCheckpoint(ctx context.Context, docID, actorUserID, label string) (*domain.Checkpoint, error)
	ListCheckpoints(ctx context.Context, docID string) ([]domain.Checkpoint, error)
	RestoreCheckpoint(ctx context.Context, docID, actorUserID string, versionNum int) (*RestoreResult, error)
	GetRevision(ctx context.Context, docID, revID string) (*domain.Revision, error)
	DeleteExpiredPending(ctx context.Context, olderThan time.Time) (int, error)
	CreateComment(ctx context.Context, tenantID, documentID, authorID string, in domain.CommentCreateInput) (*domain.Comment, error)
	ListComments(ctx context.Context, tenantID, documentID string) ([]domain.Comment, error)
	UpdateComment(ctx context.Context, tenantID, documentID string, libraryID int, userID string, in domain.CommentUpdateInput) (*domain.Comment, error)
	DeleteComment(ctx context.Context, tenantID, documentID string, libraryID int) error
}

type DocgenRenderer interface {
	RenderDocx(ctx context.Context, templateDocxKey, schemaKey, outputKey string, formData json.RawMessage) (contentHash string, sizeBytes int64, unreplaced []string, err error)
}

type Presigner interface {
	PresignRevisionPUT(ctx context.Context, tenantID, docID, contentHash string) (url, storageKey string, err error)
	PresignObjectGET(ctx context.Context, storageKey string) (url string, err error)
	AdoptTempObject(ctx context.Context, tmpKey, finalKey string) error
	DeleteObject(ctx context.Context, key string) error
	HashObject(ctx context.Context, key string) (string, error)
}

type TemplateReader interface {
	GetPublishedVersion(ctx context.Context, tenantID, templateVersionID string) (docxKey, schemaKey, schemaJSON string, err error)
}

type FormValidator interface {
	Validate(schemaJSON string, formData json.RawMessage) (valid bool, errs []string, err error)
}

type Audit interface {
	Write(ctx context.Context, tenantID, actorID, action, docID string, meta any)
}

// RegistryReader loads a ControlledDocument for validation at create time.
type RegistryReader interface {
	GetByID(ctx context.Context, tenantID, id string) (*registrydomain.ControlledDocument, error)
}

// AuthorizationChecker validates that the actor can perform an action on a resource.
type AuthorizationChecker interface {
	Check(ctx context.Context, userID, tenantID string, cap iamdomain.Capability, res iamapp.ResourceCtx) error
}

type ProfileDefaultTemplateReader interface {
	GetDefaultTemplateVersionID(ctx context.Context, tenantID, profileCode string) (*string, *string, error)
	// returns (*templateVersionID, *templateVersionStatus, error)
}

type Service struct {
	repo             Repository
	docgen           DocgenRenderer
	presigner        Presigner
	tpl              TemplateReader
	fv               FormValidator
	audit            Audit
	registry         RegistryReader
	authz            AuthorizationChecker
	profileTemplates ProfileDefaultTemplateReader
}

func New(r Repository, d DocgenRenderer, p Presigner, t TemplateReader, fv FormValidator, a Audit) *Service {
	return &Service{
		repo:      r,
		docgen:    d,
		presigner: p,
		tpl:       t,
		fv:        fv,
		audit:     a,
	}
}

func NewService(
	r Repository,
	d DocgenRenderer,
	p Presigner,
	t TemplateReader,
	fv FormValidator,
	a Audit,
	reg RegistryReader,
	authz AuthorizationChecker,
	profileTemplates ProfileDefaultTemplateReader,
) *Service {
	return &Service{
		repo:             r,
		docgen:           d,
		presigner:        p,
		tpl:              t,
		fv:               fv,
		audit:            a,
		registry:         reg,
		authz:            authz,
		profileTemplates: profileTemplates,
	}
}

var ErrControlledDocumentRequired = errors.New("controlled_document_id is required")
var errRegistryReaderNotConfigured = errors.New("registry reader not configured")
var errAuthorizationCheckerNotConfigured = errors.New("authorization checker not configured")
var errProfileTemplateReaderNotConfigured = errors.New("profile default template reader not configured")

type CreateDocumentInput struct {
	TenantID             string
	ActorUserID          string
	ControlledDocumentID string
	TemplateVersionID    string
	Name                 string
	FormData             json.RawMessage
}

type CreateDocumentCmd = CreateDocumentInput

type CreateDocumentResult struct {
	DocumentID        string
	InitialRevisionID string
	SessionID         string
}

func buildDocumentForCreate(cmd CreateDocumentInput, cd *registrydomain.ControlledDocument, resolvedTemplateVersionID string) domain.Document {
	return domain.Document{
		TenantID:                cmd.TenantID,
		TemplateVersionID:       resolvedTemplateVersionID,
		Name:                    cmd.Name,
		FormDataJSON:            cmd.FormData,
		CreatedBy:               cmd.ActorUserID,
		ControlledDocumentID:    &cmd.ControlledDocumentID,
		ProfileCodeSnapshot:     &cd.ProfileCode,
		ProcessAreaCodeSnapshot: &cd.ProcessAreaCode,
	}
}

func (s *Service) CreateDocument(ctx context.Context, cmd CreateDocumentInput) (res *CreateDocumentResult, err error) {
	if strings.TrimSpace(cmd.ControlledDocumentID) == "" {
		return nil, ErrControlledDocumentRequired
	}
	if s.registry == nil {
		return nil, errRegistryReaderNotConfigured
	}
	if s.authz == nil {
		return nil, errAuthorizationCheckerNotConfigured
	}
	if s.profileTemplates == nil {
		return nil, errProfileTemplateReaderNotConfigured
	}

	cd, err := s.registry.GetByID(ctx, cmd.TenantID, cmd.ControlledDocumentID)
	if err != nil {
		return nil, err
	}
	if !cd.IsActive() {
		return nil, registrydomain.ErrCDNotActive
	}

	if err := s.authz.Check(ctx, cmd.ActorUserID, cmd.TenantID, iamdomain.CapDocumentCreate, iamapp.ResourceCtx{
		AreaCode: cd.ProcessAreaCode,
	}); err != nil {
		return nil, err
	}

	defaultTemplateID, defaultTemplateStatus, err := s.profileTemplates.GetDefaultTemplateVersionID(ctx, cmd.TenantID, cd.ProfileCode)
	if err != nil {
		return nil, err
	}

	var overrideTemplate *registrydomain.TemplateVersionCandidate
	if cd.OverrideTemplateVersionID != nil {
		overrideStatus := "published"
		overrideTemplate = &registrydomain.TemplateVersionCandidate{
			ID:          *cd.OverrideTemplateVersionID,
			ProfileCode: cd.ProfileCode,
			Status:      &overrideStatus,
		}
	}

	var defaultTemplate *registrydomain.TemplateVersionCandidate
	if defaultTemplateID != nil {
		defaultTemplate = &registrydomain.TemplateVersionCandidate{
			ID:          *defaultTemplateID,
			ProfileCode: cd.ProfileCode,
			Status:      defaultTemplateStatus,
		}
	}

	resolution, err := registrydomain.Resolve(registrydomain.TemplateResolutionInput{
		ProfileCode:      cd.ProfileCode,
		OverrideTemplate: overrideTemplate,
		DefaultTemplate:  defaultTemplate,
	})
	if err != nil {
		return nil, err
	}
	resolvedTemplateVersionID := resolution.TemplateVersionID

	docxKey, schemaKey, schemaJSON, err := s.tpl.GetPublishedVersion(ctx, cmd.TenantID, resolvedTemplateVersionID)
	if err != nil {
		return nil, fmt.Errorf("template lookup: %w", err)
	}
	if schemaJSON != "" {
		ok, verrs, err := s.fv.Validate(schemaJSON, cmd.FormData)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("form_data_invalid: %v", verrs)
		}
	}

	var contentHash, finalKey string
	if s.docgen != nil {
		tmpKey := fmt.Sprintf("tenants/%s/documents/tmp/%s.docx", cmd.TenantID, uuid.New().String())
		var err error
		contentHash, _, _, err = s.docgen.RenderDocx(ctx, docxKey, schemaKey, tmpKey, cmd.FormData)
		if err != nil {
			return nil, fmt.Errorf("render: %w", err)
		}

		cleanupKey := tmpKey
		defer func() {
			if cleanupKey != "" {
				_ = s.presigner.DeleteObject(context.Background(), cleanupKey)
			}
		}()

		doc := buildDocumentForCreate(cmd, cd, resolvedTemplateVersionID)
		docID, revID, sessionID, err := s.repo.CreateDocument(ctx, &doc, contentHash)
		if err != nil {
			return nil, err
		}

		finalKey = fmt.Sprintf("tenants/%s/documents/%s/revisions/%s.docx", cmd.TenantID, docID, contentHash)
		if err := s.presigner.AdoptTempObject(ctx, tmpKey, finalKey); err != nil {
			return nil, fmt.Errorf("adopt tmp: %w", err)
		}
		cleanupKey = ""

		if err := s.repo.SetRevisionStorageKey(ctx, revID, finalKey); err != nil {
			return nil, fmt.Errorf("set revision key: %w", err)
		}

		s.audit.Write(ctx, cmd.TenantID, cmd.ActorUserID, "document.created", docID, map[string]any{"template_version_id": resolvedTemplateVersionID})
		return &CreateDocumentResult{DocumentID: docID, InitialRevisionID: revID, SessionID: sessionID}, nil
	}

	// docgen not configured: bootstrap document with template docx as initial revision
	h := sha256.New()
	h.Write([]byte(docxKey))
	contentHash = fmt.Sprintf("%x", h.Sum(nil))

	doc := buildDocumentForCreate(cmd, cd, resolvedTemplateVersionID)
	docID, revID, sessionID, err := s.repo.CreateDocument(ctx, &doc, contentHash)
	if err != nil {
		return nil, err
	}

	finalKey = docxKey // point to template docx directly
	if err := s.repo.SetRevisionStorageKey(ctx, revID, finalKey); err != nil {
		return nil, fmt.Errorf("set revision key: %w", err)
	}

	s.audit.Write(ctx, cmd.TenantID, cmd.ActorUserID, "document.created", docID, map[string]any{"template_version_id": resolvedTemplateVersionID})
	return &CreateDocumentResult{DocumentID: docID, InitialRevisionID: revID, SessionID: sessionID}, nil
}

func (s *Service) GetDocument(ctx context.Context, tenantID, id string) (*domain.Document, error) {
	return s.repo.GetDocument(ctx, tenantID, id)
}

func (s *Service) ListDocuments(ctx context.Context, tenantID string) ([]domain.Document, error) {
	return s.repo.ListDocuments(ctx, tenantID)
}

func (s *Service) ListDocumentsForUser(ctx context.Context, tenantID, userID string) ([]domain.Document, error) {
	return s.repo.ListDocumentsForUser(ctx, tenantID, userID)
}

func (s *Service) RenameDocument(ctx context.Context, tenantID, userID, docID, newName string) error {
	name := strings.TrimSpace(newName)
	if name == "" || len(name) > 255 {
		return domain.ErrInvalidName
	}
	doc, err := s.repo.GetDocument(ctx, tenantID, docID)
	if err != nil {
		return err
	}
	if doc.Status != domain.DocStatusDraft {
		return domain.ErrInvalidStateTransition
	}
	if err := s.repo.UpdateDocumentName(ctx, tenantID, docID, name); err != nil {
		return err
	}
	s.audit.Write(ctx, tenantID, userID, "document.renamed", docID, map[string]any{"name": name})
	return nil
}

func (s *Service) IsDocumentOwner(ctx context.Context, tenantID, docID, userID string) (bool, error) {
	return s.repo.IsDocumentOwner(ctx, tenantID, docID, userID)
}

func (s *Service) ListDocumentComments(ctx context.Context, tenantID, userID, documentID string) ([]domain.Comment, error) {
	return s.repo.ListComments(ctx, tenantID, documentID)
}

func (s *Service) AddDocumentComment(ctx context.Context, tenantID, userID, authorDisplay, documentID string, in domain.CommentCreateInput) (*domain.Comment, error) {
	if in.LibraryCommentID <= 0 {
		return nil, domain.ErrCommentInvalid
	}
	if len(in.ContentJSON) == 0 {
		return nil, domain.ErrCommentInvalid
	}
	trimmedAuthor := strings.TrimSpace(authorDisplay)
	if trimmedAuthor == "" || len(trimmedAuthor) > 255 {
		return nil, domain.ErrCommentInvalid
	}
	in.AuthorDisplay = trimmedAuthor
	return s.repo.CreateComment(ctx, tenantID, documentID, userID, in)
}

func (s *Service) UpdateDocumentComment(ctx context.Context, tenantID, userID, documentID string, libraryID int, in domain.CommentUpdateInput) (*domain.Comment, error) {
	if libraryID <= 0 {
		return nil, domain.ErrCommentInvalid
	}
	if in.ContentJSON != nil && len(*in.ContentJSON) == 0 {
		return nil, domain.ErrCommentInvalid
	}
	return s.repo.UpdateComment(ctx, tenantID, documentID, libraryID, userID, in)
}

func (s *Service) DeleteDocumentComment(ctx context.Context, tenantID, userID, documentID string, libraryID int) error {
	if libraryID <= 0 {
		return domain.ErrCommentInvalid
	}
	return s.repo.DeleteComment(ctx, tenantID, documentID, libraryID)
}

type PresignAutosaveCmd struct {
	TenantID, ActorUserID, DocumentID, SessionID, BaseRevisionID, ContentHash string
}

type PresignAutosaveResult struct {
	UploadURL       string
	PendingUploadID string
	ExpiresAt       time.Time
}

func (s *Service) PresignAutosave(ctx context.Context, cmd PresignAutosaveCmd) (*PresignAutosaveResult, error) {
	doc, err := s.repo.GetDocument(ctx, cmd.TenantID, cmd.DocumentID)
	if err != nil {
		return nil, err
	}
	if doc.Status != domain.DocStatusDraft {
		return nil, domain.ErrInvalidStateTransition
	}
	url, storageKey, err := s.presigner.PresignRevisionPUT(ctx, cmd.TenantID, cmd.DocumentID, cmd.ContentHash)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(15 * time.Minute)
	pendingID, err := s.repo.PresignReserve(ctx, cmd.SessionID, cmd.ActorUserID, cmd.DocumentID, cmd.BaseRevisionID, cmd.ContentHash, storageKey, expiresAt)
	if err != nil {
		return nil, err
	}
	return &PresignAutosaveResult{UploadURL: url, PendingUploadID: pendingID, ExpiresAt: expiresAt}, nil
}

type CommitAutosaveCmd struct {
	TenantID, ActorUserID, DocumentID, SessionID, PendingUploadID string
	FormDataSnapshot                                              json.RawMessage
}

func (s *Service) CommitAutosave(ctx context.Context, cmd CommitAutosaveCmd) (*CommitResult, error) {
	doc, err := s.repo.GetDocument(ctx, cmd.TenantID, cmd.DocumentID)
	if err != nil {
		return nil, err
	}
	if doc.Status != domain.DocStatusDraft {
		return nil, domain.ErrInvalidStateTransition
	}
	meta, err := s.repo.GetPendingForCommit(ctx, cmd.PendingUploadID)
	if err != nil {
		return nil, err
	}

	serverHash, err := s.presigner.HashObject(ctx, meta.StorageKey)
	if err != nil {
		if errors.Is(err, domain.ErrUploadMissing) {
			return nil, domain.ErrUploadMissing
		}
		return nil, fmt.Errorf("hash s3 object: %w", err)
	}
	if serverHash != meta.ExpectedContentHash {
		_ = s.presigner.DeleteObject(ctx, meta.StorageKey)
		return nil, domain.ErrContentHashMismatch
	}

	res, err := s.repo.CommitUpload(ctx, cmd.SessionID, cmd.ActorUserID, cmd.DocumentID, cmd.PendingUploadID, serverHash, cmd.FormDataSnapshot)
	if err != nil {
		return nil, err
	}
	if !res.AlreadyConsumed {
		s.audit.Write(ctx, cmd.TenantID, cmd.ActorUserID, "document.autosaved", cmd.DocumentID, map[string]any{"revision_id": res.RevisionID, "revision_num": res.RevisionNum})
	}
	return res, nil
}

func (s *Service) AcquireSession(ctx context.Context, tenantID, docID, userID string) (*domain.Session, bool, error) {
	sess, err := s.repo.AcquireSession(ctx, tenantID, docID, userID)
	if errors.Is(err, domain.ErrSessionTaken) {
		return sess, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	s.audit.Write(ctx, tenantID, userID, "session.acquired", docID, map[string]any{"session_id": sess.ID})
	return sess, false, nil
}

func (s *Service) HeartbeatSession(ctx context.Context, sessionID, userID string) error {
	return s.repo.HeartbeatSession(ctx, sessionID, userID)
}

func (s *Service) ReleaseSession(ctx context.Context, tenantID, sessionID, userID, docID string) error {
	if err := s.repo.ReleaseSession(ctx, sessionID, userID); err != nil {
		return err
	}
	s.audit.Write(ctx, tenantID, userID, "session.released", docID, map[string]any{"session_id": sessionID})
	return nil
}

func (s *Service) ForceReleaseSession(ctx context.Context, tenantID, adminID, sessionID, docID string) error {
	if err := s.repo.ForceReleaseSession(ctx, sessionID); err != nil {
		return err
	}
	s.audit.Write(ctx, tenantID, adminID, "session.force_released", docID, map[string]any{"session_id": sessionID})
	return nil
}

func (s *Service) CreateCheckpoint(ctx context.Context, tenantID, docID, actorID, label string) (*domain.Checkpoint, error) {
	cp, err := s.repo.CreateCheckpoint(ctx, docID, actorID, label)
	if err != nil {
		return nil, err
	}
	s.audit.Write(ctx, tenantID, actorID, "document.checkpoint_created", docID, map[string]any{"version_num": cp.VersionNum, "label": label})
	return cp, nil
}

func (s *Service) ListCheckpoints(ctx context.Context, tenantID, docID string) ([]domain.Checkpoint, error) {
	return s.repo.ListCheckpoints(ctx, docID)
}

func (s *Service) RestoreCheckpoint(ctx context.Context, tenantID, docID, actorID string, versionNum int) (*RestoreResult, error) {
	res, err := s.repo.RestoreCheckpoint(ctx, docID, actorID, versionNum)
	if err != nil {
		return nil, err
	}
	s.audit.Write(ctx, tenantID, actorID, "document.checkpoint_restored", docID, map[string]any{
		"version_num":        versionNum,
		"new_revision_id":    res.NewRevisionID,
		"new_revision_num":   res.NewRevisionNum,
		"source_revision_id": res.CheckpointRevID,
		"idempotent":         res.Idempotent,
	})
	return res, nil
}

func (s *Service) Finalize(ctx context.Context, tenantID, docID, actorID string) error {
	if err := s.repo.UpdateDocumentStatus(ctx, tenantID, docID, domain.DocStatusDraft, domain.DocStatusFinalized, true); err != nil {
		return err
	}
	s.audit.Write(ctx, tenantID, actorID, "document.finalized", docID, nil)
	return nil
}

func (s *Service) Archive(ctx context.Context, tenantID, docID, actorID string, fromFinalized bool) error {
	cur := domain.DocStatusDraft
	if fromFinalized {
		cur = domain.DocStatusFinalized
	}
	if err := s.repo.UpdateDocumentStatus(ctx, tenantID, docID, cur, domain.DocStatusArchived, true); err != nil {
		return err
	}
	s.audit.Write(ctx, tenantID, actorID, "document.archived", docID, nil)
	return nil
}

func (s *Service) SignedRevisionURL(ctx context.Context, tenantID, docID, revID string) (string, error) {
	rev, err := s.repo.GetRevision(ctx, docID, revID)
	if err != nil {
		return "", err
	}
	return s.presigner.PresignObjectGET(ctx, rev.StorageKey)
}
