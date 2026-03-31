package unit

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
	postgres "metaldocs/internal/modules/documents/infrastructure/postgres"
)

var documentsRepoDriverSeq atomic.Int64

type documentsRepoState struct {
	mu        sync.Mutex
	documents map[string]domain.Document
	versions  map[string]map[int]storedVersion
}

type storedVersion struct {
	documentID       string
	number           int
	content          string
	contentHash      string
	changeSummary    string
	contentSource    string
	nativeContent    string
	bodyBlocks       string
	docxStorageKey   any
	pdfStorageKey    any
	textContent      any
	fileSizeBytes    any
	originalFilename any
	pageCount        any
	createdAt        time.Time
}

func (s *documentsRepoState) seedDocument(doc domain.Document) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.documents == nil {
		s.documents = map[string]domain.Document{}
	}
	if s.versions == nil {
		s.versions = map[string]map[int]storedVersion{}
	}
	s.documents[doc.ID] = doc
}

func (s *documentsRepoState) saveVersion(version storedVersion) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.versions == nil {
		s.versions = map[string]map[int]storedVersion{}
	}
	if _, ok := s.versions[version.documentID]; !ok {
		s.versions[version.documentID] = map[int]storedVersion{}
	}
	s.versions[version.documentID][version.number] = version
}

func (s *documentsRepoState) updateVersionBodyBlocks(documentID string, versionNumber int, bodyBlocks string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	versions, ok := s.versions[documentID]
	if !ok {
		return 0
	}
	version, ok := versions[versionNumber]
	if !ok {
		return 0
	}
	version.bodyBlocks = bodyBlocks
	versions[versionNumber] = version
	return 1
}

func (s *documentsRepoState) getDocument(documentID string) (domain.Document, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.documents[documentID]
	return doc, ok
}

func (s *documentsRepoState) getVersion(documentID string, versionNumber int) (storedVersion, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	versions, ok := s.versions[documentID]
	if !ok {
		return storedVersion{}, false
	}
	version, ok := versions[versionNumber]
	return version, ok
}

func newDocumentsRepoTestHarness(t *testing.T) (*postgres.Repository, *documentsRepoState) {
	t.Helper()

	state := &documentsRepoState{}
	driverName := fmt.Sprintf("documents-postgres-test-%d", documentsRepoDriverSeq.Add(1))
	sql.Register(driverName, &documentsRepoDriver{state: state})

	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return postgres.NewRepository(db), state
}

type documentsRepoDriver struct {
	state *documentsRepoState
}

func (d *documentsRepoDriver) Open(string) (driver.Conn, error) {
	return &documentsRepoConn{state: d.state}, nil
}

type documentsRepoConn struct {
	state *documentsRepoState
}

func (c *documentsRepoConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("prepare not supported")
}

func (c *documentsRepoConn) Close() error {
	return nil
}

func (c *documentsRepoConn) Begin() (driver.Tx, error) {
	return nil, fmt.Errorf("transactions not supported")
}

func (c *documentsRepoConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	switch {
	case strings.Contains(query, "INSERT INTO metaldocs.document_versions"):
		if len(args) < 15 {
			return nil, fmt.Errorf("unexpected args for version insert: %d", len(args))
		}
		version := storedVersion{
			documentID:       stringArg(args[0]),
			number:           int(intArg(args[1])),
			content:          stringArg(args[2]),
			contentHash:      stringArg(args[3]),
			changeSummary:    stringArg(args[4]),
			contentSource:    stringArg(args[5]),
			nativeContent:    stringArg(args[6]),
			bodyBlocks:       stringArg(args[7]),
			docxStorageKey:   nullableStringArg(args[8]),
			pdfStorageKey:    nullableStringArg(args[9]),
			textContent:      nullableStringArg(args[10]),
			fileSizeBytes:    nullableInt64Arg(args[11]),
			originalFilename: nullableStringArg(args[12]),
			pageCount:        nullableIntArg(args[13]),
			createdAt:        timeArg(args[14]),
		}
		c.state.saveVersion(version)
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "UPDATE metaldocs.document_versions") && strings.Contains(query, "SET body_blocks = COALESCE($3::jsonb, '[]'::jsonb)"):
		affected := c.state.updateVersionBodyBlocks(stringArg(args[0]), int(intArg(args[1])), stringArg(args[2]))
		return driver.RowsAffected(affected), nil
	default:
		return nil, fmt.Errorf("unexpected exec query: %s", query)
	}
}

func (c *documentsRepoConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch {
	case strings.Contains(query, "FROM metaldocs.documents"):
		docID := stringArg(args[0])
		doc, ok := c.state.getDocument(docID)
		if !ok {
			return newDocumentsRepoRows([]string{
				"id", "title", "document_type_code", "document_profile_code", "document_family_code", "document_sequence",
				"document_code", "process_area_code", "subject_code", "profile_schema_version", "owner_id", "business_unit",
				"department", "classification", "status", "tags", "effective_at", "expiry_at", "metadata_json", "created_at", "updated_at",
			}, nil), nil
		}
		return newDocumentsRepoRows([]string{
			"id", "title", "document_type_code", "document_profile_code", "document_family_code", "document_sequence",
			"document_code", "process_area_code", "subject_code", "profile_schema_version", "owner_id", "business_unit",
			"department", "classification", "status", "tags", "effective_at", "expiry_at", "metadata_json", "created_at", "updated_at",
		}, [][]driver.Value{documentRow(doc)}), nil
	case strings.Contains(query, "FROM metaldocs.document_versions"):
		docID := stringArg(args[0])
		if strings.Contains(query, "AND version_number = $2") {
			version, ok := c.state.getVersion(docID, int(intArg(args[1])))
			if !ok {
				return newDocumentsRepoRows(versionColumns(), nil), nil
			}
			return newDocumentsRepoRows(versionColumns(), [][]driver.Value{versionRow(version)}), nil
		}
		c.state.mu.Lock()
		versionsByDoc := c.state.versions[docID]
		rows := make([][]driver.Value, 0, len(versionsByDoc))
		for _, version := range versionsByDoc {
			rows = append(rows, versionRow(version))
		}
		c.state.mu.Unlock()
		return newDocumentsRepoRows(versionColumns(), rows), nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

func (c *documentsRepoConn) CheckNamedValue(*driver.NamedValue) error {
	return nil
}

var _ driver.ExecerContext = (*documentsRepoConn)(nil)
var _ driver.QueryerContext = (*documentsRepoConn)(nil)
var _ driver.NamedValueChecker = (*documentsRepoConn)(nil)

type documentsRepoRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func newDocumentsRepoRows(columns []string, rows [][]driver.Value) *documentsRepoRows {
	return &documentsRepoRows{columns: columns, rows: rows}
}

func (r *documentsRepoRows) Columns() []string {
	return r.columns
}

func (r *documentsRepoRows) Close() error {
	return nil
}

func (r *documentsRepoRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func versionColumns() []string {
	return []string{
		"document_id", "version_number", "content", "content_hash", "change_summary",
		"content_source", "native_content", "body_blocks", "docx_storage_key", "pdf_storage_key",
		"text_content", "file_size_bytes", "original_filename", "page_count", "created_at",
	}
}

func documentRow(doc domain.Document) []driver.Value {
	return []driver.Value{
		doc.ID,
		doc.Title,
		doc.DocumentType,
		doc.DocumentProfile,
		doc.DocumentFamily,
		int64(doc.DocumentSequence),
		doc.DocumentCode,
		nullableStringValue(doc.ProcessArea),
		nullableStringValue(doc.Subject),
		int64(doc.ProfileSchemaVersion),
		doc.OwnerID,
		doc.BusinessUnit,
		doc.Department,
		doc.Classification,
		doc.Status,
		jsonBytes(doc.Tags, "[]"),
		nullableTimeValue(doc.EffectiveAt),
		nullableTimeValue(doc.ExpiryAt),
		jsonBytes(doc.MetadataJSON, "{}"),
		doc.CreatedAt,
		doc.UpdatedAt,
	}
}

func versionRow(version storedVersion) []driver.Value {
	return []driver.Value{
		version.documentID,
		int64(version.number),
		version.content,
		version.contentHash,
		version.changeSummary,
		version.contentSource,
		jsonBytesString(version.nativeContent, "{}"),
		jsonBytesString(version.bodyBlocks, "[]"),
		version.docxStorageKey,
		version.pdfStorageKey,
		version.textContent,
		version.fileSizeBytes,
		version.originalFilename,
		version.pageCount,
		version.createdAt,
	}
}

func jsonBytes(value any, fallback string) []byte {
	if value == nil {
		return []byte(fallback)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte(fallback)
	}
	return raw
}

func jsonBytesString(value string, fallback string) []byte {
	if strings.TrimSpace(value) == "" {
		return []byte(fallback)
	}
	return []byte(value)
}

func nullableStringValue(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableStringArg(arg driver.NamedValue) any {
	switch v := arg.Value.(type) {
	case nil:
		return nil
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return v
	default:
		return v
	}
}

func nullableInt64Arg(arg driver.NamedValue) any {
	switch v := arg.Value.(type) {
	case nil:
		return nil
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return v
	}
}

func nullableIntArg(arg driver.NamedValue) any {
	switch v := arg.Value.(type) {
	case nil:
		return nil
	case int:
		return v
	case int64:
		return int(v)
	default:
		return v
	}
}

func intArg(arg driver.NamedValue) int64 {
	switch v := arg.Value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	default:
		panic(fmt.Sprintf("unexpected int arg type %T", arg.Value))
	}
}

func stringArg(arg driver.NamedValue) string {
	if arg.Value == nil {
		return ""
	}
	value, ok := arg.Value.(string)
	if !ok {
		panic(fmt.Sprintf("unexpected string arg type %T", arg.Value))
	}
	return value
}

func timeArg(arg driver.NamedValue) time.Time {
	switch v := arg.Value.(type) {
	case time.Time:
		return v
	default:
		panic(fmt.Sprintf("unexpected time arg type %T", arg.Value))
	}
}

func nullableTimeValue(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

func TestPostgresRepositorySaveVersionRoundTripsBodyBlocks(t *testing.T) {
	repo, state := newDocumentsRepoTestHarness(t)
	state.seedDocument(seedDocument("doc-1"))

	version := domain.Version{
		DocumentID:    "doc-1",
		Number:        1,
		Content:       "v1",
		ContentHash:   "hash-1",
		ChangeSummary: "initial",
		BodyBlocks: []domain.EtapaBody{
			{Blocks: []json.RawMessage{json.RawMessage(`{"type":"paragraph","text":"primeiro"}`)}},
		},
		CreatedAt: time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC),
	}

	if err := repo.SaveVersion(context.Background(), version); err != nil {
		t.Fatalf("save version: %v", err)
	}

	got, err := repo.GetVersion(context.Background(), "doc-1", 1)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}

	if len(got.BodyBlocks) != 1 {
		t.Fatalf("expected 1 body block, got %d", len(got.BodyBlocks))
	}
	if len(got.BodyBlocks[0].Blocks) != 1 {
		t.Fatalf("expected 1 rich block, got %d", len(got.BodyBlocks[0].Blocks))
	}
	if string(got.BodyBlocks[0].Blocks[0]) != `{"type":"paragraph","text":"primeiro"}` {
		t.Fatalf("unexpected body block payload: %s", string(got.BodyBlocks[0].Blocks[0]))
	}
}

func TestPostgresRepositoryUpdateVersionBodyBlocks(t *testing.T) {
	repo, state := newDocumentsRepoTestHarness(t)
	state.seedDocument(seedDocument("doc-2"))

	initial := domain.Version{
		DocumentID:    "doc-2",
		Number:        1,
		Content:       "v1",
		ContentHash:   "hash-1",
		ChangeSummary: "initial",
		BodyBlocks: []domain.EtapaBody{
			{Blocks: []json.RawMessage{json.RawMessage(`{"type":"paragraph","text":"antigo"}`)}},
		},
		CreatedAt: time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC),
	}
	if err := repo.SaveVersion(context.Background(), initial); err != nil {
		t.Fatalf("save version: %v", err)
	}

	updatedBlocks := []domain.EtapaBody{
		{Blocks: []json.RawMessage{json.RawMessage(`{"type":"paragraph","text":"novo"}`)}},
	}
	if err := repo.UpdateVersionBodyBlocks(context.Background(), "doc-2", 1, updatedBlocks); err != nil {
		t.Fatalf("update version body blocks: %v", err)
	}

	got, err := repo.GetVersion(context.Background(), "doc-2", 1)
	if err != nil {
		t.Fatalf("get version after update: %v", err)
	}
	if len(got.BodyBlocks) != 1 || len(got.BodyBlocks[0].Blocks) != 1 {
		t.Fatalf("unexpected body blocks after update: %#v", got.BodyBlocks)
	}
	if string(got.BodyBlocks[0].Blocks[0]) != `{"type":"paragraph","text":"novo"}` {
		t.Fatalf("unexpected updated body block payload: %s", string(got.BodyBlocks[0].Blocks[0]))
	}

	if err := repo.UpdateVersionBodyBlocks(context.Background(), "doc-2", 2, updatedBlocks); err != domain.ErrVersionNotFound {
		t.Fatalf("expected version not found, got %v", err)
	}
}

func TestPostgresRepositorySaveVersionFailsOnBodyBlocksSerializationError(t *testing.T) {
	repo, state := newDocumentsRepoTestHarness(t)
	state.seedDocument(seedDocument("doc-3"))

	version := domain.Version{
		DocumentID:    "doc-3",
		Number:        1,
		Content:       "v1",
		ContentHash:   "hash-1",
		ChangeSummary: "initial",
		BodyBlocks: []domain.EtapaBody{
			{Blocks: []json.RawMessage{json.RawMessage("{")}},
		},
		CreatedAt: time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC),
	}

	if err := repo.SaveVersion(context.Background(), version); err == nil {
		t.Fatal("expected serialization error, got nil")
	}
	if _, ok := state.getVersion("doc-3", 1); ok {
		t.Fatal("expected no version write on serialization failure")
	}
}

func seedDocument(id string) domain.Document {
	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	return domain.Document{
		ID:                   id,
		Title:                "Documento",
		DocumentType:         "po",
		DocumentProfile:      "po",
		DocumentFamily:       "procedure",
		DocumentSequence:     1,
		DocumentCode:         "PO-001",
		ProfileSchemaVersion: 1,
		OwnerID:              "owner-1",
		BusinessUnit:         "quality",
		Department:           "qa",
		Classification:       domain.ClassificationInternal,
		Status:               domain.StatusDraft,
		Tags:                 []string{},
		MetadataJSON:         map[string]any{},
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}
