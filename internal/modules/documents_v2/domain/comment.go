package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID               uuid.UUID
	TenantID         uuid.UUID
	DocumentID       uuid.UUID
	LibraryCommentID int
	ParentLibraryID  *int
	AuthorID         string
	AuthorDisplay    string
	ContentJSON      json.RawMessage
	ResolvedAt       *time.Time
	ResolvedBy       *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CommentCreateInput struct {
	LibraryCommentID int
	ParentLibraryID  *int
	AuthorDisplay    string
	ContentJSON      json.RawMessage
}

type CommentUpdateInput struct {
	ContentJSON *json.RawMessage
	Done        *bool // true sets resolved_at=now()+resolved_by, false clears
}
