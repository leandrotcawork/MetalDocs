package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"metaldocs/internal/modules/templates_v2/application"
	"metaldocs/internal/modules/templates_v2/domain"
)

func TestGetTemplate_Happy(t *testing.T) {
	repo := newFakeRepo()
	tpl := &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	repo.templates[tpl.ID] = tpl

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	got, err := svc.GetTemplate(context.Background(), "tenant-a", tpl.ID)
	if err != nil {
		t.Fatalf("GetTemplate returned error: %v", err)
	}
	if got != tpl {
		t.Fatalf("expected template pointer %p, got %p", tpl, got)
	}
}

func TestGetTemplate_CrossTenant(t *testing.T) {
	repo := newFakeRepo()
	repo.ignoreTenantOnGetTemplate = true
	repo.templates["tpl-1"] = &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.GetTemplate(context.Background(), "tenant-b", "tpl-1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetVersion_Happy(t *testing.T) {
	repo := newFakeRepo()
	tpl := &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	ver := &domain.TemplateVersion{ID: "ver-2", TemplateID: tpl.ID, VersionNumber: 2}
	repo.templates[tpl.ID] = tpl
	repo.versions[ver.ID] = ver

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	got, err := svc.GetVersion(context.Background(), "tenant-a", tpl.ID, 2)
	if err != nil {
		t.Fatalf("GetVersion returned error: %v", err)
	}
	if got != ver {
		t.Fatalf("expected version pointer %p, got %p", ver, got)
	}
}

func TestGetVersion_CrossTenant(t *testing.T) {
	repo := newFakeRepo()
	repo.ignoreTenantOnGetTemplate = true
	tpl := &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	repo.templates[tpl.ID] = tpl
	repo.versions["ver-1"] = &domain.TemplateVersion{ID: "ver-1", TemplateID: tpl.ID, VersionNumber: 1}

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.GetVersion(context.Background(), "tenant-b", tpl.ID, 1)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListTemplates_Happy(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["tpl-1"] = &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	repo.templates["tpl-2"] = &domain.Template{ID: "tpl-2", TenantID: "tenant-b"}

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	got, err := svc.ListTemplates(context.Background(), application.ListFilter{TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("ListTemplates returned error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "tpl-1" {
		t.Fatalf("expected one template tpl-1, got %+v", got)
	}
}

func TestListTemplates_VisibilityFieldsPassThrough(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["tpl-1"] = &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	status := domain.VersionStatusDraft
	docType := "CONTRACT"
	filter := application.ListFilter{
		TenantID:         "tenant-a",
		AreaAny:          []string{"core"},
		DocTypeCode:      &docType,
		Status:           &status,
		Limit:            10,
		Offset:           2,
		ActorAreas:       []string{"legal", "ops"},
		IsExternalViewer: true,
	}

	_, err := svc.ListTemplates(context.Background(), filter)
	if err != nil {
		t.Fatalf("ListTemplates returned error: %v", err)
	}
	if !reflect.DeepEqual(repo.receivedFilter.ActorAreas, filter.ActorAreas) {
		t.Fatalf("expected ActorAreas %v, got %v", filter.ActorAreas, repo.receivedFilter.ActorAreas)
	}
	if repo.receivedFilter.IsExternalViewer != filter.IsExternalViewer {
		t.Fatalf("expected IsExternalViewer %v, got %v", filter.IsExternalViewer, repo.receivedFilter.IsExternalViewer)
	}
}

func TestListAudit_Happy(t *testing.T) {
	repo := newFakeRepo()
	tpl := &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	repo.templates[tpl.ID] = tpl
	repo.audit = append(repo.audit,
		&domain.AuditEvent{TenantID: "tenant-a", TemplateID: tpl.ID, Action: domain.AuditCreated},
		&domain.AuditEvent{TenantID: "tenant-a", TemplateID: "tpl-other", Action: domain.AuditCreated},
	)

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	got, err := svc.ListAudit(context.Background(), "tenant-a", tpl.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListAudit returned error: %v", err)
	}
	if len(got) != 1 || got[0].TemplateID != tpl.ID {
		t.Fatalf("expected one audit event for %q, got %+v", tpl.ID, got)
	}
}

func TestListAudit_CrossTenant(t *testing.T) {
	repo := newFakeRepo()
	repo.ignoreTenantOnGetTemplate = true
	tpl := &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	repo.templates[tpl.ID] = tpl
	repo.audit = append(repo.audit, &domain.AuditEvent{TenantID: "tenant-a", TemplateID: tpl.ID, Action: domain.AuditCreated})

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.ListAudit(context.Background(), "tenant-b", tpl.ID, 10, 0)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
