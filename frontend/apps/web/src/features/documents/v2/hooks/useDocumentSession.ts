import { useCallback, useEffect, useRef, useState } from 'react';
import { acquireSession, heartbeatSession, releaseSession, type AcquireResult } from '../api/documentsV2';

export type SessionState =
  | { phase: 'idle' }
  | { phase: 'acquiring' }
  | { phase: 'writer'; sessionID: string; lastAckRevisionID: string }
  | { phase: 'readonly'; heldBy: string; heldUntil: string }
  | { phase: 'lost'; reason: 'expired' | 'force_released' | 'network' };

const HEARTBEAT_MS = 30_000;
const VISIBILITY_REACQUIRE_MS = 2 * 60_000;

type AcquireOutcome = 'writer' | 'readonly' | 'failed' | 'skipped';

export function useDocumentSession(documentID: string) {
  const [state, setState] = useState<SessionState>({ phase: 'idle' });
  const stateRef = useRef<SessionState>(state);
  stateRef.current = state;
  const timer = useRef<number | null>(null);
  const mountedRef = useRef(true);
  const acquiringRef = useRef(false);
  const consecutiveFailuresRef = useRef(0);
  const hiddenAtRef = useRef<number | null>(null);
  const acquireRef = useRef<(opts?: { silent?: boolean }) => Promise<AcquireOutcome>>(async () => 'skipped');

  const stopHeartbeat = useCallback(() => {
    if (timer.current !== null) { window.clearInterval(timer.current); timer.current = null; }
  }, []);

  const startHeartbeat = useCallback((sessionID: string) => {
    stopHeartbeat();
    timer.current = window.setInterval(async () => {
      if (!mountedRef.current) return;
      try { await heartbeatSession(documentID, sessionID); }
      catch (e: any) {
        consecutiveFailuresRef.current += 1;
        const tooManyFailures = consecutiveFailuresRef.current >= 2;
        if (tooManyFailures) {
          if (mountedRef.current) setState({ phase: 'lost', reason: e?.status === 409 ? 'force_released' : 'network' });
          stopHeartbeat();
          return;
        }
        const reacquire = await acquireRef.current({ silent: true });
        if (reacquire !== 'writer') {
          if (mountedRef.current) setState({ phase: 'lost', reason: e?.status === 409 ? 'force_released' : 'network' });
          stopHeartbeat();
          return;
        }
        return;
      }
      consecutiveFailuresRef.current = 0;
    }, HEARTBEAT_MS);
  }, [documentID, stopHeartbeat]);

  const acquire = useCallback(async (opts?: { silent?: boolean }): Promise<AcquireOutcome> => {
    if (acquiringRef.current) return 'skipped';
    acquiringRef.current = true;
    if (!opts?.silent && mountedRef.current) setState({ phase: 'acquiring' });
    try {
      const res: AcquireResult = await acquireSession(documentID);
      if (res.mode === 'writer') {
        if (mountedRef.current) {
          setState({ phase: 'writer', sessionID: res.session_id, lastAckRevisionID: res.last_ack_revision_id });
          startHeartbeat(res.session_id);
        }
        return 'writer';
      } else {
        if (mountedRef.current) setState({ phase: 'readonly', heldBy: res.held_by, heldUntil: res.held_until });
        return 'readonly';
      }
    } catch {
      if (mountedRef.current) setState({ phase: 'lost', reason: 'network' });
      return 'failed';
    } finally {
      acquiringRef.current = false;
    }
  }, [documentID, startHeartbeat]);

  useEffect(() => {
    acquireRef.current = acquire;
  }, [acquire]);

  const release = useCallback(async () => {
    if (state.phase !== 'writer') return;
    stopHeartbeat();
    try { await releaseSession(documentID, state.sessionID); } catch {}
    if (mountedRef.current) setState({ phase: 'idle' });
  }, [documentID, state, stopHeartbeat]);

  useEffect(() => {
    mountedRef.current = true;
    // Acquire on mount.
    acquire();
    // Release on unmount + on page hide (best-effort -- browser may block async fetch).
    const onHide = () => { if (stateRef.current.phase === 'writer') navigator.sendBeacon(`/api/v2/documents/${documentID}/session/release`, JSON.stringify({ session_id: (stateRef.current as any).sessionID })); };
    const onVisibilityChange = () => {
      if (document.hidden) {
        hiddenAtRef.current = Date.now();
        stopHeartbeat();
        return;
      }
      const wasHiddenAt = hiddenAtRef.current;
      hiddenAtRef.current = null;
      if (stateRef.current.phase !== 'writer') return;
      const hiddenFor = wasHiddenAt === null ? 0 : Date.now() - wasHiddenAt;
      const maybeReacquire = async () => {
        if (hiddenFor > VISIBILITY_REACQUIRE_MS) await acquireRef.current({ silent: true });
        if (stateRef.current.phase === 'writer') startHeartbeat(stateRef.current.sessionID);
      };
      void maybeReacquire();
    };
    window.addEventListener('pagehide', onHide);
    document.addEventListener('visibilitychange', onVisibilityChange);
    return () => {
      mountedRef.current = false;
      stopHeartbeat();
      window.removeEventListener('pagehide', onHide);
      document.removeEventListener('visibilitychange', onVisibilityChange);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [documentID]);

  const setLastAck = useCallback((newAck: string) => {
    setState((cur) => (cur.phase === 'writer' ? { ...cur, lastAckRevisionID: newAck } : cur));
  }, []);

  return { state, acquire, release, setLastAck };
}
