export interface UploadLoader {
  file: Promise<File>;
  uploadTotal?: number;
  uploaded?: number;
}

export interface MddmUploadAdapterOptions {
  loader: UploadLoader;
  endpoint: string;
  getAuthHeader: () => string | null;
}

export class MddmUploadAdapter {
  private loader: UploadLoader;
  private endpoint: string;
  private getAuthHeader: () => string | null;
  private controller: AbortController | null = null;

  constructor(opts: MddmUploadAdapterOptions) {
    this.loader = opts.loader;
    this.endpoint = opts.endpoint;
    this.getAuthHeader = opts.getAuthHeader;
  }

  async upload(): Promise<{ default: string }> {
    const file = await this.loader.file;
    const form = new FormData();
    form.append('file', file);
    this.controller = new AbortController();

    const auth = this.getAuthHeader();
    const headers: Record<string, string> = {};
    if (auth) headers.Authorization = auth;

    const res = await fetch(this.endpoint, {
      method: 'POST',
      body: form,
      headers,
      signal: this.controller.signal,
    });
    if (!res.ok) {
      throw new Error(`upload failed with status ${res.status}`);
    }
    const body = (await res.json()) as { url: string };
    return { default: body.url };
  }

  abort(): void {
    this.controller?.abort();
  }
}
