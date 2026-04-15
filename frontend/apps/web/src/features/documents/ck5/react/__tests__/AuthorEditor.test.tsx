import { describe, it, expect } from 'vitest';
import { render, waitFor, cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';
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

  it('fires onChange with data after edits', async () => {
    const onChange = vi.fn();
    render(<AuthorEditor initialHtml="<p>Hello</p>" onChange={onChange} />);
    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith(expect.stringContaining('Hello'));
    });
  });
});
