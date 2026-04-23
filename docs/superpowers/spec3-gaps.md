# Spec 3 Non-Blocking Followups

Tracked out-of-band items found during Phase reviews. Not blockers; address opportunistically.

## From Phase 1 review
- `Placeholder.Computed *PHComputed` vs `Type == PHComputed` redundancy — consider single source of truth.
- Add godoc comments on exported Placeholder fields (Regex, MinNumber, etc.).
- Wire-level test for `omitempty` behavior on new optional fields.

## From Phase 4 review
- **TX boundary (4.4)**: `SnapshotFromTemplate` runs after `repo.CreateDocument` commits. Failure leaves orphan document with NULL snapshot columns + leaked S3 tmp blob (cleanupKey already neutralized). Fix via threading `*sql.Tx` through `Repository.CreateDocument` OR add reconciliation/NOT NULL + rollback in Phase 5. Decide at Phase 5 entry.
- **Partial-failure (4.5)**: `WriteSnapshot` + `SeedDefaults` not atomic with each other; no retry path. Idempotent seeder makes this recoverable on manual retry. Consider wrapping both in outer tx in Phase 5.
- **Duplicated snapshot call site**: `if s.snapshotSvc != nil` block copy-pasted in docgen/no-docgen branches of `Service.CreateDocument`. Extract helper.
- **Double required-filter**: `parseRequiredPlaceholders` filters Required; `FillInRepository.SeedDefaults` also filters `!p.Required`. Redundant. Keep filter in service layer, drop repo-side check.

## From Phase 3 review
- `ValidatePlaceholders` accreting responsibilities (~140 lines). Consider extracting `validateConstraints(p)` per-placeholder helper before Phase 4 adds more. Flag at Phase 4 entry.
- Date comparison uses lexicographic string `>` — correct only for ISO-8601 `2006-01-02`. Add comment at compare site noting the invariant.
- `ResolverRegistryReader.Known() map[string]int` exposes internal map and allows mutation. Tighten to `Has(key string) bool` or `Lookup(key) (int, bool)`. Caller only uses `_, ok`.
- Consider rename `ErrPlaceholderCycle` → `ErrVisibilityCycle` for symmetry with `DetectVisibilityCycle`. Either acceptable.
- Variadic `New(..., resolvers ...ResolverRegistryReader)` asymmetric with required deps. Revisit with functional options if a second optional dep appears.
