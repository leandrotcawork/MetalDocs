import type { ParityReport } from "./parity-diff";

type ParityStoreEntry = Readonly<{
  report: ParityReport;
  updatedAt: number;
}>;

export class ParityStore {
  private readonly reports = new Map<string, ParityStoreEntry>();

  public put(docId: string, report: ParityReport): void {
    this.reports.set(docId, { report, updatedAt: Date.now() });
  }

  public get(docId: string): ParityStoreEntry | undefined {
    return this.reports.get(docId);
  }

  public size(): number {
    return this.reports.size;
  }

  public clear(): void {
    this.reports.clear();
  }
}

export const paginationStore = new ParityStore();
