package domain

import "errors"

var (
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrLockVersionMismatch    = errors.New("lock version mismatch")
	ErrDuplicateDraft         = errors.New("duplicate draft")
	ErrUnsupportedOOXML       = errors.New("unsupported OOXML construct")
)
