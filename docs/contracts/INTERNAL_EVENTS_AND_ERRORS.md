# Internal Events and Error Contracts

## Domain Event Envelope (stable)
```json
{
  "event_id": "UUID",
  "event_type": "document.created",
  "aggregate_type": "document",
  "aggregate_id": "UUID",
  "occurred_at": "RFC3339",
  "version": 1,
  "idempotency_key": "STRING",
  "producer": "module-name",
  "trace_id": "TRACE_ID",
  "payload": {}
}
```

## Event Types (v1)
- `document.created`
- `document.version.created`
- `workflow.transitioned`
- `iam.role.assigned`
- `audit.recorded`
- `search.index.requested`

## Outbox Contract (transactional)
Table intent:
- `id` (uuid)
- `event_type` (text)
- `aggregate_type` (text)
- `aggregate_id` (uuid/text)
- `payload` (jsonb)
- `idempotency_key` (text)
- `status` (pending|published|failed)
- `attempts` (int)
- `created_at` (timestamptz)
- `published_at` (timestamptz nullable)

Rules:
- Outbox insert ocorre na mesma transacao da mutacao de negocio.
- Publisher atualiza apenas status/attempts.
- Consumer deduplica por `idempotency_key`.

## API Error Map (stable)
Payload:
```json
{
  "error": {
    "code": "DOC_NOT_FOUND",
    "message": "Document not found",
    "details": {},
    "trace_id": "TRACE_ID"
  }
}
```

Initial code catalog:
- `DOC_NOT_FOUND`
- `DOC_VERSION_IMMUTABLE`
- `WORKFLOW_INVALID_TRANSITION`
- `AUTH_UNAUTHORIZED`
- `AUTH_FORBIDDEN`
- `VALIDATION_ERROR`
- `INVALID_NATIVE_CONTENT`
- `CONFLICT_ERROR`
- `INTERNAL_ERROR`
