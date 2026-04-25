package infrastructure

import (
	"context"
	"database/sql"
	"errors"
)

type TemplateVersionChecker struct {
	db *sql.DB
}

func NewTemplateVersionChecker(db *sql.DB) *TemplateVersionChecker {
	return &TemplateVersionChecker{db: db}
}

func (c *TemplateVersionChecker) IsPublished(ctx context.Context, versionID string) (bool, string, error) {
	if c.db == nil {
		return false, "", nil
	}
	var status sql.NullString
	var profileCode sql.NullString
	err := c.db.QueryRowContext(ctx, `
		SELECT v.status, t.profile_code
		FROM templates_v2_template_version v
		JOIN templates_v2_template t ON t.id = v.template_id
		WHERE v.id = $1
	`, versionID).Scan(&status, &profileCode)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	if !status.Valid || status.String != "published" {
		return false, profileCode.String, nil
	}
	return true, profileCode.String, nil
}
