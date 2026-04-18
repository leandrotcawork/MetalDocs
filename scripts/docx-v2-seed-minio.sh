#!/usr/bin/env bash
set -euo pipefail

# Seeds MinIO with the docx-v2 tenants bucket + verifies access.
# Uses the mc client already vendored in the minio/mc image via docker run.

MINIO_HOST="${MINIO_HOST:-http://minio:9000}"
MINIO_ACCESS_KEY="${MINIO_ROOT_USER:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_ROOT_PASSWORD:-minioadmin}"
BUCKET="${DOCX_V2_BUCKET:-metaldocs-docx-v2}"
NETWORK="${COMPOSE_NETWORK:-metaldocs_default}"

docker run --rm --network "$NETWORK" \
  -e MC_HOST_local="http://${MINIO_ACCESS_KEY}:${MINIO_SECRET_KEY}@minio:9000" \
  minio/mc:RELEASE.2024-04-18T16-45-29Z \
  sh -c "
    mc mb -p local/${BUCKET} || true
    mc anonymous set none local/${BUCKET}
    mc ls local/
  "

echo "OK: bucket ${BUCKET} ready"
