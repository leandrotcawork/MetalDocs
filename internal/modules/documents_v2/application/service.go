package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents_v2/domain"
	repopkg "metaldocs/internal/modules/documents_v2/repository"
)

type PendingCommitMeta = repopkg.PendingCommitMeta
type CommitResult = repopkg.CommitResult
type RestoreResult = repopkg.RestoreResult

var (
	ErrInvalidStateTransition = domain.ErrInvalidStateTransition
	ErrSessionInactive        = domain.ErrSessionInactive
	ErrSessionNotHolder       = domain.ErrSessionNotHolder
	ErrStaleBase              = domain.ErrStaleBase
	ErrMisbound               = domain.ErrMisbound
	ErrExpiredUpload          = domain.ErrExpiredUpload
	ErrContentHashMismatch    = domain.ErrContentHashMismatch
	ErrPendingNotFound        = domain.ErrPendingNotFound
	ErrAlreadyConsumed        = domain.ErrAlreadyConsumed
	ErrSessionTaken           = domain.ErrSessionTaken
	ErrForbidden              = domain.ErrForbidden
	ErrUploadMissing          = domain.ErrUploadMissing
	ErrCheckpointNotFound     = domain.ErrCheckpointNotFound
	ErrDocumentNotOwner       = domain.ErrDocumentNotOwner
)

type Repository interface {
	CreateDocument(ctx context.Context, d *domain.Document, initialContentHash string) (docID, revID, sessionID string, err error)
	SetRevisionStorageKey(ctx context.Context, revID, storageKey string) error
	GetDocument(ctx context.Context, tenantID, id string) (*domain.Document, error)
	ListDocuments(ctx context.Context, tenantID string) ([]domain.Document, error)
	ListDocumentsForUser(ctx context.Context, tenantID, userID string) ([]domain.Document, error)
	UpdateDocumentStatus(ctx context.Context, tenantID, id string, cur, next domain.DocumentStatus, stampTime bool) error
	AcquireSession(ctx context.Context, tenantID, docID, userID string) (*domain.Session, error)
	HeartbeatSession(ctx context.Context, sessionID, userID string) error
	ReleaseSession(ctx context.Context, sessionID, userID string) error
	ForceReleaseSession(ctx context.Context, sessionID string) error
	PresignReserve(ctx context.Context, sessionID, userID, docID, baseRevisionID, contentHash, storageKey string, expiresAt time.Time) (pendingID string, err error)
	GetPendingForCommit(ctx context.Context, pendingID string) (*PendingCommitMeta, error)
	CommitUpload(ctx context.Context, sessionID, userID, docID, pendingID, serverComputedHash string, formDataSnapshot []byte) (*CommitResult, error)
	CreateCheckpoint(ctx context.Context, docID, actorUserID, label string) (*domain.Checkpoint, error)
	ListCheckpoints(ctx context.Context, docID string) ([]domain.Checkpoint, error)
	RestoreCheckpoint(ctx context.Context, docID, actorUserID string, versionNum int) (*RestoreResult, error)
	IsDocumentOwner(ctx context.Context, tenantID, docID, userID string) (bool, error)
}

type DocgenRenderer interface {
	Render(ctx context.Context, templateDocx []byte, formDataJSON []byte) (tempObjectKey string, contentHash string, err error)
}

type Presigner interface {
	PresignAutosavePUT(ctx context.Context, tenantID, docID, pendingID, storageKey string, expiresAt time.Time) (url string, err error)
	HashObject(ctx context.Context, storageKey string) (hash string, err error)
	AdoptTempObject(ctx context.Context, tempObjectKey, finalStorageKey string) error
	DeleteObject(ctx context.Context, storageKey string) error
	PresignObjectGET(ctx context.Context, storageKey string) (url string, err error)
}

type TemplateReader interface {
	ReadTemplateDocx(ctx context.Context, tenantID, templateVersionID string) ([]byte, error)
}

type FormValidator interface {
	Validate(ctx context.Context, tenantID, templateVersionID string, formDataJSON []byte) error
}

type Audit interface {
	Record(ctx context.Context, action string, fields map[string]string) error
}

type Service struct {
	repo           Repository
	docgen         DocgenRenderer
	presigner      Presigner
	templateReader TemplateReader
	validator      FormValidator
	audit          Audit
}

func New(repo Repository, docgen DocgenRenderer, presigner Presigner, templateReader TemplateReader, validator FormValidator, audit Audit) *Service {
	return &Service{
		repo:           repo,
		docgen:         docgen,
		presigner:      presigner,
		templateReader: templateReader,
		validator:      validator,
		audit:          audit,
	}
}

func (s *Service) CreateDocument(ctx context.Context, tenantID, templateVersionID, name string, formDataJSON []byte, actorUserID string) (*domain.Document, error) {
	if err := s.validator.Validate(ctx, tenantID, templateVersionID, formDataJSON); err != nil {
		return nil, err
	}

	tplDocx, err := s.templateReader.ReadTemplateDocx(ctx, tenantID, templateVersionID)
	if err != nil {
		return nil, err
	}
	tempKey, contentHash, err := s.docgen.Render(ctx, tplDocx, formDataJSON)
	if err != nil {
		return nil, err
	}

	doc := &domain.Document{
		TenantID:          tenantID,
		TemplateVersionID: templateVersionID,
		Name:              name,
		Status:            domain.DocStatusDraft,
		FormDataJSON:      formDataJSON,
		CreatedBy:         actorUserID,
	}
	docID, revID, sessionID, err := s.repo.CreateDocument(ctx, doc, contentHash)
	if err != nil {
		return nil, err
	}

	finalStorageKey := revisionStorageKey(docID, revID)
	if err := s.presigner.AdoptTempObject(ctx, tempKey, finalStorageKey); err != nil {
		return nil, err
	}
	if err := s.repo.SetRevisionStorageKey(ctx, revID, finalStorageKey); err != nil {
		return nil, err
	}

	doc.ID = docID
	doc.CurrentRevisionID = revID
	doc.ActiveSessionID = sessionID

	_ = s.audit.Record(ctx, "documents_v2.document.create", map[string]string{
		"tenant_id":   tenantID,
		"document_id": docID,
		"user_id":     actorUserID,
	})

	return doc, nil
}

func (s *Service) PresignAutosave(ctx context.Context, tenantID, sessionID, userID, docID, baseRevisionID, contentHash string) (pendingID, uploadURL string, err error) {
	pendingID = newUUID()
	storageKey := pendingUploadStorageKey(docID, pendingID)
	expiresAt := time.Now().UTC().Add(15 * time.Minute)

	pendingID, err = s.repo.PresignReserve(ctx, sessionID, userID, docID, baseRevisionID, contentHash, storageKey, expiresAt)
	if err != nil {
		return "", "", err
	}
	uploadURL, err = s.presigner.PresignAutosavePUT(ctx, tenantID, docID, pendingID, storageKey, expiresAt)
	if err != nil {
		return "", "", err
	}
	return pendingID, uploadURL, nil
}

func (s *Service) CommitAutosave(ctx context.Context, sessionID, userID, docID, pendingID string, formDataSnapshot []byte) (*CommitResult, error) {
	meta, err := s.repo.GetPendingForCommit(ctx, pendingID)
	if err != nil {
		return nil, err
	}

	serverHash, err := s.presigner.HashObject(ctx, meta.StorageKey)
	if err != nil {
		return nil, err
	}
	if serverHash != meta.ExpectedContentHash {
		_ = s.presigner.DeleteObject(ctx, meta.StorageKey)
		return nil, domain.ErrContentHashMismatch
	}

	return s.repo.CommitUpload(ctx, sessionID, userID, docID, pendingID, serverHash, formDataSnapshot)
}

func (s *Service) AcquireSession(ctx context.Context, tenantID, docID, userID string) (*domain.Session, bool, error) {
	sess, err := s.repo.AcquireSession(ctx, tenantID, docID, userID)
	if err == domain.ErrSessionTaken {
		return sess, true, nil
	}
	if err != nil {
		return nil, false, err
	}

	_ = s.audit.Record(ctx, "documents_v2.session.acquire", map[string]string{
		"tenant_id":   tenantID,
		"document_id": docID,
		"session_id":  sess.ID,
		"user_id":     userID,
	})

	return sess, false, nil
}

func (s *Service) HeartbeatSession(ctx context.Context, sessionID, userID string) error {
	return s.repo.HeartbeatSession(ctx, sessionID, userID)
}

func (s *Service) ReleaseSession(ctx context.Context, sessionID, userID string) error {
	return s.repo.ReleaseSession(ctx, sessionID, userID)
}

func (s *Service) ForceReleaseSession(ctx context.Context, sessionID string) error {
	return s.repo.ForceReleaseSession(ctx, sessionID)
}

func (s *Service) GetDocument(ctx context.Context, tenantID, docID string) (*domain.Document, error) {
	return s.repo.GetDocument(ctx, tenantID, docID)
}

func (s *Service) ListDocuments(ctx context.Context, tenantID string) ([]domain.Document, error) {
	return s.repo.ListDocuments(ctx, tenantID)
}

func (s *Service) ListDocumentsForUser(ctx context.Context, tenantID, userID string) ([]domain.Document, error) {
	return s.repo.ListDocumentsForUser(ctx, tenantID, userID)
}

func (s *Service) IsDocumentOwner(ctx context.Context, tenantID, docID, userID string) (bool, error) {
	return s.repo.IsDocumentOwner(ctx, tenantID, docID, userID)
}

func (s *Service) CreateCheckpoint(ctx context.Context, docID, actorUserID, label string) (*domain.Checkpoint, error) {
	return s.repo.CreateCheckpoint(ctx, docID, actorUserID, label)
}

func (s *Service) ListCheckpoints(ctx context.Context, docID string) ([]domain.Checkpoint, error) {
	return s.repo.ListCheckpoints(ctx, docID)
}

func (s *Service) RestoreCheckpoint(ctx context.Context, docID, actorUserID string, versionNum int) (*RestoreResult, error) {
	return s.repo.RestoreCheckpoint(ctx, docID, actorUserID, versionNum)
}

func (s *Service) Finalize(ctx context.Context, tenantID, docID string) error {
	doc, err := s.repo.GetDocument(ctx, tenantID, docID)
	if err != nil {
		return err
	}
	if !domain.CanTransitionDocument(doc.Status, domain.DocStatusFinalized) {
		return domain.ErrInvalidStateTransition
	}
	return s.repo.UpdateDocumentStatus(ctx, tenantID, docID, doc.Status, domain.DocStatusFinalized, true)
}

func (s *Service) Archive(ctx context.Context, tenantID, docID string) error {
	doc, err := s.repo.GetDocument(ctx, tenantID, docID)
	if err != nil {
		return err
	}
	if !domain.CanTransitionDocument(doc.Status, domain.DocStatusArchived) {
		return domain.ErrInvalidStateTransition
	}
	return s.repo.UpdateDocumentStatus(ctx, tenantID, docID, doc.Status, domain.DocStatusArchived, true)
}

func (s *Service) SignedRevisionURL(ctx context.Context, docID, revisionID string) (string, error) {
	return s.presigner.PresignObjectGET(ctx, revisionStorageKey(docID, revisionID))
}

const googleUUIDAvailable = true

func newUUID() string {
	if googleUUIDAvailable {
		return uuid.New().String()
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func revisionStorageKey(docID, revisionID string) string {
	return fmt.Sprintf("documents/%s/revisions/%s.docx", docID, revisionID)
}

func pendingUploadStorageKey(docID, pendingID string) string {
	return fmt.Sprintf("documents/%s/pending/%s.docx", docID, pendingID)
}
