package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeTemplateRepo struct {
	templates map[string]templateRow
}

func (f *fakeTemplateRepo) Get(ctx context.Context, templateID uuid.UUID, version int) (*templateRow, error) {
	row, ok := f.templates[templateID.String()]
	if !ok {
		return nil, errors.New("not found")
	}
	return &row, nil
}

func TestTemplateService_VerifyHash_Match(t *testing.T) {
	content := json.RawMessage(`{"mddm_version":1,"blocks":[],"template_ref":null}`)
	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:])

	templateID := uuid.New()
	repo := &fakeTemplateRepo{
		templates: map[string]templateRow{
			templateID.String(): {
				ID:            uuid.New(),
				TemplateID:    templateID,
				Version:       1,
				MDDMVersion:   1,
				ContentBlocks: content,
				ContentHash:   hash,
				IsPublished:   true,
			},
		},
	}

	svc := NewTemplateService(repo)
	ref := TemplateRef{TemplateID: templateID, TemplateVersion: 1, TemplateMDDMVersion: 1, TemplateContentHash: hash}
	_, err := svc.LoadAndVerify(context.Background(), ref)
	if err != nil {
		t.Errorf("expected hash match, got %v", err)
	}
}

func TestTemplateService_VerifyHash_Mismatch(t *testing.T) {
	content := json.RawMessage(`{"mddm_version":1,"blocks":[],"template_ref":null}`)
	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:])

	templateID := uuid.New()
	repo := &fakeTemplateRepo{
		templates: map[string]templateRow{
			templateID.String(): {
				ID:            uuid.New(),
				TemplateID:    templateID,
				Version:       1,
				MDDMVersion:   1,
				ContentBlocks: content,
				ContentHash:   hash,
				IsPublished:   true,
			},
		},
	}

	svc := NewTemplateService(repo)
	ref := TemplateRef{TemplateID: templateID, TemplateVersion: 1, TemplateMDDMVersion: 1, TemplateContentHash: "wronghash"}
	_, err := svc.LoadAndVerify(context.Background(), ref)
	if err == nil {
		t.Error("expected TEMPLATE_SNAPSHOT_MISMATCH error")
	}
}
