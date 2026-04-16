import { describe, it, expect, vi } from 'vitest';
import { render, waitFor, cleanup, act } from '@testing-library/react';
import { afterEach } from 'vitest';
import type { DecoupledEditor } from 'ckeditor5';
import { AuthorEditor } from '../AuthorEditor';

afterEach(cleanup);

describe('<AuthorEditor />', () => {
  it('renders a toolbar container and an editable container', async () => {
    const { container } = render(<AuthorEditor initialHtml="<p>Hi</p>" onChange={() => {}} />);
    await waitFor(() => {
      expect(container.querySelector('[data-ck5-role="toolbar"]')).not.toBeNull();
      expect(container.querySelector('[data-ck5-role="editable"]')).not.toBeNull();
    });
  });

  it('fires onChange with data after a programmatic edit', async () => {
    const onChange = vi.fn();
    let editorRef: DecoupledEditor | null = null;

    render(
      <AuthorEditor
        initialHtml="<p>Hello</p>"
        onChange={onChange}
        onReady={(e) => { editorRef = e; }}
      />,
    );

    // Wait for the editor to be ready.
    await waitFor(() => expect(editorRef).not.toBeNull());

    // Programmatically mutate the model to trigger change:data.
    await act(async () => {
      editorRef!.model.change((writer) => {
        const root = editorRef!.model.document.getRoot()!;
        const firstPara = root.getChild(0)!;
        writer.insertText(' World', firstPara, 'end');
      });
    });

    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith(expect.stringContaining('World'));
    });
  });
});
