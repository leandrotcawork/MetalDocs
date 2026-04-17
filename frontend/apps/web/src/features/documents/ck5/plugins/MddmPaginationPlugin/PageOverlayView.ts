import type { Editor } from 'ckeditor5';
import type { ComputedBreak } from './types';

export class PageOverlayView {
  private host: HTMLElement | null = null;

  public constructor(private readonly editor: Editor) {
    const editable = (this.editor.ui as unknown as {
      view?: { editable?: { element?: HTMLElement | null } };
    })?.view?.editable?.element;
    const parent = editable?.parentElement ?? document.body;
    this.host = document.createElement('div');
    this.host.className = 'mddm-page-overlay-host';
    this.host.style.cssText = 'position:absolute;inset:0;pointer-events:none;';
    parent.appendChild(this.host);
  }

  public update(breaks: readonly Pick<ComputedBreak, 'afterBid' | 'pageNumber'>[]): void {
    if (!this.host) return;
    this.host.innerHTML = '';
    for (const b of breaks) {
      const bar = document.createElement('div');
      bar.className = 'mddm-page-overlay';
      bar.setAttribute('data-after-bid', b.afterBid);
      bar.textContent = `Page ${b.pageNumber}`;
      bar.style.pointerEvents = 'none';
      this.host.appendChild(bar);
    }
  }

  public destroy(): void {
    this.host?.remove();
    this.host = null;
  }
}
