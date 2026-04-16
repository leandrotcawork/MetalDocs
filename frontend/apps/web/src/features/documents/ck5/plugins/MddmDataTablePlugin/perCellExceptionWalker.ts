import type { Editor } from 'ckeditor5';

export function applyPerCellExceptions( editor: Editor ): void {
	editor.model.change( writer => {
		const root = editor.model.document.getRoot();

		if ( !root ) {
			return;
		}

		visitNode( root, node => {
			if ( !node.is( 'element', 'table' ) ) {
				return;
			}

			const variant = node.getAttribute( 'mddmTableVariant' );

			if ( variant === undefined ) {
				return;
			}

			visitNode( node, innerNode => {
				if ( !innerNode.is( 'element', 'tableCell' ) ) {
					return;
				}

				// Idempotency: skip if already wrapped.
				const firstChild = Array.from( innerNode.getChildren() )[ 0 ];
				if (
					firstChild &&
					(firstChild as { is?: ( kind: string, name?: string ) => boolean }).is?.(
						'element',
						'restrictedEditingException',
					)
				) {
					return;
				}

				const children = Array.from( innerNode.getChildren() );
				const exception = writer.createElement( 'restrictedEditingException' );
				writer.append( exception, innerNode );

				for ( const child of children ) {
					writer.move(
						writer.createRangeOn( child as any ),
						writer.createPositionAt( exception, 'end' ),
					);
				}
			} );
		} );
	} );
}

function visitNode( node: any, cb: ( node: any ) => void ): void {
	cb( node );

	if ( !node.getChildren ) {
		return;
	}

	for ( const child of node.getChildren() ) {
		visitNode( child, cb );
	}
}
