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

vi.mock('@eigenpal/docx-js-editor/core', () => ({
  createEmptyDocument: () => ({ type: 'empty-doc' }),
}));

vi.mock('@eigenpal/docx-js-editor/react', () => ({
  DocxEditor: React.forwardRef(({ onChange }: { onChange?: () => void }, ref) => {
    React.useImperativeHandle(ref, () => ({
      save: () => Promise.resolve(new ArrayBuffer(1)),
      // Eigenpal 0.0.34 exposes DocxEditorRef.getAgent().getVariables(): string[].
      getAgent: () => ({
        getVariables: () => detectedVariables,
      }),
      getEditorRef: () => null,
    }));

    return (
      <button type="button" data-testid="mock-editor-change" onClick={() => onChange?.()}>
        editor change
      </button>
    );
  }),
}));

vi.mock('../v2/hooks/useTemplateDraft', () => ({
  useTemplateDraft: () => ({
    loading: false,
    error: null,
    template: { template_id: 'tpl-1', name: 'Test Template' },
    version: { template_id: 'tpl-1', version_num: 1, status: 'draft', docx_storage_key: null },
    docxBytes: null,
  }),
}));

vi.mock('../v2/hooks/useTemplateAutosave', () => ({
  useTemplateAutosave: () => ({
    queueDocx,
    flush,
    status: 'idle',
    hasPending: () => false,
  }),
}));

vi.mock('../v2/hooks/useTemplateSchemas', () => ({
  useTemplateSchemas: () => ({
    schemas: baseSchemas,
    loading: false,
    error: null,
    save: saveSchemas,
    saving: false,
  }),
}));

vi.mock('../v2/api/templatesV2', async () => {
  const actual = await vi.importActual<typeof import('../v2/api/templatesV2')>('../v2/api/templatesV2');
  return {
    ...actual,
    submitForReview: vi.fn(),
  };
});

vi.mock('sonner', () => ({
  toast: { error: vi.fn() },
}));

function renderPage() {
  return render(<TemplateAuthorPage templateId="tpl-1" versionNum={1} />);
}

async function triggerEditorChange() {
  fireEvent.click(screen.getByTestId('mock-editor-change'));
  await act(async () => {
    await Promise.resolve();
    vi.advanceTimersByTime(400);
  });
}

describe('TemplateAuthorPage placeholder convergence', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    detectedVariables = [];
    saveSchemas.mockReset();
    queueDocx.mockReset();
    flush.mockReset();
    vi.stubGlobal('crypto', {
      ...crypto,
      randomUUID: vi.fn(() => 'generated-id'),
    });
  });

  it('auto-adds a text schema placeholder when {foo_bar} is typed in the editor', async () => {
    renderPage();

    detectedVariables = ['foo_bar'];
    await triggerEditorChange();

    const chip = await screen.findByTestId('placeholder-chip-generated-id');
    expect(chip.textContent).toContain('Foo Bar');
    expect(chip.getAttribute('data-orphan')).toBe('false');

    await act(async () => {
      vi.advanceTimersByTime(400);
      await Promise.resolve();
    });

    await waitFor(() => {
      expect(saveSchemas).toHaveBeenCalledWith({
        placeholders: [{ id: 'generated-id', name: 'foo_bar', label: 'Foo Bar', type: 'text' }],
        composition: baseSchemas.composition,
      });
    });
  });

  it('marks an existing schema placeholder orphan when its token is deleted from the editor', async () => {
    renderPage();

    detectedVariables = ['foo_bar'];
    await triggerEditorChange();
    expect((await screen.findByTestId('placeholder-chip-generated-id')).getAttribute('data-orphan')).toBe('false');

    detectedVariables = [];
    await triggerEditorChange();

    expect(screen.getByTestId('placeholder-chip-generated-id').getAttribute('data-orphan')).toBe('true');
  });

  it('clears the orphan flag when the deleted token is re-typed', async () => {
    renderPage();

    detectedVariables = ['foo_bar'];
    await triggerEditorChange();
    detectedVariables = [];
    await triggerEditorChange();
    expect(screen.getByTestId('placeholder-chip-generated-id').getAttribute('data-orphan')).toBe('true');

    detectedVariables = ['foo_bar'];
    await triggerEditorChange();

    expect(screen.getByTestId('placeholder-chip-generated-id').getAttribute('data-orphan')).toBe('false');
  });

  it('demotes manual add and lets orphan placeholders be removed from the inspector', async () => {
    renderPage();

    const addManual = screen.getByRole('button', { name: '+ Add manually' });
    expect(addManual.getAttribute('title')).toBe(
      'Prefer typing {name} directly in the document - placeholders are detected automatically.',
    );

    detectedVariables = ['foo_bar'];
    await triggerEditorChange();
    detectedVariables = [];
    await triggerEditorChange();

    fireEvent.click(screen.getByTestId('placeholder-chip-generated-id'));
    fireEvent.click(screen.getByRole('button', { name: 'Remove from schema' }));

    expect(screen.queryByTestId('placeholder-chip-generated-id')).toBeNull();
  });
});
