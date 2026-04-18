// Package docx_v2_test asserts the W1 scaffold compiles and is wired to
// the same import graph as the main API. This file intentionally depends
// only on the three placeholder modules so governance-check's "internal/modules
// change needs tests/" rule is satisfied, and so a future compile break in
// those placeholders is caught at CI time.
package docx_v2_test

import (
	"testing"

	"metaldocs/internal/modules/document_revisions"
	"metaldocs/internal/modules/editor_sessions"
	"metaldocs/internal/modules/templates"
)

func TestScaffoldCompiles(t *testing.T) {
	if templates.New(nil, nil, nil) == nil {
		t.Fatal("templates.New() returned nil")
	}
	if editor_sessions.New() == nil {
		t.Fatal("editor_sessions.New() returned nil")
	}
	if document_revisions.New() == nil {
		t.Fatal("document_revisions.New() returned nil")
	}
}
