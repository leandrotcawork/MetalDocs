package fanout

import (
	"context"
	"encoding/hex"
	"testing"
)

func TestReconstruct_DriftDetection(t *testing.T) {
	original := make([]byte, 32)
	for i := range original {
		original[i] = 0xAA
	}
	differentHash := hex.EncodeToString(make([]byte, 32)) // all zeros — different from original

	client := &fakeFanoutClient{resp: FanoutResponse{
		ContentHash:    differentHash,
		FinalDocxS3Key: "should/not/be/written",
	}}
	writer := &fakeReconstructionWriter{}
	svc := NewReconstructService(
		fakeReconstructInputs{req: FanoutRequest{TenantID: "t", RevisionID: "r"}, originalHash: original},
		client,
		writer,
		EngineVersions{EigenpalVer: "v1", DocxtemplaterVer: "v2"},
		fixedNow,
	)

	entry, err := svc.Reconstruct(context.Background(), "t", "r")
	if err != nil {
		t.Fatalf("Reconstruct: %v", err)
	}

	// matches_original must be false — bytes differ
	if entry.MatchesOriginal {
		t.Errorf("MatchesOriginal=true, want false for divergent hash")
	}
	if entry.BytesHash != differentHash {
		t.Errorf("BytesHash=%q, want %q", entry.BytesHash, differentHash)
	}

	// writer received the entry
	if len(writer.calls) != 1 {
		t.Fatalf("writer calls=%d, want 1", len(writer.calls))
	}
	call := writer.calls[0]
	if call.tenant != "t" || call.docID != "r" {
		t.Errorf("writer args = %+v", call)
	}

	// original content_hash unchanged — ReconstructionWriter interface has no WriteFinalDocx method,
	// so structural proof: compiler enforces the interface has only AppendReconstruction.
	_ = (ReconstructionWriter)(writer) // compile-time check
}
