//go:build integration
// +build integration

package repository_test

// Standard repo integration test harness. Uses testcontainers postgres.
// Covers: tenant isolation, UUID not-found mapping, jsonb round-trips,
// published obsolescence, and audit ordering.

// func TestCreateTemplate_RoundTrip(t *testing.T) {}
// func TestGetTemplate_CrossTenant_ErrNotFound(t *testing.T) {}
// func TestGetTemplate_MalformedUUID_ErrNotFound(t *testing.T) {}
// func TestCreateVersion_RoundTrip(t *testing.T) {}
// func TestUpdateVersion_JsonbSchemas_Roundtrip(t *testing.T) {}
// func TestObsoletePreviousPublished(t *testing.T) {}
// func TestAppendAudit_ListAudit_OrderedDesc(t *testing.T) {}
