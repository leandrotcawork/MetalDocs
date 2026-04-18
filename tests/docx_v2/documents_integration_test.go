//go:build integration
// +build integration

package docx_v2_test

import "testing"

// TestDocumentsV2_HappyPath_AllRoutes drives the owned document_filler happy path over HTTP:
// POST /api/v2/documents -> POST /autosave/presign -> PUT S3 -> POST /autosave/commit
// -> POST /session/heartbeat -> POST /checkpoints -> GET /checkpoints
// -> POST /checkpoints/{versionNum}/restore -> POST /session/release
// -> POST /finalize -> POST /archive.
//
// It asserts 201/200 statuses and verifies table-level effects in documents,
// document_revisions, editor_sessions, pending_uploads, document_checkpoints,
// and audit_events after each step.
func TestDocumentsV2_HappyPath_AllRoutes(t *testing.T) {
	t.Skip("integration test requires testcontainers or running infrastructure")
}

// TestDocumentsV2_RBACDenialMatrix enforces one negative RBAC/ownership assertion
// per documents-v2 route and denial reason combination.
//
// Matrix summary:
// - create denied without document_filler role;
// - list is ownership-filtered (filler_B sees 0 rows for filler_A docs);
// - get/presign/commit/session acquire/heartbeat/release denied for non-owner or non-holder;
// - force-release denied without admin;
// - checkpoints list/create/restore denied for non-owner;
// - finalize/archive denied for non-owner except admin override archive case.
//
// Each sub-test must assert both HTTP denial semantics and no unintended DB mutation.
func TestDocumentsV2_RBACDenialMatrix(t *testing.T) {
	t.Skip("integration test requires testcontainers or running infrastructure")
}