package domain

import "time"

type Notification struct {
	ID              string
	RecipientUserID string
	EventType       string
	ResourceType    string
	ResourceID      string
	Title           string
	Message         string
	Status          string
	IdempotencyKey  string
	CreatedAt       time.Time
}

const (
	StatusPending = "PENDING"
	StatusSent    = "SENT"
)
