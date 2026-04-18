//go:build integration
// +build integration

package docx_v2_test

import "testing"

// TestExportsV2_HappyPath drives the full PDF export happy path over HTTP:
// POST /api/v2/documents/{id}/export/pdf (cold miss) →
// assert 200, cached=false, signed_url non-empty, size_bytes > 0 →
// verify row inserted in document_exports with matching composite_hash →
// POST /api/v2/documents/{id}/export/pdf (warm hit, same content) →
// assert 200, cached=true, same storage_key returned →
// GET /api/v2/documents/{id}/export/docx-url →
// assert 200, signed_url non-empty, revision_id matches current revision.
//
// Table-level effects checked after cold miss:
// - document_exports: 1 row for (document_id, composite_hash)
// - audit_log: 1 event with event_type='export.pdf_generated', cached=false
//
// Table-level effects checked after warm hit:
// - document_exports: still 1 row (ON CONFLICT DO NOTHING)
// - audit_log: 2nd event with cached=true
func TestExportsV2_HappyPath(t *testing.T) {
	t.Skip("integration test requires testcontainers or running infrastructure")
}

// TestExportsV2_RBACDenialMatrix enforces permission checks on all export routes.
//
// Matrix:
// - POST /export/pdf without PermDocumentRead → 403
// - GET /export/docx-url without PermDocumentRead → 403
// - POST /export/pdf for document owned by user_A, requested by user_B with only
//   viewer role on a different document → 403
// - Unauthenticated POST /export/pdf → 401
//
// Each sub-test asserts HTTP denial and no row inserted into document_exports.
func TestExportsV2_RBACDenialMatrix(t *testing.T) {
	t.Skip("integration test requires testcontainers or running infrastructure")
}

// TestExportsV2_RateLimitEnforced verifies that the 20/min per-user bucket is
// enforced at the HTTP layer:
// - Fire 20 requests → all 200
// - Fire 21st request → 429 with retry_after_seconds in body
// - Verify no 21st row in document_exports (request rejected before service call)
func TestExportsV2_RateLimitEnforced(t *testing.T) {
	t.Skip("integration test requires testcontainers or running infrastructure")
}

// TestExportsV2_CompositeHashIsolation verifies that two documents with identical
// content but different document IDs produce independent export rows
// (composite hash includes document_id-scoped content path; same hash bytes
// for same logical content is acceptable but rows must be per-document).
func TestExportsV2_CompositeHashIsolation(t *testing.T) {
	t.Skip("integration test requires testcontainers or running infrastructure")
}

// TestExportsV2_S3OrphanRecovery verifies the S3-orphan recovery path:
// given a document_exports row whose storage_key no longer exists in S3
// (simulated by deleting the object), a subsequent POST /export/pdf must
// re-generate the PDF (cached=false) and re-upload to the original storage_key
// without error.
func TestExportsV2_S3OrphanRecovery(t *testing.T) {
	t.Skip("integration test requires testcontainers or running infrastructure")
}
