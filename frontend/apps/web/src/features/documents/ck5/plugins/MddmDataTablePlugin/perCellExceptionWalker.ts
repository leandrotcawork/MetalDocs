import type { Editor } from 'ckeditor5';

export const CELL_MARKER_PREFIX = 'restrictedEditingException:mddmCell:';

export function applyPerCellExceptions( editor: Editor ): void {
	editor.model.change( writer => {
		for ( const marker of Array.from( editor.model.markers ) ) {
			if ( marker.name.startsWith( CELL_MARKER_PREFIX ) ) {
				writer.removeMarker( marker );
			}
		}

		const root = editor.model.document.getRoot();

		if ( !root ) {
			return;
		}

		let tableIdx = 0;

		visitNode( root, node => {
			if ( !node.is( 'element', 'table' ) ) {
				return;
			}

			const variant = node.getAttribute( 'mddmTableVariant' );

			if ( variant === undefined ) {
				return;
			}

			const tableKey = String( node.getAttribute( 'mddmTableId' ) ?? `t${ tableIdx }` );
			tableIdx++;

			const rows = Array.from( node.getChildren() );

			for ( let rowIdx = 0; rowIdx < rows.length; rowIdx++ ) {
				const row = rows[ rowIdx ];

				if ( !row.is( 'element', 'tableRow' ) ) {
					continue;
				}

				const cells = Array.from( row.getChildren() );

				for ( let colIdx = 0; colIdx < cells.length; colIdx++ ) {
					const cell = cells[ colIdx ];

					if ( !cell.is( 'element', 'tableCell' ) ) {
						continue;
					}

					writer.addMarker( `${ CELL_MARKER_PREFIX }${ tableKey }r${ rowIdx }c${ colIdx }`, {
						range: writer.createRangeOn( cell ),
						usingOperation: true,
						affectsData: true
					} );
				}
			}
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
