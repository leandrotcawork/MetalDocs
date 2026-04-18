package domain_test

import (
	"bytes"
	"testing"

	"metaldocs/internal/modules/documents_v2/domain"
)

func TestComputeCompositeHash_Deterministic(t *testing.T) {
	opts := domain.RenderOptions{PaperSize: "A4", LandscapeP: false}
	h1, err := domain.ComputeCompositeHash([]byte("abc123"), "tpl-v1", "grammar-v1", "docgen-v2@0.4.0", opts)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := domain.ComputeCompositeHash([]byte("abc123"), "tpl-v1", "grammar-v1", "docgen-v2@0.4.0", opts)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(h1, h2) {
		t.Fatal("same inputs must produce same hash")
	}
	if len(h1) != 32 {
		t.Fatalf("hash must be 32 bytes, got %d", len(h1))
	}
}

func TestComputeCompositeHash_DifferentInputsProduceDifferentHashes(t *testing.T) {
	opts := domain.RenderOptions{PaperSize: "A4", LandscapeP: false}
	base, _ := domain.ComputeCompositeHash([]byte("abc123"), "tpl-v1", "grammar-v1", "docgen-v2@0.4.0", opts)

	cases := []struct {
		name string
		h    func() []byte
	}{
		{"different content", func() []byte {
			h, _ := domain.ComputeCompositeHash([]byte("xyz789"), "tpl-v1", "grammar-v1", "docgen-v2@0.4.0", opts); return h
		}},
		{"different template", func() []byte {
			h, _ := domain.ComputeCompositeHash([]byte("abc123"), "tpl-v2", "grammar-v1", "docgen-v2@0.4.0", opts); return h
		}},
		{"different grammar", func() []byte {
			h, _ := domain.ComputeCompositeHash([]byte("abc123"), "tpl-v1", "grammar-v2", "docgen-v2@0.4.0", opts); return h
		}},
		{"different docgen version", func() []byte {
			h, _ := domain.ComputeCompositeHash([]byte("abc123"), "tpl-v1", "grammar-v1", "docgen-v2@0.5.0", opts); return h
		}},
		{"landscape=true", func() []byte {
			h, _ := domain.ComputeCompositeHash([]byte("abc123"), "tpl-v1", "grammar-v1", "docgen-v2@0.4.0",
				domain.RenderOptions{PaperSize: "A4", LandscapeP: true}); return h
		}},
		{"paper Letter", func() []byte {
			h, _ := domain.ComputeCompositeHash([]byte("abc123"), "tpl-v1", "grammar-v1", "docgen-v2@0.4.0",
				domain.RenderOptions{PaperSize: "Letter", LandscapeP: false}); return h
		}},
	}
	for _, c := range cases {
		got := c.h()
		if bytes.Equal(base, got) {
			t.Errorf("case %q: expected different hash but got same", c.name)
		}
	}
}

// TestComputeCompositeHash_NoCrossFieldCollision verifies the 0x1e separator
// prevents "abc" + "def" == "abcd" + "ef" style collisions across adjacent fields.
func TestComputeCompositeHash_NoCrossFieldCollision(t *testing.T) {
	opts := domain.RenderOptions{PaperSize: "A4"}
	h1, _ := domain.ComputeCompositeHash([]byte("abc"), "defghi", "grammar-v1", "docgen-v2@0.4.0", opts)
	h2, _ := domain.ComputeCompositeHash([]byte("abcdef"), "ghi", "grammar-v1", "docgen-v2@0.4.0", opts)
	if bytes.Equal(h1, h2) {
		t.Fatal("cross-field hash collision detected — separator not working")
	}
}
