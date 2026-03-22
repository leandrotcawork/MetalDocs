package domain

import "time"

type Document struct {
	ID               string
	Title            string
	DocumentType     string
	DocumentProfile  string
	DocumentFamily   string
	DocumentSequence int
	DocumentCode     string
	ProcessArea      string
	Subject          string
	OwnerID          string
	BusinessUnit     string
	Department       string
	Classification   string
	Status           string
	Tags             []string
	EffectiveAt      *time.Time
	ExpiryAt         *time.Time
	CreatedAt        time.Time
}

type Query struct {
	Text            string
	DocumentType    string
	DocumentProfile string
	DocumentFamily  string
	ProcessArea     string
	Subject         string
	OwnerID         string
	BusinessUnit    string
	Department      string
	Classification  string
	Status          string
	Tag             string
	ExpiryBefore    *time.Time
	ExpiryAfter     *time.Time
	Limit           int
}

type AccessPolicy struct {
	SubjectType   string
	SubjectID     string
	ResourceScope string
	ResourceID    string
	Capability    string
	Effect        string
}
