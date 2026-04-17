import { Plugin, FileRepository } from 'ckeditor5';
import { MddmUploadAdapter, type UploadLoader } from './MddmUploadAdapter';

export interface MddmUploadAdapterPluginConfig {
  endpoint: string;
  getAuthHeader?: () => string | null;
}

export class MddmUploadAdapterPlugin extends Plugin {
  static get pluginName(): 'MddmUploadAdapterPlugin' {
    return 'MddmUploadAdapterPlugin';
  }

  static get requires(): ReadonlyArray<typeof FileRepository> {
    return [FileRepository];
  }

  init(): void {
    const cfg = (this.editor.config.get('mddmUpload') ?? {
      endpoint: '/assets',
    }) as MddmUploadAdapterPluginConfig;

    this.editor.plugins.get('FileRepository').createUploadAdapter = (loader) =>
      new MddmUploadAdapter({
        loader: loader as unknown as UploadLoader,
        endpoint: cfg.endpoint,
        getAuthHeader: cfg.getAuthHeader ?? (() => null),
      });
  }
}

export { MddmUploadAdapter };
