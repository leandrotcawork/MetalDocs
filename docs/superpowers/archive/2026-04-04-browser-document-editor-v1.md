# Browser Document Editor V1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the governed-canvas pilot with a CKEditor-based browser document editor that opens as a single document surface, stores draft content as HTML, resolves versioned templates by type or document override, and keeps DOCX/PDF as derived artifacts.

**Architecture:** Keep the existing documents module, template-resolution tables, version snapshotting, and Gotenberg pipeline. Add an additive `browser_editor` path instead of mutating the legacy schema-native flow in place: browser templates are stored as HTML, draft versions save HTML in place on the current draft version, and the frontend cuts over to a full-width CKEditor 5 editor. DOCX/PDF stays server-owned by extending `apps/docgen` with a browser-HTML render route instead of using CKEditor Cloud conversion.

**Tech Stack:** Go, Postgres, OpenAPI, React, CKEditor 5, Vite, Node/docx, Gotenberg, Playwright

---

## File Structure & Ownership

**Architecture / Docs**
- Create: `docs/adr/0022-browser-document-editor-v1.md`
- Modify: `api/openapi/v1/openapi.yaml`

**Backend domain + persistence**
- Modify: `internal/modules/documents/domain/model.go`
- Modify: `internal/modules/documents/domain/template.go`
- Modify: `internal/modules/documents/domain/port.go`
- Modify: `internal/modules/documents/infrastructure/postgres/repository.go`
- Modify: `internal/modules/documents/infrastructure/memory/repository.go`
- Create: `migrations/0056_browser_document_editor_templates.sql`

**Backend application + HTTP**
- Modify: `internal/modules/documents/application/service.go`
- Modify: `internal/modules/documents/application/service_templates.go`
- Modify: `internal/modules/documents/application/service_editor_bundle.go`
- Create: `internal/modules/documents/application/service_browser_editor.go`
- Modify: `internal/modules/documents/application/service_content_docx.go`
- Modify: `internal/modules/documents/application/service_content_native.go`
- Modify: `internal/modules/documents/delivery/http/handler.go`
- Modify: `internal/modules/documents/delivery/http/handler_content.go`

**Docgen / render**
- Modify: `internal/platform/render/docgen/types.go`
- Modify: `internal/platform/render/docgen/client.go`
- Modify: `apps/docgen/package.json`
- Modify: `apps/docgen/src/index.ts`
- Create: `apps/docgen/src/generateBrowser.ts`

**Frontend**
- Modify: `frontend/apps/web/package.json`
- Modify: `frontend/apps/web/src/vite-env.d.ts`
- Modify: `frontend/apps/web/src/lib.types.ts`
- Modify: `frontend/apps/web/src/api/documents.ts`
- Modify: `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx`
- Create: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`
- Create: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.module.css`
- Create: `frontend/apps/web/src/features/documents/browser-editor/ckeditorConfig.ts`

**Tests**
- Modify: `internal/modules/documents/application/service_templates_test.go`
- Create: `internal/modules/documents/application/service_browser_editor_test.go`
- Create: `internal/modules/documents/delivery/http/handler_browser_editor_test.go`
- Create: `frontend/apps/web/tests/e2e/browser-document-editor.spec.ts`

---

### Task 1: Record The ADR And Add Browser Editor Contracts

**Files:**
- Create: `docs/adr/0022-browser-document-editor-v1.md`
- Modify: `api/openapi/v1/openapi.yaml`
- Modify: `frontend/apps/web/package.json`
- Modify: `frontend/apps/web/src/vite-env.d.ts`
- Modify: `frontend/apps/web/src/lib.types.ts`
- Modify: `frontend/apps/web/src/api/documents.ts`

- [ ] **Step 1: Record the ADR before adding the dependency**

```md
# ADR-0022 Browser Document Editor V1

## Context
The governed-canvas pilot kept the legacy split-pane content builder and a custom template DSL. The approved v1 design instead uses a browser-native document editor with versioned template assignment, HTML body persistence, and derived DOCX/PDF artifacts.

## Decision
Adopt self-hosted CKEditor 5 for browser document editing. Store canonical template bodies and draft bodies as HTML, keep template assignment/version snapshotting in the backend, and keep DOCX/PDF generation server-owned.

## Consequences
Template HTML and revision HTML become first-class persisted content. The pilot DSL path becomes legacy, and browser-editor routes are added alongside the current schema-native endpoints until cutover is verified.
```

- [ ] **Step 2: Add the additive browser-editor OpenAPI surface**

```yaml
  /documents/{documentId}/browser-editor-bundle:
    get:
      operationId: getDocumentBrowserEditorBundle
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/DocumentBrowserEditorBundle'

  /documents/{documentId}/content/browser:
    post:
      operationId: saveDocumentBrowserContent
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/DocumentBrowserContentSaveRequest'
      responses:
        '201':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/DocumentBrowserContentSaveResponse'
```

```yaml
    DocumentBrowserTemplateSnapshot:
      type: object
      required: [templateKey, version, profileCode, schemaVersion, editor, contentFormat, body]
      properties:
        templateKey: { type: string }
        version: { type: integer }
        profileCode: { type: string }
        schemaVersion: { type: integer }
        editor: { type: string, enum: [ckeditor5] }
        contentFormat: { type: string, enum: [html] }
        body: { type: string }
```

- [ ] **Step 3: Add the frontend dependency and typed env support**

```json
{
  "dependencies": {
    "@ckeditor/ckeditor5-react": "^11.0.0",
    "ckeditor5": "^46.0.3"
  }
}
```

```ts
interface ImportMetaEnv {
  readonly VITE_CKEDITOR_LICENSE_KEY: string;
}
```

- [ ] **Step 4: Add the browser-editor client types and API helpers**

```ts
export interface DocumentBrowserTemplateSnapshotItem {
  templateKey: string;
  version: number;
  profileCode: string;
  schemaVersion: number;
  editor: "ckeditor5";
  contentFormat: "html";
  body: string;
}

export interface DocumentBrowserEditorBundleResponse {
  document: DocumentListItem;
  versions: VersionListItem[];
  governance: DocumentProfileGovernanceItem;
  templateSnapshot: DocumentBrowserTemplateSnapshotItem;
  body: string;
  draftToken: string;
}

export function getDocumentBrowserEditorBundle(documentId: string) {
  return request<DocumentBrowserEditorBundleResponse>(`/documents/${encodeURIComponent(documentId)}/browser-editor-bundle`);
}

export function saveDocumentBrowserContent(documentId: string, body: { body: string; draftToken: string }) {
  return request<{ documentId: string; version: number; contentSource: "browser_editor"; draftToken: string }>(
    `/documents/${encodeURIComponent(documentId)}/content/browser`,
    { method: "POST", body: JSON.stringify(body) },
  );
}
```

- [ ] **Step 5: Verify the contract bootstrap**

Run: `cd frontend/apps/web; npm.cmd install`  
Expected: install completes and adds `ckeditor5` plus `@ckeditor/ckeditor5-react`

Run: `cd frontend/apps/web; npm.cmd run build`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add docs/adr/0022-browser-document-editor-v1.md api/openapi/v1/openapi.yaml frontend/apps/web/package.json frontend/apps/web/package-lock.json frontend/apps/web/src/vite-env.d.ts frontend/apps/web/src/lib.types.ts frontend/apps/web/src/api/documents.ts
git commit -m "docs(editor): record browser editor contracts and dependency"
```

---

### Task 2: Persist Browser Templates And Seed Draft Version 1 From Template HTML

**Files:**
- Create: `migrations/0056_browser_document_editor_templates.sql`
- Modify: `internal/modules/documents/domain/template.go`
- Modify: `internal/modules/documents/domain/model.go`
- Modify: `internal/modules/documents/domain/port.go`
- Modify: `internal/modules/documents/application/service.go`
- Modify: `internal/modules/documents/application/service_templates.go`
- Modify: `internal/modules/documents/infrastructure/postgres/repository.go`
- Modify: `internal/modules/documents/infrastructure/memory/repository.go`
- Modify: `internal/modules/documents/application/service_templates_test.go`

- [ ] **Step 1: Write the failing template-resolution test**

```go
func TestResolveDocumentTemplateReturnsBrowserTemplateMetadata(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)

	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   "po-default-canvas",
		Version:       1,
		ProfileCode:   "po",
		SchemaVersion: 3,
		Name:          "PO Browser Template",
		Editor:        "ckeditor5",
		ContentFormat: "html",
		Body:          `<section><span class="restricted-editing-exception">Objetivo</span></section>`,
		CreatedAt:     time.Unix(1, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}

	got, err := service.ResolveDocumentTemplate(ctx, "doc-1", "po")
	if err != nil {
		t.Fatalf("ResolveDocumentTemplate() error = %v", err)
	}
	if got.Editor != "ckeditor5" || got.ContentFormat != "html" {
		t.Fatalf("template metadata = %#v, want ckeditor5/html", got)
	}
	if !strings.Contains(got.Body, "restricted-editing-exception") {
		t.Fatalf("body = %q, want restricted-editing markup", got.Body)
	}
}
```

- [ ] **Step 2: Run the targeted test and verify it fails**

Run: `go test ./internal/modules/documents/application -run "TestResolveDocumentTemplateReturnsBrowserTemplateMetadata" -count=1`  
Expected: FAIL because `DocumentTemplateVersion` does not yet carry `Editor`, `ContentFormat`, or `Body`

- [ ] **Step 3: Add the migration and domain fields for browser templates**

```sql
ALTER TABLE metaldocs.document_template_versions
  ADD COLUMN IF NOT EXISTS editor TEXT NOT NULL DEFAULT 'ckeditor5',
  ADD COLUMN IF NOT EXISTS content_format TEXT NOT NULL DEFAULT 'html',
  ADD COLUMN IF NOT EXISTS body_html TEXT NOT NULL DEFAULT '';

UPDATE metaldocs.document_template_versions
SET
  editor = 'ckeditor5',
  content_format = 'html',
  body_html = $$
  <section class="md-doc-shell">
    <h1>Procedimento Operacional</h1>
    <p><strong>Objetivo</strong></p>
    <p><span class="restricted-editing-exception">Preencha o objetivo.</span></p>
    <p><strong>Descricao do processo</strong></p>
    <div class="restricted-editing-exception"><p>Descreva o processo.</p></div>
  </section>
  $$
WHERE template_key = 'po-default-canvas' AND version = 1;
```

```go
type DocumentTemplateVersion struct {
	TemplateKey   string
	Version       int
	ProfileCode   string
	SchemaVersion int
	Name          string
	Editor        string
	ContentFormat string
	Body          string
	Definition    map[string]any
	CreatedAt     time.Time
}
```

- [ ] **Step 4: Seed version 1 content from the resolved template**

```go
resolvedTemplate, hasTemplate, err := s.resolveDocumentTemplateOptional(ctx, doc.ID, doc.DocumentProfile)
if err != nil {
	return domain.Document{}, err
}

initialContent := cmd.InitialContent
contentSource := domain.ContentSourceNative
if hasTemplate && strings.EqualFold(resolvedTemplate.Editor, "ckeditor5") && strings.EqualFold(resolvedTemplate.ContentFormat, "html") {
	initialContent = resolvedTemplate.Body
	contentSource = domain.ContentSourceBrowserEditor
}

v1 := domain.Version{
	DocumentID:      doc.ID,
	Number:          1,
	Content:         initialContent,
	ContentHash:     contentHash(initialContent),
	ChangeSummary:   "Initial version",
	ContentSource:   contentSource,
	TemplateKey:     resolvedTemplate.TemplateKey,
	TemplateVersion: resolvedTemplate.Version,
	TextContent:     initialContent,
	CreatedAt:       now,
}
```

- [ ] **Step 5: Verify the persistence/create path**

Run: `go test ./internal/modules/documents/application -run "TestResolveDocumentTemplateReturnsBrowserTemplateMetadata|TestCreateDocumentSeedsBrowserTemplateBody" -count=1`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add migrations/0056_browser_document_editor_templates.sql internal/modules/documents/domain/template.go internal/modules/documents/domain/model.go internal/modules/documents/domain/port.go internal/modules/documents/application/service.go internal/modules/documents/application/service_templates.go internal/modules/documents/infrastructure/postgres/repository.go internal/modules/documents/infrastructure/memory/repository.go internal/modules/documents/application/service_templates_test.go
git commit -m "feat(documents): persist browser templates and seed draft bodies"
```

---

### Task 3: Add Browser Editor Bundle And Draft Save APIs

**Files:**
- Create: `internal/modules/documents/application/service_browser_editor.go`
- Create: `internal/modules/documents/application/service_browser_editor_test.go`
- Create: `internal/modules/documents/delivery/http/handler_browser_editor_test.go`
- Modify: `internal/modules/documents/domain/model.go`
- Modify: `internal/modules/documents/application/service_editor_bundle.go`
- Modify: `internal/modules/documents/delivery/http/handler.go`
- Modify: `internal/modules/documents/delivery/http/handler_content.go`

- [ ] **Step 1: Write the failing service and handler tests**

```go
func TestGetBrowserEditorBundleReturnsDraftHTML(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)
	doc := seedBrowserDocument(t, repo, `<section><p>Original</p></section>`)

	bundle, err := service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetBrowserEditorBundleAuthorized() error = %v", err)
	}
	if bundle.Body != `<section><p>Original</p></section>` || bundle.DraftToken == "" {
		t.Fatalf("bundle = %#v", bundle)
	}
}

func TestSaveBrowserContentAuthorizedUpdatesDraftInPlace(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)
	doc := seedBrowserDocument(t, repo, `<section><p>Original</p></section>`)
	current, _ := repo.GetVersion(ctx, doc.ID, 1)

	version, err := service.SaveBrowserContentAuthorized(ctx, domain.SaveBrowserContentCommand{
		DocumentID: doc.ID,
		DraftToken: draftTokenForVersion(current),
		Body:       `<section><p>Atualizado</p></section>`,
		TraceID:    "trace-test",
	})
	if err != nil {
		t.Fatalf("SaveBrowserContentAuthorized() error = %v", err)
	}
	if version.Number != 1 || version.ContentSource != domain.ContentSourceBrowserEditor {
		t.Fatalf("version = %#v", version)
	}
}
```

```go
func TestHandleDocumentBrowserContentPost(t *testing.T) {
	handler := NewHandler(newBrowserEditorStubService())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/doc-123/content/browser", strings.NewReader(`{"body":"<p>Atualizado</p>","draftToken":"v1:test"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleDocumentContentBrowserPost(rec, req, "doc-123")

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if !strings.Contains(rec.Body.String(), `"contentSource":"browser_editor"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
```

- [ ] **Step 2: Run the targeted tests and verify they fail**

Run: `go test ./internal/modules/documents/application -run "TestGetBrowserEditorBundleReturnsDraftHTML|TestSaveBrowserContentAuthorizedUpdatesDraftInPlace" -count=1`  
Expected: FAIL because the browser-editor service does not exist

Run: `go test ./internal/modules/documents/delivery/http -run "TestHandleDocumentBrowserContentPost" -count=1`  
Expected: FAIL because the handler method and route do not exist

- [ ] **Step 3: Add the browser-editor service and response types**

```go
const ContentSourceBrowserEditor = "browser_editor"

type SaveBrowserContentCommand struct {
	DocumentID string
	DraftToken string
	Body       string
	TraceID    string
}

func (s *Service) GetBrowserEditorBundleAuthorized(ctx context.Context, documentID string) (BrowserEditorBundle, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return BrowserEditorBundle{}, err
	}
	versions, err := s.repo.ListVersions(ctx, doc.ID)
	if err != nil {
		return BrowserEditorBundle{}, err
	}
	current := versions[len(versions)-1]
	templateVersion, err := s.ResolveDocumentTemplate(ctx, doc.ID, doc.DocumentProfile)
	if err != nil {
		return BrowserEditorBundle{}, err
	}
	return BrowserEditorBundle{
		Document: doc,
		Versions: versions,
		TemplateSnapshot: domain.DocumentTemplateSnapshot{
			TemplateKey:   templateVersion.TemplateKey,
			Version:       templateVersion.Version,
			ProfileCode:   templateVersion.ProfileCode,
			SchemaVersion: templateVersion.SchemaVersion,
			Editor:        templateVersion.Editor,
			ContentFormat: templateVersion.ContentFormat,
			Body:          templateVersion.Body,
		},
		Body:       current.Content,
		DraftToken: draftTokenForVersion(current),
	}, nil
}
```

- [ ] **Step 4: Save browser HTML in place on the current draft**

```go
func (s *Service) SaveBrowserContentAuthorized(ctx context.Context, cmd domain.SaveBrowserContentCommand) (domain.Version, error) {
	if strings.TrimSpace(cmd.DocumentID) == "" || strings.TrimSpace(cmd.DraftToken) == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
	if err != nil {
		return domain.Version{}, err
	}
	current, err := s.latestVersion(ctx, doc.ID)
	if err != nil {
		return domain.Version{}, err
	}
	expectedHash := strings.TrimSpace(current.ContentHash)
	if expectedHash == "" {
		expectedHash = contentHash(current.Content)
	}
	if !matchesDraftToken(cmd.DraftToken, current) {
		return domain.Version{}, domain.ErrDraftConflict
	}
	current.Content = cmd.Body
	current.ContentHash = contentHash(cmd.Body)
	current.ContentSource = domain.ContentSourceBrowserEditor
	current.TextContent = stripBrowserEditorHTML(cmd.Body)
	if err := s.repo.UpdateDraftVersionContentCAS(ctx, current, expectedHash); err != nil {
		return domain.Version{}, err
	}
	return current, nil
}
```

```go
if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && parts[1] == "browser-editor-bundle" && r.Method == http.MethodGet {
	h.handleDocumentBrowserEditorBundle(w, r, parts[0])
	return
}
if len(parts) == 3 && strings.TrimSpace(parts[0]) != "" && parts[1] == "content" && parts[2] == "browser" && r.Method == http.MethodPost {
	h.handleDocumentContentBrowserPost(w, r, parts[0])
	return
}
```

- [ ] **Step 5: Verify the browser-editor backend**

Run: `go test ./internal/modules/documents/application -run "TestGetBrowserEditorBundleReturnsDraftHTML|TestSaveBrowserContentAuthorizedUpdatesDraftInPlace" -count=1`  
Expected: PASS

Run: `go test ./internal/modules/documents/delivery/http -run "TestHandleDocumentBrowserContentPost" -count=1`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents/domain/model.go internal/modules/documents/application/service_browser_editor.go internal/modules/documents/application/service_editor_bundle.go internal/modules/documents/delivery/http/handler.go internal/modules/documents/delivery/http/handler_content.go internal/modules/documents/application/service_browser_editor_test.go internal/modules/documents/delivery/http/handler_browser_editor_test.go
git commit -m "feat(documents): add browser editor draft bundle and save API"
```

---

### Task 4: Cut Over The Frontend To A Full-Width CKEditor Document Surface

**Files:**
- Create: `frontend/apps/web/src/features/documents/browser-editor/ckeditorConfig.ts`
- Create: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`
- Create: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.module.css`
- Modify: `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx`
- Create: `frontend/apps/web/tests/e2e/browser-document-editor.spec.ts`

- [ ] **Step 1: Write the failing Playwright smoke**

```ts
test("browser document editor opens as a single document surface", async ({ page }) => {
  const createdDocument = await createBrowserTemplateDocument(page);

  await page.goto(`/#/documents/doc/${encodeURIComponent(createdDocument.documentId)}`);
  await page.getByRole("button", { name: "Abrir documento" }).click();

  await expect(page.getByTestId("browser-document-editor")).toBeVisible({ timeout: 20_000 });
  await expect(page.locator(".content-builder-preview")).toHaveCount(0);

  const editable = page.locator(".ck-editor__editable").first();
  await editable.click();
  await editable.fill("Objetivo do teste");

  await page.getByRole("button", { name: "Salvar rascunho" }).click();
  await expect(page.getByText("Salvo agora")).toBeVisible();
});
```

- [ ] **Step 2: Run the E2E spec and verify it fails**

Run: `cd frontend/apps/web; npx playwright test tests/e2e/browser-document-editor.spec.ts`  
Expected: FAIL because the browser editor component and route cutover do not exist

- [ ] **Step 3: Add a focused CKEditor config**

```ts
import {
  Bold,
  DecoupledEditor,
  Essentials,
  Heading,
  Italic,
  List,
  Paragraph,
  RestrictedEditingMode,
  Table,
  TableToolbar,
} from "ckeditor5";

export const browserDocumentEditorConfig = {
  licenseKey: import.meta.env.VITE_CKEDITOR_LICENSE_KEY,
  plugins: [Essentials, Paragraph, Heading, Bold, Italic, List, Table, TableToolbar, RestrictedEditingMode],
  toolbar: ["heading", "|", "bold", "italic", "bulletedList", "numberedList", "insertTable", "|", "undo", "redo"],
  initialData: "",
  editorClass: DecoupledEditor,
} as const;
```

- [ ] **Step 4: Replace the active content-builder path**

```tsx
export function ContentBuilderView(props: ContentBuilderViewProps) {
  if (!props.document) {
    return (
      <section className="content-builder-empty">
        <strong>Nenhum documento selecionado.</strong>
      </section>
    );
  }

  return <BrowserDocumentEditorView document={props.document} onBack={props.onBack} />;
}
```

```tsx
export function BrowserDocumentEditorView({ document, onBack }: { document: DocumentListItem; onBack: () => void }) {
  const [bundle, setBundle] = useState<DocumentBrowserEditorBundleResponse | null>(null);

  useEffect(() => {
    let active = true;
    void getDocumentBrowserEditorBundle(document.documentId).then((next) => {
      if (active) setBundle(next);
    });
    return () => { active = false; };
  }, [document.documentId]);

  return (
    <section className={styles.root} data-testid="browser-document-editor">
      <header className={styles.topbar}>
        <button type="button" onClick={onBack}>Voltar</button>
        <span>{document.documentCode ?? document.title}</span>
      </header>
      {bundle ? <CKEditor editor={browserDocumentEditorConfig.editorClass} config={{ ...browserDocumentEditorConfig, initialData: bundle.body }} /> : null}
    </section>
  );
}
```

- [ ] **Step 5: Verify the frontend cutover**

Run: `cd frontend/apps/web; npm.cmd run build`  
Expected: PASS

Run: `cd frontend/apps/web; npx playwright test tests/e2e/browser-document-editor.spec.ts`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/browser-editor frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx frontend/apps/web/tests/e2e/browser-document-editor.spec.ts
git commit -m "feat(frontend-documents): cut over to browser document editor"
```

---

### Task 5: Keep Export Server-Owned And Add Template Assignment APIs

**Files:**
- Modify: `internal/platform/render/docgen/types.go`
- Modify: `internal/platform/render/docgen/client.go`
- Modify: `internal/modules/documents/application/service_content_docx.go`
- Modify: `internal/modules/documents/application/service_content_native.go`
- Modify: `api/openapi/v1/openapi.yaml`
- Modify: `internal/modules/documents/application/service_templates.go`
- Modify: `internal/modules/documents/delivery/http/handler.go`
- Modify: `apps/docgen/package.json`
- Modify: `apps/docgen/src/index.ts`
- Create: `apps/docgen/src/generateBrowser.ts`
- Modify: `frontend/apps/web/src/api/documents.ts`
- Modify: `frontend/apps/web/src/lib.types.ts`
- Modify: `frontend/apps/web/tests/e2e/browser-document-editor.spec.ts`

- [ ] **Step 1: Write the failing export-path test**

```go
func TestExportBrowserContentUsesBrowserDocgenRoute(t *testing.T) {
	payloadCh := make(chan []byte, 1)
	docgenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generate-browser" {
			t.Fatalf("docgen path = %q, want /generate-browser", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		payloadCh <- raw
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		_, _ = w.Write([]byte("docx"))
	}))
	defer docgenServer.Close()

	service := newBrowserEditorServiceWithDocgen(t, docgenServer.URL)
	_, err := service.RenderContentPDFAuthorized(context.Background(), "doc-browser-1", "trace-test")
	if err != nil {
		t.Fatalf("RenderContentPDFAuthorized() error = %v", err)
	}
	select {
	case raw := <-payloadCh:
		if !bytes.Contains(raw, []byte(`"html":"<section><p>Atualizado</p></section>"`)) {
			t.Fatalf("payload = %s", raw)
		}
	default:
		t.Fatal("expected browser docgen payload")
	}
}
```

- [ ] **Step 2: Run the targeted test and verify it fails**

Run: `go test ./internal/modules/documents/application -run "TestExportBrowserContentUsesBrowserDocgenRoute" -count=1`  
Expected: FAIL because the Go client only knows `/generate`

- [ ] **Step 3: Extend docgen and the Go client with a browser-HTML route**

```go
type BrowserRenderPayload struct {
	DocumentCode string `json:"documentCode"`
	Title        string `json:"title"`
	Version      string `json:"version,omitempty"`
	HTML         string `json:"html"`
}
```

```go
func (c *Client) GenerateBrowser(ctx context.Context, payload BrowserRenderPayload, traceID string) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal browser docgen payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/generate-browser", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-Id", traceID)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
```

```json
{
  "dependencies": {
    "@turbodocx/html-to-docx": "^1.14.0"
  }
}
```

```ts
import HTMLtoDOCX from "@turbodocx/html-to-docx";

export async function generateBrowserDocx(payload: { html: string }) {
  if (!payload.html || !payload.html.trim()) {
    throw new Error("DOCGEN_INVALID_PAYLOAD");
  }
  return HTMLtoDOCX(payload.html);
}
```

- [ ] **Step 4: Add assignment APIs without building template-editing UI**

```yaml
  /document-templates:
    get:
      parameters:
        - in: query
          name: profileCode
          schema: { type: string }

  /documents/{documentId}/template-assignment:
    put:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [templateKey, templateVersion]
              properties:
                templateKey: { type: string }
                templateVersion: { type: integer }
```

```go
switch strings.TrimSpace(version.ContentSource) {
case domain.ContentSourceBrowserEditor:
	docxBytes, err = s.docgenClient.GenerateBrowser(ctx, docgen.BrowserRenderPayload{
		DocumentCode: doc.DocumentCode,
		Title:        doc.Title,
		Version:      strconv.Itoa(version.Number),
		HTML:         version.Content,
	}, traceID)
default:
	docxBytes, err = s.generateDocxBytes(ctx, doc, version, content, traceID, nil)
}
```

- [ ] **Step 5: Lock in the full verification set**

Run: `go test ./internal/modules/documents/... -count=1`  
Expected: PASS

Run: `cd apps/docgen; npm.cmd run build`  
Expected: PASS

Run: `cd frontend/apps/web; npm.cmd run build`  
Expected: PASS

Run: `cd frontend/apps/web; npx playwright test tests/e2e/browser-document-editor.spec.ts`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/platform/render/docgen/types.go internal/platform/render/docgen/client.go internal/modules/documents/application/service_content_docx.go internal/modules/documents/application/service_content_native.go api/openapi/v1/openapi.yaml internal/modules/documents/application/service_templates.go internal/modules/documents/delivery/http/handler.go apps/docgen/package.json apps/docgen/package-lock.json apps/docgen/src/index.ts apps/docgen/src/generateBrowser.ts frontend/apps/web/src/api/documents.ts frontend/apps/web/src/lib.types.ts frontend/apps/web/tests/e2e/browser-document-editor.spec.ts
git commit -m "feat(documents): finish browser editor export and assignment APIs"
```

---

## Self-Review Checklist

1. **Spec coverage:**  
   - CKEditor/browser document editor: Tasks 1 and 4  
   - versioned web-native templates: Tasks 1 and 2  
   - draft body stored in editor-native HTML: Tasks 2 and 3  
   - type default plus document-lineage override: Task 5  
   - exact template snapshot per revision: Tasks 2 and 3  
   - DOCX/PDF as derived outputs: Task 5

2. **Placeholder scan:**  
   - No `TODO`/`TBD` markers remain.  
   - Every task lists exact files, commands, and commit messages.  
   - Code-bearing tasks include concrete snippets for tests and implementation.

3. **Type consistency:**  
   - Browser path uses `browser_editor` consistently for content source.  
   - Template metadata uses `Editor`, `ContentFormat`, and `Body` consistently across domain, API, and frontend.  
   - Browser save payload uses `body` and `draftToken` consistently from backend to frontend.
