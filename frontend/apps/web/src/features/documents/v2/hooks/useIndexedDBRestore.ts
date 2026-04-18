import { openDB, type IDBPDatabase } from 'idb';

const DB_NAME = 'metaldocs_docs_v2';
const STORE = 'pending_autosaves';

interface PendingBlob {
  document_id: string;
  session_id: string;
  base_revision_id: string;
  content_hash: string;
  buffer: ArrayBuffer;
  created_at: number;
}

let dbPromise: Promise<IDBPDatabase> | null = null;

function getDB() {
  if (!dbPromise) {
    dbPromise = openDB(DB_NAME, 1, {
      upgrade(db) {
        if (!db.objectStoreNames.contains(STORE)) {
          db.createObjectStore(STORE, { keyPath: ['document_id', 'content_hash'] });
        }
      },
    });
  }
  return dbPromise;
}

export async function putPending(p: PendingBlob) {
  const db = await getDB();
  await db.put(STORE, p);
}
export async function getAllPending(documentID: string): Promise<PendingBlob[]> {
  const db = await getDB();
  const all = await db.getAll(STORE);
  return all.filter((x: PendingBlob) => x.document_id === documentID);
}
export async function deletePending(documentID: string, contentHash: string) {
  const db = await getDB();
  await db.delete(STORE, [documentID, contentHash]);
}
