import { afterEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Table } from 'ckeditor5';
import { registerNestedTableGuard } from '../nestedTableGuard';

describe( 'registerNestedTableGuard', () => {
	const editors: Array<ClassicEditor> = [];
	const hosts: Array<HTMLElement> = [];

	afterEach( async () => {
		while ( editors.length ) {
			const editor = editors.pop();
			await editor?.destroy();
		}

		while ( hosts.length ) {
			hosts.pop()?.remove();
		}
	} );

	it( 'returns false for table inside tableCell context', async () => {
		const el = document.createElement( 'div' );
		document.body.appendChild( el );
		hosts.push( el );

		const editor = await ClassicEditor.create( el, {
			licenseKey: 'GPL',
			plugins: [ Essentials, Paragraph, Table ]
		} );
		editors.push( editor );

		registerNestedTableGuard( editor );

		expect( editor.model.schema.checkChild( 'tableCell', 'table' ) ).toBe( false );
	} );
} );
