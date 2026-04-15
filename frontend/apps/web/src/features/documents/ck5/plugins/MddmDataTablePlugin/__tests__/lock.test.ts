import { afterEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Table } from 'ckeditor5';
import { MddmTableVariantPlugin } from '../MddmTableVariantPlugin';
import { MddmTableLockPlugin } from '../MddmTableLockPlugin';

function navigateIntoCell( editor: ClassicEditor ): void {
	editor.model.change( writer => {
		const root = editor.model.document.getRoot()!;
		const table = Array.from( root.getChildren() ).find( node => node.is( 'element', 'table' ) )!;
		const row = table.getChild( 0 )!;
		const cell = row.getChild( 0 )!;
		const para = cell.getChild( 0 )!;

		writer.setSelection( writer.createPositionAt( para, 0 ) );
	} );
}

describe( 'MddmTableLockPlugin', () => {
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

	it( 'disables insertTableRowBelow inside fixed table', async () => {
		const el = document.createElement( 'div' );
		document.body.appendChild( el );
		hosts.push( el );

		const editor = await ClassicEditor.create( el, {
			licenseKey: 'GPL',
			plugins: [ Essentials, Paragraph, Table, MddmTableVariantPlugin, MddmTableLockPlugin ],
			initialData: '<table data-mddm-variant="fixed"><tbody><tr><td>A</td></tr></tbody></table>'
		} );
		editors.push( editor );

		navigateIntoCell( editor );

		expect( editor.commands.get( 'insertTableRowBelow' )?.isEnabled ).toBe( false );
	} );

	it( 'keeps insertTableRowBelow enabled inside dynamic table', async () => {
		const el = document.createElement( 'div' );
		document.body.appendChild( el );
		hosts.push( el );

		const editor = await ClassicEditor.create( el, {
			licenseKey: 'GPL',
			plugins: [ Essentials, Paragraph, Table, MddmTableVariantPlugin, MddmTableLockPlugin ],
			initialData: '<table data-mddm-variant="dynamic"><tbody><tr><td>A</td></tr></tbody></table>'
		} );
		editors.push( editor );

		navigateIntoCell( editor );

		expect( editor.commands.get( 'insertTableRowBelow' )?.isEnabled ).toBe( true );
	} );
} );
