import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { PropsWithChildren } from 'react';
import { forwardRef, useImperativeHandle } from 'react';
import { DocumentEditorPage } from '../DocumentEditorPage';

const queueSpy = vi.fn();
const flushSpy = vi.fn();
const saveBuffer = new Uint8Array([1, 2, 3, 4]).buffer;

vi.mock('../api/documentsV2', () => ({
  getDocument: vi.fn(),
  signedRevisionURL: vi.fn(),
  finalizeDocument: vi.fn(),
  acquireSession: vi.fn(),
  heartbeatSession: vi.fn(),
  releaseSession: vi.fn(),
}));

vi.mock('../hooks/useDocumentAutosave', () => ({
  useDocumentAutosave: () => ({
    status: 'idle',
    queue: queueSpy,
    flush: flushSpy,
  }),
}));

vi.mock('../ExportMenu', () => ({
  ExportMenu: ({ children }: PropsWithChildren) => <div data-testid="export-menu">{children}</div>,
}));

vi.mock('../CheckpointsPanel', () => ({
  CheckpointsPanel: () => <div data-testid="checkpoints-panel" />,
}));

vi.mock('@metaldocs/editor-ui', () => ({
  MetalDocsEditor: forwardRef<any, any>(function MockMetalDocsEditor(props, ref) {
    useImperativeHandle(ref, () => ({
      async getDocumentBuffer() {
        return saveBuffer;
      },
      focus() {},
    }), []);
    return (
      <div data-testid="metaldocs-editor">
        <button type="button" onClick={() => void props.onAutoSave?.(new Uint8Array([9]).buffer)}>
          trigger-autosave
        </button>
      </div>
    );
  }),
}));

describe('DocumentEditorPage', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    const api = await import('../api/documentsV2');
    vi.mocked(api.getDocument).mockResolvedValue({
      ID: 'doc-1',
      Name: 'Quarterly Report',
      CurrentRevisionID: 'rev-1',
      CreatedBy: 'user-1',
      FormDataJSON: { foo: 'bar' },
      Status: 'draft',
    });
    vi.mocked(api.signedRevisionURL).mockReturnValue('/signed/url');
    vi.mocked(api.acquireSession).mockResolvedValue({
      mode: 'writer',
      session_id: 'sess-1',
      expires_at: '2099-01-01T00:00:00Z',
      last_ack_revision_id: 'rev-1',
    });
    vi.mocked(api.heartbeatSession).mockResolvedValue({});
    vi.mocked(api.releaseSession).mockResolvedValue({});

    const fetchMock = vi.fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ url: 'https://cdn.example.com/doc.docx' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        arrayBuffer: async () => new Uint8Array([7, 8, 9]).buffer,
      });

    vi.stubGlobal('fetch', fetchMock);
  });

  it('renders editor root and mounts editor after session acquisition', async () => {
    render(<DocumentEditorPage documentID="doc-1" onDone={vi.fn()} />);

    expect(document.querySelector('[data-editor-root]')).toBeTruthy();
    expect(screen.queryByTestId('metaldocs-editor')).toBeNull();

    await waitFor(() => expect(screen.getByTestId('metaldocs-editor')).toBeTruthy());
  });

  it('queues autosave from editor callback', async () => {
    render(<DocumentEditorPage documentID="doc-1" onDone={vi.fn()} />);

    await waitFor(() => expect(screen.getByTestId('metaldocs-editor')).toBeTruthy());
    fireEvent.click(screen.getByRole('button', { name: 'trigger-autosave' }));

    await waitFor(() =>
      expect(queueSpy).toHaveBeenCalledWith(saveBuffer, { foo: 'bar' }),
    );
  });
});
