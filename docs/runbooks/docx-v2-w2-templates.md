# Runbook: docx-v2 W2 Templates Vertical

## Routes added in W2

All routes require JWT auth. Role checks are done server-side via `X-User-Roles` header
injected by the gateway after token validation.

| Method | Path | Required role(s) | Purpose |
|--------|------|-----------------|---------|
| `GET` | `/api/v2/templates` | admin, template_author, template_publisher | List all templates for tenant |
| `POST` | `/api/v2/templates` | admin, template_author | Create a new template (seeds draft v1) |
| `GET` | `/api/v2/templates/{id}/versions/{n}` | admin, template_author, template_publisher | Fetch a specific version |
| `PUT` | `/api/v2/templates/{id}/versions/{n}/draft` | admin, template_author | Save draft content (storage keys + hashes) |
| `POST` | `/api/v2/templates/{id}/versions/{n}/publish` | admin, template_publisher | Validate + publish a draft version |
| `POST` | `/api/v2/templates/{id}/versions/{n}/docx-upload-url` | admin, template_author | Get presigned S3 PUT URL for DOCX |
| `POST` | `/api/v2/templates/{id}/versions/{n}/schema-upload-url` | admin, template_author | Get presigned S3 PUT URL for schema JSON |
| `GET` | `/api/v2/signed?key=<storage_key>` | admin, template_author, template_publisher | Redirect to presigned S3 GET URL |

Headers required on every request:
- `X-Tenant-ID` — tenant UUID
- `X-User-Roles` — comma-separated role list

---

## Publish flow (end-to-end)

```
Author                   API (templates handler)     docgen-v2            Postgres
  |                              |                       |                    |
  | POST /publish {docxKey,      |                       |                    |
  |   schemaKey}                 |                       |                    |
  |----------------------------> |                       |                    |
  |                              | ValidateTemplate()    |                    |
  |                              |---------------------->|                    |
  |                              |                       | Parses DOCX        |
  |                              |                       | Checks tokens vs   |
  |                              |                       | schema             |
  |                              |<-- {valid, errs} -----|                    |
  |                              |                       |                    |
  |         [if invalid]         |                       |                    |
  |<--- 422 {errors JSON} -------|                       |                    |
  |                              |                       |                    |
  |         [if valid]           |                       |                    |
  |                              | PublishVersion(tx)    |                    |
  |                              |---------------------------------------------->|
  |                              |      UPDATE status='published'            |
  |                              |      UPDATE templates.current_published.. |
  |                              |      INSERT next draft (version_num+1)    |
  |                              |<----------------------------------------------|
  |<--- 200 {published_version_id, next_draft_id, next_draft_version_num} ---|
```

Key invariants:
1. `PublishVersion` runs in a single DB transaction; if any step fails, no state changes.
2. Validation always runs before the DB write — a failed `docgen-v2` call (5xx) returns 500 and does **not** advance version state.
3. After publish, a new draft version (N+1) is automatically seeded with the same storage keys as the just-published version. Authors can then upload new files into that draft.

---

## How to reset a stuck draft (optimistic lock issue)

**Symptom:** `PUT /draft` returns `409 template_draft_stale`. This means the `lock_version` the
client sent does not match the value in Postgres — another writer already incremented it, or the
client is holding a stale copy.

**Client fix (preferred):**
1. Re-fetch the version via `GET /api/v2/templates/{id}/versions/{n}`.
2. Read the `lock_version` field from the response.
3. Retry the `PUT /draft` using the fresh `lock_version`.

**Manual DB reset (ops last resort — requires direct Postgres access):**

```sql
-- Check current state
SELECT id, status, lock_version, updated_at
FROM template_versions
WHERE template_id = '<template_id>'
ORDER BY version_num DESC
LIMIT 5;

-- Force lock_version back to a known value (only if no concurrent writers)
UPDATE template_versions
SET lock_version = <known_good_value>
WHERE id = '<version_id>'
  AND status = 'draft';
```

Warning: Only do the manual reset if you have confirmed there are no active clients editing the
draft. Resetting lock_version with concurrent writers will cause silent overwrites.

---

## docgen-v2 /validate/template error taxonomy

docgen-v2 returns a JSON array of error objects when validation fails (HTTP 200 with `valid: false`).
The `type` field classifies each error:

### Parse errors

The DOCX or schema JSON could not be read at all. No token-level validation was attempted.

| `type` | Meaning | Action |
|--------|---------|--------|
| `parse_error_docx` | DOCX file is corrupt or not a valid Office Open XML file | Re-upload the DOCX |
| `parse_error_schema` | Schema JSON is malformed or fails its own JSON Schema | Fix and re-upload schema |
| `parse_error_encoding` | Unexpected text encoding in DOCX body | Re-save DOCX as UTF-8 from Word/LibreOffice |

### Missing tokens

Tokens declared in the schema are not found anywhere in the DOCX body or headers/footers.

| `type` | Meaning | Action |
|--------|---------|--------|
| `missing_token` | Schema declares `{{token}}` but DOCX does not contain it | Add the token to the DOCX or remove it from schema |
| `missing_required_token` | As above, and the schema marks it `required: true` | Must be present before publish |

### Orphan tokens

Tokens found in the DOCX that are not declared in the schema — they would render as literal strings
at generation time.

| `type` | Meaning | Action |
|--------|---------|--------|
| `orphan_token` | DOCX contains `{{token}}` not declared in schema | Add to schema or fix typo in DOCX |

### Full error object shape

```json
{
  "type": "missing_token",
  "token": "client_name",
  "location": "body|header|footer",
  "message": "human-readable description"
}
```

---

## Feature flag: METALDOCS_DOCX_V2_ENABLED

Controls whether the W2 templates routes (`/api/v2/templates/*`) are registered in the router.

**Values:**
- `"true"` (or `"1"`) — routes are active. Default for `staging` and `prod` once W2 ships.
- anything else / unset — routes return 404. Default for `prod` during W1 roll-out.

**Where it's read:** `cmd/api/main.go` (or the router wiring file) at startup. Changing the value
requires a process restart; there is no hot-reload.

**Enabling in local dev:**
```bash
export METALDOCS_DOCX_V2_ENABLED=true
go run ./cmd/api
```

**Enabling in Kubernetes:**
Add or update the env var in the deployment manifest and roll the deployment:
```yaml
env:
  - name: METALDOCS_DOCX_V2_ENABLED
    value: "true"
```

**Roll-back:** Set the flag to `"false"` and redeploy. No DB migration is needed — the schema is
additive and harmless when the routes are dark.
