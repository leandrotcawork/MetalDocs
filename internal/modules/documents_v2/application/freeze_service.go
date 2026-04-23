package application

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	v2dom "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/documents_v2/repository"
	"metaldocs/internal/modules/render/resolvers"
	tmpldom "metaldocs/internal/modules/templates_v2/domain"
)

type FreezeFinalizer interface {
	WriteFreeze(ctx context.Context, tenantID, revisionID string, valuesHash []byte, frozenAt time.Time) error
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
}

type ResolverContextBuilder interface {
	Build(ctx context.Context, tenantID, revisionID string) (resolvers.ResolveInput, error)
}

func NewFreezeService(
	schemas SchemaReader, values FillInWriter,
	valuesRead interface {
		ListValues(ctx context.Context, tenantID, revisionID string) ([]repository.PlaceholderValue, error)
	},
	reg *resolvers.Registry, final FreezeFinalizer, ctxBuilder ResolverContextBuilder,
) *FreezeService {
	return &FreezeService{schemas, values, valuesRead, reg, final, ctxBuilder}
}

func (s *FreezeService) Freeze(ctx context.Context, tenantID, revisionID string) error {
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
	resolveIn, err := s.resolveCtx.Build(ctx, tenantID, revisionID)
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
	hashBytes, _ := hex.DecodeString(hashHex)
	return s.finalize.WriteFreeze(ctx, tenantID, revisionID, hashBytes, time.Now().UTC())
}
