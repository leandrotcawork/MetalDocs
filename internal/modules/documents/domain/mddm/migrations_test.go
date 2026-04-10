package mddm

import (
	"testing"
)

func TestMigrateForward_NoMigrationsNeeded(t *testing.T) {
	envelope := map[string]any{
		"mddm_version": 1,
		"blocks":       []any{},
		"template_ref": nil,
	}
	result, err := MigrateEnvelopeForward(envelope, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["mddm_version"] != 1 {
		t.Errorf("expected mddm_version 1, got %v", result["mddm_version"])
	}
}

func TestMigrateForward_RejectsUnknownVersion(t *testing.T) {
	envelope := map[string]any{
		"mddm_version": 99,
		"blocks":       []any{},
		"template_ref": nil,
	}
	_, err := MigrateEnvelopeForward(envelope, 1)
	if err == nil {
		t.Error("expected error for unknown source version")
	}
}
