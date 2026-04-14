package domain

import "errors"

var (
	ErrInvalidCommand                     = errors.New("invalid command")
	ErrDocumentNotFound                   = errors.New("document not found")
	ErrForbidden                          = errors.New("forbidden")
	ErrDocumentAlreadyExists              = errors.New("document already exists")
	ErrInvalidDocumentType                = errors.New("invalid document type")
	ErrInvalidDocumentProfileAlias        = errors.New("invalid document profile alias")
	ErrInvalidAccessPolicy                = errors.New("invalid access policy")
	ErrInvalidMetadata                    = errors.New("invalid metadata")
	ErrInvalidNativeContent               = errors.New("invalid native content")
	ErrDraftConflict                      = errors.New("draft conflict")
	ErrVersioningNotAllowed               = errors.New("versioning not allowed for current status")
	ErrVersionNotFound                    = errors.New("version not found")
	ErrInvalidAttachment                  = errors.New("invalid attachment")
	ErrAttachmentNotFound                 = errors.New("attachment not found")
	ErrAttachmentStoreUnavailable         = errors.New("attachment store unavailable")
	ErrRenderUnavailable                  = errors.New("render service unavailable")
	ErrEditLockActive                     = errors.New("edit lock already active")
	ErrEditLockNotFound                   = errors.New("edit lock not found")
	ErrDocumentTemplateNotFound           = errors.New("document template not found")
	ErrDocumentTemplateAssignmentNotFound = errors.New("document template assignment not found")
)
