package fanout

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// FanoutInputsReader supplies the inputs needed to re-render a frozen revision
// plus the stored content_hash that pinned the original output.
type FanoutInputsReader interface {
	ReadForReconstruction(ctx context.Context, tenantID, revisionID string) (FanoutRequest, []byte, error)
}

// FanoutClient is the narrow port implemented by *Client.
type FanoutClient interface {
	Fanout(ctx context.Context, req FanoutRequest) (FanoutResponse, error)
}

// ReconstructionWriter persists a JSON entry into documents.reconstruction_attempts
// without touching final_docx_s3_key or content_hash.
type ReconstructionWriter interface {
	AppendReconstruction(ctx context.Context, tenantID, revisionID string, entry []byte) error
}

// EngineVersions identifies the docgen stack running the reconstruction. These
// versions vary across engine upgrades and are the whole point of the forensic
// record.
type EngineVersions struct {
	EigenpalVer      string
	DocxtemplaterVer string
}

// ReconstructionEntry is the JSON shape appended to documents.reconstruction_attempts.
type ReconstructionEntry struct {
	RenderedAt       time.Time `json:"rendered_at"`
	EigenpalVer      string    `json:"eigenpal_ver"`
	DocxtemplaterVer string    `json:"docxtemplater_ver"`
	BytesHash        string    `json:"bytes_hash"`
	MatchesOriginal  bool      `json:"matches_original"`
}

// ReconstructService re-renders a frozen revision for forensic comparison. It
// never updates final_docx_s3_key or content_hash — only appends an audit entry.
type ReconstructService struct {
	inputs  FanoutInputsReader
	client  FanoutClient
	writer  ReconstructionWriter
	engine  EngineVersions
	nowFunc func() time.Time
}

func NewReconstructService(inputs FanoutInputsReader, client FanoutClient, writer ReconstructionWriter, engine EngineVersions, nowFunc func() time.Time) *ReconstructService {
	if nowFunc == nil {
		nowFunc = time.Now
	}
	return &ReconstructService{inputs: inputs, client: client, writer: writer, engine: engine, nowFunc: nowFunc}
}

func (s *ReconstructService) Reconstruct(ctx context.Context, tenantID, revisionID string) (ReconstructionEntry, error) {
	req, originalHash, err := s.inputs.ReadForReconstruction(ctx, tenantID, revisionID)
	if err != nil {
		return ReconstructionEntry{}, fmt.Errorf("reconstruct: load inputs: %w", err)
	}

	resp, err := s.client.Fanout(ctx, req)
	if err != nil {
		return ReconstructionEntry{}, fmt.Errorf("reconstruct: fanout: %w", err)
	}

	entry := ReconstructionEntry{
		RenderedAt:       s.nowFunc().UTC(),
		EigenpalVer:      s.engine.EigenpalVer,
		DocxtemplaterVer: s.engine.DocxtemplaterVer,
		BytesHash:        resp.ContentHash,
		MatchesOriginal:  resp.ContentHash == hex.EncodeToString(originalHash),
	}

	blob, err := json.Marshal(entry)
	if err != nil {
		return ReconstructionEntry{}, fmt.Errorf("reconstruct: marshal entry: %w", err)
	}
	if err := s.writer.AppendReconstruction(ctx, tenantID, revisionID, blob); err != nil {
		return ReconstructionEntry{}, fmt.Errorf("reconstruct: append attempt: %w", err)
	}
	return entry, nil
}
