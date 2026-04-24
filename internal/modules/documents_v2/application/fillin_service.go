package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	v2domain "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/documents_v2/repository"
	templatesdomain "metaldocs/internal/modules/templates_v2/domain"
)

type SchemaReader interface {
	LoadPlaceholderSchema(ctx context.Context, tenantID, revisionID string) ([]templatesdomain.Placeholder, error)
	LoadZonesSchema(ctx context.Context, tenantID, revisionID string) ([]templatesdomain.EditableZone, error)
}

type FillInWriter interface {
	UpsertValue(ctx context.Context, v repository.PlaceholderValue) error
	UpsertZoneContent(ctx context.Context, z repository.ZoneContent) error
}

type draftResolver interface {
	ResolveComputedIfStale(ctx context.Context, tenantID, revisionID string) error
}

type FillInService struct {
	db            *sql.DB
	schemas       SchemaReader
	writer        FillInWriter
	draftResolver draftResolver
	iam           IAMUserOptionsReader
}

// NewFillInService wires the service with a DB handle for authz enforcement.
// Production callers MUST use this constructor — it enforces doc.edit_draft capability.
func NewFillInService(db *sql.DB, s SchemaReader, w FillInWriter) *FillInService {
	return &FillInService{db: db, schemas: s, writer: w}
}

// NewFillInServiceNoAuthz is a TEST-ONLY constructor that skips capability checks.
// Never use in production wiring — authz bypass is intentional and audited here.
func NewFillInServiceNoAuthz(s SchemaReader, w FillInWriter) *FillInService {
	return &FillInService{schemas: s, writer: w}
}

// WithDraftResolver attaches a DraftResolverService that runs best-effort after
// each user placeholder upsert. Errors are logged but not propagated.
func (s *FillInService) WithDraftResolver(r draftResolver) *FillInService {
	s.draftResolver = r
	return s
}

// WithIAMReader attaches an IAMUserOptionsReader for validating user-typed placeholders.
func (s *FillInService) WithIAMReader(r IAMUserOptionsReader) *FillInService {
	s.iam = r
	return s
}

type SnapshotSchemaReader struct {
	db *sql.DB
}

func NewSnapshotSchemaReader(db *sql.DB) *SnapshotSchemaReader {
	return &SnapshotSchemaReader{db: db}
}

func (r *SnapshotSchemaReader) LoadPlaceholderSchema(ctx context.Context, tenantID, revisionID string) ([]templatesdomain.Placeholder, error) {
	var raw []byte
	if err := r.db.QueryRowContext(ctx, `
		SELECT placeholder_schema_snapshot
		  FROM documents
		 WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, revisionID).
		Scan(&raw); err != nil {
		return nil, err
	}

	var payload struct {
		Placeholders []templatesdomain.Placeholder `json:"placeholders"`
	}
	if len(raw) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload.Placeholders, nil
}

func (r *SnapshotSchemaReader) LoadZonesSchema(ctx context.Context, tenantID, revisionID string) ([]templatesdomain.EditableZone, error) {
	var raw []byte
	if err := r.db.QueryRowContext(ctx, `
		SELECT editable_zones_schema_snapshot
		  FROM documents
		 WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, revisionID).
		Scan(&raw); err != nil {
		return nil, err
	}

	var payload struct {
		Zones []templatesdomain.EditableZone `json:"zones"`
	}
	if len(raw) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload.Zones, nil
}

func (s *FillInService) SetPlaceholderValue(ctx context.Context, tenantID, actorID, revisionID, placeholderID, raw string) error {
	if s.db != nil {
		if err := requireDocEditDraft(ctx, s.db, tenantID, actorID, revisionID); err != nil {
			return err
		}
	}
	schema, err := s.schemas.LoadPlaceholderSchema(ctx, tenantID, revisionID)
	if err != nil {
		return err
	}

	p, ok := findPlaceholder(schema, placeholderID)
	if !ok {
		return fmt.Errorf("%w: unknown placeholder %s", v2domain.ErrValidationFailed, placeholderID)
	}
	if err := validateValue(ctx, tenantID, p, raw, s.iam); err != nil {
		return err
	}

	value := raw
	if err := s.writer.UpsertValue(ctx, repository.PlaceholderValue{
		TenantID:        tenantID,
		RevisionID:      revisionID,
		PlaceholderID:   placeholderID,
		ValueText:       &value,
		Source:          "user",
		ResolverVersion: nil,
	}); err != nil {
		return err
	}
	if s.draftResolver != nil {
		if rerr := s.draftResolver.ResolveComputedIfStale(ctx, tenantID, revisionID); rerr != nil {
			log.Printf("draft resolver best-effort error: %v", rerr)
		}
	}
	return nil
}

func findPlaceholder(phs []templatesdomain.Placeholder, id string) (templatesdomain.Placeholder, bool) {
	for _, p := range phs {
		if p.ID == id {
			return p, true
		}
	}
	return templatesdomain.Placeholder{}, false
}

func validateValue(ctx context.Context, tenantID string, p templatesdomain.Placeholder, raw string, iam IAMUserOptionsReader) error {
	if p.Required && raw == "" {
		return fmt.Errorf("%w: %s required", v2domain.ErrValidationFailed, p.ID)
	}
	if p.MaxLength != nil && len(raw) > *p.MaxLength {
		return fmt.Errorf("%w: %s max_length exceeded", v2domain.ErrValidationFailed, p.ID)
	}
	if p.Regex != nil {
		re, err := regexp.Compile(*p.Regex)
		if err != nil {
			return err
		}
		if !re.MatchString(raw) {
			return fmt.Errorf("%w: %s regex mismatch", v2domain.ErrValidationFailed, p.ID)
		}
	}

	switch p.Type {
	case templatesdomain.PHNumber:
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fmt.Errorf("%w: %s not a number", v2domain.ErrValidationFailed, p.ID)
		}
		if p.MinNumber != nil && n < *p.MinNumber {
			return fmt.Errorf("%w: %s < min_number", v2domain.ErrValidationFailed, p.ID)
		}
		if p.MaxNumber != nil && n > *p.MaxNumber {
			return fmt.Errorf("%w: %s > max_number", v2domain.ErrValidationFailed, p.ID)
		}
	case templatesdomain.PHDate:
		if _, err := time.Parse("2006-01-02", raw); err != nil {
			return fmt.Errorf("%w: %s not YYYY-MM-DD", v2domain.ErrValidationFailed, p.ID)
		}
		if p.MinDate != nil && raw < *p.MinDate {
			return fmt.Errorf("%w: %s < min_date", v2domain.ErrValidationFailed, p.ID)
		}
		if p.MaxDate != nil && raw > *p.MaxDate {
			return fmt.Errorf("%w: %s > max_date", v2domain.ErrValidationFailed, p.ID)
		}
	case templatesdomain.PHSelect:
		for _, opt := range p.Options {
			if opt == raw {
				return nil
			}
		}
		return fmt.Errorf("%w: %s not in options", v2domain.ErrValidationFailed, p.ID)
	case templatesdomain.PHUser:
		if iam == nil {
			// IAM not wired: skip user validation. Production wiring MUST call WithIAMReader.
			return nil
		}
		opts, err := iam.ListUserOptions(ctx, tenantID)
		if err != nil {
			return err
		}
		for _, o := range opts {
			if o.UserID == raw {
				return nil
			}
		}
		return fmt.Errorf("%w: %s unknown user %s", v2domain.ErrValidationFailed, p.ID, raw)
	}

	return nil
}

func (s *FillInService) SetZoneContent(ctx context.Context, tenantID, actorID, revisionID, zoneID, ooxml string) error {
	if s.db != nil {
		if err := requireDocEditDraft(ctx, s.db, tenantID, actorID, revisionID); err != nil {
			return err
		}
	}
	zones, err := s.schemas.LoadZonesSchema(ctx, tenantID, revisionID)
	if err != nil {
		return err
	}

	zone, ok := findZone(zones, zoneID)
	if !ok {
		return fmt.Errorf("%w: unknown zone %s", v2domain.ErrValidationFailed, zoneID)
	}
	if zone.MaxLength != nil && len(ooxml) > *zone.MaxLength {
		return fmt.Errorf("%w: zone %s exceeds max_length", v2domain.ErrValidationFailed, zoneID)
	}
	if !zone.ContentPolicy.AllowTables && strings.Contains(ooxml, "<w:tbl") {
		return fmt.Errorf("%w: zone %s disallows tables", v2domain.ErrValidationFailed, zoneID)
	}
	if !zone.ContentPolicy.AllowImages && strings.Contains(ooxml, "<w:drawing") {
		return fmt.Errorf("%w: zone %s disallows images", v2domain.ErrValidationFailed, zoneID)
	}
	if !zone.ContentPolicy.AllowHeadings && strings.Contains(ooxml, `<w:pStyle w:val="Heading`) {
		return fmt.Errorf("%w: zone %s disallows headings", v2domain.ErrValidationFailed, zoneID)
	}
	if !zone.ContentPolicy.AllowLists && strings.Contains(ooxml, "<w:numPr") {
		return fmt.Errorf("%w: zone %s disallows lists", v2domain.ErrValidationFailed, zoneID)
	}

	return s.writer.UpsertZoneContent(ctx, repository.ZoneContent{
		TenantID:     tenantID,
		RevisionID:   revisionID,
		ZoneID:       zoneID,
		ContentOOXML: ooxml,
	})
}

func findZone(zones []templatesdomain.EditableZone, id string) (templatesdomain.EditableZone, bool) {
	for _, z := range zones {
		if z.ID == id {
			return z, true
		}
	}
	return templatesdomain.EditableZone{}, false
}
