// All routes under /api/v2/documents*. All requests rely on IAM cookies +
// tenant/role headers stamped by the middleware chain; we do not set X-* from
// the client.

export type DocumentRow = {
  id: string;
  name: string;
  status: 'draft' | 'finalized' | 'archived';
  template_version_id: string;
  updated_at: string;
  current_revision_id?: string;
};

export type CreateDocumentResult = { DocumentID: string; InitialRevisionID: string; SessionID: string };
export type AcquireWriter = { mode: 'writer'; session_id: string; expires_at: string; last_ack_revision_id: string };
export type AcquireReadonly = { mode: 'readonly'; held_by: string; held_until: string };
export type AcquireResult = AcquireWriter | AcquireReadonly;
export type PresignResult = { upload_url: string; pending_upload_id: string; expires_at: string };
export type CommitResult = { revision_id: string; revision_num: number; idempotent_replay?: boolean };
export type Checkpoint = { ID: string; DocumentID: string; RevisionID: string; VersionNum: number; Label: string; CreatedAt: string; CreatedBy: string };

async function json<T>(res: Response): Promise<T> {
  if (!res.ok) throw Object.assign(new Error(`http_${res.status}`), { status: res.status, body: await res.text() });
  return res.json() as Promise<T>;
}

export async function listDocuments(): Promise<DocumentRow[]> {
  return json(await fetch('/api/v2/documents'));
}
export async function getDocument(id: string): Promise<any> {
  return json(await fetch(`/api/v2/documents/${id}`));
}
export async function renameDocument(id: string, name: string): Promise<any> {
  return json(await fetch(`/api/v2/documents/${id}`, {
    method: 'PATCH',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ name }),
  }));
}
export async function createDocument(req: { template_version_id: string; name: string; form_data: unknown }): Promise<CreateDocumentResult> {
  return json(await fetch('/api/v2/documents', {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify(req),
  }));
}
export async function finalizeDocument(id: string) {
  return json(await fetch(`/api/v2/documents/${id}/finalize`, { method: 'POST' }));
}
export async function archiveDocument(id: string) {
  return json(await fetch(`/api/v2/documents/${id}/archive`, { method: 'POST' }));
}

export async function acquireSession(id: string): Promise<AcquireResult> {
  return json(await fetch(`/api/v2/documents/${id}/session/acquire`, { method: 'POST' }));
}
export async function heartbeatSession(id: string, sessionID: string) {
  return json(await fetch(`/api/v2/documents/${id}/session/heartbeat`, {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ session_id: sessionID }),
  }));
}
export async function releaseSession(id: string, sessionID: string) {
  return json(await fetch(`/api/v2/documents/${id}/session/release`, {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ session_id: sessionID }),
  }));
}
export async function forceReleaseSession(id: string, sessionID: string) {
  return json(await fetch(`/api/v2/documents/${id}/session/force-release`, {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ session_id: sessionID }),
  }));
}

export async function presignAutosave(id: string, req: { session_id: string; base_revision_id: string; content_hash: string }): Promise<PresignResult> {
  return json(await fetch(`/api/v2/documents/${id}/autosave/presign`, {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify(req),
  }));
}
// Server is authoritative for content_hash -- it re-computes SHA256 from S3 on
// commit. Client does NOT forward a client-computed hash.
export async function commitAutosave(id: string, req: { session_id: string; pending_upload_id: string; form_data_snapshot?: unknown }): Promise<CommitResult> {
  return json(await fetch(`/api/v2/documents/${id}/autosave/commit`, {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify(req),
  }));
}

export async function listCheckpoints(id: string): Promise<Checkpoint[]> {
  return json(await fetch(`/api/v2/documents/${id}/checkpoints`));
}
export async function createCheckpoint(id: string, label: string): Promise<Checkpoint> {
  return json(await fetch(`/api/v2/documents/${id}/checkpoints`, {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ label }),
  }));
}

export type RestoreCheckpointResult = {
  new_revision_id: string;
  new_revision_num: number;
  source_checkpoint_version_num: number;
  idempotent: boolean;
};
export async function restoreCheckpoint(id: string, versionNum: number): Promise<RestoreCheckpointResult> {
  return json(await fetch(`/api/v2/documents/${id}/checkpoints/${versionNum}/restore`, { method: 'POST' }));
}

export function signedRevisionURL(documentID: string, revisionID: string): string {
  return `/api/v2/documents/${documentID}/revisions/${revisionID}/url`;
}
