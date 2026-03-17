package domain

import "time"

type Document struct {
	ID             string
	Title          string
	DocumentType   string
	OwnerID        string
	BusinessUnit   string
	Department     string
	Classification string
	Status         string
	Tags           []string
	EffectiveAt    *time.Time
	ExpiryAt       *time.Time
	CreatedAt      time.Time
}

type Query struct {
	Text           string
	DocumentType   string
	OwnerID        string
	BusinessUnit   string
	Department     string
	Classification string
	Status         string
	Limit          int
}
