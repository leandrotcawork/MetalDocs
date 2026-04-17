import { Plugin } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../MddmBlockIdentityPlugin';

export class MddmPaginationPlugin extends Plugin {
  public static get pluginName() { return 'MddmPagination' as const; }
  public static get requires() { return [MddmBlockIdentityPlugin] as const; }

  public init(): void {
    // Sub-modules wired in Tasks 13-18.
  }
}
