package objectstore_test

import (
	"testing"

	"metaldocs/internal/platform/objectstore"
)

func TestTemplateDocxKey(t *testing.T) {
	k := objectstore.TemplateDocxKey("t1", "tpl1", 3)
	if k != "tenants/t1/templates/tpl1/v3.docx" {
		t.Fatalf("unexpected key: %s", k)
	}
}

func TestTemplateSchemaKey(t *testing.T) {
	k := objectstore.TemplateSchemaKey("t1", "tpl1", 3)
	if k != "tenants/t1/templates/tpl1/v3.schema.json" {
		t.Fatalf("unexpected key: %s", k)
	}
}
