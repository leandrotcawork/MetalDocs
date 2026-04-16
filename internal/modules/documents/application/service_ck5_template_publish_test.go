package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"metaldocs/internal/modules/documents/domain"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
)

func TestPublishTemplateForReview_OK(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)

	draft := &domain.TemplateDraft{
		TemplateKey: "tpl-publish-ok",
		ProfileCode: "po",
		Name:        "Template",
		DraftStatus: domain.TemplateStatusDraft,
		BlocksJSON:  json.RawMessage(`{"_ck5":{"contentHtml":"<p>hello</p>"}}`),
	}
	if _, err := repo.UpsertTemplateDraftCAS(ctx, draft, 0); err != nil {
		t.Fatalf("UpsertTemplateDraftCAS() error = %v", err)
	}

	if err := service.PublishTemplateForReview(ctx, draft.TemplateKey); err != nil {
		t.Fatalf("PublishTemplateForReview() error = %v", err)
	}

	got, err := repo.GetTemplateDraft(ctx, draft.TemplateKey)
	if err != nil {
		t.Fatalf("GetTemplateDraft() error = %v", err)
	}
	if got.DraftStatus != domain.TemplateStatusPendingReview {
		t.Fatalf("DraftStatus = %q, want %q", got.DraftStatus, domain.TemplateStatusPendingReview)
	}
}

func TestPublishTemplateForReview_EmptyHtml(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)

	draft := &domain.TemplateDraft{
		TemplateKey: "tpl-publish-empty",
		ProfileCode: "po",
		Name:        "Template",
		DraftStatus: domain.TemplateStatusDraft,
		BlocksJSON:  json.RawMessage(`{"_ck5":{"contentHtml":"   "}}`),
	}
	if _, err := repo.UpsertTemplateDraftCAS(ctx, draft, 0); err != nil {
		t.Fatalf("UpsertTemplateDraftCAS() error = %v", err)
	}

	err := service.PublishTemplateForReview(ctx, draft.TemplateKey)
	if !errors.Is(err, domain.ErrEmptyTemplateContent) {
		t.Fatalf("err = %v, want ErrEmptyTemplateContent", err)
	}
}

func TestPublishTemplateForReview_WrongStatus(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)

	draft := &domain.TemplateDraft{
		TemplateKey: "tpl-publish-wrong-status",
		ProfileCode: "po",
		Name:        "Template",
		DraftStatus: domain.TemplateStatusPendingReview,
		BlocksJSON:  json.RawMessage(`{"_ck5":{"contentHtml":"<p>hello</p>"}}`),
	}
	if _, err := repo.UpsertTemplateDraftCAS(ctx, draft, 0); err != nil {
		t.Fatalf("UpsertTemplateDraftCAS() error = %v", err)
	}

	err := service.PublishTemplateForReview(ctx, draft.TemplateKey)
	if !errors.Is(err, domain.ErrInvalidTemplateDraftStatus) {
		t.Fatalf("err = %v, want ErrInvalidTemplateDraftStatus", err)
	}
}

func TestApproveTemplate_OK(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)

	draft := &domain.TemplateDraft{
		TemplateKey: "tpl-approve-ok",
		ProfileCode: "po",
		Name:        "Template",
		DraftStatus: domain.TemplateStatusPendingReview,
		BlocksJSON:  json.RawMessage(`{"_ck5":{"contentHtml":"<p>approved</p>"}}`),
	}
	if _, err := repo.UpsertTemplateDraftCAS(ctx, draft, 0); err != nil {
		t.Fatalf("UpsertTemplateDraftCAS() error = %v", err)
	}

	if err := service.ApproveTemplate(ctx, draft.TemplateKey); err != nil {
		t.Fatalf("ApproveTemplate() error = %v", err)
	}

	got, err := repo.GetTemplateDraft(ctx, draft.TemplateKey)
	if err != nil {
		t.Fatalf("GetTemplateDraft() error = %v", err)
	}
	if got.DraftStatus != domain.TemplateStatusPublished {
		t.Fatalf("DraftStatus = %q, want %q", got.DraftStatus, domain.TemplateStatusPublished)
	}
	if got.PublishedHTML == nil {
		t.Fatal("PublishedHTML is nil, want non-nil")
	}
}

func TestApproveTemplate_WrongStatus(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)

	draft := &domain.TemplateDraft{
		TemplateKey: "tpl-approve-wrong-status",
		ProfileCode: "po",
		Name:        "Template",
		DraftStatus: domain.TemplateStatusDraft,
		BlocksJSON:  json.RawMessage(`{"_ck5":{"contentHtml":"<p>hello</p>"}}`),
	}
	if _, err := repo.UpsertTemplateDraftCAS(ctx, draft, 0); err != nil {
		t.Fatalf("UpsertTemplateDraftCAS() error = %v", err)
	}

	err := service.ApproveTemplate(ctx, draft.TemplateKey)
	if !errors.Is(err, domain.ErrInvalidTemplateDraftStatus) {
		t.Fatalf("err = %v, want ErrInvalidTemplateDraftStatus", err)
	}
}

func TestApproveTemplate_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)

	err := service.ApproveTemplate(ctx, "missing-template")
	if !errors.Is(err, domain.ErrTemplateNotFound) {
		t.Fatalf("err = %v, want ErrTemplateNotFound", err)
	}
}
