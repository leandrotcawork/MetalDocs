package domain

type schemaRuntimeError struct {
	code string
}

func (e schemaRuntimeError) Error() string {
	return e.code
}

func (e schemaRuntimeError) Code() string {
	return e.code
}

func (e schemaRuntimeError) Is(target error) bool {
	typed, ok := target.(schemaRuntimeError)
	return ok && e.code == typed.code
}

var (
	ErrDocumentSchemaInvalid        = schemaRuntimeError{code: "DOCUMENT_SCHEMA_INVALID"}
	ErrDocumentSchemaInvalidSection = schemaRuntimeError{code: "DOCUMENT_SCHEMA_INVALID_SECTION"}
	ErrDocumentSchemaInvalidField   = schemaRuntimeError{code: "DOCUMENT_SCHEMA_INVALID_FIELD"}
)
