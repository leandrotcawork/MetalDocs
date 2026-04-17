import { Plugin } from 'ckeditor5';
import { registerBidClipboardHandler } from './clipboard';
import { registerBidConverters } from './converters';
import { registerBidPostFixer } from './post-fixer';
import { extendSchemaWithBid } from './schema';

export class MddmBlockIdentityPlugin extends Plugin {
  public static get pluginName() { return 'MddmBlockIdentity' as const; }

  public static override get requires() {
    return ['ClipboardPipeline'] as const;
  }

  public init(): void {
    extendSchemaWithBid(this.editor);
    registerBidConverters(this.editor);
    registerBidPostFixer(this.editor);
    registerBidClipboardHandler(this.editor);
  }
}
