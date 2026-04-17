import type { Editor } from 'ckeditor5';

export function registerSchemaV4Migration( editor: Editor ): void {
	editor.data.on( 'set', ( _evt, args ) => {
		const input = args[ 0 ];
		if ( typeof input !== 'string' ) {
			return;
		}

		const hasAnyBid = /data-mddm-bid=/.test( input );
		if ( hasAnyBid ) {
			return;
		}

		// Post-fixer mints bids after upcast — just log.
		// eslint-disable-next-line no-console
		console.info( 'mddm:schema-upgrade-v4 — legacy doc migrated on load' );
	}, { priority: 'high' } );
}
