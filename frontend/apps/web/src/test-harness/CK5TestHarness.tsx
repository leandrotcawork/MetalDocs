import { useEffect, useState } from 'react';
import { AuthorPage } from '../features/documents/ck5/react/AuthorPage';
import { FillPage } from '../features/documents/ck5/react/FillPage';

type Mode = 'author' | 'fill';

interface HarnessParams {
  mode: Mode;
  tplId: string;
  docId: string;
}

function parseHash(): HarnessParams | { error: string } {
  const raw = window.location.hash.split('?')[1] ?? '';
  const params = new URLSearchParams(raw);
  const mode = params.get('mode') as Mode | null;
  if (mode !== 'author' && mode !== 'fill') {
    return { error: 'missing or invalid `mode` (expected `author` or `fill`)' };
  }
  const tplId = params.get('tpl') ?? 'sandbox';
  const docId = params.get('doc') ?? `${tplId}-doc`;
  return { mode, tplId, docId };
}

export function CK5TestHarness() {
  const [state, setState] = useState<HarnessParams | { error: string }>(() => parseHash());

  useEffect(() => {
    if (!import.meta.env.DEV) {
      setState({ error: 'CK5 test harness is disabled in production builds' });
      return;
    }
    const onHashChange = () => setState(parseHash());
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  if ('error' in state) {
    return (
      <div data-testid="ck5-harness-error" style={{ padding: 24 }}>
        CK5 test harness error: {state.error}
      </div>
    );
  }

  return state.mode === 'author' ? (
    <AuthorPage tplId={state.tplId} />
  ) : (
    <FillPage tplId={state.tplId} docId={state.docId} />
  );
}
