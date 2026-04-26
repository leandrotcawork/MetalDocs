import React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { TemplateAuthorPage } from '../v2/TemplateAuthorPage';
import type { TemplateSchemas } from '../v2/api/templatesV2';

let detectedVariables: string[] = [];
const saveSchemas = vi.fn();
const queueDocx = vi.fn();
const flush = vi.fn();

const baseSchemas: TemplateSchemas = {
  placeholders: [],
  composition: { headerSubBlocks: [], footerSubBlocks: [], subBlockParams: {} },
};

vi.mock('@eigenpal/docx-js-editor/styles.css', () => ({}));
vi.mock('@eigenpal/docx-js-editor/core', () => ({ createEmptyDocument: () => ({ type: 'empty-doc' }) }));
vi.mock('@eigenpal/docx-js-editor/react', () => ({
  DocxEditor: React.forwardRef(({ onChange }: { onChange?: () => void }, ref) => {
    React.useImperativeHandle(ref, () => ({
      save: () => Promise.resolve(new ArrayBuffer(1)),
      getAgent: () => ({ getVariables: () => detectedVariables }),
      getEditorRef: () => null,
    }));
    return <button data-testid="mock-editor-change" onClick={() => onChange?.()}>change</button>;
  }),
}));

vi.mock('../v2/hooks/useTemplateDraft', () => ({
  useTemplateDraft: () => ({ loading: false, error: null,
    template: { template_id: 'tpl-1', name: 'Test Template' },
    version: { template_id: 'tpl-1', version_num: 1, status: 'draft', docx_storage_key: null },
    docxBytes: null }),
}));
vi.mock('../v2/hooks/useTemplateAutosave', () => ({
  useTemplateAutosave: () => ({ queueDocx, flush, status: 'idle', hasPending: () => false }),
}));
vi.mock('../v2/hooks/useTemplateSchemas', () => ({
  useTemplateSchemas: () => ({ schemas: baseSchemas, loading: false, error: null, save: saveSchemas, saving: false }),
}));
vi.mock('../v2/api/catalog', () => ({
  fetchPlaceholderCatalog: () => Promise.resolve([
    { key: 'doc_code', label: 'Código do documento', description: '' },
    { key: 'doc_title', label: 'Título do documento', description: '' },
    { key: 'revision_number', label: 'Número da revisão', description: '' },
    { key: 'author', label: 'Autor', description: '' },
    { key: 'effective_date', label: 'Data efetiva', description: '' },
    { key: 'approvers', label: 'Aprovadores', description: '' },
    { key: 'controlled_by_area', label: 'Área controladora', description: '' },
  ]),
}));
vi.mock('sonner', () => ({ toast: { error: vi.fn() } }));

function renderPage() {
  return render(<TemplateAuthorPage templateId="tpl-1" versionNum={1} />);
}

async function triggerEditorChange() {
  fireEvent.click(screen.getByTestId('mock-editor-change'));
  await act(async () => { await Promise.resolve(); vi.advanceTimersByTime(400); });
}

describe('TemplateAuthorPage placeholder catalog', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    detectedVariables = [];
    saveSchemas.mockReset();
    queueDocx.mockReset();
    flush.mockReset();
    vi.stubGlobal('crypto', { ...crypto, randomUUID: vi.fn(() => 'generated-id') });
  });

  it('renders the 7 catalog entries', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId('catalog-doc_code')).toBeInTheDocument());
    expect(screen.getByTestId('catalog-doc_title')).toBeInTheDocument();
    expect(screen.getByTestId('catalog-revision_number')).toBeInTheDocument();
    expect(screen.getByTestId('catalog-author')).toBeInTheDocument();
    expect(screen.getByTestId('catalog-effective_date')).toBeInTheDocument();
    expect(screen.getByTestId('catalog-approvers')).toBeInTheDocument();
    expect(screen.getByTestId('catalog-controlled_by_area')).toBeInTheDocument();
  });

  it('does not render "+ Add manually" button', () => {
    renderPage();
    expect(screen.queryByRole('button', { name: '+ Add manually' })).toBeNull();
  });

  it('marks detected catalog tokens as detected in the panel', async () => {
    renderPage();
    detectedVariables = ['doc_code', 'author'];
    await triggerEditorChange();
    await waitFor(() => {
      expect(screen.getByTestId('catalog-doc_code').getAttribute('data-detected')).toBe('true');
      expect(screen.getByTestId('catalog-author').getAttribute('data-detected')).toBe('true');
      expect(screen.getByTestId('catalog-doc_title').getAttribute('data-detected')).toBe('false');
    });
  });

  it('saves catalog tokens to schema with computed type and resolverKey', async () => {
    renderPage();
    detectedVariables = ['doc_code', 'author'];
    await triggerEditorChange();
    await act(async () => { vi.advanceTimersByTime(400); await Promise.resolve(); });
    await waitFor(() => {
      expect(saveSchemas).toHaveBeenCalledWith({
        placeholders: [
          { id: 'generated-id', name: 'doc_code', label: 'Código do documento', type: 'computed', resolverKey: 'doc_code' },
          { id: 'generated-id', name: 'author', label: 'Autor', type: 'computed', resolverKey: 'author' },
        ],
        composition: baseSchemas.composition,
      });
    });
  });

  it('ignores non-catalog tokens silently', async () => {
    renderPage();
    detectedVariables = ['customer_name', 'doc_code'];
    await triggerEditorChange();
    await act(async () => { vi.advanceTimersByTime(400); await Promise.resolve(); });
    await waitFor(() => {
      expect(saveSchemas).toHaveBeenCalledWith({
        placeholders: [
          { id: 'generated-id', name: 'doc_code', label: 'Código do documento', type: 'computed', resolverKey: 'doc_code' },
        ],
        composition: baseSchemas.composition,
      });
    });
  });
});
