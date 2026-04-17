import type { Editor } from 'ckeditor5';

export function registerNestedTableGuard( editor: Editor ): void {
	editor.model.schema.addChildCheck( ( context, definition ) => {
		if ( context.endsWith( 'tableCell' ) && definition.name === 'table' ) {
			return false;
		}

		return undefined;
	} );
}
