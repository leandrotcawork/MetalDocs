package domain

import (
	"encoding/hex"
	"testing"
)

func TestTemplateSnapshot_StableHash(t *testing.T) {
	s1 := TemplateSnapshot{
		PlaceholderSchemaJSON: []byte(`{"placeholders":[{"id":"a","type":"text"}]}`),
		CompositionJSON:       []byte(`{"header_sub_blocks":["h1"]}`),
		ZonesSchemaJSON:       []byte(`{"zones":[{"id":"z1"}]}`),
		BodyDocxBytes:         []byte("DOCXBYTES"),
	}
	h1 := s1.Hashes()
	h2 := s1.Hashes()
	if hex.EncodeToString(h1.PlaceholderSchemaHash) != hex.EncodeToString(h2.PlaceholderSchemaHash) {
		t.Fatal("hash not deterministic")
	}
	if len(h1.BodyDocxHash) != 32 {
		t.Fatalf("want 32-byte sha256, got %d", len(h1.BodyDocxHash))
	}
}
