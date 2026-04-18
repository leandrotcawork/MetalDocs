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
