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

: "${PGHOST:?PGHOST must be exported}"
: "${PGDATABASE:?PGDATABASE must be exported}"
if [ "${ENV_TARGET}" = "staging" ] && { [[ "${PGHOST}" == *"prod"* ]] || [[ "${PGHOST}" == *"production"* ]]; }; then
  echo "REFUSED: --env staging but PGHOST looks like production (${PGHOST})." >&2
  exit 3
fi

cat <<SUMMARY

================= W5 ROLLBACK PREFLIGHT =================
 env:        ${ENV_TARGET}$([ "${ENV_TARGET}" = "production" ] && echo ' (--force-production)' || true)
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
