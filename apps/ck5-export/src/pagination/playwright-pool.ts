import { chromium, type Browser } from 'playwright';

export type Worker = Browser;

export class PlaywrightPool {
  private workers: Worker[] = [];
  private idle: Worker[] = [];
  private waiters: Array<(w: Worker) => void> = [];

  public constructor(private readonly opts: { size: number }) {}

  public async init(): Promise<void> {
    for (let i = 0; i < this.opts.size; i++) {
      const b = await chromium.launch({ headless: true });
      this.workers.push(b);
      this.idle.push(b);
    }
  }

  public async acquire(): Promise<Worker> {
    if (this.idle.length) return this.idle.shift()!;
    return new Promise(resolve => this.waiters.push(resolve));
  }

  public release(w: Worker): void {
    const waiter = this.waiters.shift();
    if (waiter) { waiter(w); return; }
    this.idle.push(w);
  }

  public async replace(w: Worker): Promise<void> {
    const i = this.workers.indexOf(w);
    if (i >= 0) this.workers.splice(i, 1);
    try { await w.close(); } catch { /* already crashed */ }
    const fresh = await chromium.launch({ headless: true });
    this.workers.push(fresh);
    const waiter = this.waiters.shift();
    if (waiter) waiter(fresh); else this.idle.push(fresh);
  }

  public async shutdown(): Promise<void> {
    await Promise.all(this.workers.map(w => w.close()));
    this.workers = [];
    this.idle = [];
    this.waiters = [];
  }
}