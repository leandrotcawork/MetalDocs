import type { Editor } from 'ckeditor5';
import { PAGINABLE_ELEMENT_NAMES } from './schema';

export function registerBidConverters(editor: Editor): void {
	const conversion = editor.conversion;
	const schema = editor.model.schema;

	for ( const name of PAGINABLE_ELEMENT_NAMES ) {
		if ( !schema.isRegistered( name ) ) {
			continue;
		}

		conversion.for( 'upcast' ).attributeToAttribute( {
			view: { name, key: 'data-mddm-bid' },
			model: { name, key: 'mddmBid' },
		} );

		conversion.for( 'dataDowncast' ).attributeToAttribute( {
			model: { name, key: 'mddmBid' },
			view: 'data-mddm-bid',
		} );

		conversion.for( 'editingDowncast' ).attributeToAttribute( {
			model: { name, key: 'mddmBid' },
			view: 'data-mddm-bid',
		} );
	}
}
