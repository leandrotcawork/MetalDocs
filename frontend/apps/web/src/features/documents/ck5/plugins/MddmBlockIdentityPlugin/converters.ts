import type { Editor } from 'ckeditor5';
import { PAGINABLE_ELEMENT_NAMES } from './schema';

export function registerBidConverters(editor: Editor): void {
	const conversion = editor.conversion;
	const schema = editor.model.schema;

	// Upcast: read data-mddm-bid from any HTML element → mddmBid model attribute.
	// No name filter — model schema allowAttributes guards which elements accept it.
	conversion.for( 'upcast' ).attributeToAttribute( {
		view: 'data-mddm-bid',
		model: 'mddmBid',
	} );

	// Downcast: write mddmBid back to data-mddm-bid in data/editing view per element type.
	for ( const name of PAGINABLE_ELEMENT_NAMES ) {
		if ( !schema.isRegistered( name ) ) {
			continue;
		}

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
