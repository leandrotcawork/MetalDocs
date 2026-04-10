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
		`<table`,
		`class="md-doc-header"`,
		`PO-110`,
		`Rev. 03`,
		`Test Document`,
		`Procedimento Operacional`,
		`06/04/2026`,
		`rascunho`,
		`Metal Nobre`,
		`Tipo`,
		`Elaborado por`,
		`Data`,
		`Status`,
		`Aprovado por`,
	}
	for _, want := range checks {
		if !strings.Contains(result, want) {
			t.Errorf("header HTML missing %q", want)
		}
	}
	if strings.Contains(result, `<div class="md-doc-header"`) {
		t.Error("header must use <table>, not <div>")
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
	if !strings.Contains(result, `A &amp; B &lt;Test&gt;`) {
		t.Errorf("header HTML must escape & and <> in title, got: %q", result)
	}
}

func TestBrowserRenderMarginsFromExportConfig(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		result := browserRenderMarginsFromExportConfig(nil)
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})
	t.Run("non-nil config maps all fields", func(t *testing.T) {
		cfg := &domain.TemplateExportConfig{
			MarginTop: 0.625, MarginRight: 0.625,
			MarginBottom: 0.625, MarginLeft: 0.625,
		}
		result := browserRenderMarginsFromExportConfig(cfg)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Top != 0.625 {
			t.Errorf("Top = %v, want 0.625", result.Top)
		}
		if result.Right != 0.625 {
			t.Errorf("Right = %v, want 0.625", result.Right)
		}
		if result.Bottom != 0.625 {
			t.Errorf("Bottom = %v, want 0.625", result.Bottom)
		}
		if result.Left != 0.625 {
			t.Errorf("Left = %v, want 0.625", result.Left)
		}
	})
}

func TestMDDMBlocksToHTML(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		want     string
		wantSubs []string
	}{
		{
			name: "empty body",
			body: "",
			want: "",
		},
		{
			name:     "single paragraph",
			body:     `{"mddm_version":1,"template_ref":null,"blocks":[{"id":"b1","type":"paragraph","props":{},"children":[{"text":"Hello"}]}]}`,
			wantSubs: []string{"<p>Hello</p>"},
		},
		{
			name:     "section with nested field",
			body:     `{"mddm_version":1,"template_ref":null,"blocks":[{"id":"s1","type":"section","props":{"title":"My Section"},"children":[{"id":"f1","type":"field","props":{"label":"Field"},"children":[{"id":"p1","type":"paragraph","props":{},"children":[{"text":"Value"}]}]}]}]}`,
			wantSubs: []string{"My Section", "Field", "<p>Value</p>"},
		},
		{
			name:     "heading level two",
			body:     `{"mddm_version":1,"template_ref":null,"blocks":[{"id":"h1","type":"heading","props":{"level":2},"children":[{"text":"Title"}]}]}`,
			wantSubs: []string{"<h2>Title</h2>"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mddmBlocksToHTML(tc.body)
			if tc.want != "" && got != tc.want {
				t.Fatalf("mddmBlocksToHTML(%q) = %q, want %q", tc.body, got, tc.want)
			}
			for _, sub := range tc.wantSubs {
				if !strings.Contains(got, sub) {
					t.Fatalf("mddmBlocksToHTML(%q) missing substring %q; got %q", tc.body, sub, got)
				}
			}
		})
	}
}
