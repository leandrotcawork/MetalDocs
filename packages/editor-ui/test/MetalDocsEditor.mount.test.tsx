import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';
import type { ReactNode } from 'react';

afterEach(cleanup);
import { MetalDocsEditor } from '../src/MetalDocsEditor';

vi.mock('@eigenpal/docx-js-editor', () => ({
  templatePlugin: { id: 'template', name: 'template' },
  PluginHost: ({ children }: { children: ReactNode }) => <>{children}</>,
  DocxEditor: ({
    documentBuffer,
    mode,
    renderTitleBarRight,
  }: {
    documentBuffer?: ArrayBuffer;
    mode?: 'editing' | 'viewing';
    renderTitleBarRight?: () => ReactNode;
  }) => (
    <div data-testid="docx-editor-mock" data-has-buffer={documentBuffer ? 'yes' : 'no'}>
      {mode === 'editing' ? <div role="toolbar" data-testid="toolbar" /> : null}
      {renderTitleBarRight ? renderTitleBarRight() : null}
    </div>
  ),
}));

describe('MetalDocsEditor', () => {
  it('shows toolbar in document-edit mode', () => {
    const buf = new ArrayBuffer(8);
    render(<MetalDocsEditor mode="document-edit" documentBuffer={buf} author="u1" />);
    const el = screen.getByTestId('docx-editor-mock');
    expect(el.getAttribute('data-has-buffer')).toBe('yes');
    expect(screen.getByRole('toolbar')).toBeInTheDocument();
  });

  it('hides toolbar in readonly mode', () => {
    render(<MetalDocsEditor mode="readonly" author="u1" />);
    const el = screen.getByTestId('docx-editor-mock');
    expect(el.getAttribute('data-has-buffer')).toBe('no');
    expect(screen.queryByRole('toolbar')).toBeNull();
  });

  it('renders renderTitleBarRight slot', () => {
    render(
      <MetalDocsEditor
        mode="document-edit"
        author="u1"
        renderTitleBarRight={() => <span data-testid="sentinel" />}
      />
    );
    expect(screen.getByTestId('sentinel')).toBeInTheDocument();
  });
});
