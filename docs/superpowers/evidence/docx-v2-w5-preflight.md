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
