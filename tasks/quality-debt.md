# tasks/quality-debt.md
# Known quality issues that don't block delivery but should be addressed.
# Updated by $metaldocs-review when verdict is COMPLIANT BUT WEAK.

---

## QD-001 — Some domain validation lives in service layer
Pattern: field validation in application/service.go instead of domain model
Target: domain.Entity.Validate() called by service
Priority: refactor as each module is touched

## QD-002 — Events with sparse payload
Pattern: some events published with only aggregate ID
Target: self-contained payload for primary consumer
Priority: review each event contract before next worker feature

## QD-003 — Happy-path-only tests
Pattern: tests cover success cases, not error/invariant/permission cases
Target: each test covers valid + invalid + boundary cases
Priority: add error cases alongside any new test
