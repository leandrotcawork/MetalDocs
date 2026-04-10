package mddm

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalizeMDDM_ParityWithTSFixture(t *testing.T) {
	canonicalDir := filepath.Join("..", "..", "..", "..", "..", "shared", "schemas", "test-fixtures", "canonical")
	inputBytes, err := os.ReadFile(filepath.Join(canonicalDir, "input-mixed-order.json"))
	if err != nil {
		t.Fatal(err)
	}
	expectedBytes, err := os.ReadFile(filepath.Join(canonicalDir, "output-mixed-order.json"))
	if err != nil {
		t.Fatal(err)
	}

	var input map[string]any
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		t.Fatal(err)
	}

	canonical, err := CanonicalizeMDDM(input)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}

	actualBytes, err := MarshalCanonical(canonical)
	if err != nil {
		t.Fatal(err)
	}

	// Re-marshal expected via json.Marshal for normalization
	var expected map[string]any
	if err := json.Unmarshal(expectedBytes, &expected); err != nil {
		t.Fatal(err)
	}
	expectedNormalized, err := MarshalCanonical(expected)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(actualBytes, expectedNormalized) {
		t.Errorf("canonical mismatch:\nexpected: %s\nactual:   %s", expectedNormalized, actualBytes)
	}
}

func TestCanonicalizeMDDM_FieldInlineParity(t *testing.T) {
	canonicalDir := filepath.Join("..", "..", "..", "..", "..", "shared", "schemas", "test-fixtures", "canonical")
	inputBytes, err := os.ReadFile(filepath.Join(canonicalDir, "input-field-inline.json"))
	if err != nil {
		t.Fatal(err)
	}
	expectedBytes, err := os.ReadFile(filepath.Join(canonicalDir, "output-field-inline.json"))
	if err != nil {
		t.Fatal(err)
	}

	var input map[string]any
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		t.Fatal(err)
	}
	canonical, err := CanonicalizeMDDM(input)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}
	actualBytes, err := MarshalCanonical(canonical)
	if err != nil {
		t.Fatal(err)
	}

	var expected map[string]any
	if err := json.Unmarshal(expectedBytes, &expected); err != nil {
		t.Fatal(err)
	}
	expectedNormalized, err := MarshalCanonical(expected)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(actualBytes, expectedNormalized) {
		t.Errorf("field inline parity mismatch:\nexpected: %s\nactual:   %s", expectedNormalized, actualBytes)
	}
}
