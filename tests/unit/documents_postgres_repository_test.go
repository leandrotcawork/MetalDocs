package unit

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
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
	types     map[string]domain.DocumentTypeDefinition
	schemas   map[string]string
}

type storedVersion struct {
	documentID       string
	number           int
	content          string
	contentHash      string
	changeSummary    string
	contentSource    string
	nativeContent    string
	values           string
	bodyBlocks       string
	docxStorageKey   any
	pdfStorageKey    any
	textContent      any
	fileSizeBytes    any
	originalFilename any
	pageCount        any
	templateKey      any
	templateVersion  any
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
	if s.types == nil {
		s.types = map[string]domain.DocumentTypeDefinition{}
	}
	if s.schemas == nil {
		s.schemas = map[string]string{}
	}
	s.documents[doc.ID] = doc
}

func (s *documentsRepoState) upsertTypeDefinition(item domain.DocumentTypeDefinition) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.types == nil {
		s.types = map[string]domain.DocumentTypeDefinition{}
	}
	s.types[strings.ToLower(strings.TrimSpace(item.Key))] = item
}

func (s *documentsRepoState) upsertTypeSchema(key string, schema string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.schemas == nil {
		s.schemas = map[string]string{}
	}
	s.schemas[strings.ToLower(strings.TrimSpace(key))] = schema
}

func (s *documentsRepoState) listTypeDefinitions() []domain.DocumentTypeDefinition {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]domain.DocumentTypeDefinition, 0, len(s.types))
	for _, item := range s.types {
		out = append(out, item)
	}
	return out
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

func (s *documentsRepoState) updateVersionValues(documentID string, versionNumber int, values string) int64 {
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
	version.values = values
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

func newPostgresRepositoryForTest(t *testing.T) *postgres.Repository {
	t.Helper()
	repo, _ := newDocumentsRepoTestHarness(t)
	return repo
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
	case strings.Contains(query, "INSERT INTO metaldocs.document_types"):
		if len(args) < 3 {
			return nil, fmt.Errorf("unexpected args for type upsert: %d", len(args))
		}
		c.state.upsertTypeDefinition(domain.DocumentTypeDefinition{
			Key:           stringArg(args[0]),
			Name:          stringArg(args[1]),
			ActiveVersion: int(intArg(args[2])),
		})
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "INSERT INTO metaldocs.document_type_schema_versions"):
		if len(args) < 3 {
			return nil, fmt.Errorf("unexpected args for type schema upsert: %d", len(args))
		}
		c.state.upsertTypeSchema(stringArg(args[0]), stringArg(args[2]))
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "INSERT INTO metaldocs.document_versions"):
		if len(args) < 18 {
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
			values:           stringArg(args[7]),
			bodyBlocks:       stringArg(args[8]),
			docxStorageKey:   nullableStringArg(args[9]),
			pdfStorageKey:    nullableStringArg(args[10]),
			textContent:      nullableStringArg(args[11]),
			fileSizeBytes:    nullableInt64Arg(args[12]),
			originalFilename: nullableStringArg(args[13]),
			pageCount:        nullableIntArg(args[14]),
			templateKey:      nullableStringArg(args[15]),
			templateVersion:  nullableIntArg(args[16]),
			createdAt:        timeArg(args[17]),
		}
		c.state.saveVersion(version)
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "UPDATE metaldocs.document_versions") && strings.Contains(query, "SET values_json = $3::jsonb"):
		affected := c.state.updateVersionValues(stringArg(args[0]), int(intArg(args[1])), stringArg(args[2]))
		return driver.RowsAffected(affected), nil
	case strings.Contains(query, "UPDATE metaldocs.document_versions") && strings.Contains(query, "SET body_blocks = COALESCE($3::jsonb, '[]'::jsonb)"):
		affected := c.state.updateVersionBodyBlocks(stringArg(args[0]), int(intArg(args[1])), stringArg(args[2]))
		return driver.RowsAffected(affected), nil
	default:
		return nil, fmt.Errorf("unexpected exec query: %s", query)
	}
}

func (c *documentsRepoConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch {
	case strings.Contains(query, "FROM metaldocs.document_types"):
		if strings.Contains(query, "WHERE t.type_key = $1") {
			key := stringArg(args[0])
			c.state.mu.Lock()
			item, ok := c.state.types[strings.ToLower(strings.TrimSpace(key))]
			schema := c.state.schemas[strings.ToLower(strings.TrimSpace(key))]
			c.state.mu.Unlock()
			if !ok {
				return newDocumentsRepoRows([]string{"type_key", "name", "active_version", "schema_json"}, nil), nil
			}
			if strings.TrimSpace(schema) == "" {
				schema = `{"sections":[]}`
			}
			return newDocumentsRepoRows([]string{"type_key", "name", "active_version", "schema_json"}, [][]driver.Value{[]driver.Value{item.Key, item.Name, int64(item.ActiveVersion), []byte(schema)}}), nil
		}
		items := c.state.listTypeDefinitions()
		rows := make([][]driver.Value, 0, len(items))
		for _, item := range items {
			schema := c.state.schemas[strings.ToLower(strings.TrimSpace(item.Key))]
			if strings.TrimSpace(schema) == "" {
				schema = `{"sections":[]}`
			}
			rows = append(rows, []driver.Value{item.Key, item.Name, int64(item.ActiveVersion), []byte(schema)})
		}
		return newDocumentsRepoRows([]string{"type_key", "name", "active_version", "schema_json"}, rows), nil
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
		"content_source", "native_content", "values_json", "body_blocks", "docx_storage_key", "pdf_storage_key",
		"text_content", "file_size_bytes", "original_filename", "page_count", "template_key", "template_version", "created_at",
		"renderer_pin",
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
		jsonBytesString(version.values, "{}"),
		jsonBytesString(version.bodyBlocks, "[]"),
		version.docxStorageKey,
		version.pdfStorageKey,
		version.textContent,
		version.fileSizeBytes,
		version.originalFilename,
		version.pageCount,
		version.templateKey,
		version.templateVersion,
		version.createdAt,
		nil, // renderer_pin (nullable)
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

func TestPostgresRepository_SaveDocumentTypeSchemaRuntime(t *testing.T) {
	repo, state := newDocumentsRepoTestHarness(t)
	ctx := context.Background()

	item := documentTypeDefinitionRuntime{
		Key:           "po",
		Name:          "Procedimento Operacional",
		ActiveVersion: 1,
		Schema: documentTypeSchemaRuntime{
			Sections: []sectionDefRuntime{
				{Key: "identificacao", Num: "1", Title: "Identificacao"},
			},
		},
	}

	method := reflect.ValueOf(repo).MethodByName("UpsertDocumentTypeDefinition")
	if !method.IsValid() {
		t.Fatalf("missing UpsertDocumentTypeDefinition")
	}
	if method.Type().NumIn() != 2 {
		t.Fatalf("unexpected UpsertDocumentTypeDefinition signature: %s", method.Type())
	}

	arg, err := prepareReflectionArg(method.Type().In(1), reflect.ValueOf(item))
	if err != nil {
		t.Fatalf("prepare argument: %v", err)
	}

	results := method.Call([]reflect.Value{reflect.ValueOf(ctx), arg})
	if len(results) != 1 {
		t.Fatalf("unexpected return count: %d", len(results))
	}
	if errValue := results[0]; !errValue.IsNil() {
		t.Fatalf("upsert type: %v", errValue.Interface())
	}

	definitionsMethod := reflect.ValueOf(repo).MethodByName("ListDocumentTypeDefinitions")
	if !definitionsMethod.IsValid() {
		t.Fatalf("missing ListDocumentTypeDefinitions")
	}
	definitionResults := definitionsMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(definitionResults) != 2 {
		t.Fatalf("unexpected list return count: %d", len(definitionResults))
	}
	if !definitionResults[1].IsNil() {
		t.Fatalf("list document type definitions: %v", definitionResults[1].Interface())
	}

	state.seedDocument(seedDocument("doc-runtime"))
	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:    "doc-runtime",
		Number:        1,
		Content:       "{}",
		ContentHash:   "hash-runtime",
		ChangeSummary: "initial",
		Values:        map[string]any{"objetivo": "antigo"},
		CreatedAt:     time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("save runtime version: %v", err)
	}
	if err := repo.UpdateVersionValues(ctx, "doc-runtime", 1, map[string]any{"objetivo": "novo"}); err != nil {
		t.Fatalf("update version values: %v", err)
	}
	got, err := repo.GetVersion(ctx, "doc-runtime", 1)
	if err != nil {
		t.Fatalf("get updated version: %v", err)
	}
	if got.Values["objetivo"] != "novo" {
		t.Fatalf("expected updated runtime value, got %#v", got.Values)
	}
}

type documentTypeDefinitionRuntime struct {
	Key           string
	Name          string
	ActiveVersion int
	Schema        documentTypeSchemaRuntime
}

type documentTypeSchemaRuntime struct {
	Sections []sectionDefRuntime
}

type sectionDefRuntime struct {
	Key   string
	Num   string
	Title string
}

func prepareReflectionArg(target reflect.Type, src reflect.Value) (reflect.Value, error) {
	if !src.IsValid() {
		return reflect.Zero(target), nil
	}
	if src.Type().AssignableTo(target) {
		return src, nil
	}
	if src.Type().ConvertibleTo(target) {
		return src.Convert(target), nil
	}
	if target.Kind() == reflect.Pointer {
		value, err := prepareReflectionArg(target.Elem(), src)
		if err != nil {
			return reflect.Value{}, err
		}
		ptr := reflect.New(target.Elem())
		ptr.Elem().Set(value)
		return ptr, nil
	}
	if target.Kind() == reflect.Struct {
		if src.Kind() == reflect.Pointer {
			src = src.Elem()
		}
		if src.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("cannot map %s to %s", src.Type(), target)
		}
		out := reflect.New(target).Elem()
		for i := 0; i < target.NumField(); i++ {
			field := target.Field(i)
			if !field.IsExported() {
				continue
			}
			sourceField := src.FieldByName(field.Name)
			if !sourceField.IsValid() {
				continue
			}
			value, err := prepareReflectionArg(field.Type, sourceField)
			if err != nil {
				return reflect.Value{}, err
			}
			if out.Field(i).CanSet() {
				out.Field(i).Set(value)
			}
		}
		return out, nil
	}
	if target.Kind() == reflect.Slice {
		if src.Kind() == reflect.Pointer {
			src = src.Elem()
		}
		if src.Kind() != reflect.Slice {
			return reflect.Value{}, fmt.Errorf("cannot map %s to %s", src.Type(), target)
		}
		out := reflect.MakeSlice(target, src.Len(), src.Len())
		for i := 0; i < src.Len(); i++ {
			value, err := prepareReflectionArg(target.Elem(), src.Index(i))
			if err != nil {
				return reflect.Value{}, err
			}
			out.Index(i).Set(value)
		}
		return out, nil
	}
	return reflect.Value{}, fmt.Errorf("cannot map %s to %s", src.Type(), target)
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
