package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	dsn             = "host=127.0.0.1 port=5433 dbname=metaldocs user=metaldocs_app password='Lepa12<>!' sslmode=disable"
	minioEndpoint   = "127.0.0.1:9000"
	minioAccessKey  = "minioadmin"
	minioSecretKey  = "minioadmin"
	minioBucket     = "metaldocs-attachments"
	tenantID        = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	adminUsername   = "e2e.admin"
	formDataJSON    = `{}`
	emptyJSON       = `{}`
	placeholderJSON = `[]`
)

type dbSchema struct {
	templatesSchema   string
	versionsSchema    string
	controlledSchema  string
	documentsSchema   string
	sessionsSchema    string
	revisionsSchema   string
	iamSchema         string
	templateHasStatus bool
	docHasBridge      bool
	docHasSnapshots   bool
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	docx, err := buildDOCX()
	if err != nil {
		log.Fatalf("build docx: %v", err)
	}
	contentHash := sha256Hex(docx)
	contentHashBytes := sha256.Sum256(docx)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	schema, err := discoverSchema(ctx, db)
	if err != nil {
		log.Fatalf("discover schema: %v", err)
	}

	userID, err := lookupAdminUser(ctx, db, schema.iamSchema)
	if err != nil {
		log.Fatalf("lookup %s: %v", adminUsername, err)
	}

	profileCode, err := lookupFirstCode(ctx, db, "metaldocs", "document_profiles")
	if err != nil {
		log.Fatalf("lookup document profile: %v", err)
	}
	areaCode, err := lookupFirstCode(ctx, db, "metaldocs", "document_process_areas")
	if err != nil {
		log.Fatalf("lookup process area: %v", err)
	}

	revisionID := uuid.NewString()
	storageKey := fmt.Sprintf("tenants/%s/revisions/%s.docx", tenantID, revisionID)
	if err := uploadDOCX(ctx, docx, storageKey); err != nil {
		log.Fatalf("upload docx: %v", err)
	}

	documentID, err := insertRows(ctx, db, schema, seedInput{
		UserID:           userID,
		ProfileCode:      profileCode,
		ProcessAreaCode:  areaCode,
		RevisionID:       revisionID,
		StorageKey:       storageKey,
		ContentHash:      contentHash,
		ContentHashBytes: contentHashBytes[:],
	})
	if err != nil {
		log.Fatalf("insert seed rows: %v", err)
	}

	fmt.Printf("document_id: %s\n", documentID)
}

type seedInput struct {
	UserID           string
	ProfileCode      string
	ProcessAreaCode  string
	RevisionID       string
	StorageKey       string
	ContentHash      string
	ContentHashBytes []byte
}

func discoverSchema(ctx context.Context, db *sql.DB) (dbSchema, error) {
	s := dbSchema{}
	var err error
	if s.templatesSchema, err = findTableSchema(ctx, db, "templates_v2_template", "public", "metaldocs"); err != nil {
		return s, err
	}
	if s.versionsSchema, err = findTableSchema(ctx, db, "templates_v2_template_version", "public", "metaldocs"); err != nil {
		return s, err
	}
	if s.controlledSchema, err = findTableSchema(ctx, db, "controlled_documents", "public", "metaldocs"); err != nil {
		return s, err
	}
	if s.documentsSchema, err = findTableSchema(ctx, db, "documents", "public", "metaldocs"); err != nil {
		return s, err
	}
	if s.sessionsSchema, err = findTableSchema(ctx, db, "editor_sessions", "public", "metaldocs"); err != nil {
		return s, err
	}
	if s.revisionsSchema, err = findTableSchema(ctx, db, "document_revisions", "public", "metaldocs"); err != nil {
		return s, err
	}
	if s.iamSchema, err = findTableSchema(ctx, db, "iam_users", "metaldocs", "public"); err != nil {
		return s, err
	}
	s.templateHasStatus = columnExists(ctx, db, s.templatesSchema, "templates_v2_template", "status")
	s.docHasBridge = columnExists(ctx, db, s.documentsSchema, "documents", "controlled_document_id")
	s.docHasSnapshots = columnExists(ctx, db, s.documentsSchema, "documents", "body_docx_snapshot_s3_key")
	return s, nil
}

func findTableSchema(ctx context.Context, db *sql.DB, table string, preferred ...string) (string, error) {
	for _, schema := range preferred {
		var exists bool
		err := db.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1 FROM information_schema.tables
  WHERE table_schema = $1 AND table_name = $2
)`, schema, table).Scan(&exists)
		if err != nil {
			return "", err
		}
		if exists {
			return schema, nil
		}
	}
	return "", fmt.Errorf("table %s not found in %s", table, strings.Join(preferred, ", "))
}

func columnExists(ctx context.Context, db *sql.DB, schema, table, column string) bool {
	var exists bool
	err := db.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1 FROM information_schema.columns
  WHERE table_schema = $1 AND table_name = $2 AND column_name = $3
)`, schema, table, column).Scan(&exists)
	return err == nil && exists
}

func lookupAdminUser(ctx context.Context, db *sql.DB, schema string) (string, error) {
	idColumn := "id"
	if !columnExists(ctx, db, schema, "iam_users", idColumn) {
		idColumn = "user_id"
	}
	if columnExists(ctx, db, schema, "iam_users", "username") {
		q := fmt.Sprintf(`SELECT %s::text FROM %s WHERE username = $1 LIMIT 1`, pqIdent(idColumn), pqTable(schema, "iam_users"))
		var id string
		if err := db.QueryRowContext(ctx, q, adminUsername).Scan(&id); err != nil {
			return "", err
		}
		return id, nil
	}
	authSchema, err := findTableSchema(ctx, db, "auth_identities", schema, "metaldocs", "public")
	if err != nil {
		return "", err
	}
	q := fmt.Sprintf(`
SELECT u.%s::text
  FROM %s u
  JOIN %s a ON a.user_id = u.%s
 WHERE a.username = $1
 LIMIT 1`, pqIdent(idColumn), pqTable(schema, "iam_users"), pqTable(authSchema, "auth_identities"), pqIdent(idColumn))
	var id string
	if err := db.QueryRowContext(ctx, q, adminUsername).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func lookupFirstCode(ctx context.Context, db *sql.DB, schema, table string) (string, error) {
	q := fmt.Sprintf(`SELECT code FROM %s WHERE tenant_id = $1::uuid ORDER BY code LIMIT 1`, pqTable(schema, table))
	var code string
	if err := db.QueryRowContext(ctx, q, tenantID).Scan(&code); err != nil {
		return "", err
	}
	return code, nil
}

func uploadDOCX(ctx context.Context, docx []byte, key string) error {
	client, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccessKey, minioSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		return err
	}
	exists, err := client.BucketExists(ctx, minioBucket)
	if err != nil {
		return err
	}
	if !exists {
		if err := client.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}
	_, err = client.PutObject(ctx, minioBucket, key, bytes.NewReader(docx), int64(len(docx)), minio.PutObjectOptions{
		ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	})
	return err
}

func insertRows(ctx context.Context, db *sql.DB, schema dbSchema, in seedInput) (string, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
		return "", err
	}

	templateID := uuid.NewString()
	templateVersionID := uuid.NewString()
	controlledID := uuid.NewString()
	documentID := uuid.NewString()
	sessionID := uuid.NewString()
	now := time.Now().UTC()
	title := "Outline Plugin Seed Document"
	code := "OUTLINE-" + strings.ToUpper(documentID[:8])

	if err := insertTemplate(ctx, tx, schema, templateID, in.ProfileCode, in.UserID, now); err != nil {
		return "", fmt.Errorf("insert template: %w", err)
	}
	if err := insertTemplateVersion(ctx, tx, schema, templateVersionID, templateID, in.StorageKey, in.ContentHash, in.UserID, now); err != nil {
		return "", fmt.Errorf("insert template version: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		fmt.Sprintf(`UPDATE %s SET latest_version = 1, published_version_id = $1 WHERE id = $2`, pqTable(schema.templatesSchema, "templates_v2_template")),
		templateVersionID, templateID,
	); err != nil {
		return "", fmt.Errorf("publish template: %w", err)
	}
	if err := insertControlledDocument(ctx, tx, schema, controlledID, in.ProfileCode, in.ProcessAreaCode, code, title, in.UserID, templateVersionID, now); err != nil {
		return "", fmt.Errorf("insert controlled document: %w", err)
	}
	if err := insertDocument(ctx, tx, schema, documentID, templateVersionID, title, in.UserID, controlledID, in.ProfileCode, in.ProcessAreaCode, in.StorageKey, in.ContentHashBytes, now); err != nil {
		return "", fmt.Errorf("insert document: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO %s (id, document_id, user_id, expires_at, last_acknowledged_revision_id, status)
VALUES ($1::uuid, $2::uuid, $3, now() + interval '5 minutes', $4::uuid, 'active')`, pqTable(schema.sessionsSchema, "editor_sessions")),
		sessionID, documentID, in.UserID, in.RevisionID,
	); err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO %s (id, document_id, parent_revision_id, session_id, storage_key, content_hash, form_data_snapshot)
VALUES ($1::uuid, $2::uuid, NULL, $3::uuid, $4, $5, $6::jsonb)`, pqTable(schema.revisionsSchema, "document_revisions")),
		in.RevisionID, documentID, sessionID, in.StorageKey, in.ContentHash, formDataJSON,
	); err != nil {
		return "", fmt.Errorf("insert revision: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		fmt.Sprintf(`UPDATE %s SET current_revision_id = $1::uuid, active_session_id = $2::uuid, updated_at = $3 WHERE id = $4::uuid`, pqTable(schema.documentsSchema, "documents")),
		in.RevisionID, sessionID, now, documentID,
	); err != nil {
		return "", fmt.Errorf("update document pointers: %w", err)
	}

	return documentID, tx.Commit()
}

func insertTemplate(ctx context.Context, tx *sql.Tx, schema dbSchema, templateID, profileCode, userID string, now time.Time) error {
	table := pqTable(schema.templatesSchema, "templates_v2_template")
	if schema.templateHasStatus {
		_, err := tx.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO %s
  (id, tenant_id, doc_type_code, key, name, description, areas, visibility, specific_areas, latest_version, created_by, created_at, status)
VALUES
  ($1::uuid, $2, $3, $4, $5, '', ARRAY[$6]::text[], 'internal', '{}'::text[], 0, $7, $8, 'published')`,
			table),
			templateID, tenantID, profileCode, "outline-seed-"+templateID, "Outline Seed Template", profileCode, userID, now,
		)
		return err
	}
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO %s
  (id, tenant_id, doc_type_code, key, name, description, areas, visibility, specific_areas, latest_version, created_by, created_at)
VALUES
  ($1::uuid, $2, $3, $4, $5, '', ARRAY[$6]::text[], 'internal', '{}'::text[], 0, $7, $8)`,
		table),
		templateID, tenantID, profileCode, "outline-seed-"+templateID, "Outline Seed Template", profileCode, userID, now,
	)
	return err
}

func insertTemplateVersion(ctx context.Context, tx *sql.Tx, schema dbSchema, versionID, templateID, storageKey, contentHash, userID string, now time.Time) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO %s
  (id, template_id, version_number, status, docx_storage_key, content_hash, metadata_schema,
   placeholder_schema, author_id, pending_approver_role, published_at, created_at)
VALUES
  ($1::uuid, $2::uuid, 1, 'published', $3, $4, $5::jsonb, $6::jsonb, $7, '', $8, $8)`,
		pqTable(schema.versionsSchema, "templates_v2_template_version")),
		versionID, templateID, storageKey, contentHash, emptyJSON, placeholderJSON, userID, now,
	)
	return err
}

func insertControlledDocument(ctx context.Context, tx *sql.Tx, schema dbSchema, id, profileCode, areaCode, code, title, userID, templateVersionID string, now time.Time) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO %s
  (id, tenant_id, profile_code, process_area_code, code, sequence_num, title, owner_user_id,
   override_template_version_id, status, created_at, updated_at)
VALUES
  ($1::uuid, $2::uuid, $3, $4, $5, NULL, $6, $7, $8::uuid, 'active', $9, $9)`,
		pqTable(schema.controlledSchema, "controlled_documents")),
		id, tenantID, profileCode, areaCode, code, title, userID, templateVersionID, now,
	)
	return err
}

func insertDocument(ctx context.Context, tx *sql.Tx, schema dbSchema, documentID, templateVersionID, name, userID, controlledID, profileCode, areaCode, storageKey string, contentHash []byte, now time.Time) error {
	columns := []string{"id", "tenant_id", "template_version_id", "name", "status", "form_data_json", "created_by", "created_at", "updated_at"}
	values := []string{"$1::uuid", "$2::uuid", "$3::uuid", "$4", "'draft'", "$5::jsonb", "$6", "$7", "$7"}
	args := []any{documentID, tenantID, templateVersionID, name, formDataJSON, userID, now}

	if schema.docHasBridge {
		columns = append(columns, "controlled_document_id", "profile_code_snapshot", "process_area_code_snapshot")
		values = append(values, "$8::uuid", "$9", "$10")
		args = append(args, controlledID, profileCode, areaCode)
	}
	if schema.docHasSnapshots {
		base := len(args) + 1
		columns = append(columns,
			"placeholder_schema_snapshot",
			"composition_config_snapshot",
			"body_docx_snapshot_s3_key",
			"body_docx_hash",
		)
		values = append(values,
			fmt.Sprintf("$%d::jsonb", base),
			fmt.Sprintf("$%d::jsonb", base+1),
			fmt.Sprintf("$%d", base+2),
			fmt.Sprintf("$%d", base+3),
		)
		args = append(args, placeholderJSON, emptyJSON, storageKey, contentHash)
	}

	q := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`,
		pqTable(schema.documentsSchema, "documents"),
		strings.Join(columns, ", "),
		strings.Join(values, ", "),
	)
	_, err := tx.ExecContext(ctx, q, args...)
	return err
}

func buildDOCX() ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"[Content_Types].xml":          contentTypesXML,
		"_rels/.rels":                  rootRelsXML,
		"word/_rels/document.xml.rels": documentRelsXML,
		"word/document.xml":            documentXML(),
		"word/styles.xml":              stylesXML,
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(body)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func documentXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    ` + heading("Heading1", "Section One") + `
    ` + heading("Heading2", "Scope") + `
    ` + heading("Heading3", "Detailed Requirements") + `
    ` + heading("Heading1", "Section Two") + `
    <w:p><w:r><w:t>This seed document exists to validate outline rendering.</w:t></w:r></w:p>
    <w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>
  </w:body>
</w:document>`
}

func heading(style, text string) string {
	return fmt.Sprintf(`<w:p><w:pPr><w:pStyle w:val="%s"/></w:pPr><w:r><w:t>%s</w:t></w:r></w:p>`, style, xmlEscape(text))
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func pqTable(schema, table string) string {
	return pqIdent(schema) + "." + pqIdent(table)
}

func pqIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`

const rootRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

const documentRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`

const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="heading 1"/>
    <w:basedOn w:val="Normal"/>
    <w:uiPriority w:val="9"/>
    <w:qFormat/>
    <w:pPr><w:outlineLvl w:val="0"/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="32"/></w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading2">
    <w:name w:val="heading 2"/>
    <w:basedOn w:val="Normal"/>
    <w:uiPriority w:val="9"/>
    <w:qFormat/>
    <w:pPr><w:outlineLvl w:val="1"/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="28"/></w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading3">
    <w:name w:val="heading 3"/>
    <w:basedOn w:val="Normal"/>
    <w:uiPriority w:val="9"/>
    <w:qFormat/>
    <w:pPr><w:outlineLvl w:val="2"/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="24"/></w:rPr>
  </w:style>
</w:styles>`
