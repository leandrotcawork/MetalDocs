package domain_test

import (
	"testing"

	registrydomain "metaldocs/internal/modules/registry/domain"
)

func TestAutoCode_Format(t *testing.T) {
	got := registrydomain.AutoCode("po", 5)
	want := "PO-05"
	if got != want {
		t.Errorf("AutoCode: got %q, want %q", got, want)
	}
}
