import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, waitFor, cleanup } from '@testing-library/react';
import { FillEditor } from '../FillEditor';

afterEach(cleanup);

describe('<FillEditor />', () => {
  it('renders an editable', async () => {
    const { container } = render(<FillEditor documentHtml="<p>Fill me</p>" onChange={() => {}} />);
    await waitFor(() => {
      expect(container.querySelector('.ck-editor__editable')).not.toBeNull();
    });
  });

  it('calls onReady after mount', async () => {
    const onReady = vi.fn();
    render(<FillEditor documentHtml="<p>Hi</p>" onReady={onReady} onChange={() => {}} />);
    await waitFor(() => expect(onReady).toHaveBeenCalled());
  });
});
