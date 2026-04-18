package domain

import "errors"

// Export is an immutable row in document_exports representing one cached PDF.
type Export struct {
	ID            string
	DocumentID    string
	RevisionID    string
	CompositeHash []byte // 32 bytes
	StorageKey    string
	SizeBytes     int64
	PaperSize     string
	Landscape     bool
	DocgenV2Ver   string
}

// ExportResult carries the export row and whether it was retrieved from cache.
type ExportResult struct {
	Export *Export
	Cached bool
}

var (
	ErrExportGotenbergFailed = errors.New("gotenberg_conversion_failed")
	ErrExportDocxMissing     = errors.New("docx_missing")
)
