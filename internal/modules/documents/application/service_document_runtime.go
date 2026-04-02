package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	auditdomain "metaldocs/internal/modules/audit/domain"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/authn"
	"metaldocs/internal/platform/messaging"
	"metaldocs/internal/platform/render/docgen"
)

type DocumentRuntimeBundle struct {
	Document domain.Document
	Version  domain.Version
	Schema   domain.DocumentProfileSchemaVersion
}

func (s *Service) resolveUserDisplayName(ctx context.Context, userID string) string {
	if userID == "" {
		return "—"
	}
	if s.userResolver == nil {
		return userID
	}
	name, err := s.userResolver.ResolveDisplayName(ctx, userID)
	if err != nil || name == "" {
		return userID
	}
	return name
}

func (s *Service) resolveLatestApproval(ctx context.Context, documentID string) (approverName string, approvedAt string) {
	if s.approvalReader == nil {
		return "—", ""
	}
	approvals, err := s.approvalReader.ListApprovals(ctx, documentID)
	if err != nil || len(approvals) == 0 {
		return "—", ""
	}
	for i := len(approvals) - 1; i >= 0; i-- {
		a := approvals[i]
		if strings.EqualFold(a.Status, "APPROVED") && a.DecidedAt != nil {
			name := s.resolveUserDisplayName(ctx, a.ApproverID)
			return name, a.DecidedAt.Format("2006-01-02")
		}
	}
	return "—", ""
}

func (s *Service) GetDocumentRuntimeBundle(ctx context.Context, documentID string) (DocumentRuntimeBundle, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return DocumentRuntimeBundle{}, err
	}

	version, err := s.latestVersion(ctx, doc.ID)
	if err != nil {
		return DocumentRuntimeBundle{}, err
	}

	runtimeSchema, err := s.resolveActiveProfileSchema(ctx, doc.DocumentProfile)
	if err != nil {
		return DocumentRuntimeBundle{}, err
	}

	return DocumentRuntimeBundle{
		Document: doc,
		Version:  version,
		Schema:   runtimeSchema,
	}, nil
}

func (s *Service) SaveDocumentValues(ctx context.Context, cmd domain.SaveDocumentValuesCommand) (domain.Version, error) {
	documentID := strings.TrimSpace(cmd.DocumentID)
	if documentID == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}

	doc, err := s.repo.GetDocument(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}

	typeDefinition, err := s.resolveDocumentTypeDefinition(ctx, firstNonEmpty(doc.DocumentType, doc.DocumentProfile))
	if err != nil {
		return domain.Version{}, err
	}

	values := cloneRuntimeValues(cmd.Values)
	if err := validateDocumentTypeValues(typeDefinition.Schema, values); err != nil {
		return domain.Version{}, err
	}

	latest, err := s.latestVersion(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}

	if doc.Status == domain.StatusDraft {
		updated := latest
		updated.Values = values
		if err := s.repo.UpdateVersionValues(ctx, documentID, updated.Number, values); err != nil {
			return domain.Version{}, err
		}
		return updated, nil
	}

	next, err := s.repo.NextVersionNumber(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}

	nextVersion := latest
	nextVersion.Number = next
	nextVersion.Values = values
	nextVersion.CreatedAt = s.clock.Now()
	nextVersion.ChangeSummary = fmt.Sprintf("Runtime values update %d", next)

	if err := s.repo.SaveVersion(ctx, nextVersion); err != nil {
		return domain.Version{}, err
	}

	return nextVersion, nil
}

func (s *Service) SaveDocumentValuesAuthorized(ctx context.Context, cmd domain.SaveDocumentValuesCommand) (domain.Version, error) {
	documentID := strings.TrimSpace(cmd.DocumentID)
	if documentID == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}

	doc, err := s.repo.GetDocument(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}

	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return domain.Version{}, err
	}
	if !allowed {
		return domain.Version{}, domain.ErrDocumentNotFound
	}

	var previousValues domain.DocumentValues
	if doc.Status == domain.StatusDraft {
		previous, err := s.latestVersion(ctx, documentID)
		if err != nil {
			return domain.Version{}, err
		}
		previousValues = cloneRuntimeValues(previous.Values)
	}

	version, err := s.SaveDocumentValues(ctx, cmd)
	if err != nil {
		return domain.Version{}, err
	}

	now := s.clock.Now()
	if err := s.recordRuntimeValuesAudit(ctx, doc, version, len(cmd.Values), cmd.TraceID, now); err != nil {
		if doc.Status == domain.StatusDraft && len(previousValues) > 0 {
			_ = s.repo.UpdateVersionValues(ctx, documentID, version.Number, previousValues)
		}
		return domain.Version{}, err
	}
	s.publishRuntimeValuesUpdated(ctx, doc, version, len(cmd.Values), cmd.TraceID, now)

	return version, nil
}

func (s *Service) buildDocgenPayload(ctx context.Context, doc domain.Document, schema domain.DocumentProfileSchemaVersion, version domain.Version) (docgen.RenderPayload, error) {
	schemaMap, err := toDocgenSchema(schema.ContentSchema)
	if err != nil {
		return docgen.RenderPayload{}, err
	}

	ownerName := s.resolveUserDisplayName(ctx, doc.OwnerID)
	approverName, approvedAt := s.resolveLatestApproval(ctx, doc.ID)

	payload := docgen.RenderPayload{
		DocumentType: firstNonEmpty(doc.DocumentType, doc.DocumentProfile),
		DocumentCode: doc.DocumentCode,
		Title:        doc.Title,
		Version:      fmt.Sprintf("%d", version.Number),
		Status:       doc.Status,
		Schema:       schemaMap,
		Values:       cloneRuntimeValues(version.Values),
		Metadata: &docgen.RenderMetadata{
			ElaboradoPor: ownerName,
			AprovadoPor:  approverName,
			CreatedAt:    doc.CreatedAt.Format("2006-01-02"),
			ApprovedAt:   approvedAt,
		},
	}

	versions, err := s.repo.ListVersions(ctx, doc.ID)
	if err == nil && len(versions) > 0 {
		revisions := make([]docgen.RenderRevision, 0, len(versions))
		for _, v := range versions {
			summary := v.ChangeSummary
			if summary == "" && v.Number == 1 {
				summary = "Criação do documento"
			}
			revisions = append(revisions, docgen.RenderRevision{
				Versao:    fmt.Sprintf("%d", v.Number),
				Data:      v.CreatedAt.Format("2006-01-02"),
				Descricao: summary,
				Por:       ownerName,
			})
		}
		payload.Revisions = revisions
	}

	return payload, nil
}

func (s *Service) ExportDocumentDocxAuthorized(ctx context.Context, documentID, traceID string) ([]byte, error) {
	if s.docgenClient == nil {
		return nil, domain.ErrRenderUnavailable
	}

	bundle, err := s.GetDocumentRuntimeBundle(ctx, documentID)
	if err != nil {
		return nil, err
	}

	payload, err := s.buildDocgenPayload(ctx, bundle.Document, bundle.Schema, bundle.Version)
	if err != nil {
		return nil, err
	}

	return s.docgenClient.Generate(ctx, payload, traceID)
}

func (s *Service) generateDocxBytes(ctx context.Context, doc domain.Document, version domain.Version, content map[string]any, traceID string) ([]byte, error) {
	if s.docgenClient == nil {
		return nil, domain.ErrRenderUnavailable
	}

	schema, err := s.resolveActiveProfileSchema(ctx, doc.DocumentProfile)
	if err != nil {
		return nil, err
	}

	versionWithValues := version
	if len(content) > 0 {
		versionWithValues.Values = content
	}

	payload, err := s.buildDocgenPayload(ctx, doc, schema, versionWithValues)
	if err != nil {
		return nil, err
	}

	return s.docgenClient.Generate(ctx, payload, traceID)
}

func (s *Service) exportDocumentDocxAuthorizedLegacy(ctx context.Context, documentID, traceID string) ([]byte, error) {
	if s.docgenClient == nil {
		return nil, domain.ErrRenderUnavailable
	}

	bundle, err := s.GetDocumentRuntimeBundle(ctx, documentID)
	if err != nil {
		return nil, err
	}

	schema, err := toDocgenSchema(bundle.Schema.ContentSchema)
	if err != nil {
		return nil, err
	}

	payload := docgen.RenderPayload{
		DocumentType: firstNonEmpty(bundle.Document.DocumentType, bundle.Document.DocumentProfile),
		DocumentCode: bundle.Document.DocumentCode,
		Title:        bundle.Document.Title,
		Version:      fmt.Sprintf("%d", bundle.Version.Number),
		Status:       bundle.Document.Status,
		Schema:       schema,
		Values:       cloneRuntimeValues(bundle.Version.Values),
	}

	ownerName := s.resolveUserDisplayName(ctx, bundle.Document.OwnerID)
	approverName, approvedAt := s.resolveLatestApproval(ctx, documentID)
	payload.Metadata = &docgen.RenderMetadata{
		ElaboradoPor: ownerName,
		AprovadoPor:  approverName,
		CreatedAt:    bundle.Document.CreatedAt.Format("2006-01-02"),
		ApprovedAt:   approvedAt,
	}

	versions, err := s.repo.ListVersions(ctx, documentID)
	if err == nil && len(versions) > 0 {
		revisions := make([]docgen.RenderRevision, 0, len(versions))
		for _, v := range versions {
			summary := v.ChangeSummary
			if summary == "" && v.Number == 1 {
				summary = "Criação do documento"
			}
			revisions = append(revisions, docgen.RenderRevision{
				Versao:    fmt.Sprintf("%d", v.Number),
				Data:      v.CreatedAt.Format("2006-01-02"),
				Descricao: summary,
				Por:       ownerName,
			})
		}
		payload.Revisions = revisions
	}

	return s.docgenClient.Generate(ctx, payload, traceID)
}

func (s *Service) recordRuntimeValuesAudit(ctx context.Context, doc domain.Document, version domain.Version, valueCount int, traceID string, now time.Time) error {
	if s.audit == nil {
		return nil
	}

	payload, err := json.Marshal(map[string]any{
		"document_id":    doc.ID,
		"version_number": version.Number,
		"value_count":    valueCount,
	})
	if err != nil {
		return fmt.Errorf("marshal runtime values audit payload: %w", err)
	}

	return s.audit.Record(ctx, auditdomain.Event{
		ID:           mustNewID(),
		OccurredAt:   now,
		ActorID:      strings.TrimSpace(authn.UserIDFromContext(ctx)),
		Action:       "document.runtime.values.updated",
		ResourceType: "document",
		ResourceID:   doc.ID,
		PayloadJSON:  string(payload),
		TraceID:      strings.TrimSpace(traceID),
	})
}

func (s *Service) publishRuntimeValuesUpdated(ctx context.Context, doc domain.Document, version domain.Version, valueCount int, traceID string, now time.Time) {
	if s.publisher == nil {
		return
	}

	_ = s.publisher.Publish(ctx, messaging.Event{
		EventID:           fmt.Sprintf("evt-document-runtime-values-updated-%s-%d", doc.ID, version.Number),
		EventType:         "document.runtime.values.updated",
		AggregateType:     "document",
		AggregateID:       doc.ID,
		OccurredAtRFC3339: now.Format(time.RFC3339),
		Version:           version.Number,
		IdempotencyKey:    fmt.Sprintf("document.runtime.values.updated:%s", doc.ID),
		Producer:          "documents",
		TraceID:           strings.TrimSpace(traceID),
		Payload: map[string]any{
			"document_id":    doc.ID,
			"version_number": version.Number,
			"value_count":    valueCount,
		},
	})
}

func toDocgenSchema(schema map[string]any) (docgen.RenderSchema, error) {
	rawSections := toRuntimeMapSlice(schema["sections"])
	sections := make([]docgen.RenderSection, 0, len(rawSections))
	for index, rawSection := range rawSections {
		sectionMap, ok := rawSection.(map[string]any)
		if !ok {
			return docgen.RenderSchema{}, domain.ErrInvalidCommand
		}
		section, err := toDocgenSection(sectionMap, index+1)
		if err != nil {
			return docgen.RenderSchema{}, err
		}
		sections = append(sections, section)
	}
	if len(sections) == 0 {
		return docgen.RenderSchema{}, domain.ErrInvalidCommand
	}
	return docgen.RenderSchema{Sections: sections}, nil
}

func toDocgenSection(section map[string]any, fallbackNum int) (docgen.RenderSection, error) {
	key, ok := toRuntimeString(section["key"])
	if !ok {
		return docgen.RenderSection{}, domain.ErrInvalidCommand
	}
	title, ok := toRuntimeString(section["title"])
	if !ok {
		return docgen.RenderSection{}, domain.ErrInvalidCommand
	}
	num, ok := toRuntimeString(section["num"])
	if !ok {
		num = fmt.Sprintf("%d", fallbackNum)
	}

	fields := make([]docgen.RenderField, 0)
	for _, rawField := range toRuntimeMapSlice(section["fields"]) {
		fieldMap, ok := rawField.(map[string]any)
		if !ok {
			return docgen.RenderSection{}, domain.ErrInvalidCommand
		}
		field, err := toDocgenField(fieldMap, false)
		if err != nil {
			return docgen.RenderSection{}, err
		}
		fields = append(fields, field)
	}
	if len(fields) == 0 {
		return docgen.RenderSection{}, domain.ErrInvalidCommand
	}

	sectionOut := docgen.RenderSection{
		Key:    key,
		Num:    num,
		Title:  title,
		Fields: fields,
	}
	if color, ok := toRuntimeString(section["color"]); ok {
		sectionOut.Color = color
	}
	return sectionOut, nil
}

func toDocgenField(field map[string]any, inTable bool) (docgen.RenderField, error) {
	key, ok := toRuntimeString(field["key"])
	if !ok {
		return docgen.RenderField{}, domain.ErrInvalidCommand
	}
	label, ok := toRuntimeString(field["label"])
	if !ok {
		label = key
	}
	fieldType, ok := toRuntimeString(field["type"])
	if !ok {
		fieldType = "text"
	}

	renderType := fieldType
	if inTable {
		renderType = normalizeDocgenScalarType(renderType)
	}

	out := docgen.RenderField{
		Key:   key,
		Label: label,
		Type:  renderType,
	}
	if options := toRuntimeStringSlice(field["options"]); len(options) > 0 {
		out.Options = options
	}

	if inTable {
		switch fieldType {
		case "array", "checklist", "repeat", "rich", "rich_blocks", "table":
			out.Type = "textarea"
			return out, nil
		}
	}

	switch fieldType {
	case "table":
		columns := make([]docgen.RenderField, 0)
		for _, rawColumn := range toRuntimeMapSlice(field["columns"]) {
			columnMap, ok := rawColumn.(map[string]any)
			if !ok {
				return docgen.RenderField{}, domain.ErrInvalidCommand
			}
			column, err := toDocgenField(columnMap, true)
			if err != nil {
				return docgen.RenderField{}, err
			}
			columns = append(columns, column)
		}
		if len(columns) == 0 {
			return docgen.RenderField{}, domain.ErrInvalidCommand
		}
		out.Type = "table"
		out.Columns = columns
	case "repeat":
		itemFields := make([]docgen.RenderField, 0)
		for _, rawItemField := range toRuntimeMapSlice(field["itemFields"]) {
			itemFieldMap, ok := rawItemField.(map[string]any)
			if !ok {
				return docgen.RenderField{}, domain.ErrInvalidCommand
			}
			itemField, err := toDocgenField(itemFieldMap, false)
			if err != nil {
				return docgen.RenderField{}, err
			}
			itemFields = append(itemFields, itemField)
		}
		if len(itemFields) == 0 {
			itemType := normalizeDocgenScalarType(toRuntimeStringFallback(field["itemType"], "text"))
			itemFields = []docgen.RenderField{
				{
					Key:   "value",
					Label: "Item",
					Type:  itemType,
				},
			}
		}
		out.Type = "repeat"
		out.ItemFields = itemFields
	case "array":
		itemType := normalizeDocgenScalarType(toRuntimeStringFallback(field["itemType"], "text"))
		out.Type = "repeat"
		out.ItemFields = []docgen.RenderField{
			{
				Key:   "value",
				Label: "Item",
				Type:  itemType,
			},
		}
	case "checklist":
		out.Type = "repeat"
		out.ItemFields = []docgen.RenderField{
			{
				Key:   "label",
				Label: "Item",
				Type:  "text",
			},
			{
				Key:   "checked",
				Label: "Concluido",
				Type:  "checkbox",
			},
		}
	case "rich_blocks":
		out.Type = "textarea"
	case "rich":
		out.Type = "rich"
	default:
		out.Type = normalizeDocgenScalarType(renderType)
	}

	return out, nil
}

func normalizeDocgenScalarType(fieldType string) string {
	switch strings.ToLower(strings.TrimSpace(fieldType)) {
	case "text", "textarea", "number", "date", "select", "checkbox", "table", "rich", "repeat":
		return strings.ToLower(strings.TrimSpace(fieldType))
	case "rich_blocks":
		return "textarea"
	case "array", "checklist":
		return "repeat"
	default:
		return "text"
	}
}

func toRuntimeString(value any) (string, bool) {
	str, ok := value.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(str)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func toRuntimeStringFallback(value any, fallback string) string {
	if str, ok := toRuntimeString(value); ok {
		return str
	}
	return fallback
}

func toRuntimeStringSlice(value any) []string {
	items := toRuntimeMapSlice(value)
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if str, ok := item.(string); ok {
			if trimmed := strings.TrimSpace(str); trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}

func toRuntimeMapSlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []map[string]any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	case []string:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}
