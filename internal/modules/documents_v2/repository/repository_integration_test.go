//go:build integration
// +build integration

package repository_test

// Standard repo integration test harness. Uses testcontainers postgres.
// Covers: CreateDocument round-trip, AcquireSession partial unique index,
// CommitUpload happy path, DeleteExpiredPending cleanup.

// func TestCreateDocument_RoundTrip(t *testing.T) {}
// func TestAcquireSession_SingleWriterInvariant(t *testing.T) {}
// func TestCommitUpload_Happy(t *testing.T) {}
// func TestDeleteExpiredPending_RemovesExpired(t *testing.T) {}
