package domain_test

import (
	"errors"
	"testing"

	"metaldocs/internal/modules/templates_v2/domain"
)

func TestCheckSegregation(t *testing.T) {
	reviewerID := "reviewer-1"

	tests := []struct {
		name       string
		role       string
		actorID    string
		authorID   string
		reviewerID *string
		wantErr    error
	}{
		{
			name:     "reviewer OK - distinct from author",
			role:     "reviewer",
			actorID:  "reviewer-1",
			authorID: "author-1",
		},
		{
			name:     "reviewer FAIL - same as author",
			role:     "reviewer",
			actorID:  "author-1",
			authorID: "author-1",
			wantErr:  domain.ErrISOSegregationViolation,
		},
		{
			name:       "approver OK - distinct from both",
			role:       "approver",
			actorID:    "approver-1",
			authorID:   "author-1",
			reviewerID: &reviewerID,
		},
		{
			name:       "approver FAIL - same as author",
			role:       "approver",
			actorID:    "author-1",
			authorID:   "author-1",
			reviewerID: &reviewerID,
			wantErr:    domain.ErrISOSegregationViolation,
		},
		{
			name:       "approver FAIL - same as reviewer",
			role:       "approver",
			actorID:    "reviewer-1",
			authorID:   "author-1",
			reviewerID: &reviewerID,
			wantErr:    domain.ErrISOSegregationViolation,
		},
		{
			name:     "approver OK - reviewerID nil, distinct from author",
			role:     "approver",
			actorID:  "approver-1",
			authorID: "author-1",
		},
		{
			name:     "unknown role returns ErrForbiddenRole",
			role:     "admin",
			actorID:  "actor-1",
			authorID: "author-1",
			wantErr:  domain.ErrForbiddenRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.CheckSegregation(tt.role, tt.actorID, tt.authorID, tt.reviewerID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CheckSegregation(%q, actor=%q, author=%q, reviewer=%v) error = %v, want %v", tt.role, tt.actorID, tt.authorID, tt.reviewerID, err, tt.wantErr)
			}
		})
	}
}
