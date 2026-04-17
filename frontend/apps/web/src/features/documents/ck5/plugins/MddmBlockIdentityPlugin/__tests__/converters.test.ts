import { afterEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../index';

const BID = 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa';

describe( 'MddmBlockIdentityPlugin converters', () => {
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

	async function createEditor(): Promise<ClassicEditor> {
		const el = document.createElement( 'div' );
		document.body.appendChild( el );
		hosts.push( el );

		const editor = await ClassicEditor.create( el, {
			licenseKey: 'GPL',
			plugins: [ Essentials, Paragraph, MddmBlockIdentityPlugin ],
		} );
		editors.push( editor );
		return editor;
	}

	it( 'upcasts data-mddm-bid into model', async () => {
		const editor = await createEditor();

		editor.setData( `<p data-mddm-bid="${ BID }">x</p>` );

		const root = editor.model.document.getRoot()!;
		const paragraph = root.getChild( 0 )!;
		const bid = ( paragraph as { getAttribute( key: string ): unknown } ).getAttribute( 'mddmBid' );

		expect( bid ).toBe( BID );
	} );

	it( 'downcasts mddmBid to data-mddm-bid on getData()', async () => {
		const editor = await createEditor();

		editor.setData( `<p data-mddm-bid="${ BID }">x</p>` );

		const out = editor.getData();
		expect( out ).toContain( `data-mddm-bid="${ BID }"` );
	} );

	it( 'round-trips data-mddm-bid', async () => {
		const editor = await createEditor();

		editor.setData( `<p data-mddm-bid="${ BID }">x</p>` );

		const out = editor.getData();
		expect( out ).toContain( `data-mddm-bid="${ BID }"` );
	} );
} );

