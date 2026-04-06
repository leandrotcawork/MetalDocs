package application

import (
	"strings"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
)

func TestBuildBrowserDocumentHeaderHTML(t *testing.T) {
	doc := domain.Document{
		DocumentCode: "PO-110",
		Title:        "Test Document",
		DocumentType: "Procedimento Operacional",
		OwnerID:      "owner-1",
		Status:       "rascunho",
		CreatedAt:    time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC),
	}
	version := domain.Version{Number: 3}
	result := buildBrowserDocumentHeaderHTML(doc, version)

	checks := []string{
		`class="md-doc-header"`,
		`PO-110`,
		`Rev. 03`,
		`Test Document`,
		`Procedimento Operacional`,
		`06/04/2026`,
		`rascunho`,
	}
	for _, want := range checks {
		if !strings.Contains(result, want) {
			t.Errorf("header HTML missing %q", want)
		}
	}
}

func TestBuildBrowserDocumentHeaderHTMLEmptyFields(t *testing.T) {
	doc := domain.Document{
		Title: "Minimal Document",
	}
	version := domain.Version{Number: 1}
	result := buildBrowserDocumentHeaderHTML(doc, version)

	if !strings.Contains(result, `class="md-doc-header"`) {
		t.Error("header HTML missing md-doc-header class")
	}
	if !strings.Contains(result, `Rev. 01`) {
		t.Error("header HTML missing Rev. 01")
	}
	if !strings.Contains(result, `Minimal Document`) {
		t.Error("header HTML missing title")
	}
	// Empty fields should fall back to em dash
	if !strings.Contains(result, `—`) {
		t.Error("header HTML missing em dash fallback for empty fields")
	}
}

func TestBuildBrowserDocumentHeaderHTMLEscapesSpecialChars(t *testing.T) {
	doc := domain.Document{
		DocumentCode: `<script>alert("xss")</script>`,
		Title:        `A & B <Test>`,
		DocumentType: "PO",
		OwnerID:      "owner",
		Status:       "ativo",
		CreatedAt:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	version := domain.Version{Number: 1}
	result := buildBrowserDocumentHeaderHTML(doc, version)

	// Raw script tag must not appear unescaped
	if strings.Contains(result, `<script>`) {
		t.Error("header HTML must escape <script> in document code")
	}
	if !strings.Contains(result, `&lt;script&gt;`) {
		t.Error("header HTML must contain escaped script tag")
	}
}
