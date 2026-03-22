---
name: metaldocs-openapi
description: Create or update MetalDocs OpenAPI contract in api/openapi/v1/openapi.yaml. Source of truth for all API endpoints. Called by $md for T1. No endpoint exists without an OpenAPI entry.
---

# MetalDocs OpenAPI

Use `$md` as the workflow owner when this change is part of a larger implementation. This skill only owns the contract update itself.

## Workflow
1. Read `docs/standards/ENGINEERING_STANDARDS.md` (API section)
2. Open `api/openapi/v1/openapi.yaml`
3. Add path, method, request/response schemas
4. Follow existing patterns in the file
5. Finish with `references/openapi-checklist.md`

## Rules
- Single file: `api/openapi/v1/openapi.yaml`
- No endpoint in code without entry in OpenAPI
- Breaking change → `/api/v2` (never break v1)
- Error responses follow standard format:
  ```yaml
  error:
    type: object
    properties:
      code:     { type: string }
      message:  { type: string }
      details:  { type: object }
      trace_id: { type: string }
  ```
- OpenAPI updated in same PR as the API change

## References
- Contract: `api/openapi/v1/openapi.yaml`
- Standards: `docs/standards/ENGINEERING_STANDARDS.md`
- Checklist: `references/openapi-checklist.md`
