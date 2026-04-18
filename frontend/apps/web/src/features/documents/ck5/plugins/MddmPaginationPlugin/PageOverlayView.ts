import type { Editor } from 'ckeditor5';
import type { ComputedBreak } from './types';

export class PageOverlayView {
  private host: HTMLElement | null = null;

  public constructor(private readonly editor: Editor) {
    const editable = this.getEditableElement();
    const parent = editable?.parentElement ?? document.body;
    this.host = document.createElement('div');
    this.host.className = 'mddm-page-overlay-host';
    this.host.style.cssText = 'position:absolute;inset:0;pointer-events:none;';
    parent.appendChild(this.host);
  }

  public update(breaks: readonly Pick<ComputedBreak, 'afterBid' | 'pageNumber'>[]): void {
    this.host?.replaceChildren();

    const editable = this.getEditableElement();
    if (!editable) return;

    const stale = editable.querySelectorAll<HTMLElement>('[data-mddm-page-break-after]');
    for (const node of stale) {
      node.removeAttribute('data-mddm-page-break-after');
      node.removeAttribute('data-mddm-next-page');
    }

    for (const b of breaks) {
      const target = editable.querySelector<HTMLElement>(`[data-mddm-bid="${b.afterBid}"]`);
      if (!target) continue;
      target.setAttribute('data-mddm-page-break-after', '');
      target.setAttribute('data-mddm-next-page', String(b.pageNumber));
    }
  }

  public destroy(): void {
    this.host?.remove();
    this.host = null;
  }

  private getEditableElement(): HTMLElement | null {
    return (this.editor.ui as unknown as {
      view?: { editable?: { element?: HTMLElement | null } };
    })?.view?.editable?.element;
  }
}
