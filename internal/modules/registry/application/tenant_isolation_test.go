//go:build integration

package application

import "testing"

func TestTenantIsolation_CrossTenantFK_DocumentProfile(t *testing.T) {
	t.Skip("requires live DB")
}

func TestTenantIsolation_CrossTenantFK_DocumentsV2Trigger(t *testing.T) {
	t.Skip("requires live DB")
}

func TestTenantIsolation_ListControlledDocuments_CrossTenant(t *testing.T) {
	t.Skip("requires live DB")
}

func TestTenantIsolation_ListProfiles_CrossTenant(t *testing.T) {
	t.Skip("requires live DB")
}

func TestTenantIsolation_ListMemberships_CrossTenant(t *testing.T) {
	t.Skip("requires live DB")
}

func TestTenantIsolation_CreateCD_CrossTenantProfile_Returns404(t *testing.T) {
	t.Skip("requires live DB")
}
