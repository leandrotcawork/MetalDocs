import { Plugin } from 'ckeditor5';
import { extendSchemaWithBid } from './schema';

export class MddmBlockIdentityPlugin extends Plugin {
  public static get pluginName() { return 'MddmBlockIdentity' as const; }

  public init(): void {
    extendSchemaWithBid(this.editor);
  }
}
