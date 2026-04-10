package application

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/messaging"
	"metaldocs/internal/platform/render/docgen"
)

func (s *Service) UploadDocxContentAuthorized(ctx context.Context, cmd domain.UploadDocxContentCommand) (domain.Version, error) {
	if s.attachmentStore == nil {
		return domain.Version{}, domain.ErrAttachmentStoreUnavailable
	}
	if strings.TrimSpace(cmd.DocumentID) == "" || strings.TrimSpace(cmd.FileName) == "" || len(cmd.Content) == 0 {
		return domain.Version{}, domain.ErrInvalidAttachment
	}
	if len(cmd.Content) > 10*1024*1024 {
		return domain.Version{}, domain.ErrInvalidAttachment
	}
	if !isDocxPayload(cmd.Content) {
		return domain.Version{}, domain.ErrInvalidAttachment
	}

	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
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
	if !isVersioningAllowed(doc) {
		return domain.Version{}, domain.ErrVersioningNotAllowed
	}

	next, err := s.repo.NextVersionNumber(ctx, doc.ID)
	if err != nil {
		return domain.Version{}, err
	}
	now := s.clock.Now()

	docxKey := documentContentStorageKey(doc.ID, next, "docx")
	if err := s.attachmentStore.Save(ctx, docxKey, cmd.Content); err != nil {
		return domain.Version{}, err
	}

	pdfBytes, err := s.convertDocxToPDF(ctx, cmd.Content, cmd.TraceID)
	if err != nil {
		_ = s.attachmentStore.Delete(ctx, docxKey)
		return domain.Version{}, err
	}
	pdfKey := documentContentStorageKey(doc.ID, next, "pdf")
	if err := s.attachmentStore.Save(ctx, pdfKey, pdfBytes); err != nil {
		_ = s.attachmentStore.Delete(ctx, docxKey)
		return domain.Version{}, err
	}

	textContent := extractDocxText(cmd.Content)

	version := domain.Version{
		DocumentID:       doc.ID,
		Number:           next,
		Content:          textContent,
		ContentHash:      contentHash(textContent),
		ChangeSummary:    fmt.Sprintf("Content version %d", next),
		ContentSource:    domain.ContentSourceDocxUpload,
		DocxStorageKey:   docxKey,
		PdfStorageKey:    pdfKey,
		TextContent:      textContent,
		FileSizeBytes:    int64(len(cmd.Content)),
		OriginalFilename: strings.TrimSpace(cmd.FileName),
		CreatedAt:        now,
	}

	if err := s.repo.SaveVersion(ctx, version); err != nil {
		_ = s.attachmentStore.Delete(ctx, pdfKey)
		_ = s.attachmentStore.Delete(ctx, docxKey)
		return domain.Version{}, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-version-create-%s-%d", doc.ID, next),
			EventType:         "document.version.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: now.Format(time.RFC3339),
			Version:           next,
			IdempotencyKey:    fmt.Sprintf("document.version.created:%s:%d", doc.ID, next),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id": doc.ID,
				"version":     next,
				"source":      version.ContentSource,
			},
		})
	}

	return version, nil
}

func isDocxPayload(content []byte) bool {
	return len(content) >= 4 && bytes.Equal(content[:4], []byte("PK\x03\x04"))
}

func extractDocxText(content []byte) string {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return ""
	}
	for _, file := range reader.File {
		if file.Name != "word/document.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return ""
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		decoder := xml.NewDecoder(bytes.NewReader(data))
		var builder strings.Builder
		for {
			token, err := decoder.Token()
			if err != nil {
				break
			}
			switch value := token.(type) {
			case xml.CharData:
				text := strings.TrimSpace(string(value))
				if text != "" {
					if builder.Len() > 0 {
						builder.WriteString(" ")
					}
					builder.WriteString(text)
				}
			}
		}
		return builder.String()
	}
	return ""
}

func (s *Service) convertDocxToPDF(ctx context.Context, content []byte, traceID string) ([]byte, error) {
	if s.gotenbergClient == nil {
		return nil, fmt.Errorf("gotenberg client not configured: PDF conversion unavailable")
	}
	return s.gotenbergClient.ConvertDocxToPDF(ctx, content)
}

func (s *Service) generateBrowserDocxBytes(ctx context.Context, doc domain.Document, version domain.Version, exportConfig *domain.TemplateExportConfig, traceID string) ([]byte, error) {
	var template *domain.DocumentTemplateVersion
	if strings.TrimSpace(version.TemplateKey) != "" && version.TemplateVersion > 0 {
		tmpl, err := s.repo.GetDocumentTemplateVersion(ctx, version.TemplateKey, version.TemplateVersion)
		if err != nil {
			return nil, err
		}
		template = &tmpl
	}
	return s.generateBrowserDocxBytesWithTemplate(ctx, doc, version, exportConfig, template, traceID)
}

func (s *Service) generateBrowserDocxBytesWithTemplate(ctx context.Context, doc domain.Document, version domain.Version, exportConfig *domain.TemplateExportConfig, template *domain.DocumentTemplateVersion, traceID string) ([]byte, error) {
	if s.docgenClient == nil {
		return nil, domain.ErrRenderUnavailable
	}
	if strings.TrimSpace(version.Content) == "" {
		return nil, domain.ErrInvalidCommand
	}

	payload := docgen.MDDMExportPayload{
		Envelope: json.RawMessage([]byte(version.Content)),
		Metadata: docgen.MDDMExportMetadata{
			DocumentCode:  doc.DocumentCode,
			Title:         doc.Title,
			RevisionLabel: fmt.Sprintf("REV%02d", version.Number),
			Mode:          "production",
		},
	}
	if theme := mddmTemplateThemeFromDefinition(template); theme != nil {
		payload.TemplateTheme = theme
	}

	rendered, err := s.docgenClient.GenerateMDDM(ctx, payload, traceID)
	if err != nil {
		if errors.Is(err, docgen.ErrUnavailable) {
			return nil, domain.ErrRenderUnavailable
		}
		return nil, err
	}
	return rendered, nil
}

func mddmTemplateThemeFromDefinition(template *domain.DocumentTemplateVersion) *docgen.MDDMTemplateTheme {
	if template == nil || len(template.Definition) == 0 {
		return nil
	}

	rawTheme, ok := template.Definition["theme"].(map[string]any)
	if !ok || len(rawTheme) == 0 {
		return nil
	}

	theme := docgen.MDDMTemplateTheme{}
	if value, ok := toRuntimeString(rawTheme["accent"]); ok {
		theme.Accent = value
	}
	if value, ok := toRuntimeString(rawTheme["accentLight"]); ok {
		theme.AccentLight = value
	}
	if value, ok := toRuntimeString(rawTheme["accentDark"]); ok {
		theme.AccentDark = value
	}
	if value, ok := toRuntimeString(rawTheme["accentBorder"]); ok {
		theme.AccentBorder = value
	}

	if theme == (docgen.MDDMTemplateTheme{}) {
		return nil
	}

	return &theme
}

func browserRenderMarginsFromExportConfig(cfg *domain.TemplateExportConfig) *docgen.BrowserRenderMargins {
	if cfg == nil {
		return nil
	}
	return &docgen.BrowserRenderMargins{
		Top:    cfg.MarginTop,
		Right:  cfg.MarginRight,
		Bottom: cfg.MarginBottom,
		Left:   cfg.MarginLeft,
	}
}

func mddmBlocksToHTML(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	var envelope struct {
		Blocks []json.RawMessage `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		return ""
	}

	var sb strings.Builder
	for _, block := range envelope.Blocks {
		sb.WriteString(mddmBlockHTML(block))
	}
	return sb.String()
}

func mddmBlockHTML(raw json.RawMessage) string {
	var node struct {
		Type     string            `json:"type"`
		Props    map[string]any    `json:"props"`
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(raw, &node); err != nil {
		return ""
	}

	leafText := func() string {
		var parts []string
		for _, child := range node.Children {
			var leaf struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(child, &leaf); err == nil && strings.TrimSpace(leaf.Text) != "" {
				parts = append(parts, html.EscapeString(strings.TrimSpace(leaf.Text)))
			}
		}
		return strings.Join(parts, "")
	}

	childrenHTML := func() string {
		var sb strings.Builder
		for _, child := range node.Children {
			sb.WriteString(mddmBlockHTML(child))
		}
		return sb.String()
	}

	switch node.Type {
	case "paragraph":
		text := leafText()
		if text == "" {
			return "<p>&nbsp;</p>"
		}
		return "<p>" + text + "</p>"
	case "heading":
		level := 2
		if rawLevel, ok := node.Props["level"].(float64); ok && rawLevel >= 1 && rawLevel <= 6 {
			level = int(rawLevel)
		}
		text := leafText()
		if text == "" {
			text = "&nbsp;"
		}
		return fmt.Sprintf("<h%d>%s</h%d>", level, text, level)
	case "section":
		title := ""
		if rawTitle, ok := node.Props["title"].(string); ok {
			title = html.EscapeString(strings.TrimSpace(rawTitle))
		}
		var sb strings.Builder
		if title != "" {
			sb.WriteString(fmt.Sprintf(
				`<table style="width:100%%;border-collapse:collapse;margin-bottom:0.75rem;">`+
					`<tr><td style="background-color:#6b1f2a;color:#fff;padding:8px 14px;font-size:13px;font-weight:700;">%s</td></tr></table>`,
				title,
			))
		}
		sb.WriteString(childrenHTML())
		return sb.String()
	case "field":
		label := ""
		if rawLabel, ok := node.Props["label"].(string); ok {
			label = html.EscapeString(strings.TrimSpace(rawLabel))
		}
		content := childrenHTML()
		if content == "" {
			content = "<p>&nbsp;</p>"
		}
		return fmt.Sprintf(
			`<table style="width:100%%;border-collapse:collapse;margin-bottom:0.5rem;">`+
				`<tr><td style="width:30%%;background:#f9f3f3;border:1px solid #dfc8c8;padding:0.5rem;font-weight:600;">%s</td>`+
				`<td style="border:1px solid #dfc8c8;padding:0.5rem;">%s</td></tr></table>`,
			label, content,
		)
	case "bulletListItem", "numberedListItem":
		text := leafText()
		if text == "" {
			text = "&nbsp;"
		}
		return "<p>" + text + "</p>"
	default:
		return childrenHTML()
	}
}

// buildBrowserDocumentHeaderHTML produces the locked identity header block
// as a <table> so that HTMLtoDOCX can render it as a proper Word table with
// background colors and bordered cells. The React DocumentEditorHeader component
// handles the browser view; this function serves the DOCX export path only.
func buildBrowserDocumentHeaderHTML(doc domain.Document, version domain.Version) string {
	revision := fmt.Sprintf("Rev. %02d", version.Number)
	code := doc.DocumentCode
	if code == "" {
		code = "—"
	}
	createdAt := "—"
	if !doc.CreatedAt.IsZero() {
		createdAt = html.EscapeString(doc.CreatedAt.Format("02/01/2006"))
	}
	status := doc.Status
	if status == "" {
		status = "—"
	}
	owner := doc.OwnerID
	if owner == "" {
		owner = "—"
	}

	topCell := `background-color:#6b1f2a;color:#ffffff;padding:6px 14px;font-size:11px;font-weight:600;letter-spacing:1px;text-transform:uppercase;`
	metaCell := func(label, value, sep string) string {
		return fmt.Sprintf(
			`<td style="background-color:#3e1018;color:#ffffff;padding:6px 14px;%s">`+
				`<p style="margin:0;font-size:10px;font-weight:600;text-transform:uppercase;letter-spacing:1px;color:#b6a5a7;">%s</p>`+
				`<p style="margin:0;font-size:12px;font-weight:500;">%s</p>`+
				`</td>`,
			sep, label, value,
		)
	}
	sep := `border-right:1px solid rgba(255,255,255,0.18);`

	return fmt.Sprintf(
		`<table class="md-doc-header" style="width:100%%;border-collapse:collapse;margin-bottom:2rem;font-family:DM Sans,sans-serif;">`+
			`<tr>`+
			`<td colspan="4" style="%s">Metal Nobre</td>`+
			`<td style="%sfont-size:11px;font-weight:600;text-align:right;white-space:nowrap;">%s · %s</td>`+
			`</tr>`+
			`<tr>`+
			`<td colspan="5" style="background-color:#3e1018;color:#ffffff;padding:10px 14px 6px;font-size:16px;font-weight:700;line-height:1.35;">%s</td>`+
			`</tr>`+
			`<tr>`+
			`%s%s%s%s%s`+
			`</tr>`+
			`</table>`,
		topCell,
		`background-color:#6b1f2a;color:#ffffff;padding:6px 14px;`,
		html.EscapeString(code),
		html.EscapeString(revision),
		html.EscapeString(doc.Title),
		metaCell("Tipo", html.EscapeString(doc.DocumentType), sep),
		metaCell("Elaborado por", html.EscapeString(owner), sep),
		metaCell("Data", createdAt, sep),
		metaCell("Status", html.EscapeString(status), sep),
		metaCell("Aprovado por", "—", ""),
	)
}
