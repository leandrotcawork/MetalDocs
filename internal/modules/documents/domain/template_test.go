package domain

import "testing"

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
	if template.IsBrowserHTML() {
		t.Fatal("IsBrowserHTML() must return false for po-mddm-canvas")
	}
}
