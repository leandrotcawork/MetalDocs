package domain

import (
	"os"
	"strings"
	"testing"
)

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
		t.Fatalf("last PO template key = %q, want po-mddm-canvas (in-memory repo uses last entry as default)", lastPO.TemplateKey)
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
	if template.IsBrowserHTML() {
		t.Fatal("IsBrowserHTML() must return false for po-mddm-canvas")
	}
}

func TestPOMDDMCanvasGoSQLParity(t *testing.T) {
	migrationPath := "../../../../migrations/0065_seed_po_mddm_canvas_template.sql"
	sqlBytes, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}
	sqlContent := string(sqlBytes)

	checks := map[string]string{
		"template_key":   "'po-mddm-canvas'",
		"profile_code":   "'po'",
		"schema_version": "3",
		"editor":         "'mddm-blocknote'",
		"content_format": "'mddm'",
	}
	for field, expected := range checks {
		if !strings.Contains(sqlContent, expected) {
			t.Errorf("migration SQL missing %s = %s", field, expected)
		}
	}
}

