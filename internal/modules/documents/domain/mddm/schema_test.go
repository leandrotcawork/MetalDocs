package mddm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaValidation_AcceptsValidFixtures(t *testing.T) {
	validDir := filepath.Join("..", "..", "..", "..", "..", "shared", "schemas", "test-fixtures", "valid")
	entries, err := os.ReadDir(validDir)
	if err != nil {
		t.Fatalf("read valid fixtures: %v", err)
	}
	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(validDir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateMDDMBytes(data); err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
		})
	}
}

func TestSchemaValidation_RejectsInvalidFixtures(t *testing.T) {
	invalidDir := filepath.Join("..", "..", "..", "..", "..", "shared", "schemas", "test-fixtures", "invalid")
	entries, err := os.ReadDir(invalidDir)
	if err != nil {
		t.Fatalf("read invalid fixtures: %v", err)
	}
	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(invalidDir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateMDDMBytes(data); err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}
