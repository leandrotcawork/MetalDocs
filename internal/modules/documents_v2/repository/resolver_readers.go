package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"metaldocs/internal/modules/render/fanout"
	"metaldocs/internal/modules/render/resolvers"
)

// RevisionReader implements resolvers.RevisionReader backed by the documents table.
type RevisionReader struct{ db *sql.DB }

func NewRevisionReader(db *sql.DB) *RevisionReader { return &RevisionReader{db: db} }

func (r *RevisionReader) GetRevisionNumber(ctx context.Context, tenantID, revisionID string) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT revision_number FROM documents WHERE tenant_id=$1::uuid AND id=$2::uuid`,
		tenantID, revisionID).Scan(&n)
	return n, err
}

func (r *RevisionReader) GetEffectiveFrom(ctx context.Context, tenantID, revisionID string) (time.Time, error) {
	var t time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT coalesce(effective_from, now()) FROM documents WHERE tenant_id=$1::uuid AND id=$2::uuid`,
		tenantID, revisionID).Scan(&t)
	return t, err
}

func (r *RevisionReader) GetAuthor(ctx context.Context, tenantID, revisionID string) (resolvers.AuthorInfo, error) {
	var userID string
	err := r.db.QueryRowContext(ctx,
		`SELECT created_by FROM documents WHERE tenant_id=$1::uuid AND id=$2::uuid`,
		tenantID, revisionID).Scan(&userID)
	if err != nil {
		return resolvers.AuthorInfo{}, err
	}
	return resolvers.AuthorInfo{UserID: userID, DisplayName: userID}, nil
}

// WorkflowReader implements resolvers.WorkflowReader backed by approval tables.
type WorkflowReader struct{ db *sql.DB }

func NewWorkflowReader(db *sql.DB) *WorkflowReader { return &WorkflowReader{db: db} }

func (r *WorkflowReader) GetApprovers(ctx context.Context, tenantID, revisionID string) ([]resolvers.ApproverInfo, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.actor_user_id, s.signed_at
		  FROM approval_signoffs s
		  JOIN approval_instances ai ON ai.id = s.approval_instance_id
		 WHERE ai.tenant_id=$1::uuid AND ai.document_v2_id=$2::uuid
		   AND s.decision='approve'
		 ORDER BY s.signed_at`,
		tenantID, revisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []resolvers.ApproverInfo
	for rows.Next() {
		var a resolvers.ApproverInfo
		if err := rows.Scan(&a.UserID, &a.SignedAt); err != nil {
			return nil, err
		}
		a.DisplayName = a.UserID
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *WorkflowReader) GetFinalApprovalDate(ctx context.Context, tenantID, revisionID string) (time.Time, error) {
	var t time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT coalesce(max(s.signed_at), now())
		  FROM approval_signoffs s
		  JOIN approval_instances ai ON ai.id = s.approval_instance_id
		 WHERE ai.tenant_id=$1::uuid AND ai.document_v2_id=$2::uuid
		   AND s.decision='approve'`,
		tenantID, revisionID).Scan(&t)
	return t, err
}

// FanoutInputsReader reads stored fanout inputs for forensic reconstruction.
type FanoutInputsReader struct{ db *sql.DB }

func NewFanoutInputsReader(db *sql.DB) *FanoutInputsReader { return &FanoutInputsReader{db: db} }

// ReadForReconstruction loads the snapshot + stored values needed to re-render a frozen revision.
// Returns the FanoutRequest to replay and the original content_hash for comparison.
func (r *FanoutInputsReader) ReadForReconstruction(ctx context.Context, tenantID, revisionID string) (fanout.FanoutRequest, []byte, error) {
	var bodyDocxKey string
	var compositionRaw []byte
	var contentHash []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT coalesce(body_docx_snapshot_s3_key, ''),
		       coalesce(composition_config_snapshot, '{}')::text,
		       content_hash
		  FROM documents
		 WHERE tenant_id=$1::uuid AND id=$2::uuid`,
		tenantID, revisionID).Scan(&bodyDocxKey, &compositionRaw, &contentHash)
	if err != nil {
		return fanout.FanoutRequest{}, nil, err
	}

	fillIn := NewFillInRepository(r.db)

	values, err := fillIn.ListValues(ctx, tenantID, revisionID)
	if err != nil {
		return fanout.FanoutRequest{}, nil, err
	}
	placeholders := make(map[string]string, len(values))
	for _, v := range values {
		if v.ValueText != nil {
			placeholders[v.PlaceholderID] = *v.ValueText
		}
	}

	req := fanout.FanoutRequest{
		TenantID:          tenantID,
		RevisionID:        revisionID,
		BodyDocxS3Key:     bodyDocxKey,
		PlaceholderValues: placeholders,
		Composition:       json.RawMessage(compositionRaw),
	}
	return req, contentHash, nil
}
