// Package docx_v2_test asserts the W1 scaffold compiles and is wired to
// the same import graph as the main API.
package docx_v2_test

import (
	"testing"

	"metaldocs/internal/modules/templates_v2/domain"
)

func TestScaffoldCompiles(t *testing.T) {
	_ = domain.ApprovalConfig{}
}
