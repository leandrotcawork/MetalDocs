import { afterEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, StandardEditingMode, Table, Widget } from 'ckeditor5';
import { MddmTableVariantPlugin } from '../MddmTableVariantPlugin';
import { applyPerCellExceptions } from '../perCellExceptionWalker';

function countCellExceptions( editor: ClassicEditor ): number {
	let count = 0;
	function visit( node: any ): void {
		if ( node.is && node.is( 'element', 'restrictedEditingException' ) ) {
			count++;
		}
		if ( node.getChildren ) {
			for ( const child of node.getChildren() ) {
				visit( child );
			}
		}
	}
	visit( editor.model.document.getRoot() );
	return count;
}

describe( 'applyPerCellExceptions', () => {
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

	it( 'wraps 2 cells in restrictedEditingException elements for fixed table', async () => {
		const el = document.createElement( 'div' );
		document.body.appendChild( el );
		hosts.push( el );

		const editor = await ClassicEditor.create( el, {
			licenseKey: 'GPL',
			plugins: [ Essentials, Paragraph, Table, Widget, StandardEditingMode, MddmTableVariantPlugin ],
			initialData: '<table data-mddm-variant="fixed"><tbody><tr><td>A</td><td>B</td></tr></tbody></table>'
		} );
		editors.push( editor );

		applyPerCellExceptions( editor );

		expect( countCellExceptions( editor ) ).toBe( 2 );
	} );

	it( 'wraps 0 cells when table has no mddmTableVariant', async () => {
		const el = document.createElement( 'div' );
		document.body.appendChild( el );
		hosts.push( el );

		const editor = await ClassicEditor.create( el, {
			licenseKey: 'GPL',
			plugins: [ Essentials, Paragraph, Table, Widget, StandardEditingMode, MddmTableVariantPlugin ],
			initialData: '<table><tbody><tr><td>A</td><td>B</td></tr></tbody></table>'
		} );
		editors.push( editor );

		applyPerCellExceptions( editor );

		expect( countCellExceptions( editor ) ).toBe( 0 );
	} );

	it( 'is idempotent when run twice', async () => {
		const el = document.createElement( 'div' );
		document.body.appendChild( el );
		hosts.push( el );

		const editor = await ClassicEditor.create( el, {
			licenseKey: 'GPL',
			plugins: [ Essentials, Paragraph, Table, Widget, StandardEditingMode, MddmTableVariantPlugin ],
			initialData: '<table data-mddm-variant="fixed"><tbody><tr><td>A</td><td>B</td></tr></tbody></table>'
		} );
		editors.push( editor );

		applyPerCellExceptions( editor );
		const firstPass = countCellExceptions( editor );
		applyPerCellExceptions( editor );
		const secondPass = countCellExceptions( editor );

		expect( firstPass ).toBe( 2 );
		expect( secondPass ).toBe( 2 );
	} );
} );
