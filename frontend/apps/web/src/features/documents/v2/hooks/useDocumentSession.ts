import { useCallback, useEffect, useRef, useState } from 'react';
import { acquireSession, heartbeatSession, releaseSession, type AcquireResult } from '../api/documentsV2';

export type SessionState =
  | { phase: 'idle' }
  | { phase: 'acquiring' }
  | { phase: 'writer'; sessionID: string; lastAckRevisionID: string }
  | { phase: 'readonly'; heldBy: string; heldUntil: string }
  | { phase: 'lost'; reason: 'expired' | 'force_released' | 'network' };

const HEARTBEAT_MS = 30_000;

export function useDocumentSession(documentID: string) {
  const [state, setState] = useState<SessionState>({ phase: 'idle' });
  const stateRef = useRef<SessionState>(state);
  stateRef.current = state;
  const timer = useRef<number | null>(null);

  const stopHeartbeat = useCallback(() => {
    if (timer.current) { window.clearInterval(timer.current); timer.current = null; }
  }, []);

  const startHeartbeat = useCallback((sessionID: string) => {
    stopHeartbeat();
    timer.current = window.setInterval(async () => {
      try { await heartbeatSession(documentID, sessionID); }
      catch (e: any) {
        if (e?.status === 409) setState({ phase: 'lost', reason: 'force_released' });
        else setState({ phase: 'lost', reason: 'network' });
        stopHeartbeat();
      }
    }, HEARTBEAT_MS);
  }, [documentID, stopHeartbeat]);

  const acquire = useCallback(async () => {
    setState({ phase: 'acquiring' });
    const res: AcquireResult = await acquireSession(documentID);
    if (res.mode === 'writer') {
      setState({ phase: 'writer', sessionID: res.session_id, lastAckRevisionID: res.last_ack_revision_id });
      startHeartbeat(res.session_id);
    } else {
      setState({ phase: 'readonly', heldBy: res.held_by, heldUntil: res.held_until });
    }
  }, [documentID, startHeartbeat]);

  const release = useCallback(async () => {
    if (state.phase !== 'writer') return;
    stopHeartbeat();
    try { await releaseSession(documentID, state.sessionID); } catch {}
    setState({ phase: 'idle' });
  }, [documentID, state, stopHeartbeat]);

  useEffect(() => {
    // Acquire on mount.
    acquire();
    // Release on unmount + on page hide (best-effort -- browser may block async fetch).
    const onHide = () => { if (stateRef.current.phase === 'writer') navigator.sendBeacon(`/api/v2/documents/${documentID}/session/release`, JSON.stringify({ session_id: (stateRef.current as any).sessionID })); };
    window.addEventListener('pagehide', onHide);
    return () => { stopHeartbeat(); window.removeEventListener('pagehide', onHide); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [documentID]);

  const setLastAck = useCallback((newAck: string) => {
    setState((cur) => (cur.phase === 'writer' ? { ...cur, lastAckRevisionID: newAck } : cur));
  }, []);

  return { state, acquire, release, setLastAck };
}
