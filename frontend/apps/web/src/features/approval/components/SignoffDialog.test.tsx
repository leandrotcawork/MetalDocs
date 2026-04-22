import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ApprovalError } from '../api/mutationClient';
import { SignoffDialog } from './SignoffDialog';

import * as approvalApi from '../api/approvalApi';

vi.mock('../api/approvalApi');

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

function renderDialog() {
  const onClose = vi.fn();
  const onSuccess = vi.fn();
  const renderResult = render(
    <SignoffDialog
      documentId="doc-1"
      contentHash="hash-1"
      instanceId="inst-1"
      onClose={onClose}
      onSuccess={onSuccess}
    />,
  );
  return { onClose, onSuccess, ...renderResult };
}

describe('SignoffDialog', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.useRealTimers();
  });

  it('happy approve path — submitting=true during call, success state shown', async () => {
    const deferred = createDeferred<{ signoff_id: string; was_replay: boolean }>();
    vi.mocked(approvalApi.signoff).mockReturnValue(deferred.promise);

    renderDialog();

    fireEvent.change(screen.getByLabelText('Senha'), { target: { value: 'secret' } });
    fireEvent.click(screen.getByRole('button', { name: 'Confirmar assinatura' }));

    expect(screen.getByRole('button', { name: 'Enviando...' })).toBeTruthy();
    expect(vi.mocked(approvalApi.signoff)).toHaveBeenCalledWith('doc-1', {
      decision: 'approve',
      reason: undefined,
      password: 'secret',
      content_hash: 'hash-1',
    });

    deferred.resolve({ signoff_id: 'sig-1', was_replay: false });
    await waitFor(() => {
      expect(screen.getByText('Assinatura registrada com sucesso.')).toBeTruthy();
    });
  });

  it('happy reject path — reason required, sent in payload', async () => {
    const deferred = createDeferred<{ signoff_id: string; was_replay: boolean }>();
    vi.mocked(approvalApi.signoff).mockReturnValue(deferred.promise);

    renderDialog();

    fireEvent.click(screen.getByLabelText('Rejeitado'));
    fireEvent.change(screen.getByLabelText('Motivo'), { target: { value: 'Não atende aos requisitos.' } });
    fireEvent.change(screen.getByLabelText('Senha'), { target: { value: 'secret' } });
    fireEvent.click(screen.getByRole('button', { name: 'Confirmar assinatura' }));

    await waitFor(() => {
      expect(vi.mocked(approvalApi.signoff)).toHaveBeenCalledWith('doc-1', {
        decision: 'reject',
        reason: 'Não atende aos requisitos.',
        password: 'secret',
        content_hash: 'hash-1',
      });
    });
  });

  it('validation — reject without reason shows inline error, submit blocked', async () => {
    vi.mocked(approvalApi.signoff).mockResolvedValue({ signoff_id: 'sig-1', was_replay: false });
    renderDialog();

    fireEvent.click(screen.getByLabelText('Rejeitado'));
    fireEvent.change(screen.getByLabelText('Senha'), { target: { value: 'secret' } });
    fireEvent.click(screen.getByRole('button', { name: 'Confirmar assinatura' }));

    expect(screen.getByText('Informe o motivo da rejeição.')).toBeTruthy();
    expect(vi.mocked(approvalApi.signoff)).not.toHaveBeenCalled();
  });

  it('error_bad_password — code=authn.signature_invalid shows error message, password cleared, form values preserved', async () => {
    vi.mocked(approvalApi.signoff).mockRejectedValue(
      new ApprovalError('authn.signature_invalid', 403, 'invalid signature'),
    );
    renderDialog();

    fireEvent.click(screen.getByLabelText('Rejeitado'));
    fireEvent.change(screen.getByLabelText('Motivo'), { target: { value: 'Falta evidência técnica.' } });
    fireEvent.change(screen.getByLabelText('Senha'), { target: { value: 'bad-password' } });
    fireEvent.click(screen.getByRole('button', { name: 'Confirmar assinatura' }));

    await waitFor(() => {
      expect(screen.getByText('Senha incorreta. Verifique e tente novamente.')).toBeTruthy();
    });

    expect((screen.getByLabelText('Senha') as HTMLInputElement).value).toBe('');
    expect((screen.getByLabelText('Motivo') as HTMLTextAreaElement).value).toBe(
      'Falta evidência técnica.',
    );
    expect((screen.getByLabelText('Rejeitado') as HTMLInputElement).checked).toBe(true);
  });

  it('error_rate_limited — code=authn.rate_limited shows rate limit message', async () => {
    vi.mocked(approvalApi.signoff).mockRejectedValue(
      new ApprovalError('authn.rate_limited', 429, 'too many attempts'),
    );
    renderDialog();

    fireEvent.change(screen.getByLabelText('Senha'), { target: { value: 'secret' } });
    fireEvent.click(screen.getByRole('button', { name: 'Confirmar assinatura' }));

    await waitFor(() => {
      expect(
        screen.getByText('Muitas tentativas. Aguarde 30 segundos antes de tentar novamente.'),
      ).toBeTruthy();
    });
  });

  it('412 conflict — shows stale banner (simulate via ApprovalError code=conflict.stale status=412)', async () => {
    vi.mocked(approvalApi.signoff).mockRejectedValue(new ApprovalError('conflict.stale', 412, 'stale'));
    renderDialog();

    fireEvent.change(screen.getByLabelText('Senha'), { target: { value: 'secret' } });
    fireEvent.click(screen.getByRole('button', { name: 'Confirmar assinatura' }));

    await waitFor(() => {
      expect(
        screen.getByText('Documento foi alterado. Atualize a página antes de tentar novamente.'),
      ).toBeTruthy();
    });
  });

  it('password cleared on both success and error', async () => {
    vi.useFakeTimers();

    const { promise, resolve } = createDeferred<{ signoff_id: string; was_replay: boolean }>();
    vi.mocked(approvalApi.signoff).mockReturnValueOnce(promise);
    const firstRender = renderDialog();

    const passwordInput = screen.getByLabelText('Senha') as HTMLInputElement;

    fireEvent.change(passwordInput, { target: { value: 'secret' } });
    fireEvent.click(screen.getByRole('button', { name: 'Confirmar assinatura' }));

    // runAllTimersAsync exits early when no timers are queued yet (the 1500ms timer is
    // created by handleSubmit's continuation, which runs in a later microtask). Use
    // queueMicrotask (not faked) inside act() to flush the promise chain first.
    await act(async () => {
      resolve({ signoff_id: 'sig-1', was_replay: false });
      await new Promise<void>((r) => queueMicrotask(r));
      await new Promise<void>((r) => queueMicrotask(r));
    });
    expect(screen.getByText('Assinatura registrada com sucesso.')).toBeTruthy();
    // On success the form is replaced by the success message — the password field is
    // removed from the DOM. Verify it is gone (state was cleared, form hidden).
    expect(screen.queryByLabelText('Senha')).toBeNull();

    await vi.advanceTimersByTimeAsync(1500);
    firstRender.unmount();

    vi.mocked(approvalApi.signoff).mockRejectedValueOnce(
      new ApprovalError('authn.signature_invalid', 403, 'invalid signature'),
    );
    renderDialog();

    const retryPasswordInput = screen.getByLabelText('Senha') as HTMLInputElement;
    fireEvent.change(retryPasswordInput, { target: { value: 'wrong' } });
    fireEvent.click(screen.getByRole('button', { name: 'Confirmar assinatura' }));

    await act(async () => {
      await new Promise<void>((r) => queueMicrotask(r));
      await new Promise<void>((r) => queueMicrotask(r));
    });
    expect(screen.getByText('Senha incorreta. Verifique e tente novamente.')).toBeTruthy();
    expect(retryPasswordInput.value).toBe('');
  });
});
