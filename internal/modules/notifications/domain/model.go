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
	ReadAt          *time.Time
}

type ListNotificationsQuery struct {
	RecipientUserID string
	Status          string
	Limit           int
}

const (
	StatusPending = "PENDING"
	StatusSent    = "SENT"
	StatusRead    = "READ"
)
