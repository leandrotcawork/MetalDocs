package application_test

import (
	"context"
	"errors"
	"testing"

	"metaldocs/internal/modules/templates/application"
	"metaldocs/internal/modules/templates/domain"
)

type fakeDocgen struct {
	valid bool
	errs  []byte
}

func (f *fakeDocgen) ValidateTemplate(_ context.Context, _, _ string) (bool, []byte, error) {
	return f.valid, f.errs, nil
}

func TestPublish_RejectedByValidator(t *testing.T) {
	repo := newFakeRepo()
	svc := application.New(repo, &fakeDocgen{valid: false, errs: []byte(`{"parse_errors":[{"type":"unsupported_construct","element":"w:ins"}]}`)}, nil)
	_, ver, _ := svc.CreateTemplate(context.Background(), application.CreateTemplateCmd{
		TenantID: "t1", Key: "po", Name: "N", CreatedBy: "u1",
	})
	_, err := svc.PublishVersion(context.Background(), application.PublishCmd{
		VersionID: ver.ID, ActorUserID: "u1", DocxKey: "d", SchemaKey: "s",
	})
	var ve application.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestPublish_OK_CreatesNextDraft(t *testing.T) {
	repo := newFakeRepo()
	svc := application.New(repo, &fakeDocgen{valid: true}, nil)
	_, ver, _ := svc.CreateTemplate(context.Background(), application.CreateTemplateCmd{
		TenantID: "t1", Key: "po", Name: "N", CreatedBy: "u1",
	})
	res, err := svc.PublishVersion(context.Background(), application.PublishCmd{
		VersionID: ver.ID, ActorUserID: "u1", DocxKey: "d", SchemaKey: "s",
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if repo.versions[ver.ID].Status != domain.StatusPublished {
		t.Fatal("expected published")
	}
	if res.NewDraftVersion != ver.VersionNum+1 {
		t.Fatalf("next draft num: got %d want %d", res.NewDraftVersion, ver.VersionNum+1)
	}
	if res.NewDraftID == "" {
		t.Fatal("expected new draft id")
	}
	if _, ok := repo.versions[res.NewDraftID]; !ok {
		t.Fatal("next draft not persisted in fake")
	}
}
