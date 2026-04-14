package application

import (
	"testing"

	"metaldocs/internal/modules/iam/domain"
)

func TestStaticAuthorizerAdminCanAllTemplatePermissions(t *testing.T) {
	a := NewStaticAuthorizer()

	perms := []domain.Permission{
		domain.PermTemplateView,
		domain.PermTemplateEdit,
		domain.PermTemplatePublish,
		domain.PermTemplateExport,
	}

	for _, perm := range perms {
		if !a.Can(domain.RoleAdmin, perm) {
			t.Errorf("expected admin to have permission %q, but got false", perm)
		}
	}
}

func TestStaticAuthorizerEditorCanViewAndExportTemplates(t *testing.T) {
	a := NewStaticAuthorizer()

	if !a.Can(domain.RoleEditor, domain.PermTemplateView) {
		t.Errorf("expected editor to have permission %q, but got false", domain.PermTemplateView)
	}
	if !a.Can(domain.RoleEditor, domain.PermTemplateExport) {
		t.Errorf("expected editor to have permission %q, but got false", domain.PermTemplateExport)
	}
}

func TestStaticAuthorizerEditorCannotEditOrPublishTemplates(t *testing.T) {
	a := NewStaticAuthorizer()

	if a.Can(domain.RoleEditor, domain.PermTemplateEdit) {
		t.Errorf("expected editor NOT to have permission %q, but got true", domain.PermTemplateEdit)
	}
	if a.Can(domain.RoleEditor, domain.PermTemplatePublish) {
		t.Errorf("expected editor NOT to have permission %q, but got true", domain.PermTemplatePublish)
	}
}

func TestStaticAuthorizerReviewerCanViewAndExportTemplates(t *testing.T) {
	a := NewStaticAuthorizer()

	if !a.Can(domain.RoleReviewer, domain.PermTemplateView) {
		t.Errorf("expected reviewer to have permission %q, but got false", domain.PermTemplateView)
	}
	if !a.Can(domain.RoleReviewer, domain.PermTemplateExport) {
		t.Errorf("expected reviewer to have permission %q, but got false", domain.PermTemplateExport)
	}
}

func TestStaticAuthorizerReviewerCannotEditOrPublishTemplates(t *testing.T) {
	a := NewStaticAuthorizer()

	if a.Can(domain.RoleReviewer, domain.PermTemplateEdit) {
		t.Errorf("expected reviewer NOT to have permission %q, but got true", domain.PermTemplateEdit)
	}
	if a.Can(domain.RoleReviewer, domain.PermTemplatePublish) {
		t.Errorf("expected reviewer NOT to have permission %q, but got true", domain.PermTemplatePublish)
	}
}

func TestStaticAuthorizerViewerCannotAccessTemplates(t *testing.T) {
	a := NewStaticAuthorizer()

	if a.Can(domain.RoleViewer, domain.PermTemplateView) {
		t.Errorf("expected viewer NOT to have permission %q, but got true", domain.PermTemplateView)
	}
	if a.Can(domain.RoleViewer, domain.PermTemplateEdit) {
		t.Errorf("expected viewer NOT to have permission %q, but got true", domain.PermTemplateEdit)
	}
	if a.Can(domain.RoleViewer, domain.PermTemplatePublish) {
		t.Errorf("expected viewer NOT to have permission %q, but got true", domain.PermTemplatePublish)
	}
	if a.Can(domain.RoleViewer, domain.PermTemplateExport) {
		t.Errorf("expected viewer NOT to have permission %q, but got true", domain.PermTemplateExport)
	}
}
