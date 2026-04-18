package domain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// RenderOptions captures all user-controlled PDF rendering knobs.
// It is deterministically serialised into ComputeCompositeHash.
type RenderOptions struct {
	PaperSize  string // "A4" or "Letter"
	LandscapeP bool
}

// domainSep is a byte that cannot appear in a JSON string or UUID,
// used to prevent cross-field hash collisions.
const domainSep = byte(0x1e)

// ComputeCompositeHash returns a 32-byte SHA-256 over the inputs that
// determine whether two export calls would produce identical PDFs.
//
// Inputs: contentHash (hex string from revision.StorageKey), templateVersionID,
// grammarVersion (a semver string for the rendering grammar), docgenV2Version
// (e.g. "docgen-v2@0.4.0"), and canonical-JSON-encoded RenderOptions.
//
// Any change to any input produces a different hash -> cache miss.
func ComputeCompositeHash(
	contentHash []byte,
	templateVersionID, grammarVersion, docgenV2Version string,
	opts RenderOptions,
) ([]byte, error) {
	optsJSON, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("marshal render options: %w", err)
	}
	h := sha256.New()
	h.Write(contentHash)
	h.Write([]byte{domainSep})
	h.Write([]byte(templateVersionID))
	h.Write([]byte{domainSep})
	h.Write([]byte(grammarVersion))
	h.Write([]byte{domainSep})
	h.Write([]byte(docgenV2Version))
	h.Write([]byte{domainSep})
	h.Write(optsJSON)
	return h.Sum(nil), nil
}
