import { afterEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Table } from 'ckeditor5';
import { MddmTableVariantPlugin } from '../MddmTableVariantPlugin';

describe( 'MddmTableVariantPlugin', () => {
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

	it( 'upcasts data-mddm-variant="fixed" into model attribute', async () => {
		const el = document.createElement( 'div' );
		document.body.appendChild( el );
		hosts.push( el );

		const editor = await ClassicEditor.create( el, {
			licenseKey: 'GPL',
			plugins: [ Essentials, Paragraph, Table, MddmTableVariantPlugin ],
			initialData: '<table data-mddm-variant="fixed"><tbody><tr><td>A</td></tr></tbody></table>'
		} );
		editors.push( editor );

		const root = editor.model.document.getRoot()!;
		const table = Array.from( root.getChildren() ).find( node => node.is( 'element', 'table' ) );
		expect( table?.getAttribute( 'mddmTableVariant' ) ).toBe( 'fixed' );
	} );

	it( 'downcasts mddmTableVariant to data-mddm-variant', async () => {
		const el = document.createElement( 'div' );
		document.body.appendChild( el );
		hosts.push( el );

		const editor = await ClassicEditor.create( el, {
			licenseKey: 'GPL',
			plugins: [ Essentials, Paragraph, Table, MddmTableVariantPlugin ]
		} );
		editors.push( editor );

		editor.model.change( writer => {
			const table = writer.createElement( 'table', { mddmTableVariant: 'fixed' } );
			const row = writer.createElement( 'tableRow' );
			const cell = writer.createElement( 'tableCell' );
			const paragraph = writer.createElement( 'paragraph' );

			writer.append( paragraph, cell );
			writer.append( cell, row );
			writer.append( row, table );
			writer.insert( table, editor.model.document.getRoot()!, 0 );
		} );

		expect( editor.getData() ).toContain( 'data-mddm-variant="fixed"' );
	} );
} );
