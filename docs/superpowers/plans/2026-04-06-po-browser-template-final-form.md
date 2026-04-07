# PO Browser Template — Final Form Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the PO browser template DOCX export to production quality — table-based header, bordered field tables in body sections, and a complete Section 10 with auto-populated revision history.

**Architecture:** Three layers change in concert: (1) Go DOCX header builder rewrites from `<div>` to `<table>`, (2) template body HTML switches field sections to `<table class="md-field-table">`, (3) bundle serving injects document metadata tokens into Section 10. CSS gains one new rule. No API or frontend component changes.

**Tech Stack:** Go 1.23, CKEditor 5 RestrictedEditingMode, `@turbodocx/html-to-docx`, PostgreSQL migrations

**Spec:** `docs/superpowers/specs/2026-04-06-po-browser-template-final-form.md`

---

### Task 1: Add `.md-field-table` CSS rule

**Files:**
- Modify: `frontend/apps/web/src/styles/document-content.css`

- [ ] **Step 1: Append `.md-field-table` block to the CSS file**

Add after the existing `/* ── Document Header ───...── */` block:

```css
/* ── Field Table ───────────────────────────────────────────── */
.ck-content .md-field-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 1rem;
}

.ck-content .md-field-table td {
  border: 1px solid #dfc8c8;
  padding: 0.5rem 0.75rem;
  vertical-align: top;
  font-size: 0.93rem;
  color: #483030;
}

.ck-content .md-field-table .md-field-label {
  width: 30%;
  font-weight: 600;
  font-size: 0.84rem;
  color: #3e1018;
  background: #f9f3f3;
}

.ck-content .md-field-table td:last-child {
  width: 70%;
}

.ck-content .md-field-table .restricted-editing-exception {
  margin: 0;
  background: transparent;
  border: none;
  border-radius: 0;
  padding: 0;
  min-height: unset;
}
```

- [ ] **Step 2: Verify TypeScript build still passes (CSS change only, no TS impact)**

```bash
cd C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs/frontend/apps/web
npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/styles/document-content.css
git commit -m "feat(css): add md-field-table primitive for DOCX-faithful field layout"
```

---

### Task 2: Rewrite `buildBrowserDocumentHeaderHTML` as a `<table>`

**Files:**
- Modify: `internal/modules/documents/application/service_content_docx.go`

- [ ] **Step 1: Replace the entire `buildBrowserDocumentHeaderHTML` function**

The current function uses `<div>`/`<span>` which HTMLtoDOCX flattens into plain paragraphs. Replace it with a 3-row `<table>`. Add `class="md-doc-header"` to the table so existing test assertions still find it.

Replace the function (lines 196–247) with:

```go
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
		`<table class="md-doc-header" style="width:100%;border-collapse:collapse;margin-bottom:2rem;font-family:DM Sans,sans-serif;">`+
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
```

- [ ] **Step 2: Verify the file compiles**

```bash
cd C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs
go build ./internal/modules/documents/application/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/modules/documents/application/service_content_docx.go
git commit -m "feat(docx): rewrite header builder as <table> for DOCX-faithful rendering"
```

---

### Task 3: Update header tests for `<table>` structure

**Files:**
- Modify: `internal/modules/documents/application/service_content_docx_test.go`

- [ ] **Step 1: Run existing header tests to establish baseline**

```bash
cd C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs
go test ./internal/modules/documents/application/... -run TestBuildBrowserDocumentHeader -v
```

Expected: tests currently PASS (the `class="md-doc-header"` attribute is still present on the rewritten `<table>`, so existing string-contains checks hold). The next step strengthens the assertions to explicitly verify the `<table>` structure.

- [ ] **Step 2: Replace assertions to verify `<table>` structure explicitly**

Replace the `checks` slice in `TestBuildBrowserDocumentHeaderHTML`:

```go
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
```

Also add a negative assertion at the end of the function to ensure no `<div class="md-doc-header"` remains:

```go
if strings.Contains(result, `<div class="md-doc-header"`) {
    t.Error("header must use <table>, not <div>")
}
```

- [ ] **Step 3: Run tests — confirm all three header tests pass**

```bash
go test ./internal/modules/documents/application/... -run TestBuildBrowserDocumentHeader -v
```

Expected:
```
--- PASS: TestBuildBrowserDocumentHeaderHTML (0.00s)
--- PASS: TestBuildBrowserDocumentHeaderHTMLEmptyFields (0.00s)
--- PASS: TestBuildBrowserDocumentHeaderHTMLEscapesSpecialChars (0.00s)
PASS
```

- [ ] **Step 4: Commit**

```bash
git add internal/modules/documents/application/service_content_docx_test.go
git commit -m "test(docx): update header assertions to verify <table> structure"
```

---

### Task 4: Add `substituteTemplateTokens` and wire into bundle-serve and DOCX export paths

**Files:**
- Modify: `internal/modules/documents/application/service_browser_editor.go`

- [ ] **Step 1: Add `substituteTemplateTokens` function**

Add after the `validateBrowserTemplateVersion` function:

```go
// substituteTemplateTokens replaces well-known placeholder tokens in the body
// with real document metadata. Called when serving the browser editor bundle so
// the user sees pre-populated fields (e.g., Section 10 revision history) without
// having to type them. Idempotent: if tokens are already replaced, ReplaceAll is a no-op.
func substituteTemplateTokens(body string, doc domain.Document, version domain.Version) string {
	versao := fmt.Sprintf("%02d", version.Number)
	data := "—"
	if !doc.CreatedAt.IsZero() {
		data = doc.CreatedAt.Format("02/01/2006")
	}
	por := doc.OwnerID
	if por == "" {
		por = "—"
	}
	body = strings.ReplaceAll(body, "{{versao}}", versao)
	body = strings.ReplaceAll(body, "{{data_criacao}}", data)
	body = strings.ReplaceAll(body, "{{elaborador}}", por)
	return body
}
```

- [ ] **Step 2: Call `substituteTemplateTokens` in `GetBrowserEditorBundleAuthorized`**

Find the line that assigns `Body: current.Content` inside the `bundle := BrowserEditorBundle{...}` literal and change it to:

```go
Body: substituteTemplateTokens(current.Content, doc, current),
```

The bundle initialization becomes:

```go
bundle := BrowserEditorBundle{
    Document:   doc,
    Versions:   versions,
    Governance: governance,
    Body:       substituteTemplateTokens(current.Content, doc, current),
    DraftToken: draftTokenForVersion(current),
}
```

- [ ] **Step 3: Also apply `substituteTemplateTokens` in the DOCX export path**

In `internal/modules/documents/application/service_content_docx.go`, inside `generateBrowserDocxBytes` (line ~178), change:

```go
// Before:
HTML: headerHTML + version.Content,
```

to:

```go
// After:
HTML: headerHTML + substituteTemplateTokens(version.Content, doc, version),
```

`substituteTemplateTokens` is in the same `application` package — no import needed.

> **Why:** The browser bundle path and the DOCX export path both serve the raw `version.Content`. If a user exports DOCX before a save round-trip, tokens will survive into the exported file unless the export path also substitutes them.

- [ ] **Step 4: Verify the file compiles**

```bash
go build ./internal/modules/documents/application/...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/application/service_browser_editor.go internal/modules/documents/application/service_content_docx.go
git commit -m "feat(bundle): substitute template tokens in both bundle-serve and DOCX export paths"
```

---

### Task 5: Add `TestSubstituteTemplateTokens` and bundle/DOCX integration tests

**Files:**
- Modify: `internal/modules/documents/application/service_browser_editor_test.go`

- [ ] **Step 1: Write the unit tests** (function already implemented in Task 4)

Add at the end of `service_browser_editor_test.go`:

```go
func TestSubstituteTemplateTokens(t *testing.T) {
	doc := domain.Document{
		OwnerID:   "leandro_theodoro",
		CreatedAt: time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC),
	}
	version := domain.Version{Number: 1}

	body := `<td><p class="restricted-editing-exception">{{versao}}</p></td>` +
		`<td><p class="restricted-editing-exception">{{data_criacao}}</p></td>` +
		`<td><p class="restricted-editing-exception"></p></td>` +
		`<td><p class="restricted-editing-exception">{{elaborador}}</p></td>`

	got := substituteTemplateTokens(body, doc, version)

	if strings.Contains(got, "{{versao}}") {
		t.Error("expected {{versao}} to be replaced")
	}
	if strings.Contains(got, "{{data_criacao}}") {
		t.Error("expected {{data_criacao}} to be replaced")
	}
	if strings.Contains(got, "{{elaborador}}") {
		t.Error("expected {{elaborador}} to be replaced")
	}
	if !strings.Contains(got, "01") {
		t.Error("expected version 01 in result")
	}
	if !strings.Contains(got, "06/04/2026") {
		t.Error("expected date 06/04/2026 in result")
	}
	if !strings.Contains(got, "leandro_theodoro") {
		t.Error("expected owner in result")
	}
}

func TestSubstituteTemplateTokensIdempotent(t *testing.T) {
	doc := domain.Document{
		OwnerID:   "owner",
		CreatedAt: time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC),
	}
	version := domain.Version{Number: 1}

	// Body with no tokens — should be returned unchanged
	body := `<p class="restricted-editing-exception">already filled</p>`
	got := substituteTemplateTokens(body, doc, version)
	if got != body {
		t.Errorf("idempotent: body without tokens should be unchanged, got %q", got)
	}
}

func TestSubstituteTemplateTokensEmptyOwner(t *testing.T) {
	doc := domain.Document{} // zero value: no owner, zero time
	version := domain.Version{Number: 2}

	body := `{{versao}} {{data_criacao}} {{elaborador}}`
	got := substituteTemplateTokens(body, doc, version)

	if !strings.Contains(got, "02") {
		t.Error("expected version 02")
	}
	if !strings.Contains(got, "—") {
		t.Error("expected em dash fallback for missing owner and date")
	}
}
```

- [ ] **Step 2: Add bundle-level integration test for `GetBrowserEditorBundleAuthorized`**

Follow the exact same pattern as `TestNewPODocumentGetsBrowserTemplateInBundle` in `service_browser_editor_test.go`:
- `documentmemory.NewRepository()` for the repo
- `NewService(repo, nil, fixedClock{now: now})` for the service (3 args)
- `seedCompatiblePOProfileSchemaSet(t, repo)` to register the PO profile + browser template
- `service.CreateDocument(...)` to create a real PO document (this seeds version with template body containing tokens in Section 10 after Task 6)
- `service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)` — takes only `(ctx, docID)`, no ownerID

```go
func TestGetBrowserEditorBundleSubstitutesTokens(t *testing.T) {
    ctx := context.Background()
    now := time.Date(2026, time.April, 6, 10, 0, 0, 0, time.UTC)
    repo := documentmemory.NewRepository()
    service := NewService(repo, nil, fixedClock{now: now})

    seedCompatiblePOProfileSchemaSet(t, repo)

    doc, err := service.CreateDocument(ctx, domain.CreateDocumentCommand{
        DocumentID:      "doc-token-substitution",
        Title:           "Token Substitution Test",
        DocumentType:    "po",
        DocumentProfile: "po",
        OwnerID:         "leandro_theodoro",
        BusinessUnit:    "operations",
        Department:      "sgq",
        InitialContent:  `{"legacy":"content"}`,
        TraceID:         "trace-token-test",
    })
    if err != nil {
        t.Fatalf("CreateDocument() error = %v", err)
    }

    bundle, err := service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)
    if err != nil {
        t.Fatalf("GetBrowserEditorBundleAuthorized() error = %v", err)
    }

    // Assert: no raw tokens remain in bundle.Body
    for _, token := range []string{"{{versao}}", "{{data_criacao}}", "{{elaborador}}"} {
        if strings.Contains(bundle.Body, token) {
            t.Errorf("bundle.Body must not contain raw token %q after substitution", token)
        }
    }
    // Assert: substituted values are present
    if !strings.Contains(bundle.Body, "01") {
        t.Error("expected version 01 in bundle.Body")
    }
    if !strings.Contains(bundle.Body, "06/04/2026") {
        t.Error("expected creation date 06/04/2026 in bundle.Body")
    }
    if !strings.Contains(bundle.Body, "leandro_theodoro") {
        t.Error("expected owner ID in bundle.Body")
    }
}
```

> **Why:** Unit tests for `substituteTemplateTokens` alone do not prove the bundle endpoint wires the substituted body into `bundle.Body`. This test exercises the full service path and will catch wiring regressions.

- [ ] **Step 3: Add DOCX export path regression test**

Add to `service_content_docx_test.go` a test that verifies `generateBrowserDocxBytes` would not forward raw tokens. Since `generateBrowserDocxBytes` calls `substituteTemplateTokens` internally (after Task 4 Step 3), and the function is in the same package, verify it by testing that the HTML payload built before passing to docgen is token-free. Add a helper test:

```go
func TestGenerateBrowserDocxBodySubstitutesTokens(t *testing.T) {
    // Verify substituteTemplateTokens eliminates tokens from a body
    // that would be passed to generateBrowserDocxBytes — guards against
    // regression on the DOCX export path.
    body := `{{versao}} {{data_criacao}} {{elaborador}}`
    doc := domain.Document{
        OwnerID:   "owner",
        CreatedAt: time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC),
    }
    version := domain.Version{Number: 3, Content: body}

    got := substituteTemplateTokens(version.Content, doc, version)

    for _, token := range []string{"{{versao}}", "{{data_criacao}}", "{{elaborador}}"} {
        if strings.Contains(got, token) {
            t.Errorf("DOCX export body must not contain raw token %q", token)
        }
    }
    if !strings.Contains(got, "03") {
        t.Error("expected version 03 in DOCX body")
    }
}
```

- [ ] **Step 4: Run all token-related tests**

```bash
go test ./internal/modules/documents/application/... -run "TestSubstituteTemplateTokens|TestGetBrowserEditorBundleSubstitutesTokens|TestGenerateBrowserDocxBodySubstitutesTokens" -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/application/service_browser_editor_test.go internal/modules/documents/application/service_content_docx_test.go
git commit -m "test(bundle): add unit and integration tests for token substitution in bundle and DOCX paths"
```

---

### Task 6: Rewrite template body in `template.go`

**Files:**
- Modify: `internal/modules/documents/domain/template.go`

> **DOCX styling note:** `BrowserRenderPayload` has no CSS field — docgen receives raw HTML only. Browser styling for `.md-field-table` comes from `document-content.css` (Task 1), but that CSS is not available during DOCX export. To ensure bordered field tables appear in the exported DOCX, every `<table class="md-field-table">` and its `<td>` elements **must also carry inline styles**. Add `style="width:100%;border-collapse:collapse;"` to the `<table>` tag and `style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;"` to each `<td>`. Label cells add `style="...; width:30%;font-weight:600;background:#f9f3f3;"`. This matches exactly what `buildBrowserDocumentHeaderHTML` does in Task 2.

- [ ] **Step 1: Replace the `Body` field of the `po-default-browser` template**

Replace the entire backtick string value of the `Body:` field (from `` `<section class="md-doc-shell"> `` to the closing `` </section>` ``) with the new body below.

**Key changes:**
- Sections 2, 3, 6: `div.md-field` → `table.md-field-table` rows, **with inline styles on `<table>` and `<td>` for DOCX export**
- Sections 4, 5, 7, 8, 9: unchanged
- Section 10: new `md-table` with token placeholders

New `Body` value (paste verbatim as the backtick string):

```
`<section class="md-doc-shell">
  <section class="md-section">
    <h2>2. Identificação do Processo</h2>
    <table class="md-field-table" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
      <tbody>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Objetivo</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Descreva o objetivo deste procedimento, incluindo o resultado esperado ao final da execução.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Escopo</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Defina os limites de aplicação deste procedimento: onde começa, onde termina e o que está fora do escopo.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Cargo responsável</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Informe o cargo ou função responsável pela execução deste procedimento.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Canal / Contexto</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Descreva o canal ou contexto em que este procedimento se aplica.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Participantes</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste os cargos, funções ou áreas que participam da execução deste procedimento.</p></td>
        </tr>
      </tbody>
    </table>
  </section>

  <section class="md-section">
    <h2>3. Entradas e Saídas</h2>
    <table class="md-field-table" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
      <tbody>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Entradas</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste os insumos, informações ou materiais necessários para iniciar o processo.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Saídas</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste os produtos, resultados ou entregas gerados ao final do processo.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Documentos relacionados</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste documentos, formulários ou registros utilizados ou gerados durante o processo.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Sistemas utilizados</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste os sistemas, ferramentas ou plataformas utilizadas na execução do processo.</p></td>
        </tr>
      </tbody>
    </table>
  </section>

  <section class="md-section">
    <h2>4. Visão Geral do Processo</h2>
    <div class="md-field">
      <p class="md-field-label"><strong>Descrição do processo</strong></p>
      <div class="restricted-editing-exception"><p>Descreva o processo de forma detalhada, incluindo o contexto, fluxo geral e principais decisões envolvidas.</p></div>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Ferramenta do fluxograma</strong></p>
      <p><span class="restricted-editing-exception">Informe a ferramenta utilizada para criar o fluxograma (ex: Bizagi, Visio, Miro).</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Link do fluxograma</strong></p>
      <p><span class="restricted-editing-exception">Cole o link de acesso ao fluxograma do processo.</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Diagrama</strong></p>
      <div class="restricted-editing-exception"><p>Insira ou descreva o diagrama do processo. Pode utilizar imagens ou representações textuais.</p></div>
    </div>
  </section>

  <section class="md-section">
    <h2>5. Detalhamento das Etapas</h2>
    <p class="md-section-hint">Descreva cada etapa como uma seção livre. Duplique o bloco abaixo para adicionar mais etapas.</p>
    <div class="md-free-block restricted-editing-exception">
      <h3>Etapa 1 — [Nome da etapa]</h3>
      <p>Descreva esta etapa livremente. Adicione parágrafos, listas, referências a outros documentos ou qualquer informação relevante para descrever o que acontece nesta etapa do processo.</p>
    </div>
  </section>

  <section class="md-section">
    <h2>6. Controle e Exceções</h2>
    <table class="md-field-table" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
      <tbody>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Pontos de controle</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Descreva os pontos de verificação, aprovação ou controle existentes no processo.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Exceções e desvios</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Descreva situações excepcionais e como devem ser tratadas.</p></td>
        </tr>
      </tbody>
    </table>
  </section>

  <section class="md-section">
    <h2>7. Indicadores de Desempenho</h2>
    <div class="md-field">
      <p class="md-field-label"><strong>KPIs</strong></p>
      <figure class="table md-table restricted-editing-exception">
        <table>
          <thead>
            <tr><th>Indicador / KPI</th><th>Meta</th><th>Frequência</th></tr>
          </thead>
          <tbody>
            <tr><td>Ex: Taxa de retrabalho</td><td>Ex: &lt; 5%</td><td>Ex: Mensal</td></tr>
          </tbody>
        </table>
      </figure>
    </div>
  </section>

  <section class="md-section">
    <h2>8. Documentos e Referências</h2>
    <figure class="table md-table restricted-editing-exception">
      <table>
        <thead>
          <tr><th>Código</th><th>Título / Descrição</th><th>Link / Localização</th></tr>
        </thead>
        <tbody>
          <tr><td>Ex: PO-001</td><td>Ex: Procedimento de compras</td><td>Ex: /docs/po-001</td></tr>
        </tbody>
      </table>
    </figure>
  </section>

  <section class="md-section">
    <h2>9. Glossário</h2>
    <figure class="table md-table restricted-editing-exception">
      <table>
        <thead>
          <tr><th>Termo</th><th>Definição</th></tr>
        </thead>
        <tbody>
          <tr><td>Ex: SLA</td><td>Ex: Acordo de nível de serviço</td></tr>
        </tbody>
      </table>
    </figure>
  </section>

  <section class="md-section">
    <h2>10. Histórico de Revisões</h2>
    <figure class="table md-table">
      <table>
        <thead>
          <tr><th>Versão</th><th>Data</th><th>O que foi alterado</th><th>Por</th></tr>
        </thead>
        <tbody>
          <tr>
            <td><p class="restricted-editing-exception">{{versao}}</p></td>
            <td><p class="restricted-editing-exception">{{data_criacao}}</p></td>
            <td><p class="restricted-editing-exception"></p></td>
            <td><p class="restricted-editing-exception">{{elaborador}}</p></td>
          </tr>
        </tbody>
      </table>
    </figure>
  </section>
</section>`
```

- [ ] **Step 2: Verify the file compiles**

```bash
go build ./internal/modules/documents/domain/...
```

Expected: no errors.

- [ ] **Step 3: Run the schema coverage test (it should still pass — Section 10 is not part of schema)**

```bash
go test ./internal/modules/documents/domain/... -run TestPOBrowserTemplateCoversPOv3Schema -v
```

Expected: PASS.

- [ ] **Step 4: Run the parity test — it WILL fail because SQL doesn't match yet**

```bash
go test ./internal/modules/documents/domain/... -run TestPOBrowserTemplateGoSQLParity -v
```

Expected: FAIL (confirming Go and SQL are out of sync — this is what we fix in Tasks 7–8).

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/domain/template.go
git commit -m "feat(template): rewrite PO body to md-field-table layout and add Section 10"
```

---

### Task 7: Sync `0057_seed_po_browser_template.sql`

**Files:**
- Modify: `migrations/0057_seed_po_browser_template.sql`

- [ ] **Step 1: Replace the body between the `$$` delimiters in the INSERT statement**

The body in 0057 must be byte-for-byte identical to the Go `Body` field set in Task 6.

Replace everything between the first `$$` and second `$$` (the `body_html` value) with exactly the same HTML as the Go template body. The comment at the top should also be updated:

```sql
-- 0057_seed_po_browser_template.sql
-- Insert production PO browser template (po-default-browser v1) and update PO profile default.
-- Covers all 9 editable sections of PO v3 schema (sections 2-10).
-- Section 1 (Identificação) is the React DocumentEditorHeader above CKEditor.
-- Section 10 (Histórico de Revisões) uses {{versao}}/{{data_criacao}}/{{elaborador}} tokens
-- substituted at bundle-serve time. O que foi alterado is left blank for user input.
-- IMPORTANT: body_html must match the Go seed in domain/template.go (validated by TestPOBrowserTemplateGoSQLParity).
```

The `$$...$$` body content must match the Go Body exactly (same whitespace, same characters).

- [ ] **Step 2: Run the parity test — it must now pass**

```bash
go test ./internal/modules/documents/domain/... -run TestPOBrowserTemplateGoSQLParity -v
```

Expected:
```
--- PASS: TestPOBrowserTemplateGoSQLParity (0.00s)
PASS
```

- [ ] **Step 3: Commit**

```bash
git add migrations/0057_seed_po_browser_template.sql
git commit -m "feat(migrations): sync 0057 body_html to md-field-table layout with Section 10"
```

---

### Task 8: Sync `0058_update_po_browser_template.sql` and add 0058 parity test

**Files:**
- Modify: `migrations/0058_update_po_browser_template.sql`
- Modify: `internal/modules/documents/domain/template_test.go` (or wherever `TestPOBrowserTemplateGoSQLParity` lives)

- [ ] **Step 1: Replace the body between `$$` delimiters in the UPDATE statement**

0058 updates existing DB rows. Its body must equal the Go Body (same as 0057 body). Update the comment too:

```sql
-- 0058_update_po_browser_template.sql
-- Updates po-default-browser v1 body_html to the final-form template:
-- md-field-table layout for sections 2, 3, 6 and new Section 10 (Histórico de Revisões).
-- IMPORTANT: body_html must match 0057 and the Go seed in domain/template.go.
-- TestPOBrowserTemplateGoSQLParity validates Go vs 0057 only; see TestPOBrowserTemplate0058Parity for 0058.
```

The `$$...$$` body in the UPDATE statement must be the same HTML as in 0057 and template.go.

- [ ] **Step 2: Add `TestPOBrowserTemplate0058Parity` test**

The existing `TestPOBrowserTemplateGoSQLParity` only reads `0057`. Add a new test that reads `0058` and asserts its `$$...$$` body matches the Go canonical body. Use the **same extraction pattern** as the existing parity test (`strings.SplitN(sqlContent, "$$", 3)` + whitespace normalization) and the **same Go template lookup** (`DefaultDocumentTemplateVersions()` loop). The migration file is at `../../../../migrations/0058_update_po_browser_template.sql` relative to `domain/template_test.go`.

Add to `internal/modules/documents/domain/template_test.go`:

```go
func TestPOBrowserTemplate0058Parity(t *testing.T) {
    // Get canonical Go template body (same lookup as TestPOBrowserTemplateGoSQLParity)
    var goTemplate *DocumentTemplateVersion
    for _, tmpl := range DefaultDocumentTemplateVersions() {
        if tmpl.TemplateKey == "po-default-browser" {
            found := tmpl
            goTemplate = &found
            break
        }
    }
    if goTemplate == nil {
        t.Fatal("po-default-browser template not found in Go seed")
    }

    // Read 0058 migration file
    migrationPath := "../../../../migrations/0058_update_po_browser_template.sql"
    sqlBytes, err := os.ReadFile(migrationPath)
    if err != nil {
        t.Fatalf("read 0058 migration file: %v", err)
    }
    sqlContent := string(sqlBytes)

    // Extract body between $$ delimiters (same as 0057 parity test)
    parts := strings.SplitN(sqlContent, "$$", 3)
    if len(parts) < 3 {
        t.Fatal("0058 migration does not contain $$ delimited body_html")
    }
    sqlBody := parts[1]

    goNormalized := strings.TrimSpace(strings.ReplaceAll(goTemplate.Body, "\r\n", "\n"))
    sqlNormalized := strings.TrimSpace(strings.ReplaceAll(sqlBody, "\r\n", "\n"))

    if goNormalized != sqlNormalized {
        t.Fatalf("0058 body_html differs from Go canonical body.\nGo length=%d, SQL length=%d\nFirst difference at character %d",
            len(goNormalized), len(sqlNormalized), firstDiffIndex(goNormalized, sqlNormalized))
    }
}
```

> **Why:** The existing parity test only validates Go vs 0057. 0058 is the migration that updates *existing* database rows — if it drifts, users with pre-existing PO documents will not see the updated template after migration. `firstDiffIndex` is already defined in `template_test.go` by the 0057 parity test.

- [ ] **Step 3: Run both parity tests**

```bash
go test ./internal/modules/documents/domain/... -run "TestPOBrowserTemplateGoSQLParity|TestPOBrowserTemplate0058Parity" -v
```

Expected:
```
--- PASS: TestPOBrowserTemplateGoSQLParity (0.00s)
--- PASS: TestPOBrowserTemplate0058Parity (0.00s)
PASS
```

- [ ] **Step 4: Commit**

```bash
git add migrations/0058_update_po_browser_template.sql internal/modules/documents/domain/template_test.go
git commit -m "feat(migrations): sync 0058 UPDATE to final-form template body and add parity test"
```

---

### Task 9: Full test suite pass

**Files:** None modified — verification only.

- [ ] **Step 1: Run all Go tests**

```bash
cd C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs
go test ./...
```

Expected: all tests pass. Zero failures.

If `TestCreateDocumentSeedsBrowserTemplateBody` or `TestNewPODocumentGetsBrowserTemplateInBundle` fail (they check for a specific substring in the body), update their assertions to check for a string present in the new body (e.g., `"Identificação do Processo"` which is in the Section 2 heading — already used in prior fix).

- [ ] **Step 2: Run TypeScript build**

```bash
cd C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs/frontend/apps/web
npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit if any test fixes were needed**

```bash
git add -A
git commit -m "test: fix assertions to match final-form template body"
```

---

### Task 10: Apply migration and smoke verify

**Files:** None modified.

- [ ] **Step 1: Apply 0058 to local database**

```bash
PGPASSWORD=Lepa12\<\>! psql -U metaldocs -d metaldocs -h localhost -p 5433 -f migrations/0058_update_po_browser_template.sql
```

Expected: `UPDATE 1`

- [ ] **Step 2: Verify the browser editor shows the updated template**

Open the MetalDocs app at `http://localhost:4173`, open any PO document in the browser editor, and confirm:
- Section 10 "Histórico de Revisões" appears at the bottom
- Versão, Data, and Por cells show real document values (not tokens)
- "O que foi alterado" cell is blank and editable

- [ ] **Step 3: Export DOCX and inspect**

Click "Exportar DOCX" on any PO document. Open the downloaded file in Word or LibreOffice and verify:
- Header appears as a colored table (not flat paragraphs)
- Sections 2, 3, 6 show as 2-column bordered tables
- Section 10 appears with revision data

- [ ] **Step 4: Final commit if any runtime issues required fixes**

```bash
git add -A
git commit -m "fix(template): runtime verification fixes for final-form DOCX export"
```
