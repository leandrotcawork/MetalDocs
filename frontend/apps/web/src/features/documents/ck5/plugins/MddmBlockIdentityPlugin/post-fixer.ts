import { v4 as uuidv4 } from 'uuid';
import { Editor, Element } from 'ckeditor5';
import { PAGINABLE_ELEMENT_NAMES } from './schema';

const PAGINABLE = new Set<string>(PAGINABLE_ELEMENT_NAMES);

export function registerBidPostFixer(editor: Editor): void {
	editor.model.document.registerPostFixer(writer => {
		let changed = false;

		for ( const root of editor.model.document.getRoots() ) {
			for ( const node of editor.model.createRangeIn( root ).getItems() ) {
				if ( !node.is( 'element' ) ) {
					continue;
				}

				const element = node as Element;
				if ( !PAGINABLE.has( element.name ) ) {
					continue;
				}

				if ( element.hasAttribute( 'mddmBid' ) ) {
					continue;
				}

				writer.setAttribute( 'mddmBid', uuidv4(), element );
				changed = true;
			}
		}

		return changed;
	} );
}
