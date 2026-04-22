// @ts-expect-error O pacote existe no workspace, mas este app não expõe typings de uuid.
import { v4 as uuidv4 } from 'uuid';
import { toast } from 'sonner';

import { etagCache } from './etagCache';

export class ApprovalError extends Error {
  constructor(
    public readonly code: string,
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = 'ApprovalError';
  }
}

export interface MutateOptions {
  idempotencyKey?: string;
  resourceId?: string;
  ifMatch?: string;
  on412?: (resourceId: string) => void;
}

export async function mutate<TReq, TRes>(
  method: 'POST' | 'PUT' | 'PATCH' | 'DELETE',
  url: string,
  body?: TReq,
  opts: MutateOptions = {},
): Promise<TRes> {
  const idempotencyKey = opts.idempotencyKey ?? uuidv4();
  const ifMatch = opts.ifMatch ?? (opts.resourceId ? etagCache.get(opts.resourceId) : undefined);

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'Idempotency-Key': idempotencyKey,
  };
  if (ifMatch) headers['If-Match'] = ifMatch;

  const res = await fetch(url, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  const newETag = res.headers.get('ETag');
  if (newETag && opts.resourceId) {
    etagCache.set(opts.resourceId, newETag);
  }

  if (res.status === 412) {
    if (opts.on412 && opts.resourceId) {
      opts.on412(opts.resourceId);
    } else {
      toast.error('Documento foi alterado. Por favor, atualize a página.');
    }
    const responseBody = (await res.json().catch(() => ({}))) as { error?: { code?: string } };
    throw new ApprovalError(responseBody.error?.code ?? 'conflict.stale', 412, 'Stale resource');
  }

  if (res.status === 401) {
    toast.error('Sessão expirada. Por favor, autentique novamente.');
    throw new ApprovalError('authn.expired', 401, 'Não autorizado');
  }

  if (res.status === 403) {
    const responseBody = (await res.json().catch(() => ({}))) as {
      error?: { code?: string; message?: string };
    };
    toast.error('Permissão negada.');
    throw new ApprovalError(
      responseBody.error?.code ?? 'authz.denied',
      403,
      responseBody.error?.message ?? 'Proibido',
    );
  }

  if (res.status === 429) {
    throw new ApprovalError('authn.rate_limited', 429, 'Muitas tentativas. Aguarde 30 segundos.');
  }

  if (!res.ok) {
    const responseBody = (await res.json().catch(() => ({}))) as {
      error?: { code?: string; message?: string };
    };
    throw new ApprovalError(
      responseBody.error?.code ?? `http_${res.status}`,
      res.status,
      responseBody.error?.message ?? 'Erro interno',
    );
  }

  return res.json() as Promise<TRes>;
}
