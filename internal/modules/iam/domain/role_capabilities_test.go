package domain

import "testing"

func TestRoleCapabilities_AllRolesHaveEntries(t *testing.T) {
	for role, caps := range RoleCapabilities {
		if len(caps) == 0 {
			t.Fatalf("expected role %q to have capabilities", role)
		}
	}
}

func TestRoleCapabilities_VersionIsPositive(t *testing.T) {
	if RoleCapabilitiesVersion <= 0 {
		t.Fatalf("expected positive RoleCapabilitiesVersion, got %d", RoleCapabilitiesVersion)
	}
}

func TestRoleCapabilities_RoleEditorExactSet(t *testing.T) {
	expected := map[Capability]bool{
		CapDocumentView:   true,
		CapDocumentCreate: true,
		CapDocumentEdit:   true,
		CapTemplateView:   true,
	}

	editorCaps := RoleCapabilities[RoleEditor]
	if len(editorCaps) != len(expected) {
		t.Fatalf("expected %d editor capabilities, got %d", len(expected), len(editorCaps))
	}

	for _, cap := range editorCaps {
		if !expected[cap] {
			t.Fatalf("unexpected editor capability: %s", cap)
		}
	}
}
