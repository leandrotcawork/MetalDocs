package application

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

type projectedRichRun struct {
	Text      string `json:"text"`
	Bold      bool   `json:"bold,omitempty"`
	Italic    bool   `json:"italic,omitempty"`
	Underline bool   `json:"underline,omitempty"`
	Color     string `json:"color,omitempty"`
}

type projectedRichBlock struct {
	Type     string             `json:"type"`
	Runs     []projectedRichRun `json:"runs,omitempty"`
	Data     string             `json:"data,omitempty"`
	MimeType string             `json:"mimeType,omitempty"`
	AltText  string             `json:"altText,omitempty"`
	Width    int                `json:"width,omitempty"`
	Height   int                `json:"height,omitempty"`
	Ordered  bool               `json:"ordered,omitempty"`
	Items    []string           `json:"items,omitempty"`
	Rows     [][]string         `json:"rows,omitempty"`
	Header   bool               `json:"header,omitempty"`
}

func parseRichEnvelope(value any) (domain.RichEnvelope, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return domain.RichEnvelope{}, domain.ErrInvalidNativeContent
	}

	var envelope domain.RichEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return domain.RichEnvelope{}, domain.ErrInvalidNativeContent
	}
	if strings.TrimSpace(envelope.Format) != domain.RichEnvelopeFormatTipTap {
		return domain.RichEnvelope{}, domain.ErrInvalidNativeContent
	}
	if envelope.Version != domain.RichEnvelopeVersionV1 {
		return domain.RichEnvelope{}, domain.ErrInvalidNativeContent
	}
	if len(envelope.Content) == 0 {
		return domain.RichEnvelope{}, domain.ErrInvalidNativeContent
	}
	if kind := toRuntimeStringFallback(envelope.Content["type"], ""); kind != "doc" {
		return domain.RichEnvelope{}, domain.ErrInvalidNativeContent
	}
	if _, ok := envelope.Content["content"]; !ok {
		return domain.RichEnvelope{}, domain.ErrInvalidNativeContent
	}
	return envelope, nil
}

func validateRichEnvelopeValue(value any) error {
	_, err := parseRichEnvelope(value)
	return err
}

func (s *Service) projectDocumentValuesForDocgen(schema map[string]any, values map[string]any) (map[string]any, error) {
	projected := cloneRuntimeValues(values)
	if len(schema) == 0 || len(projected) == 0 {
		return projected, nil
	}

	for _, rawSection := range toRuntimeMapSlice(schema["sections"]) {
		section, ok := rawSection.(map[string]any)
		if !ok {
			return nil, domain.ErrInvalidCommand
		}
		sectionKey, ok := toRuntimeString(section["key"])
		if !ok {
			return nil, domain.ErrInvalidCommand
		}

		sectionValue, hasSection := projected[sectionKey].(map[string]any)
		if !hasSection {
			sectionValue = map[string]any{}
		}

		fields := toRuntimeMapSlice(section["fields"])
		for _, rawField := range fields {
			field, ok := rawField.(map[string]any)
			if !ok {
				return nil, domain.ErrInvalidCommand
			}
			if err := projectDocumentFieldValue(field, projected, sectionValue, true); err != nil {
				return nil, err
			}
		}

		if hasSection {
			projected[sectionKey] = sectionValue
		}
	}

	return projected, nil
}

func projectDocumentFieldValue(field map[string]any, root, container map[string]any, allowRootFallback bool) error {
	key, ok := toRuntimeString(field["key"])
	if !ok {
		return domain.ErrInvalidCommand
	}
	fieldType, _ := toRuntimeString(field["type"])
	value, exists, source := lookupProjectedValue(root, container, key, allowRootFallback)
	if !exists || isEmptyContentValue(value) {
		return nil
	}

	switch fieldType {
	case "rich":
		projected, err := projectRichFieldValue(value)
		if err != nil {
			return err
		}
		source[key] = projected
	case "table":
		rows, ok := value.([]any)
		if !ok {
			return domain.ErrInvalidNativeContent
		}
		for _, rawRow := range rows {
			row, ok := rawRow.(map[string]any)
			if !ok {
				return domain.ErrInvalidNativeContent
			}
			for _, rawColumn := range toRuntimeMapSlice(field["columns"]) {
				column, ok := rawColumn.(map[string]any)
				if !ok {
					return domain.ErrInvalidCommand
				}
				if err := projectDocumentFieldValue(column, row, row, false); err != nil {
					return err
				}
			}
		}
	case "repeat":
		items, ok := value.([]any)
		if !ok {
			return domain.ErrInvalidNativeContent
		}
		for _, rawItem := range items {
			item, ok := rawItem.(map[string]any)
			if !ok {
				return domain.ErrInvalidNativeContent
			}
			for _, rawItemField := range toRuntimeMapSlice(field["itemFields"]) {
				itemField, ok := rawItemField.(map[string]any)
				if !ok {
					return domain.ErrInvalidCommand
				}
				if err := projectDocumentFieldValue(itemField, item, item, false); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func lookupProjectedValue(root, container map[string]any, key string, allowRootFallback bool) (any, bool, map[string]any) {
	if container != nil {
		if value, exists := container[key]; exists {
			return value, true, container
		}
	}
	if allowRootFallback && root != nil {
		if value, exists := root[key]; exists {
			return value, true, root
		}
	}
	return nil, false, nil
}

func projectRichFieldValue(value any) ([]projectedRichBlock, error) {
	envelope, err := parseRichEnvelope(value)
	if err != nil {
		return nil, err
	}
	return projectTipTapDocument(envelope.Content)
}

func projectTipTapDocument(doc map[string]any) ([]projectedRichBlock, error) {
	nodes := toRuntimeMapSlice(doc["content"])
	if len(nodes) == 0 {
		return []projectedRichBlock{}, nil
	}
	return projectTipTapNodeSequence(nodes)
}

func projectTipTapNodeSequence(nodes []any) ([]projectedRichBlock, error) {
	out := make([]projectedRichBlock, 0)
	for _, rawNode := range nodes {
		blocks, err := projectTipTapNode(rawNode)
		if err != nil {
			return nil, err
		}
		out = append(out, blocks...)
	}
	return out, nil
}

func projectTipTapNode(rawNode any) ([]projectedRichBlock, error) {
	node, ok := rawNode.(map[string]any)
	if !ok {
		return nil, domain.ErrInvalidNativeContent
	}

	nodeType := strings.ToLower(strings.TrimSpace(toRuntimeStringFallback(node["type"], "")))
	switch nodeType {
	case "doc":
		return projectTipTapNodeSequence(toRuntimeMapSlice(node["content"]))
	case "paragraph", "heading", "blockquote":
		return projectTipTapTextContainer(node)
	case "bulletlist":
		block, err := projectTipTapListBlock(node, false)
		if err != nil {
			return nil, err
		}
		return []projectedRichBlock{block}, nil
	case "orderedlist":
		block, err := projectTipTapListBlock(node, true)
		if err != nil {
			return nil, err
		}
		return []projectedRichBlock{block}, nil
	case "table":
		block, err := projectTipTapTableBlock(node)
		if err != nil {
			return nil, err
		}
		return []projectedRichBlock{block}, nil
	case "image":
		block, err := projectTipTapImageBlock(node)
		if err != nil {
			return nil, err
		}
		return []projectedRichBlock{block}, nil
	}

	if content := toRuntimeMapSlice(node["content"]); len(content) > 0 {
		return projectTipTapNodeSequence(content)
	}
	if text, ok := toRuntimeString(node["text"]); ok {
		runs, err := projectedRichRunsFromTextNode(node)
		if err != nil {
			return nil, err
		}
		if len(runs) == 0 && text == "" {
			return []projectedRichBlock{}, nil
		}
		return []projectedRichBlock{{Type: "text", Runs: runs}}, nil
	}

	return nil, domain.ErrInvalidNativeContent
}

func projectTipTapTextContainer(node map[string]any) ([]projectedRichBlock, error) {
	blocks := make([]projectedRichBlock, 0)
	runs := make([]projectedRichRun, 0)

	flushRuns := func(forceEmpty bool) {
		if len(runs) > 0 {
			blockRuns := append([]projectedRichRun(nil), runs...)
			blocks = append(blocks, projectedRichBlock{Type: "text", Runs: blockRuns})
			runs = runs[:0]
			return
		}
		if forceEmpty && len(blocks) == 0 {
			blocks = append(blocks, projectedRichBlock{Type: "text", Runs: []projectedRichRun{}})
		}
	}

	for _, rawChild := range toRuntimeMapSlice(node["content"]) {
		child, ok := rawChild.(map[string]any)
		if !ok {
			return nil, domain.ErrInvalidNativeContent
		}
		childType := strings.ToLower(strings.TrimSpace(toRuntimeStringFallback(child["type"], "")))
		switch childType {
		case "text":
			childRuns, err := projectedRichRunsFromTextNode(child)
			if err != nil {
				return nil, err
			}
			runs = append(runs, childRuns...)
		case "hardbreak":
			if len(runs) == 0 {
				runs = append(runs, projectedRichRun{Text: "\n"})
				break
			}
			runs[len(runs)-1].Text += "\n"
		case "image":
			flushRuns(false)
			block, err := projectTipTapImageBlock(child)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, block)
		default:
			childBlocks, err := projectTipTapNode(child)
			if err != nil {
				return nil, err
			}
			flushRuns(false)
			blocks = append(blocks, childBlocks...)
		}
	}

	flushRuns(true)
	return blocks, nil
}

func projectedRichRunsFromTextNode(node map[string]any) ([]projectedRichRun, error) {
	text, ok := toRuntimeString(node["text"])
	if !ok {
		return nil, domain.ErrInvalidNativeContent
	}

	run := projectedRichRun{Text: text}
	marks := toRuntimeMapSlice(node["marks"])
	for _, rawMark := range marks {
		mark, ok := rawMark.(map[string]any)
		if !ok {
			return nil, domain.ErrInvalidNativeContent
		}
		switch strings.ToLower(strings.TrimSpace(toRuntimeStringFallback(mark["type"], ""))) {
		case "bold":
			run.Bold = true
		case "italic":
			run.Italic = true
		case "underline":
			run.Underline = true
		case "textstyle", "color":
			if color := projectedRichColor(mark); color != "" {
				run.Color = color
			}
		}
	}
	return []projectedRichRun{run}, nil
}

func projectedRichColor(mark map[string]any) string {
	if attrs, ok := mark["attrs"].(map[string]any); ok {
		if color, ok := toRuntimeString(attrs["color"]); ok {
			return normalizeDocgenColor(color)
		}
	}
	if color, ok := toRuntimeString(mark["color"]); ok {
		return normalizeDocgenColor(color)
	}
	return ""
}

func projectTipTapImageBlock(node map[string]any) (projectedRichBlock, error) {
	attrs, _ := node["attrs"].(map[string]any)
	if attrs == nil {
		attrs = map[string]any{}
	}
	src, ok := toRuntimeString(attrs["src"])
	if !ok {
		return projectedRichBlock{}, domain.ErrInvalidNativeContent
	}
	data, mimeType, err := decodeRichImageSource(src, attrs)
	if err != nil {
		return projectedRichBlock{}, err
	}

	altText := toRuntimeStringFallback(attrs["alt"], "")
	if altText == "" {
		altText = toRuntimeStringFallback(attrs["title"], "")
	}

	width := projectedRichInt(attrs["width"], 320)
	height := projectedRichInt(attrs["height"], 180)

	return projectedRichBlock{
		Type:     "image",
		Data:     data,
		MimeType: mimeType,
		AltText:  altText,
		Width:    width,
		Height:   height,
	}, nil
}

func decodeRichImageSource(src string, attrs map[string]any) (string, string, error) {
	trimmed := strings.TrimSpace(src)
	if trimmed == "" {
		return "", "", domain.ErrInvalidNativeContent
	}

	if strings.HasPrefix(trimmed, "data:") {
		parts := strings.SplitN(trimmed, ",", 2)
		if len(parts) != 2 {
			return "", "", domain.ErrInvalidNativeContent
		}
		meta := parts[0]
		payload := parts[1]
		mimeType := "image/png"
		if head := strings.TrimPrefix(meta, "data:"); head != "" {
			if semi := strings.Index(head, ";"); semi >= 0 {
				if candidate := strings.TrimSpace(head[:semi]); candidate != "" {
					mimeType = candidate
				}
			} else if candidate := strings.TrimSpace(head); candidate != "" {
				mimeType = candidate
			}
		}
		if mimeType == "image/webp" {
			return "", "", domain.ErrInvalidNativeContent
		}
		if !isBase64Payload(payload) {
			return "", "", domain.ErrInvalidNativeContent
		}
		return payload, mimeType, nil
	}

	if rawBase64, ok := attrs["data"].(string); ok && strings.TrimSpace(rawBase64) != "" {
		mimeType := toRuntimeStringFallback(attrs["mimeType"], "image/png")
		if mimeType == "image/webp" {
			return "", "", domain.ErrInvalidNativeContent
		}
		if !isBase64Payload(rawBase64) {
			return "", "", domain.ErrInvalidNativeContent
		}
		return strings.TrimSpace(rawBase64), mimeType, nil
	}

	return "", "", domain.ErrInvalidNativeContent
}

func isBase64Payload(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(value)
	return err == nil
}

func projectTipTapListBlock(node map[string]any, ordered bool) (projectedRichBlock, error) {
	items := make([]string, 0)
	for _, rawItem := range toRuntimeMapSlice(node["content"]) {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return projectedRichBlock{}, domain.ErrInvalidNativeContent
		}
		itemText, err := flattenTipTapText(item)
		if err != nil {
			return projectedRichBlock{}, err
		}
		if trimmed := strings.TrimSpace(itemText); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return projectedRichBlock{
		Type:    "list",
		Ordered: ordered,
		Items:   items,
	}, nil
}

func projectTipTapTableBlock(node map[string]any) (projectedRichBlock, error) {
	rows := make([][]string, 0)
	header := false
	for rowIndex, rawRow := range toRuntimeMapSlice(node["content"]) {
		row, ok := rawRow.(map[string]any)
		if !ok {
			return projectedRichBlock{}, domain.ErrInvalidNativeContent
		}
		cells := make([]string, 0)
		for _, rawCell := range toRuntimeMapSlice(row["content"]) {
			cell, ok := rawCell.(map[string]any)
			if !ok {
				return projectedRichBlock{}, domain.ErrInvalidNativeContent
			}
			if strings.EqualFold(toRuntimeStringFallback(cell["type"], ""), "tableHeader") && rowIndex == 0 {
				header = true
			}
			cellText, err := flattenTipTapText(cell)
			if err != nil {
				return projectedRichBlock{}, err
			}
			cells = append(cells, strings.TrimSpace(cellText))
		}
		if len(cells) > 0 {
			rows = append(rows, cells)
		}
	}
	return projectedRichBlock{
		Type:   "table",
		Rows:   rows,
		Header: header,
	}, nil
}

func flattenTipTapText(node map[string]any) (string, error) {
	var builder strings.Builder
	if err := appendTipTapText(&builder, node); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func appendTipTapText(builder *strings.Builder, node map[string]any) error {
	nodeType := strings.ToLower(strings.TrimSpace(toRuntimeStringFallback(node["type"], "")))
	switch nodeType {
	case "text":
		if text, ok := toRuntimeString(node["text"]); ok {
			builder.WriteString(text)
		}
		return nil
	case "hardbreak":
		builder.WriteString("\n")
		return nil
	case "image":
		if attrs, ok := node["attrs"].(map[string]any); ok {
			if alt := toRuntimeStringFallback(attrs["alt"], ""); alt != "" {
				builder.WriteString(alt)
				return nil
			}
			if title := toRuntimeStringFallback(attrs["title"], ""); title != "" {
				builder.WriteString(title)
			}
		}
		return nil
	}

	for _, rawChild := range toRuntimeMapSlice(node["content"]) {
		child, ok := rawChild.(map[string]any)
		if !ok {
			return domain.ErrInvalidNativeContent
		}
		if err := appendTipTapText(builder, child); err != nil {
			return err
		}
		if childType := strings.ToLower(strings.TrimSpace(toRuntimeStringFallback(child["type"], ""))); childType == "paragraph" || childType == "heading" || childType == "blockquote" || childType == "listitem" {
			builder.WriteString(" ")
		}
	}

	return nil
}

func projectedRichInt(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case int32:
		if typed > 0 {
			return int(typed)
		}
	case int64:
		if typed > 0 {
			return int(typed)
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case float32:
		if typed > 0 {
			return int(typed)
		}
	}
	return fallback
}

func normalizeDocgenColor(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.TrimPrefix(strings.ToUpper(trimmed), "#")
}

func toRuntimeMapSlice(value any) []any {
	slice, ok := toRuntimeSlice(value)
	if !ok {
		return []any{}
	}
	return slice
}

func toRuntimeString(value any) (string, bool) {
	raw, ok := value.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func toRuntimeStringFallback(value any, fallback string) string {
	if resolved, ok := toRuntimeString(value); ok {
		return resolved
	}
	return fallback
}
