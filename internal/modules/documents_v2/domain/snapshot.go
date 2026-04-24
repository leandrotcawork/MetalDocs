package domain

import "crypto/sha256"

// TemplateSnapshot holds the template artifacts copied onto a document at create time.
type TemplateSnapshot struct {
	PlaceholderSchemaJSON []byte
	CompositionJSON       []byte
	ZonesSchemaJSON       []byte
	BodyDocxBytes         []byte
	BodyDocxS3Key         string
}

// SnapshotHashes holds the sha256 digests of each snapshot field.
type SnapshotHashes struct {
	PlaceholderSchemaHash []byte
	CompositionHash       []byte
	BodyDocxHash          []byte
}

// Hashes computes deterministic sha256 digests for the snapshot fields.
func (s TemplateSnapshot) Hashes() SnapshotHashes {
	ph := sha256.Sum256(s.PlaceholderSchemaJSON)
	ch := sha256.Sum256(s.CompositionJSON)
	bh := sha256.Sum256(s.BodyDocxBytes)
	return SnapshotHashes{ph[:], ch[:], bh[:]}
}
