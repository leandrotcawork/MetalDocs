package domain

import "errors"

var (
	ErrInvalidCommand        = errors.New("invalid command")
	ErrDocumentNotFound      = errors.New("document not found")
	ErrDocumentAlreadyExists = errors.New("document already exists")
	ErrInvalidDocumentType   = errors.New("invalid document type")
)
