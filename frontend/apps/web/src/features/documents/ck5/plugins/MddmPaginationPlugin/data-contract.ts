import type { Editor } from 'ckeditor5';
import type { ComputedBreak } from './types';

export function installPaginationDataContract(
  editor: Editor,
  getBreaks: () => readonly ComputedBreak[],
): void {
  const original = editor.data.get.bind(editor.data);
  (editor.data as any).get = (options?: any) => {
    const html: string = original(options);
    if (!options?.pagination) return html;
    return injectPageAttrs(html, getBreaks());
  };
}

function injectPageAttrs(html: string, breaks: readonly ComputedBreak[]): string {
  if (!breaks.length) return html;
  const byAfter = new Map(breaks.map(b => [b.afterBid, b.pageNumber]));
  const tagRe = /<([a-z][a-z0-9]*)(?:\s[^>]*?)?\bdata-mddm-bid="([^"]+)"[^>]*>/gi;
  const positions: Array<{ bid: string; tagEnd: number }> = [];
  let m: RegExpExecArray | null;
  while ((m = tagRe.exec(html))) {
    positions.push({ bid: m[2], tagEnd: tagRe.lastIndex });
  }
  const insertions: Array<{ offset: number; page: number }> = [];
  for (let i = 0; i < positions.length - 1; i++) {
    const page = byAfter.get(positions[i].bid);
    if (page !== undefined) {
      const nextTagStart = html.lastIndexOf('<', positions[i + 1].tagEnd - 1);
      const closingGt = html.indexOf('>', nextTagStart);
      if (closingGt !== -1) {
        insertions.push({ offset: closingGt, page });
      }
    }
  }
  insertions.sort((a, b) => b.offset - a.offset);
  let out = html;
  for (const { offset, page } of insertions) {
    out = out.slice(0, offset) + ` data-pagination-page="${page}"` + out.slice(offset);
  }
  return out;
}
