package application

import (
	"fmt"
	"html"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

// Test-only compatibility shim for legacy token-based assertions that still
// compile in this package while the production browser editor path has moved on.
func substituteTemplateTokens(body string, doc domain.Document, version domain.Version) string {
	versao := fmt.Sprintf("%02d", version.Number)
	data := "—"
	if !doc.CreatedAt.IsZero() {
		data = doc.CreatedAt.Format("02/01/2006")
	}
	por := html.EscapeString(doc.OwnerID)
	if por == "" {
		por = "—"
	}
	body = strings.ReplaceAll(body, "{{versao}}", versao)
	body = strings.ReplaceAll(body, "{{data_criacao}}", data)
	body = strings.ReplaceAll(body, "{{elaborador}}", por)
	return body
}
