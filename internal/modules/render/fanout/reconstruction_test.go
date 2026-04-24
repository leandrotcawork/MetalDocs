package fanout

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeReconstructInputs struct {
	req          FanoutRequest
	originalHash []byte
	err          error
}

func (f fakeReconstructInputs) ReadForReconstruction(_ context.Context, _, _ string) (FanoutRequest, []byte, error) {
	if f.err != nil {
		return FanoutRequest{}, nil, f.err
	}
	return f.req, f.originalHash, nil
}

type fakeFanoutClient struct {
	resp FanoutResponse
	err  error
	got  FanoutRequest
}

func (f *fakeFanoutClient) Fanout(_ context.Context, req FanoutRequest) (FanoutResponse, error) {
	f.got = req
	return f.resp, f.err
}

type fakeReconstructionWriter struct {
	calls []reconstructionCall
	err   error
}

type reconstructionCall struct {
	tenant, docID string
	entry         []byte
}

func (w *fakeReconstructionWriter) AppendReconstruction(_ context.Context, tenant, docID string, entry []byte) error {
	w.calls = append(w.calls, reconstructionCall{tenant: tenant, docID: docID, entry: append([]byte(nil), entry...)})
	return w.err
}

func fixedNow() time.Time { return time.Date(2026, 4, 23, 15, 4, 5, 0, time.UTC) }

func newReconstructService(t *testing.T, inputs fakeReconstructInputs, client *fakeFanoutClient, writer *fakeReconstructionWriter) *ReconstructService {
	t.Helper()
	return NewReconstructService(
		inputs,
		client,
		writer,
		EngineVersions{EigenpalVer: "eigenpal@1.2.3", DocxtemplaterVer: "docxtemplater@3.45.0"},
		fixedNow,
	)
}

func TestReconstruct_MatchesOriginal(t *testing.T) {
	original, _ := hex.DecodeString("aa" + "bb" + "cc" + "dd" +
		"00000000000000000000000000000000000000000000000000000000")
	client := &fakeFanoutClient{resp: FanoutResponse{
		ContentHash:    hex.EncodeToString(original),
		FinalDocxS3Key: "ignored/by/reconstruct",
	}}
	writer := &fakeReconstructionWriter{}
	svc := newReconstructService(t, fakeReconstructInputs{
		req:          FanoutRequest{TenantID: "t", RevisionID: "r"},
		originalHash: original,
	}, client, writer)

	entry, err := svc.Reconstruct(context.Background(), "t", "r")
	if err != nil {
		t.Fatalf("Reconstruct err=%v", err)
	}
	if !entry.MatchesOriginal {
		t.Errorf("MatchesOriginal=false, want true")
	}
	if entry.BytesHash != hex.EncodeToString(original) {
		t.Errorf("BytesHash=%q", entry.BytesHash)
	}
	if entry.EigenpalVer != "eigenpal@1.2.3" || entry.DocxtemplaterVer != "docxtemplater@3.45.0" {
		t.Errorf("engine vers not propagated: %+v", entry)
	}
	if !entry.RenderedAt.Equal(fixedNow()) {
		t.Errorf("RenderedAt=%v", entry.RenderedAt)
	}
	if len(writer.calls) != 1 {
		t.Fatalf("AppendReconstruction called %d times, want 1", len(writer.calls))
	}
	if writer.calls[0].tenant != "t" || writer.calls[0].docID != "r" {
		t.Errorf("writer args = %+v", writer.calls[0])
	}
	var persisted ReconstructionEntry
	if err := json.Unmarshal(writer.calls[0].entry, &persisted); err != nil {
		t.Fatalf("persisted entry not JSON: %v", err)
	}
	if persisted.BytesHash != entry.BytesHash || !persisted.MatchesOriginal {
		t.Errorf("persisted=%+v", persisted)
	}
}

func TestReconstruct_MismatchMarksDivergence(t *testing.T) {
	original := make([]byte, 32)
	original[0] = 0x01
	client := &fakeFanoutClient{resp: FanoutResponse{
		ContentHash: hex.EncodeToString([]byte("different-hash-000000000000000000")),
	}}
	writer := &fakeReconstructionWriter{}
	svc := newReconstructService(t, fakeReconstructInputs{
		req:          FanoutRequest{TenantID: "t", RevisionID: "r"},
		originalHash: original,
	}, client, writer)

	entry, err := svc.Reconstruct(context.Background(), "t", "r")
	if err != nil {
		t.Fatalf("Reconstruct err=%v", err)
	}
	if entry.MatchesOriginal {
		t.Errorf("MatchesOriginal=true on divergent hash")
	}
	if len(writer.calls) != 1 {
		t.Fatalf("writer calls=%d", len(writer.calls))
	}
}

func TestReconstruct_FanoutRequestPropagated(t *testing.T) {
	req := FanoutRequest{
		TenantID:          "t",
		RevisionID:        "r",
		BodyDocxS3Key:     "body.docx",
		PlaceholderValues: map[string]string{"title": "Hello"},
		ZoneContent:       map[string]string{"z1": "<w:p/>"},
	}
	client := &fakeFanoutClient{resp: FanoutResponse{ContentHash: "00"}}
	svc := newReconstructService(t, fakeReconstructInputs{req: req, originalHash: []byte{0}}, client, &fakeReconstructionWriter{})

	if _, err := svc.Reconstruct(context.Background(), "t", "r"); err != nil {
		t.Fatal(err)
	}
	if client.got.BodyDocxS3Key != "body.docx" || client.got.PlaceholderValues["title"] != "Hello" {
		t.Errorf("fanout got=%+v", client.got)
	}
}

func TestReconstruct_FanoutErrorPropagates(t *testing.T) {
	svc := newReconstructService(t,
		fakeReconstructInputs{req: FanoutRequest{}, originalHash: []byte{0}},
		&fakeFanoutClient{err: errors.New("boom")},
		&fakeReconstructionWriter{},
	)
	if _, err := svc.Reconstruct(context.Background(), "t", "r"); err == nil {
		t.Fatal("want error")
	}
}
