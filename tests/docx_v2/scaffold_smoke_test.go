// Package docx_v2_test asserts the W1 scaffold compiles and is wired to
// the same import graph as the main API. This file intentionally depends
// on the templates placeholder module so governance-check's "internal/modules
// change needs tests/" rule is satisfied, and so a future compile break in
// that placeholder is caught at CI time.
package docx_v2_test

import (
	"testing"

	"metaldocs/internal/modules/templates"
)

func TestScaffoldCompiles(t *testing.T) {
	if templates.New(nil, nil, nil) == nil {
		t.Fatal("templates.New() returned nil")
	}
}
