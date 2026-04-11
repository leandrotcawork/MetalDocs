package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
)

func TestShadowDiffRepository_Insert_Roundtrip(t *testing.T) {
	db := newTestDB(t)
	repo := NewShadowDiffRepository(db)

	event := domain.ShadowDiffEvent{
		DocumentID:        "doc-1",
		VersionNumber:     3,
		UserIDHash:        "hashed-user-id",
		CurrentXMLHash:    "current-hash",
		ShadowXMLHash:     "shadow-hash",
		DiffSummary:       map[string]any{"blocks_equal": 42, "blocks_different": 0},
		CurrentDurationMs: 1200,
		ShadowDurationMs:  900,
		ShadowError:       "",
		RecordedAt:        time.Now().UTC().Truncate(time.Millisecond),
		TraceID:           "trace-xyz",
	}

	if err := repo.Insert(context.Background(), event); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	var got struct {
		DocumentID     string
		VersionNumber  int
		UserIDHash     string
		DiffSummaryRaw []byte
	}
	err := db.QueryRowContext(context.Background(),
		`SELECT document_id, version_number, user_id_hash, diff_summary
         FROM metaldocs.mddm_shadow_diff_events
         WHERE document_id = $1 AND version_number = $2
         ORDER BY id DESC LIMIT 1`,
		event.DocumentID, event.VersionNumber).
		Scan(&got.DocumentID, &got.VersionNumber, &got.UserIDHash, &got.DiffSummaryRaw)
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if got.DocumentID != event.DocumentID || got.VersionNumber != event.VersionNumber {
		t.Fatalf("row mismatch: %+v", got)
	}

	var summary map[string]any
	if err := json.Unmarshal(got.DiffSummaryRaw, &summary); err != nil {
		t.Fatalf("decode diff_summary: %v", err)
	}
	if summary["blocks_equal"].(float64) != 42 {
		t.Fatalf("diff_summary lost data: %+v", summary)
	}
}
