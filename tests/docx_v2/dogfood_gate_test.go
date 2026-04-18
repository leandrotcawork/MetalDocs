package docx_v2_test

import (
	"os"
	"strings"
	"testing"
)

// TestW4DogfoodGate fails CI if the soak evidence file has not been filled in
// and signed off before W5 changes are merged. It reads the evidence file and
// checks that the gate status line is set to "PASS" (not "PENDING" or "FAIL").
//
// To pass this gate:
// 1. Complete the dogfood soak per docs/runbooks/docx-v2-w4-dogfood.md
// 2. Fill in docs/runbooks/docx-v2-w4-soak-evidence.md
// 3. Change "GATE: PENDING" to "GATE: PASS"
// 4. Commit and push
func TestW4DogfoodGate(t *testing.T) {
	evidencePath := "../../docs/runbooks/docx-v2-w4-soak-evidence.md"
	data, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatalf("soak evidence file not found at %s: %v", evidencePath, err)
	}

	content := string(data)

	if strings.Contains(content, "GATE: PENDING") {
		t.Fatal("W4 dogfood gate is PENDING — complete the soak and set GATE: PASS in " + evidencePath)
	}
	if strings.Contains(content, "GATE: FAIL") {
		t.Fatal("W4 dogfood gate is FAIL — resolve issues before merging W5")
	}
	if !strings.Contains(content, "GATE: PASS") {
		t.Fatal("W4 dogfood gate status not found in " + evidencePath + " — expected 'GATE: PASS'")
	}
}
