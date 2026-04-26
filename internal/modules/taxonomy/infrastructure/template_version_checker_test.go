package infrastructure

import (
	"strings"
	"testing"
)

func TestTemplateVersionQueryUsesDocumentTypeCode(t *testing.T) {
	if !strings.Contains(templateVersionQuery, "t.doc_type_code") {
		t.Fatalf("templateVersionQuery should select t.doc_type_code, got: %s", templateVersionQuery)
	}
	if strings.Contains(templateVersionQuery, "t.profile_code") {
		t.Fatalf("templateVersionQuery should not select t.profile_code, got: %s", templateVersionQuery)
	}
}
