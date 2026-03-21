# Folder Patterns — MetalDocs

## Standard module (ref: documents/, workflow/, notifications/)
```
internal/modules/<n>/
  domain/
    model.go        entities, value objects, constants, invariants
    port.go         interfaces: Repository, Writer, Reader, Store
    errors.go       sentinel errors (ErrNotFound, ErrInvalidStatus)
  application/
    service.go      use case orchestration — no HTTP, no direct DB
  infrastructure/
    postgres/
      repository.go implements domain port using *sql.DB
    memory/
      repository.go  in-memory test double (map-based)
  delivery/
    http/
      handler.go    parse → auth check → service → response
```

## Read-only module (ref: search/)
```
internal/modules/<n>/
  domain/
    model.go
    port.go         Reader interface only
  application/
    service.go
  infrastructure/
    <source>/
      reader.go     reads from another module's repo or view
  delivery/
    http/
      handler.go
```

## Platform package (ref: platform/authn/, platform/messaging/, platform/observability/)
```
internal/platform/<capability>/
  <capability>.go   or split into focused files
```
Platform packages provide cross-cutting capabilities.
They do not contain business logic.

## When to add memory/ impl
Always — every postgres/ repo has a parallel memory/ test double.
This enables unit tests without DB.

## Module naming
- Package: lowercase single word `documents`, `workflow`, `audit`
- Files: `model.go`, `port.go`, `errors.go`, `service.go`, `repository.go`, `handler.go`
- Errors: `ErrDocumentNotFound`, `ErrVersionImmutable`
- Error codes: `MODULE_ENTITY_REASON` — `DOCUMENT_NOT_FOUND`, `VERSION_IMMUTABLE`
- Routes: `/api/v1/<resource>` or `/api/v1/<module>/<resource>`

## Request flow (frozen — never violate)
delivery → application → domain → infrastructure

delivery: parse HTTP, read auth context, call service, write response
application: orchestrate use cases, call domain + infrastructure + publisher
domain: invariants, entities, interfaces — no IO
infrastructure: implements domain interfaces — DB, storage, memory

## Bootstrap registration (main.go)
Every new module:
1. Add repo/service init in `apps/api/cmd/metaldocs-api/main.go`
2. Add handler.RegisterRoutes(mux)
3. Add permission mapping in `apps/api/cmd/metaldocs-api/permissions.go`

## Cross-module rules
ALLOWED: import domain types (model.go) from another module
ALLOWED: communicate via messaging.Publisher events
ALLOWED: inject other module's repo via interface parameter
FORBIDDEN: import infrastructure or application packages of another module
FORBIDDEN: direct SQL queries on another module's tables
