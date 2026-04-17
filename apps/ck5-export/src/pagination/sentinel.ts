import { parseHTML } from 'linkedom';

export function injectSentinels(html: string): string {
  const { document } = parseHTML(`<!DOCTYPE html><html><body>${html}</body></html>`);
  const targets = Array.from(document.querySelectorAll('[data-mddm-bid]'));
  for (const el of targets) {
    const bid = (el as Element).getAttribute('data-mddm-bid')!;
    const first = (el as Element).firstElementChild;
    if (first && first.tagName.toLowerCase() === 'span' && first.getAttribute('data-pb-marker') === bid) continue;
    const s = document.createElement('span');
    s.setAttribute('data-pb-marker', bid);
    s.setAttribute('style', 'display:inline;width:0;height:0');
    el.insertBefore(s, el.firstChild);
  }
  return (document.body as any).innerHTML;
}