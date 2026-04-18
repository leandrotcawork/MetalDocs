# W5 Cutover + CK5/MDDM Destruction (docx-editor platform) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the greenfield docx-editor pivot. Flip the `METALDOCS_DOCX_V2_ENABLED` env var to `true` on every API / worker / frontend replica (the flag mechanism introduced by Plan A is process-level env, NOT a DB row); soak production for one business week with a committed, time-enforced evidence artifact; drop the CK5 + MDDM Postgres schema; delete every CK5 / MDDM / old-docgen code path (apps, modules, Go service/handler files, frontend features, Docker Compose services, CI workflows); promote `documents_v2` / `templates_v2` → canonical Go package names without breaking the newly-flipped system; strip the now-dead `DocxV2Enabled` config field + every call site; and land a working rollback kit before the destructive commits.

**Architecture invariant during W5:** Main branch MUST remain deployable after EVERY commit. No commit in this plan may leave the tree in a half-renamed or half-deleted state. Each destructive step is preceded by a verifiable soak / gate / snapshot. The flag-flip commit and the destructive-code commit are separated by **at minimum** a 5-business-day soak + a committed evidence artifact.

**Tech stack additions:** none. W5 is pure subtraction + rollout mechanics.

**Depends on:** Plan A (W1) + Plan B (W2) + Plan C (W3) + Plan D (W4) all executed, green in CI, and the W4 dogfood evidence artifact (`docs/superpowers/evidence/docx-v2-w4-dogfood-log.md`) filled with a `**Decision:** GO` line that passes `go test -tags=w5_gate ./tests/docx_v2/...`.

**Spec reference:** `docs/superpowers/specs/2026-04-18-docx-editor-platform-design.md` §§ Rollout → **W5**, Out of Scope, Architecture (for the DELETED-at-W5 annotations in the Monorepo layout).

**Codex hardening status:** Written per co-plan protocol. R1 + R2 outcomes recorded at end of file (Section "Codex Hardening Log"). Max 2 rounds enforced.

**Caveat up-front — one-way door risk:** Task 5 (destructive migration) and Task 6 (code destruction) are irreversible without a prior `pg_dump` + git tag. Task 0 + Task 10 (rollback kit) are prerequisites, not nice-to-haves. Do NOT advance to Task 5 before Task 10's `scripts/w5-rollback.sh` has been smoke-tested on a disposable staging DB.

---

## File Structure

**New files:**

```
# Migrations (destructive — one file; idempotent re-run guarded by schema_migrations)
migrations/
  0112_docx_v2_schema_migrations_ledger.sql  # bootstraps schema_migrations (idempotent)
  0113_docx_v2_destroy_ck5_tables.sql        # DROP TABLE ... (CK5 + MDDM + legacy-docgen)

# Evidence + gate
docs/superpowers/evidence/
  docx-v2-w5-post-flip-soak.md           # 7-business-day soak artifact (post flag flip)
  docx-v2-w5-destruction-receipt.md      # checklist after Task 6 commits; links to dumps

tests/docx_v2/
  cutover_gate_test.go                   # //go:build w5_cutover_gate — guards post-flip soak completeness
  destruction_receipt_gate_test.go       # //go:build w5_destruction_gate — guards receipt + rollback assets

# Rollback kit
scripts/
  w5-preflight-dump.sh                   # pg_dump --format=custom tagged artifact
  w5-rollback.sh                         # restore-from-dump + git reset guide
docs/runbooks/
  docx-v2-w5-cutover.md                  # the W5 operator runbook (this plan's operational twin)
  docx-v2-w5-rollback.md                 # rollback-specific runbook

# Post-cutover
docs/CHANGELOG.md                        # append W5 cutover entry (created if missing)
```

**Modified files (touched by the destruction + promotion commits):**

```
# Code destruction — subset; the full glob is enumerated in Task 6
apps/                                    # DELETE: docgen/, ck5-export/, ck5-studio/
docker-compose.yml                       # REMOVE old docgen + ck5-* services; keep docgen-v2 + gotenberg
frontend/apps/web/src/features/documents/ck5/   # DELETE entire dir
internal/modules/documents/
  application/service_ck5*.go            # DELETE 7 files
  application/service_ck5*_test.go       # DELETE paired tests
  delivery/http/handler_ck5*.go          # DELETE 5 files
  delivery/http/handler_ck5*_test.go     # DELETE paired tests
  (MDDM + collaboration code audited — see Task 6 for the full deletion list)

# Promotion — Task 7
internal/modules/documents_v2/           # RENAME → internal/modules/documents/ (after old deleted)
internal/modules/templates_v2/           # RENAME → internal/modules/templates/
frontend/apps/web/src/features/documents/v2/  # RENAME → features/documents/
frontend/apps/web/src/features/templates/v2/  # RENAME → features/templates/
api/openapi/v1/partials/documents-v2.yaml     # RENAME → documents.yaml, paths /api/v2/* stay
apps/api/cmd/metaldocs-api/main.go            # update import paths
apps/api/cmd/metaldocs-api/permissions.go     # strip "_v2" suffixes

# Feature flag removal — Task 8
apps/api/internal/platform/featureflag/docx_v2.go   # DELETE
frontend/apps/web/src/lib/featureFlags.ts           # DELETE DOCX_V2 export + callers
(additional grep-enumerated call sites in Task 8 Step 2)

# CI consolidation — Task 9
.github/workflows/
  ck5-ci.yml                              # DELETE (if present)
  docgen-ci.yml                           # DELETE (if present)
  mddm-ci.yml                             # DELETE (if present)
  docx-v2-ci.yml                          # RENAME → ci.yml (or merge into primary CI file)

# Docs cleanup — Task 10
CLAUDE.md                                 # remove CK5/MDDM sections
README.md                                 # rewrite "Architecture" + "Running locally"
docs/superpowers/archive/                 # MOVE: all legacy CK5/MDDM specs + plans under here
```

**Deleted entirely at W5:** Enumerated in Task 6 Step 2 (the full `git rm -r` list) and Task 5 (the `DROP TABLE` list).

---

## Task 0: Preflight — W4 gate + tag + snapshot

**Goal:** Before any flag flip, produce three committed, immutable artifacts: (1) a green `-tags=w5_gate` test run on the W4 dogfood log, (2) a `git tag w5-preflight` on the current main commit, (3) a `pg_dump --format=custom` snapshot of production stored off-cluster and referenced by path + sha256 in an evidence row.

**Files:**
- Create: `scripts/w5-preflight-dump.sh`
- Create: `docs/superpowers/evidence/docx-v2-w5-preflight.md`

- [ ] **Step 1: Confirm W4 dogfood gate is green**

```bash
cd "$(git rev-parse --show-toplevel)"
go test -tags=w5_gate ./tests/docx_v2/...
# Expected: PASS — TestDogfoodLogComplete + any other -tags=w5_gate suites.
# If FAIL: STOP. Return to Plan D Task 21; fill the log and re-run.
```

- [ ] **Step 2: Write `scripts/w5-preflight-dump.sh`**

```bash
#!/usr/bin/env bash
# scripts/w5-preflight-dump.sh
# Writes a pg_dump --format=custom snapshot of the production schema + data
# immediately before the W5 flag flip. The resulting dump is the ONLY path
# back if Task 5 (destructive migration) corrupts or loses data.
#
# Usage (from an operator shell with $PGHOST / $PGUSER / $PGDATABASE set to prod):
#   ./scripts/w5-preflight-dump.sh /path/to/secure/backup/dir
set -euo pipefail

if [ $# -lt 1 ]; then
  echo "usage: $0 <output-dir>" >&2
  exit 2
fi

OUT_DIR="$1"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
DUMP="${OUT_DIR}/metaldocs-w5-preflight-${STAMP}.dump"

mkdir -p "${OUT_DIR}"

pg_dump \
  --format=custom \
  --no-owner \
  --no-privileges \
  --verbose \
  --file="${DUMP}"

SHA="$(sha256sum "${DUMP}" | awk '{print $1}')"
SIZE="$(stat -c%s "${DUMP}")"

cat <<EOF
=== W5 preflight dump complete ===
path: ${DUMP}
sha256: ${SHA}
size_bytes: ${SIZE}
timestamp_utc: ${STAMP}
next steps:
  1. Record the fields above in docs/superpowers/evidence/docx-v2-w5-preflight.md
  2. Copy the .dump file to at least one off-cluster location (S3 + encrypted local NAS).
  3. 'git tag w5-preflight' on the commit intended as the rollback target.
EOF
```

Make it executable: `chmod +x scripts/w5-preflight-dump.sh`.

- [ ] **Step 3: Write evidence template**

`docs/superpowers/evidence/docx-v2-w5-preflight.md`:

```markdown
# W5 Preflight Evidence

Date of dump (UTC): YYYY-MM-DDTHH:MM:SSZ
Operator: @handle
Git commit SHA tagged `w5-preflight`: <40-char sha>

## pg_dump artifact
- Primary location: <s3://... or NAS path>
- Secondary location: <s3://... or NAS path>
- sha256: <64-char hex>
- size_bytes: <int>
- pg_dump options: `--format=custom --no-owner --no-privileges`

## Verification
- [ ] `pg_restore --list <path>` returned a non-empty TOC.
- [ ] sha256 recomputed at secondary location MATCHES primary.
- [ ] Git tag `w5-preflight` pushed to origin (`git push origin w5-preflight`).
- [ ] W4 dogfood gate (`go test -tags=w5_gate`) green on the tagged commit.

## Attestation
- [ ] Admin on-call sign-off: @handle — YYYY-MM-DD
- [ ] Product manager sign-off: @handle — YYYY-MM-DD
```

- [ ] **Step 4: Tag + commit**

```bash
rtk git add scripts/w5-preflight-dump.sh docs/superpowers/evidence/docx-v2-w5-preflight.md
rtk git commit -m "gate(docx-v2/w5): preflight dump script + evidence template"
# Operator (after filling the evidence file with real dump details):
#   rtk git tag w5-preflight
#   rtk git push origin w5-preflight
```

---

## Task 1: Global flag flip — env var `METALDOCS_DOCX_V2_ENABLED=true` on every replica

**Goal:** Turn `METALDOCS_DOCX_V2_ENABLED` to `true` on every API, worker, and frontend replica in production. Per Plan A Task 9, this flag is a PROCESS-LEVEL env var read into `config.FeatureFlagsConfig.DocxV2Enabled` at boot — it is NOT a DB column. There is therefore no SQL migration here; the flip is a deploy-config change propagated by the repo's existing deploy mechanism (`deploy/compose/.env` → Docker Compose for dev/staging; whatever prod's equivalent is).

**Why this is the failure mode Codex flagged:** if the env var is set on only N-of-M replicas, traffic hashes unpredictably between new + old behavior and soak telemetry becomes meaningless. The post-apply verification below asserts every replica has the flag set AND the running binary has the Plan A config field, not a stale image.

**Files:**
- Modify: `deploy/compose/.env.production` (and/or the prod equivalent — Terraform var, Helm values, Kubernetes ConfigMap — whichever is authoritative in this repo's deploy pipeline)
- Modify: `deploy/compose/.env.staging` (first, for rehearsal)
- Modify: `.env.example` — flip the example value to `true` so any new dev env inherits the cutover default
- Create: `docs/superpowers/evidence/docx-v2-w5-flip-verification.md` (records per-replica proof of the flip)

- [ ] **Step 1: Identify the authoritative deploy surface**

```bash
cd "$(git rev-parse --show-toplevel)"
rtk grep -rn 'METALDOCS_DOCX_V2_ENABLED' \
  deploy/ .env* infra/ chart/ k8s/ 2>/dev/null
# Expected: hits in deploy/compose/.env*, possibly infra/*.tf or chart/*.yaml.
# The authoritative prod surface is whichever file the prod deploy pipeline
# reads. If multiple surfaces exist (e.g. staging reads .env but prod reads
# Terraform), DO NOT flip one without flipping the other — document both.
```

- [ ] **Step 2: Stage the flip on staging first**

```bash
# On staging deploy:
#   - Update METALDOCS_DOCX_V2_ENABLED=true in the staging env surface.
#   - Re-deploy (image unchanged, just env restart).
make deploy TARGET=staging   # or the equivalent command in this repo
```

- [ ] **Step 3: Per-replica verification — ALL THREE WORKLOAD CLASSES — DO NOT SKIP**

Three workload classes carry the flag: **API**, **worker**, **frontend**. Each class has its own verification path. A partial rollout (e.g., API flipped but worker on old image) produces split-brain behavior during soak and invalidates the GO/NO-GO signal before Task 3's destructive migration. EVERY replica of EVERY class must be verified before the soak clock in Task 2 starts.

**3a. API replicas — `GET /api/v1/feature-flags`:**

```bash
# Docker Compose staging (one API replica behind LB):
curl -sS http://staging.metaldocs.example.com/api/v1/feature-flags | jq '.'
# Expected: { ..., "METALDOCS_DOCX_V2_ENABLED": true, ... }

# K8s — iterate over API pods, hitting each directly via kubectl exec:
kubectl -n metaldocs get pods -l app=api -o name | while read pod; do
  kubectl -n metaldocs exec "$pod" -- curl -sS http://localhost:8080/api/v1/feature-flags \
    | jq -r --arg pod "$pod" '[$pod, .METALDOCS_DOCX_V2_ENABLED // false] | @tsv'
done
# Expected: one line per pod, all `true`.
```

**3b. Worker replicas — direct `printenv` since workers have no HTTP surface:**

```bash
kubectl -n metaldocs get pods -l app=worker -o name | while read pod; do
  flag=$(kubectl -n metaldocs exec "$pod" -- printenv METALDOCS_DOCX_V2_ENABLED 2>/dev/null || echo 'UNSET')
  image=$(kubectl -n metaldocs get "$pod" -o jsonpath='{.spec.containers[0].image}')
  printf '%s\t%s\t%s\n' "$pod" "$image" "$flag"
done
# Expected: one line per pod, flag=`true`, image tag identical across pods.
# If flag=UNSET or image tag drifts, STOP — roll the replica before proceeding.
```

**3c. Frontend replicas — `printenv` + built-asset sanity:**

The frontend image may bake the flag at build time (Vite env inlining) OR read it at runtime from a mounted config. If inlined, the running container's `printenv` may show UNSET even when the build is correct.

```bash
kubectl -n metaldocs get pods -l app=web -o name | while read pod; do
  flag=$(kubectl -n metaldocs exec "$pod" -- printenv METALDOCS_DOCX_V2_ENABLED 2>/dev/null || echo 'UNSET')
  image=$(kubectl -n metaldocs get "$pod" -o jsonpath='{.spec.containers[0].image}')
  printf '%s\t%s\t%s\n' "$pod" "$image" "$flag"
done

# Asset-build sanity: fetch one frontend replica's config JSON (emitted by
# Plan A Task 9's client-side wrapper) and assert the flag is true.
kubectl -n metaldocs get pods -l app=web -o name | head -1 | xargs -I{} \
  kubectl -n metaldocs exec {} -- curl -sS http://localhost:3000/config.json \
  | jq '.METALDOCS_DOCX_V2_ENABLED'
# Expected: true
```

Record every replica's response, per class, in `docs/superpowers/evidence/docx-v2-w5-flip-verification.md`:

```markdown
# W5 Flag-Flip Verification

Staging flip UTC: YYYY-MM-DDTHH:MM:SSZ
Prod flip UTC:    YYYY-MM-DDTHH:MM:SSZ
Operator:         @handle

## Staging — API replicas
| Replica ID | Image tag | Flag value (from /api/v1/feature-flags) |
|------------|-----------|------------------------------------------|
| pod/api-staging-1 | v1.42.0 | true |

## Staging — Worker replicas
| Replica ID | Image tag | Flag value (printenv) |
|------------|-----------|------------------------|
| pod/worker-staging-1 | v1.42.0 | true |

## Staging — Frontend replicas
| Replica ID | Image tag | printenv value | /config.json value |
|------------|-----------|-----------------|---------------------|
| pod/web-staging-1 | v1.42.0 | true (or baked) | true |

## Production — API replicas
| Replica ID | Image tag | Flag value |
|------------|-----------|------------|
| pod/api-prod-1 | v1.42.0 | true |

## Production — Worker replicas
| Replica ID | Image tag | Flag value |
|------------|-----------|------------|
| pod/worker-prod-1 | v1.42.0 | true |

## Production — Frontend replicas
| Replica ID | Image tag | printenv value | /config.json value |
|------------|-----------|-----------------|---------------------|
| pod/web-prod-1 | v1.42.0 | true (or baked) | true |

## Attestation — ALL THREE WORKLOAD CLASSES
- [ ] Every staging API replica returned `true`.
- [ ] Every staging worker replica has `METALDOCS_DOCX_V2_ENABLED=true`.
- [ ] Every staging frontend replica serves `/config.json` with the flag `true`.
- [ ] Every production API replica returned `true`.
- [ ] Every production worker replica has `METALDOCS_DOCX_V2_ENABLED=true`.
- [ ] Every production frontend replica serves `/config.json` with the flag `true`.
- [ ] Image tag is identical across replicas within each class (no mid-flip rolling deploy).
- [ ] Admin sign-off: @handle — YYYY-MM-DD
- [ ] SRE sign-off: @handle — YYYY-MM-DD
```

**All three workload classes must attest `true` before Step 4 (prod apply) completes and the soak clock starts. A missing class is a BLOCKER, not a warning.**

- [ ] **Step 4: Apply on prod**

After staging soaks for ≥ 30 minutes with the verification above green, repeat Step 2 + Step 3 for prod. This is the moment the soak clock in Task 2 starts.

- [ ] **Step 5: Commit**

```bash
rtk git add deploy/compose/.env.production deploy/compose/.env.staging .env.example \
         docs/superpowers/evidence/docx-v2-w5-flip-verification.md
rtk git commit -m "feat(docx-v2/w5): flip METALDOCS_DOCX_V2_ENABLED=true on all replicas"
```

---

## Task 2: Post-flip soak — 5 business days, evidence artifact + gate test

**Goal:** A committed, filled evidence artifact with 5 dated business-day entries that covers the post-flip soak window AND a Go build-tagged test that refuses to pass if the artifact is incomplete. The destructive migration in Task 5 MUST NOT be applied before this gate is green.

**Files:**
- Create: `docs/superpowers/evidence/docx-v2-w5-post-flip-soak.md`
- Create: `tests/docx_v2/cutover_gate_test.go`

- [ ] **Step 1: Evidence log template**

`docs/superpowers/evidence/docx-v2-w5-post-flip-soak.md` is committed empty-shell with the structure below; it MUST be filled in daily for 5 business days **starting from the Task 1 production flag-flip timestamp** (recorded in `docs/superpowers/evidence/docx-v2-w5-flip-verification.md` as "Prod flip UTC"). Task 3's destructive migration is what Task 2 gates — migrations 0112/0113 do NOT start the soak clock; Task 1 does. A soak log whose Day 1 date precedes the Task 1 prod flip timestamp fails the gate.

```markdown
# W5 Post-Flip Soak Log

Flag flip applied (UTC): YYYY-MM-DDTHH:MM:SSZ (env-var rollout — see Task 1 evidence)
Soak window:            5 business days, zero P0 incidents.
Target audience:        100% of tenants (all API/worker/frontend replicas report `METALDOCS_DOCX_V2_ENABLED=true` in `GET /api/v1/feature-flags`).

## Monitored SLOs (rolling 24h windows)
- /api/v2/documents POST availability ≥ 99.9%
- /api/v2/export/pdf p95 < 5000ms
- cached=false ratio < 50% (target 30% after day 2)
- 429 rate per user < 5/day max
- Gotenberg OOM events: 0

## Participants
- Admin on-call: @handle
- SRE: @handle
- Product manager: @handle

## Daily results

### Day 1 — YYYY-MM-DD
- /api/v2/documents availability: __.__%
- /api/v2/export/pdf p95: __ms
- cached=false ratio: __%
- 429 rate per user (max): __/day
- Gotenberg OOM events: __
- Tenants exercising /api/v2: __/__ (active/total)
- Incidents: none | P0 | P1 | P2 (describe)
- Sign-off: @admin @sre

### Day 2 — YYYY-MM-DD
- /api/v2/documents availability: __.__%
- /api/v2/export/pdf p95: __ms
- cached=false ratio: __%
- 429 rate per user (max): __/day
- Gotenberg OOM events: __
- Tenants exercising /api/v2: __/__ (active/total)
- Incidents: none | P0 | P1 | P2 (describe)
- Sign-off: @admin @sre

### Day 3 — YYYY-MM-DD
(same structure as Day 1)

### Day 4 — YYYY-MM-DD
(same structure as Day 1)

### Day 5 — YYYY-MM-DD
(same structure as Day 1)

## Exit decision

- [ ] All 5 days logged with within-SLO values.
- [ ] Zero P0 incidents.
- [ ] Tenants-exercising fraction reached ≥ 80% on at least one day (proves real traffic, not dark launch).
- [ ] Admin sign-off: @handle — YYYY-MM-DD
- [ ] Product manager sign-off: @handle — YYYY-MM-DD
- [ ] SRE sign-off: @handle — YYYY-MM-DD

**Decision:** GO / NO-GO → W5 destruction (Task 5 + Task 6).
```

- [ ] **Step 2: Gate test**

`tests/docx_v2/cutover_gate_test.go`:

```go
//go:build w5_cutover_gate

package docx_v2_test

import (
    "os"
    "regexp"
    "strconv"
    "strings"
    "testing"
    "time"
)

// TestPostFlipSoakLogComplete runs under the w5_cutover_gate build tag and
// is the CI prerequisite for Task 3 (destructive migration 0113) and
// Task 6 (code destruction).
//
// Time-enforcement (Codex R1 fix): the gate parses the flip timestamp from
// the log header and asserts
//   (a) all 5 Day dates are unique, in strict chronological order,
//   (b) every Day date is on-or-after the flip date,
//   (c) every Day date is not in the future relative to time.Now().UTC(),
//   (d) the span from Day 1 to Day 5 is at least 5 business days (Mon–Fri),
//       which means the earliest possible GO is 4 business days after flip.
// This prevents same-day backfill of all 5 entries, which would otherwise
// satisfy the structural checks without a real soak.
func TestPostFlipSoakLogComplete(t *testing.T) {
    raw, err := os.ReadFile("../../docs/superpowers/evidence/docx-v2-w5-post-flip-soak.md")
    if err != nil {
        t.Fatalf("post-flip soak log missing: %v", err)
    }
    content := string(raw)

    // Flip timestamp header: "Flag flip applied (UTC): YYYY-MM-DDTHH:MM:SSZ (env-var rollout — see Task 1 evidence)".
    flipRe := regexp.MustCompile(`(?m)^Flag flip applied \(UTC\):\s+(\d{4}-\d{2}-\d{2})T`)
    flipMatch := flipRe.FindStringSubmatch(content)
    if flipMatch == nil {
        t.Fatalf("soak log missing or malformed 'Flag flip applied (UTC):' header")
    }
    flipDate, err := time.Parse("2006-01-02", flipMatch[1])
    if err != nil {
        t.Fatalf("unparsable flip date %q: %v", flipMatch[1], err)
    }

    // 5 dated Day headers, no placeholder dates.
    headerRe := regexp.MustCompile(`(?m)^### Day [1-5] — (\d{4}-\d{2}-\d{2})$`)
    headers := headerRe.FindAllStringSubmatch(content, -1)
    if len(headers) < 5 {
        t.Fatalf("soak log must contain 5 dated Day sections; found %d", len(headers))
    }
    for _, h := range headers {
        if h[1] == "YYYY-MM-DD" {
            t.Fatalf("soak log Day header uses placeholder date: %q", h[0])
        }
    }

    // Parse Day dates; enforce unique, chronological, on-or-after flipDate,
    // not in the future.
    dayDates := make([]time.Time, 0, len(headers))
    seenDates := map[string]bool{}
    nowUTC := time.Now().UTC().Truncate(24 * time.Hour)
    for i, h := range headers {
        if seenDates[h[1]] {
            t.Fatalf("duplicate Day date %q at position %d", h[1], i+1)
        }
        seenDates[h[1]] = true
        d, err := time.Parse("2006-01-02", h[1])
        if err != nil {
            t.Fatalf("unparsable Day %d date %q: %v", i+1, h[1], err)
        }
        if d.Before(flipDate) {
            t.Fatalf("Day %d date %s is BEFORE flip date %s — soak cannot start before the flip",
                i+1, d.Format("2006-01-02"), flipDate.Format("2006-01-02"))
        }
        if d.After(nowUTC) {
            t.Fatalf("Day %d date %s is in the future (now=%s)",
                i+1, d.Format("2006-01-02"), nowUTC.Format("2006-01-02"))
        }
        if i > 0 && !d.After(dayDates[i-1]) {
            t.Fatalf("Day %d date %s is not strictly after Day %d date %s",
                i+1, d.Format("2006-01-02"), i, dayDates[i-1].Format("2006-01-02"))
        }
        dayDates = append(dayDates, d)
    }

    // Business-day span: Day 1 → Day 5 must span at least 5 business days
    // (inclusive). 5 consecutive Mon–Fri = 5 business days; weekends or
    // holidays may extend the calendar span — we only enforce the minimum.
    bd := businessDaysInclusive(dayDates[0], dayDates[len(dayDates)-1])
    if bd < 5 {
        t.Fatalf("soak span Day 1 (%s) → Day 5 (%s) is %d business days; require >= 5",
            dayDates[0].Format("2006-01-02"), dayDates[len(dayDates)-1].Format("2006-01-02"), bd)
    }

    placeholderTokens := []string{
        "__.__%", "__ms", "__%", "__/day", "__/__",
        "YYYY-MM-DD",
        "GO / NO-GO",
        "(same structure as Day 1)",
    }
    ellipsisLineRe := regexp.MustCompile(`(?m)^\s*\.\.\.\s*$`)
    if ellipsisLineRe.FindString(content) != "" {
        t.Fatalf("soak log contains an unfilled '...' line")
    }
    for _, tok := range placeholderTokens {
        if strings.Contains(content, tok) {
            t.Fatalf("soak log contains unfilled placeholder token %q", tok)
        }
    }

    dayBlocks := splitCutoverDayBlocks(content)
    if len(dayBlocks) < 5 {
        t.Fatalf("could not isolate 5 day blocks; got %d", len(dayBlocks))
    }

    requiredRows := []struct {
        name string
        re   *regexp.Regexp
    }{
        {"availability", regexp.MustCompile(`(?m)^- /api/v2/documents availability:\s+(\d{2,3}\.\d{1,2})\s*%`)},
        {"p95", regexp.MustCompile(`(?m)^- /api/v2/export/pdf p95:\s+(\d+)\s*ms\b`)},
        {"cached ratio", regexp.MustCompile(`(?m)^- cached=false ratio:\s+(\d{1,3})\s*%`)},
        {"429 max", regexp.MustCompile(`(?m)^- 429 rate per user \(max\):\s+(\d+)/day\b`)},
        {"OOM events", regexp.MustCompile(`(?m)^- Gotenberg OOM events:\s+(\d+)\b`)},
        {"tenants exercising", regexp.MustCompile(`(?m)^- Tenants exercising /api/v2:\s+(\d+)/(\d+)\b`)},
        {"Incidents", regexp.MustCompile(`(?m)^- Incidents:\s+(none|P0|P1|P2)\b`)},
        {"Sign-off", regexp.MustCompile(`(?m)^- Sign-off:\s+@\S+\s+@\S+\s*$`)},
    }

    for i, block := range dayBlocks {
        dayNum := i + 1
        for _, row := range requiredRows {
            if row.re.FindString(block) == "" {
                t.Fatalf("Day %d: missing or malformed %q row", dayNum, row.name)
            }
        }
        // At least one day must have tenants-exercising fraction >= 80%.
        // (Checked once, outside the loop.)
    }

    // Aggregate: at least one day with (active/total) >= 0.80.
    exerciseRe := regexp.MustCompile(`(?m)^- Tenants exercising /api/v2:\s+(\d+)/(\d+)\b`)
    anyHigh := false
    for _, m := range exerciseRe.FindAllStringSubmatch(content, -1) {
        a, errA := strconv.Atoi(m[1])
        b, errB := strconv.Atoi(m[2])
        if errA != nil || errB != nil || b == 0 {
            continue
        }
        if float64(a)/float64(b) >= 0.80 {
            anyHigh = true
            break
        }
    }
    if !anyHigh {
        t.Fatalf("no day with >=80%% tenants exercising /api/v2 — this is a dark-launch proof requirement")
    }

    // Three sign-offs with @handle + ISO date.
    for _, label := range []string{"Admin sign-off:", "Product manager sign-off:", "SRE sign-off:"} {
        re := regexp.MustCompile(regexp.QuoteMeta(label) + `\s+@\S+\s+—\s+\d{4}-\d{2}-\d{2}`)
        if re.FindString(content) == "" {
            t.Fatalf("soak log missing filled %q line", label)
        }
    }

    // Explicit GO decision.
    goRe := regexp.MustCompile(`(?m)^\*\*Decision:\*\*\s+GO\b`)
    if goRe.FindString(content) == "" {
        t.Fatalf("soak log missing explicit **Decision:** GO line")
    }
}

// businessDaysInclusive counts Monday–Friday days from `from` through `to`
// inclusive. Does NOT subtract holidays (the log operator records holidays
// as Incidents: none; the count merely guarantees ≥ 5 calendar working
// days have elapsed, which is the minimum observable-operations window).
func businessDaysInclusive(from, to time.Time) int {
    if to.Before(from) {
        return 0
    }
    count := 0
    d := from
    for !d.After(to) {
        wd := d.Weekday()
        if wd != time.Saturday && wd != time.Sunday {
            count++
        }
        d = d.AddDate(0, 0, 1)
    }
    return count
}

// splitCutoverDayBlocks returns the body text of each "### Day N — …" section,
// up to the next "### " or "## " header, in order.
func splitCutoverDayBlocks(content string) []string {
    blocks := []string{}
    headerRe := regexp.MustCompile(`(?m)^### Day [1-5] — \d{4}-\d{2}-\d{2}\s*$`)
    idxs := headerRe.FindAllStringIndex(content, -1)
    for i, start := range idxs {
        bodyStart := start[1]
        var bodyEnd int
        if i+1 < len(idxs) {
            bodyEnd = idxs[i+1][0]
        } else {
            rest := content[bodyStart:]
            nextSec := regexp.MustCompile(`(?m)^##[^#]`).FindStringIndex(rest)
            if nextSec != nil {
                bodyEnd = bodyStart + nextSec[0]
            } else {
                bodyEnd = len(content)
            }
        }
        blocks = append(blocks, content[bodyStart:bodyEnd])
    }
    return blocks
}

```

- [ ] **Step 3: CI wiring — real changed-path gate**

Add two jobs to `.github/workflows/docx-v2-ci.yml` (pattern identical to W4 Task 21, with the path filter widened to the W5 artifacts):

```yaml
  detect-w5-artifacts:
    runs-on: ubuntu-latest
    outputs:
      w5_dest: ${{ steps.filter.outputs.w5_dest }}
    steps:
      - uses: actions/checkout@v4
      - id: filter
        uses: dorny/paths-filter@v3
        with:
          filters: |
            w5_dest:
              - 'migrations/0112_docx_v2_schema_migrations_ledger.sql'
              - 'migrations/0113_docx_v2_destroy_ck5_tables.sql'
              - 'docs/superpowers/plans/2026-04-*-docx-editor-w5-*.md'

  w5-cutover-gate:
    needs: detect-w5-artifacts
    if: needs.detect-w5-artifacts.outputs.w5_dest == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: 1.25.x }
      - run: go test -tags=w5_cutover_gate ./tests/docx_v2/...
```

As in Plan D, a second workflow file with top-level `pull_request.paths` is also acceptable; **NEVER** use `github.event.pull_request.changed_files` — it is a numeric count, not a path list.

- [ ] **Step 4: Commit (empty-shell artifact + test + CI)**

```bash
go test -tags=w5_cutover_gate ./tests/docx_v2/... || echo "expected to fail until log is filled"
rtk git add docs/superpowers/evidence/docx-v2-w5-post-flip-soak.md \
         tests/docx_v2/cutover_gate_test.go \
         .github/workflows/docx-v2-ci.yml
rtk git commit -m "gate(docx-v2/w5): post-flip soak evidence artifact + cutover gate"
```

---

## Task 3: Destructive migration `0113_docx_v2_destroy_ck5_tables.sql` (+ ledger bootstrap `0112`)

**Goal:** Drop every Postgres table owned by the CK5 / MDDM / old-docgen code paths in one transaction. Before the DROP statements run, a WHERE-IS-MY-DATA sentinel asserts the W4-era tables (`templates`, `template_versions`, `documents`, `document_revisions`, `document_checkpoints`, `editor_sessions`, `autosave_pending_uploads`, `document_exports`, `template_audit_log`) all exist — if any are missing, the migration aborts rather than destroying irrecoverable state. The migration is ALSO idempotent: re-applying it after success becomes a no-op (detected via the `schema_migrations` ledger bootstrapped in 0112), so an accidental re-run during a deploy cycle does NOT fail the deploy.

**Codex R1 Fix 2 applied:** This repo previously applied migrations with raw `psql -f` and no ledger. The W5 destructive step MUST be tracked in a `schema_migrations` table so that (a) a re-run is a no-op instead of a spurious sentinel failure, (b) operators can verify the migration is applied without re-running, and (c) a future migration runner has a checkpoint to cut against. Migration `0112` bootstraps the ledger; `0113` consults and updates it.

**Blocked until:** Task 2 gate green (`go test -tags=w5_cutover_gate` PASS) AND Task 10 rollback kit smoke-tested on staging AND a fresh `w5-preflight` pg_dump is on disk in ≥ 2 locations.

**Files:**
- Create: `migrations/0112_docx_v2_schema_migrations_ledger.sql`
- Create: `migrations/0113_docx_v2_destroy_ck5_tables.sql`

- [ ] **Step 0: Ledger bootstrap — `0112_docx_v2_schema_migrations_ledger.sql`**

```sql
-- 0112_docx_v2_schema_migrations_ledger.sql
-- Introduces a minimal schema_migrations table so destructive migrations
-- (0113 onwards) can be idempotent under re-run, and operators can query
-- applied-state without re-executing. Idempotent: IF NOT EXISTS guards
-- every CREATE; INSERT is ON CONFLICT DO NOTHING.

BEGIN;

CREATE TABLE IF NOT EXISTS public.schema_migrations (
  version     TEXT PRIMARY KEY,
  applied_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  description TEXT
);

-- Record this migration itself.
INSERT INTO public.schema_migrations (version, description)
VALUES ('0112', 'bootstrap schema_migrations ledger (docx-v2/w5)')
ON CONFLICT (version) DO NOTHING;

COMMIT;
```

- [ ] **Step 1: Enumerate the kill list**

Run locally first to produce the ground-truth list of CK5 / MDDM tables to drop. **DO NOT** paste the raw output into the migration — manually review each row against the spec §DELETED-at-W5 annotations + the CK5 code paths enumerated in Task 6.

```bash
PGPASSWORD=metaldocs psql -h 127.0.0.1 -U metaldocs -d metaldocs -v ON_ERROR_STOP=1 <<'SQL'
SELECT table_name
  FROM information_schema.tables
 WHERE table_schema = 'public'
   AND (
        table_name LIKE 'mddm_%'
     OR table_name LIKE 'ck5_%'
     OR table_name LIKE 'document_template_versions%'
     OR table_name LIKE 'template_drafts%'
     OR table_name IN (
       'documents',            -- OLD one; re-created by Plan B 0101+ with same name? verify below
       'document_versions',
       'document_profile_schemas',
       'document_collaboration',
       'document_departments',
       'document_profile_registry',
       'document_family',
       'document_taxonomy',
       'document_type_runtime',
       'blocks_json_artifacts',
       'renderer_pin_events',
       'rich_envelope_events',
       'shadow_diff_events'
     )
   )
 ORDER BY table_name;
SQL
```

**WARNING — name collision check:** Plan B introduced `templates` and `template_versions`; the legacy CK5 path ALSO named things `templates` and `document_template_versions`. If greenfield migrations 0101+ re-used the legacy names, the legacy tables were already renamed or dropped during W1. Verify with:

```bash
PGPASSWORD=metaldocs psql -h 127.0.0.1 -U metaldocs -d metaldocs -c "\d templates"
# Expected: the schema matches spec §Database schema (tenant_id, key, current_published_version_id, ...).
# If the schema matches legacy CK5 (has body_blocks, renderer_pin, etc.), STOP — Plan B renamed on collision and the new schema lives under a different name; fix Plan E references before proceeding.
```

- [ ] **Step 2: Write migration**

```sql
-- 0113_docx_v2_destroy_ck5_tables.sql
-- DESTRUCTIVE. Drops every CK5 / MDDM / old-docgen table.
-- Guarded by sentinel assertions on the W4-era schema; if ANY W4 table is
-- missing the migration aborts before executing a single DROP.
-- Idempotent: a successful prior apply is recorded in `schema_migrations`.
-- A re-run detects the ledger entry and short-circuits to a clean no-op
-- COMMIT so an accidental re-apply during a deploy cycle does NOT fail.
-- Depends on: 0101-0112 applied and Task 2 gate green.

BEGIN;

-- Idempotency model: the ledger row `schema_migrations(version='0113')` is
-- the authoritative "already applied" marker. Every subsequent DO block
-- begins with `IF EXISTS (...ledger...) THEN RETURN; END IF;` so a re-run
-- short-circuits to a clean empty transaction. The final INSERT records
-- 0113 atomically with the DROPs so partial apply is impossible.

-- Informational NOTICE for operator feedback on re-run.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM public.schema_migrations WHERE version = '0113') THEN
    RAISE NOTICE '0113 already recorded in schema_migrations — running as no-op';
  END IF;
END $$;

-- Sentinel: every W4-era table must exist. Missing any of these aborts.
-- Skipped on re-run (ledger already has 0113).
DO $$
DECLARE
  required_tables TEXT[] := ARRAY[
    'templates', 'template_versions',
    'documents', 'document_revisions', 'document_checkpoints',
    'editor_sessions', 'autosave_pending_uploads',
    'document_exports', 'template_audit_log'
  ];
  t TEXT;
BEGIN
  IF EXISTS (SELECT 1 FROM public.schema_migrations WHERE version = '0113') THEN
    RETURN;
  END IF;
  FOREACH t IN ARRAY required_tables LOOP
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.tables
       WHERE table_schema = 'public' AND table_name = t
    ) THEN
      RAISE EXCEPTION
        'W4-era table %.% missing — refusing to run destructive migration 0113',
        'public', t;
    END IF;
  END LOOP;
END $$;

-- Drop CK5 / MDDM / legacy-docgen tables. CASCADE handles FKs from dead tables
-- onto other dead tables; W4 tables were audited to ensure NONE of them FK
-- into these legacy tables (confirmed in Plan B Task 2 + Plan C Task 1).
-- Wrapped in a DO block so we can ledger-guard the whole thing.
DO $$
DECLARE
  kill_tables TEXT[] := ARRAY[
    'mddm_shadow_diff_events','mddm_audit_events','mddm_releases',
    'mddm_block_versions','mddm_blocks','mddm_templates',
    'template_drafts','document_template_versions_audit',
    'document_template_versions','document_templates_ck5',
    'document_profile_schemas','document_collaboration',
    'document_departments','document_profile_registry',
    'document_family','document_taxonomy','document_type_runtime',
    -- Legacy `documents` + `document_versions` (pre-pivot). Name MUST differ
    -- from the greenfield `documents` table — verified in Step 1. If the
    -- greenfield migration reused the name, the pre-drop sentinel above
    -- catches it. If W1 suffixed legacy tables with `_legacy`, drop those.
    'documents_legacy','document_versions_legacy','document_versions',
    'blocks_json_artifacts','renderer_pin_events',
    'rich_envelope_events','shadow_diff_events'
  ];
  t TEXT;
BEGIN
  IF EXISTS (SELECT 1 FROM public.schema_migrations WHERE version = '0113') THEN
    RETURN;
  END IF;
  FOREACH t IN ARRAY kill_tables LOOP
    EXECUTE format('DROP TABLE IF EXISTS public.%I CASCADE', t);
  END LOOP;
END $$;

-- Post-drop assertion: every W4-era table still exists. Skipped on re-run.
DO $$
DECLARE
  required_tables TEXT[] := ARRAY[
    'templates', 'template_versions',
    'documents', 'document_revisions', 'document_checkpoints',
    'editor_sessions', 'autosave_pending_uploads',
    'document_exports', 'template_audit_log'
  ];
  t TEXT;
BEGIN
  IF EXISTS (SELECT 1 FROM public.schema_migrations WHERE version = '0113') THEN
    RETURN;
  END IF;
  FOREACH t IN ARRAY required_tables LOOP
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.tables
       WHERE table_schema = 'public' AND table_name = t
    ) THEN
      RAISE EXCEPTION
        'CATASTROPHE: W4-era table %.% went missing during 0113 — ABORT',
        'public', t;
    END IF;
  END LOOP;
END $$;

-- Record 0113 as applied. Safe on re-run (ON CONFLICT DO NOTHING).
INSERT INTO public.schema_migrations (version, description)
VALUES ('0113', 'destroy CK5/MDDM/legacy-docgen tables (docx-v2/w5)')
ON CONFLICT (version) DO NOTHING;

COMMIT;
```

- [ ] **Step 3: Dry-run on a restored dump**

```bash
# Restore the w5-preflight dump to a throwaway DB.
createdb metaldocs_w5_dryrun
pg_restore --dbname=metaldocs_w5_dryrun \
           --no-owner --no-privileges --verbose \
           /path/to/metaldocs-w5-preflight-*.dump
# Apply 0112 + 0113 in order; expect CLEAN COMMIT on both.
PGPASSWORD=*** psql -d metaldocs_w5_dryrun -v ON_ERROR_STOP=1 \
  -f migrations/0112_docx_v2_schema_migrations_ledger.sql \
  -f migrations/0113_docx_v2_destroy_ck5_tables.sql
# Expected output:
#   BEGIN
#   (sentinel DO blocks)
#   DROP TABLE (xN) via EXECUTE inside DO block — no explicit DROP lines
#   (post-drop sentinel DO block)
#   INSERT 0 1   (schema_migrations row for 0113)
#   COMMIT
# If RAISE EXCEPTION fires, re-read Step 1.

# Ledger verification — both rows MUST be present after first apply.
PGPASSWORD=*** psql -d metaldocs_w5_dryrun -v ON_ERROR_STOP=1 -c \
  "SELECT version, description FROM public.schema_migrations WHERE version IN ('0112','0113') ORDER BY version;"
# Expected: exactly two rows, 0112 then 0113.

# Idempotency verification — re-apply 0113; expect clean no-op COMMIT.
PGPASSWORD=*** psql -d metaldocs_w5_dryrun -v ON_ERROR_STOP=1 \
  -f migrations/0113_docx_v2_destroy_ck5_tables.sql
# Expected: BEGIN; NOTICE '0113 already recorded...'; INSERT 0 0; COMMIT.
# No sentinel failure, no DROP attempts, no exit code != 0.
```

- [ ] **Step 4: Commit (apply only on prod AFTER all gates green)**

```bash
rtk git add migrations/0112_docx_v2_schema_migrations_ledger.sql \
            migrations/0113_docx_v2_destroy_ck5_tables.sql
rtk git commit -m "feat(migrations/docx-v2): 0112 ledger + 0113 drop CK5/MDDM/legacy-docgen tables (destructive)"
# Operator applies on prod ONLY after:
#   - Task 2 cutover gate green
#   - Task 10 rollback kit smoke-tested
#   - fresh pg_dump on disk (redundant with Task 0; re-snapshot the day-of)
#   - 0112 applied first, verified via `SELECT * FROM schema_migrations WHERE version='0112'`
```

---

## Task 4: Dead-code census — produce the exact kill list

**Goal:** Before a single `git rm` runs, produce an auditable list of every file, every directory, and every Docker Compose service slated for deletion. The list becomes a commit-text paste so reviewers can spot-check against the spec §DELETED-at-W5 annotations.

**Files:**
- Create: `docs/superpowers/evidence/docx-v2-w5-destruction-census.md`

- [ ] **Step 1: Filesystem census**

```bash
cd "$(git rev-parse --show-toplevel)"

{
  echo "## Apps to delete"
  ls -d apps/ck5-export apps/ck5-studio apps/docgen 2>/dev/null | sed 's/^/- /'

  echo
  echo "## Frontend feature trees to delete"
  find frontend/apps/web/src/features/documents/ck5 -maxdepth 0 -type d 2>/dev/null | sed 's/^/- /'
  find frontend/apps/web/src/features -maxdepth 2 -type d -name 'mddm*' 2>/dev/null | sed 's/^/- /'

  echo
  echo "## Go files to delete (internal/modules/documents)"
  find internal/modules/documents -type f \( \
        -name 'service_ck5*.go' -o \
        -name 'handler_ck5*.go' -o \
        -name 'service_mddm*.go' -o \
        -name 'handler_mddm*.go' -o \
        -name 'service_collaboration*.go' -o \
        -name 'service_template_lifecycle*.go' -o \
        -name 'service_browser_editor*.go' -o \
        -name 'service_etapa_body*.go' -o \
        -name 'service_document_runtime*.go' -o \
        -name 'service_schema_runtime*.go' -o \
        -name 'service_runtime_validation*.go' -o \
        -name 'service_registry*.go' -o \
        -name 'service_profile_bundle*.go' -o \
        -name 'service_editor_bundle*.go' -o \
        -name 'service_rich_content*.go' -o \
        -name 'service_content_docx*.go' -o \
        -name 'service_content_native*.go' -o \
        -name 'service_attachments*.go' -o \
        -name 'capture_renderer_pin*.go' -o \
        -name 'adapters*.go' -o \
        -name 'service_policies*.go' -o \
        -name 'service_core*.go' -o \
        -name 'service_helpers*.go' -o \
        -name 'service_templates*.go' \
        \) 2>/dev/null | sort | sed 's/^/- /'

  echo
  echo "## Handler files to delete"
  find internal/modules/documents/delivery/http -type f \( \
        -name 'handler_ck5*.go' -o \
        -name 'handler_mddm*.go' -o \
        -name 'handler_content.go' -o \
        -name 'handler_attachments.go' -o \
        -name 'handler_runtime.go' -o \
        -name 'handler_telemetry_shadow_diff*.go' -o \
        -name 'image_handler*.go' -o \
        -name 'load_handler*.go' -o \
        -name 'release_handler*.go' -o \
        -name 'submit_for_approval_handler.go' -o \
        -name 'template_admin_handler*.go' -o \
        -name 'create_document_handler*.go' -o \
        -name 'path_helpers.go' -o \
        -name 'handler.go' \
        \) 2>/dev/null | sort | sed 's/^/- /'

  echo
  echo "## Domain files to delete"
  find internal/modules/documents/domain -type f \( \
        -name 'collaboration.go' -o \
        -name 'etapa_body.go' -o \
        -name 'image_storage.go' -o \
        -name 'renderer_pin*.go' -o \
        -name 'rich_envelope.go' -o \
        -name 'schema_runtime*.go' -o \
        -name 'shadow_diff.go' -o \
        -name 'template*.go' \
        \) 2>/dev/null | sort | sed 's/^/- /'

  echo
  echo "## Docker Compose services to remove"
  grep -E '^  (docgen|ck5-export|ck5-studio):' docker-compose.yml 2>/dev/null | sed 's/^/- /'

  echo
  echo "## CI workflows to delete"
  ls .github/workflows/ck5*.yml .github/workflows/mddm*.yml .github/workflows/docgen*.yml 2>/dev/null | sed 's/^/- /'

  echo
  echo "## Migrations to archive (NOT delete — kept for history)"
  ls migrations/00[0-9][0-9]_*ck5* migrations/00[0-9][0-9]_*mddm* migrations/00[0-9][0-9]_*browser_editor* 2>/dev/null | sed 's/^/- /'
} > docs/superpowers/evidence/docx-v2-w5-destruction-census.md
```

- [ ] **Step 2: Human review the census**

Open `docs/superpowers/evidence/docx-v2-w5-destruction-census.md`. For EACH line, confirm: (a) the file/dir is CK5/MDDM/legacy-docgen; (b) nothing in `/api/v2/*`, `internal/modules/documents_v2/`, `internal/modules/templates_v2/`, `apps/docgen-v2/`, or `frontend/apps/web/src/features/documents/v2/` imports from it. Use `rtk grep` if uncertain.

If any line is ambiguous, move it to a new section `## AMBIGUOUS — needs pre-destruction investigation` and DO NOT proceed to Task 6 until resolved.

- [ ] **Step 3: Commit the census**

```bash
rtk git add docs/superpowers/evidence/docx-v2-w5-destruction-census.md
rtk git commit -m "evidence(docx-v2/w5): destruction census — files + services to delete"
```

---

## Task 5: Break compile-time dependencies on CK5 from the canonical tree

**Goal:** Before `git rm -r` lands, make sure the canonical tree (`/api/v2/*`, `documents_v2`, `templates_v2`, `docgen-v2`, `documents/v2/` frontend) does NOT import ANY symbol from the condemned tree. If the canonical tree compiles without CK5, Task 6's `git rm -r` becomes a mechanical change; otherwise it would break main.

**Files:** none created; `rtk grep` is the working tool.

- [ ] **Step 1: Grep for imports that cross the kill boundary**

```bash
# Go — any documents_v2/templates_v2 package importing legacy paths.
rtk grep -E 'internal/modules/documents(/|")' \
  internal/modules/documents_v2/ internal/modules/templates_v2/ apps/api/
# Expected: 0 matches. If ANY match, triage:
#   (a) If the import is to `internal/modules/documents/domain/errors.go` or another
#       shared primitive, relocate it to internal/shared/ before Task 6.
#   (b) If the import is a real cross-module dependency, the canonical code is
#       not yet independent — STOP, fix the dependency, loop back.

# Frontend — any /v2/ feature importing from /ck5/.
rtk grep -E "from ['\"].*documents/ck5" frontend/apps/web/src/features/documents/v2/
# Expected: 0 matches.

# API wiring — main.go + permissions.go must not reference ck5-* handlers.
rtk grep -E '(ck5|mddm)' apps/api/cmd/metaldocs-api/main.go apps/api/cmd/metaldocs-api/permissions.go
# Expected: 0 matches (except possibly a commented-out TODO — remove those now).

# docker-compose — docgen-v2 service must NOT depend_on ck5-*/docgen.
rtk grep -A2 -B2 'depends_on' docker-compose.yml | rtk grep -E '(docgen|ck5)'
# Expected: every docgen-v2 depends_on row references gotenberg / minio / postgres ONLY.
```

- [ ] **Step 2: Relocate any shared primitives**

For each Go file the canonical tree imports out of the condemned tree, do ONE of:

1. **Move** to `internal/shared/<pkg>/` if genuinely cross-cutting. Update imports in both the canonical file and the condemned file (the condemned file is about to be deleted, but its imports must compile until Task 6).
2. **Duplicate-and-diverge** into `internal/modules/documents_v2/domain/` if the semantic is `_v2`-specific. Delete the import from the canonical side; leave the old copy in the condemned tree.
3. **Inline** if trivial (< 20 LOC).

Run `go build ./...` + `npm run build --workspace @metaldocs/web` between each relocation. No test additions in this task — the behavior being changed is "canonical tree is no longer coupled to CK5", which is proven by the Step 1 greps returning empty AND the builds going green.

- [ ] **Step 3: Commit per primitive**

Each relocation is a separate commit, small enough to revert if `go build` breaks:

```bash
rtk git add internal/shared/<pkg>/ internal/modules/<canonical>/...
rtk git commit -m "refactor(docx-v2/w5): move <pkg> into internal/shared for decoupling"
```

Loop Step 1 until all four greps return 0 matches.

---

## Task 6: `git rm -r` the condemned tree — one reviewable commit

**Goal:** Atomically delete every file enumerated in Task 4's census. One commit. Reviewable diff. Pre-condition: Task 5 proved the canonical tree is independent; Task 4's census is signed off; Tasks 0 + 1 + 2 + 3 all green.

**Files:** all files listed in `docs/superpowers/evidence/docx-v2-w5-destruction-census.md`.

- [ ] **Step 1: Re-run Task 5 Step 1 greps to confirm 0 matches**

If any grep has a non-empty result, STOP — go back to Task 5.

- [ ] **Step 2: Delete**

```bash
# Apps
rtk git rm -r apps/ck5-export apps/ck5-studio apps/docgen

# Frontend features
rtk git rm -r frontend/apps/web/src/features/documents/ck5
# MDDM features (if any survived decomposition):
rtk git rm -r frontend/apps/web/src/features/mddm 2>/dev/null || true

# Backend Go — follow the census exactly. The census already resolved ambiguity.
# Use `xargs git rm` with the census as input; keep the census committed as evidence.
awk '/^## Go files to delete/{p=1;next} /^## Handler files to delete/{p=1;next} /^## Domain files to delete/{p=1;next} /^## /{p=0} p && /^- /{sub(/^- /,""); print}' \
  docs/superpowers/evidence/docx-v2-w5-destruction-census.md \
  | xargs -r rtk git rm

# Docker Compose — edit by hand, do not rm the file.
# Remove docgen, ck5-export, ck5-studio service blocks; keep docgen-v2 + gotenberg + minio + postgres.

# CI workflows
rtk git rm .github/workflows/ck5*.yml .github/workflows/mddm*.yml .github/workflows/docgen*.yml 2>/dev/null || true
```

- [ ] **Step 3: Verify builds**

```bash
go build ./...
# Expected: clean. If errors, they'll name the remaining import site — fix by
# either adding to the rm list (if also condemned) or relocating per Task 5 Step 2.

go vet ./...
# Expected: no unused-import errors.

npm run build --workspace @metaldocs/web
# Expected: clean.

npm run build --workspace @metaldocs/docgen-v2
# Expected: clean (no change expected; docgen-v2 has no CK5 deps).
```

- [ ] **Step 4: Verify test suite**

```bash
go test ./...
# Expected: every remaining test passes. Tests on condemned files were rm'd
# with their subjects; no orphan.

npm test --workspace @metaldocs/web
# Expected: clean.
```

- [ ] **Step 5: Commit — one big destructive commit**

```bash
rtk git add -u
rtk git commit -m "feat(docx-v2/w5): destroy CK5 + MDDM + legacy-docgen (commit of record)"
```

The commit body should inline the Task 4 census so reviewers see the kill list in-commit without opening a separate artifact. Use:

```bash
rtk git commit --amend -F - <<'EOF'
feat(docx-v2/w5): destroy CK5 + MDDM + legacy-docgen (commit of record)

Removes, in one atomic commit, every file enumerated in
docs/superpowers/evidence/docx-v2-w5-destruction-census.md. The canonical
tree (documents_v2 / templates_v2 / docgen-v2 / /api/v2) was proven
independent by Task 5's four greps returning 0 matches before this
commit was made.

Post-conditions verified:
  - go build ./...                              clean
  - go vet ./...                                clean
  - go test ./...                               pass
  - npm run build --workspace @metaldocs/web    clean
  - npm test --workspace @metaldocs/web         pass

Rollback: `git revert <this-sha>` restores the tree; DB state is
independently governed by migration 0113 (destructive) + Task 10 rollback
kit (pg_dump restore).

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
```

---

## Task 7: Promote `documents_v2` / `templates_v2` / `/api/v2` → canonical

**Goal:** Rename the `_v2` Go packages back to their unversioned names now that the legacy packages are gone. Frontend `features/documents/v2/` becomes `features/documents/`. HTTP paths stay at `/api/v2/*` — the `v2` is now the API contract version, not a migration scaffold, so renaming would be breaking the external contract for no benefit.

**Decision recorded:** We are KEEPING `/api/v2/*` as the canonical URL namespace. External callers (the tenant web app, future API consumers) rely on the string. Renaming to `/api/v1/*` would be a breaking change with no observable benefit; renaming to `/api/` (unversioned) forfeits our ability to add `/api/v3/*` in the future. `/api/v2/*` it is. Only internal code moves.

**Files:**
```
internal/modules/documents_v2/   → internal/modules/documents/
internal/modules/templates_v2/   → internal/modules/templates/
frontend/apps/web/src/features/documents/v2/  → frontend/apps/web/src/features/documents/
frontend/apps/web/src/features/templates/v2/  → frontend/apps/web/src/features/templates/
api/openapi/v1/partials/documents-v2.yaml     → api/openapi/v1/partials/documents.yaml
api/openapi/v1/partials/templates-v2.yaml     → api/openapi/v1/partials/templates.yaml
```

- [ ] **Step 1: `git mv` the directories**

```bash
rtk git mv internal/modules/documents_v2 internal/modules/documents
rtk git mv internal/modules/templates_v2 internal/modules/templates

rtk git mv frontend/apps/web/src/features/documents/v2 frontend/apps/web/src/features/documents_new
rtk git mv frontend/apps/web/src/features/documents_new frontend/apps/web/src/features/documents
# Two-step because git is case-preserving and on case-insensitive filesystems
# the intermediate rename avoids collision with the (now-deleted) ck5/ path.
# Skip the two-step on Linux/macOS case-sensitive FS.

rtk git mv frontend/apps/web/src/features/templates/v2 frontend/apps/web/src/features/templates_new
rtk git mv frontend/apps/web/src/features/templates_new frontend/apps/web/src/features/templates

rtk git mv api/openapi/v1/partials/documents-v2.yaml api/openapi/v1/partials/documents.yaml
rtk git mv api/openapi/v1/partials/templates-v2.yaml api/openapi/v1/partials/templates.yaml
```

- [ ] **Step 2: Update imports (mechanical, via sed or an IDE)**

Go:

```bash
# Rewrite every import `internal/modules/documents_v2` → `internal/modules/documents`
find internal apps tests -type f -name '*.go' -print0 | \
  xargs -0 rtk sed -i 's|internal/modules/documents_v2|internal/modules/documents|g'
find internal apps tests -type f -name '*.go' -print0 | \
  xargs -0 rtk sed -i 's|internal/modules/templates_v2|internal/modules/templates|g'

# Update the package declaration line where it differs.
rtk grep -rn 'package documents_v2' internal apps tests
# Expected: 0 results after bulk rewrite; package names in /documents/ subtrees already said `documents_v2` so we need:
find internal/modules/documents internal/modules/templates -type f -name '*.go' -print0 | \
  xargs -0 rtk sed -i 's|^package documents_v2$|package documents|; s|^package templates_v2$|package templates|'
```

Frontend:

```bash
# TS imports
find frontend/apps/web -type f \( -name '*.ts' -o -name '*.tsx' \) -print0 | \
  xargs -0 rtk sed -i 's|features/documents/v2|features/documents|g; s|features/templates/v2|features/templates|g'
```

OpenAPI:

```bash
rtk sed -i 's|documents-v2\.yaml|documents.yaml|g; s|templates-v2\.yaml|templates.yaml|g' \
  api/openapi/v1/openapi.yaml
```

- [ ] **Step 3: Rename symbols with `_v2` suffix ONLY if they're internal + trivial**

Do NOT bulk-rewrite `v2` out of route strings, OpenAPI paths, or external contract surfaces. DO:

- Rename Go struct `DocumentV2Service` → `DocumentService` if (a) unused in tests importing the old name and (b) not referenced in external docs.
- Rename React component `DocumentEditorPageV2` → `DocumentEditorPage`.

Do NOT:
- Touch `/api/v2/*` route literals in server routers or OpenAPI paths.
- Touch the `v2` in metric names, audit action constants, or DB feature-flag keys that have escaped to dashboards.

Use the grep output from:

```bash
rtk grep -rE '(V2|_v2|v2)' internal apps frontend --glob '!**/node_modules/**'
```

to build a case-by-case list. Apply case-by-case.

- [ ] **Step 4: Verify builds + tests**

```bash
go build ./...
go test ./...
npm run build --workspace @metaldocs/web
npm test --workspace @metaldocs/web
# All expected: clean.
```

- [ ] **Step 5: Commit per rename, or as a single "rename" commit if small enough**

```bash
rtk git add -A
rtk git commit -m "refactor(docx-v2/w5): rename documents_v2 / templates_v2 / features/v2 → canonical paths"
```

---

## Task 8: Feature flag removal — code-only (no DB migration)

**Goal:** Strip every read of `METALDOCS_DOCX_V2_ENABLED` / `DocxV2Enabled` from Go + TS + deploy-config. Post-removal, the route wiring unconditionally serves `/api/v2/*`; there is nothing to disable and the `GET /api/v1/feature-flags` endpoint no longer returns the key.

**Why code-only (no SQL migration):** Plan A Task 9 implemented the flag as a process-level env var read into `config.FeatureFlagsConfig.DocxV2Enabled` — NOT a row in a `feature_flags` table. There is no SQL to run; removal is entirely a code change + a deploy-config cleanup pass (Helm / .env templates / Kubernetes manifests / CI secrets).

**Files:**
- Delete: `apps/api/internal/platform/featureflag/docx_v2.go` (and its test file) if the scaffold placed it here
- Modify: `apps/api/internal/config/feature_flags.go` (remove `DocxV2Enabled` field + env binding)
- Modify: `apps/api/internal/delivery/http/handler_feature_flags.go` (drop key from response)
- Modify: `frontend/apps/web/src/lib/featureFlags.ts` (drop DOCX_V2 export + callers)
- Modify: all Go/TS call sites found by grep (Step 2)
- Modify: deploy-config surfaces — Helm values, .env templates, Kubernetes manifests, CI env blocks (Step 4)

- [ ] **Step 1: Inventory call sites**

```bash
rtk grep -rn 'METALDOCS_DOCX_V2_ENABLED\|DocxV2Enabled\|featureflag\.DocxV2\|FeatureDocxV2Enabled\|DOCX_V2_ENABLED\|docxV2Enabled' \
  apps internal frontend .github .env.example charts deploy --glob '!**/node_modules/**' \
  > /tmp/docx-v2-flag-callsites.txt
wc -l /tmp/docx-v2-flag-callsites.txt
# Expected: > 0 matches across Go config, Go handler wiring, TS client, deploy configs.
```

- [ ] **Step 2: Strip Go reads**

For each Go match:
- **Config struct field:** delete `DocxV2Enabled bool` from `config.FeatureFlagsConfig` + the matching `envconfig:"METALDOCS_DOCX_V2_ENABLED"` tag line.
- **Handler/middleware guard:** if the guard wraps an if/else that enables the v2 path when true, delete the guard; keep the v2 branch; delete the v1 branch (which Task 6 already removed — grep should confirm).
- **Full-middleware guard:** delete the middleware from the router; the route now serves unconditionally.
- **`GET /api/v1/feature-flags` response:** delete the `METALDOCS_DOCX_V2_ENABLED` key from the JSON response map. Update the handler test's golden JSON.
- **Tests proving flag-off behavior:** delete them entirely.
- **Tests proving flag-on behavior:** simplify — drop the setter, drop the assertion on the flag state, keep the behavior assertion.

- [ ] **Step 3: Strip TS reads**

```bash
rtk grep -rn 'DOCX_V2\|docxV2\|docx_v2_enabled' frontend --glob '!**/node_modules/**'
```

For each TS match: delete the export from `frontend/apps/web/src/lib/featureFlags.ts`; remove every import + guarded branch; keep the v2 path unconditionally.

- [ ] **Step 4: Strip deploy-config surfaces**

Grep every deploy-config file for the env var and remove it:

```bash
rtk grep -rn 'METALDOCS_DOCX_V2_ENABLED' charts deploy .github/workflows .env.example docker-compose.yml 2>/dev/null
```

For each match: delete the key + value line. Deploy-config surfaces typically include:
- `charts/metaldocs/values.yaml` (Helm)
- `charts/metaldocs/templates/deployment.yaml` (Helm manifest envFrom)
- `deploy/kubernetes/*.yaml` (direct manifests)
- `.env.example` (local dev template)
- `docker-compose.yml` (service `environment:` blocks — these were already trimmed in Task 6 but the key may survive)
- `.github/workflows/ci.yml` (CI env block, if the test suite set the flag)

- [ ] **Step 5: Build + test**

```bash
go build ./...
go vet ./...
go test -race ./...
npm run build --workspace @metaldocs/web
npm test --workspace @metaldocs/web
# All: clean. The handler_feature_flags test's golden JSON must have been updated.
```

- [ ] **Step 6: Re-verify zero call sites**

```bash
rtk grep -rn 'METALDOCS_DOCX_V2_ENABLED\|DocxV2Enabled\|featureflag\.DocxV2\|FeatureDocxV2Enabled\|DOCX_V2_ENABLED\|docxV2Enabled' \
  apps internal frontend .github .env.example charts deploy 2>/dev/null
# Expected: 0 matches.
```

- [ ] **Step 7: Commit**

```bash
rtk git add -A
rtk git commit -m "refactor(docx-v2/w5): remove METALDOCS_DOCX_V2_ENABLED — flag is now permanent (code + deploy-config)"
```

**Operator follow-up (NOT a commit):** after this commit deploys, operators MUST remove `METALDOCS_DOCX_V2_ENABLED` from every cluster's live secret / configmap / pipeline secret store so a future re-introduction of the same key name cannot silently reactivate dead guards. Tracked in `docs/superpowers/evidence/docx-v2-w5-destruction-receipt.md` as a checkbox.

---

## Task 9: CI consolidation

**Goal:** Delete CK5 / MDDM / legacy-docgen workflows; fold `docx-v2-ci.yml` into the primary `ci.yml` (or rename if no `ci.yml` exists). The resulting workflow runs a single pipeline per PR.

**Files:**
- Delete: `.github/workflows/ck5-ci.yml` (if present)
- Delete: `.github/workflows/mddm-ci.yml` (if present)
- Delete: `.github/workflows/docgen-ci.yml` (if present)
- Rename: `.github/workflows/docx-v2-ci.yml` → `.github/workflows/ci.yml` (or merge into existing `ci.yml`)

- [ ] **Step 1: Delete dead workflows**

```bash
rtk git rm .github/workflows/ck5-ci.yml .github/workflows/mddm-ci.yml .github/workflows/docgen-ci.yml 2>/dev/null || true
ls .github/workflows/
```

- [ ] **Step 2: Rename / merge**

If there is no existing `.github/workflows/ci.yml`, rename:

```bash
rtk git mv .github/workflows/docx-v2-ci.yml .github/workflows/ci.yml
```

If there IS an existing `ci.yml` (e.g. a generic lint job), merge the `e2e-documents`, `e2e-exports`, `w5-cutover-gate` jobs into it and delete `docx-v2-ci.yml` with `rtk git rm`.

- [ ] **Step 3: Strip now-permanent build-tag jobs**

The `w5_cutover_gate` test exists to keep the post-flip soak log honest. After the destruction commits land, the gate has served its purpose. Two options:

- **Keep the job** as archaeology (the build-tag test still passes on a filled log; it costs one CI second).
- **Remove the job** now that the cutover is irreversible.

**Default decision: remove.** The soak is complete; the gate is no longer load-bearing.

```yaml
# Remove these jobs from ci.yml:
# - detect-w5-artifacts
# - w5-cutover-gate
# - detect-w4-changes     (the W4 dogfood gate has also served its purpose)
# - w4-gate / w5-gate
```

Also delete the now-inert test files:

```bash
rtk git rm tests/docx_v2/dogfood_gate_test.go tests/docx_v2/cutover_gate_test.go
```

- [ ] **Step 4: Verify CI passes on a throwaway branch**

Push to a test branch; confirm the renamed `ci.yml` triggers and runs the full suite. Document the test branch URL in the W5 runbook before deleting the branch.

- [ ] **Step 5: Commit**

```bash
rtk git add -A
rtk git commit -m "ci(docx-v2/w5): consolidate workflows; drop CK5/MDDM/docgen + retired gates"
```

---

## Task 10: Rollback kit — `scripts/w5-rollback.sh` + runbook

**Goal:** A single, smoke-tested shell script that takes a pg_dump path + a git tag and produces a restored-to-pre-cutover staging environment. This is a BLOCKING prerequisite for Task 3 (destructive migration) and Task 6 (code destruction). The kit must be committed AND smoke-tested on a staging DB before Task 3 applies in prod.

**Files:**
- Create: `scripts/w5-rollback.sh`
- Create: `docs/runbooks/docx-v2-w5-rollback.md`

- [ ] **Step 1: Write the rollback script**

```bash
#!/usr/bin/env bash
# scripts/w5-rollback.sh
# Restore MetalDocs to the pre-W5 state.
#
# WHAT THIS DOES:
#   1. Parses an EXPLICIT --env <staging|production> flag (required).
#   2. Refuses to run against production unless --force-production ALSO passed.
#   3. Verifies the target git tag exists.
#   4. Prints a preflight summary and requires the operator to type YES.
#   5. Restores the pg_dump into a FRESH database (destroys the target DB).
#   6. Prints the git commands the operator must run manually to re-deploy code.
#
# WHAT THIS DOES NOT DO:
#   - It does not run `git reset --hard` or `git push --force`. The operator
#     performs those under change-management supervision. This script prints
#     the exact commands.
#   - It does not restore S3/MinIO blobs. Blob state is additive and content-
#     addressed; rolling back the DB leaves orphan blobs, which is safe.
#   - It does not flip the METALDOCS_DOCX_V2_ENABLED env var back to `false`.
#     That is a deploy-config change; the runbook enumerates the manual step.
#
# Usage:
#   ./scripts/w5-rollback.sh --dump <file> --tag <git-tag> --env <staging|production> [--force-production]
#
# Env expected (exported, NOT passed as args, so accidental shell history
# disclosure does not leak secrets):
#   PGHOST PGUSER PGPASSWORD PGPORT PGDATABASE   — target DB

set -euo pipefail

DUMP=""
TAG="w5-preflight"
ENV_TARGET=""
FORCE_PROD=0

usage() {
  cat >&2 <<USAGE
usage: $0 --dump <file> --tag <git-tag> --env <staging|production> [--force-production]

  --dump <file>         Path to pg_dump --format=custom artifact (required).
  --tag <git-tag>       Git tag marking pre-W5 code state (default: w5-preflight).
  --env <env>           MUST be 'staging' or 'production'. No default.
  --force-production    Mandatory second flag if --env production. Without it
                        production is hard-denied even with --env production.
USAGE
  exit 2
}

while [ $# -gt 0 ]; do
  case "$1" in
    --dump)              DUMP="${2:-}"; shift 2 ;;
    --tag)               TAG="${2:-}"; shift 2 ;;
    --env)               ENV_TARGET="${2:-}"; shift 2 ;;
    --force-production)  FORCE_PROD=1; shift ;;
    -h|--help)           usage ;;
    *)                   echo "unknown arg: $1" >&2; usage ;;
  esac
done

# Explicit arg validation — no defaults except --tag.
[ -n "${DUMP}" ]       || { echo "--dump is required" >&2; usage; }
[ -n "${ENV_TARGET}" ] || { echo "--env is required (staging|production)" >&2; usage; }
[ -f "${DUMP}" ]       || { echo "dump file not found: ${DUMP}" >&2; exit 1; }

case "${ENV_TARGET}" in
  staging)    : ;;
  production)
    if [ "${FORCE_PROD}" -ne 1 ]; then
      echo "REFUSED: --env production requires --force-production to proceed." >&2
      echo "This is a destructive cross-environment DB restore." >&2
      exit 3
    fi
    ;;
  *)
    echo "--env must be 'staging' or 'production' (got: ${ENV_TARGET})" >&2
    exit 2
    ;;
esac

if ! git rev-parse --verify "${TAG}" >/dev/null 2>&1; then
  echo "git tag not found: ${TAG}" >&2
  exit 1
fi

# Env sanity: PGHOST / PGDATABASE must be set and must not point at a
# production cluster if --env staging was passed.
: "${PGHOST:?PGHOST must be exported}"
: "${PGDATABASE:?PGDATABASE must be exported}"
if [ "${ENV_TARGET}" = "staging" ] && { [[ "${PGHOST}" == *"prod"* ]] || [[ "${PGHOST}" == *"production"* ]]; }; then
  echo "REFUSED: --env staging but PGHOST looks like production (${PGHOST})." >&2
  exit 3
fi

# Preflight summary + interactive YES gate (skippable only via `yes YES | ...`
# which is still an explicit operator action).
cat <<SUMMARY

================= W5 ROLLBACK PREFLIGHT =================
 env:        ${ENV_TARGET}$([ "${ENV_TARGET}" = "production" ] && echo ' (--force-production)' )
 PGHOST:     ${PGHOST}
 PGDATABASE: ${PGDATABASE}
 dump file:  ${DUMP} ($(wc -c < "${DUMP}") bytes)
 dump sha256: $(sha256sum "${DUMP}" | awk '{print $1}')
 git tag:    ${TAG} ($(git rev-parse --short "${TAG}"))
 action:     dropdb + createdb + pg_restore (DESTRUCTIVE)
=========================================================

Type YES to proceed, anything else to abort:
SUMMARY
read -r CONFIRM
if [ "${CONFIRM}" != "YES" ]; then
  echo "aborted by operator" >&2
  exit 4
fi

TARGET_DB="${PGDATABASE}"
echo ">> Rolling back database ${TARGET_DB} on ${PGHOST} to tag ${TAG}"

dropdb --if-exists "${TARGET_DB}"
createdb "${TARGET_DB}"
pg_restore --dbname="${TARGET_DB}" \
           --no-owner --no-privileges --verbose \
           "${DUMP}"

cat <<EOF

=== Database restored to state at tag ${TAG} ===

Next manual steps for the operator (under change-management):

  # 1. Flip METALDOCS_DOCX_V2_ENABLED back to false on every API/worker/
  #    frontend replica (deploy-config change, not a DB row — see Task 1).
  # 2. On a deploy host, check out the rollback commit:
  git fetch --tags
  git checkout ${TAG}

  # 3. Build + deploy:
  make build            # or: npm run build --workspace @metaldocs/web
  make deploy TARGET=${ENV_TARGET}

  # 4. Verify:
  curl -sS https://<host>/api/v2/templates   # should fail (old routes)
  curl -sS https://<host>/api/v1/templates   # should succeed (CK5 back up)

Rollback window validity: 30 days from dump timestamp. Past that, blob references
may not reconcile. Do NOT restore against a DB with real user writes after the
tag was cut.
EOF
```

Make executable:

```bash
chmod +x scripts/w5-rollback.sh
```

- [ ] **Step 2: Write the rollback runbook**

`docs/runbooks/docx-v2-w5-rollback.md`:

```markdown
# W5 Rollback Runbook

**When to invoke:** A P0 incident during the Task 2 post-flip soak, a failure
during Task 3 migration apply, or a post-Task-6 regression that cannot be
hot-fixed within a single 24h window.

## Decision tree

```
Incident detected
       │
       ▼
Has destructive migration 0113 applied on prod?
       │
       ├── NO  → Flag-only rollback: redeploy API/worker/frontend replicas
       │        with METALDOCS_DOCX_V2_ENABLED=false (the flag is a
       │        process-level env var, NOT a DB row — see Plan A Task 9 +
       │        Plan E Task 1). Verify via
       │        `curl /api/v1/feature-flags` per replica. CK5 code path
       │        wakes back up. No DB restore needed.
       │
       └── YES → Full rollback required:
                 1. Bring all API + worker replicas offline (503 page).
                 2. Run scripts/w5-rollback.sh with the w5-preflight dump.
                 3. Redeploy with METALDOCS_DOCX_V2_ENABLED=false.
                 4. `git checkout w5-preflight` on all deploy hosts.
                 5. Re-deploy.
                 6. Bring back online.
```

## Prerequisites verified before any cutover commit

- [ ] Tag `w5-preflight` exists on origin.
- [ ] pg_dump at $DUMP_PATH has sha256 matching evidence file.
- [ ] Blob store (S3/MinIO) is append-only; no rollback action needed.
- [ ] Staging rehearsal of `w5-rollback.sh` passed (see Task 10 Step 4).

## Full-rollback procedure (step-by-step)

1. Open incident ticket; notify @admin + @sre + @pm in #incident channel.
2. `make maintenance-mode ENABLE=true` — 503 all /api/v2 routes.
3. On deploy host:
   ```bash
   export PGHOST=... PGDATABASE=... PGUSER=... PGPASSWORD=... PGPORT=...
   ./scripts/w5-rollback.sh \
     --dump /secure/backup/metaldocs-w5-preflight-*.dump \
     --tag  w5-preflight \
     --env  production \
     --force-production
   # Interactive: type YES at the preflight gate to proceed.
   ```
4. Flip env var back: redeploy every API/worker/frontend replica with
   `METALDOCS_DOCX_V2_ENABLED=false`. Verify each replica via
   `curl http://<replica>:8080/api/v1/feature-flags`.
5. `git fetch --tags && git checkout w5-preflight`
6. Build + deploy rolled-back image.
7. Smoke test:
   - `curl /api/v1/templates` → 200
   - `curl /api/v2/templates` → 404 or unreachable
8. `make maintenance-mode ENABLE=false`
9. Incident retro within 48h.

## What is NOT rolled back

- Audit log entries created during soak remain (immutable).
- Export PDFs generated during soak remain in S3 (content-addressed; orphans).
- User form-data changes made during soak are LOST if they were made through
  the W5 UI after flip — the dump is from before flip. This is accepted risk;
  the W5 soak window must reject full-rollback once any tenant has used
  /api/v2 for real production data.
```

- [ ] **Step 3: Commit the kit**

```bash
rtk git add scripts/w5-rollback.sh docs/runbooks/docx-v2-w5-rollback.md
rtk git commit -m "feat(docx-v2/w5): rollback kit — w5-rollback.sh + runbook"
```

- [ ] **Step 4: Smoke-test on staging — BLOCKING for Task 3**

```bash
# 1. Take a staging pg_dump (simulates a prod w5-preflight dump).
./scripts/w5-preflight-dump.sh /tmp/w5-staging-test

# 2. Apply migrations 0112 + 0113 on staging. (There is NO 0114 migration —
#    the DocxV2Enabled flag is an env var, removed in Task 8 as code-only.)
export PGHOST=staging.metaldocs.internal PGDATABASE=metaldocs_staging \
       PGUSER=metaldocs PGPASSWORD=*** PGPORT=5432
psql -v ON_ERROR_STOP=1 -f migrations/0112_docx_v2_schema_migrations_ledger.sql
psql -v ON_ERROR_STOP=1 -f migrations/0113_docx_v2_destroy_ck5_tables.sql

# 2b. Idempotency: re-apply 0113; expect no-op COMMIT.
psql -v ON_ERROR_STOP=1 -f migrations/0113_docx_v2_destroy_ck5_tables.sql
# Expected NOTICE: '0113 already recorded in schema_migrations — running as no-op'

# 3. Use the rollback script to bring staging back to pre-0112 state.
git tag w5-staging-test HEAD   # simulated tag
./scripts/w5-rollback.sh \
  --dump /tmp/w5-staging-test/metaldocs-w5-preflight-*.dump \
  --tag  w5-staging-test \
  --env  staging
# Interactive: type YES at the preflight gate.

# 4. Verify: staging DB has all pre-W5 tables back; ledger rows for 0112/0113 are gone.
psql -c "\d template_drafts"            # expected: back
psql -c "SELECT version FROM public.schema_migrations WHERE version IN ('0112','0113')"
#         expected: 0 rows (dump was pre-0112)

# 5. Document the staging rehearsal in docs/superpowers/evidence/docx-v2-w5-preflight.md.
```

If any step fails, STOP — do not advance to Task 3 in prod until the rollback kit is proven.

---

## Task 11: Final regression — full `./...` + Playwright + smoke on staging

**Goal:** Prove, as the last signal before Task 12's close-out, that main is green and deployable after all renames, deletions, and flag removals.

**Files:** none.

- [ ] **Step 1: Full test suite**

```bash
go build ./...
go vet ./...
go test -race ./...
go test -tags=integration ./...
npm run build --workspace @metaldocs/web
npm run build --workspace @metaldocs/docgen-v2
npm test --workspace @metaldocs/web
npm test --workspace @metaldocs/docgen-v2
npx playwright test --project=chromium
```

All: green.

- [ ] **Step 2: OpenAPI lint**

```bash
npx @redocly/cli lint api/openapi/v1/openapi.yaml
# Expected: 0 errors, 0 warnings.
```

- [ ] **Step 3: Staging smoke**

Deploy main to staging. Run:

```bash
STAGING_HOST=staging.metaldocs.example.com

# Templates
curl -fsS -H "Authorization: Bearer $STAGING_TOKEN" \
  https://$STAGING_HOST/api/v2/templates | jq '.items | length'
# Expected: >= 1 (seed templates present).

# Create document → autosave → export
# Walk the filler-happy-path E2E manually or via the existing Playwright spec
# pointed at staging.

# CK5 legacy routes
curl -sS -o /dev/null -w '%{http_code}\n' https://$STAGING_HOST/api/v1/templates
# Expected: 404 (route removed by Task 6).
```

- [ ] **Step 4: Write the destruction receipt**

`docs/superpowers/evidence/docx-v2-w5-destruction-receipt.md`:

```markdown
# W5 Destruction Receipt

Cutover completion date (UTC): YYYY-MM-DDTHH:MM:SSZ
Operator: @handle

## Commits
- Flag flip (env-var rollout, per-replica verification): <sha>
- Post-flip soak gate: <sha>
- Ledger bootstrap + destructive migration (0112+0113): <sha>
- Destruction commit of record: <sha>
- Rename commit: <sha>
- Flag removal (code + deploy-config, no SQL): <sha>
- CI consolidation: <sha>

## Tags
- `w5-preflight` → <sha>  (rollback target)
- `w5-complete`  → <sha>  (current main after Task 11)

## Verification
- [ ] `go test -race ./...` pass at `w5-complete`
- [ ] `npm run build + test` pass at `w5-complete`
- [ ] OpenAPI lint 0 errors
- [ ] Staging smoke passed (links to dashboards)
- [ ] Production smoke passed (links to dashboards)
- [ ] No CK5 / MDDM / docgen (legacy) files remain in the tree:
  `rtk grep -rn 'ck5\|mddm' internal apps frontend --glob '!**/archive/**'` → 0 matches.

## Sign-off
- [ ] Admin: @handle — YYYY-MM-DD
- [ ] SRE: @handle — YYYY-MM-DD
- [ ] Product manager: @handle — YYYY-MM-DD
```

```bash
rtk git add docs/superpowers/evidence/docx-v2-w5-destruction-receipt.md
rtk git commit -m "evidence(docx-v2/w5): cutover destruction receipt (pre-sign-off)"
```

The receipt is filled after Step 3 passes; sign-offs come from humans in a follow-up commit.

- [ ] **Step 5: Tag `w5-complete`**

```bash
rtk git tag w5-complete
rtk git push origin w5-complete
```

---

## Task 12: Docs + CLAUDE.md cleanup + close-out

**Goal:** Remove every dangling reference to CK5 / MDDM / docgen-legacy from project-level documentation. Archive the legacy plans + specs under `docs/superpowers/archive/` so history is preserved but not indexed by agent searches. Write a CHANGELOG entry.

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`
- Create: `docs/CHANGELOG.md` (if missing)
- Move: legacy plans + specs → `docs/superpowers/archive/`

- [ ] **Step 1: Strip CK5/MDDM from CLAUDE.md**

Identify every paragraph, bullet, or rule in `CLAUDE.md` that references CKEditor, CK5, MDDM, block rendering, restricted-editing, nested tables, the template engine, or "docgen must be restarted". Each of these was relevant to the retired architecture and is now harmful — it would nudge future sessions toward dead code.

```bash
rtk grep -n 'CK5\|CKEditor\|MDDM\|restricted.editing\|blocks_json\|docgen must be\|ck5' CLAUDE.md
```

For each match, decide: delete the rule, rewrite it to point at docx-v2 analog, or move to `docs/superpowers/archive/CLAUDE-legacy.md` for historical reference.

- [ ] **Step 2: Rewrite README Architecture**

The README section describing the old service topology (CKEditor + CK5 plugins + docgen) must be replaced with the docx-v2 topology (API + docgen-v2 + Gotenberg + docx-editor frontend). Use the ASCII diagram from `docs/superpowers/specs/2026-04-18-docx-editor-platform-design.md` §Service topology as source-of-truth.

- [ ] **Step 3: Archive legacy plans/specs**

```bash
mkdir -p docs/superpowers/archive/
rtk git mv \
  docs/superpowers/plans/2026-04-01-*.md \
  docs/superpowers/plans/2026-04-02-*.md \
  docs/superpowers/plans/2026-04-04-*.md \
  docs/superpowers/plans/2026-04-06-*.md \
  docs/superpowers/plans/2026-04-07-*.md \
  docs/superpowers/plans/2026-04-08-*.md \
  docs/superpowers/plans/2026-04-09-*.md \
  docs/superpowers/plans/2026-04-10-*.md \
  docs/superpowers/plans/2026-04-12-*.md \
  docs/superpowers/plans/2026-04-13-*.md \
  docs/superpowers/plans/2026-04-14-*.md \
  docs/superpowers/plans/2026-04-15-ck5-*.md \
  docs/superpowers/plans/2026-04-16-ck5-*.md \
  docs/superpowers/plans/2026-04-17-ck5-*.md \
  docs/superpowers/plans/2026-04-18-ck5-*.md \
  docs/superpowers/archive/
# Keep: 2026-04-18-docx-editor-*.md (the canonical W1-W5 record).
```

- [ ] **Step 4: Write CHANGELOG entry**

`docs/CHANGELOG.md` (append):

```markdown
## 2026-04-18 — docx-editor platform cutover (W5)

### Added
- `@eigenpal/docx-js-editor`-based editor under `/api/v2/*`.
- Content-addressed revision + export pipeline (docgen-v2 + Gotenberg).
- Pessimistic-lock editor sessions + autosave-crash recovery.
- Per-route rate-limit middleware.
- W4 dogfood + W5 post-flip soak evidence artifacts.

### Removed
- `apps/ck5-export`, `apps/ck5-studio`, `apps/docgen` (legacy).
- CKEditor 5 frontend feature tree.
- MDDM Postgres tables + Go service/handler files.
- `METALDOCS_DOCX_V2_ENABLED` env var + `DocxV2Enabled` config field (now permanent behavior).

### Breaking changes
- `/api/v1/documents/*` and `/api/v1/templates/*` removed. External callers
  must use `/api/v2/*` paths.
- DB tables dropped: (full list in migration 0113).

### Rollback
- Tag `w5-preflight` preserves the pre-cutover tree + DB snapshot path.
- `scripts/w5-rollback.sh` + `docs/runbooks/docx-v2-w5-rollback.md`.
```

- [ ] **Step 5: Commit**

```bash
rtk git add CLAUDE.md README.md docs/CHANGELOG.md docs/superpowers/archive/
rtk git commit -m "docs(docx-v2/w5): strip CK5/MDDM from CLAUDE.md + README; archive legacy plans; CHANGELOG"
```

---

## Sanity pass

- [ ] `go build ./...` passes at `w5-complete`.
- [ ] `go vet ./...` clean.
- [ ] `go test -race ./...` passes.
- [ ] `go test -tags=integration ./...` passes.
- [ ] `npm run build --workspace @metaldocs/web` passes.
- [ ] `npm test --workspace @metaldocs/web` passes.
- [ ] `npm run build --workspace @metaldocs/docgen-v2` passes.
- [ ] `npx playwright test --project=chromium` passes.
- [ ] `npx @redocly/cli lint api/openapi/v1/openapi.yaml` reports 0 errors.
- [ ] `rtk grep -rn 'ck5\|mddm\|CKEditor' internal apps frontend --glob '!**/archive/**' --glob '!**/node_modules/**'` returns 0 matches.
- [ ] `rtk grep -rn 'METALDOCS_DOCX_V2_ENABLED\|DocxV2Enabled\|featureflag\.DocxV2\|FeatureDocxV2Enabled\|docx_v2_enabled' . --glob '!**/archive/**' --glob '!**/node_modules/**' --glob '!docs/CHANGELOG.md'` returns 0 matches.
- [ ] `rtk grep -rn 'documents_v2\|templates_v2' internal apps frontend` returns 0 matches.
- [ ] Every Task 0-12 commit is green in CI.
- [ ] Tags `w5-preflight` + `w5-complete` pushed to origin.
- [ ] Destruction receipt filled + signed off by admin, SRE, PM.

---

## Codex Hardening Log

### R1

- **verdict:** `APPROVE_WITH_FIXES`
- **mode:** `OPERATIONS`
- **upgrade_required:** `true`
- **confidence:** `high`
- **issues (5):**
  1. **STRUCTURAL — Soak gate not time-enforced.** `w5_cutover_gate` regex accepts any 5 `### Day N — YYYY-MM-DD` blocks whose dates fall in any order; an operator could backfill all 5 day entries on the same day and pass the gate.
     - **Applied Fix 1:** `tests/docx_v2/cutover_gate_test.go` now parses a `Flag flip applied (UTC): YYYY-MM-DDT...` header line, extracts each Day's date, enforces **(a)** strict chronological order, **(b)** all dates ≥ flip date, **(c)** no future dates, **(d)** a **≥ 5 business days inclusive** span via a new `businessDaysInclusive(from, to)` helper. A backfill-in-a-single-day log cannot pass.
  2. **STRUCTURAL — Migration ledger sync missing.** The plan applied `psql -f` directly with no `schema_migrations` row. A re-run after success would re-fire sentinels and fail; also, nothing records the apply-state for audit.
     - **Applied Fix 2:** Added migration `0112_docx_v2_schema_migrations_ledger.sql` (bootstraps `public.schema_migrations(version PK, applied_at, description)`) and rewrote `0113` to (a) consult the ledger at the top of every DO block — `IF EXISTS (SELECT 1 FROM schema_migrations WHERE version='0113') THEN RETURN; END IF;` — so a re-run is a clean no-op, and (b) atomically record its own apply with `INSERT ... ON CONFLICT DO NOTHING` before `COMMIT`. The DROPs are now wrapped in a single DO block with `EXECUTE format('DROP TABLE IF EXISTS public.%I CASCADE', t)` so the ledger guard covers them.
  3. **STRUCTURAL — 0114 conditionally drops `feature_flags`.** The original plan's `0114_docx_v2_remove_feature_flag.sql` deleted a row and conditionally `DROP TABLE feature_flags`, which Codex flagged as a cross-tenant / cross-feature blast radius.
     - **REDIRECTED (mechanism mismatch):** Plan A Task 9 implemented the flag as a **process-level env var** `METALDOCS_DOCX_V2_ENABLED` read into `config.FeatureFlagsConfig.DocxV2Enabled`, NOT a `feature_flags` table row. There is no SQL table to drop. The `0114` migration was **removed entirely** from Plan E. Task 8 is rewritten as a code-only + deploy-config removal of the field, the env-var key, the handler response key, and every call site, followed by a build-verification grep expecting zero matches. The CI `paths-filter` list and the destruction receipt were updated to match.
  4. **LOCAL — Production guard too weak.** The original `w5-rollback.sh` used a `$PGHOST == *prod*` substring check, which is skip-able on a host named `prod-readonly-snapshot` and which also incorrectly fires on staging hosts named `metaldocs-prod-stag` etc.
     - **Applied Fix 4:** Rewrote `w5-rollback.sh` with explicit `--env <staging|production>` flag (required, no default), additional `--force-production` flag (mandatory if `--env production`), an interactive preflight summary (env / host / DB / dump sha256 / tag sha / action) that requires typing `YES` to proceed, and cross-environment validation (refuses `--env staging` if PGHOST looks like production). The rollback runbook's Full Procedure + Decision Tree were updated to pass `--env production --force-production` and to mention the YES gate.
  5. **STRUCTURAL — 0112 per-tenant flag coverage.** The original plan's `0112` migration was claimed to set the flag true per-tenant in a `feature_flags` table with `tenant_id`; Codex flagged that missing tenants would silently retain the CK5 path.
     - **REDIRECTED (mechanism mismatch):** Same root cause as finding 3 — there is no per-tenant row. Risk is replaced by the new Task 1, which iterates every API / worker / frontend **replica** via `kubectl exec` + `curl /api/v1/feature-flags`, and the post-flip evidence artifact `docs/superpowers/evidence/docx-v2-w5-flip-verification.md` tables per-replica boolean outputs. Missing replicas are caught at the tabular verification step before the soak clock in Task 2 starts.

### R2

- **verdict:** `APPROVE_WITH_FIXES`
- **mode:** `OPERATIONS`
- **upgrade_required:** `true`
- **confidence:** `high`
- **issues (2, both STRUCTURAL, both applied inline):**
  1. **Flag-flip verification was API-only in practice.** Task 1 Step 3 only asserted per-replica flag truth via `GET /api/v1/feature-flags`, which doesn't reach worker replicas (no HTTP surface) or frontend replicas (may bake flag at build time). A partial rollout (API flipped, worker on old env) produces split-brain behavior during soak and invalidates the GO/NO-GO signal.
     - **Applied R2 Fix 1:** Task 1 Step 3 is now split into **3a (API — `GET /api/v1/feature-flags`)**, **3b (worker — `kubectl exec printenv METALDOCS_DOCX_V2_ENABLED` + image-tag parity)**, and **3c (frontend — `printenv` + `/config.json` asset-build sanity)**. The verification artifact `docs/superpowers/evidence/docx-v2-w5-flip-verification.md` gains three tables (API / Worker / Frontend, per environment) and a six-checkbox attestation block covering all three classes in staging AND prod. The per-class attestation is now a BLOCKER for the Task 1 → Task 2 transition.
  2. **Task 2 soak anchor contradicted its own dependencies.** The original Task 2 Step 1 prose said "filled daily for 5 business days after migration 0112 applies in production", but Task 2 blocks Task 3, and Task 3 is where 0112 applies. This created a circular dependency and weakened the soak-start signal.
     - **Applied R2 Fix 2:** Task 2 Step 1 now anchors the soak window to **"the Task 1 production flag-flip timestamp"** (recorded as "Prod flip UTC" in `docx-v2-w5-flip-verification.md`). Explicit note: migrations 0112/0113 do NOT start the soak clock; Task 1 does. The cutover gate test comment `// Flip timestamp header: ...` is updated from "(migration 0112)" to "(env-var rollout — see Task 1 evidence)" to match. Task 3's prerequisites (Task 2 gate green + rollback rehearsal + fresh dump) are unchanged.

### Final note

Per co-plan protocol, max 2 Codex rounds total. No further rounds will be triggered. Both R2 findings were structural and both were applied inline above, so no unresolved caveats remain from Codex feedback. The plan is considered hardened and ready for execution handoff.

---
