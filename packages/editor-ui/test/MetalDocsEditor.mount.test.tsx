import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';

afterEach(cleanup);
import { MetalDocsEditor } from '../src/MetalDocsEditor';

vi.mock('@eigenpal/docx-js-editor', () => ({
  DocxEditor: ({ documentBuffer }: { documentBuffer?: ArrayBuffer }) => (
    <div data-testid="docx-editor-mock" data-has-buffer={documentBuffer ? 'yes' : 'no'} />
  ),
}));

describe('MetalDocsEditor', () => {
  it('mounts with documentBuffer and template-draft mode', () => {
    const buf = new ArrayBuffer(8);
    render(<MetalDocsEditor mode="template-draft" documentBuffer={buf} userId="u1" />);
    const el = screen.getByTestId('docx-editor-mock');
    expect(el.getAttribute('data-has-buffer')).toBe('yes');
  });

  it('renders readonly mode without buffer', () => {
    render(<MetalDocsEditor mode="readonly" userId="u1" />);
    const el = screen.getByTestId('docx-editor-mock');
    expect(el.getAttribute('data-has-buffer')).toBe('no');
  });
});
