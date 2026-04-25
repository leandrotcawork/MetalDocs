import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import * as approvalApi from '../api/approvalApi';
import { SupersedePublishDialog } from './SupersedePublishDialog';

vi.mock('../api/approvalApi');

describe('SupersedePublishDialog', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-04-22T09:00:00.000Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('publish now happy path', async () => {
    const onClose = vi.fn();
    const onSuccess = vi.fn();
    vi.mocked(approvalApi.publish).mockResolvedValue({ document_id: 'doc-1' });

    render(
      <SupersedePublishDialog
        documentId="doc-1"
        contentHash="hash-1"
        onClose={onClose}
        onSuccess={onSuccess}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Confirmar publicação' }));

    // With fake timers, waitFor's internal setInterval never fires.
    // runAllTimersAsync advances fake timers AND flushes promise microtasks.
    await vi.runAllTimersAsync();
    expect(vi.mocked(approvalApi.publish)).toHaveBeenCalledWith('doc-1', {
      content_hash: 'hash-1',
    });
  });

  it('schedule happy path - datetime converted to UTC', async () => {
    const onClose = vi.fn();
    const onSuccess = vi.fn();
    vi.mocked(approvalApi.schedulePublish).mockResolvedValue({
      document_id: 'doc-1',
      scheduled_at: '2026-04-22T13:10:00.000Z',
    });

    render(
      <SupersedePublishDialog
        documentId="doc-1"
        contentHash="hash-1"
        onClose={onClose}
        onSuccess={onSuccess}
      />,
    );

    fireEvent.click(screen.getByLabelText('Agendar publicação'));
    fireEvent.change(screen.getByLabelText('Data e hora da publicação'), {
      target: { value: '2026-04-22T10:10' },
    });

    fireEvent.click(screen.getByRole('button', { name: 'Confirmar publicação' }));

    await vi.runAllTimersAsync();
    expect(vi.mocked(approvalApi.schedulePublish)).toHaveBeenCalledWith('doc-1', {
      content_hash: 'hash-1',
      effective_from: '2026-04-22T13:10:00.000Z',
    });
  });

  it('past date shows validation error and blocks submit', async () => {
    vi.mocked(approvalApi.schedulePublish).mockResolvedValue({
      document_id: 'doc-1',
      scheduled_at: '2026-04-22T13:10:00.000Z',
    });

    render(
      <SupersedePublishDialog
        documentId="doc-1"
        contentHash="hash-1"
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByLabelText('Agendar publicação'));
    fireEvent.change(screen.getByLabelText('Data e hora da publicação'), {
      target: { value: '2026-04-20T09:00' },
    });

    fireEvent.click(screen.getByRole('button', { name: 'Confirmar publicação' }));

    expect(screen.getByText('A data deve ser pelo menos 5 minutos no futuro.')).toBeTruthy();
    expect(vi.mocked(approvalApi.schedulePublish)).not.toHaveBeenCalled();
  });

  it('supersede path when checkbox checked and publishedDocumentId exists', async () => {
    vi.mocked(approvalApi.supersede).mockResolvedValue({ document_id: 'doc-2' });

    render(
      <SupersedePublishDialog
        documentId="doc-1"
        contentHash="hash-2"
        publishedDocumentId="doc-published"
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByLabelText('Substituir versão publicada atual'));
    fireEvent.click(screen.getByRole('button', { name: 'Confirmar publicação' }));

    await vi.runAllTimersAsync();
    expect(vi.mocked(approvalApi.supersede)).toHaveBeenCalledWith('doc-1', {
      content_hash: 'hash-2',
      supersedes_document_id: 'doc-published',
    });
    expect(vi.mocked(approvalApi.publish)).not.toHaveBeenCalled();
  });

  it('capability gate render', async () => {
    vi.mocked(approvalApi.publish).mockResolvedValue({ document_id: 'doc-1' });

    render(
      <SupersedePublishDialog
        documentId="doc-1"
        contentHash="hash-1"
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
    );

    expect(screen.getByRole('dialog', { name: 'Publicação' })).toBeTruthy();
    expect(screen.getByText('Publicar agora')).toBeTruthy();
  });
});

