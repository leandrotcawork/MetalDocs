import { afterEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, StandardEditingMode, Table, Widget } from 'ckeditor5';
import { MddmTableVariantPlugin } from '../MddmTableVariantPlugin';
import { applyPerCellExceptions, CELL_MARKER_PREFIX } from '../perCellExceptionWalker';

function getCellMarkerNames( editor: ClassicEditor ): Array<string> {
	return Array.from( editor.model.markers )
		.map( marker => marker.name )
		.filter( name => name.startsWith( CELL_MARKER_PREFIX ) )
		.sort();
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

	it( 'creates 2 markers for 2 cells in fixed table', async () => {
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

		expect( getCellMarkerNames( editor ) ).toHaveLength( 2 );
	} );

	it( 'creates 0 markers when table has no mddmTableVariant', async () => {
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

		expect( getCellMarkerNames( editor ) ).toHaveLength( 0 );
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
		const firstPass = getCellMarkerNames( editor );
		applyPerCellExceptions( editor );
		const secondPass = getCellMarkerNames( editor );

		expect( firstPass ).toEqual( secondPass );
		expect( secondPass ).toHaveLength( 2 );
	} );
} );
