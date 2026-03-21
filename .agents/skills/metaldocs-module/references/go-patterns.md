# Go Patterns — MetalDocs

## Error wrapping pattern
```go
// Wrap with context on every layer crossing
return fmt.Errorf("create document: %w", err)
return fmt.Errorf("get document by id %s: %w", id, err)

// Map ErrNoRows to domain sentinel
if errors.Is(err, sql.ErrNoRows) {
    return domain.Document{}, domain.ErrDocumentNotFound
}

// Handler maps domain errors to HTTP
if errors.Is(err, domain.ErrDocumentNotFound) {
    writeError(w, 404, "DOCUMENT_NOT_FOUND", "Document not found", traceID)
    return
}
if errors.Is(err, domain.ErrVersionImmutable) {
    writeError(w, 409, "VERSION_IMMUTABLE", "Document version cannot be modified", traceID)
    return
}
```

## ID generation
```go
func generateID() string {
    b := make([]byte, 16)
    rand.Read(b)
    return hex.EncodeToString(b)
}
// Or use existing pattern from documents module
```

## Nullable columns
```go
var decidedAt sql.NullTime
row.Scan(..., &decidedAt)
if decidedAt.Valid {
    t := decidedAt.Time.UTC()
    approval.DecidedAt = &t
}
```

## Pagination
```go
const countQ = `SELECT COUNT(*) FROM metaldocs.<table> WHERE <filter>`
var total int
db.QueryRowContext(ctx, countQ, ...).Scan(&total)

const listQ = `SELECT ... FROM metaldocs.<table> WHERE <filter> ORDER BY created_at DESC LIMIT $1 OFFSET $2`
rows, _ := db.QueryContext(ctx, listQ, limit, offset)
```

## Event publishing pattern
```go
s.publisher.Publish(ctx, messaging.Event{
    EventID:        generateID(),
    EventType:      "document.status_changed",  // stable, versioned name
    AggregateType:  "document",
    AggregateID:    docID,
    Version:        1,
    IdempotencyKey: "document.status_changed:" + docID + ":" + newStatus,
    Producer:       "documents",
    TraceID:        traceID,
    Payload: map[string]any{
        "document_id": docID,
        "from_status": fromStatus,
        "to_status":   newStatus,
        "actor_id":    actorID,
    },
})
```

## requestTraceID helper (standard across handlers)
```go
func requestTraceID(r *http.Request) string {
    if id := strings.TrimSpace(r.Header.Get("X-Trace-Id")); id != "" {
        return id
    }
    return "trace-local"
}
```

## Immutability rules (MetalDocs-specific)
- Document versions: never UPDATE content of existing version row
- Audit events: never UPDATE or DELETE — append-only
- When needing a "new version": INSERT new row with incremented version number
