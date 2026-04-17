import { parseHTML } from 'linkedom';

const PAGINABLE_TAGS = new Set([
  'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
  'li', 'blockquote', 'tr',
  'figure',
]);
const MDDM_WIDGET_CLASSES = ['mddm-section', 'mddm-repeatable', 'mddm-repeatable-item', 'mddm-data-table', 'mddm-field-group', 'mddm-rich-block'];

export type ValidationResult =
  | { ok: true }
  | { ok: false; severity: 'error'; error: 'bid-collision'; bids: string[] }
  | { ok: false; severity: 'warn'; error: 'paginable-missing-bid'; elements: string[] };

export function validateBids(html: string): ValidationResult {
  const { document } = parseHTML(`<!DOCTYPE html><html><body>${html}</body></html>`);
  const all = Array.from(document.querySelectorAll('[data-mddm-bid]'));
  const seen = new Map<string, number>();
  for (const el of all) {
    const bid = (el as Element).getAttribute('data-mddm-bid')!;
    seen.set(bid, (seen.get(bid) ?? 0) + 1);
  }
  const dups = [...seen.entries()].filter(([, n]) => n > 1).map(([bid]) => bid);
  if (dups.length) return { ok: false, severity: 'error', error: 'bid-collision', bids: dups };

  const paginableWithoutBid: string[] = [];
  for (const el of document.querySelectorAll('*')) {
    const tag = (el as Element).tagName.toLowerCase();
    const isPaginable =
      PAGINABLE_TAGS.has(tag) ||
      MDDM_WIDGET_CLASSES.some(c => (el as Element).classList?.contains(c));
    if (isPaginable && !(el as Element).hasAttribute('data-mddm-bid')) {
      paginableWithoutBid.push(tag);
    }
  }
  if (paginableWithoutBid.length) {
    return { ok: false, severity: 'warn', error: 'paginable-missing-bid', elements: paginableWithoutBid };
  }
  return { ok: true };
}
