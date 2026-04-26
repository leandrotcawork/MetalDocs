import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import * as approvalApi from '../api/approvalApi';
import { etagCache } from '../api/etagCache';
import type { ApprovalInstance } from '../api/approvalTypes';
import { RegistryDetailPanel } from './RegistryDetailPanel';

vi.mock('../api/approvalApi');
vi.mock('./SignoffDialog', () => ({
  SignoffDialog: () => <div>SignoffDialogMock</div>,
}));
vi.mock('./SupersedePublishDialog', () => ({
  SupersedePublishDialog: () => <div>SupersedePublishDialogMock</div>,
}));

function makeInstance(): ApprovalInstance {
  return {
    id: 'inst-1',
    document_id: 'doc-1',
    route_id: 'route-1',
    status: 'in_progress',
    submitted_by: 'maria',
    submitted_at: '2026-04-15T10:00:00.000Z',
    stages: [],
  };
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe('RegistryDetailPanel', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    etagCache.clear();
    vi.mocked(approvalApi.getInstance).mockResolvedValue(makeInstance());
    vi.mocked(approvalApi.listRoutes).mockResolvedValue({ routes: [], total: 0 });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('loading state renders skeleton', () => {
    const pending = deferred<ApprovalInstance>();
    vi.mocked(approvalApi.getInstance).mockReturnValue(pending.promise);

    render(
      <RegistryDetailPanel
        documentId="doc-1"
        approvalState="draft"
        contentHash="abcdef1234567890"
        revisionVersion={4}
      />,
    );

    expect(screen.getByText('Carregando painel de aprovação...')).toBeTruthy();
  });

  it('draft state shows "Submeter" button', async () => {
    render(
      <RegistryDetailPanel
        documentId="doc-1"
        approvalState="draft"
        contentHash="abcdef1234567890"
        revisionVersion={4}
      />,
    );

    expect(await screen.findByRole('button', { name: 'Submeter para revisão' })).toBeTruthy();
  });

  it('under_review shows LockBadge + "Assinar" + disabled edit reason', async () => {
    render(
      <RegistryDetailPanel
        documentId="doc-1"
        approvalState="under_review"
        contentHash="abcdef1234567890"
        revisionVersion={4}
        lockedByInstanceId="inst-1"
        lockedByActor="joao"
        lockAcquiredAt="2026-04-22T10:00:00.000Z"
      />,
    );

    expect(await screen.findByRole('button', { name: 'Assinar' })).toBeTruthy();
    expect(screen.getByText(/Documento em revisão por/i)).toBeTruthy();
    // Text may be split across child elements — use partial/regex match
    expect(screen.getByText(/Documento em revisão — edição bloqueada/i)).toBeTruthy();
  });

  it('approved state shows "Publicar / Agendar" button', async () => {
    render(
      <RegistryDetailPanel
        documentId="doc-1"
        approvalState="approved"
        contentHash="abcdef1234567890"
        revisionVersion={4}
      />,
    );

    expect(await screen.findByRole('button', { name: 'Publicar / Agendar' })).toBeTruthy();
  });

  it('obsolete state shows read-only label', async () => {
    render(
      <RegistryDetailPanel
        documentId="doc-1"
        approvalState="obsolete"
        contentHash="abcdef1234567890"
        revisionVersion={4}
      />,
    );

    expect(await screen.findByText('Somente leitura')).toBeTruthy();
  });

  it('integrity panel shows truncated hash + copy button', async () => {
    render(
      <RegistryDetailPanel
        documentId="doc-1"
        approvalState="draft"
        contentHash="abcdef1234567890"
        revisionVersion={4}
      />,
    );

    await screen.findByText('Integridade');
    expect(screen.getByText('abcdef12…')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Copiar' })).toBeTruthy();
    expect(screen.getByTitle('abcdef1234567890')).toBeTruthy();
  });

  it('stale banner appears when >30s', async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-04-22T09:00:00.000Z'));

    render(
      <RegistryDetailPanel
        documentId="doc-1"
        approvalState="draft"
        contentHash="abcdef1234567890"
        revisionVersion={4}
      />,
    );

    // findByRole polls via setTimeout — blocked by fake timers.
    // Use queueMicrotask (not faked) inside act() to flush fetchInstance's promise chain.
    await act(async () => {
      await new Promise<void>((r) => queueMicrotask(r));
      await new Promise<void>((r) => queueMicrotask(r));
    });
    expect(screen.getByRole('button', { name: 'Submeter para revisão' })).toBeTruthy();

    // Advance fake clock 31s: setInterval fires 31×, setNow updates, isStale becomes true.
    await act(async () => {
      vi.advanceTimersByTime(31_000);
    });
    expect(screen.getByText('Dados podem estar desatualizados.')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Atualizar' })).toBeTruthy();
  });

  it('embedded timeline renders when instance present', async () => {
    render(
      <RegistryDetailPanel
        documentId="doc-1"
        approvalState="draft"
        contentHash="abcdef1234567890"
        revisionVersion={4}
      />,
    );

    expect(await screen.findByText('Submetido')).toBeTruthy();
    expect(document.querySelector('#approval-timeline')).toBeTruthy();
  });
});
