package resolvers

import (
	"bytes"
	"context"
	"testing"
	"time"
)

// resolverContractCase bundles a resolver with the minimal ResolveInput it needs.
type resolverContractCase struct {
	name     string
	resolver ComputedResolver
	input    ResolveInput
}

func contractCases() []resolverContractCase {
	baseInput := ResolveInput{
		TenantID:             "tenant-contract",
		RevisionID:           "rev-contract",
		ControlledDocumentID: "ctrl-contract",
		AreaCodeSnapshot:     "CONTRACT",
		RevisionReader: fakeRevisionReader{
			revisionNumber: 3,
			effectiveFrom:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			author:         AuthorInfo{UserID: "u-contract", DisplayName: "Contract User"},
		},
		RegistryReader: fakeRegistryReader{
			record: ControlledDocumentInfo{DocCode: "CONTRACT-001"},
		},
		WorkflowReader: fakeWorkflowReader{
			approvers: []ApproverInfo{
				{UserID: "approver-1", DisplayName: "Approver One", SignedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)},
			},
			finalApprovalDate: time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC),
		},
	}
	return []resolverContractCase{
		{"doc_code", DocCodeResolver{}, baseInput},
		{"revision_number", RevisionNumberResolver{}, baseInput},
		{"effective_date", EffectiveDateResolver{}, baseInput},
		{"author", AuthorResolver{}, baseInput},
		{"approvers", ApproversResolver{}, baseInput},
		{"approval_date", ApprovalDateResolver{}, baseInput},
		{"controlled_by_area", ControlledByAreaResolver{}, baseInput},
	}
}

func TestResolverContracts(t *testing.T) {
	for _, tc := range contractCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			v1, err := tc.resolver.Resolve(ctx, tc.input)
			if err != nil {
				t.Fatalf("Resolve #1: %v", err)
			}
			v2, err := tc.resolver.Resolve(ctx, tc.input)
			if err != nil {
				t.Fatalf("Resolve #2: %v", err)
			}

			// InputsHash must be stable across calls with identical inputs.
			if !bytes.Equal(v1.InputsHash, v2.InputsHash) {
				t.Errorf("InputsHash not stable: %x vs %x", v1.InputsHash, v2.InputsHash)
			}

			// ResolverKey must match the registered key.
			if v1.ResolverKey != tc.resolver.Key() {
				t.Errorf("ResolverKey=%q, want %q", v1.ResolverKey, tc.resolver.Key())
			}

			// ResolverVer must be positive.
			if v1.ResolverVer <= 0 {
				t.Errorf("ResolverVer=%d, want >0", v1.ResolverVer)
			}

			// Version must match.
			if v1.ResolverVer != tc.resolver.Version() {
				t.Errorf("ResolverVer=%d, want %d", v1.ResolverVer, tc.resolver.Version())
			}

			// Value must be non-nil.
			if v1.Value == nil {
				t.Errorf("Value is nil")
			}
		})
	}
}
