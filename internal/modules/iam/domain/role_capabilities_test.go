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
