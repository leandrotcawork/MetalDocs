package domain

import "errors"

var (
	ErrDocumentSchemaInvalid        = errors.New("document schema invalid")
	ErrDocumentSchemaInvalidSection = errors.New("document schema invalid section")
	ErrDocumentSchemaInvalidField   = errors.New("document schema invalid field")
)
