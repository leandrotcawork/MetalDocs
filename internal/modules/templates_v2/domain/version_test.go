package domain_test

import (
	"errors"
	"testing"

	"metaldocs/internal/modules/templates_v2/domain"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name        string
		from        domain.VersionStatus
		next        domain.VersionStatus
		hasReviewer bool
		wantErr     error
	}{
		{
			name:        "draft to in_review",
			from:        domain.VersionStatusDraft,
			next:        domain.VersionStatusInReview,
			hasReviewer: true,
		},
		{
			name:        "in_review to approved when reviewer required",
			from:        domain.VersionStatusInReview,
			next:        domain.VersionStatusApproved,
			hasReviewer: true,
		},
		{
			name:        "in_review to published when reviewer not required",
			from:        domain.VersionStatusInReview,
			next:        domain.VersionStatusPublished,
			hasReviewer: false,
		},
		{
			name:        "approved to published",
			from:        domain.VersionStatusApproved,
			next:        domain.VersionStatusPublished,
			hasReviewer: true,
		},
		{
			name:        "published to obsolete",
			from:        domain.VersionStatusPublished,
			next:        domain.VersionStatusObsolete,
			hasReviewer: true,
		},
		{
			name:        "in_review to draft reject",
			from:        domain.VersionStatusInReview,
			next:        domain.VersionStatusDraft,
			hasReviewer: true,
		},
		{
			name:        "approved to draft reject",
			from:        domain.VersionStatusApproved,
			next:        domain.VersionStatusDraft,
			hasReviewer: true,
		},
		{
			name:        "in_review to approved denied when reviewer not required",
			from:        domain.VersionStatusInReview,
			next:        domain.VersionStatusApproved,
			hasReviewer: false,
			wantErr:     domain.ErrInvalidStateTransition,
		},
		{
			name:        "in_review to published denied when reviewer required",
			from:        domain.VersionStatusInReview,
			next:        domain.VersionStatusPublished,
			hasReviewer: true,
			wantErr:     domain.ErrInvalidStateTransition,
		},
		{
			name:        "draft to published invalid",
			from:        domain.VersionStatusDraft,
			next:        domain.VersionStatusPublished,
			hasReviewer: true,
			wantErr:     domain.ErrInvalidStateTransition,
		},
		{
			name:        "draft to approved invalid",
			from:        domain.VersionStatusDraft,
			next:        domain.VersionStatusApproved,
			hasReviewer: true,
			wantErr:     domain.ErrInvalidStateTransition,
		},
		{
			name:        "draft to obsolete invalid",
			from:        domain.VersionStatusDraft,
			next:        domain.VersionStatusObsolete,
			hasReviewer: true,
			wantErr:     domain.ErrInvalidStateTransition,
		},
		{
			name:        "approved to obsolete invalid",
			from:        domain.VersionStatusApproved,
			next:        domain.VersionStatusObsolete,
			hasReviewer: true,
			wantErr:     domain.ErrInvalidStateTransition,
		},
		{
			name:        "published to draft invalid",
			from:        domain.VersionStatusPublished,
			next:        domain.VersionStatusDraft,
			hasReviewer: true,
			wantErr:     domain.ErrInvalidStateTransition,
		},
		{
			name:        "obsolete to draft invalid",
			from:        domain.VersionStatusObsolete,
			next:        domain.VersionStatusDraft,
			hasReviewer: true,
			wantErr:     domain.ErrInvalidStateTransition,
		},
		{
			name:        "obsolete to published invalid",
			from:        domain.VersionStatusObsolete,
			next:        domain.VersionStatusPublished,
			hasReviewer: true,
			wantErr:     domain.ErrInvalidStateTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := domain.TemplateVersion{Status: tt.from}

			err := v.CanTransition(tt.next, tt.hasReviewer)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CanTransition(%q -> %q, hasReviewer=%v) error = %v, want %v", tt.from, tt.next, tt.hasReviewer, err, tt.wantErr)
			}
		})
	}
}
