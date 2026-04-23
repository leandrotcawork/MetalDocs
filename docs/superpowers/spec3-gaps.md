# Spec 3 Non-Blocking Followups

Tracked out-of-band items found during Phase reviews. Not blockers; address opportunistically.

## Phase 8 followups
- **XML escaping gap for user-provided text interpolation**: all five v1 sub-block renderers (`apps/docgen-v2/src/render/subblocks/doc_header_standard.ts`, `revision_box.ts`, `approval_signatures_block.ts`, `footer_controlled_copy_notice.ts`) concatenate values from `ctx.values` / `ctx.params` directly into OOXML strings via `${text}` inside `<w:t>…</w:t>`. Inputs such as `title`, `doc_code`, `display_name`, `description`, and `notice_text` can legally contain `<`, `>`, `&`, `"`, or the sequence `]]>` and will produce malformed XML or enable attribute/element injection at downstream assembly. The Phase 8 plan is silent on escaping at this layer (freeze/fanout phases handle final XML assembly). Fix: add a small `escapeXmlText(s string)` helper at `apps/docgen-v2/src/render/subblocks/xml.ts` that replaces `&` → `&amp;`, `<` → `&lt;`, `>` → `&gt;` (and `"` → `&quot;` for attribute contexts if later needed), then wire it through the shared `cell(text)` helpers in each sub-block. Add one unit test per sub-block asserting that a value containing `<` round-trips as `&lt;`. Consider lifting `str()` + `cell()` into that shared module while there (currently duplicated across `doc_header_standard.ts`, `revision_box.ts`, `approval_signatures_block.ts`).
- **Duplicated `str()` + `cell()` + `row()` helpers across sub-blocks**: `doc_header_standard.ts`, `revision_box.ts`, and `approval_signatures_block.ts` each redefine the same `str`/`cell`/`row` helpers. Low-cost DRY: extract into `apps/docgen-v2/src/render/subblocks/xml.ts` alongside the escape helper above. Defer until a fourth consumer (freeze/fanout) needs the same primitives to avoid premature abstraction.
- **`FooterPageNumbers` hardcodes fallback text `1` / `1`**: the `<w:t>1</w:t>` inside the `<w:fldSimple>` elements is the static fallback rendered by Word if it cannot evaluate the field. Acceptable per plan example verbatim, but worth verifying against final Word rendering in Phase 11 DOCX acceptance that the field actually updates (not that `1` is shown literally).
- **`approvers` / `revision_history` shape not validated**: sub-blocks accept any object in the arrays and silently produce empty cells for missing properties. Phase 9/10 freeze should validate shape at the snapshot boundary; sub-blocks should not grow defensive branching. Leave as-is; note for freeze contract.

## From Phase 7 review
- **`InputsHash` does not mix ResolverKey/Version**: `revision_number`, `effective_date`, `author`, `approvers`, `approval_date` all hash `{tenant_id, revision_id}` with identical bytes. If a consumer dedupes on `InputsHash` alone, it will collide across resolvers. Safer: include `resolver_key` + `resolver_version` in the hashed struct (or rely on the caller storing `(ResolverKey, ResolverVer, InputsHash)` as the composite cache key). Defer to Phase 9 freeze service contract — if freeze already stores the tuple, no fix needed.
- **Shared `hashInputs` helper**: plan Phase 7 intro said "copy per resolver; no reuse". Codex extracted `hash.go`. Minor spec deviation, DRY win. Accepted.
- **`TaxonomyReader` port is empty**: `type TaxonomyReader interface{}` defined in `resolver.go` but no v1 resolver uses it. Either delete (YAGNI) or leave as placeholder for Phase 5b / Phase 7b future resolvers. Tracking here so it isn't forgotten.
- **Tests don't assert `ResolverKey` / `ResolverVer` on `ResolvedValue`**: current tests only check `Value` and `InputsHash`. Add one-line assertions per resolver test to catch accidental typos in the literal strings.
- **Error-path not tested**: no test exercises reader returning an error (all fakes return `err: nil`). Low priority — the wiring is `if err != nil { return ..., err }` and trivially correct.
- **Zero-time date formatting**: `EffectiveDateResolver` and `ApprovalDateResolver` emit `"0001-01-01"` when reader returns `time.Time{}`. Phase 5b may resolve these while still unset; decide there whether to short-circuit or emit empty string.
- **Plan text expected `registry.Reader` / `v2docs.RevisionReader` / `workflow.Reader` / `taxonomy.Reader` imports**: overridden by Opus in Phase 7 brief to consumer-defined local ports. Document this deviation so Phase 9/10 wiring knows to build adapters in the wiring layer, not in the resolvers package.

## From Phase 6 review
- **Stale spike test** (`src/editor-adapters/__spike__/eigenpal-zone-spike.test.ts`): intentionally RED, calls old `wrapZone("observations", [...])` two-arg signature. Header comment says "kept RED for documentation only". Decide: delete now that Phase 6 regression test covers the current API, or convert to GREEN against the bookmark-pair signature. Low priority.
- **No pnpm-workspace.yaml**: monorepo uses `file:` deps across `frontend/apps/web`, `packages/editor-ui`, `packages/shared-tokens`. Works but fragile (any new `"*"` specifier in a package.json breaks fresh install). Consider adding `pnpm-workspace.yaml` + standardizing on `workspace:*` for true workspace semantics.

## From Phase 1 review
- `Placeholder.Computed *PHComputed` vs `Type == PHComputed` redundancy — consider single source of truth.
- Add godoc comments on exported Placeholder fields (Regex, MinNumber, etc.).
- Wire-level test for `omitempty` behavior on new optional fields.

## From Phase 5 review
- **Authz coverage**: `requireDocEditDraft` wired in `FillInService.SetPlaceholderValue`/`SetZoneContent`. No integration test yet proves `authz.ErrCapabilityDenied` returns for user without `doc.edit_draft`. Add integration test that seeds `user_process_areas` with a role lacking `doc.edit_draft`, calls service, expects denial. Follow pattern of approval-module integration tests.
- **Non-atomic authz+write**: authz tx separate from writer tx. Small race window if perms change mid-request. Acceptable for draft edits. Revisit if contention observed.
- **Duplicated authz helpers**: `setAuthzGUC`/`loadDocumentAreaCode` now exist in both `approval/application` and `documents_v2/application`. Move to shared `internal/modules/iam/authz` package when third consumer appears.
- **Regex compile per request**: `validateValue` calls `regexp.Compile(*p.Regex)` on every placeholder write. For hot paths with large schemas, precompile at schema-load time and cache on the `Placeholder` struct (or a sibling map keyed by placeholder ID). Also adds DoS surface — a malicious template with pathological regex costs CPU per request. Consider `regexp.MustCompile` at validate time with size/complexity guard.
- **ContentPolicy substring matching**: `SetZoneContent` uses `strings.Contains(ooxml, "<w:tbl")` etc. Fragile — matches `<w:tblPr>` too, and a crafted payload could evade via attribute ordering or namespace aliasing. Long-term: parse OOXML with a real tokenizer before enforcing policy. Short-term acceptable since OOXML comes from trusted editor.
- **`qms_admin` capability question**: migration `0154_capability_doc_edit_draft.sql` grants `doc.edit_draft` to `author` and `qms_admin`. Confirm with product whether QMS admins should edit drafts directly or only approve/reject. If approval-only, drop qms_admin row.

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
