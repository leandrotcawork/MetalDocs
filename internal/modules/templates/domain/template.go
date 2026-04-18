package domain

import "time"

type Template struct {
	ID                        string
	TenantID                  string
	Key                       string
	Name                      string
	Description               string
	CurrentPublishedVersionID *string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	CreatedBy                 string
}

type TemplateListItem struct {
	ID            string
	TenantID      string
	Key           string
	Name          string
	Description   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CreatedBy     string
	LatestVersion int
	LatestVersionID string
}
