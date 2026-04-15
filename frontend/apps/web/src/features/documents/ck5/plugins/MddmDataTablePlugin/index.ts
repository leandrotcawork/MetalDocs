import { Plugin } from 'ckeditor5';
import { MddmTableVariantPlugin } from './MddmTableVariantPlugin';
import { MddmTableLockPlugin } from './MddmTableLockPlugin';
import { registerNestedTableGuard } from './nestedTableGuard';

export class MddmDataTablePlugin extends Plugin {
	public static get pluginName() {
		return 'MddmDataTablePlugin' as const;
	}

	public static get requires() {
		return [ MddmTableVariantPlugin, MddmTableLockPlugin ] as const;
	}

	public init(): void {
		registerNestedTableGuard( this.editor );
	}
}

export { applyPerCellExceptions } from './perCellExceptionWalker';
