import { Plugin } from 'ckeditor5';
import { registerBidConverters } from './converters';
import { registerBidPostFixer } from './post-fixer';
import { extendSchemaWithBid } from './schema';

export class MddmBlockIdentityPlugin extends Plugin {
  public static get pluginName() { return 'MddmBlockIdentity' as const; }

  public init(): void {
    extendSchemaWithBid(this.editor);
    registerBidConverters(this.editor);
    registerBidPostFixer(this.editor);
  }
}
