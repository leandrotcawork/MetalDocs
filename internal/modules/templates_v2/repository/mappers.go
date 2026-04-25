package repository

import (
	"database/sql"
	"encoding/json"

	"metaldocs/internal/modules/templates_v2/domain"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTemplate(row rowScanner) (*domain.Template, error) {
	var (
		t                  domain.Template
		visibility         string
		areasJSON          []byte
		specificAreasJSON  []byte
		publishedVersionID sql.NullString
		archivedAt         sql.NullTime
	)
	if err := row.Scan(
		&t.ID, &t.TenantID, &t.DocTypeCode, &t.Key, &t.Name, &t.Description, &areasJSON, &visibility, &specificAreasJSON,
		&t.LatestVersion, &publishedVersionID, &t.CreatedBy, &t.CreatedAt, &archivedAt,
	); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(areasJSON, &t.Areas); err != nil {
		t.Areas = []string{}
	}
	if err := json.Unmarshal(specificAreasJSON, &t.SpecificAreas); err != nil {
		t.SpecificAreas = []string{}
	}
	t.Visibility = domain.Visibility(visibility)
	if publishedVersionID.Valid {
		t.PublishedVersionID = &publishedVersionID.String
	}
	if archivedAt.Valid {
		t.ArchivedAt = &archivedAt.Time
	}
	return &t, nil
}

func scanTemplateVersion(row rowScanner) (*domain.TemplateVersion, error) {
	var (
		v                   domain.TemplateVersion
		status              string
		metadataJSON        []byte
		placeholderJSON     []byte
		pendingReviewerRole sql.NullString
		reviewerID          sql.NullString
		approverID          sql.NullString
		submittedAt         sql.NullTime
		reviewedAt          sql.NullTime
		approvedAt          sql.NullTime
		publishedAt         sql.NullTime
		obsoletedAt         sql.NullTime
	)
	if err := row.Scan(
		&v.ID, &v.TemplateID, &v.VersionNumber, &status, &v.DocxStorageKey, &v.ContentHash,
		&metadataJSON, &placeholderJSON, &v.AuthorID,
		&pendingReviewerRole, &v.PendingApproverRole, &reviewerID, &approverID,
		&submittedAt, &reviewedAt, &approvedAt, &publishedAt, &obsoletedAt, &v.CreatedAt,
	); err != nil {
		return nil, err
	}
	v.Status = domain.VersionStatus(status)
	if pendingReviewerRole.Valid {
		v.PendingReviewerRole = &pendingReviewerRole.String
	}
	if reviewerID.Valid {
		v.ReviewerID = &reviewerID.String
	}
	if approverID.Valid {
		v.ApproverID = &approverID.String
	}
	if submittedAt.Valid {
		v.SubmittedAt = &submittedAt.Time
	}
	if reviewedAt.Valid {
		v.ReviewedAt = &reviewedAt.Time
	}
	if approvedAt.Valid {
		v.ApprovedAt = &approvedAt.Time
	}
	if publishedAt.Valid {
		v.PublishedAt = &publishedAt.Time
	}
	if obsoletedAt.Valid {
		v.ObsoletedAt = &obsoletedAt.Time
	}
	if err := json.Unmarshal(metadataJSON, &v.MetadataSchema); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(placeholderJSON, &v.PlaceholderSchema); err != nil {
		return nil, err
	}
	return &v, nil
}

func marshalVersionSchemas(v *domain.TemplateVersion) (metadataJSON, placeholderJSON []byte, err error) {
	metadataJSON, err = json.Marshal(v.MetadataSchema)
	if err != nil {
		return nil, nil, err
	}
	placeholderJSON, err = json.Marshal(v.PlaceholderSchema)
	if err != nil {
		return nil, nil, err
	}
	return metadataJSON, placeholderJSON, nil
}

func marshalAuditDetails(details map[string]any) ([]byte, error) {
	if details == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(details)
}

func unmarshalAuditDetails(raw []byte, details *map[string]any) error {
	return json.Unmarshal(raw, details)
}

func normalizedTextArray(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
