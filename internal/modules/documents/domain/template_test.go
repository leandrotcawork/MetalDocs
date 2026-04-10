package domain

import (
	"os"
	"regexp"
	"testing"
)

func TestDefaultDocumentTemplateVersionsDoesNotIncludeCKEditorTemplates(t *testing.T) {
	for _, tmpl := range DefaultDocumentTemplateVersions() {
		if tmpl.TemplateKey == "po-default-canvas" || tmpl.TemplateKey == "po-default-browser" {
			t.Fatalf("ckeditor template key must not be present in defaults: %s", tmpl.TemplateKey)
		}
		if tmpl.Editor == "ckeditor5" || tmpl.ContentFormat == "html" {
			t.Fatalf(
				"ckeditor editor/contentFormat must not be present in defaults: key=%s editor=%s contentFormat=%s",
				tmpl.TemplateKey,
				tmpl.Editor,
				tmpl.ContentFormat,
			)
		}
	}
}

func TestPOMDDMCanvasSeedSQLParity(t *testing.T) {
	sqlBytes, err := os.ReadFile("../../../../migrations/0065_seed_po_mddm_canvas_template.sql")
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	sqlContent := string(sqlBytes)
	assertMatch := func(name, pattern string) {
		t.Helper()
		re := regexp.MustCompile(pattern)
		if !re.MatchString(sqlContent) {
			t.Fatalf("migration SQL missing %s: pattern %q", name, pattern)
		}
	}

	assertMatch(
		"seed insert target",
		`(?s)INSERT INTO\s+metaldocs\.document_template_versions\s*\(\s*template_key,\s*version,\s*profile_code,\s*schema_version,\s*name,\s*definition_json,\s*editor,\s*content_format,\s*body_html\s*\)`,
	)
	assertMatch(
		"template row values",
		`(?s)VALUES\s*\(\s*'po-mddm-canvas'\s*,\s*1\s*,\s*'po'\s*,\s*3\s*,\s*'PO MDDM Canvas v1'\s*,\s*'\{"type":\s*"page",\s*"id":\s*"po-mddm-root",\s*"children":\s*\[\]\}'::jsonb\s*,\s*'mddm-blocknote'\s*,\s*'mddm'\s*,\s*''\s*\)`,
	)
}

func TestDefaultDocumentTemplateVersionsPODefaultIsLast(t *testing.T) {
	var lastPO *DocumentTemplateVersion
	for _, tmpl := range DefaultDocumentTemplateVersions() {
		if tmpl.ProfileCode == "po" {
			found := tmpl
			lastPO = &found
		}
	}
	if lastPO == nil {
		t.Fatal("no PO template found in DefaultDocumentTemplateVersions()")
	}
	if lastPO.TemplateKey != "po-mddm-canvas" {
		t.Fatalf(
			"last PO template key = %q, want po-mddm-canvas (in-memory repo uses last entry as default)",
			lastPO.TemplateKey,
		)
	}
	if lastPO.Version != 1 {
		t.Fatalf("last PO template version = %d, want 1", lastPO.Version)
	}
}

func TestPOMDDMCanvasTemplateInDefaults(t *testing.T) {
	var template *DocumentTemplateVersion
	for _, tmpl := range DefaultDocumentTemplateVersions() {
		if tmpl.TemplateKey == "po-mddm-canvas" {
			found := tmpl
			template = &found
			break
		}
	}
	if template == nil {
		t.Fatal("po-mddm-canvas template not found in DefaultDocumentTemplateVersions()")
	}
	if template.ProfileCode != "po" {
		t.Fatalf("template profile = %q, want po", template.ProfileCode)
	}
	if template.SchemaVersion != 3 {
		t.Fatalf("template schema version = %d, want 3", template.SchemaVersion)
	}
	if template.Editor != "mddm-blocknote" {
		t.Fatalf("template editor = %q, want mddm-blocknote", template.Editor)
	}
	if template.ContentFormat != "mddm" {
		t.Fatalf("template content format = %q, want mddm", template.ContentFormat)
	}
	if !template.IsMDDMEditor() {
		t.Fatal("IsMDDMEditor() must return true for po-mddm-canvas")
	}
	if !template.IsBrowserEditor() {
		t.Fatal("IsBrowserEditor() must return true for po-mddm-canvas")
	}
}
