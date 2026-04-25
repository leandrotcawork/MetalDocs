package application

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	v2dom "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/documents_v2/repository"
	"metaldocs/internal/modules/render/fanout"
	"metaldocs/internal/modules/render/resolvers"
	tmpldom "metaldocs/internal/modules/templates_v2/domain"
)

type FreezeFinalizer interface {
	WriteFreeze(ctx context.Context, tenantID, revisionID string, valuesHash []byte, frozenAt time.Time, q ...repository.DBTX) error
}

type SnapshotReader interface {
	ReadSnapshotWithFreezeAt(ctx context.Context, tenantID, revisionID string, q ...repository.DBTX) (v2dom.TemplateSnapshot, *time.Time, error)
}

type FinalDocxWriter interface {
	WriteFinalDocx(ctx context.Context, tenantID, revisionID, s3Key string, contentHash []byte, q ...repository.DBTX) error
}

type FanoutClient interface {
	Fanout(ctx context.Context, req fanout.FanoutRequest) (fanout.FanoutResponse, error)
}

type FreezeService struct {
	schemas    SchemaReader
	values     FillInWriter
	valuesRead interface {
		ListValues(ctx context.Context, tenantID, revisionID string) ([]repository.PlaceholderValue, error)
	}
	resolvers  *resolvers.Registry
	finalize   FreezeFinalizer
	resolveCtx ResolverContextBuilder
	snapshots  SnapshotReader
	finalDocx  FinalDocxWriter
	fanout     FanoutClient
}

type ApproverContext struct {
	UserID       string
	Capabilities []string
}

type ResolverContextBuilder interface {
	Build(ctx context.Context, tenantID, revisionID string, approver ApproverContext) (resolvers.ResolveInput, error)
	BuildForDraft(ctx context.Context, tenantID, revisionID string) (resolvers.ResolveInput, error)
}

func NewFreezeService(
	schemas SchemaReader, values FillInWriter,
	valuesRead interface {
		ListValues(ctx context.Context, tenantID, revisionID string) ([]repository.PlaceholderValue, error)
	},
	reg *resolvers.Registry, final FreezeFinalizer, ctxBuilder ResolverContextBuilder,
	snapshots SnapshotReader, finalDocx FinalDocxWriter,
	fanoutClient FanoutClient,
) *FreezeService {
	return &FreezeService{
		schemas: schemas, values: values, valuesRead: valuesRead,
		resolvers: reg, finalize: final, resolveCtx: ctxBuilder,
		snapshots: snapshots, finalDocx: finalDocx,
		fanout: fanoutClient,
	}
}

func (s *FreezeService) Freeze(ctx context.Context, tx *sql.Tx, tenantID, revisionID string, approver ApproverContext) error {
	var (
		snap           v2dom.TemplateSnapshot
		valuesFrozenAt *time.Time
		err            error
	)
	if tx != nil {
		snap, valuesFrozenAt, err = s.snapshots.ReadSnapshotWithFreezeAt(ctx, tenantID, revisionID, tx)
	} else {
		snap, valuesFrozenAt, err = s.snapshots.ReadSnapshotWithFreezeAt(ctx, tenantID, revisionID)
	}
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}
	if valuesFrozenAt != nil {
		return nil
	}

	schema, err := s.schemas.LoadPlaceholderSchema(ctx, tenantID, revisionID)
	if err != nil {
		return err
	}
	existing, err := s.valuesRead.ListValues(ctx, tenantID, revisionID)
	if err != nil {
		return err
	}
	byID := map[string]repository.PlaceholderValue{}
	for _, v := range existing {
		byID[v.PlaceholderID] = v
	}

	// Validate required are filled for all non-computed
	for _, p := range schema {
		if !p.Required || p.Computed {
			continue
		}
		v, ok := byID[p.ID]
		if !ok || v.ValueText == nil || *v.ValueText == "" {
			return fmt.Errorf("%w: placeholder %s required", v2dom.ErrValidationFailed, p.ID)
		}
	}

	// Resolve computed
	resolveIn, err := s.resolveCtx.Build(ctx, tenantID, revisionID, approver)
	if err != nil {
		return err
	}
	for _, p := range schema {
		if !p.Computed {
			continue
		}
		if p.ResolverKey == nil {
			return fmt.Errorf("%w: placeholder %s computed without resolver_key",
				v2dom.ErrValidationFailed, p.ID)
		}
		r, ok := s.resolvers.Get(*p.ResolverKey)
		if !ok {
			return fmt.Errorf("%w: placeholder %s resolver %s",
				tmpldom.ErrUnknownResolver, p.ID, *p.ResolverKey)
		}
		rv, err := r.Resolve(ctx, resolveIn)
		if err != nil {
			return fmt.Errorf("resolver %s failed: %w", *p.ResolverKey, err)
		}
		strVal := fmt.Sprintf("%v", rv.Value)
		key, ver := *p.ResolverKey, rv.ResolverVer
		// UpsertValue writes via s.db (not the caller-supplied tx). These upserts are
		// intentionally outside the approval transaction: the primary key is
		// (tenant_id, revision_id, placeholder_id) so they are idempotent on retry.
		// If the approval tx rolls back the caller retries and recomputes the same values.
		if err := s.values.UpsertValue(ctx, repository.PlaceholderValue{
			TenantID: tenantID, RevisionID: revisionID, PlaceholderID: p.ID,
			ValueText: &strVal, Source: "computed",
			ComputedFrom: &key, ResolverVersion: &ver,
			InputsHash: rv.InputsHash,
		}); err != nil {
			return err
		}
		byID[p.ID] = repository.PlaceholderValue{ValueText: &strVal}
	}

	// Compute values_hash
	valMap := make(map[string]any, len(byID))
	for _, p := range schema {
		if v, ok := byID[p.ID]; ok && v.ValueText != nil {
			valMap[p.ID] = *v.ValueText
		}
	}
	hashHex := v2dom.ComputeValuesHash(valMap)
	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		return fmt.Errorf("decode values_hash: %w", err)
	}
	if tx != nil {
		if err := s.finalize.WriteFreeze(ctx, tenantID, revisionID, hashBytes, time.Now().UTC(), tx); err != nil {
			return err
		}
	} else {
		if err := s.finalize.WriteFreeze(ctx, tenantID, revisionID, hashBytes, time.Now().UTC()); err != nil {
			return err
		}
	}
	placeholderVals := map[string]string{}
	resolvedForSubblocks := map[string]any{}
	for id, v := range valMap {
		if sv, ok := v.(string); ok {
			placeholderVals[id] = sv
			resolvedForSubblocks[id] = sv
		}
	}

	composition := snap.CompositionJSON
	if len(composition) == 0 {
		composition = json.RawMessage(`{}`)
	}

	resp, err := s.fanout.Fanout(ctx, fanout.FanoutRequest{
		TenantID:          tenantID,
		RevisionID:        revisionID,
		BodyDocxS3Key:     snap.BodyDocxS3Key,
		PlaceholderValues: placeholderVals,
		Composition:       json.RawMessage(composition),
		ResolvedValues:    resolvedForSubblocks,
	})
	if err != nil {
		return fmt.Errorf("fanout: %w", err)
	}

	contentHashBytes, err := hex.DecodeString(resp.ContentHash)
	if err != nil {
		return fmt.Errorf("decode content_hash: %w", err)
	}
	if tx != nil {
		return s.finalDocx.WriteFinalDocx(ctx, tenantID, revisionID, resp.FinalDocxS3Key, contentHashBytes, tx)
	}
	return s.finalDocx.WriteFinalDocx(ctx, tenantID, revisionID, resp.FinalDocxS3Key, contentHashBytes)
}
