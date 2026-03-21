---
name: metaldocs-module
description: "Implement any MetalDocs Go module layer: domain, infrastructure (postgres + memory), application service, or delivery handler. Called by $md for T2–T6. Uses real repo patterns anchored to existing modules."
---

# MetalDocs Module

## Before writing code
Read `tasks/lessons.md`. Apply every lesson in this task.

## Domain layer (model.go, port.go, errors.go)
```go
// domain/model.go — entities and invariants
type Document struct {
    ID             string
    Title          string
    Status         string
    CreatedAt      time.Time
}

func (d Document) CanTransitionTo(newStatus string) bool {
    // invariant lives here — not in service, not in handler
}

// domain/port.go — interfaces only
type Repository interface {
    Create(ctx context.Context, doc Document) error
    GetByID(ctx context.Context, id string) (Document, error)
    List(ctx context.Context, filter ListFilter) ([]Document, error)
}

// domain/errors.go — sentinel errors
var (
    ErrDocumentNotFound  = errors.New("document not found")
    ErrVersionImmutable  = errors.New("document version is immutable")
    ErrInvalidTransition = errors.New("invalid status transition")
)
```

## Infrastructure — postgres
```go
// infrastructure/postgres/repository.go
type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) GetByID(ctx context.Context, id string) (domain.Document, error) {
    const q = `SELECT id, title, status, created_at FROM metaldocs.documents WHERE id = $1`
    var d domain.Document
    err := r.db.QueryRowContext(ctx, q, id).Scan(&d.ID, &d.Title, &d.Status, &d.CreatedAt)
    if errors.Is(err, sql.ErrNoRows) {
        return domain.Document{}, domain.ErrDocumentNotFound
    }
    if err != nil {
        return domain.Document{}, fmt.Errorf("get document by id: %w", err)
    }
    return d, nil
}
```

## Infrastructure — memory (test double)
```go
// infrastructure/memory/repository.go
type Repository struct {
    mu   sync.RWMutex
    data map[string]domain.Document
}

func NewRepository() *Repository {
    return &Repository{data: make(map[string]domain.Document)}
}

func (r *Repository) GetByID(_ context.Context, id string) (domain.Document, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    if d, ok := r.data[id]; ok { return d, nil }
    return domain.Document{}, domain.ErrDocumentNotFound
}
```

## Application service
```go
// application/service.go
type Service struct {
    repo      domain.Repository
    publisher messaging.Publisher
}

func (s *Service) CreateDocument(ctx context.Context, cmd CreateDocumentCommand) (domain.Document, error) {
    doc := domain.Document{ID: generateID(), Title: cmd.Title, ...}

    if err := s.repo.Create(ctx, doc); err != nil {
        return domain.Document{}, fmt.Errorf("create document: %w", err)
    }

    // Publish event — always with idempotency_key
    s.publisher.Publish(ctx, messaging.Event{
        EventType:      "document.created",
        AggregateType:  "document",
        AggregateID:    doc.ID,
        IdempotencyKey: "document.created:" + doc.ID,
        TraceID:        cmd.TraceID,
        Payload:        map[string]any{"id": doc.ID, "title": doc.Title},
    })
    return doc, nil
}
```

## Delivery handler
```go
// delivery/http/handler.go
func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
    traceID := requestTraceID(r)

    // Auth: IAM middleware already validated — just read context
    userID := authn.UserIDFromContext(r.Context())
    if userID == "" {
        writeError(w, 401, "AUTH_REQUIRED", "Authentication required", traceID)
        return
    }

    var req CreateDocumentRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, 400, "INVALID_REQUEST_BODY", "Invalid request body", traceID)
        return
    }

    result, err := h.service.CreateDocument(r.Context(), application.CreateDocumentCommand{
        ActorID: userID, TraceID: traceID, Title: req.Title,
    })
    if err != nil {
        // Map domain errors to HTTP status
        if errors.Is(err, domain.ErrDocumentNotFound) {
            writeError(w, 404, "DOCUMENT_NOT_FOUND", "Document not found", traceID)
            return
        }
        writeError(w, 500, "INTERNAL_ERROR", "Failed to create document", traceID)
        return
    }

    writeJSON(w, 201, toDocumentResponse(result))
}

// Error response format (standard across all modules)
func writeError(w http.ResponseWriter, status int, code, message, traceID string) {
    writeJSON(w, status, map[string]any{
        "error": map[string]any{
            "code": code, "message": message,
            "details": map[string]any{}, "trace_id": traceID,
        },
    })
}
```

## Permissions registration (always for new endpoints)
```go
// apps/api/cmd/metaldocs-api/permissions.go
if method == http.MethodPost && path == "/api/v1/documents" {
    return iamdomain.PermDocumentCreate, true
}
```

## Tests (minimum required per feature)
```go
// application/service_test.go — unit test with memory repo
func TestCreateDocument(t *testing.T) {
    repo := memory.NewRepository()
    svc  := application.NewService(repo, messaging.NoopPublisher{})
    result, err := svc.CreateDocument(ctx, application.CreateDocumentCommand{Title: "test"})
    assert.NoError(t, err)
    assert.Equal(t, "test", result.Title)
}
```

## After task
1. `go build ./...` passes
2. `go test ./internal/modules/<n>/...` passes
3. Mark `[x]` in `tasks/todo.md`
4. `git commit -m "feat(<m>): <what>"`

## References
- `references/go-patterns.md` — error handling, ID generation, pagination
