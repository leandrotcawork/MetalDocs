package application

import (
	"bytes"
	"context"
	"fmt"

	"metaldocs/internal/modules/documents_v2/repository"
	"metaldocs/internal/modules/render/resolvers"
	tmpldom "metaldocs/internal/modules/templates_v2/domain"
)

type DraftResolverService struct {
	schemas    SchemaReader
	values     FillInWriter
	valuesRead interface {
		ListValues(ctx context.Context, tenantID, revisionID string) ([]repository.PlaceholderValue, error)
	}
	reg        *resolvers.Registry
	resolveCtx ResolverContextBuilder
}

func NewDraftResolverService(
	schemas SchemaReader,
	values FillInWriter,
	valuesRead interface {
		ListValues(ctx context.Context, tenantID, revisionID string) ([]repository.PlaceholderValue, error)
	},
	reg *resolvers.Registry,
	ctxBuilder ResolverContextBuilder,
) *DraftResolverService {
	return &DraftResolverService{schemas, values, valuesRead, reg, ctxBuilder}
}

// ResolveComputedIfStale resolves all computed placeholders whose inputs_hash
// differs from the stored value. Called on draft load and after any user-placeholder upsert.
func (s *DraftResolverService) ResolveComputedIfStale(ctx context.Context, tenantID, revisionID string) error {
	schema, err := s.schemas.LoadPlaceholderSchema(ctx, tenantID, revisionID)
	if err != nil {
		return err
	}
	existing, err := s.valuesRead.ListValues(ctx, tenantID, revisionID)
	if err != nil {
		return err
	}
	byID := make(map[string]repository.PlaceholderValue, len(existing))
	for _, v := range existing {
		byID[v.PlaceholderID] = v
	}

	rin, err := s.resolveCtx.BuildForDraft(ctx, tenantID, revisionID)
	if err != nil {
		return err
	}

	for _, p := range schema {
		if !p.Computed || p.ResolverKey == nil {
			continue
		}
		r, ok := s.reg.Get(*p.ResolverKey)
		if !ok {
			return fmt.Errorf("%w: %s", tmpldom.ErrUnknownResolver, *p.ResolverKey)
		}
		rv, err := r.Resolve(ctx, rin)
		if err != nil {
			return err
		}
		if cur, ok := byID[p.ID]; ok && bytes.Equal(cur.InputsHash, rv.InputsHash) {
			continue // cache hit — skip write
		}
		strVal := fmt.Sprintf("%v", rv.Value)
		key, ver := *p.ResolverKey, rv.ResolverVer
		if err := s.values.UpsertValue(ctx, repository.PlaceholderValue{
			TenantID:        tenantID,
			RevisionID:      revisionID,
			PlaceholderID:   p.ID,
			ValueText:       &strVal,
			Source:          "computed",
			ComputedFrom:    &key,
			ResolverVersion: &ver,
			InputsHash:      rv.InputsHash,
		}); err != nil {
			return err
		}
	}
	return nil
}
