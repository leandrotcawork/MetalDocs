package application

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// IdempotencyInput holds the fields for key derivation.
// Timestamp source MUST be server-authoritative (time.Now()) — client timestamps rejected at handler.
// Godoc contract: "server clock only — never trust client timestamp".
type IdempotencyInput struct {
	ActorUserID     string
	DocumentID      string
	StageInstanceID string
	Decision        string
	Timestamp       time.Time // server-set; truncated to second inside ComputeIdempotencyKey
}

// ComputeIdempotencyKey returns a lowercase hex SHA-256 idempotency key.
// Second-bucket granularity prevents double-click dupes while allowing
// intentional re-sign after stage reopen (different second window).
//
// Server clock only — never trust client timestamp.
func ComputeIdempotencyKey(input IdempotencyInput) string {
	// Defense-in-depth: normalize timestamp regardless of caller.
	ts := input.Timestamp.UTC().Truncate(time.Second)

	m := map[string]any{
		"actor_user_id":     input.ActorUserID,
		"document_id":       input.DocumentID,
		"stage_instance_id": input.StageInstanceID,
		"decision":          input.Decision,
		"timestamp":         ts.Format(time.RFC3339),
	}

	// Use canonicalize for deterministic key ordering.
	b, _ := canonicalize(m)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
