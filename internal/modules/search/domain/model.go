package domain

import "time"

type Document struct {
	ID             string
	Title          string
	OwnerID        string
	Classification string
	Status         string
	CreatedAt      time.Time
}

type Query struct {
	Text           string
	OwnerID        string
	Classification string
	Status         string
	Limit          int
}
